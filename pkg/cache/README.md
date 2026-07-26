# Cache Package

Distributed cache abstraction with in-memory (local) and distributed (Valkey) adapters. Type-safe, generic API with support for cache-aside loading, TTL expiry, and concurrent request deduplication.

## Motivation

Caching is critical for performance but often implemented ad-hoc with locks, TTL logic, and repeated loading. This package provides:

- **Type-safe generic API**: `Cache[T]` ensures compile-time type safety.
- **Contract-based**: Switch between adapters (in-memory, Valkey) without code changes.
- **Cache-aside pattern**: Built-in `GetOrSet` with automatic deduplication of concurrent misses.
- **Graceful degradation**: Read errors don't fail the request; they degrade to a cache miss.

## API Overview

```go
type Cache[T any] interface {
    Get(ctx context.Context, key string) (*T, bool, error)
    Set(ctx context.Context, key string, val *T, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    GetOrSet(ctx context.Context, key string, ttl time.Duration, load Loader[T]) (*T, error)
    HealthCheck(ctx context.Context) error
}

type Loader[T any] func(ctx context.Context) (*T, error)
```

### Methods

#### Get
```go
val, hit, err := cache.Get(ctx, "user:123")
```
Returns the cached value and whether it was a hit. A miss (absent or expired key) returns `(*T, false, nil)`, not an error. An actual backend error (e.g., distributed cache connection failure) returns `(*T, false, err)`.

#### Set
```go
err := cache.Set(ctx, "user:123", &user, 5*time.Minute)
```
Caches a value with a TTL. A `ttl` of `0` means no expiry. Returns an error only if the write itself fails (rare in local cache, more common in Valkey).

#### Delete
```go
err := cache.Delete(ctx, "user:123")
```
Removes a key. Deleting an absent key is not an error.

#### Exists
```go
exists, err := cache.Exists(ctx, "user:123")
```
Checks whether a key is present and unexpired, without retrieving its value.

#### GetOrSet (cache-aside)
```go
user, err := cache.GetOrSet(ctx, "user:123", 5*time.Minute, func(ctx context.Context) (*User, error) {
    return db.FindUser(ctx, 123)  // Only called on cache miss
})
```
The Swiss Army knife. On a cache hit, returns the cached value. On a miss, runs the `load` function, caches the result, and returns it. If `load` errors, nothing is cached and the error is propagated.

**Concurrency:** Concurrent `GetOrSet` calls on the **same key** are deduplicated. Only one caller (the "leader") runs the `load` function; others block and reuse its result. Distinct keys load in parallel.

```go
// 10 goroutines call GetOrSet("user:123") at the same time
// Expectation: load is called exactly 1 time, 9 others wait for the result
results := make(chan *User, 10)
for i := 0; i < 10; i++ {
    go func() {
        u, _ := cache.GetOrSet(ctx, "user:123", 5*time.Minute, loadUser)
        results <- u
    }()
}
```

#### HealthCheck
```go
err := cache.HealthCheck(ctx)
```
Verifies the cache backend is reachable. Safe for initialization checks.

## Adapters

### Local (In-Memory)

Single-process cache backed by a map and a Mutex. No network, no serialization. Ideal for:
- Dev/test environments
- Single-instance services
- Session caches that don't need to be shared

```go
cache := local.NewCache[User]()
```

#### Concurrent Characteristics
- Read-heavy workloads: Mutex is a bottleneck compared to RWMutex, but vastly simpler and safer against panics.
- TTL expiry: Lazy evaluation on access; no background janitor. Keys written with TTL that are never read persist until overwritten.

### Valkey (Distributed)

Redis-compatible distributed cache via Valkey client. Ideal for:
- Multi-instance services
- Long-lived session/token caches
- High-traffic caches where throughput > locality

Coming in phase 2. API is identical; swap `local.NewCache` → `valkey.NewCache`.

## Examples

### Basic Usage: User Service

```go
package userservice

import (
    "context"
    "time"
    "github.com/brunojet/go-infra-adapters/v4/pkg/cache/local"
)

type User struct {
    ID   int
    Name string
    Email string
}

type Service struct {
    cache local.Cache[User]
    db    *Database
}

func NewService(db *Database) *Service {
    return &Service{
        cache: local.NewCache[User](),
        db:    db,
    }
}

func (s *Service) GetUser(ctx context.Context, id int) (*User, error) {
    key := fmt.Sprintf("user:%d", id)
    return s.cache.GetOrSet(ctx, key, 5*time.Minute, func(ctx context.Context) (*User, error) {
        return s.db.FindUser(ctx, id)  // DB hit only on cache miss
    })
}
```

### With Structured Logging

```go
import (
    "log/slog"
    "github.com/brunojet/go-infra-adapters/v4/pkg/cache/local"
)

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

cache := local.NewCache[User](
    local.WithLogger(logger),
)
// Cache read/write errors are logged as warnings, never propagated
```

### Invalidation Pattern

```go
func (s *Service) UpdateUser(ctx context.Context, id int, patch *UserPatch) error {
    user, err := s.db.UpdateUser(ctx, id, patch)
    if err != nil {
        return err
    }
    
    // Invalidate the cache entry
    key := fmt.Sprintf("user:%d", id)
    s.cache.Delete(ctx, key)  // No-op if not cached
    
    return nil
}
```

### Batch Operations (Future)

When Valkey adapter lands, batch operations will be available:

```go
// Hypothetical future API (not yet implemented)
users, err := cache.MGet(ctx, []string{"user:1", "user:2", "user:3"})
// Single round-trip for 3 keys with Valkey; in-memory uses sequential gets
```

## Configuration

### In-Memory Cache

No configuration options beyond logger:

```go
cache := local.NewCache[T](
    local.WithLogger(customLogger),  // Optional; defaults to noop logger
)
```

The noop logger discards all output (zero overhead when no logger is passed).

### Valkey Cache (Phase 2)

```go
cache := valkey.NewCache[T](
    valkey.WithAddr("localhost:6379"),
    valkey.WithLogger(logger),
)
```

## Error Handling

### Cache Errors Don't Fail Requests

By design, cache operations are optional:

- **Read error (Get)**: Logged as a warning, treated as a cache miss. Caller's `GetOrSet` load function runs.
- **Write error (Set/GetOrSet)**: Logged as a warning. The loaded value is still available to return.

Example: if Valkey is down but your app uses cache-aside correctly, requests still succeed (just slower).

```go
// Even if cache.Get fails, GetOrSet degrades gracefully:
user, err := cache.GetOrSet(ctx, "user:123", 5*time.Minute, func(ctx context.Context) (*User, error) {
    // This runs if Get fails OR if key is missing
    return db.FindUser(ctx, 123)
})
// err is only from db.FindUser, never from the cache backend
```

### Load Errors Are Propagated

Only origin/backend errors are propagated. This is intentional:

```go
user, err := cache.GetOrSet(ctx, "user:123", 5*time.Minute, func(ctx context.Context) (*User, error) {
    return db.FindUser(ctx, 123)  // If this errors, the error propagates
})
if err != nil {
    // This is a real database error, not a cache error
    return fmt.Errorf("failed to load user: %w", err)
}
```

## Concurrency & Safety

### Panic Safety

All mutex operations use `defer` to guarantee unlocks even if a handler panics. In-memory cache is robust against panics in user code.

### Race-Free GetOrSet

Concurrent `GetOrSet` calls on the same key are deduplicated via `singleflight.Group`:

```
Caller A, B, C all call GetOrSet("user:123") simultaneously:
  A gets the lock, runs load(), waits for result
  B and C block in singleflight, then reuse A's result
  All three return the same *User with zero duplicate loads
```

Different keys are unaffected:

```
Caller A calls GetOrSet("user:123")  ← runs load() once
Caller B calls GetOrSet("user:456")  ← runs load() in parallel (no blocking)
```

### TTL Expiry

Lazy evaluation: expired keys are deleted only when accessed. Keys with TTL that are never read again persist in the map (acceptable for local cache; Valkey has server-side expiry).

## Type Safety & Serialization

`Cache[T]` is generic and type-safe:

```go
userCache := local.NewCache[User]()
postCache := local.NewCache[Post]()

user, _ := userCache.Get(ctx, "u1")  // *User
post, _ := postCache.Get(ctx, "p1")  // *Post
// No type assertions, no runtime reflection
```

For distributed adapters (Valkey), `T` must be JSON-serialisable:

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
// works with Valkey: JSON marshalled to bytes, stored, unmarshalled on Get
```

In-memory cache stores `T` directly (no serialization cost).

## Testing

Mock the `Cache[T]` interface for unit tests:

```go
type mockCache[T any] struct {
    data map[string]*T
}

func (m *mockCache[T]) Get(ctx context.Context, key string) (*T, bool, error) {
    val, ok := m.data[key]
    return val, ok, nil
}
// ... implement other methods
```

Or use the real local cache in integration tests (it's fast and has no dependencies).

## Migration Path (Local → Valkey)

No code changes required:

```go
// Dev / single-instance
cache := local.NewCache[User]()

// Production / multi-instance (switch only the import)
cache := valkey.NewCache[User](valkey.WithAddr("valkey:6379"))

// Calling code is identical
user, err := cache.GetOrSet(ctx, "user:123", 5*time.Minute, loadUser)
```

## Performance Notes

### Local Cache
- **Get**: O(1) map lookup + mutex lock.
- **Set**: O(1) map write + mutex lock.
- **GetOrSet (hit)**: O(1), singleflight skipped.
- **GetOrSet (miss, dedup)**: Caller blocks until leader finishes + singleflight overhead (~microseconds).

No GC pressure from expiry (lazy deletion). Keys written with TTL that are never accessed stick around (planned: optional background janitor for phase 2).

### Valkey (Phase 2)
- Network latency dominates.
- Batch operations (MGet/MSet) reduce round-trips.
- Server-side TTL expiry (no lazy evaluation overhead).

---

See [contracts.go](contracts/contracts.go) for the full interface definition.
