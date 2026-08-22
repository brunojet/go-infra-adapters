package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanControlChars(t *testing.T) {
	type payload struct {
		Name  string
		Email string
	}

	type nestedPayload struct {
		Comment string
	}

	type withUnexported struct {
		Name    string
		private string //nolint:unused
	}

	type level3 struct {
		Value string
	}
	type level2 struct {
		Next level3
	}
	type level1 struct {
		Next level2
	}

	type withPointer struct {
		Inner *nestedPayload
	}

	type withInterface struct {
		Extra interface{}
	}

	// coverageGaps packs one field per early-return branch that collectIssues
	// only takes once a sibling field has already made the whole object
	// invalid: a nil pointer, an interface holding a non-string dynamic
	// value, a []int, a map[int]int, and an all-scalar nested struct sit
	// alongside the one dirty field, forcing collectStructIssues to visit
	// every one of them.
	type coverageGaps struct {
		Ptr     *nestedPayload
		Extra   interface{}
		Numbers []int
		Counts  map[int]int
		Point   struct{ X, Y int }
		Name    string
	}

	tests := []struct {
		name        string
		object      any
		wantErr     bool
		wantContain string
	}{
		// --- simple values: scalars ---
		{
			name:    "bare int valid",
			object:  42,
			wantErr: false,
		},
		{
			name:    "bare bool valid",
			object:  true,
			wantErr: false,
		},
		{
			name:    "bare float valid",
			object:  3.14,
			wantErr: false,
		},
		{
			name:    "empty string valid",
			object:  "",
			wantErr: false,
		},
		{
			name:    "accented unicode text valid",
			object:  "João Ítalo café",
			wantErr: false,
		},
		{
			name:    "nil input valid",
			object:  nil,
			wantErr: false,
		},

		// --- simple values: strings with different control character classes ---
		{
			name:        "string with tab invalid",
			object:      "safe\tvalue",
			wantErr:     true,
			wantContain: "contains invalid characters",
		},
		{
			name:        "string with NUL byte invalid",
			object:      "bad\x00value",
			wantErr:     true,
			wantContain: "contains invalid characters",
		},
		{
			name:        "string with DEL invalid",
			object:      "bad\x7fvalue",
			wantErr:     true,
			wantContain: "contains invalid characters",
		},
		{
			name:        "string with C1 control invalid",
			object:      "bad\u0085value",
			wantErr:     true,
			wantContain: "contains invalid characters",
		},

		// --- structs: flat ---
		{
			name:    "struct valid",
			object:  payload{Name: "bruno", Email: "bruno@example.com"},
			wantErr: false,
		},
		{
			name:        "struct invalid",
			object:      payload{Name: "bruno\tjack", Email: "bruno@example.com"},
			wantErr:     true,
			wantContain: "Name: contains invalid characters",
		},
		{
			name: "struct with scalar fields valid",
			object: struct {
				ID     int
				Active bool
				Name   string
			}{ID: 1, Active: true, Name: "safe"},
			wantErr: false,
		},
		{
			name: "struct with only scalar fields has nothing scannable",
			object: struct {
				ID    int
				Score float64
			}{ID: 1, Score: 9.5},
			wantErr: false,
		},
		{
			name:    "struct with unexported dirty field is ignored",
			object:  withUnexported{Name: "safe", private: "bad\tvalue"},
			wantErr: false,
		},

		// --- structs: pointers ---
		{
			name:    "pointer to struct valid",
			object:  &payload{Name: "bruno", Email: "bruno@example.com"},
			wantErr: false,
		},
		{
			name:        "pointer to struct invalid",
			object:      &payload{Name: "bruno\tjack", Email: "bruno@example.com"},
			wantErr:     true,
			wantContain: "Name: contains invalid characters",
		},
		{
			name:    "nil pointer to struct valid",
			object:  (*payload)(nil),
			wantErr: false,
		},
		{
			name:        "double pointer to struct invalid",
			object:      func() **payload { p := &payload{Name: "bruno\tjack", Email: "ok@x.com"}; return &p }(),
			wantErr:     true,
			wantContain: "Name: contains invalid characters",
		},
		{
			name:    "struct with pointer field valid",
			object:  withPointer{Inner: &nestedPayload{Comment: "ok"}},
			wantErr: false,
		},
		{
			name:        "struct with pointer field invalid",
			object:      withPointer{Inner: &nestedPayload{Comment: "bad\nvalue"}},
			wantErr:     true,
			wantContain: "Inner.Comment: contains invalid characters",
		},
		{
			name:    "struct with nil pointer field valid",
			object:  withPointer{Inner: nil},
			wantErr: false,
		},

		// --- structs: interface fields ---
		{
			name:    "struct with interface field valid",
			object:  withInterface{Extra: "safe"},
			wantErr: false,
		},
		{
			name:        "struct with interface field invalid",
			object:      withInterface{Extra: "bad\tvalue"},
			wantErr:     true,
			wantContain: "Extra: contains invalid characters",
		},
		{
			name:    "struct with nil interface field valid",
			object:  withInterface{Extra: nil},
			wantErr: false,
		},

		// --- structs: deep nesting (3 levels) ---
		{
			name:    "deeply nested struct valid",
			object:  level1{Next: level2{Next: level3{Value: "safe"}}},
			wantErr: false,
		},
		{
			name:        "deeply nested struct invalid",
			object:      level1{Next: level2{Next: level3{Value: "bad\tvalue"}}},
			wantErr:     true,
			wantContain: "Next.Next.Value: contains invalid characters",
		},
		{
			name: "nested collection invalid",
			object: struct {
				Name   string
				Inner  nestedPayload
				Emails []string
				Meta   map[string]string
			}{
				Name:   "bruno",
				Inner:  nestedPayload{Comment: "ok"},
				Emails: []string{"bruno@example.com", "bad\nvalue"},
				Meta:   map[string]string{"token": "safe"},
			},
			wantErr:     true,
			wantContain: "Emails[1]: contains invalid characters",
		},

		// --- slices / arrays: simple ---
		{
			name:    "slice valid",
			object:  []string{"safe", "value"},
			wantErr: false,
		},
		{
			name:        "slice invalid",
			object:      []string{"safe", "bad\tvalue"},
			wantErr:     true,
			wantContain: "[1]: contains invalid characters",
		},
		{
			name:        "slice reports every invalid entry, not just the first",
			object:      []string{"bad\tone", "safe", "bad\ttwo"},
			wantErr:     true,
			wantContain: "[0]: contains invalid characters; [2]: contains invalid characters",
		},
		{
			name:    "empty slice valid",
			object:  []string{},
			wantErr: false,
		},
		{
			name:    "nil slice valid",
			object:  []string(nil),
			wantErr: false,
		},
		{
			name:    "array valid",
			object:  [2]string{"safe", "value"},
			wantErr: false,
		},
		{
			name:        "array invalid",
			object:      [2]string{"safe", "bad\tvalue"},
			wantErr:     true,
			wantContain: "[1]: contains invalid characters",
		},
		{
			name:    "slice of ints is never walked",
			object:  []int{1, 2, 3},
			wantErr: false,
		},

		// --- slices: complex elements ---
		{
			name:    "slice of structs valid",
			object:  []payload{{Name: "ok", Email: "a@b.com"}, {Name: "fine", Email: "c@d.com"}},
			wantErr: false,
		},
		{
			name:        "slice of structs invalid",
			object:      []payload{{Name: "ok", Email: "a@b.com"}, {Name: "bad\tname", Email: "c@d.com"}},
			wantErr:     true,
			wantContain: "[1].Name: contains invalid characters",
		},
		{
			name:    "slice of pointers to struct valid",
			object:  []*nestedPayload{{Comment: "ok"}, {Comment: "fine"}},
			wantErr: false,
		},
		{
			name:        "slice of pointers to struct invalid",
			object:      []*nestedPayload{{Comment: "ok"}, {Comment: "bad\nvalue"}},
			wantErr:     true,
			wantContain: "[1].Comment: contains invalid characters",
		},
		{
			name:    "slice of interface values valid",
			object:  []interface{}{"safe", "ok"},
			wantErr: false,
		},
		{
			name:        "slice of interface values invalid",
			object:      []interface{}{"safe", "bad\tvalue"},
			wantErr:     true,
			wantContain: "[1]: contains invalid characters",
		},

		// --- maps: simple ---
		{
			name:    "map valid",
			object:  map[string]string{"safe": "ok", "token": "allowed"},
			wantErr: false,
		},
		{
			name:        "map invalid",
			object:      map[string]string{"safe": "ok", "unsafe": "bad\tvalue"},
			wantErr:     true,
			wantContain: "[unsafe]: contains invalid characters",
		},
		{
			name:        "map key invalid",
			object:      map[string]string{"bad\tkey": "value"},
			wantErr:     true,
			wantContain: "key: contains invalid characters",
		},
		{
			name:    "empty map valid",
			object:  map[string]string{},
			wantErr: false,
		},
		{
			name:    "nil map valid",
			object:  map[string]string(nil),
			wantErr: false,
		},
		{
			name:    "map of ints is never walked",
			object:  map[int]int{1: 2, 3: 4},
			wantErr: false,
		},
		{
			name:        "non-string map key renders as its actual value",
			object:      map[int]string{42: "bad\tvalue"},
			wantErr:     true,
			wantContain: "[42]: contains invalid characters",
		},

		// --- maps: complex values ---
		{
			name:    "map with interface value valid",
			object:  map[string]interface{}{"name": "safe"},
			wantErr: false,
		},
		{
			name:        "map with interface value invalid",
			object:      map[string]interface{}{"name": "bad\tvalue"},
			wantErr:     true,
			wantContain: "[name]: contains invalid characters",
		},
		{
			name:    "map with struct values valid",
			object:  map[string]nestedPayload{"a": {Comment: "ok"}, "b": {Comment: "fine"}},
			wantErr: false,
		},
		{
			name:        "map with struct values invalid",
			object:      map[string]nestedPayload{"a": {Comment: "ok"}, "b": {Comment: "bad\nvalue"}},
			wantErr:     true,
			wantContain: "[b].Comment: contains invalid characters",
		},

		// --- structs: nested payload (regression case) ---
		{
			name:        "nested payload invalid",
			object:      nestedPayload{Comment: "bad\nvalue"},
			wantErr:     true,
			wantContain: "Comment: contains invalid characters",
		},

		// --- invalid object with clean siblings hitting every early-return ---
		{
			name: "invalid object exercises every early-return branch a sibling field can take",
			object: coverageGaps{
				Ptr:     nil,
				Extra:   42,
				Numbers: []int{1, 2, 3},
				Counts:  map[int]int{1: 2},
				Point:   struct{ X, Y int }{X: 1, Y: 2},
				Name:    "bad\tvalue",
			},
			wantErr:     true,
			wantContain: "Name: contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ScanControlChars(tt.object)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantContain)
		})
	}
}
