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

## Data Architecture

Navidrums uses a two-table design that separates work queue state from track metadata:

### Jobs Table (Work Queue)
Minimal state for tracking background work:
- `ID`, `Type`, `Status`, `SourceID`, `Progress`, `Error`, timestamps
- Status: `queued → running → completed | failed | cancelled`
- `SourceID` links to `Track.ProviderID`

### Tracks Table (Download Domain)
Full metadata and download state for all tracks:
- Identity: `ID`, `ProviderID`, `AlbumID`
- Metadata: Title, Artist, Album, TrackNumber, ISRC, Lyrics, etc.
- Extended: BPM, Key, ReplayGain, AudioQuality, etc.
- Processing: `Status`, `Error`, `ParentJobID`
- File: `FilePath`, `FileExtension`, `FileHash`, `ETag`
- Verification: `LastVerifiedAt`
- Status: `missing → queued → downloading → processing → completed | failed`

### Key Data Invariants
1. Track file must exist before tagging
2. Duplicate downloads prevented via unique `provider_id` constraint
3. Deleting job doesn't delete files
4. Job.SourceID links to Track.ProviderID
5. Container jobs decompose into track jobs via Track records

### Workflow
```
HTTP Request → App Workflow → Store State
Worker observes state → Downloader executes → Storage writes → App finalizes
```

See [DOMAIN.md](DOMAIN.md) for detailed domain model specifications.

---

## Metadata Enrichment

### Data Sources

**Hi-Fi API (Primary)**
All track metadata from the streaming service:
- Identity: `ProviderID`, `ISRC`, `AlbumID`
- Basic: `Title`, `Artist`, `Artists`, `Album`, `AlbumArtist`, `AlbumArtists`
- Position: `TrackNumber`, `DiscNumber`, `TotalTracks`, `TotalDiscs`
- Release: `Year`, `ReleaseDate`, `Genre`, `Label`, `Copyright`
- Audio: `BPM`, `Key`, `KeyScale`, `ReplayGain`, `Peak`, `AudioQuality`
- URLs: `URL`, `AlbumArtURL`

**MusicBrainz (Secondary Enrichment)**
Only fills empty fields - never overwrites existing Hi-Fi data:
- `Artist`, `Artists`, `ArtistIDs`
- `Title`, `Duration`, `Year`
- `Barcode`, `CatalogNumber`, `ReleaseType`
- `AlbumArtistIDs`, `AlbumArtists`
- `Composer`, `Genre`
- `ReleaseID` ← Exception: Always overwritten

### Precedence Rule

**Hi-Fi data > MusicBrainz data**

MusicBrainz uses a "fill-in-the-blanks" pattern (`worker.go:enrichFromMusicBrainz`):
```go
if track.Artist == "" && meta.Artist != "" {
    track.Artist = meta.Artist
}
```

MusicBrainz enrichment only triggers when `track.ISRC != ""`.

### Sync Job Types

| Job Type | Hi-Fi API | MusicBrainz | Behavior |
|----------|-----------|-------------|----------|
| `JobTypeSyncFile` | No | No | Re-tags file with existing DB metadata only |
| `JobTypeSync` | No | Yes (fill gaps) | MusicBrainz enrichment → update DB → re-tag |
| `JobTypeSyncHiFi` | Yes (overwrite) | Yes (fill gaps) | Hi-Fi refresh → MusicBrainz enrichment → update DB → re-tag |

### Sync Scenarios

| Action | Job Type | Description |
|--------|----------|-------------|
| Per-track "Sync to File" button | `JobTypeSyncFile` | Re-tags with current DB metadata |
| Per-track "Enrich from MusicBrainz" button | `JobTypeSync` | Fetches MusicBrainz, fills gaps, re-tags |
| Per-track "Enrich from Hi-Fi" button | `JobTypeSyncHiFi` | Fetches fresh Hi-Fi data, then MusicBrainz fills gaps, re-tags |
| "Sync All" | `JobTypeSyncHiFi` | Batch refresh from Hi-Fi + MusicBrainz enrichment for all completed tracks |

### Key Points

1. Initial download: Hi-Fi data written first, then MusicBrainz fills gaps
2. Resyncing via "Enrich from MusicBrainz" only fetches MusicBrainz - never re-fetches Hi-Fi data
3. "Enrich from Hi-Fi" and "Sync All" fetch fresh Hi-Fi data (overwrites all metadata), then MusicBrainz fills remaining gaps
4. Manual edits via form are saved before sync jobs run, so they're preserved (unless overwritten by Hi-Fi enrichment)
5. `ReleaseID` is the only field MusicBrainz can overwrite (for release grouping)
