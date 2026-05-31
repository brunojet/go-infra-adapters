package retry

import (
	"errors"
	"testing"
	"time"
)

func TestStandardRetryStrategy(t *testing.T) {
	s := NewStandard()

	t.Run("MaxAttempts", func(t *testing.T) {
		if s.MaxAttempts() != 3 {
			t.Fatalf("expected 3, got %d", s.MaxAttempts())
		}
	})

	t.Run("IsRetryable", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
			want bool
		}{
			{"nil error", nil, false},
			{"5xx error", errors.New("StatusCode: 500"), true},
			{"429 throttling", errors.New("StatusCode: 429"), true},
			{"409 conflict", errors.New("StatusCode: 409"), true},
			{"timeout", errors.New("context deadline exceeded: timeout"), true},
			{"connection refused", errors.New("connection refused"), true},
			{"4xx other", errors.New("StatusCode: 400"), false},
			{"non-retryable", errors.New("validation error"), false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := s.IsRetryable(tt.err)
				if got != tt.want {
					t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.want)
				}
			})
		}
	})

	t.Run("BackoffFor", func(t *testing.T) {
		tests := []struct {
			attempt  int
			expected time.Duration
		}{
			{0, 0},
			{1, 500 * time.Millisecond},
			{2, 1 * time.Second},
			{3, 2 * time.Second},
		}

		for _, tt := range tests {
			got := s.BackoffFor(tt.attempt)
			if got != tt.expected {
				t.Errorf("BackoffFor(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		}
	})
}

func TestNoRetry(t *testing.T) {
	nr := &NoRetry{}

	if nr.MaxAttempts() != 1 {
		t.Fatalf("expected 1, got %d", nr.MaxAttempts())
	}

	if nr.IsRetryable(errors.New("any error")) {
		t.Fatal("NoRetry should not retry")
	}

	if nr.BackoffFor(1) != 0 {
		t.Fatal("NoRetry should have 0 backoff")
	}
}
