## Qobuz API

Base: `https://qobuz.kennyy.com.br/api`

### Endpoints

**Search** — `GET /get-music?q={query}&offset=0` — returns albums, tracks, artists, playlists, stories, most_popular (each: total + items).

**Album** — `GET /get-album?album_id={uuid}` — metadata + embedded tracks with audio_info, performers, isrc, maximum_bit_depth, hires.

**Artist** — `GET /get-artist?artist_id={int}` — metadata, albums (grouped by type), toptracks, similar_artists, playlists.

**Track** — `GET /get-track?isrc={int}` — **param is track ID**, not ISRC code. Returns track + embedded album.

**Download** — `GET /download-music?track_id={int}&quality={int}` — returns signed Akamai URL (time-limited). Quality: `6` = LOSSLESS.

### Data Types

**Album**: id (UUID), title, artist, artists[], label, upc, genre, image, release_date_original, tracks_count, tracks.items[], copyright, parental_warning.

**Track**: id, title, track_number, media_number (disc), duration, isrc, performers, composer, performer, audio_info, copyright, parental_warning, version.

**Audio**: maximum_bit_depth, maximum_sampling_rate, hires, hires_streamable, audio_info.replaygain_*.

### Design Notes

- Album IDs are UUIDs (strings), not integers
- Tracks embedded in album response
- `/get-track` param `isrc` is misnamed — takes track ID
- Download returns time-limited signed CDN URL
- Search returns all types at once
- Cover art: `static.qobuz.com` at `_50.jpg`, `_230.jpg`, `_600.jpg`

See `api-examples/qobuz-api/` for JSON examples.