package net_http_server

import (
	"encoding/json"
	"net/http"
)

// ProblemDetails represents RFC 7807 problem details response
type ProblemDetails struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// DefaultCrossOriginDenyHandler returns a handler that responds with RFC 7807 problem details
// when a cross-origin request is rejected.
func DefaultCrossOriginDenyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusForbidden)

		statusText := http.StatusText(http.StatusForbidden)
		problem := ProblemDetails{
			Type:   "https://api.example.com/errors/" + statusText,
			Title:  "Cross-Origin Request Denied",
			Status: http.StatusForbidden,
			Detail: "The request origin is not allowed to access this resource",
		}

		// Ignore encoding errors - if we can't encode JSON, we still sent the status
		_ = json.NewEncoder(w).Encode(problem)
	})
}
