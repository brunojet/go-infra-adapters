package validation

import "testing"

type benchSimplePayload struct {
	Name  string
	Email string
}

type benchNestedPayload struct {
	Comment string
}

type benchComplexPayload struct {
	ID      int
	Active  bool
	Name    string
	Emails  []string
	Meta    map[string]string
	Inner   benchNestedPayload
	Pointer *benchNestedPayload
	Extra   interface{}
}

type benchScalarOnly struct {
	ID    int
	Score float64
	Count int64
}

func newBenchComplexPayload() benchComplexPayload {
	return benchComplexPayload{
		ID:      1,
		Active:  true,
		Name:    "bruno",
		Emails:  []string{"a@b.com", "c@d.com", "e@f.com"},
		Meta:    map[string]string{"token": "abc", "role": "admin"},
		Inner:   benchNestedPayload{Comment: "all good"},
		Pointer: &benchNestedPayload{Comment: "also fine"},
		Extra:   "dynamic value",
	}
}

func BenchmarkScanControlChars_String(b *testing.B) {
	body := "a reasonably normal string value without anything dangerous in it"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

func BenchmarkScanControlChars_SimpleStruct(b *testing.B) {
	body := benchSimplePayload{Name: "bruno", Email: "bruno@example.com"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

func BenchmarkScanControlChars_ScalarOnlyStruct(b *testing.B) {
	body := benchScalarOnly{ID: 1, Score: 9.5, Count: 100}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

func BenchmarkScanControlChars_ComplexStruct(b *testing.B) {
	body := newBenchComplexPayload()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

func BenchmarkScanControlChars_SliceOfStructs(b *testing.B) {
	body := make([]benchSimplePayload, 50)
	for i := range body {
		body[i] = benchSimplePayload{Name: "bruno", Email: "bruno@example.com"}
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

func BenchmarkScanControlChars_SliceOfInts(b *testing.B) {
	body := make([]int, 50)
	for i := range body {
		body[i] = i
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

func BenchmarkScanControlChars_Map(b *testing.B) {
	body := map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

func BenchmarkScanControlChars_InvalidStruct(b *testing.B) {
	body := benchSimplePayload{Name: "bad\tvalue", Email: "bruno@example.com"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

// BenchmarkScanControlChars_ManySubIssues stresses a single field that by
// itself contributes many issues at once (every element of a 20-item slice
// is dirty), the exact case slices.Grow(issues, len(fieldIssues)) targets:
// without it, appending those 20 items one at a time into a fresh nil slice
// would double-and-copy repeatedly (nil->1->2->4->8->16->32) instead of
// growing to the right capacity in one shot.
func BenchmarkScanControlChars_ManySubIssues(b *testing.B) {
	items := make([]string, 20)
	for i := range items {
		items[i] = "bad\tvalue"
	}
	body := struct{ Items []string }{Items: items}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ScanControlChars(body)
	}
}

// BenchmarkScanControlChars_Parallel hammers the shared field cache
// (fieldCache, a sync.Map) from many goroutines at once. Run with -race to
// catch any concurrency bug in cachedScannableFields/getScannableFields.
func BenchmarkScanControlChars_Parallel(b *testing.B) {
	body := newBenchComplexPayload()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = ScanControlChars(body)
		}
	})
}

// BenchmarkScanControlChars_ParallelColdCache validates concurrent
// first-touch access to the field cache: many goroutines racing to populate
// the cache entry for the same brand-new type at once, on every b.N
// iteration via distinct anonymous struct types would defeat -benchmem
// comparisons, so instead this drives many *different* pre-existing types
// concurrently to stress LoadOrStore's dedup path under real contention.
func BenchmarkScanControlChars_ParallelMixedTypes(b *testing.B) {
	bodies := []any{
		"plain string",
		benchSimplePayload{Name: "bruno", Email: "bruno@example.com"},
		benchScalarOnly{ID: 1, Score: 9.5, Count: 100},
		newBenchComplexPayload(),
		map[string]string{"a": "1", "b": "2"},
		[]int{1, 2, 3},
	}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = ScanControlChars(bodies[i%len(bodies)])
			i++
		}
	})
}
