# Domain Model

> See @AGENTS.md for job lifecycle and data invariants. See @ARCHITECTURE.md for worker behavior.

## Entities

### CatalogTrack
Remote metadata from provider (search results). Fields: identity, basic, artist info, genre/label, release, audio, lyrics, commercial, position. Not guaranteed locally.

### Track
Local download entity. All CatalogTrack fields plus:
- **Processing**: Status (`missing → queued → downloading → downloaded → processing → completed | failed`), ParentJobID, Error
- **File**: FilePath, FileExtension, FileHash, ETag
- **Timestamps**: CreatedAt, UpdatedAt, CompletedAt, LastVerifiedAt

Stored in `tracks` table. `provider_id` UNIQUE prevents duplicates.

### Album, Artist, Playlist
Collections of tracks. Album: ID, title, artist, year, genre, label, tracks, UPC, artwork. Artist: ID, name, picture, albums, top tracks. Playlist: ID, provider_id, title, description, image, tracks, timestamps.

## Job

Background download task. Types: `track`, `album`, `playlist`, `artist`, `discography`, `sync_file`, `sync_musicbrainz`, `sync_hifi`.

Minimal fields: ID, Type, Status, SourceID, Progress, Error, timestamps, ParentJobID. `SourceID` → Track.ProviderID. Container jobs decompose into track jobs.

## Provider

External catalog adapter interface with two implementations: **HiFi** (Tidal API proxy) and **Qobuz** (Qobuz API proxy). Methods: `Search`, `GetArtist`, `GetAlbum`, `GetPlaylist`, `GetTrack`, `GetStream`, `GetSimilarAlbums`, `GetLyrics`. Returns CatalogTrack.

**ProviderManager** orchestrates three independent provider chains:
- Metadata provider (search/browse)
- Download provider (track downloads via `GetStream`)
- Streaming provider (playback previews via `GetStream`)

Each chain is `FallbackProvider → CachedProvider` — tries multiple URLs of the same type, caches responses. Provider type per operation is stored in settings (`active_metadata_provider`, `active_download_provider`, `active_streaming_provider`).

Qobuz limitations: `GetPlaylist`, `GetSimilarAlbums`, `GetSimilarArtists`, `GetLyrics`, `GetRecommendations` return `ErrQobuzNotSupported`.

## Repository (internal/store)

**Tracks**: CreateTrack, GetTrackByID/ProviderID, UpdateTrack, UpdateTrackPartial, UpdateTrackStatus, MarkTrackCompleted/Failed, ListTracks, ListCompletedTracks, IsTrackDownloaded, SearchTracks, DeleteTrack, FindInterruptedTracks, RecomputeAlbumState.

**Jobs**: CreateJob, CreateJobBatch, GetJob, UpdateJobStatus/Progress, MarkJobFailed, CountJobsForParent, CancelJobsByParentID, TrySetM3UGenerating, ClearM3UGenerating.

**Playlists**: CRUD, AddTrackToPlaylist, RemoveTrackFromPlaylist, GetTracksByPlaylistID, GetPlaylistsByTrackID, PlaylistExists, ClearPlaylistTracks.

## Playlist Persistence

`playlists` table: id, provider_id, title, description, image_url, timestamps. `playlist_tracks` junction: playlist_id, track_id, position, added_at (CASCADE delete). Many-to-many, upsert on re-download.

## M3U Generation

After playlist tracks complete. Uses database (`GenerateFromDB`) with provider fallback. Protected by `m3u_generating` advisory lock.

Location: `<DownloadsDir>/playlists/<title>_<provider_id>.m3u` — extended M3U format.

## Caching

**CachedProvider**: SQLite cache (CACHE_TTL=12h default). Cached: Search, GetArtist, GetAlbum, GetPlaylist, GetTrack, GetSimilar*. Not cached: GetStream, GetLyrics.

**MusicBrainzCache**: SQLite cache (MUSICBRAINZ_CACHE_TTL=7d default). Cached: GetRecording, GetGenres. Shared `cache` table.

**In-Memory**: Recommendations cached 5min per handler.

## SearchResult

Container for artists, albums, playlists, tracks from search.