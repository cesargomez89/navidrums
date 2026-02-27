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
- Status: `missing → queued → downloading → downloaded → processing → completed | failed`

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
Only fills empty fields - never overwrites existing locally saved data:
- `Artist`, `Artists`, `ArtistIDs`
- `Title`, `Duration`, `Year`
- `Barcode`, `CatalogNumber`, `ReleaseType`
- `AlbumArtistIDs`, `AlbumArtists`
- `Composer`, `Genre`
- `ReleaseID`

*Note: MusicBrainz API requests are throttled strictly to ~0.6 requests per second (1100ms intervals) to prevent IP blocking.*

### Precedence Rule

**Local Edits > Hi-Fi data > MusicBrainz data**

Both Hi-Fi and MusicBrainz use a "fill-in-the-blanks" pattern (`internal/app/enricher.go`). They will never overwrite fields that are already populated on the track record.
If a track is already fully populated with all the fields an API can natively provide, the API request is skipped entirely to optimize performance and prevent rate limiting.

```go
if track.Artist == "" && meta.Artist != "" {
    track.Artist = meta.Artist
}
```

MusicBrainz enrichment only triggers when `track.ISRC != ""` or `track.RecordingID != ""`.

### Sync Job Types

| Job Type | Hi-Fi API | MusicBrainz | Behavior |
|----------|-----------|-------------|----------|
| `JobTypeSyncFile` | No | No | Re-tags file with existing DB metadata only |
| `JobTypeSyncMusicBrainz` | No | Yes (fill gaps) | MusicBrainz API (if needed) → fill gaps → update DB → re-tag |
| `JobTypeSyncHiFi` | Yes (fill gaps) | Yes (fill gaps) | Hi-Fi API (if needed) → fill gaps → MusicBrainz API (if needed) → fill gaps → update DB → re-tag |

### Sync Scenarios

| Action | Job Type | Description |
|--------|----------|-------------|
| Per-track "Sync to File" button | `JobTypeSyncFile` | Re-tags with current DB metadata |
| Per-track "Enrich from MusicBrainz" button | `JobTypeSyncMusicBrainz` | Fetches MusicBrainz, fills remaining gaps, re-tags |
| Per-track "Enrich from Hi-Fi" button | `JobTypeSyncHiFi` | Fetches Hi-Fi data, fills gaps, then MusicBrainz fills remaining gaps, re-tags |
| "Sync All" | `JobTypeSyncHiFi` | Batch enrichment from Hi-Fi + MusicBrainz for all completed tracks (filling missing gaps) |

### Key Points

1. API requests are skipped entirely if a track is already fully populated with the fields that API could potentially provide.
2. Initial download: Hi-Fi data is fetched and applied, then MusicBrainz fills gaps.
3. Resyncing via "Enrich from MusicBrainz" only fetches MusicBrainz - never re-fetches Hi-Fi data.
4. "Enrich from Hi-Fi" and "Sync All" fetch Hi-Fi data (to fill gaps), then MusicBrainz fills remaining gaps.
5. Manual edits via the web UI form are always preserved, because neither sync job will overwrite a populated field.

### Genre Extraction & Normalization

MusicBrainz genre tags are processed to extract a main genre and optional sub-genre, which are combined into the `Genre` field as `"maingenre; subgenre"`:

1. **Fetch tags** from MusicBrainz (with vote counts)
2. **Aggregate counts** by raw tag name
3. **Sort tags** by count (descending)
4. **Determine Main Genre**: Find the first tag (highest count) that maps to a canonical genre via the genre map. If none map, use the raw tag with the highest overall count.
5. **Determine Sub-Genre**: Use the raw tag with the highest overall count.
6. **Suppress Sub-Genre**: If the sub-genre is functionally equivalent to the main genre (ignoring case, spaces, hyphens, and underscores), no sub-genre suffix is added.
7. **Combine**: If a distinct sub-genre exists, the final genre is written as `"maingenre; subgenre"` into the single `Genre` field.

**Example:**
- MusicBrainz returns: `["death metal": 5, "thrash metal": 3, "rock": 2]`
- Highest count tag: `death metal`
- `death metal` maps to: `metal`
- Main Genre: `metal`, Sub-Genre: `death metal`
- Stored as: `"metal; death metal"`

**Configuration:**
- Default genre map in `internal/musicbrainz/client.go` (`DefaultGenreMap`)
- Custom map stored in `settings` table (`genre_map` key)
- UI for customization in Settings page

If no tags match the genre map, the original tag with highest count is used.

