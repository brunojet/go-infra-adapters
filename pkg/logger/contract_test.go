package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestField_String(t *testing.T) {
	field := String("key", "value")
	assert.Equal(t, "key", field.Key)
	assert.Equal(t, "value", field.Value)
}

func TestField_Int(t *testing.T) {
	field := Int("count", 42)
	assert.Equal(t, "count", field.Key)
	assert.Equal(t, 42, field.Value)
}

func TestField_Bool(t *testing.T) {
	field := Bool("enabled", true)
	assert.Equal(t, "enabled", field.Key)
	assert.Equal(t, true, field.Value)
}

func TestField_Error(t *testing.T) {
	err := context.Canceled
	field := Error("err", err)
	assert.Equal(t, "err", field.Key)
	assert.Equal(t, err, field.Value)
}

func TestField_Any(t *testing.T) {
	obj := map[string]int{"a": 1}
	field := Any("data", obj)
	assert.Equal(t, "data", field.Key)
	assert.Equal(t, obj, field.Value)
}

func TestField_Types(t *testing.T) {
	// Ensure Field struct can hold any type
	tests := []struct {
		name  string
		field Field
	}{
		{
			name:  "string value",
			field: String("msg", "hello"),
		},
		{
			name:  "int value",
			field: Int("count", 123),
		},
		{
			name:  "bool value",
			field: Bool("active", false),
		},
		{
			name:  "error value",
			field: Error("err", context.DeadlineExceeded),
		},
		{
			name:  "interface value",
			field: Any("custom", []string{"a", "b"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.field.Key)
			assert.NotNil(t, tt.field.Value)
		})
	}
}
