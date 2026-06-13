package local

import (
	"context"
	"testing"
	"time"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
)

func TestGet_Hit(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()
	val := "hello"

	err := c.Set(ctx, "key", &val, 0)
	if err != nil {
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

func TestGet_Miss(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	got, hit, err := c.Get(ctx, "absent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if hit {
		t.Fatal("expected miss, got hit")
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestGet_EmptyKey(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	_, _, err := c.Get(ctx, "")
	if err != errEmptyKey {
		t.Fatalf("expected errEmptyKey, got %v", err)
	}
}

func TestSet_Success(t *testing.T) {
	c := NewLocalCache[int]()
	ctx := context.Background()
	val := 42

	err := c.Set(ctx, "num", &val, 0)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, hit, _ := c.Get(ctx, "num")
	if !hit || *got != val {
		t.Fatal("Set didn't persist value")
	}
}

func TestSet_NilValue(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	err := c.Set(ctx, "key", nil, 0)
	if err != errNilValue {
		t.Fatalf("expected errNilValue, got %v", err)
	}
}

func TestSet_EmptyKey(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()
	val := "x"

	err := c.Set(ctx, "", &val, 0)
	if err != errEmptyKey {
		t.Fatalf("expected errEmptyKey, got %v", err)
	}
}

func TestTTL_Expiry(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()
	val := "expires"

	err := c.Set(ctx, "key", &val, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, hit, _ := c.Get(ctx, "key")
	if !hit {
		t.Fatal("expected hit immediately after Set")
	}
	_ = got

	time.Sleep(15 * time.Millisecond)

	_, hit, _ = c.Get(ctx, "key")
	if hit {
		t.Fatal("expected miss after TTL expiry")
	}
}

func TestTTL_NoExpiry(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()
	val := "forever"

	err := c.Set(ctx, "key", &val, 0)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	_, hit, _ := c.Get(ctx, "key")
	if !hit {
		t.Fatal("expected hit after delay with no TTL")
	}
}

func TestDelete(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()
	val := "x"

	c.Set(ctx, "key", &val, 0)

	err := c.Delete(ctx, "key")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, hit, _ := c.Get(ctx, "key")
	if hit {
		t.Fatal("expected miss after Delete")
	}
}

func TestDelete_EmptyKey(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	err := c.Delete(ctx, "")
	if err != errEmptyKey {
		t.Fatalf("expected errEmptyKey, got %v", err)
	}
}

func TestDelete_Absent(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	err := c.Delete(ctx, "absent")
	if err != nil {
		t.Fatalf("Delete of absent key should not error, got %v", err)
	}
}

func TestExists_Hit(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()
	val := "x"

	c.Set(ctx, "key", &val, 0)

	exists, err := c.Exists(ctx, "key")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected exists=true")
	}
}

func TestExists_Miss(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	exists, err := c.Exists(ctx, "absent")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("expected exists=false")
	}
}

func TestExists_ExpiredKey(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()
	val := "x"

	c.Set(ctx, "key", &val, 10*time.Millisecond)
	time.Sleep(15 * time.Millisecond)

	exists, err := c.Exists(ctx, "key")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Fatal("expected exists=false for expired key")
	}
}


func TestHealthCheck(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	err := c.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
}

func TestWithLoggerNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil logger")
		}
	}()

	NewLocalCache[string](WithLogger(nil))
	t.Fatal("should have panicked")
}

type testLogger struct {
	warnCalls int
}

func (tl *testLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {}
func (tl *testLogger) Info(ctx context.Context, msg string, fields ...logger.Field)  {}
func (tl *testLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	tl.warnCalls++
}
func (tl *testLogger) Error(ctx context.Context, msg string, err error, fields ...logger.Field) {}

