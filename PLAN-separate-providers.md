# Plan: Separate Metadata & Download API Providers

## Problem

A single `HifiProvider` with one `httpclient.Client` (1200ms rate limit) handles ALL API calls — metadata and downloads — sequentially. With concurrent workers (concurrency=2), playlist downloads with 25+ tracks trigger 50+ serialized API calls × 1.2s = 60+ seconds of rate-limit waiting alone, plus 429 backoff.

## Solution

Split into two `HifiProvider` instances, each with their own HTTP client and rate limiter:

- **Metadata API** — handles `/info/`, `/album/`, `/artist/`, `/playlist/`, `/lyrics/`, etc.
- **Download API** — handles `/track/` (stream/manifest endpoints)

This allows the user to run separate infrastructure (e.g., Triton for metadata, Binarium for downloads) with independent rate limits.

## UX

### Settings Page

```
┌─ API Configuration ──────────────────────────────────────┐
│                                                         │
│  Metadata API                                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │ triton                                    [Select] │  │
│  │ http://localhost:8001                              │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│  Download API                                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │ binarium                                   [Select] │  │
│  │ http://localhost:8002                              │  │
│  └───────────────────────────────────────────────────┘  │
│                                                         │
│  env vars override: PROVIDER_METADATA_URL,               │
│  PROVIDER_DOWNLOAD_URL (shown with "(env)" badge)       │
└─────────────────────────────────────────────────────────┘
```

- Each active API shows name + URL in a card
- "Select" opens a modal with the provider list (env defaults + custom)
- Any saved provider can be chosen for either role
- Changes save immediately to DB
- "Add Provider" form stays unchanged
- env vars (`PROVIDER_METADATA_URL`, `PROVIDER_DOWNLOAD_URL`) override DB settings — shown with "(env)" badge

## Changes

### 1. `internal/config/config.go`

Add two new config fields:
- `ProviderMetadataURL` — from env `PROVIDER_METADATA_URL`
- `ProviderDownloadURL` — from env `PROVIDER_DOWNLOAD_URL`

Validation:
- If both new vars are set: validate both URLs
- If only `PROVIDER_URL` is set: validate it, use it for both (backward compat, log deprecation warning)
- If both new vars are set alongside `PROVIDER_URL`: new vars take precedence, log deprecation warning

### 2. `internal/store/settings.go`

Add two new setting constants:
- `SettingActiveMetadataProvider = "active_metadata_provider"`
- `SettingActiveDownloadProvider = "active_download_provider"`

Existing `SetProvider` continues to set both (backward compat for existing DB rows).

### 3. `internal/catalog/hifi.go`

`HifiProvider` struct gains a second client:
```go
type HifiProvider struct {
    BaseURL    string
    client     *httpclient.Client  // metadata client (1200ms)
    downloadClient *httpclient.Client  // download client (1200ms)
}
```

Constructor changes:
- `NewHifiProvider(baseURL string)` — single client, backward compat
- `NewHifiProviderDual(metadataURL, downloadURL string)` — two clients

Method routing:
- All `p.get()` calls → `p.client.Do()` (metadata)
- `GetStream()` manifest fetch → `p.client.Do()` (metadata)
- `GetStream()` segment/CDN downloads → `p.downloadClient.Do()` (download)
- `multiSegmentReader` uses `p.downloadClient.GetUnderlyingClient()`

### 4. `internal/catalog/manager.go`

`ProviderManager` struct:
```go
type ProviderManager struct {
    provider          Provider
    metadataProvider  *HifiProvider
    downloadProvider  *HifiProvider
    cached           *CachedProvider
    metadataURL      string
    downloadURL      string
    defaultMetadataURL string
    defaultDownloadURL string
    mu               sync.RWMutex
}
```

Constructor `NewProviderManager(metURL, dlURL string, ...)`:
- If either URL is empty, use the same URL for both (backward compat)
- Creates `metadataProvider` and `downloadProvider` via `NewHifiProviderDual`
- Cached provider wraps metadata provider

Methods:
- `GetProvider()` — returns cached provider (metadata)
- `GetDownloadProvider()` — returns download provider (not cached)
- `GetBaseURL()` → returns `metadataURL`
- `GetDefaultURL()` → returns `defaultMetadataURL`
- `GetSettingsJSON()` → includes `metadataURL`, `downloadURL`, `defaultMetadataURL`, `defaultDownloadURL`
- `SetProvider(url)` — sets both metadata and download URLs + saves both settings (backward compat)
- `SetMetadataProvider(url)` — sets metadata URL + saves `active_metadata_provider`
- `SetDownloadProvider(url)` — sets download URL + saves `active_download_provider`
- `SetProvider(baseURL string)` — deprecated, sets both to same URL

### 5. `internal/http/routes.go`

`GetProvidersHTMX` response gains:
```json
{
  "predefined": [...],
  "custom": [...],
  "metadataActive": "http://localhost:8001",
  "metadataDefault": "http://localhost:8000",
  "downloadActive": "http://localhost:8002",
  "downloadDefault": "http://localhost:8000"
}
```

New routes:
- `SetMetadataProviderHTMX` — sets metadata only
- `SetDownloadProviderHTMX` — sets download only

`SetProviderHTMX` continues to set both (backward compat for existing UI calls).

`AddCustomProviderHTMX` / `RemoveCustomProviderHTMX` — unchanged.

### 6. `internal/http/handler.go`

Register new routes:
```go
r.Post("/htmx/provider/metadata/set", h.SetMetadataProviderHTMX)
r.Post("/htmx/provider/download/set", h.SetDownloadProviderHTMX)
```

### 7. `cmd/server/main.go`

Pass both URLs to `NewProviderManager`:
```go
providerManager := catalog.NewProviderManager(
    cfg.ProviderMetadataURL,
    cfg.ProviderDownloadURL,
    db,
    cfg.CacheTTL,
    appLogger,
)
```

Existing settings override (`SetProvider`) still works — sets both to same URL.

### 8. `internal/app/downloader.go`

`Download()` calls stream from the download provider:
```go
stream, mimeType, err := provider.GetStream(ctx, track.ProviderID, d.config.Quality)
```
becomes:
```go
provider := d.providerManager.GetDownloadProvider()
stream, mimeType, err := provider.GetStream(ctx, track.ProviderID, d.config.Quality)
```

### 9. `web/templates/settings.html`

Replace "Current Provider" section with two API cards (Metadata API + Download API).

JavaScript changes:
- `loadProviders()` reads new fields, shows "(env)" badge
- `setMetadataProvider(url)` calls `/htmx/provider/metadata/set`
- `setDownloadProvider(url)` calls `/htmx/provider/download/set`
- "Select" button opens modal with provider list
- "Add Provider" form unchanged

### 10. `AGENTS.md`

Update env vars table:
- Add `PROVIDER_METADATA_URL` — Music catalog API URL for metadata endpoints
- Add `PROVIDER_DOWNLOAD_URL` — Music catalog API URL for stream/download endpoints
- Mark `PROVIDER_URL` as deprecated (falls back to both if new vars not set)

## Backward Compatibility

| Scenario | Behavior |
|---|---|
| Only `PROVIDER_URL` set | Both APIs use same URL + client (current behavior) |
| Both new vars set | Dual clients, dual URLs |
| Only new var set | Error on startup ("both required") |
| No env vars | Use defaults (both same) |
| DB has `active_provider` setting | `SetProvider` sets both to same, migration-friendly |
| Existing code calling `SetProvider` | Works, sets both (backward compat) |

## Rate Limiting Summary

| Endpoint Type | Rate Limit | Notes |
|---|---|---|
| Metadata (`/info/`, `/album/`, etc.) | 1200ms per `metadataClient` | Shares same limiter as MusicBrainz |
| Streaming (`/track/` manifest fetch) | 1200ms per `metadataClient` | Same API as metadata |
| Stream segments/CDN | 1200ms per `downloadClient` | If URLs differ, separate limiter |

## Files Changed

1. `internal/config/config.go`
2. `internal/store/settings.go`
3. `internal/catalog/hifi.go`
4. `internal/catalog/manager.go`
5. `internal/http/routes.go`
6. `internal/http/handler.go`
7. `cmd/server/main.go`
8. `internal/app/downloader.go`
9. `web/templates/settings.html`
10. `AGENTS.md`
