# Music Downloader (Go Version)

A lightweight self-hosted web application for browsing and downloading music to your Navidrome library.
Optimized for Raspberry Pi 4B.

## Features

- Browse Artists, Albums, and Playlists from remote Hifi API.
- Download queuing system with concurrency control (Max 2 downloads).
- Automatic retries and resume support.
- FLAC metadata tagging using ffmpeg.
- HTMX-powered responsive UI (no JSON APIs for frontend).
- Efficient SQLite database.

## Prerequisites

- **Go 1.22+**
- **ffmpeg** (must be in PATH for tagging)
- **Hifi API** running (default: `http://127.0.0.1:8000`)

## Installation

1.  Clone the repository.
2.  Navigate to `golang_version`.
3.  Build the server:
    ```bash
    go build -o server ./cmd/server
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

## Usage

1.  Start the server:
    ```bash
    ./server
    ```
2.  Open browser at `http://localhost:8080`.
3.  Search for music and click download.
4.  Check the "Queue" tab for progress.

## Development

Run tests:
```bash
go test ./...
```
