# Domain Model

## CatalogTrack
Remote metadata entity describing a song from the provider/catalog.

Fields include title, artist, album, track number, disc number, duration, year, genre, label, ISRC, copyright, composer, album art URL, explicit lyrics, BPM, key, replay gain, peak, version, description, URL, audio quality, audio modes, lyrics, subtitles, and release date.

Not guaranteed to exist locally. Used by providers for search results.

---

## Track
Local download domain entity representing a track stored in the database.

Contains full metadata (same fields as CatalogTrack) plus:
- `ProviderID` - Links to the provider's track ID
- `Status` - pending | downloading | completed | failed
- `ParentJobID` - Reference to the container job that created this track
- `FilePath` - Local filesystem path after download
- `FileExtension` - Audio file extension (.flac, .mp3, .m4a)
- `Error` - Error message if download failed
- Timestamps for creation, update, and completion

Stored in the `tracks` table. Prevents duplicate downloads via unique `provider_id` constraint.

---

## Album
Collection of tracks grouped under a release.

Fields include ID, title, artist, year, release date, genre, label, copyright, total tracks, total discs, album art URL, track list (CatalogTrack), UPC, album type, URL, and explicit flag.

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

Status machine:
```
queued → running → completed | failed | cancelled
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

## Worker
Executes jobs asynchronously with concurrency control.

Workers:
- Poll for queued jobs at regular intervals
- Process jobs with configurable max concurrency
- Handle job lifecycle: running → download → tagging → completion
- Decompose container jobs (album/playlist/artist) into track records + child jobs
- Look up Track metadata for downloads (no duplicate provider calls)
- Update Track status throughout lifecycle
- Recover from panics gracefully

Workers never decide business rules. They only execute service instructions.

---

## SearchResult
Container for search results across all entity types.

Contains lists of artists, albums, playlists, and catalog tracks.
