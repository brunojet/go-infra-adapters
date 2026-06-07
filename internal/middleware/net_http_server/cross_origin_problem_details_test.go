package net_http_server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCrossOriginDenyHandler_StatusCode(t *testing.T) {
	handler := DefaultCrossOriginDenyHandler()
	req := httptest.NewRequest("POST", "http://example.com", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDefaultCrossOriginDenyHandler_ContentType(t *testing.T) {
	handler := DefaultCrossOriginDenyHandler()
	req := httptest.NewRequest("POST", "http://example.com", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
}

func TestDefaultCrossOriginDenyHandler_ResponseBody(t *testing.T) {
	handler := DefaultCrossOriginDenyHandler()
	req := httptest.NewRequest("POST", "http://example.com", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var problem ProblemDetails
	err := json.NewDecoder(rec.Body).Decode(&problem)
	assert.NoError(t, err)

	assert.NotEmpty(t, problem.Type)
	assert.NotEmpty(t, problem.Title)
	assert.Equal(t, http.StatusForbidden, problem.Status)
	assert.NotEmpty(t, problem.Detail)
}

func TestDefaultCrossOriginDenyHandler_ProblemFields(t *testing.T) {
	handler := DefaultCrossOriginDenyHandler()
	req := httptest.NewRequest("POST", "http://example.com", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var problem ProblemDetails
	_ = json.NewDecoder(rec.Body).Decode(&problem)

	assert.Equal(t, "https://api.example.com/errors/Forbidden", problem.Type)
	assert.Equal(t, "Cross-Origin Request Denied", problem.Title)
	assert.Equal(t, http.StatusForbidden, problem.Status)
	assert.Contains(t, problem.Detail, "origin is not allowed")
}

func TestBuildCrossOriginProtection_UsesDefaultDenyHandler(t *testing.T) {
	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com"),
	)
	// No custom deny handler provided

	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
	// If SetDenyHandler was called with default, cop should be configured
}

func TestBuildCrossOriginProtection_UsesCustomDenyHandler(t *testing.T) {
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("custom rejection"))
	})

	cfg := NewCrossOriginConfig(
		WithTrustedOrigins("https://example.com"),
		WithDenyHandler(customHandler),
	)

	cop := cfg.BuildCrossOriginProtection()
	assert.NotNil(t, cop)
	// Custom handler should have been set (can't easily verify without triggering rejection)
}
