## Hifi API (External Service)

**Status**: The HiFi/Tidal API proxy is reliable for metadata browsing (search, artist, album, track info) but **unreliable for downloads and streaming** — it frequently returns 30-second previews instead of full-length tracks, especially at HI_RES_LOSSLESS quality. For reliable downloads and streaming, use a Qobuz provider instead (see [QOBUZ_API.md](QOBUZ_API.md)). Navidrums supports separate provider selection per operation — configure in Settings.

### Base URL
Default: `http://127.0.0.1:8000` (Replace with a real hifi-api url. Some urls may return HI_RES_LOSSLESS 30 seconds track previews instead of full-length streams.)
Override via `PROVIDER_URL` environment variable.

### Endpoints

#### Track & Playback
- **GET `/info/`**: Get track metadata.
  - `id`: int (required)
- **GET `/track/`**: Get playback info/manifest.
  - `id`: int (required)
  - `quality`: str (optional: `HI_RES_LOSSLESS`, `LOSSLESS`, `HIGH`, `LOW`)
- **GET `/recommendations/`**: Get similar tracks.
  - `id`: int (required)
- **GET `/lyrics/`**: Get track lyrics.
  - `id`: int (required)

#### Search
- **GET `/search/`**: Search across types.
  - `s`: track query
  - `a`: artist query
  - `al`: album query
  - `v`: video query
  - `p`: playlist query
  - `limit`: int (default 25)
  - `offset`: int (default 0)

#### Artist
- **GET `/artist/`**: 
  - `id`: int. Get artist metadata + cover.
  - `f`: int. Get artist content (albums, EPs/Singles).
  - `skip_tracks`: bool (default `false`). If `true` with `f`, returns `toptracks` (15) instead of aggregating all tracks from all albums.
- **GET `/artist/similar/`**: Get similar artists.
  - `id`: int (required)
  - `cursor`: string/int (optional)

#### Album & Playlist
- **GET `/album/`**: Get album metadata + tracks.
  - `id`: int (required)
  - `limit`: int (default 100)
  - `offset`: int (default 0)
- **GET `/album/similar/`**: Get similar albums.
  - `id`: int (required)
  - `cursor`: string/int (optional)
- **GET `/playlist/`**: Get playlist metadata + tracks.
  - `id`: string (required)
  - `limit`: int (default 100)
  - `offset`: int (default 0)
- **GET `/mix/`**: Get mix items.
  - `id`: string (required)

#### Images
- **GET `/cover/`**: Get album cover URLs.
  - `id`: int (track/album id) OR `q`: search query

#### Videos
- **GET `/topvideos/`**: Get recommended videos.
  - `limit`: int (default 25)
  - `offset`: int (default 0)
- **GET `/video/`**: Get video playback info.
  - `id`: int (required)
  - `quality`: string (`HIGH`, `MEDIUM`, `LOW`)

### Design Notes
- Uses `COUNTRY_CODE` (default `US`) for all requests.
- Artist view uses a capped concurrency (6) for track aggregation when `skip_tracks=false`.
- API requests are throttled to ~1.2 requests per second (1100ms intervals) to prevent rate limiting.

See `api-examples/hifi-api/` for example responses.
