package local

import (
	"context"
	"testing"
)

func TestNewCache_RoundTrip(t *testing.T) {
	c := NewCache[string]()
	ctx := context.Background()
	val := "hello"

	if err := c.Set(ctx, "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, hit, err := c.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !hit {
		t.Fatal("expected hit, got miss")
	}
	if *got != val {
		t.Fatalf("expected %q, got %q", val, *got)
	}
}

func TestWithLogger_PanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil logger")
		}
	}()

	NewCache[string](WithLogger(nil))
	t.Fatal("should have panicked")
}
