package local

import (
	"context"
	"testing"
	"time"

	pkglogger "github.com/brunojet/go-infra-adapters/v4/pkg/logger"
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

	if err := c.Set(ctx, "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

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

	if err := c.Set(ctx, "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

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

	if err := c.Set(ctx, "key", &val, 10*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}
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

type spyLogger struct{ warnCalls int }

func (s *spyLogger) Debug(context.Context, string, ...pkglogger.Field) {}
func (s *spyLogger) Info(context.Context, string, ...pkglogger.Field)  {}
func (s *spyLogger) Warn(context.Context, string, ...pkglogger.Field) {
	s.warnCalls++
}
func (s *spyLogger) Error(context.Context, string, error, ...pkglogger.Field) {}

func TestWithLogger_Applied(t *testing.T) {
	spy := &spyLogger{}
	c := NewLocalCache[string](WithLogger(spy))
	if c.logger != pkglogger.Logger(spy) {
		t.Fatal("expected configured logger to be used")
	}
}

func TestWithMaxBytes_Applied(t *testing.T) {
	c := NewLocalCache[string](WithMaxBytes(10 * 1024 * 1024))
	ctx := context.Background()
	val := "x"

	if err := c.Set(ctx, "key", &val, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, hit, _ := c.Get(ctx, "key"); !hit {
		t.Fatal("expected hit after custom maxBytes")
	}
}

func TestWithMaxBytes_PanicOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-positive maxBytes")
		}
	}()

	WithMaxBytes(0)
	t.Fatal("should have panicked")
}

func TestExists_EmptyKey(t *testing.T) {
	c := NewLocalCache[string]()
	ctx := context.Background()

	_, err := c.Exists(ctx, "")
	if err != errEmptyKey {
		t.Fatalf("expected errEmptyKey, got %v", err)
	}
}

func TestEstimateSize_NilValue(t *testing.T) {
	if got := estimateSize[string](nil); got != 0 {
		t.Fatalf("expected 0 for nil value, got %d", got)
	}
}
