package net_http_client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreakerConfig_NilOptionSkipped(t *testing.T) {
	cfg := newCircuitBreakerConfig(nil, WithCircuitBreakerMaxFailures(3), nil)
	assert.Equal(t, 3, cfg.MaxFailures)
}

func TestNewCircuitBreakerConfig_Defaults(t *testing.T) {
	cfg := newCircuitBreakerConfig()
	assert.Equal(t, 0, cfg.MaxFailures)
	assert.Equal(t, time.Duration(0), cfg.ResetTimeout)
	assert.Equal(t, 0, cfg.HalfOpenRequests)
}

func TestBreakerOptions_Applied(t *testing.T) {
	cfg := newCircuitBreakerConfig(WithCircuitBreakerMaxFailures(5), WithCircuitBreakerResetTimeout(10*time.Second), WithCircuitBreakerHalfOpenRequests(2))
	assert.Equal(t, 5, cfg.MaxFailures)
	assert.Equal(t, 10*time.Second, cfg.ResetTimeout)
	assert.Equal(t, 2, cfg.HalfOpenRequests)
}

func TestBreakerOptions_PanicOnInvalid(t *testing.T) {
	require.Panics(t, func() { newCircuitBreakerConfig(WithCircuitBreakerMaxFailures(0)) })
	require.Panics(t, func() { newCircuitBreakerConfig(WithCircuitBreakerResetTimeout(0)) })
	require.Panics(t, func() { newCircuitBreakerConfig(WithCircuitBreakerHalfOpenRequests(-1)) })
}
