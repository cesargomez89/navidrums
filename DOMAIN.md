# Domain Model

## CatalogTrack
Remote metadata entity describing a song from the provider/catalog.

Fields include:
- Identity: `ID`, `ProviderID`, `AlbumID`, `ArtistID`
- Basic: `Title`, `Artist`, `Album`, `AlbumArtist`, `TrackNumber`, `DiscNumber`, `Year`, `Duration`
- Artist info: `Artists`, `AlbumArtists`, `ArtistIDs`, `AlbumArtistIDs`
- Genre/Label: `Genre`, `SubGenre`, `Label`, `Compilation`
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
- Identity: `ID`, `ProviderID`, `AlbumID`, `ReleaseID`
- Basic: `Title`, `Artist`, `Album`, `AlbumArtist`, `TrackNumber`, `DiscNumber`, `Year`, `Duration`
- Artist info: `Artists`, `AlbumArtists`, `ArtistIDs`, `AlbumArtistIDs`
- Genre/Label: `Genre`, `SubGenre`, `Label`, `Compilation`
- Release: `ReleaseDate`, `ReleaseType`, `Barcode`, `CatalogNumber`
- Audio: `BPM`, `Key`, `KeyScale`, `ReplayGain`, `Peak`, `AudioQuality`, `AudioModes`
- Lyrics: `Lyrics`, `Subtitles`
- Commercial: `ISRC`, `Copyright`, `Version`, `Description`, `URL`
- Explicit: `Explicit`
- Credit: `Credit`, `Composer`
- Position: `TotalTracks`, `TotalDiscs`

Processing:
- `Status` - missing | queued | downloading | processing | completed | failed
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

Fields include ID, title, description, image URL, and tracks list (CatalogTrack).

---

## Job
Represents a background task for downloading content.

Types:
- `track` - Single track download
- `album` - Album download (decomposes into track jobs)
- `playlist` - Playlist download (decomposes into track jobs)
- `artist` - Artist top tracks download (decomposes into track jobs)
- `sync_file` - Re-tag file with existing DB metadata
- `sync_musicbrainz` - Enrich from MusicBrainz, fill gaps, re-tag
- `sync_hifi` - Fetch fresh Hi-Fi data, then MusicBrainz enrichment, re-tag

Status machine:
```
queued → running → completed | failed | cancelled
```

Track status machine:
```
missing → queued → downloading → processing → completed | failed
```

Structure:
- Minimal fields: ID, Type, Status, SourceID, Progress, Error, timestamps
- `SourceID` links to Track.ProviderID
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
- `CreateJob`, `GetJob`, `UpdateJobStatus`, `MarkJobFailed` - Job lifecycle

---

## Worker
Executes jobs asynchronously with concurrency control.

Workers:
- Poll for queued jobs at regular intervals
- Process jobs with configurable max concurrency
- Handle job lifecycle: running → download → tagging → completion
- Decompose container jobs (album/playlist/artist) into track records + child jobs
- Look up Track metadata for downloads (no duplicate provider calls)
- Update Track status throughout lifecycle (missing → queued → downloading → processing → completed)
- Recover interrupted tracks on startup (reset downloading/processing to queued)
- Verify file hash for idempotent downloads (skip if file exists and hash matches)
- Recompute album state after track completion
- Recover from panics gracefully

Workers never decide business rules. They only execute service instructions.

---

## SearchResult
Container for search results across all entity types.

Contains lists of artists, albums, playlists, and catalog tracks.
