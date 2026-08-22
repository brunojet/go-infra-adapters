package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsControlChars(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "plain text", in: "hello world", want: false},
		{name: "tab character", in: "hello\tworld", want: true},
		{name: "newline character", in: "hello\nworld", want: true},
		{name: "carriage return", in: "hello\rworld", want: true},
		{name: "unicode control", in: "line\u001f", want: true},
		{name: "escaped literal slash t", in: `hello\tworld`, want: false},
		{name: "escaped literal slash n", in: `hello\nworld`, want: false},
		{name: "html fragment escaped newline", in: `<div class="x">hello</div>\n<p>ok</p>`, want: false},
		{name: "html fragment with real newline", in: "<div class=\"x\">hello</div>\n<p>ok</p>", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ContainsControlChars(tt.in))
		})
	}
}
