# Circuit Breaker: Classificação de Erros

## 🎯 Objetivo

Implementar um `FailureClassifier` que **diferencia indisponibilidade real** (incrementa CB) de **erros do cliente ou transitórios** (não incrementa CB).

---

## 📊 Regra Simples

```
if err != nil {
    → INCREMENTA CB (servidor down/indisponível)
}

if resp.StatusCode >= 500 {
    → INCREMENTA CB (servidor degradado)
}

Tudo mais (2xx, 3xx, 4xx) {
    → NÃO incrementa CB
}
```

---

## 💾 Implementação

### 1. Tipos

```go
// pkg/middleware/net_http_client/failure_classifier.go

package net_http_client

import (
    "context"
    "errors"
    "net/http"
)

type ErrorCategory int

const (
    // ErrorCategoryServer: Indisponibilidade (err != nil ou status >= 500)
    ErrorCategoryServer ErrorCategory = iota
    
    // ErrorCategoryClient: Não é indisponibilidade (2xx, 3xx, 4xx)
    ErrorCategoryClient
)

// FailureClassifier decide se um erro incrementa o circuit breaker
type FailureClassifier interface {
    ClassifyError(err error, resp *http.Response, req *http.Request) ErrorCategory
}

// DefaultFailureClassifier implementa a classificação padrão
type DefaultFailureClassifier struct{}

func (c *DefaultFailureClassifier) ClassifyError(
    err error,
    resp *http.Response,
    req *http.Request,
) ErrorCategory {
    
    // Sem erro = sucesso
    if err == nil {
        // HTTP 5xx = servidor degradado
        if resp != nil && resp.StatusCode >= 500 {
            return ErrorCategoryServer
        }
        // 2xx, 3xx, 4xx = não é indisponibilidade
        return ErrorCategoryClient
    }
    
    // Cliente cancelou = não é indisponibilidade do servidor
    if errors.Is(err, context.Canceled) {
        return ErrorCategoryClient
    }
    
    // Cliente atingiu deadline = não é indisponibilidade do servidor
    if errors.Is(err, context.DeadlineExceeded) {
        return ErrorCategoryClient
    }
    
    // Qualquer outro err != nil = indisponibilidade (rede, DNS, conexão, EOF, etc)
    return ErrorCategoryServer
}
```

### 2. Integração no Circuit Breaker

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
        
        // Classificar erro
        category := b.failureClassifier.ClassifyError(err, r, req)
        
        if category != ErrorCategoryServer {
            // Não incrementa CB: fecha body e retorna sem erro
            if err != nil && r != nil && r.Body != nil {
                _ = r.Body.Close()
            }
            return r, nil
        }
        
        // Incrementa CB: retorna erro
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

### 3. Config

```go
// internal/middleware/net_http_client/net_http_circuit_breaker_config.go

type circuitBreakerConfig struct {
    MaxFailures       int
    ResetTimeout      time.Duration
    HalfOpenRequests  int
    FailureClassifier FailureClassifier
}

func WithFailureClassifier(classifier FailureClassifier) BreakerOption {
    return func(cfg *circuitBreakerConfig) {
        if classifier != nil {
            cfg.FailureClassifier = classifier
        }
    }
}
```

---

## 🧪 Testes

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
        // === INCREMENTA CB ===
        {
            name:     "500 Server Error",
            resp:     &http.Response{StatusCode: 500},
            expected: ErrorCategoryServer,
        },
        {
            name:     "503 Service Unavailable",
            resp:     &http.Response{StatusCode: 503},
            expected: ErrorCategoryServer,
        },
        {
            name:     "Connection refused (err != nil)",
            err:      errors.New("connection refused"),
            expected: ErrorCategoryServer,
        },
        {
            name:     "DNS error (err != nil)",
            err:      errors.New("no such host"),
            expected: ErrorCategoryServer,
        },
        {
            name:     "EOF (err != nil)",
            err:      io.EOF,
            expected: ErrorCategoryServer,
        },
        
        // === NÃO INCREMENTA CB ===
        {
            name:     "200 OK",
            resp:     &http.Response{StatusCode: 200},
            expected: ErrorCategoryClient,
        },
        {
            name:     "400 Bad Request",
            resp:     &http.Response{StatusCode: 400},
            expected: ErrorCategoryClient,
        },
        {
            name:     "404 Not Found",
            resp:     &http.Response{StatusCode: 404},
            expected: ErrorCategoryClient,
        },
        {
            name:     "429 Too Many Requests",
            resp:     &http.Response{StatusCode: 429},
            expected: ErrorCategoryClient,
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

## ✅ Checklist

- [ ] Criar `failure_classifier.go` com ErrorCategory + DefaultFailureClassifier
- [ ] Integrar FailureClassifier em breakerRoundTripper
- [ ] Adicionar BreakerOption WithFailureClassifier()
- [ ] Implementar classificação no RoundTrip()
- [ ] Escrever testes unitários
- [ ] Testar com código existente (regressão)
