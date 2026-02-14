## Hifi API (External Service)

### Base URL
Default: `http://127.0.0.1:8000`  
Override via `PROVIDER_URL` environment variable.

For manual/real testing, you can use: `https://tidal-api.binimum.org`

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

#### Artist
- **GET `/artist/`**: 
  - `id`: Get artist metadata + cover.
  - `f`: Get artist content (albums, EPs/Singles).
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

### Design Notes
- Uses `COUNTRY_CODE` (default `US`) for all requests.
- Artist view uses a capped concurrency (6) for track aggregation when `skip_tracks=false`.

### JSON Examples
See the `api-examples/hifi-api/` directory for example responses from all endpoints.
