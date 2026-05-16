package provider

// lazyWrapper is a small generic helper that wraps a Lazy[T] instance and
// exposes a simple `get()` helper for concrete wrappers to call. This
// reduces boilerplate in the per-feature wrapper types while keeping the
// actual per-feature methods typed to the target API.
type lazyWrapper[T any] struct {
	lazy *Lazy[T]
}

func newLazyWrapper[T any](ctor func() (T, error)) *lazyWrapper[T] {
	return &lazyWrapper[T]{lazy: NewLazy(ctor)}
}

func (w *lazyWrapper[T]) get() (T, error) { return w.lazy.Get() }
