package contracts

import (
	"context"
	"testing"
	"time"
)

type mockCache[T any] struct{}

func (m mockCache[T]) Get(_ context.Context, _ string) (*T, bool, error) { return nil, false, nil }
func (m mockCache[T]) Set(_ context.Context, _ string, _ *T, _ time.Duration) error {
	return nil
}
func (m mockCache[T]) Delete(_ context.Context, _ string) error         { return nil }
func (m mockCache[T]) Exists(_ context.Context, _ string) (bool, error) { return false, nil }
func (m mockCache[T]) HealthCheck(_ context.Context) error              { return nil }

func TestCache_Interface(t *testing.T) {
	var _ Cache[string] = mockCache[string]{}
}

func TestLoader_Signature(t *testing.T) {
	var loader Loader[string] = func(_ context.Context) (*string, error) {
		s := "value"
		return &s, nil
	}

	val, err := loader(context.Background())
	if err != nil {
		t.Fatalf("loader: %v", err)
	}
	if val == nil || *val != "value" {
		t.Fatalf("expected 'value', got %v", val)
	}
}
