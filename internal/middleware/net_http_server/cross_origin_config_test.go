package net_http_server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCrossOriginConfig_Empty(t *testing.T) {
	cfg := NewCrossOriginConfig()
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.trustedOrigins, 0)
	assert.Len(t, cfg.insecureBypassPatterns, 0)
	// denyHandler defaults to DefaultCrossOriginDenyHandler
	assert.NotNil(t, cfg.denyHandler)
}

func TestWithTrustedOrigins_ValidOrigins(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com", "https://trusted.com"),
	)
	assert.Len(t, cfg.trustedOrigins, 2)
	assert.True(t, cfg.trustedOrigins["https://example.com"])
	assert.True(t, cfg.trustedOrigins["https://trusted.com"])
}

func TestWithTrustedOrigins_NoOriginsPanic(t *testing.T) {
	assert.Panics(t, func() {
		NewCrossOriginConfig(WithTrustedOrigins())
	})
}

func TestWithTrustedOrigins_Composable(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://first.com"),
		WithTrustedOrigins("https://second.com", "https://third.com"),
	)
	assert.Len(t, cfg.trustedOrigins, 3)
	assert.True(t, cfg.trustedOrigins["https://first.com"])
	assert.True(t, cfg.trustedOrigins["https://second.com"])
	assert.True(t, cfg.trustedOrigins["https://third.com"])
}

func TestWithTrustedOrigins_Duplicates(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com", "https://example.com"),
	)
	// Set semantics: duplicates ignored
	assert.Len(t, cfg.trustedOrigins, 1)
	assert.True(t, cfg.trustedOrigins["https://example.com"])
}

func TestWithInsecureBypassPatterns_Valid(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithInsecureBypassPatterns("^/webhooks/.*", "^/public/.*"),
	)
	assert.Len(t, cfg.insecureBypassPatterns, 2)
	assert.Contains(t, cfg.insecureBypassPatterns, "^/webhooks/.*")
	assert.Contains(t, cfg.insecureBypassPatterns, "^/public/.*")
}

func TestWithInsecureBypassPatterns_EmptyPanic(t *testing.T) {
	assert.Panics(t, func() {
		NewCrossOriginConfig(WithInsecureBypassPatterns())
	})
}

func TestWithInsecureBypassPatterns_EmptyPatternPanic(t *testing.T) {
	assert.Panics(t, func() {
		NewCrossOriginConfig(WithInsecureBypassPatterns("^/valid/.*", ""))
	})
}

func TestWithInsecureBypassPatterns_Composable(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithInsecureBypassPatterns("^/webhooks/.*"),
		WithInsecureBypassPatterns("^/public/.*", "^/assets/.*"),
	)
	assert.Len(t, cfg.insecureBypassPatterns, 3)
}

func TestWithDenyHandler_Valid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	cfg := NewCrossOriginConfig(WithDenyHandler(handler))
	assert.NotNil(t, cfg.denyHandler)
	// Can't compare function values, just verify it's set
}

func TestWithDenyHandler_NilPanic(t *testing.T) {
	assert.Panics(t, func() {
		NewCrossOriginConfig(WithDenyHandler(nil))
	})
}

func TestNewCrossOriginConfig_NilOptionSkipped(t *testing.T) {
	cfg := NewCrossOriginConfig(nil, WithTrustedOrigins("https://example.com"), nil)
	assert.Len(t, cfg.trustedOrigins, 1)
	assert.True(t, cfg.trustedOrigins["https://example.com"])
}

func TestBuildCrossOriginProtection_NoOrigins(t *testing.T) {
	cfg := NewCrossOriginConfig()
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
	// Should not panic, just empty protection
}

func TestBuildCrossOriginProtection_WithOrigins(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com", "https://trusted.com"),
	)
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
	// Verify cop is properly configured by checking it doesn't panic
	// (actual validation requires making requests)
}

func TestBuildCrossOriginProtection_WithBypassPatterns(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com"),
		WithInsecureBypassPatterns("^/webhooks/.*", "^/public/.*"),
	)
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
}

func TestBuildCrossOriginProtection_WithDenyHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com"),
		WithDenyHandler(handler),
	)
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
}

func TestBuildCrossOriginProtection_Full(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com", "https://trusted.com"),
		WithInsecureBypassPatterns("^/webhooks/.*", "^/public/.*"),
		WithDenyHandler(handler),
	)
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
}

func TestBuildCrossOriginProtection_MultipleOrigins(t *testing.T) {
	// Test that multiple origins are properly registered
	origins := []string{
		"https://api.example.com",
		"https://admin.example.com",
		"http://localhost:3000",
	}
	cfg := NewCrossOriginConfig(WithTrustedOrigins(origins...))
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
	// Verify config stored all origins
	assert.Len(t, cfg.trustedOrigins, 3)
}

func TestBuildCrossOriginProtection_BypassPatterns(t *testing.T) {
	// Test that bypass patterns are properly set
	patterns := []string{
		"^/webhooks/.*",
		"^/public/.*",
		"^/health",
	}
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com"),
		WithInsecureBypassPatterns(patterns...),
	)
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
	// Verify patterns were stored
	assert.Len(t, cfg.insecureBypassPatterns, 3)
}

func TestBuildCrossOriginProtection_DefaultDenyHandler(t *testing.T) {
	// When no deny handler provided, should use default
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com"),
	)
	// Verify denyHandler is set to default
	assert.NotNil(t, cfg.denyHandler)
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
}

func TestBuildCrossOriginProtection_CustomDenyHandler(t *testing.T) {
	// Verify custom deny handler is used
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com"),
		WithDenyHandler(customHandler),
	)
	// Verify denyHandler is set (can't directly compare func types)
	assert.NotNil(t, cfg.denyHandler)
	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
}
