# Plano de Implementação: Circuit Breaker com Classificação de Erros

## 1. CLASSIFICAÇÃO DE ERROS (Definição Executiva)

### 🔴 **INCREMENTA ConsecutiveFailures** (Indisponibilidade Real)

Estes erros indicam que o **servidor está genuinamente indisponível** ou degradado:

#### HTTP Status Codes
```
500 Internal Server Error      → Erro interno do servidor
502 Bad Gateway                → Gateway/proxy com problema
503 Service Unavailable        → Serviço fora
504 Gateway Timeout            → Timeout no gateway/upstream
```

#### Network Errors (conexão)
```
net.OpError:
  └─ "connection refused"      → Servidor não está escutando
  └─ "connection reset by peer" → Servidor forçou desconexão
  └─ "broken pipe"             → Conexão perdeu
  └─ "no route to host"        → Problema de rede

net.DNSError (não temporário):
  └─ Permanent DNS failure      → Domain não resolve
  └─ NXDOMAIN, SERVFAIL        → Problema de DNS real
```

#### Timeouts de Rede
```
context.Context timeout na leitura/write
  └─ io.ReadDeadlineExceeded    → Timeout lendo resposta
  └─ io.WriteDeadlineExceeded   → Timeout enviando requisição
```

---

### 🟡 **NÃO INCREMENTA** (Não é indisponibilidade)

Estes erros **não indicam indisponibilidade do servidor**:

#### Cancelamento do Cliente
```
context.Canceled
  └─ Cliente cancelou a requisição (Ctrl+C, context.cancel())
  └─ Não é falha do servidor
  └─ Não deve tripar breaker

context.DeadlineExceeded (timeout do cliente, não da rede)
  └─ Cliente definiu timeout muito curto
  └─ Servidor pode estar OK, mas cliente desistiu
  └─ Não deve tripar breaker
```

#### Erros do Cliente (4xx HTTP)
```
400 Bad Request              → Payload inválido (cliente)
401 Unauthorized             → Autenticação falhou (cliente)
402 Payment Required         → Billing issue (cliente)
403 Forbidden                → Permissão negada (cliente)
404 Not Found                → Recurso não existe (não é indisponibilidade)
405 Method Not Allowed       → Método errado (cliente)
406 Not Acceptable           → Formato não aceito (cliente)
409 Conflict                 → Estado conflitante (cliente)
410 Gone                     → Recurso foi deletado (cliente)
422 Unprocessable Entity     → Validação falhou (cliente)
```

#### Rate Limiting (429)
```
429 Too Many Requests
  └─ MAS: Deve ser tratado com RETRY + BACKOFF
  └─ NÃO deve tripar breaker (servidor está OK, só rejeitando)
  └─ Breaker abre APENAS se persistir após backoff
  
Estratégia recomendada:
  1. Requisição recebe 429
  2. Aplicação faz backoff (2s, 4s, 8s...)
  3. Se continuar levando 429 após N tentativas: tripar breaker
  └─ Implementar em retry layer, não no breaker
```

#### DNS Temporário
```
net.DNSError (temporary: true)
  └─ DNS server indisponível momentaneamente
  └─ Provável que se recupere em segundos
  └─ Não deve tripar breaker
  
Estratégia recomendada:
  1. Aplicação faz retry imediato
  2. Se continuar depois de 3 tentativas: considera como server error
```

#### EOF/Broken Connection
```
io.EOF durante leitura do body
  └─ Servidor fechou conexão inesperadamente
  └─ Pode ser transitório (servidor reiniciando)
  └─ Incrementar ConsecutiveFailures, MAS com timeout menor

net.OpError (timeout: true) - timeout DE REDE
  └─ Pode ser congestionamento de rede
  └─ Transitório
  └─ Não deve tripar breaker, usar retry
```

---

## 2. MATRIZ DE DECISÃO

```
┌─────────────────────────────────────────────────────────────────┐
│ ERRO                          │ INCREMENTA CB? │ RECOMENDAÇÃO   │
├─────────────────────────────────────────────────────────────────┤
│ 5xx Server Error              │ ✅ SIM        │ CB + Retry     │
│ 429 Too Many Requests         │ ❌ NÃO        │ Retry + Backoff│
│ 4xx Client Error              │ ❌ NÃO        │ Fail fast      │
│ Connection refused            │ ✅ SIM        │ CB + Retry     │
│ DNS error (temp)              │ ❌ NÃO        │ Retry          │
│ DNS error (perm)              │ ✅ SIM        │ CB + Fail      │
│ context.Canceled              │ ❌ NÃO        │ Fail immediate │
│ context.DeadlineExceeded      │ ⚠️ DEPENDE    │ Ver abaixo     │
│ Timeout de rede (write/read)  │ ❌ NÃO        │ Retry          │
│ EOF inesperado                │ ✅ SIM        │ CB + Retry     │
│ Connection reset              │ ✅ SIM        │ CB + Retry     │
│ Broken pipe                   │ ✅ SIM        │ CB + Retry     │
└─────────────────────────────────────────────────────────────────┘
```

### Nota: context.DeadlineExceeded
```
Caso A: Client timeout (req.Context().Deadline())
  └─ Cliente definiu deadline muito curto
  └─ Não é erro do servidor
  └─ ❌ NÃO incrementa CB

Caso B: Network timeout (TCP read/write timeout)
  └─ Servidor não respondeu a tempo
  └─ Pode indicar indisponibilidade
  └─ ⚠️ Incrementa CB com hesitação
  └─ Implementação: Checar se é timeout de cliente vs rede

Heurística simples:
  if errors.Is(err, context.DeadlineExceeded) && req.Context().Done() != nil {
      // É timeout do cliente, não incrementa
  } else if errors.Is(err, context.DeadlineExceeded) {
      // É timeout de rede, incrementa
  }
```

---

## 3. IMPLEMENTAÇÃO: Interface de Classificação

### Passo 1: Definir tipos

```go
// pkg/middleware/net_http_client/failure_classifier.go

package net_http_client

import (
    "context"
    "errors"
    "fmt"
    "net"
    "net/http"
)

// ErrorCategory define como classificar um erro para o circuit breaker
type ErrorCategory int

const (
    // ErrorCategoryServer: Erro do servidor, incrementa circuit breaker
    // Exemplos: 5xx, connection refused, DNS permanente
    ErrorCategoryServer ErrorCategory = iota
    
    // ErrorCategoryClient: Erro do cliente, NÃO incrementa circuit breaker
    // Exemplos: 4xx, context.Canceled, timeout do cliente
    ErrorCategoryClient
    
    // ErrorCategoryTransient: Erro temporário, NÃO incrementa circuit breaker
    // Mas aplicação deve fazer retry automaticamente
    // Exemplos: 429, DNS temporário, network timeout
    ErrorCategoryTransient
)

// FailureClassifier classifica se um erro deve incrementar o circuit breaker
type FailureClassifier interface {
    ClassifyError(err error, resp *http.Response, req *http.Request) ErrorCategory
}

// DefaultFailureClassifier implementa a lógica de classificação padrão
type DefaultFailureClassifier struct{}

func (c *DefaultFailureClassifier) ClassifyError(
    err error,
    resp *http.Response,
    req *http.Request,
) ErrorCategory {
    
    // Sem erro = sucesso
    if err == nil {
        if resp != nil && resp.StatusCode >= 500 {
            return ErrorCategoryServer
        }
        return ErrorCategoryClient  // 2xx, 3xx, 4xx = sucesso ou erro client
    }
    
    // === CANCELAMENTO DO CLIENTE ===
    if errors.Is(err, context.Canceled) {
        return ErrorCategoryClient
    }
    
    // === DEADLINE EXCEEDED ===
    if errors.Is(err, context.DeadlineExceeded) {
        // Heurística: É timeout do cliente ou da rede?
        if req != nil && req.Context().Err() == context.DeadlineExceeded {
            // É timeout do cliente, não incrementa CB
            return ErrorCategoryClient
        }
        // É timeout de rede, incrementa CB
        return ErrorCategoryServer
    }
    
    // === ERROS DE REDE ===
    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        if dnsErr.IsTemporary || dnsErr.IsTimeout {
            // DNS temporário: retry, não CB
            return ErrorCategoryTransient
        }
        // DNS permanente: tripar CB
        return ErrorCategoryServer
    }
    
    var opErr *net.OpError
    if errors.As(err, &opErr) {
        switch opErr.Op {
        case "dial":
            // Connection refused, no route to host, etc
            return ErrorCategoryServer
        case "read", "write":
            if opErr.Timeout() {
                // Network timeout: retry, não CB
                return ErrorCategoryTransient
            }
            // Outros erros de read/write: server error
            return ErrorCategoryServer
        }
    }
    
    // === URL/PARSING ERRORS ===
    if _, ok := err.(*url.Error); ok {
        // Erro ao parsear URL ou conectar: server
        return ErrorCategoryServer
    }
    
    // === TLS/SSL ERRORS ===
    // TLS handshake failure: possível que servidor não suporte a versão
    // Mas geralmente é problema do cliente (versão TLS antiga)
    // Classificação: Server (pode ser certificado inválido no servidor)
    if errors.As(err, &tls.Error{}) {
        return ErrorCategoryServer
    }
    
    // === IO ERRORS ===
    if errors.Is(err, io.EOF) {
        // Servidor fechou conexão: pode ser indisponibilidade
        return ErrorCategoryServer
    }
    
    // === DEFAULT ===
    // Qualquer outro erro desconhecido: conservador = server
    return ErrorCategoryServer
}
```

### Passo 2: Integrar no Circuit Breaker

```go
// internal/middleware/net_http_client/net_http_circuit_breaker.go

type breakerRoundTripper struct {
    next                http.RoundTripper
    cb                  *gobreaker.CircuitBreaker
    failureClassifier   FailureClassifier
}

func NewBreakerMiddleware(
    base http.RoundTripper,
    opts ...BreakerOption,
) http.RoundTripper {
    if base == nil {
        base = http.DefaultTransport
    }
    
    cfg := newCircuitBreakerConfig(opts...)
    
    // Se não forneceu classifier, usar default
    if cfg.FailureClassifier == nil {
        cfg.FailureClassifier = &DefaultFailureClassifier{}
    }
    
    // ... resto do código ...
    
    return &breakerRoundTripper{
        next:              base,
        cb:                gobreaker.NewCircuitBreaker(settings),
        failureClassifier: cfg.FailureClassifier,
    }
}

func (b *breakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    if b.next == nil {
        return nil, errors.New("breaker middleware: next RoundTripper is nil")
    }
    if b.cb == nil {
        return b.next.RoundTrip(req)
    }
    
    var resp *http.Response
    _, err := b.cb.Execute(func() (any, error) {
        r, err := b.next.RoundTrip(req)
        
        // === CLASSIFICAR ERRO ===
        category := b.failureClassifier.ClassifyError(err, r, req)
        
        switch category {
        case ErrorCategoryServer:
            // Incrementa circuit breaker: verdadeira indisponibilidade
            if r != nil && r.Body != nil {
                _ = r.Body.Close()
            }
            if err != nil {
                return nil, err
            }
            // Se é 5xx sem erro, converte em erro
            return nil, fmt.Errorf("server error: %d", r.StatusCode)
            
        case ErrorCategoryClient:
            // Não incrementa circuit breaker: erro do cliente
            // Fecha body se houve erro
            if err != nil && r != nil && r.Body != nil {
                _ = r.Body.Close()
            }
            // Retorna nil para não incrementar breaker
            if err != nil {
                return nil, nil  // Erro, mas não conta para CB
            }
            return r, nil
            
        case ErrorCategoryTransient:
            // Não incrementa circuit breaker: erro transitório
            // Aplicação vai fazer retry em camada superior
            if err != nil && r != nil && r.Body != nil {
                _ = r.Body.Close()
            }
            if err != nil {
                return nil, nil  // Erro, mas não conta para CB
            }
            // Se é 429, não incrementa CB
            return r, nil
        }
        
        return nil, err
    })
    
    return resp, err
}
```

### Passo 3: Adicionar Option para Custom Classifier

```go
// BreakerOption para permitir custom classifier
func WithFailureClassifier(classifier FailureClassifier) BreakerOption {
    return func(cfg *circuitBreakerConfig) {
        if classifier != nil {
            cfg.FailureClassifier = classifier
        }
    }
}

// Uso:
customClassifier := &MyCustomFailureClassifier{}
breaker := NewBreakerMiddleware(
    transport,
    WithFailureClassifier(customClassifier),
    WithCircuitBreakerMaxFailures(5),
)
```

---

## 4. IMPLEMENTAÇÃO SIMPLIFICADA

### Mudança no RoundTrip

```go
func (b *breakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    if b.next == nil {
        return nil, errors.New("breaker middleware: next RoundTripper is nil")
    }
    if b.cb == nil {
        return b.next.RoundTrip(req)
    }
    
    var resp *http.Response
    _, err := b.cb.Execute(func() (any, error) {
        r, err := b.next.RoundTrip(req)
        
        // Classificar erro
        if b.failureClassifier.ClassifyError(err, r, req) != ErrorCategoryServer {
            // Não é indisponibilidade real: fecha body e não incrementa CB
            if err != nil && r != nil && r.Body != nil {
                _ = r.Body.Close()
            }
            // Retorna nil para não incrementar breaker
            return r, nil
        }
        
        // É indisponibilidade real: incrementa CB
        if err != nil {
            if r != nil && r.Body != nil {
                _ = r.Body.Close()
            }
            return nil, err
        }
        
        resp = r
        return resp, nil
    })
    return resp, err
}
```

---

## 5. DETALHES DA IMPLEMENTAÇÃO

### Imports necessários
```go
import (
    "crypto/tls"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "context"
)
```

### Config struct
```go
type circuitBreakerConfig struct {
    MaxFailures       int
    ResetTimeout      time.Duration
    HalfOpenRequests  int
    FailureClassifier FailureClassifier  // ← NOVO
}

func newCircuitBreakerConfig(opts ...BreakerOption) circuitBreakerConfig {
    cfg := circuitBreakerConfig{
        MaxFailures:       5,
        ResetTimeout:      10 * time.Second,
        HalfOpenRequests:   3,
        FailureClassifier: &DefaultFailureClassifier{},  // ← DEFAULT
    }
    
    for _, opt := range opts {
        if opt != nil {
            opt(&cfg)
        }
    }
    
    return cfg
}
```

---

## 5. CASOS DE TESTE

```go
// internal/middleware/net_http_client/failure_classifier_test.go

func TestFailureClassifier(t *testing.T) {
    classifier := &DefaultFailureClassifier{}
    
    tests := []struct {
        name     string
        err      error
        resp     *http.Response
        expected ErrorCategory
    }{
        {
            name:     "500 Server Error",
            resp:     &http.Response{StatusCode: 500},
            expected: ErrorCategoryServer,
        },
        {
            name:     "400 Bad Request",
            resp:     &http.Response{StatusCode: 400},
            expected: ErrorCategoryClient,
        },
        {
            name:     "429 Too Many Requests",
            resp:     &http.Response{StatusCode: 429},
            expected: ErrorCategoryClient,  // Não incrementa CB
        },
        {
            name:     "Connection refused",
            err:      &net.OpError{Op: "dial", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)},
            expected: ErrorCategoryServer,
        },
        {
            name:     "Context canceled",
            err:      context.Canceled,
            expected: ErrorCategoryClient,
        },
        {
            name:     "DNS temporary",
            err:      &net.DNSError{IsTemporary: true},
            expected: ErrorCategoryTransient,
        },
        {
            name:     "EOF",
            err:      io.EOF,
            expected: ErrorCategoryServer,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := classifier.ClassifyError(tt.err, tt.resp, nil)
            if got != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, got)
            }
        })
    }
}
```

---

## 6. IMPACTO NA LÓGICA ReadyToTrip

### Antes (atual)
```go
ReadyToTrip: func(counts gobreaker.Counts) bool {
    // Qualquer erro conta: 4xx, 5xx, client error, tudo igual
    if counts.ConsecutiveFailures < maxFailures || counts.Requests == 0 {
        return false
    }
    failureRate := float64(counts.TotalFailures) / float64(counts.Requests)
    return failureRate >= 0.5
}
```

### Depois (com classificação)
```go
ReadyToTrip: func(counts gobreaker.Counts) bool {
    // Apenas erros ErrorCategoryServer contam
    // ErrorCategoryClient e ErrorCategoryTransient retornam nil do Execute()
    
    // Lógica mantém igual, mas apenas conta erros de verdadeira indisponibilidade
    if counts.ConsecutiveFailures < maxFailures || counts.Requests == 0 {
        return false
    }
    failureRate := float64(counts.TotalFailures) / float64(counts.Requests)
    return failureRate >= 0.5
}
```

---

## 7. EXEMPLOS DE COMPORTAMENTO

### Exemplo 1: Cliente envia JSON inválido

```
Antes (PROBLEMA):
├─ POST /api/users {"email": "invalid"}
├─ Resposta: 400 Bad Request
├─ Breaker: ConsecutiveFailures++
├─ Após 5 tentativas ruins: CB abre
└─ Breaker está ABERTO, cliente não consegue nem tentar corrigir

Depois (CORRETO):
├─ POST /api/users {"email": "invalid"}
├─ Resposta: 400 Bad Request
├─ Classifier: ErrorCategoryClient
├─ Breaker: NÃO incrementa (Execute retorna nil do erro)
├─ CB permanece CLOSED
└─ Cliente pode tentar novamente com JSON correto
```

### Exemplo 2: Servidor retorna 503

```
Antes:
├─ GET /api/health
├─ Resposta: 503 Service Unavailable
├─ Breaker: Sem erro HTTP, mas status 503
├─ Depende se código cliente tratar como erro

Depois (GARANTIDO):
├─ GET /api/health
├─ Resposta: 503 Service Unavailable
├─ Classifier: ErrorCategoryServer
├─ Breaker: ConsecutiveFailures++
├─ Após 5: CB abre
└─ Protege backend de sobrecarga
```

### Exemplo 3: DNS temporário

```
Antes (PROBLEMA):
├─ net.DNSError (temporary=true)
├─ Breaker: ConsecutiveFailures++
├─ Após 5 tentativas: CB abre
├─ DNS se recupera em 2s, mas CB fica aberto por 10s

Depois (CORRETO):
├─ net.DNSError (temporary=true)
├─ Classifier: ErrorCategoryTransient
├─ Breaker: NÃO incrementa
├─ Aplicação faz retry imediato
├─ DNS se recupera: requisição passa
└─ CB nunca abriu desnecessariamente
```

---

## 8. ROADMAP DE IMPLEMENTAÇÃO

### Fase 1: Core (Semana 1)
- [ ] Definir ErrorCategory enum
- [ ] Implementar DefaultFailureClassifier
- [ ] Integrar classifier no breakerRoundTripper.RoundTrip()
- [ ] Adicionar BreakerOption WithFailureClassifier()
- [ ] Escrever testes unitários

### Fase 2: Observabilidade (Semana 2)
- [ ] Adicionar callbacks OnStateChange()
- [ ] Adicionar métricas (PrometheusMetrics ou similar)
- [ ] Logs de transições de estado

### Fase 3: Production (Semana 3)
- [ ] Permitir nome customizável do breaker
- [ ] Documentação completa
- [ ] Migration guide para código existente
- [ ] Exemplos de uso

---

## 9. MIGRATION GUIDE

Para código existente:

```go
// Antes:
breaker := NewBreakerMiddleware(transport)

// Depois (comportamento igual se usar defaults):
breaker := NewBreakerMiddleware(transport)

// Ou com custom classifier:
breaker := NewBreakerMiddleware(
    transport,
    WithFailureClassifier(&MyClassifier{}),
)
```

**Mudança principal**: Erros 4xx e transitórios **não incrementam mais o circuit breaker**. Isso é **bug fix**, não breaking change.

---

## 10. RISCOS E MITIGAÇÕES

| Risco | Mitigação |
|-------|-----------|
| Novo ErrorCategory pode ser ambíguo | Documentação clara com exemplos |
| Custom classifier retornar categoria errada | Testes, validação em testes |
| Comportamento diferente muda SLA | Monitorar métricas antes/depois |
| Alguém classifica 5xx como Client | Code review, defaults sensatos |

---

## CONCLUSÃO

Com essa implementação:

✅ **5xx real** → CB abre (proteção)
✅ **4xx cliente** → CB não mexe (não é indisponibilidade)
✅ **429 rate limit** → Deixa pra retry layer
✅ **DNS temp** → Não abre CB
✅ **Client timeout** → Não conta como erro

**Resultado**: Circuit breaker funciona como deve — **protegendo contra indisponibilidade real**, não contra qualquer erro.
