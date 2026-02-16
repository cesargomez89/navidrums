# Architecture

Navidrums follows a layered architecture with clear separation of concerns.

> **Quick Reference:** See @AGENTS.md for job lifecycle, coding conventions, and critical rules.

## Package Structure

```
cmd/server/           # Application entry point
internal/
├── app/              # Application services (JobService, Downloader, etc.)
├── catalog/          # Provider interface and implementations
├── config/           # Configuration management
├── constants/        # Application constants
├── domain/           # Domain models (Job, Track, Album, etc.)
├── downloader/       # Worker implementation
├── http/             # HTTP handlers and routing
├── logger/           # Structured logging
├── server/           # HTTP server setup
├── storage/          # Filesystem operations
├── store/            # Database repository
└── tagging/          # Audio file metadata tagging
web/                  # Embedded UI templates and assets
```

## Layer Flow

```
┌─────────────────────────────────────────────────────────────┐
│                         UI / Web                            │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│              HTTP Handlers (internal/http)                  │
│         - Request parsing, HTML rendering                   │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│              Application Services (internal/app)            │
│         - JobService, Downloader, PlaylistGenerator         │
│         - Business logic orchestration                      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────┬───────────────┬───────────────────────┐
│   Repository        │   Providers   │   Filesystem          │
│   (internal/store)  │(internal/     │   (internal/storage)  │
│   - Job persistence │ catalog)      │   - File operations   │
│   - Track state     │   - External  │   - Path sanitization │
│                     │     API calls │   - Directory mgmt    │
└─────────────────────┴───────────────┴───────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│              Worker (internal/downloader)                   │
│         - Background job processing                         │
│         - Download execution                                │
│         - Tagging integration                               │
└─────────────────────────────────────────────────────────────┘
```

## Layer Responsibilities

### Handlers (internal/http)
- HTTP parsing and response formatting only
- Template rendering (HTML fragments for HTMX)
- Route registration
- No business logic

### Services (internal/app)
- Business workflows and orchestration
- JobService: Job lifecycle management
- Downloader: Track download with retry logic
- PlaylistGenerator: M3U playlist file creation
- AlbumArtService: Cover art download
- Storage utilities: File hashing, path building, sanitization

### Repository (internal/store)
- Persistent state and queries
- Job CRUD operations (minimal work queue state)
- Track persistence (full metadata + download state)
- Settings storage
- Database migrations with WAL mode for concurrency

### Providers (internal/catalog)
- External API adapters
- Music catalog interface
- Stream fetching
- Lyrics retrieval

### Filesystem (internal/storage)
- All local disk I/O
- Path sanitization
- Directory management

### Worker (internal/downloader)
- Background execution engine
- Concurrent job processing
- Job decomposition (albums → tracks)
- Download and tagging coordination

### Tagging (internal/tagging)
- Audio file metadata writing
- FLAC/MP3 tag support
- Album art embedding

## Concurrency Model

- Workers poll database for jobs at regular intervals
- Semaphore controls max concurrent downloads (default: 2)
- Each job runs in its own goroutine
- Container jobs (album/playlist/artist) spawn child track jobs
- Context cancellation stops downloads gracefully
