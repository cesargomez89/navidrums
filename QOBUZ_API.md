## Qobuz API (External Service)

### Base URL

`https://qobuz.kennyy.com.br/api`

### Endpoints

---

#### Search

- **GET `/get-music`**: Search across all content types simultaneously.
  - `q`: string (required) — search query
  - `offset`: int (default 0) — pagination offset

Returns a single object with sections for `albums`, `tracks`, `artists`, `playlists`, `stories`, and `most_popular`. Each section includes `total` and `items`.

---

#### Album

- **GET `/get-album`**: Get album metadata with full embedded track list.
  - `album_id`: string (required) — album UUID (e.g. `sef9i244rd865`)

Returns complete album info plus a `tracks` object with `total`, `offset`, `limit`, and `items[]`. Each track item includes `audio_info` (ReplayGain), `performers`, `isrc`, `composer`, `maximum_bit_depth`, `maximum_sampling_rate`, `hires`, and availability flags.

---

#### Artist

- **GET `/get-artist`**: Get artist metadata, discography, similar artists, and playlists.
  - `artist_id`: int (required) — artist numeric ID (e.g. `3467343`)

Returns a rich payload with `albums` (grouped by type: album, EP/single), `toptracks`, `similar_artists`, `playlists`, images, and metadata profiles.

---

#### Track

- **GET `/get-track`**: Get track metadata with embedded album info.
  - `isrc`: int (required) — **track ID**, not an ISRC code. The parameter is misnamed.

Returns track detail plus an embedded `album` object containing the parent album's metadata. The actual ISRC code is available in the `isrc` field of the response (e.g. `"UYB282636622"`).

---

#### Download / Stream

- **GET `/download-music`**: Get a signed streaming/download URL.
  - `track_id`: int (required) — track ID
  - `quality`: int (required) — quality code

Returns a JSON object with a single `url` field containing a signed URL (Akamai CDN). The URL is time-limited (see `etsp` query parameter for expiration timestamp).

**Quality Codes**: `6` = LOSSLESS (observed). Other values are not yet confirmed.

---

### Key Data Types

#### Audio Info (per track)

| Field | Type | Description |
|-------|------|-------------|
| `maximum_bit_depth` | int | Bit depth (e.g. 24) |
| `maximum_sampling_rate` | float | Sample rate in kHz (e.g. 44.1, 48) |
| `maximum_channel_count` | int | Channel count (e.g. 2 for stereo) |
| `hires` | bool | Whether hi-res quality is available |
| `hires_streamable` | bool | Whether hi-res is streamable |
| `audio_info.replaygain_track_gain` | float | ReplayGain track gain (dB) |
| `audio_info.replaygain_track_peak` | float | ReplayGain track peak |

#### Album

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Album UUID (e.g. `sef9i244rd865`) |
| `title` | string | Album title |
| `artist` | object | Main artist (id, name, slug) |
| `artists[]` | array | Contributing artists with roles |
| `label` | object | Label (id, name) |
| `upc` | string | UPC/EAN barcode |
| `qobuz_id` | int | Qobuz numeric album ID |
| `genre` | object | Primary genre (id, name, path) |
| `image` | object | Cover art URLs (small, large, thumbnail) |
| `release_date_original` | string | Original release date (YYYY-MM-DD) |
| `released_at` | int | Unix timestamp |
| `tracks_count` | int | Total track count |
| `tracks.items[]` | array | Embedded track list (with audio info) |
| `copyright` | string | Copyright string |
| `product_type` | string | Type (e.g. "album") |
| `parental_warning` | bool | Explicit content flag |
| `maximum_technical_specifications` | string | Human-readable spec (e.g. "24 bits / 44.1 kHz - Stereo") |

#### Track (within album or /get-track)

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Track numeric ID |
| `title` | string | Track title |
| `track_number` | int | Track number on album |
| `media_number` | int | Disc number |
| `duration` | int | Duration in seconds |
| `isrc` | string | ISRC code (e.g. `UYB282636622`) |
| `performers` | string | Credits string (artists, composers, producers) |
| `composer` | object | Composer (id, name) |
| `performer` | object | Main performer (id, name) |
| `audio_info` | object | ReplayGain data |
| `copyright` | string | Track copyright |
| `parental_warning` | bool | Explicit content flag |
| `version` | string/null | Track version/variant |
| `work` | object/null | Musical work reference |

#### Artist

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Artist numeric ID |
| `name` | string | Artist name |
| `slug` | string | URL slug |
| `albums_count` | int | Total album count |
| `image` / `picture` | object/null | Artist images |
| `albums` | object | Grouped by type: "album", "ep", "single", "compilation" |
| `toptracks` | array | Top tracks |
| `similar_artists` | array | Similar artists |
| `playlists` | array | Playlists featuring this artist |

#### Search Result

| Field | Type | Description |
|-------|------|-------------|
| `albums` | { total, items[] } | Matching albums |
| `tracks` | { total, items[] } | Matching tracks |
| `artists` | { total, items[] } | Matching artists |
| `playlists` | { total, items[] } | Matching playlists |
| `stories` | { total, items[] } | Matching stories |
| `most_popular` | { total, items[] } | Popular results for the query |
| `switchTo` | string/null | Suggested search refinement |

---

### Design Notes

- **Album IDs are strings** (UUIDs like `sef9i244rd865`), not integers. This differs from the HiFi API.
- **Tracks are embedded** in album responses — no separate track listing endpoint is needed. Use `/get-album` to get all tracks for an album.
- **`/get-track` parameter is misnamed**: the `isrc` query parameter actually takes a track ID (int). The real ISRC is in the response body.
- **Download endpoint returns a raw signed URL** — no manifest parsing required. The URL points directly to an Akamai CDN stream. URLs are time-limited (check `etsp` in the URL).
- **Quality is numeric** (`quality=6` for LOSSLESS), not a string label. Other quality levels are not yet confirmed.
- **Search returns all types at once** — `/get-music` includes albums, tracks, artists, and playlists in a single response.
- **Artist endpoint is comprehensive** — returns bio, albums grouped by type, top tracks, similar artists, and playlists in one call.
- **Availability flags**: `streamable`, `downloadable`, `purchasable`, `previewable`, `sampleable`, `displayable` — check these before attempting to stream or download.
- **Cover art**: Images use the `static.qobuz.com` CDN in three sizes: `_50.jpg` (thumbnail), `_230.jpg` (small), `_600.jpg` (large).

### JSON Examples

See the `api-examples/qobuz-api/` directory for example responses from all endpoints.

| File | Endpoint | Description |
|------|----------|-------------|
| `search.json` | `GET /get-music?q=love%20love%20flak&offset=0` | Multi-type search |
| `album.json` | `GET /get-album?album_id=sef9i244rd865` | Album with embedded tracks |
| `artist.json` | `GET /get-artist?artist_id=3467343` | Artist discography |
| `track.json` | `GET /get-track?isrc=416711618` | Track with album |
| `download.json` | `GET /download-music?track_id=416711618&quality=6` | Streaming URL |
