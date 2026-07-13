# CLAUDE.md

## Project Overview

<!-- One paragraph: what this service/tool does and who consumes it. -->

## Commands

```bash
make build          # or: go build ./...
make test           # or: go test ./...
make lint           # golangci-lint run
go test -run TestName ./path/to/pkg   # single test
go vet ./...
```

## Architecture

<!-- Adjust to actual layout -->

- `cmd/` — entrypoints (one subdir per binary)
- `internal/` — private application code
- `pkg/` — public reusable packages (only if intentionally exported)
- `api/` — protobuf / OpenAPI definitions
- `configs/` — configuration files

## Conventions

- Go version: 1.22+ (match `go.mod`; do not change it without asking)
- Formatting: `gofmt` / `goimports` — never hand-format
- Errors: wrap with `fmt.Errorf("context: %w", err)`; no `panic` outside `main` or init paths
- Errors are values: check every error; don't discard with `_` unless justified with a comment
- Naming: idiomatic Go (short receiver names, `MixedCaps`, no `Get` prefixes)
- Context: `context.Context` is always the first parameter of functions that do I/O
- Concurrency: prefer channels/errgroup over raw goroutines; every goroutine must have a clear exit path
- Logging: use the project logger (`log/slog` by default); no `fmt.Println` in non-test code

## Testing

- Table-driven tests preferred
- Use `t.Parallel()` where safe
- Mocks: prefer small interfaces defined at the consumer; avoid heavy mocking frameworks
- Run `go test -race ./...` before considering work done

## Dependencies

- Standard library first; justify any new dependency
- Run `go mod tidy` after changing imports
- Do not upgrade major versions of dependencies without asking

## Git workflow

- Every new feature goes on its own branch off `main`, named `feature/<short-name>`.
- One feature per branch: a feature branch must contain exactly one feature — never bundle
  multiple features, or mix a feature with unrelated fixes/refactors. Unrelated fixes and
  chores get their own branch too (`fix/<name>`, `chore/<name>`).
- Never commit a new feature directly to `main`; open the feature branch first.

## What NOT to do

- Don't create new top-level packages without asking
- Don't add global state (`init()` with side effects, package-level mutable vars)
- Don't commit generated files unless the repo already does
- Don't refactor unrelated code while fixing a bug
