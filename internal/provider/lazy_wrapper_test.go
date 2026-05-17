package provider

import (
	"errors"
	"testing"
)

func TestNewLazy_CachesValueAndError(t *testing.T) {
	calls := 0
	l := NewLazy(func() (int, error) {
		calls++
		return 42, nil
	})
	v, err := l.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 42 {
		t.Fatalf("unexpected value: %v", v)
	}
	// second call should not increment constructor calls
	v2, _ := l.Get()
	if v2 != 42 || calls != 1 {
		t.Fatalf("caching failed, calls=%d v2=%v", calls, v2)
	}

	// error caching
	calls = 0
	e := errors.New("boom")
	l2 := NewLazy(func() (int, error) {
		calls++
		return 0, e
	})
	if _, err := l2.Get(); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := l2.Get(); err == nil || calls != 1 {
		t.Fatalf("expected cached error and single constructor call, calls=%d", calls)
	}
	if l2.Validate() == nil {
		t.Fatalf("Validate should return error when ctor fails")
	}
}

func TestNewLazyWrapper_Get_CachesValue(t *testing.T) {
	calls := 0
	w := newLazyWrapper(func() (int, error) {
		calls++
		return 7, nil
	})
	v, err := w.get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 7 {
		t.Fatalf("unexpected value: %v", v)
	}
	// second call should not increment
	v2, _ := w.get()
	if v2 != 7 || calls != 1 {
		t.Fatalf("caching failed, calls=%d v2=%v", calls, v2)
	}
}
