package net_http_base

import (
	"context"
	"net/http"
	"testing"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/stretchr/testify/assert"
)

type testTarget struct {
	headers http.Header
	ctx     context.Context
}

func (t *testTarget) Headers() http.Header {
	return t.headers
}

func (t *testTarget) SanitizationContext() context.Context {
	return t.ctx
}

type testLogger struct {
	eventCount int
}

func (t *testLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {
	// Count debug events (header removals)
	if msg == "header removed" {
		t.eventCount++
	}
}

func (t *testLogger) Info(ctx context.Context, msg string, fields ...logger.Field)             {}
func (t *testLogger) Warn(ctx context.Context, msg string, fields ...logger.Field)             {}
func (t *testLogger) Error(ctx context.Context, msg string, err error, fields ...logger.Field) {}

func TestSanitizeHeaders_RemovesDeniedHeaders(t *testing.T) {
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer token"},
		"X-Custom":      []string{"value"},
	}

	target := &testTarget{
		headers: headers,
		ctx:     context.Background(),
	}

	logger := &testLogger{}
	denied := map[string]bool{
		"Authorization": true,
	}
	allowed := map[string]bool{
		"Content-Type": true,
		"X-Custom":     true,
	}

	SanitizeHeaders(target, logger, denied, allowed, OutboundRequest)

	// Authorization should be removed (in denied list)
	assert.Empty(t, headers.Get("Authorization"))
	// Others should remain (in allowed list)
	assert.NotEmpty(t, headers.Get("Content-Type"))
	assert.NotEmpty(t, headers.Get("X-Custom"))
	// Logging should happen for Authorization removal
	assert.Equal(t, 1, logger.eventCount)
}

func TestSanitizeHeaders_RespectsWhitelist(t *testing.T) {
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Custom":     []string{"value"},
		"X-Other":      []string{"value"},
	}

	target := &testTarget{
		headers: headers,
		ctx:     context.Background(),
	}

	logger := &testLogger{}
	denied := map[string]bool{}
	allowed := map[string]bool{
		"Content-Type": true,
	}

	SanitizeHeaders(target, logger, denied, allowed, InboundRequest)

	// Content-Type should remain (in whitelist)
	assert.NotEmpty(t, headers.Get("Content-Type"))
	// Others should be removed (not in whitelist)
	assert.Empty(t, headers.Get("X-Custom"))
	assert.Empty(t, headers.Get("X-Other"))
	// Logging should happen for removed headers
	assert.Equal(t, 2, logger.eventCount)
}

func TestSanitizeHeaders_CaseInsensitive(t *testing.T) {
	headers := make(http.Header)
	// Use Set() to ensure canonical form (how it works in production)
	headers.Set("authorization", "Bearer token")
	headers.Set("Content-Type", "application/json")

	target := &testTarget{
		headers: headers,
		ctx:     context.Background(),
	}

	logger := &testLogger{}
	denied := map[string]bool{
		"Authorization": true,
	}
	allowed := map[string]bool{
		"Content-Type": true,
	}

	SanitizeHeaders(target, logger, denied, allowed, OutboundRequest)

	// Authorization should be removed (case-insensitive via canonical form)
	assert.Empty(t, headers.Get("authorization"))
	assert.Empty(t, headers.Get("Authorization"))
	// Content-Type should remain (not in denied list)
	assert.NotEmpty(t, headers.Get("Content-Type"))
	// Logging should happen for Authorization removal
	assert.Equal(t, 1, logger.eventCount)
}
