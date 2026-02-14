# Domain Model

## Track
Remote metadata entity describing a song.

Fields include title, artist, album, track number, disc number, duration, year, genre, label, ISRC, copyright, composer, album art URL, explicit lyrics, BPM, key, replay gain, peak, version, description, URL, audio quality, audio modes, lyrics, subtitles, and release date.

Not guaranteed to exist locally.

---

## Download
A local file produced from a Track.

Has filesystem path, provider ID, and completion timestamp. Stored in database to track already-downloaded tracks and prevent duplicates.

---

## Album
Collection of tracks grouped under a release.

Fields include ID, title, artist, year, release date, genre, label, copyright, total tracks, total discs, album art URL, track list, UPC, album type, URL, and explicit flag.

---

## Artist
Collection representing a music artist.

Fields include ID, name, picture URL, albums list, and top tracks list.

---

## Playlist
Collection of tracks curated by users or the system.

Fields include ID, title, description, image URL, and tracks list.

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
queued → resolving_tracks → downloading → completed | failed | cancelled
```

A container job (album/playlist/artist) resolves to track jobs which are enqueued separately. Track jobs handle the actual download, tagging, and file writing.

---

## Provider
External music catalog source adapter interface.

Methods:
- `Search` - Search for artists, albums, playlists, tracks
- `GetArtist` - Fetch artist with albums and top tracks
- `GetAlbum` - Fetch album with track list
- `GetPlaylist` - Fetch playlist with tracks
- `GetTrack` - Fetch single track metadata
- `GetStream` - Get audio stream for a track
- `GetSimilarAlbums` - Get recommendations
- `GetLyrics` - Fetch lyrics and subtitles for a track

Providers do not persist state.

---

## Worker
Executes jobs asynchronously with concurrency control.

Workers:
- Poll for queued jobs at regular intervals
- Process jobs with configurable max concurrency
- Handle job lifecycle: resolution → download → tagging → completion
- Decompose container jobs (album/playlist/artist) into track jobs
- Record downloads in database after successful completion
- Recover from panics gracefully

Workers never decide business rules. They only execute service instructions.

---

## SearchResult
Container for search results across all entity types.

Contains lists of artists, albums, playlists, and tracks.
