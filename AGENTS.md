# API Specifications

This application provides a web interface for browsing and downloading music, consuming an external Hifi API for music data.

---

# Application API (Your Server)

## Base URL
Default: `http://localhost:8080`  
Override via `PORT` environment variable.

## Web Interface Routes

### Navigation & Search
- **GET `/`** - Main search page
- **GET `/htmx/search`** - HTMX search endpoint
  - `q`: search query string
  - `type`: search type (`album`, `artist`, `playlist`, `track`) - default: `album`
  - Returns: HTML fragment with search results

### Entity Pages
- **GET `/artist/{id}`** - Artist detail page with albums and tracks
- **GET `/album/{id}`** - Album detail page with track listing
- **GET `/playlist/{id}`** - Playlist detail page with tracks

### Download Management
- **POST `/htmx/download/{type}/{id}`** - Queue a download job
  - `type`: resource type (`album`, `artist`, `playlist`, `track`)
  - `id`: resource identifier
- **GET `/queue`** - Active downloads queue page
- **GET `/htmx/queue`** - HTMX queue status (auto-refresh)
- **POST `/htmx/cancel/{id}`** - Cancel a queued job
  - `id`: job ID to cancel
- **GET `/history`** - Completed/failed downloads history (last 20 items)

### Static Assets
- **GET `/static/*`** - Static files (CSS, JS, images)

## Configuration

### Environment Variables
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DB_PATH` | `navidrums.db` | SQLite database file path |
| `DOWNLOADS_DIR` | `~/Downloads/navidrums` | Download destination directory |
| `PROVIDER_URL` | `http://127.0.0.1:8000` | Hifi API base URL |
| `QUALITY` | `LOSSLESS` | Default download quality |
| `USE_MOCK` | `false` | Use mock provider for testing |

## Architecture Notes
- Uses **HTMX** for dynamic content updates without full page reloads
- Download jobs are processed asynchronously by background workers
- SQLite database tracks job states (pending, processing, completed, failed, cancelled)
- Artist downloads aggregate tracks from all albums with capped concurrency (6)

---

# Hifi API (External Service)

## Base URL
Default: `http://127.0.0.1:8000`  
Override via `PROVIDER_URL` environment variable.

## Endpoints

### Track & Playback
- **GET `/info/`**: Get track metadata.
  - `id`: int (required)
- **GET `/track/`**: Get playback info/manifest.
  - `id`: int (required)
  - `quality`: str (optional: `HI_RES_LOSSLESS`, `LOSSLESS`, `HIGH`, `LOW`)
- **GET `/recommendations/`**: Get similar tracks.
  - `id`: int (required)
- **GET `/lyrics/`**: Get track lyrics.
  - `id`: int (required)

### Search
- **GET `/search/`**: Search across types.
  - `s`: track query
  - `a`: artist query
  - `al`: album query
  - `v`: video query
  - `p`: playlist query

### Artist
- **GET `/artist/`**: 
  - `id`: Get artist metadata + cover.
  - `f`: Get artist content (albums, EPs/Singles).
  - `skip_tracks`: bool (default `false`). If `true` with `f`, returns `toptracks` (15) instead of aggregating all tracks from all albums.
- **GET `/artist/similar/`**: Get similar artists.
  - `id`: int (required)
  - `cursor`: string/int (optional)

### Album & Playlist
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

### Images
- **GET `/cover/`**: Get album cover URLs.
  - `id`: int (track/album id) OR `q`: search query

## Authentication
Requires `token.json` populated with Tidal credentials. Uses `auth.tidal.com` for token refreshes.
- Internal env vars: `CLIENT_ID`, `CLIENT_SECRET`, `REFRESH_TOKEN`, `USER_ID`.

## Design Notes
- Uses `COUNTRY_CODE` (default `US`) for all requests.
- Artist view uses a capped concurrency (6) for track aggregation when `skip_tracks=false`.
