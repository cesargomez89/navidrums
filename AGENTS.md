## What this project is

Navidrums is a download orchestrator and metadata browser. It is NOT a streaming server - all downloads happen asynchronously via background jobs and workers.

---

## Build/Test/Lint Commands

```bash
# Build
go build -o navidrums ./cmd/server

# Run
go run ./cmd/server

# Test
go test ./...
go test -race ./...

# Lint
golangci-lint run

# Format
go fmt ./...
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | HTTP server port |
| `DB_PATH` | navidrums.db | SQLite database path |
| `DOWNLOADS_DIR` | ~/Downloads/navidrums | Download destination |
| `PROVIDER_URL` | http://127.0.0.1:8000 | Music catalog API URL |
| `QUALITY` | LOSSLESS | Audio quality |
| `LOG_LEVEL` | info | Logging level |
| `LOG_FORMAT` | text | Log format (text, json) |
| `NAVIDRUMS_USERNAME` | navidrums | Basic auth username |
| `NAVIDRUMS_PASSWORD` | (empty) | Basic auth password |

---

## Code Style

**Imports order**: stdlib → third-party → internal (separate groups with blank lines)

**Naming**:
- PascalCase for types, interfaces, exported functions
- camelCase for variables, unexported functions
- `Err` prefix for exported errors (e.g., `ErrJobCancelled`)

**Error handling**:
- Services: `fmt.Errorf("failed to X: %w", err)`
- Handlers: `http.Error()` with appropriate status codes
- Workers: `defer` with `recover()` to catch panics

**Formatting**: `go fmt`, tabs, aim for 100 chars, hard limit 120

---

## Architecture Rules

**Flow**: http request → app workflow → store state
worker observes state → downloader executes → storage writes → app finalizes

**Allowed**:
```
handlers → services
services → repository, providers, storage
worker → services, providers, storage, tagging
```

**Forbidden**:
```
repository → services
providers → repository
handlers → repository, providers, storage
downloading inside handlers
spawning goroutines in handlers
```

**Filesystem writes ONLY in `internal/storage` package**

---

## Job Lifecycle (Invariant)

```
queued → resolving_tracks → downloading → completed | failed | cancelled
```

**Container jobs** (album/playlist/artist): resolve tracks → create child track jobs → complete

**Track jobs**: resolve metadata → check if downloaded → download stream → tag → save art → record

Rules:
- Cancelled jobs must stop work
- Jobs cannot return to queued
- Workers persist all state transitions

---

## Data Invariants

- Track file must exist before tagging
- Providers are stateless, responses not stored raw
- Downloads decompose into track jobs
- Duplicate downloads prevented via download tracking
- Deleting job does not delete files

---

## Implementing Features

Order matters:
1. Add/modify service (`internal/app`)
2. Update repository if needed (`internal/store`)
3. Extend worker (`internal/downloader`)
4. Update handler LAST (`internal/http`)

Never start from handlers.

---

## Critical Don'ts

- no downloads in http
- no goroutines in http
- no DB access outside store
- no blocking requests
- no provider calls from UI
- no file writes outside storage
- no job mutation outside app

---

## Project Notes

- The application requires a writable downloads directory
- Workers start automatically with the server
- SQLite database is created automatically on first run
- Stuck jobs (resolving_tracks, downloading) are reset on startup
