package net_http_client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultFailureClassifier(t *testing.T) {
	classifier := &DefaultFailureClassifier{}

	tests := []struct {
		name       string
		err        error
		resp       *http.Response
		shouldIncr bool // true if should increment CB
	}{
		// === NO ERRORS ===
		{
			name:       "no error with 200 OK",
			err:        nil,
			resp:       &http.Response{StatusCode: 200},
			shouldIncr: false,
		},
		{
			name:       "no error with 404 Not Found",
			err:        nil,
			resp:       &http.Response{StatusCode: 404},
			shouldIncr: false,
		},
		{
			name:       "429 Too Many Requests (server overloaded)",
			err:        nil,
			resp:       &http.Response{StatusCode: 429},
			shouldIncr: true,
		},

		// === SERVER ERRORS (5xx) ===
		{
			name:       "500 Internal Server Error",
			err:        nil,
			resp:       &http.Response{StatusCode: 500},
			shouldIncr: true,
		},
		{
			name:       "503 Service Unavailable",
			err:        nil,
			resp:       &http.Response{StatusCode: 503},
			shouldIncr: true,
		},
		{
			name:       "504 Gateway Timeout",
			err:        nil,
			resp:       &http.Response{StatusCode: 504},
			shouldIncr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classErr := classifier.ClassifyError(tt.resp)
			if tt.shouldIncr {
				assert.NotNil(t, classErr, "should increment CB")
			} else {
				assert.Nil(t, classErr, "should NOT increment CB")
			}
		})
	}
}
