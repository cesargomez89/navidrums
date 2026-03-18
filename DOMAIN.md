# Domain Model

## CatalogTrack
Remote metadata entity describing a song from the provider/catalog.

Fields include:
- Identity: `ID`, `ProviderID`, `AlbumID`, `ArtistID`
- Basic: `Title`, `Artist`, `Album`, `AlbumArtist`, `TrackNumber`, `DiscNumber`, `Year`, `Duration`
- Artist info: `Artists`, `AlbumArtists`, `ArtistIDs`, `AlbumArtistIDs`
- Genre/Label: `Genre`, `Mood`, `Style`, `Tags`, `Label`, `Compilation`
- Release: `ReleaseDate`, `ReleaseType`, `Barcode`, `CatalogNumber`
- Audio: `BPM`, `Key`, `KeyScale`, `ReplayGain`, `Peak`, `AudioQuality`, `AudioModes`
- Lyrics: `Lyrics`, `Subtitles`
- Commercial: `ISRC`, `Copyright`, `Composer`, `Version`, `Description`, `URL`, `AlbumArtURL`
- Explicit: `ExplicitLyrics`
- Position: `TotalTracks`, `TotalDiscs`

Not guaranteed to exist locally. Used by providers for search results.

---

## Track
Local download domain entity representing a track stored in the database.

Metadata fields:
- Identity: `ID`, `ProviderID`, `AlbumID`, `ReleaseID`, `RecordingID`
- Basic: `Title`, `Artist`, `Album`, `AlbumArtist`, `TrackNumber`, `DiscNumber`, `Year`, `Duration`
- Artist info: `Artists`, `AlbumArtists`, `ArtistIDs`, `AlbumArtistIDs`
- Genre/Label: `Genre`, `Mood`, `Style`, `Tags`, `Label`, `Compilation`
- Release: `ReleaseDate`, `ReleaseType`, `Barcode`, `CatalogNumber`
- Audio: `BPM`, `Key`, `KeyScale`, `ReplayGain`, `Peak`, `AudioQuality`, `AudioModes`
- Lyrics: `Lyrics`, `Subtitles`
- Commercial: `ISRC`, `Copyright`, `Version`, `Description`, `URL`
- Explicit: `Explicit`
- Credit: `Credit`, `Composer`
- Position: `TotalTracks`, `TotalDiscs`

Processing:
- `Status` - missing | queued | downloading | downloaded | processing | completed | failed
- `ParentJobID` - Reference to the container job that created this track
- `Error` - Error message if download failed

File:
- `FilePath` - Local filesystem path after download
- `FileExtension` - Audio file extension (.flac, .mp3, .m4a)
- `FileHash` - SHA256 hash of downloaded file for verification
- `ETag` - HTTP ETag from download source

Timestamps:
- `CreatedAt`, `UpdatedAt`, `CompletedAt`, `LastVerifiedAt`

Stored in the `tracks` table. Prevents duplicate downloads via unique `provider_id` constraint.

---

## Album
Collection of tracks grouped under a release.

Fields include: `ID`, `Title`, `Artist`, `ArtistID`, `Artists`, `ArtistIDs`, `Year`, `ReleaseDate`, `Genre`, `Label`, `Copyright`, `TotalTracks`, `TotalDiscs`, `AlbumArtURL`, `Tracks` (CatalogTrack list), `UPC`, `AlbumType`, `URL`, `Explicit`.

---

## Artist
Collection representing a music artist.

Fields include ID, name, picture URL, albums list, and top tracks list (CatalogTrack).

---

## Playlist
Collection of tracks curated by users or the system.

Fields include: `ID` (internal), `ProviderID` (external), `Title`, `Description`, `ImageURL`, `Tracks` (CatalogTrack list for provider data), `CreatedAt`, `UpdatedAt`.

Persisted to database with many-to-many relationship to tracks via `playlist_tracks` junction table.

---

## Job
Represents a background task for downloading content.

Types:
- `track` - Single track download
- `album` - Album download (decomposes into track jobs)
- `playlist` - Playlist download (decomposes into track jobs)
- `artist` - Artist top tracks download (decomposes into track jobs)
- `discography` - Artist discography download (decomposes into album/track jobs)
- `sync_file` - Re-tag file with existing DB metadata
- `sync_musicbrainz` - Enrich from MusicBrainz, fill gaps, re-tag
- `sync_hifi` - Fetch fresh Hi-Fi data, then MusicBrainz enrichment, re-tag

Status machine:
```
queued → running → decomposed → completed | failed | cancelled
```

Note: Container jobs (album/playlist/artist) transition to `decomposed` after creating child track jobs. They remain in `decomposed` status until all children complete, then transition to `completed`.

Track status machine:
```
missing → queued → downloading → downloaded → processing → completed | failed
```

Structure:
- Minimal fields: ID, Type, Status, SourceID, Progress, Error, timestamps, ParentJobID
- `SourceID` links to Track.ProviderID
- `ParentJobID` links container jobs to their child track jobs (for cancellation/progress tracking)
- `m3u_generating` flag prevents race conditions in playlist M3U generation
- No metadata stored (get from Tracks table)

A container job (album/playlist/artist) creates Track records and child track jobs. Track jobs look up stored metadata and handle the actual download, tagging, and file writing.

---

## Provider
External music catalog source adapter interface.

Methods:
- `Search` - Search for artists, albums, playlists, tracks
- `GetArtist` - Fetch artist with albums and top tracks
- `GetAlbum` - Fetch album with track list
- `GetPlaylist` - Fetch playlist with tracks
- `GetTrack` - Fetch single track metadata (returns CatalogTrack)
- `GetStream` - Get audio stream for a track
- `GetSimilarAlbums` - Get recommendations
- `GetLyrics` - Fetch lyrics and subtitles for a track

Providers do not persist state. They return CatalogTrack types which are converted to Track when stored.

---

## Repository Store
Database persistence methods (internal/store):

Track operations:
- `CreateTrack` - Create new track record
- `GetTrackByID` / `GetTrackByProviderID` - Lookup tracks
- `UpdateTrack` - Update full track metadata
- `UpdateTrackPartial` - Update specific track fields
- `UpdateTrackStatus` - Update track status and file path
- `MarkTrackCompleted` - Mark track completed with file path and hash
- `MarkTrackFailed` - Mark track failed with error message
- `ListTracks`, `ListTracksByStatus`, `ListTracksByParentJobID` - List queries
- `ListCompletedTracks`, `ListCompletedTracksWithISRC` - Completed track queries
- `IsTrackDownloaded`, `GetDownloadedTrack` - Download verification
- `SearchTracks` - Search tracks by title/artist/album
- `DeleteTrack` - Delete track record
- `FindInterruptedTracks` - Find tracks stuck in downloading/processing
- `RecomputeAlbumState` - Recompute album download state (missing/partial/completed)

Job operations:
- `CreateJob`, `CreateJobBatch` - Job lifecycle (batch for atomic decomposition)
- `GetJob`, `UpdateJobStatus`, `UpdateJobProgress`, `MarkJobFailed` - Job lifecycle
- `CountJobsForParent` - Count total/pending child jobs for progress tracking
- `CancelJobsByParentID` - Cancel all child jobs when parent is cancelled
- `TrySetM3UGenerating`, `ClearM3UGenerating` - Advisory lock for playlist generation

Playlist operations:
- `CreatePlaylist`, `GetPlaylistByID`, `GetPlaylistByProviderID`, `UpdatePlaylist`, `ListPlaylists`, `DeletePlaylist` - Playlist CRUD
- `AddTrackToPlaylist`, `RemoveTrackFromPlaylist`, `GetTracksByPlaylistID`, `GetPlaylistsByTrackID` - Junction table operations
- `PlaylistExists`, `ClearPlaylistTracks` - Utility operations

---

## Worker
Executes jobs asynchronously with concurrency control.

Workers:
- Poll for queued jobs at regular intervals
- Process jobs with configurable max concurrency
- Handle job lifecycle: running → download → tagging → completion
- Decompose container jobs (album/playlist/artist) into track records + child jobs atomically (using batch operations with transactions)
- Track container job progress by aggregating child job completion counts
- Mark container job as `decomposed` after creating children, then `completed` when all children finish
- Use advisory lock (`m3u_generating` flag) to prevent race conditions in playlist M3U generation
- Look up Track metadata for downloads (no duplicate provider calls)
- Update Track status throughout lifecycle (missing → queued → downloading → downloaded → processing → completed)
- Update parent container job progress when child track jobs complete
- Recover interrupted tracks on startup (reset downloading/processing to queued)
- Verify file hash for idempotent downloads (skip if file exists and hash matches)
- Recompute album state after track completion
- Recover from panics gracefully

Workers never decide business rules. They only execute service instructions.

---

## Playlist Persistence

Playlists are persisted to the database with the following structure:

**playlists table:**
- `id` - Auto-increment primary key (internal)
- `provider_id` - Unique identifier from the provider (prevents duplicates on re-download)
- `title` - Playlist title
- `description` - Optional description
- `image_url` - Playlist cover image URL
- `created_at`, `updated_at` - Timestamps

**playlist_tracks junction table:**
- `id` - Auto-increment primary key
- `playlist_id` - Foreign key to playlists (CASCADE delete)
- `track_id` - Foreign key to tracks (CASCADE delete)
- `position` - Track order within playlist
- `added_at` - Timestamp when track was added

This enables:
- Many-to-many relationship (tracks can belong to multiple playlists)
- Playlist metadata persistence across sessions
- M3U generation from database instead of job hierarchy
- Upsert behavior: re-downloading a playlist updates existing record

---

## Playlist Generation

M3U playlist files are generated after all tracks in a playlist download complete. Generation uses database-backed playlist-tracks data (via `GenerateFromDB` method) with fallback to provider API.

Protected by an advisory lock (`m3u_generating` flag on the job record) to prevent race conditions when multiple track jobs complete simultaneously.

Location: `<DownloadsDir>/playlists/<sanitized_title>_<provider_id>.m3u`

Format: Extended M3U with `#EXTM3U`, `#PLAYLIST:`, and `#EXTINF:` tags containing track duration and metadata.

---

## CachedProvider

A caching wrapper around the HiFi provider that stores responses in SQLite with configurable TTL (default: 12h, controlled by `CACHE_TTL`). Prevents duplicate API calls for the same content.

Cached methods:
- `Search`, `GetArtist`, `GetAlbum`, `GetPlaylist`, `GetTrack`
- `GetSimilarAlbums`, `GetSimilarArtists`, `GetRecommendations`

Not cached (streaming/dynamic data):
- `GetStream`, `GetLyrics`

---

## MusicBrainzCache

A caching wrapper around the MusicBrainz client that stores API responses in SQLite with configurable TTL (default: 7 days, controlled by `MUSICBRAINZ_CACHE_TTL`). MusicBrainz has strict rate limits, so extended caching is recommended.

Cached methods:
- `GetRecording` (by MBID or ISRC)
- `GetGenres`

Cache is shared with the provider cache in the same SQLite `cache` table.

---

## In-Memory Caching

### Recommendations Cache

The HTTP handler caches recommendations data in memory for 5 minutes to reduce provider calls. This is not configurable and is scoped per handler instance.

## SearchResult
Container for search results across all entity types.

Contains lists of artists, albums, playlists, and catalog tracks.
