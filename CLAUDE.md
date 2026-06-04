# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Git Workflow (GitFlow)

This project follows a simplified GitFlow branching model:

```
feature/* → develop → main
```

**Branch structure:**
- `main` - Production-ready code, always stable
- `develop` - Integration branch, accumulates features
- `feature/*` - Feature branches from `develop` (e.g., `feature/add-retry-strategy`)
- `bugfix/*` - Bug fixes from `develop` (e.g., `bugfix/fix-circuit-breaker`)
- `release/*` - Release preparation from `develop` (optional, for version bumps)
- `hotfix/*` - Critical fixes directly from `main` (optional, for production emergencies)

**Workflow:**
1. Create feature branch: `git checkout -b feature/your-feature develop`
2. Make commits with conventional format: `feat:`, `fix:`, `refactor:`, `test:`
3. Push and open PR targeting `develop`
4. After review/merge to `develop`, periodically merge `develop` → `main` for releases
5. Tag releases on `main` with semantic versioning (e.g., `v3.1.0`)

**Commit conventions:**
- Use conventional commits: `type(scope): description`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`
- Keep subject under 50 chars, add body if "why" isn't obvious
- Example: `feat(retry): integrate retry strategy into AWS SDK`

---

## Project Structure

**go-infra-adapters** is a library providing cloud service abstractions with adapters per provider.

### Architecture

- **pkg/** - Public contracts and adapters (exported APIs)
  - `storage/` - Storage abstraction (S3 adapter)
  - `secret/` - Secret management abstraction (Secrets Manager adapter)
  - `cdn/` - CDN abstraction (CloudFront adapter)
  - `crypto/` - Cryptography operations (RSA)
  - `http_client/` - HTTP client abstraction (net/http adapter with middleware)
  - `retry/` - Retry strategy contracts

- **internal/** - Provider-specific implementations (not exported)
  - `storage/aws/s3/` - S3 client implementation
  - `secret/aws/` - Secrets Manager client implementation
  - `cdn/aws/` - CloudFront client implementation
  - `http_client/net_http/` - net/http adapter with middleware
  - `middleware/` - Middleware implementations (circuit breaker, retry)
  - `crypto/` - RSA implementation
  - `retry/` - Retry strategy implementation
  - `retryer/aws/` - AWS SDK native retry integration

- **debugassert/** - Debug-only assertions (compiled out in release builds)

### Design Patterns

1. **Contract-based abstraction** - Interfaces in `pkg/*/contracts` define APIs
2. **Adapter pattern** - Internal implementations provide provider-specific logic
3. **Middleware chain** - HTTP client supports configurable middleware (circuit breaker, retry)
4. **Mock generation** - Uses mockgen for contract testing
5. **Retry strategy** - AWS SDK native retry mechanism with custom configuration

---

## Common Commands

### Testing
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/storage/contracts/...

# Run with coverage
go test -cover ./...

# Run single test
go test -run TestName ./package
```

### Linting & Formatting
```bash
# Lint with golangci-lint (requires: brew install golangci-lint)
golangci-lint run

# Format code
go fmt ./...

# Format imports
goimports -w .

# Check for unused code
go vet ./...
```

### Dependencies
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Check for updates
go list -u -m all
```

### Building
```bash
# Build library (no binary, it's a package)
go build ./...

# List exported APIs
go doc ./pkg/storage
```

---

## Development Notes

- All public APIs are in `pkg/`. Internal logic is in `internal/`.
- Tests use **testify** assertions and **mockgen** for mocking.
- Linter config in `.golangci.yml` enforces conventions (revive, errcheck, gosec, etc.).
- The http client supports middleware (circuit breaker, custom retry logic).
- AWS SDK v2 is used with custom retry strategy integration.
- Debug assertions in `debugassert/` are compiled out in release builds.
