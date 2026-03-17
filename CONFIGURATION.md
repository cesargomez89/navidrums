# Configuration

Navidrums is configured via environment variables with sensible defaults. All configuration is validated at startup.

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8080` | No | HTTP server port (1-65535) |
| `DB_PATH` | `navidrums.db` | No | SQLite database file path (Docker: `/data/navidrums.db`) |
| `DOWNLOADS_DIR` | `~/Downloads/navidrums` | No | Output directory for downloaded music (Docker: `/music`) |
| `SUBDIR_TEMPLATE` | `{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}` | No | Go template for file organization |
| `PROVIDER_URL` | `http://127.0.0.1:8000` | No | Primary music catalog API URL (fallback providers managed via Settings UI) |
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

**Rate limiting**: Each provider enforces a 200ms minimum interval between requests. The global rate limit (`RATE_LIMIT_*`) applies across all providers.
| `SKIP_AUTH` | `false` | No | Set to `true` to disable authentication entirely |
| `THEME` | `golden` | No | Default application theme (can be overridden in Settings) |
| `FFMPEG_PATH` | (system) | No | Path to ffmpeg binary (required for MP4/M4A tagging - hi-res downloads often come as MP4) |
| `FFPROBE_PATH` | (system) | No | Path to ffprobe binary |

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

### MusicBrainz Cache

The `MUSICBRAINZ_CACHE_TTL` controls how long MusicBrainz API responses are cached:

```bash
# Cache for 1 day
MUSICBRAINZ_CACHE_TTL=1d

# Cache for 2 weeks
MUSICBRAINZ_CACHE_TTL=336h  # or 14d
```

MusicBrainz has strict rate limits, so caching their responses for extended periods is recommended.

## Genre Map Settings

Genre mapping normalizes MusicBrainz subgenre tags into main genres. Configured via the Settings page UI.

### How It Works

1. MusicBrainz returns genre tags with vote counts (e.g., `"death metal": 5, "thrash metal": 3`)
2. Each tag is mapped through the genre map (lowercase key → normalized genre)
3. Counts are aggregated by normalized genre
4. The genre with the highest total count is selected

### Default Categories

| Category | Example Mappings |
|----------|------------------|
| Rock | rock, alternative rock, indie rock, punk, grunge |
| Metal | metal, death metal, black metal, thrash metal |
| Pop | pop, indie pop, synthpop, dance pop |
| Hip-Hop | hip hop, rap, trap, drill |
| R&B | r&b, soul, neo soul, funk |
| Electronic | electronic, edm, house, techno, dubstep |
| Latin | latin, reggaeton, salsa, bachata |
| Regional Mexican | banda, norteño, corridos, mariachi |
| Country | country, americana, alt-country |
| Jazz | jazz, smooth jazz, bebop |
| Classical | classical, opera, baroque |
| Folk | folk, indie folk, acoustic |
| Reggae | reggae, dancehall, ska |
| Blues | blues |
| Soundtrack | soundtrack, film score |

### Custom Map

To override or extend the default map, enter JSON in the Settings page:

```json
{
  "dark ambient": "Electronic",
  "indie folk": "Folk",
  "synthwave": "Electronic"
}
```

Click **Reset to Default** to clear the custom map and revert to the built-in mappings.

## Authentication

Basic HTTP authentication is optional:
- Set `NAVIDRUMS_PASSWORD` to enable authentication
- Leave `NAVIDRUMS_PASSWORD` empty to disable authentication
- When password is set, `NAVIDRUMS_USERNAME` must also be set

## Provider Management

The primary provider is configured via `PROVIDER_URL`. Additional fallback providers can be managed through the Settings UI:

### Providers Table

| Field | Description |
|-------|-------------|
| Position | Order of preference (lower = higher priority) |
| Name | Display name for the provider |
| URL | Provider API endpoint |

### How Fallback Works

1. Requests are attempted against the primary provider first
2. If the primary provider fails or returns no results, the next provider in the list is tried
3. Providers are tried in order until one succeeds or the list is exhausted

### Managing Providers

- Add new providers via the Settings UI
- Reorder providers by dragging rows in the table
- Edit or delete providers from the Settings UI
- The primary provider (from `PROVIDER_URL`) is automatically added as the first provider on first run

## Validation

Configuration is validated at startup. Common validation errors:

- Invalid `PORT` (not a number or out of range)
- Invalid `PROVIDER_URL` (not a valid URL)
- Invalid `QUALITY` (not one of allowed values)
- Invalid `SUBDIR_TEMPLATE` (template parsing error)
- Invalid `CACHE_TTL` (not a valid duration)
- Invalid `MUSICBRAINZ_CACHE_TTL` (not a valid duration)
- Missing `NAVIDRUMS_USERNAME` when password is set

## Docker Configuration

When running in Docker, mount volumes for persistence:

```bash
# Downloads directory
-v /host/path/to/music:/music

# Persistent data directory (database, etc.)
-v /host/path/data:/data
```

The container uses these internal paths by default:
- Downloads: `/music`
- Data: `/data` (database is stored at `/data/navidrums.db`)

## Example .env File

```bash
PORT=8080
DB_PATH=/data/navidrums.db
DOWNLOADS_DIR=/music
PROVIDER_URL=https://your-hifi-api.com
QUALITY=LOSSLESS
LOG_LEVEL=info
LOG_FORMAT=text
NAVIDRUMS_USERNAME=navidrums
NAVIDRUMS_PASSWORD=secure-password
SUBDIR_TEMPLATE={{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}
CACHE_TTL=12h
MUSICBRAINZ_CACHE_TTL=7d
MUSICBRAINZ_URL=https://musicbrainz.org/ws/2
THEME=golden
```

See [.env.sample](../.env.sample) for a minimal example.