# Navidrums

A lightweight self-hosted web application for browsing and downloading music to your Navidrome library.
Optimized for low-end hardware.

## Features

- Browse Artists, Albums, and Playlists from remote Hifi API.
- Download queuing system with concurrency control (Max 2 downloads).
- **Provider Management**: Switch between multiple Hifi API endpoints and add custom providers.
- **Download History**: View last 20 completed/failed downloads.
- **Comprehensive Metadata Tagging**: Automatically tags downloaded files with:
  - Basic tags: Title, Artist, Album Artist, Album, Track/Disc Numbers
  - Extended metadata: Year, Genre, Label, ISRC, Copyright, Composer
  - Embedded album artwork in audio files
  - Album cover images saved to album folders (`cover.jpg`)
  - Playlist cover images saved to playlists folder
- Supports FLAC, MP3, and MP4 audio formats.
- Automatic retries and resume support.
- HTMX-powered responsive UI (no JSON APIs for frontend).
- Efficient SQLite database.

## Prerequisites

- **Go 1.22+** (for building from source)
- **Hifi API** running (default: `http://127.0.0.1:8000`)

## Installation

### Option 1: Download Pre-built Binary (Recommended)

1. Download the latest release for your platform from the [Releases page](https://github.com/cesargomez89/navidrums/releases):
   - **Linux (x86_64)**: `navidrums-linux-amd64`
   - **Linux (ARM64/Raspberry Pi)**: `navidrums-linux-arm64`
   - **macOS (Intel)**: `navidrums-darwin-amd64`
   - **macOS (Apple Silicon)**: `navidrums-darwin-arm64`
   - **Windows (x86_64)**: `navidrums-windows-amd64.exe`

2. Make the binary executable (Linux/macOS):
   ```bash
   chmod +x navidrums-*
   ```

3. Optionally, move it to a directory in your PATH:
   ```bash
   sudo mv navidrums-* /usr/local/bin/navidrums
   ```

**Note:** The binary is self-contained with all templates and assets embedded. You only need the single executable file to run the application.

### Option 2: Build from Source

1. Clone the repository.
2. Build the server:
   ```bash
   go build -o navidrums ./cmd/server
   ```

## Configuration

Environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `DB_PATH` | `navidrums.db` | SQLite database path |
| `DOWNLOADS_DIR` | `~/Downloads/navidrums` | Output directory for music |
| `PROVIDER_URL` | `http://127.0.0.1:8000` | URL of the Hifi API |
| `QUALITY` | `LOSSLESS` | Download quality (`LOSSLESS`, `HI_RES_LOSSLESS`, `HIGH`, `LOW`) |
| `USE_MOCK` | `false` | Set to `true` to use Mock provider |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `text` | Log output format (`text`, `json`) |
| `NAVIDRUMS_USERNAME` | `navidrums` | Username for the Navidrome web interface |
| `NAVIDRUMS_PASSWORD` |  | Password for the Navidrome web interface |

## Usage

1. Start the server:
   ```bash
   ./navidrums
   ```
2. Open browser at `http://localhost:8080`.
3. Search for music and click download.
4. Check the "Queue" tab for progress.

## Self-Hosted Server Setup

### Running as a Systemd Service (Linux)

1. Create a systemd service file at `/etc/systemd/system/navidrums.service`:

   ```ini
   [Unit]
   Description=Navidrums Music Downloader
   After=network.target

   [Service]
   Type=simple
   User=YOUR_USERNAME
   WorkingDirectory=/home/YOUR_USERNAME/navidrums
   Environment="PORT=8080"
   Environment="DB_PATH=/home/YOUR_USERNAME/navidrums/navidrums.db"
   Environment="DOWNLOADS_DIR=/home/YOUR_USERNAME/Music"
   Environment="PROVIDER_URL=http://127.0.0.1:8000"
   Environment="QUALITY=LOSSLESS"
   Environment="NAVIDRUMS_USERNAME=navidrums"
   Environment="NAVIDRUMS_PASSWORD=password"
   ExecStart=/usr/local/bin/navidrums
   Restart=always
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   ```

2. Replace `YOUR_USERNAME` with your actual username and adjust paths as needed.

3. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable navidrums
   sudo systemctl start navidrums
   ```

4. Check service status:
   ```bash
   sudo systemctl status navidrums
   ```

### Reverse Proxy Setup (Optional)

To expose Navidrums securely with HTTPS, use a reverse proxy like Nginx or Caddy.

**Example Caddy configuration:**
```
navidrums.yourdomain.com {
    reverse_proxy localhost:8080
}
```

**Example Nginx configuration:**
```nginx
server {
    listen 80;
    server_name navidrums.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Development

Run tests:
```bash
go test ./...
```

## Creating a Release

To create a new release:

1. Tag the commit:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically build binaries for all platforms and create a release.

## Architecture

Navidrums uses a two-table architecture:
- **Jobs table**: Minimal work queue (id, type, status, source_id)
- **Tracks table**: Full metadata and download state for all tracks

This separation allows storing complete track metadata for features like custom download paths and better history tracking.

See [ARCHITECTURE.md](ARCHITECTURE.md) and [DOMAIN.md](DOMAIN.md) for technical details.

