# Configuration

Navidrums is configured via environment variables with sensible defaults. All configuration is validated at startup.

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8080` | No | HTTP server port (1-65535) |
| `DB_PATH` | `navidrums.db` | No | SQLite database file path (Docker: `/data/navidrums.db`) |
| `DOWNLOADS_DIR` | `~/Downloads/navidrums` | No | Output directory for downloaded music (Docker: `/music`) |
| `SUBDIR_TEMPLATE` | `{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}` | No | Go template for file organization |
| `PROVIDER_URL` | `http://127.0.0.1:8000` | No | Default HiFi (Tidal) API URL for metadata browsing (additional providers managed via Settings UI) |
| `QUALITY` | `LOSSLESS` | No | Audio quality preference (`LOSSLESS`, `HI_RES_LOSSLESS`, `HIGH`, `LOW`) |
| `LOG_LEVEL` | `info` | No | Logging level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `text` | No | Log output format (`text`, `json`) |
| `NAVIDRUMS_USERNAME` | `navidrums` | No* | Username for HTTP basic authentication |
| `NAVIDRUMS_PASSWORD` | (empty) | No | Password for HTTP basic authentication (empty disables auth) |
| `CACHE_TTL` | `12h` | No | Provider response cache TTL (e.g., `1h`, `24h`, `7d`) |
| `MUSICBRAINZ_CACHE_TTL` | `7d` | No | MusicBrainz API response cache TTL (e.g., `1d`, `168h`) |
| `MUSICBRAINZ_URL` | `https://musicbrainz.org/ws/2` | No | MusicBrainz API endpoint for metadata enrichment |
| `RATE_LIMIT_REQUESTS` | `200` | No | Maximum requests per rate limit window |
| `RATE_LIMIT_WINDOW` | `1m` | No | Rate limit time window (e.g., `30s`, `1m`) |
| `RATE_LIMIT_BURST` | `10` | No | Burst requests allowed beyond rate limit |
| `SKIP_AUTH` | `false` | No | Set to `true` to disable authentication entirely |
| `THEME` | `golden` | No | Default application theme (can be overridden in Settings) |
| `FFMPEG_PATH` | (system) | No | Path to ffmpeg binary (required for MP4/M4A tagging - hi-res downloads often come as MP4) |
| `FFPROBE_PATH` | (system) | No | Path to ffprobe binary |

**Rate limiting**: Each provider enforces a 200ms minimum interval between requests. The global rate limit (`RATE_LIMIT_*`) applies across all providers.

**Note:** ffmpeg is only required when tagging MP4/M4A files (common for hi-res audio). FLAC and MP3 files are tagged using native Go libraries.

\* `NAVIDRUMS_USERNAME` is required only when `NAVIDRUMS_PASSWORD` is set.

## Template Variables

The `SUBDIR_TEMPLATE` uses Go's `text/template` syntax with these available variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.AlbumArtist}}` | Album artist (falls back to track artist if empty) | `Pink Floyd` |
| `{{.OriginalYear}}` | Release year (integer) | `1973` |
| `{{.Album}}` | Album name | `The Dark Side of the Moon` |
| `{{.Disc}}` | Disc number, zero-padded (01, 02, etc.) | `01` |
| `{{.Track}}` | Track number, zero-padded (01, 02, etc.) | `01` |
| `{{.Title}}` | Track title | `Speak to Me` |

The file extension (`.flac`, `.mp3`, or `.mp4`) is appended automatically.

### Example

`{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}` → `Pink Floyd/1973 - The Dark Side/01-01 Speak to Me.flac`

**Note:** Invalid filesystem characters (`<>:"/\|?*`) are automatically sanitized from paths.

> Cache TTL: `CACHE_TTL=12h`, `MUSICBRAINZ_CACHE_TTL=7d`. SQLite storage, auto-invalidated on provider change.

## Genre Map

Normalizes MusicBrainz subgenre tags → main genres. Configure in Settings UI.

- Default: Rock, Metal, Pop, Hip-Hop, R&B, Electronic, Latin, Regional Mexican, Country, Jazz, Classical, Folk, Reggae, Blues, Soundtrack
- Custom: JSON `{"dark ambient": "Electronic", ...}` — "Reset to Default" clears

## Authentication

Basic HTTP authentication is optional:
- Set `NAVIDRUMS_PASSWORD` to enable authentication
- Leave `NAVIDRUMS_PASSWORD` empty to disable authentication
- When password is set, `NAVIDRUMS_USERNAME` must also be set

## Provider Management

Navidrums supports two provider types: **HiFi** (Tidal API proxy) and **Qobuz** (Qobuz API proxy). Each type can have multiple endpoint URLs configured as fallbacks.

**Per-operation selection**: Three independent settings control which provider type is used for each operation:
- **Metadata (search/browse)**: defaults to HiFi
- **Download**: defaults to HiFi (switch to Qobuz for reliable full-track downloads)
- **Streaming**: defaults to HiFi (switch to Qobuz for full-length playback previews)

**Why separate providers**: The HiFi/Tidal API is reliable for metadata browsing but frequently returns 30-second previews instead of full tracks for downloads and streaming. Qobuz provides reliable full-track downloads and streaming.

Managing providers:
- **Primary provider**: Sets the default HiFi URL via `PROVIDER_URL` environment variable
- **Settings UI**: Add, reorder (drag), edit, delete provider URLs per type; select which provider type per operation
- **Fallback within type**: Multiple URLs of the same type are tried in position order until one succeeds

## Validation

Startup validation — common errors: invalid PORT, PROVIDER_URL, QUALITY, SUBDIR_TEMPLATE, CACHE_TTL, or missing username with password set.

## Docker

Mount: `-v /host/music:/music -v /host/data:/data`. Internal paths: `/music` (downloads), `/data/navidrums.db` (db).

See [.env.sample](../.env.sample) for minimal example.