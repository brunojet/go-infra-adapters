//go:build debug

package debugassert

// Assert panics when condition is false. Enabled only in debug builds.
func Assert(ok bool, msg string) {
	if !ok {
		panic(msg)
	}
}
