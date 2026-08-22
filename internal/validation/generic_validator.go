package validation

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
)

const (
	controlCharsMessage = "contains invalid characters"
	messageConnector    = ": "
	errorsHeader        = "validation errors: %s"
	issueSeparator      = "; "
	fieldSeparator      = "."
	indexOpen           = "["
	indexClose          = "]"
	mapKeySuffix        = " key"
)

type cachedField struct {
	index []int
	name  string
}

var fieldCache sync.Map

// ScanControlChars performs a generic scan over any value — not just an HTTP
// body, hence no hardcoded root label in the output — and checks every
// exported string it can reach — struct fields (including nested structs),
// slice/array elements, and both map keys and map values — for control
// characters, without requiring a custom tag registration per struct. Fields
// of kinds that can't carry a control character (int, bool, float, etc.) are
// skipped rather than rejected.
//
// The walk happens exactly once. Each level returns the issues found below
// it as path suffixes (e.g. ".Name: contains invalid characters" when it has
// a parent to connect to) and only pays for a string concatenation to
// prepend its own segment when a child actually reported something — a
// clean field, element, or entry returns a nil slice and contributes zero
// allocations.
func ScanControlChars(object any) error {
	issues := collectIssues(reflect.ValueOf(object), false)
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf(errorsHeader, strings.Join(issues, issueSeparator))
}

// collectIssues walks v and returns the control-character issues found,
// each already formatted as a full path suffix. hasParent tells it (and
// whoever it delegates to) whether some ancestor segment already precedes
// this value in the final path: true everywhere except the single top-level
// call from ScanControlChars, so a struct field or a bare string at the
// absolute root renders as "Name: ..." / "contains invalid characters"
// instead of the connector-only ".Name: ..." / ": contains invalid
// characters" a nested one needs to attach to its parent's segment.
func collectIssues(v reflect.Value, hasParent bool) []string {
	v = dereferenceValue(v)
	if !v.IsValid() {
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		if ContainsControlChars(v.String()) {
			if hasParent {
				return []string{messageConnector + controlCharsMessage}
			}
			return []string{controlCharsMessage}
		}
	case reflect.Struct:
		return collectStructIssues(v, hasParent)
	case reflect.Slice, reflect.Array:
		return collectSliceIssues(v)
	case reflect.Map:
		return collectMapIssues(v)
	default:
		// bool, int, float, complex, chan, func, unsafe pointer: none of these
		// can carry a control character, so there's nothing to check.
	}
	return nil
}

// dereferenceValue unwraps pointers and interfaces until it reaches the
// underlying concrete value, or an invalid Value for a nil pointer/interface.
func dereferenceValue(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// collectSliceIssues never needs its own hasParent: a bracketed index like
// "[0]" reads fine whether or not something precedes it, so there's nothing
// to trim either way. Its elements always have this slice as their parent.
func collectSliceIssues(v reflect.Value) []string {
	if !mayContainString(v.Type().Elem().Kind()) {
		return nil
	}
	var issues []string
	for i := 0; i < v.Len(); i++ {
		itemIssues := collectIssues(v.Index(i), true)
		issuesLen := len(itemIssues)
		if issuesLen == 0 {
			continue
		}
		prefix := indexOpen + strconv.Itoa(i) + indexClose
		issues = slices.Grow(issues, issuesLen)
		for _, issue := range itemIssues {
			issues = append(issues, prefix+issue)
		}
	}
	return issues
}

// collectMapIssues never needs its own hasParent for the same reason as
// collectSliceIssues: a bracketed key never needs trimming. Its keys and
// values always have this map as their parent.
func collectMapIssues(v reflect.Value) []string {
	t := v.Type()
	checkKeys := mayContainString(t.Key().Kind())
	checkValues := mayContainString(t.Elem().Kind())
	if !checkKeys && !checkValues {
		return nil
	}
	var issues []string
	for _, key := range v.MapKeys() {
		var keyIssues, valueIssues []string
		issuesLen := 0
		if checkKeys {
			keyIssues = collectIssues(key, true)
			issuesLen += len(keyIssues)
		}
		if checkValues {
			valueIssues = collectIssues(v.MapIndex(key), true)
			issuesLen += len(valueIssues)
		}
		if issuesLen == 0 {
			continue
		}
		prefix := indexOpen + formatMapKey(key) + indexClose
		issues = slices.Grow(issues, issuesLen)
		for _, issue := range keyIssues {
			issues = append(issues, prefix+mapKeySuffix+issue)
		}
		for _, issue := range valueIssues {
			issues = append(issues, prefix+issue)
		}
	}
	return issues
}

// formatMapKey renders a map key for use in an error path. reflect.Value.String
// only returns the underlying value for Kind() == String; every other kind
// falls back to Interface() so e.g. an int key reads as "42" instead of the
// generic "<int Value>" placeholder String() would otherwise produce.
func formatMapKey(key reflect.Value) string {
	if key.Kind() == reflect.String {
		return key.String()
	}
	return fmt.Sprintf("%v", key.Interface())
}

// collectStructIssues is the one place hasParent actually changes the
// prefix: a field's own name only needs a leading "." when this struct has
// something before it in the path. A field always has this struct as its
// own parent, regardless of whether the struct itself does.
func collectStructIssues(v reflect.Value, hasParent bool) []string {
	fields := cachedScannableFields(v.Type())
	if len(fields) == 0 {
		return nil
	}
	var issues []string
	for _, field := range fields {
		fieldIssues := collectIssues(v.FieldByIndex(field.index), true)
		issuesLen := len(fieldIssues)
		if issuesLen == 0 {
			continue
		}
		prefix := field.name
		if hasParent {
			prefix = fieldSeparator + prefix
		}
		issues = slices.Grow(issues, issuesLen)
		for _, issue := range fieldIssues {
			issues = append(issues, prefix+issue)
		}
	}
	return issues
}

// mayContainString reports whether a value of kind k could possibly hold, or
// recursively reach, a string. Kinds that structurally can never carry a
// string — bool, every numeric kind, chan, func, unsafe pointer — are
// excluded here once per struct type when the field cache is built, instead
// of being walked and discarded by collectIssues on every single call.
func mayContainString(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Struct, reflect.Slice, reflect.Array,
		reflect.Map, reflect.Pointer, reflect.Interface:
		return true
	default:
		return false
	}
}

func getScannableFields(t reflect.Type) []cachedField {
	var fields []cachedField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if !mayContainString(f.Type.Kind()) {
			continue
		}
		fields = append(fields, cachedField{
			index: f.Index,
			name:  f.Name,
		})
	}
	return fields
}

func cachedScannableFields(t reflect.Type) []cachedField {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.([]cachedField)
	}
	// getScannableFields is pure and side-effect free, so a concurrent cache
	// miss just recomputes the same result twice — LoadOrStore alone is
	// enough to make the stored value consistent, with no separate lock.
	fields := getScannableFields(t)
	actual, _ := fieldCache.LoadOrStore(t, fields)
	return actual.([]cachedField)
}
