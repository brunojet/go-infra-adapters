package chain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/local"
	pkglogger "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

type mockCache[T any] struct {
	mu        sync.Mutex
	data      map[string]*T
	ttls      map[string]time.Duration
	hits      int
	misses    int
	setErr    error
	getErr    error
	existsErr error
	healthErr error
	delErr    error
}

func newMockCache[T any]() *mockCache[T] {
	return &mockCache[T]{
		data: make(map[string]*T),
		ttls: make(map[string]time.Duration),
	}
}

func (m *mockCache[T]) Get(_ context.Context, key string) (*T, bool, error) {
	if m.getErr != nil {
		return nil, false, m.getErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.data[key]
	if ok {
		m.hits++
	} else {
		m.misses++
	}
	return val, ok, nil
}

func (m *mockCache[T]) Set(_ context.Context, key string, val *T, ttl time.Duration) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	m.ttls[key] = ttl
	return nil
}

func (m *mockCache[T]) Delete(_ context.Context, key string) error {
	if m.delErr != nil {
		return m.delErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockCache[T]) Exists(_ context.Context, key string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockCache[T]) HealthCheck(_ context.Context) error {
	return m.healthErr
}

// spyLogger records Warn calls so tests can assert on error-logging paths
// without depending on log output formatting.
type spyLogger struct {
	warnCalls int64
}

func (s *spyLogger) Debug(context.Context, string, ...pkglogger.Field) {}
func (s *spyLogger) Info(context.Context, string, ...pkglogger.Field)  {}
func (s *spyLogger) Warn(context.Context, string, ...pkglogger.Field) {
	atomic.AddInt64(&s.warnCalls, 1)
}
func (s *spyLogger) Error(context.Context, string, error, ...pkglogger.Field) {}

func (s *spyLogger) calls() int64 { return atomic.LoadInt64(&s.warnCalls) }

func TestGetOrSet_Hit(t *testing.T) {
	tier1 := newMockCache[string]()
	cached := "cached"
	if err := tier1.Set(context.Background(), "key", &cached, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1},
		),
	)

	var loadCalls int
	got, err := chain.GetOrSet(context.Background(), "key", 5*time.Minute, func(ctx context.Context) (*string, error) {
		loadCalls++
		return nil, errors.New("should not be called")
	})

	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if *got != "cached" {
		t.Fatalf("expected 'cached', got %q", *got)
	}
	if loadCalls != 0 {
		t.Fatalf("expected 0 load calls for hit, got %d", loadCalls)
	}
}

func TestGetOrSet_Miss_SingleLoad(t *testing.T) {
	tier1 := newMockCache[string]()
	tier2 := newMockCache[string]()

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1, TTL: 5 * time.Minute},
			Layer[string]{Name: "tier2", Cache: tier2, TTL: 1 * time.Hour},
		),
	)

	var loadCalls int
	expected := "loaded"
	got, err := chain.GetOrSet(context.Background(), "key", 5*time.Minute, func(ctx context.Context) (*string, error) {
		loadCalls++
		return &expected, nil
	})

	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if *got != expected {
		t.Fatalf("expected %q, got %q", expected, *got)
	}
	if loadCalls != 1 {
		t.Fatalf("expected 1 load call for miss, got %d", loadCalls)
	}

	// Verify both tiers were populated asynchronously
	time.Sleep(50 * time.Millisecond)
	_, hit, _ := tier1.Get(context.Background(), "key")
	if !hit {
		t.Fatal("expected tier1 to be populated")
	}
	_, hit, _ = tier2.Get(context.Background(), "key")
	if !hit {
		t.Fatal("expected tier2 to be populated")
	}
}

func TestGetOrSet_Dedup(t *testing.T) {
	tier1 := newMockCache[int]()
	chain := NewChainedCache[int](
		WithLayers(
			Layer[int]{Name: "tier1", Cache: tier1},
		),
	)

	n := 10
	var loadCalls int64
	results := make([]*int, n)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			val, err := chain.GetOrSet(context.Background(), "key", 5*time.Minute, func(ctx context.Context) (*int, error) {
				atomic.AddInt64(&loadCalls, 1)
				time.Sleep(10 * time.Millisecond) // simulate slow load
				v := 42
				return &v, nil
			})
			if err != nil {
				t.Errorf("GetOrSet: %v", err)
			}
			mu.Lock()
			results[idx] = val
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if loadCalls != 1 {
		t.Fatalf("expected 1 load call for concurrent GetOrSet on same key, got %d", loadCalls)
	}

	for i, r := range results {
		if r == nil || *r != 42 {
			t.Fatalf("result[%d]: expected 42, got %v", i, r)
		}
	}
}

func TestGetOrSet_LoadError(t *testing.T) {
	tier1 := newMockCache[string]()
	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1},
		),
	)

	testErr := errors.New("load error")
	_, err := chain.GetOrSet(context.Background(), "key", 5*time.Minute, func(ctx context.Context) (*string, error) {
		return nil, testErr
	})

	if err != testErr {
		t.Fatalf("expected %v, got %v", testErr, err)
	}

	_, hit, _ := tier1.Get(context.Background(), "key")
	if hit {
		t.Fatal("expected miss after load error")
	}
}

func TestGet_TierPriority(t *testing.T) {
	tier1 := newMockCache[string]()
	tier2 := newMockCache[string]()

	// Populate only tier2
	val2 := "from_tier2"
	if err := tier2.Set(context.Background(), "key", &val2, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1, TTL: 5 * time.Minute},
			Layer[string]{Name: "tier2", Cache: tier2, TTL: 1 * time.Hour},
		),
	)

	got, hit, err := chain.Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected hit in tier2")
	}
	if *got != val2 {
		t.Fatalf("expected %q, got %q", val2, *got)
	}

	// Verify tier1 was warmed up asynchronously
	time.Sleep(50 * time.Millisecond)
	_, hit, _ = tier1.Get(context.Background(), "key")
	if !hit {
		t.Fatal("expected tier1 to be warmed up")
	}
}

func TestDelete_AllLayers(t *testing.T) {
	tier1 := newMockCache[string]()
	tier2 := newMockCache[string]()

	val := "x"
	if err := tier1.Set(context.Background(), "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := tier2.Set(context.Background(), "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1, TTL: 5 * time.Minute},
			Layer[string]{Name: "tier2", Cache: tier2, TTL: 1 * time.Hour},
		),
	)

	err := chain.Delete(context.Background(), "key")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	time.Sleep(50 * time.Millisecond) // wait for async delete

	_, hit, _ := tier1.Get(context.Background(), "key")
	if hit {
		t.Fatal("expected tier1 miss after delete")
	}
	_, hit, _ = tier2.Get(context.Background(), "key")
	if hit {
		t.Fatal("expected tier2 miss after delete")
	}
}

func TestHealthCheck_AllLayers(t *testing.T) {
	tier1 := newMockCache[string]()
	tier2 := newMockCache[string]()

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1, TTL: 5 * time.Minute},
			Layer[string]{Name: "tier2", Cache: tier2, TTL: 1 * time.Hour},
		),
	)

	err := chain.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
}

func TestWith_RealLocalCaches(t *testing.T) {
	tier1 := local.NewCache[string]()
	tier2 := local.NewCache[string]()

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "local1", Cache: tier1, TTL: 5 * time.Minute},
			Layer[string]{Name: "local2", Cache: tier2, TTL: 1 * time.Hour},
		),
	)

	ctx := context.Background()
	var loadCalls int
	expected := "value"

	// Miss in both tiers
	got, err := chain.GetOrSet(ctx, "key", 5*time.Minute, func(ctx context.Context) (*string, error) {
		loadCalls++
		return &expected, nil
	})

	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if *got != expected {
		t.Fatalf("expected %q, got %q", expected, *got)
	}
	if loadCalls != 1 {
		t.Fatalf("expected 1 load call, got %d", loadCalls)
	}

	// Wait for async population
	time.Sleep(100 * time.Millisecond)

	// Next GetOrSet should hit tier1 (no load)
	loadCalls = 0
	_, err = chain.GetOrSet(ctx, "key", 5*time.Minute, func(ctx context.Context) (*string, error) {
		loadCalls++
		return nil, errors.New("should not be called")
	})

	if err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if loadCalls != 0 {
		t.Fatalf("expected 0 load calls on tier1 hit, got %d", loadCalls)
	}
}

func TestPanic_EmptyLayers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty layers")
		}
	}()

	NewChainedCache[string]() // no WithLayers option
	t.Fatal("should have panicked")
}

func TestSet_AllLayers(t *testing.T) {
	tier1 := newMockCache[string]()
	tier2 := newMockCache[string]()
	spy := &spyLogger{}

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1, TTL: 5 * time.Minute},
			Layer[string]{Name: "tier2", Cache: tier2, TTL: 1 * time.Hour},
		),
		WithLogger[string](spy),
	)

	val := "x"
	if err := chain.Set(context.Background(), "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if _, hit, _ := tier1.Get(context.Background(), "key"); !hit {
		t.Fatal("expected tier1 to be populated")
	}
	if _, hit, _ := tier2.Get(context.Background(), "key"); !hit {
		t.Fatal("expected tier2 to be populated")
	}
}

func TestSet_LayerErrorLoggedNotPropagated(t *testing.T) {
	tier1 := newMockCache[string]()
	tier1.setErr = errors.New("write failed")
	spy := &spyLogger{}

	chain := NewChainedCache[string](
		WithLayers(Layer[string]{Name: "tier1", Cache: tier1}),
		WithLogger[string](spy),
	)

	val := "x"
	if err := chain.Set(context.Background(), "key", &val, 0); err != nil {
		t.Fatalf("expected Set to swallow layer errors, got %v", err)
	}
	if spy.calls() == 0 {
		t.Fatal("expected the layer error to be logged")
	}
}

func TestExists_HitAndMiss(t *testing.T) {
	tier1 := newMockCache[string]()
	chain := NewChainedCache[string](
		WithLayers(Layer[string]{Name: "tier1", Cache: tier1}),
	)

	exists, err := chain.Exists(context.Background(), "absent")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("expected miss")
	}

	val := "x"
	if err := tier1.Set(context.Background(), "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	exists, err = chain.Exists(context.Background(), "key")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected hit")
	}
}

func TestExists_LayerErrorContinues(t *testing.T) {
	tier1 := newMockCache[string]()
	tier1.existsErr = errors.New("backend down")
	tier2 := newMockCache[string]()

	val := "x"
	if err := tier2.Set(context.Background(), "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1},
			Layer[string]{Name: "tier2", Cache: tier2},
		),
		WithLogger[string](&spyLogger{}),
	)

	exists, err := chain.Exists(context.Background(), "key")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected tier2 hit despite tier1 error")
	}
}

func TestGet_LayerErrorContinues(t *testing.T) {
	tier1 := newMockCache[string]()
	tier1.getErr = errors.New("backend down")
	tier2 := newMockCache[string]()

	expected := "from_tier2"
	if err := tier2.Set(context.Background(), "key", &expected, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1},
			Layer[string]{Name: "tier2", Cache: tier2},
		),
		WithLogger[string](&spyLogger{}),
	)

	got, hit, err := chain.Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected hit in tier2 despite tier1 error")
	}
	if *got != expected {
		t.Fatalf("expected %q, got %q", expected, *got)
	}
}

func TestHealthCheck_ErrorShortCircuits(t *testing.T) {
	tier1 := newMockCache[string]()
	tier1.healthErr = errors.New("tier1 down")
	tier2 := newMockCache[string]()

	chain := NewChainedCache[string](
		WithLayers(
			Layer[string]{Name: "tier1", Cache: tier1},
			Layer[string]{Name: "tier2", Cache: tier2},
		),
	)

	err := chain.HealthCheck(context.Background())
	if !errors.Is(err, tier1.healthErr) {
		t.Fatalf("expected tier1 error, got %v", err)
	}
}

func TestLoadAndStore_DoubleCheckHit(t *testing.T) {
	tier1 := newMockCache[string]()
	cached := "already_there"
	if err := tier1.Set(context.Background(), "key", &cached, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	chain := NewChainedCache[string](
		WithLayers(Layer[string]{Name: "tier1", Cache: tier1}),
	)

	var loadCalls int
	val, err := chain.loadAndStore(context.Background(), "key", 5*time.Minute, func(context.Context) (*string, error) {
		loadCalls++
		return nil, errors.New("should not be called")
	})
	if err != nil {
		t.Fatalf("loadAndStore: %v", err)
	}
	if *(val.(*string)) != cached {
		t.Fatalf("expected %q, got %v", cached, val)
	}
	if loadCalls != 0 {
		t.Fatalf("expected double-check to short-circuit the loader, got %d calls", loadCalls)
	}
}

func TestSetAsync_LogsErrorOnFailure(t *testing.T) {
	tier1 := newMockCache[string]()
	tier1.setErr = errors.New("write failed")
	spy := &spyLogger{}

	chain := NewChainedCache[string](
		WithLayers(Layer[string]{Name: "tier1", Cache: tier1}),
		WithLogger[string](spy),
	)

	expected := "value"
	if _, err := chain.GetOrSet(context.Background(), "key", 5*time.Minute, func(context.Context) (*string, error) {
		return &expected, nil
	}); err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if spy.calls() == 0 {
		t.Fatal("expected setAsync failure to be logged")
	}
}

func TestDeleteAsync_LogsErrorOnFailure(t *testing.T) {
	tier1 := newMockCache[string]()
	tier1.delErr = errors.New("delete failed")
	spy := &spyLogger{}

	chain := NewChainedCache[string](
		WithLayers(Layer[string]{Name: "tier1", Cache: tier1}),
		WithLogger[string](spy),
	)

	if err := chain.Delete(context.Background(), "key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if spy.calls() == 0 {
		t.Fatal("expected deleteAsync failure to be logged")
	}
}
