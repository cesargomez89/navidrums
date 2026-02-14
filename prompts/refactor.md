# Refactoring Prompts for Navidrums

> **Context Required:** @AGENTS.md
> **Optional:** @ARCHITECTURE.md for detailed layer info, @DOMAIN.md for data models

## Extract Service Method

I need to refactor the `{ServiceName}` service to extract `{methodName}` logic into a separate method.

Context:
- The method is currently doing too much (violating single responsibility)
- Located in `internal/app/{service_file}.go`
- Current method handles: {describe current responsibilities}

Requirements:
1. Extract the `{specific_logic}` logic into a new unexported method
2. Keep the public method signature unchanged for backward compatibility
3. Follow the error handling pattern: `fmt.Errorf("failed to X: %w", err)`
4. Ensure imports follow the order: stdlib → third-party → internal
5. Run `go fmt` and `golangci-lint run` after changes

Example target:
```go
// Before: GetTrack does metadata fetch, download check, and path building
// After: GetTrack calls getTrackPath() and checkDownloaded() helpers
```

---

## Reduce Handler Complexity

The `{HandlerName}` handler in `internal/http/{handlers_file}.go` is too complex and needs refactoring.

Issues:
- Handler contains business logic that should be in services
- Direct repository access found at line {line_number}
- Goroutine spawning in handler (violation of Critical Don'ts)

Refactor steps:
1. Move business logic to `internal/app/{appropriate_service}.go`
2. Handler should only: parse request → call service → format response
3. No direct database access or provider calls from handler
4. Use existing service methods or add new ones to services layer

---

## Standardize Error Handling

Find all places in `{package}` that don't follow the error handling conventions and refactor them.

Current issues found:
- Raw errors returned without wrapping
- Using `errors.New()` instead of exported error variables
- Inconsistent error messages

Requirements:
1. Services: Use `fmt.Errorf("failed to {action}: %w", err)`
2. Define exported errors with `Err` prefix: `var ErrJobCancelled = errors.New("job cancelled")`
3. Handlers: Use `http.Error()` with appropriate status codes
4. Workers: Add `defer` with `recover()` for panic handling

Run tests after: `go test ./...`

---

## Rename for Clarity

Rename `{oldName}` to `{newName}` across the codebase for better clarity.

Scope:
- Type names (PascalCase)
- Function names (camelCase for unexported, PascalCase for exported)
- Variable names (camelCase)
- Update all references

Files likely affected:
- `internal/domain/*.go`
- `internal/app/*.go`
- `internal/store/*.go`
- `internal/http/*.go`

After renaming:
```bash
go build ./...
go test ./...
go fmt ./...
```
