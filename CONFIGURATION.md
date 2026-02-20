# Configuration

Navidrums is configured via environment variables with sensible defaults. All configuration is validated at startup.

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8080` | No | HTTP server port (1-65535) |
| `DB_PATH` | `navidrums.db` | No | SQLite database file path |
| `DOWNLOADS_DIR` | `~/Downloads/navidrums` | No | Output directory for downloaded music |
| `SUBDIR_TEMPLATE` | `{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}` | No | Go template for file organization |
| `PROVIDER_URL` | `http://127.0.0.1:8000` | No | URL of the Hifi API provider |
| `QUALITY` | `LOSSLESS` | No | Audio quality preference (`LOSSLESS`, `HI_RES_LOSSLESS`, `HIGH`, `LOW`) |
| `LOG_LEVEL` | `info` | No | Logging level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `text` | No | Log output format (`text`, `json`) |
| `NAVIDRUMS_USERNAME` | `navidrums` | No* | Username for HTTP basic authentication |
| `NAVIDRUMS_PASSWORD` | (empty) | No | Password for HTTP basic authentication (empty disables auth) |
| `CACHE_TTL` | `12h` | No | Provider response cache TTL (e.g., `1h`, `24h`, `7d`) |
| `MUSICBRAINZ_URL` | `https://musicbrainz.org/ws/2` | No | MusicBrainz API endpoint for metadata enrichment |

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

### Example Templates

**Default:**
```bash
{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}
```
Produces: `Pink Floyd/1973 - The Dark Side of the Moon/01-01 Speak to Me.flac`

**Flat structure:**
```bash
{{.AlbumArtist}} - {{.Album}}/{{.Track}} {{.Title}}
```
Produces: `Pink Floyd - The Dark Side of the Moon/01 Speak to Me.flac`

**Year-first:**
```bash
{{.OriginalYear}}/{{.AlbumArtist}}/{{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}
```
Produces: `1973/Pink Floyd/The Dark Side of the Moon/01-01 Speak to Me.flac`

**Note:** Invalid filesystem characters (`<>:"/\|?*`) are automatically sanitized from paths.

## Quality Settings

| Quality | Description | Typical Bitrate |
|---------|-------------|-----------------|
| `LOSSLESS` | Lossless FLAC format | 16-bit/44.1kHz |
| `HI_RES_LOSSLESS` | High-resolution lossless | 24-bit/96kHz+ |
| `HIGH` | High-quality lossy | 320kbps MP3 |
| `LOW` | Standard quality lossy | 128kbps MP3 |

## Logging Configuration

### Log Levels
- `debug`: Detailed debugging information
- `info`: General operational information (default)
- `warn`: Warning conditions
- `error`: Error conditions

### Log Formats
- `text`: Human-readable text format
- `json`: Structured JSON format for log aggregation

## Cache Configuration

The `CACHE_TTL` controls how long provider responses are cached:

```bash
# Cache for 1 hour
CACHE_TTL=1h

# Cache for 1 day
CACHE_TTL=24h

# Cache for 1 week
CACHE_TTL=168h  # or 7d
```

Cache is stored in SQLite and automatically invalidated when providers change.

## Authentication

Basic HTTP authentication is optional:
- Set `NAVIDRUMS_PASSWORD` to enable authentication
- Leave `NAVIDRUMS_PASSWORD` empty to disable authentication
- When password is set, `NAVIDRUMS_USERNAME` must also be set

## Validation

Configuration is validated at startup. Common validation errors:

- Invalid `PORT` (not a number or out of range)
- Invalid `PROVIDER_URL` (not a valid URL)
- Invalid `QUALITY` (not one of allowed values)
- Invalid `SUBDIR_TEMPLATE` (template parsing error)
- Invalid `CACHE_TTL` (not a valid duration)
- Missing `NAVIDRUMS_USERNAME` when password is set

## Docker Configuration

When running in Docker, mount volumes for persistence:

```bash
# Downloads directory
-v /host/path/to/music:/downloads

# Database file
-v /host/path/navidrums.db:/app/navidrums.db
```

The container uses these internal paths:
- Downloads: `/downloads`
- Database: `/app/navidrums.db`

## Example .env File

```bash
PORT=8080
DB_PATH=navidrums.db
DOWNLOADS_DIR=/downloads
PROVIDER_URL=https://your-hifi-api.com
QUALITY=LOSSLESS
LOG_LEVEL=info
LOG_FORMAT=text
NAVIDRUMS_USERNAME=navidrums
NAVIDRUMS_PASSWORD=secure-password
SUBDIR_TEMPLATE={{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}
CACHE_TTL=12h
MUSICBRAINZ_URL=https://musicbrainz.org/ws/2
```

See [.env.sample](../.env.sample) for a minimal example.