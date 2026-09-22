# CLAUDE.md

## Project Overview

jira-tui is a terminal UI for Jira Cloud, built on Bubble Tea v2. It is a single
binary distributed via Homebrew and GitHub releases. Time tracking is backed by
Tempo and is optional; everything else talks to the Jira Cloud REST API v3.

## Commands

```bash
make build          # or: go build ./...
make test           # or: go test ./...
make lint           # golangci-lint run
go test -run TestName ./path/to/pkg   # single test
go vet ./...
```

## Architecture

- `cmd/jira-tui/` — the whole TUI (package `main`): the model struct, `Update`
  and `View` dispatch by `viewMode`, one file per view/modal, and `commands.go`
  for every async `tea.Cmd` and message type
- `internal/jira/` — REST client, domain types, Markdown↔ADF conversion
- `internal/ui/` — styles, themes, column width math
- `internal/config/` — environment configuration
- `internal/store/` — local JSON state (issue tags) and XDG paths

## Conventions

- Go version: match `go.mod` (currently 1.25.8); do not change it without asking
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
