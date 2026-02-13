# Agent Guidelines for Navidrums

## Project Overview
Navidrums is a Go web application for browsing and downloading music. It uses Chi router, HTMX for frontend, SQLite for persistence, and integrates with an external Hifi API.

## Build Commands

```bash
# Build the server binary
go build -o navidrums ./cmd/server

# Build with optimizations (for releases)
go build -o navidrums -ldflags="-s -w" ./cmd/server

# Cross-compilation examples
GOOS=linux GOARCH=amd64 go build -o navidrums-linux-amd64 -ldflags="-s -w" ./cmd/server
GOOS=darwin GOARCH=arm64 go build -o navidrums-darwin-arm64 -ldflags="-s -w" ./cmd/server
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/config
go test ./internal/filesystem

# Run a single test
go test ./internal/config -run TestLoad

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

## Lint/Format Commands

```bash
# Format all Go files
go fmt ./...

# Run go vet (static analysis)
go vet ./...

# Import organization (if goimports is installed)
goimports -w .
```

## Code Style Guidelines

### Imports
- Group imports: stdlib first, then third-party, then internal packages
- Separate groups with blank lines
- Use full module path for internal imports: `github.com/cesargomez89/navidrums/internal/...`

Example:
```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/go-chi/chi/v5"

    "github.com/cesargomez89/navidrums/internal/config"
)
```

### Naming Conventions
- **Types**: PascalCase (e.g., `JobService`, `DownloadConfig`)
- **Interfaces**: PascalCase with verb/noun (e.g., `Provider`, `Downloader`)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase (e.g., `maxConcurrent`, `jobCount`)
- **Constants**: Use `constants` package, PascalCase or ALL_CAPS for enums
- **Files**: snake_case.go (e.g., `job_service.go`)
- **Packages**: lowercase, short but clear (e.g., `repository`, `services`)

### Error Handling
- Wrap errors with context using `fmt.Errorf("...: %w", err)`
- Define sentinel errors at package level (e.g., `var ErrJobCancelled = errors.New(...)`)
- Return errors immediately rather than logging and continuing
- Validate configuration in `config.Validate()` with detailed error messages

### Types
- Use structs with tags for JSON marshaling
- Define domain types in `internal/models/models.go`
- Use type aliases for enums with constants (e.g., `type JobStatus string`)
- Prefer interfaces for dependencies (see `internal/providers/provider.go`)

### Structure
- Follow standard Go project layout (cmd/, internal/, web/)
- Business logic in `internal/services/`
- Data access in `internal/repository/`
- HTTP handlers in `internal/handlers/`
- Configuration in `internal/config/`
- Constants in `internal/constants/`

### Concurrency
- Use `sync.WaitGroup` for goroutine coordination
- Pass `context.Context` as first parameter
- Cap concurrent operations (default: 2 downloads, 6 for artist aggregation)
- Use channels for communication between goroutines

### Testing
- Use table-driven tests with `tests := []struct{...}`
- Name test cases descriptively (e.g., "valid config", "empty port")
- Clean up environment variables after tests (`defer os.Unsetenv()`)
- Test both success and error cases

### Comments
- Package comments should start with `// Package name ...`
- Export all public types and functions with doc comments
- Use inline comments sparingly, prefer self-documenting code

### Database
- Use `modernc.org/sqlite` driver
- Schema migrations in `internal/repository/schema.go`
- Repository pattern for data access

### HTTP Handlers
- Use Chi router for routing
- Handler methods receive `(w http.ResponseWriter, r *http.Request)`
- Use HTMX for partial page updates
- Return appropriate HTTP status codes

## Configuration
- Environment variables in `internal/config/config.go`
- Defaults defined in `internal/constants/constants.go`
- Validate all configuration on startup

## Running the Application
```bash
# With defaults
./navidrums

# With custom settings
PORT=9090 PROVIDER_URL=http://api:8000 ./navidrums
```

---

# API Specifications

This application provides a web interface for browsing and downloading music, consuming an external Hifi API for music data.

---

## Application API (Your Server)

### Base URL
Default: `http://localhost:8080`  
Override via `PORT` environment variable.

### Web Interface Routes

#### Navigation & Search
- **GET `/`** - Main search page
- **GET `/htmx/search`** - HTMX search endpoint
  - `q`: search query string
  - `type`: search type (`album`, `artist`, `playlist`, `track`) - default: `album`
  - Returns: HTML fragment with search results

#### Entity Pages
- **GET `/artist/{id}`** - Artist detail page with albums and tracks
- **GET `/album/{id}`** - Album detail page with track listing
- **GET `/playlist/{id}`** - Playlist detail page with tracks

#### Download Management
- **POST `/htmx/download/{type}/{id}`** - Queue a download job
  - `type`: resource type (`album`, `artist`, `playlist`, `track`)
  - `id`: resource identifier
- **GET `/queue`** - Active downloads queue page
- **GET `/htmx/queue`** - HTMX queue status (auto-refresh)
- **POST `/htmx/cancel/{id}`** - Cancel a queued job
  - `id`: job ID to cancel
- **GET `/history`** - Completed/failed downloads history (last 20 items)

#### Static Assets
- **GET `/static/*`** - Static files (CSS, JS, images)

### Configuration

#### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `navidrums.db` | SQLite database file path |
| `DOWNLOADS_DIR` | `~/Downloads/navidrums` | Download destination directory |
| `PROVIDER_URL` | `http://127.0.0.1:8000` | Hifi API base URL |
| `QUALITY` | `LOSSLESS` | Default download quality |
| `USE_MOCK` | `false` | Use mock provider for testing |
| `NAVIDRUMS_USERNAME` | `navidrums` | Basic auth username |
| `NAVIDRUMS_PASSWORD` | (required) | Basic auth password |

### Architecture Notes
- Uses **HTMX** for dynamic content updates without full page reloads
- Download jobs are processed asynchronously by background workers
- SQLite database tracks job states (pending, processing, completed, failed, cancelled)
- Artist downloads aggregate tracks from all albums with capped concurrency (6)

---

## Hifi API (External Service)

### Base URL
Default: `http://127.0.0.1:8000`  
Override via `PROVIDER_URL` environment variable.

For manual testing, you can use: `https://tidal-api.binimum.org`

### Endpoints

#### Track & Playback
- **GET `/info/`**: Get track metadata.
  - `id`: int (required)
- **GET `/track/`**: Get playback info/manifest.
  - `id`: int (required)
  - `quality`: str (optional: `HI_RES_LOSSLESS`, `LOSSLESS`, `HIGH`, `LOW`)
- **GET `/recommendations/`**: Get similar tracks.
  - `id`: int (required)
- **GET `/lyrics/`**: Get track lyrics.
  - `id`: int (required)

#### Search
- **GET `/search/`**: Search across types.
  - `s`: track query
  - `a`: artist query
  - `al`: album query
  - `v`: video query
  - `p`: playlist query

#### Artist
- **GET `/artist/`**: 
  - `id`: Get artist metadata + cover.
  - `f`: Get artist content (albums, EPs/Singles).
  - `skip_tracks`: bool (default `false`). If `true` with `f`, returns `toptracks` (15) instead of aggregating all tracks from all albums.
- **GET `/artist/similar/`**: Get similar artists.
  - `id`: int (required)
  - `cursor`: string/int (optional)

#### Album & Playlist
- **GET `/album/`**: Get album metadata + tracks.
  - `id`: int (required)
  - `limit`: int (default 100)
  - `offset`: int (default 0)
- **GET `/album/similar/`**: Get similar albums.
  - `id`: int (required)
  - `cursor`: string/int (optional)
- **GET `/playlist/`**: Get playlist metadata + tracks.
  - `id`: string (required)
  - `limit`: int (default 100)
  - `offset`: int (default 0)
- **GET `/mix/`**: Get mix items.
  - `id`: string (required)

#### Images
- **GET `/cover/`**: Get album cover URLs.
  - `id`: int (track/album id) OR `q`: search query

### Authentication
Requires `token.json` populated with Tidal credentials. Uses `auth.tidal.com` for token refreshes.
- Internal env vars: `CLIENT_ID`, `CLIENT_SECRET`, `REFRESH_TOKEN`, `USER_ID`.

### Design Notes
- Uses `COUNTRY_CODE` (default `US`) for all requests.
- Artist view uses a capped concurrency (6) for track aggregation when `skip_tracks=false`.

### JSON Examples
See the `api-examples/hifi-api/` directory for example responses from all endpoints.
