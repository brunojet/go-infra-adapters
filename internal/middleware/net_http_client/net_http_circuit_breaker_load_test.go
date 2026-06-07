package net_http_client

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestCircuitBreakerUnderLoad simula alta carga com múltiplas goroutines
func TestCircuitBreakerUnderLoad(t *testing.T) {
	t.Run("Load Test: 100 goroutines, 1000 requisições", func(t *testing.T) {
		stats := &loadTestStats{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simula latência variável
			time.Sleep(time.Duration(10) * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		breaker := NewBreakerMiddleware(nil,
			WithCircuitBreakerMaxFailures(20),
			WithCircuitBreakerResetTimeout(2*time.Second),
			WithCircuitBreakerHalfOpenRequests(10),
		)

		client := &http.Client{Transport: breaker}

		// 100 goroutines fazendo requisições
		numWorkers := 100
		refsPerWorker := 10

		var wg sync.WaitGroup
		start := time.Now()

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for i := 0; i < refsPerWorker; i++ {
					resp, err := client.Get(server.URL)
					if err != nil {
						atomic.AddInt64(&stats.errors, 1)
						t.Logf("Worker %d error: %v", workerID, err)
					} else {
						atomic.AddInt64(&stats.success, 1)
						_ = resp.Body.Close()
					}
				}
			}(w)
		}

		wg.Wait()
		elapsed := time.Since(start)

		total := stats.success + stats.errors
		successRate := float64(stats.success) / float64(total) * 100

		t.Logf("\n=== LOAD TEST RESULTS ===")
		t.Logf("Total Requests: %d", total)
		t.Logf("Successful: %d (%.2f%%)", stats.success, successRate)
		t.Logf("Errors: %d", stats.errors)
		t.Logf("Duration: %v", elapsed)
		t.Logf("Throughput: %.2f req/s", float64(total)/elapsed.Seconds())

		if stats.errors == 0 {
			t.Log("✅ All requests succeeded under load")
		}
	})
}

// TestCircuitBreakerDegradedServer simula servidor degradado
func TestCircuitBreakerDegradedServer(t *testing.T) {
	t.Run("Degraded Server: Taxa de erro 60% com spike", func(t *testing.T) {
		stats := &loadTestStats{}
		requestCount := atomic.Int64{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := requestCount.Add(1)

			// Primeiros 300 requisições: 60% falham (degradação)
			// Depois: servidor recupera
			if count <= 300 {
				// 60% falha
				if count%10 < 6 {
					w.WriteHeader(http.StatusServiceUnavailable)
					atomic.AddInt64(&stats.errors, 1)
					return
				}
			}

			w.WriteHeader(http.StatusOK)
			atomic.AddInt64(&stats.success, 1)
		}))
		defer server.Close()

		breaker := NewBreakerMiddleware(nil,
			WithCircuitBreakerMaxFailures(10),
			WithCircuitBreakerResetTimeout(1*time.Second),
			WithCircuitBreakerHalfOpenRequests(5),
		)

		client := &http.Client{Transport: breaker}

		// 50 goroutines x 10 requisições = 500 total
		numWorkers := 50
		refsPerWorker := 10

		var wg sync.WaitGroup
		start := time.Now()

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for i := 0; i < refsPerWorker; i++ {
					resp, err := client.Get(server.URL)
					if err != nil {
						// Circuit breaker pode estar rejeitando
						atomic.AddInt64(&stats.circuitOpen, 1)
						return
					}
					_ = resp.Body.Close()
				}
			}(w)
		}

		wg.Wait()
		elapsed := time.Since(start)

		total := requestCount.Load()
		successRate := float64(stats.success) / float64(total) * 100

		t.Logf("\n=== DEGRADED SERVER TEST ===")
		t.Logf("Total Requests: %d", total)
		t.Logf("Successful: %d (%.2f%%)", stats.success, successRate)
		t.Logf("Failures: %d", stats.errors)
		t.Logf("Circuit Open Rejections: %d", stats.circuitOpen)
		t.Logf("Duration: %v", elapsed)

		if stats.circuitOpen > 0 {
			t.Log("✅ Circuit breaker opened during degradation (protected backend)")
		}
	})
}

// TestCircuitBreakerRajada simula picos de tráfego
func TestCircuitBreakerRajada(t *testing.T) {
	t.Run("Traffic Spike: 500 requisições simultâneas", func(t *testing.T) {
		stats := &loadTestStats{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		breaker := NewBreakerMiddleware(nil,
			WithCircuitBreakerMaxFailures(50),
			WithCircuitBreakerResetTimeout(1*time.Second),
			WithCircuitBreakerHalfOpenRequests(20),
		)

		client := &http.Client{Transport: breaker}

		// 500 goroutines simultâneas (pico)
		numWorkers := 500

		var wg sync.WaitGroup
		start := time.Now()

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, err := client.Get(server.URL)
				if err != nil {
					atomic.AddInt64(&stats.errors, 1)
				} else {
					atomic.AddInt64(&stats.success, 1)
					_ = resp.Body.Close()
				}
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)

		total := stats.success + stats.errors
		successRate := float64(stats.success) / float64(total) * 100

		t.Logf("\n=== TRAFFIC SPIKE TEST ===")
		t.Logf("Concurrent Requests: %d", numWorkers)
		t.Logf("Successful: %d (%.2f%%)", stats.success, successRate)
		t.Logf("Errors: %d", stats.errors)
		t.Logf("Duration: %v", elapsed)
		t.Logf("Throughput: %.2f req/s", float64(total)/elapsed.Seconds())

		if stats.errors == 0 {
			t.Log("✅ Handled traffic spike without errors")
		}
	})
}

// TestCircuitBreakerRecoveryPattern simula falha -> recuperação -> nova falha
func TestCircuitBreakerRecoveryPattern(t *testing.T) {
	t.Run("Recovery Pattern: Fail -> Open -> Recover -> Fail", func(t *testing.T) {
		phase := atomic.Int32{}
		requestCount := atomic.Int64{}
		stats := &loadTestStats{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := requestCount.Add(1)
			p := phase.Load()

			switch p {
			case 0: // Fase 1: 100 requisições, 40% falham
				if count%10 < 4 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
			case 1: // Fase 2: 100 requisições, todas sucesso (recupera)
				// OK
			case 2: // Fase 3: 100 requisições, 70% falham
				if count%10 < 7 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		breaker := NewBreakerMiddleware(nil,
			WithCircuitBreakerMaxFailures(15),
			WithCircuitBreakerResetTimeout(500*time.Millisecond),
			WithCircuitBreakerHalfOpenRequests(5),
		)

		client := &http.Client{Transport: breaker}

		// Fase 1: Degradação (40% erro)
		t.Log("Phase 1: Degradation (40% error rate)")
		phase.Store(0)
		for i := 0; i < 100; i++ {
			resp, err := client.Get(server.URL)
			if err == nil {
				_ = resp.Body.Close()
			}
		}

		// Aguarda recuperação do circuito
		time.Sleep(700 * time.Millisecond)

		// Fase 2: Recuperação (0% erro)
		t.Log("Phase 2: Recovery (0% error rate)")
		phase.Store(1)
		for i := 0; i < 100; i++ {
			resp, err := client.Get(server.URL)
			if err == nil {
				atomic.AddInt64(&stats.success, 1)
				_ = resp.Body.Close()
			} else {
				atomic.AddInt64(&stats.errors, 1)
			}
		}

		// Aguarda recuperação novamente
		time.Sleep(700 * time.Millisecond)

		// Fase 3: Nova degradação (70% erro)
		t.Log("Phase 3: New degradation (70% error rate)")
		phase.Store(2)
		for i := 0; i < 100; i++ {
			resp, err := client.Get(server.URL)
			if err == nil {
				_ = resp.Body.Close()
			}
		}

		t.Logf("\n=== RECOVERY PATTERN TEST ===")
		t.Logf("Phase 2 (Recovery) Success Rate: %d/100", stats.success)
		t.Logf("Phase 2 (Recovery) Errors: %d", stats.errors)

		if stats.success > 90 {
			t.Log("✅ Recovered successfully during phase 2")
		}
	})
}

// TestCircuitBreakerConcurrentAccess testa thread-safety
func TestCircuitBreakerConcurrentAccess(t *testing.T) {
	t.Run("Thread Safety: 1000 concurrent requests", func(t *testing.T) {
		stats := &loadTestStats{}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		breaker := NewBreakerMiddleware(nil,
			WithCircuitBreakerMaxFailures(100),
			WithCircuitBreakerResetTimeout(2*time.Second),
			WithCircuitBreakerHalfOpenRequests(50),
		)

		client := &http.Client{Transport: breaker}

		// 1000 goroutines simultâneas
		numGoroutines := 1000
		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				resp, err := client.Get(server.URL)
				if err != nil {
					atomic.AddInt64(&stats.errors, 1)
					t.Logf("Goroutine %d error: %v", id, err)
				} else {
					atomic.AddInt64(&stats.success, 1)
					_ = resp.Body.Close()
				}
			}(i)
		}

		wg.Wait()
		elapsed := time.Since(start)

		total := stats.success + stats.errors
		successRate := float64(stats.success) / float64(total) * 100

		t.Logf("\n=== CONCURRENCY TEST ===")
		t.Logf("Goroutines: %d", numGoroutines)
		t.Logf("Successful: %d (%.2f%%)", stats.success, successRate)
		t.Logf("Errors: %d", stats.errors)
		t.Logf("Duration: %v", elapsed)
		t.Logf("Throughput: %.2f req/s", float64(total)/elapsed.Seconds())

		if stats.errors == 0 {
			t.Log("✅ Thread-safe under 1000 concurrent requests")
		}
	})
}

// ============ Helper Types ============

type loadTestStats struct {
	success     int64
	errors      int64
	circuitOpen int64
}
