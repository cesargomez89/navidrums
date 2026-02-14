# Configuration

## Required

DOWNLOAD_PATH=./downloads
DB_PATH=./navidrums.db
NAVIDRUMS_PASSWORD=password

## Optional

PORT=8080
WORKER_CONCURRENCY=3
PROVIDER_TIMEOUT=30s
NAVIDRUMS_USERNAME=navidrums

## Behavior

Increasing WORKER_CONCURRENCY increases parallel downloads.
Changing DB_PATH requires migration.
