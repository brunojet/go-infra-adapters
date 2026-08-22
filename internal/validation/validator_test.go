package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBody(t *testing.T) {
	t.Run("accepts clean payload", func(t *testing.T) {
		type payload struct {
			Name string `validate:"required,control_chars"`
		}

		require.NoError(t, ValidateBody(payload{Name: "bruno"}))
	})

	t.Run("returns the validation error for the invalid payload", func(t *testing.T) {
		type payload struct {
			Name    string `validate:"required,control_chars"`
			Email   string `validate:"required,email,control_chars"`
			Comment string
		}

		err := ValidateBody(payload{
			Name:    "bruno\tjack",
			Email:   "123",
			Comment: "ok",
		})
		require.Error(t, err)

		msg := err.Error()
		assert.Contains(t, msg, "Email")
		assert.Contains(t, msg, "invalid value")
		assert.Contains(t, msg, "contains invalid characters")
	})
}
