//go:build !debug

package debugassert

// Assert is a no-op in non-debug (release) builds.
func Assert(ok bool, msg string) {}
