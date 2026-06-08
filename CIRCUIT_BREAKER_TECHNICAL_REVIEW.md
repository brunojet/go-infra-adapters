# Revisão Técnica Aprofundada: Circuit Breaker Middleware

## Executive Summary

A implementação do circuit breaker é **limpa e defensiva**, mas sofre de um **problema arquitetural crítico**: classifica todos os erros como falhas de indisponibilidade, quando na realidade muitos indicam problemas do **cliente** ou erros **transitórios não-retentáveis**. Isso causa trips indevidos e prejudica a eficácia da resiliência.

---

## 1. CLASSIFICAÇÃO DE FALHAS: O Problema Central

### 🔴 **CRÍTICO: Qualquer erro = Falha do Breaker**

```go
if err != nil {
    return nil, err  // ← TRATA TODOS OS ERROS IGUAL
}
```

**Problema**: A linha 67-73 trata como falha:
```
✓ 500 Server Error       → Indisponibilidade real
✓ 503 Service Unavailable → Indisponibilidade real
✓ 429 Too Many Requests   → Degradação real
✗ 400 Bad Request         → Erro do cliente, NÃO do servidor
✗ 401 Unauthorized        → Erro de autenticação, NÃO indisponibilidade
✗ 404 Not Found           → Recurso não existe, NÃO indisponibilidade
✗ context.Canceled        → Cliente cancelou, NÃO indisponibilidade
✗ net.DNSError            → Problema de DNS, transitório
✗ net.OpError (timeout)   → Timeout de rede, pode ser transitório
✗ connection reset        → Pode ser erro de cliente
```

### ⚠️ **Cenários problemáticos**

#### Cenário 1: Rate Limiting do Backend
```
Requisição 1: 429 Too Many Requests (erro legítimo)
ConsecutiveFailures = 1

Requisição 2-5: 429 (mais requisições rápido)
ConsecutiveFailures = 5 ✓ Trip! ← CORRETO

MAS: Se cliente local for rate-limited por servidor:
- Circuit breaker abre
- Cliente não consegue nem tentar mais
- Perde a chance de backoff natural do 429
```

#### Cenário 2: Erro de Cliente (4xx)
```
POST /api/users
Body: { "email": "invalid-email" } ← Erro do cliente

Resposta: 400 Bad Request
ConsecutiveFailures = 1
... após 5 requisições ruins: CB abre ✗ PROBLEMA!

- Servidor está funcionando perfeitamente
- Circuit breaker abriu por erro do cliente
- Quando o cliente corrige, ele não consegue porque CB está OPEN
```

#### Cenário 3: DNS Intermitente
```
net.DNSError (temporary: true)
↓
Error incrementa ConsecutiveFailures
↓
Após 5 tentativas: CB abre
↓
MAS: DNS pode se recuperar em 2 segundos
↓
Client: "Espera 10s para CB fechar, mas já poderia estar falando com servidor"
```

### ✅ **Solução: FailureClassifier**

```go
type FailureClassifier interface {
    // ClassifyError determina se um erro indica indisponibilidade do servidor
    // Retorna:
    //   - IndisponibilityServerError: Servidor genuinamente indisponível
    //   - IndisponibilityTransient: Transitório (pode se recuperar rápido)
    //   - IndisponibilityClient: Erro do cliente (nunca tripar CB)
    //   - IndisponibilityUnknown: Desconhecido (comportamento padrão: tripar)
    ClassifyError(err error, resp *http.Response) ErrorClassification
}

type ErrorClassification int
const (
    IndisponibilityServerError  ErrorClassification = iota
    IndisponibilityTransient
    IndisponibilityClient
    IndisponibilityUnknown
)
```

### 🎯 **Classificação proposta**

```
TRIPAR o circuit breaker (=Indisponibilidade real):
├─ HTTP 500 Internal Server Error
├─ HTTP 502 Bad Gateway
├─ HTTP 503 Service Unavailable
├─ HTTP 504 Gateway Timeout
├─ net.OpError (connection refused)
└─ net.DNSError (NOT temporary)

USAR RETRY, NÃO TRIPAR (=Transitório):
├─ HTTP 429 Too Many Requests
├─ net.DNSError (temporary=true)
├─ net.OpError (timeout)
├─ io.EOF durante leitura
└─ connection reset/broken pipe

NUNCA TRIPAR (=Erro do cliente):
├─ HTTP 4xx (400, 401, 403, 404, 405, 409, 410)
├─ context.Canceled
├─ context.DeadlineExceeded (timeout DO REQUEST, não da rede)
└─ Validação de input
```

---

## 2. TRATAMENTO DE HTTP 5xx

### 🔴 **PROBLEMA: RoundTrip não retorna erro em 5xx**

```go
func (b *breakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    _, err := b.cb.Execute(func() (any, error) {
        r, err := b.next.RoundTrip(req)
        if err != nil {
            return nil, err
        }
        resp = r
        return resp, nil  // ← Retorna nil error mesmo se StatusCode == 500!
    })
    return resp, err
}
```

**Consequência**:
```go
resp, err := client.Do(req)
if err != nil {  // ← NÃO vai entrar para 5xx!
    // Não considera 5xx como erro
}

// Código cliente precisa fazer:
if resp.StatusCode >= 500 {
    // Tripar breaker manualmente?
}
```

### ✅ **Solução: Permitir classificação de status codes**

```go
type StatusCodeClassifier interface {
    IsServerError(statusCode int) bool
}

type DefaultStatusCodeClassifier struct{}

func (c *DefaultStatusCodeClassifier) IsServerError(code int) bool {
    return code >= 500 && code != 501 // 501 Not Implemented é raro
}
```

**Uso**:
```go
_, err := b.cb.Execute(func() (any, error) {
    r, err := b.next.RoundTrip(req)
    
    // Classificar status code como erro
    if r != nil && b.statusCodeClassifier.IsServerError(r.StatusCode) {
        return nil, fmt.Errorf("server error: %d", r.StatusCode)
    }
    
    return r, err
})
```

---

## 3. OBSERVABILIDADE: Invisibilidade Total

### 🔴 **PROBLEMA: Sem métricas, callbacks ou logs**

Cenários reais:
```
🤔 "Por que meu endpoint está retornando erro?"
   → Circuit breaker pode estar OPEN, mas você não consegue saber

🤔 "Quantas vezes o CB abriu hoje?"
   → Impossível medir

🤔 "Qual foi a métrica que causou o trip?"
   → Sem visibilidade

🤔 "Quanto tempo leva para recuperar?"
   → Não há alertas
```

### ✅ **Solução: Callbacks de observabilidade**

```go
type BreakerMetrics interface {
    OnStateChange(oldState, newState gobreaker.State)
    OnRequest(statusCode int, latency time.Duration, err error)
    OnFailure(failure FailureType)
}

type FailureType int
const (
    FailureConsecutive FailureType = iota
    FailureRate
    FailureTimeout
)

// Uso em NewBreakerMiddleware:
breaker.OnStateChange = func(old, new gobreaker.State) {
    log.Infof("Circuit breaker transitioned: %v → %v", old, new)
    metrics.Gauge("breaker.state", float64(new))
}
```

---

## 4. NOME FIXO DO BREAKER

### 🟡 **PROBLEMA: Impossível distinguir múltiplos clientes**

```go
settings := gobreaker.Settings{
    Name: "breaker",  // ← FIXO! Não diferencia clientes
}
```

**Cenário**:
```go
// Cliente para API A
breakerA := NewBreakerMiddleware(transport1)

// Cliente para API B
breakerB := NewBreakerMiddleware(transport2)

// Ambos têm Name="breaker"
// → Métricas/logs são ambíguos
// → Você não sabe qual cliente está falhando
```

### ✅ **Solução: Permitir nome customizável**

```go
type BreakerConfig struct {
    MaxFailures        int
    ResetTimeout       time.Duration
    HalfOpenRequests   int
    Name               string  // ← NOVO
    FailureClassifier  FailureClassifier
    Metrics            BreakerMetrics
}

func NewBreakerMiddleware(base http.RoundTripper, opts ...BreakerOption) http.RoundTripper {
    cfg := newCircuitBreakerConfig(opts...)
    // Use cfg.Name ao invés de "breaker"
    settings.Name = cfg.Name
}

// Uso:
breakerA := NewBreakerMiddleware(
    transport1,
    WithBreakerName("api-auth-service"),
)

breakerB := NewBreakerMiddleware(
    transport2,
    WithBreakerName("api-payment-service"),
)
```

---

## 5. CONTEXTO E CANCELAMENTO

### 🟡 **PROBLEMA: Ignora req.Context()**

```go
_, err := b.cb.Execute(func() (any, error) {
    r, err := b.next.RoundTrip(req)  // ← Respeita req.Context()
    // MAS: Breaker não verifica context
})
```

**Cenário problemático**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req := req.WithContext(ctx)

// Cliente cancela após 2 segundos
go func() {
    time.Sleep(2*time.Second)
    cancel()
}()

// Breaker em HALF_OPEN:
// - MaxRequests = 3
// - Cada probe leva 10 segundos
// - Client timeout = 5 segundos
// - Context cancela após 2s

Result:
├─ Client gets: context.DeadlineExceeded
├─ Breaker vê: error (context.DeadlineExceeded)
├─ Breaker contabiliza: ConsecutiveFailures++
└─ Injusto: Não foi falha do servidor!
```

### ⚠️ **Mitigações (não são soluções perfeitas)**

**Opção 1: Classificar context errors**
```go
if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
    // Não contar como falha do servidor
    return nil, nil  // ← Deixa resposta vazia, não ideal
}
```

**Opção 2: Usar timeout do breaker, não do cliente**
```go
// MAS: Breaker não pode forçar timeout do cliente
// Cada cliente tem seu próprio contexto
```

### 🎯 **Implementação melhorada**

```go
func (b *breakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    _, err := b.cb.Execute(func() (any, error) {
        r, err := b.next.RoundTrip(req)
        
        // Não contar context errors como falha do servidor
        if errors.Is(err, context.Canceled) {
            return r, nil  // Retornar sem erro
        }
        if errors.Is(err, context.DeadlineExceeded) {
            // Depende: É timeout do CLIENT ou da rede?
            // Heurística: Se req.Context().Done(), é do cliente
            if req.Context().Done() != nil {
                return r, nil
            }
            return r, err  // É timeout de rede
        }
        
        return r, err
    })
    return resp, err
}
```

---

## 6. BODY RESPONSE E LIFECYCLE

### 🟡 **PROBLEMA: Assumir que RoundTrip = Requisição concluída**

```go
_, err := b.cb.Execute(func() (any, error) {
    r, err := b.next.RoundTrip(req)
    if err != nil {
        // Close body
    }
    resp = r
    return resp, nil  // ← Success!
})
return resp, err  // ← Body ainda não foi lido!
```

**Cenário**:
```go
resp, _ := client.Do(req)
// RoundTrip retornou, breaker marcou como "sucesso"

// MAS o cliente ainda não leu o body:
body, err := io.ReadAll(resp.Body)  // ← Pode falhar aqui!

if err != nil {
    // Erro ocorreu DEPOIS do breaker marcar sucesso
    // Breaker não sabe disso
}
```

### ⚠️ **Implicações**

| Etapa | Breaker sabe? |
|-------|--------------|
| TCP Connect | ✓ Sim |
| HTTP Response Headers | ✓ Sim |
| Leitura do body | ❌ Não |
| Processamento do body | ❌ Não |

### 🎯 **Soluções arquiteturais**

**Opção 1: Wrapper de Response (compatível com HTTP semantics)**
```go
type TrackedResponse struct {
    *http.Response
    breaker *breakerRoundTripper
}

func (tr *TrackedResponse) Read(p []byte) (n int, err error) {
    n, err = tr.Response.Body.Read(p)
    if err != nil && err != io.EOF {
        // Reportar falha durante leitura do body
        tr.breaker.OnBodyReadError(err)
    }
    return
}
```

**Opção 2: Aceitar a limitação, documentar**
```go
// Documentação explícita:
// Circuit breaker monitora apenas a conectividade e headers da resposta.
// Erros durante a leitura do body NÃO incrementam o contador de falhas.
// Recomendação: Use circuit breaker + timeout de read.
```

**Opção 3: Usar contexto e watchdog (overhead)**
```go
// Muito complexo para o benefício
// Não recomendado
```

---

## 7. ANÁLISE DA LÓGICA ReadyToTrip

### ✅ PONTOS FORTES

```go
if counts.ConsecutiveFailures < maxFailures || counts.Requests == 0 {
    return false
}

failureRate := float64(counts.TotalFailures) / float64(counts.Requests)
return failureRate >= 0.5
```

**Bem feito**:
- ✅ Early exit se não atingiu threshold de falhas consecutivas
- ✅ Evita divisão por zero (`counts.Requests == 0`)
- ✅ Duplo critério: rajada OU degradação
- ✅ Sem lock manual (gobreaker cuida)

### 🟡 MELHORIAS POSSÍVEIS

**1. Threshold de falhas consecutivas muito alto?**
```
maxFailures = 5 (default)
↓
Esperada 5 falhas seguidas antes de trip
↓
MAS: Se 5 requisições levarem 30 segundos (6s cada)
↓
Usuário vê 30s de latência antes de breaker abrir
```

**Sugestão**: Tornar configurável e menor por default
```go
maxFailures := 3  // Ao invés de 5
```

**2. Taxa de falha de 50% é muito generosa?**
```
Cenário: Servidor degradado
├─ 10 requisições
├─ 5 sucessos, 5 falhas = 50% exatamente
├─ breaker.trip? SIM (failureRate >= 0.5)
└─ OK, mas margem é apertada

Sugestão: Usar 50%+ ou 45%?
```

**3. Sem considerar janela de tempo**
```
Cenário:
├─ Hora 1: 100 requisições, 50 falharam
├─ Breaker abre (50% >= 50%)
├─ Hora 2: Servidor recuperado
├─ Breaker ainda OPEN por 30s
├─ Janela de 30s perdida

Solução: gobreaker já tem timeout, OK.
MAS: Reset de contadores só ao trocar estado.
```

---

## 8. DEFENSIVE PROGRAMMING

### ✅ MUITO BEM FEITO

```go
// 1. Validação de nil
if b.next == nil {
    return nil, errors.New("...")
}
if b.cb == nil {
    return b.next.RoundTrip(req)
}

// 2. Fechamento defensivo de body
if r != nil && r.Body != nil {
    _ = r.Body.Close()
}

// 3. Conversão segura de int → uint32
maxUint32Int := int(^uint32(0))
halfOpen := uint32(min(max(cfg.HalfOpenRequests, 0), maxUint32Int))
```

**Nota crítica**: A conversão defensiva é **exagerada** para o use case. `MaxFailures` e `HalfOpenRequests` nunca serão negativos ou maiores que `int(^uint32(0))`. Mas melhor exagerado que inseguro. ✓

---

## 9. COMPOSIÇÃO DE MIDDLEWARE

### ✅ BEM ARQUITETADO

```go
type breakerRoundTripper struct {
    next http.RoundTripper
    cb   *gobreaker.CircuitBreaker
}

func NewBreakerMiddleware(base http.RoundTripper, opts ...BreakerOption) http.RoundTripper
```

**Vantagens**:
- ✅ Implementa `http.RoundTripper` (composável)
- ✅ Funciona com qualquer `RoundTripper` anterior
- ✅ Padrão idiomático Go
- ✅ Fácil de testar com mock

**Exemplo de composição**:
```go
transport := &http.Transport{}
transport = WithRetry(transport)
transport = WithCircuitBreaker(transport)
transport = WithMetrics(transport)

client := &http.Client{Transport: transport}
```

---

## 10. QUESTÕES ABERTAS ARQUITETURAIS

### ❓ **Múltiplos breakers vs um global?**

**Implementação atual**: Um breaker por cliente
```go
client1 := &http.Client{Transport: NewBreakerMiddleware(http.DefaultTransport)}
client2 := &http.Client{Transport: NewBreakerMiddleware(http.DefaultTransport)}
```

**Problema**: Dois breakers independentes
```
API A falha → Breaker A abre
API B está OK → Breaker B fechado
```

**Alternativa**: Breaker compartilhado por destino
```
API A falha → Breaker para "api-a" abre
Todas as requisições para "api-a" rejeitadas

MAS: Requer pool de breakers
```

### ❓ **Que fazer quando breaker está OPEN?**

**Atual**:
```go
_, err := b.cb.Execute(func() (any, error) { ... })
// Se breaker OPEN: erro "circuit breaker is open"
```

**Alternativas**:
1. Retornar erro como hoje ✓ (atual)
2. Fallback automático
3. Queue requisições
4. Customizar erro

---

## RESUMO DE RECOMENDAÇÕES

| Problema | Severidade | Solução | Esforço |
|----------|-----------|---------|--------|
| Classificação de falhas | 🔴 CRÍTICO | FailureClassifier | Alto |
| Trata 5xx como sucesso | 🔴 CRÍTICO | StatusCodeClassifier | Médio |
| Ignora context | 🟡 MÉDIO | Classificar context errors | Baixo |
| Sem observabilidade | 🟡 MÉDIO | Callbacks de métricas | Médio |
| Nome fixo do breaker | 🟡 MÉDIO | Permitir customização | Baixo |
| Body lifecycle | 🟡 MÉDIO | Documentar limitação | Muito baixo |
| Threshold de falhas | 🟢 BAIXO | Considerar reduzir default | Muito baixo |

---

## CONCLUSÃO

**Atual**: ⭐⭐⭐ (3/5)
- Defensive, limpo, bem composto
- MAS: Classifica erros incorretamente

**Com melhorias**: ⭐⭐⭐⭐⭐ (5/5)
- Enterprise-ready
- Production-proof
- Observable

**Prioridade #1**: Implementar `FailureClassifier` — sem isso, qualquer erro (até do cliente) abre o breaker.
