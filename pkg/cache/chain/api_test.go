package chain

import (
	"context"
	"testing"
	"time"

	"github.com/brunojet/go-infra-adapters/v4/pkg/cache/local"
)

func TestNewChainedCache_RoundTrip(t *testing.T) {
	tier1 := local.NewCache[string]()

	cache := NewChainedCache[string](
		WithLayers(Layer[string]{Name: "tier1", Cache: tier1, TTL: 5 * time.Minute}),
	)

	ctx := context.Background()
	var loadCalls int
	expected := "value"

	got, err := cache.GetOrSet(ctx, "key", 5*time.Minute, func(_ context.Context) (*string, error) {
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

	// Layer population after a miss is asynchronous; wait for it before
	// asserting the second call is a cache hit.
	time.Sleep(50 * time.Millisecond)

	// Second call should hit the cache, not invoke the loader again.
	loadCalls = 0
	if _, err := cache.GetOrSet(ctx, "key", 5*time.Minute, func(_ context.Context) (*string, error) {
		loadCalls++
		return &expected, nil
	}); err != nil {
		t.Fatalf("GetOrSet: %v", err)
	}
	if loadCalls != 0 {
		t.Fatalf("expected 0 load calls on hit, got %d", loadCalls)
	}
}

func TestNewChainedCache_PanicsWithoutLayers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing layers")
		}
	}()

	NewChainedCache[string]()
}
