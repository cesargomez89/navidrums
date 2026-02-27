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
| `SUBDIR_TEMPLATE` | `{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}` | Template for subdirectory and filename structure |
| `PROVIDER_URL` | http://127.0.0.1:8000 | Music catalog API URL |
| `QUALITY` | LOSSLESS | Audio quality |
| `LOG_LEVEL` | info | Logging level |
| `LOG_FORMAT` | text | Log format (text, json) |
| `NAVIDRUMS_USERNAME` | navidrums | Basic auth username |
| `NAVIDRUMS_PASSWORD` | (empty) | Basic auth password |
| `CACHE_TTL` | 12h | Provider response cache TTL |
| `MUSICBRAINZ_URL` | https://musicbrainz.org/ws/2 | MusicBrainz API endpoint for metadata enrichment |
| `RATE_LIMIT_REQUESTS` | 60 | Maximum requests per rate limit window |
| `RATE_LIMIT_WINDOW` | 1m | Rate limit time window |
| `RATE_LIMIT_BURST` | 10 | Burst requests allowed beyond rate limit |
| `DISABLE_RATE_LIMIT` | false | Disable rate limiting (use when behind Cloudflare) |
| `SKIP_AUTH` | false | Disable authentication entirely |

**Template Variables:** Uses Go's `text/template` syntax. Available: `AlbumArtist`, `OriginalYear`, `Album`, `Disc`, `Track`, `Title`. File extension appended automatically.

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
queued → running → completed | failed | cancelled
```

**Container jobs** (album/playlist/artist): fetch tracks from provider → create Track records → create child track jobs → complete

**Track jobs**: lookup stored Track metadata → check if downloaded → download stream → tag → save art → update Track record

Rules:
- Cancelled jobs must stop work
- Jobs cannot return to queued
- Workers persist all state transitions

---

## Data Invariants

- Track file must exist before tagging
- Providers are stateless, responses not stored raw
- Container jobs decompose into track jobs via Track records
- Duplicate downloads prevented via tracks table (provider_id unique)
- Deleting job does not delete files
- Job.SourceID links to Track.ProviderID
- Tracks table stores full metadata; Jobs table stores minimal state

---

## Domain Models

### Job (Work Queue)
Minimal state for work tracking:
- `ID`, `Type`, `Status`, `SourceID`, `Progress`, `Error`, timestamps
- Status: queued | running | completed | failed | cancelled
- Types: track, album, playlist, artist, sync_file, sync_musicbrainz, sync_hifi

### Track (Download Domain)
Full metadata for downloaded/pending tracks:
- Identity: `ID`, `ProviderID`, `AlbumID`, `ReleaseID`, `RecordingID`
- Basic: `Title`, `Artist`, `Album`, `AlbumArtist`, `TrackNumber`, `DiscNumber`, `Year`, `Duration`
- Artist info: `Artists`, `AlbumArtists`, `ArtistIDs`, `AlbumArtistIDs`
- Genre/Label: `Genre`, `Mood`, `Style`, `Tags`, `Label`, `Compilation`
- Release: `ReleaseDate`, `ReleaseType`, `Barcode`, `CatalogNumber`
- Audio: `BPM`, `Key`, `KeyScale`, `ReplayGain`, `Peak`, `AudioQuality`, `AudioModes`
- Credit: `Credit`, `Composer`
- Lyrics: `Lyrics`, `Subtitles`
- Commercial: `ISRC`, `Copyright`, `Version`, `Description`, `URL`, `AlbumArtURL`
- Explicit: `Explicit`
- Position: `TotalTracks`, `TotalDiscs`
- Processing: `Status`, `Error`, `ParentJobID`
- File: `FilePath`, `FileExtension`, `FileHash`, `ETag`
- Verification: `LastVerifiedAt`, `CompletedAt`
- Status: missing | queued | downloading | downloaded | processing | completed | failed

### CatalogTrack (Provider Data)
Used by catalog providers for search results:
- Identity: `ID`, `ProviderID`, `AlbumID`, `ArtistID`
- Basic: `Title`, `Artist`, `Album`, `AlbumArtist`, `TrackNumber`, `DiscNumber`, `Year`, `Duration`
- Artist info: `Artists`, `AlbumArtists`, `ArtistIDs`, `AlbumArtistIDs`
- Genre/Label: `Genre`, `Label`, `Compilation`
- Release: `ReleaseDate`, `ReleaseType`, `Barcode`, `CatalogNumber`
- Audio: `BPM`, `Key`, `KeyScale`, `ReplayGain`, `Peak`, `AudioQuality`, `AudioModes`
- Lyrics: `Lyrics`, `Subtitles`
- Commercial: `ISRC`, `Copyright`, `Composer`, `Version`, `Description`, `URL`, `AlbumArtURL`
- Explicit: `ExplicitLyrics`
- Position: `TotalTracks`, `TotalDiscs`
- Converted to Track when persisting to database

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
- Stuck jobs (running) are reset on startup
- Two-table architecture: `jobs` (work queue) + `tracks` (metadata)

