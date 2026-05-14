# Multi-Provider Type Support

**Date:** 2026-05-13

## Goal

Add provider types (HIFI, QOBUZ) with separate ordered URL lists, plus per-operation default API type selectors (metadata, download, streaming). Currently all providers are identical `HifiProvider` instances in one flat list with no type awareness.

---

## 1. Database

### Schema Migration

Add `type` column to `providers`:
```sql
ALTER TABLE providers ADD COLUMN type TEXT NOT NULL DEFAULT 'hifi';
```

Existing rows auto-migrate to `type='hifi'`. Positions remain per-type (each type has its own 0,1,2... ordering).

### Settings

Three settings keys (already defined as constants in `store/settings.go`):
```
active_metadata_provider  → "hifi" | "qobuz"  (default: "hifi")
active_download_provider  → "hifi" | "qobuz"  (default: "hifi")
active_streaming_provider → "hifi" | "qobuz"  (default: "hifi")
```

No migration needed — first read returns `""`, code defaults to `"hifi"`.

---

## 2. Store Layer (`internal/store`)

### ProviderRecord

Add `Type` field:
```go
type ProviderRecord struct {
    ID       int64  `json:"id"`
    Type     string `json:"type"`      // NEW
    Position int    `json:"position"`
    URL      string `json:"url"`
    Name     string `json:"name"`
}
```

### ProvidersRepo Changes

| Method | Current | New |
|--------|---------|-----|
| `Create(url, name)` | global position | `Create(ptype, url, name)` — position scoped to type |
| `ListOrdered()` | all providers | `ListByType(ptype)` — filtered by type |
| `GetByPosition(pos)` | global position | `GetByPosition(ptype, pos)` — scoped to type |
| `Delete(id)` | — | unchanged |
| `Reorder(ids)` | — | unchanged (works within type since positions are type-scoped) |

Query for `ListByType`:
```sql
SELECT id, type, url, name, position FROM providers WHERE type = ? ORDER BY position ASC
```

### Schema Update

In `schema.go`, `providers` table definition gains `type TEXT NOT NULL DEFAULT 'hifi'`.

---

## 3. Domain (`internal/catalog`)

### ProviderType

```go
type ProviderType string

const (
    ProviderTypeHifi  ProviderType = "hifi"
    ProviderTypeQobuz ProviderType = "qobuz"
)
```

### QobuzProvider (Stub)

New file `internal/catalog/qobuz.go`. Implements `Provider` interface. All methods return `errors.New("qobuz provider not yet implemented")`. Full implementation is out of scope for this change.

### ProviderFactory

```go
func NewProvider(providerType ProviderType, baseURL string) Provider {
    switch providerType {
    case ProviderTypeQobuz:
        return NewQobuzProvider(baseURL)
    default:
        return NewHifiProvider(baseURL)
    }
}
```

---

## 4. FallbackProvider

Gains `providerType ProviderType` field. `getProviders()` queries only providers of that type:

```go
func (f *FallbackProvider) getProviders() []Provider {
    // cached list check...
    var providers []Provider
    dbProviders, _ := f.manager.providers.ListByType(f.providerType)
    for _, p := range dbProviders {
        providers = append(providers, NewProvider(f.providerType, p.URL))
    }
    f.cachedProviders = providers
    return providers
}
```

No system default URL prepended. Empty list → all operations return errors.

---

## 5. ProviderManager

### Removed
- `defaultURL` field
- `SetProvider(url)` method
- `GetBaseURL()` method
- `GetSettingsJSON()` method
- `ProviderSettings`, `CustomProvider` types

### New Structure

```go
type ProviderManager struct {
    logger    Logger
    providers *store.ProvidersRepo
    settings  *store.SettingsRepo
    cacheTTL  time.Duration
    db        *store.DB

    metadataChain  *CachedProvider
    downloadChain  *CachedProvider
    streamingChain *CachedProvider
    mu             sync.RWMutex
}
```

### New Constructor

```go
func NewProviderManager(db *store.DB, settings *store.SettingsRepo, cacheTTL time.Duration, logger Logger) *ProviderManager
```

### New Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `GetMetadataProvider()` | `Provider` | Reads `active_metadata_provider` setting → typed chain |
| `GetDownloadProvider()` | `Provider` | Reads `active_download_provider` setting → typed chain |
| `GetStreamingProvider()` | `Provider` | Reads `active_streaming_provider` setting → typed chain |
| `GetProvider(ptype ProviderType)` | `Provider` | Low-level: returns cached chain for explicit type |
| `InvalidateAllCaches()` | — | Invalidates all three provider chains |

`getOrCreateChain(ptype)` lazily creates a `CachedProvider` wrapping `FallbackProvider{providerType: ptype}`.

---

## 6. App Layer

### `internal/app/downloader.go`

```go
// Before: d.providerManager.GetProvider()
// After:  d.providerManager.GetDownloadProvider()
```

### `internal/app/enricher.go`

```go
// Before: e.providerManager.GetProvider()
// After:  e.providerManager.GetMetadataProvider()
```

---

## 7. Config (`internal/config`)

Remove `ProviderURL` field from `Config` struct. Remove env var loading (`PROVIDER_URL`). Remove validation for `ProviderURL`.

---

## 8. Main.go

Replace:
```go
providerManager := catalog.NewProviderManager(cfg.ProviderURL, db, cfg.CacheTTL, appLogger)
// + savedMetURL loading + SetProvider
```

With:
```go
settingsRepo := store.NewSettingsRepo(db)
providerManager := catalog.NewProviderManager(db, settingsRepo, cfg.CacheTTL, appLogger)
```

No startup URL loading — the manager reads settings per-call.

---

## 9. HTTP Handlers

### Provider Endpoints

| Endpoint | Change |
|----------|--------|
| `GET /htmx/providers` | Returns `{"hifi": [...], "qobuz": [...]}` instead of flat list |
| `POST /htmx/provider` | Gains `?type=` query param |
| `POST /htmx/providers/reorder` | Gains `?type=` query param — reorder within type only |
| `DELETE /htmx/provider` | Unchanged |

### New Settings Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /htmx/settings/providers` | Returns `{"active_metadata_provider": "hifi", "active_download_provider": "hifi", "active_streaming_provider": "hifi"}` |
| `POST /htmx/settings/providers` | Body: `{"key": "active_download_provider", "value": "qobuz"}` — saves setting, invalidates caches |

### Handler Updates

All callers of `h.ProviderManager.GetProvider()` are updated:
- `SearchHTMX` → `GetMetadataProvider()`
- `LuckyHTMX` → `GetMetadataProvider()`
- Artist/Album/Playlist/Search/Track pages → `GetMetadataProvider()`
- Similar albums/artists → `GetMetadataProvider()`
- `StreamTrack` → `GetStreamingProvider()`
- `EnrichHiFiHTMX` → `GetMetadataProvider()` (renamed conceptually to "enrich from metadata provider")

---

## 10. Template (`web/templates/settings.html`)

### Remove
- `data-default-url` attribute on provider-list div
- Single provider section with "Primary" system default badge
- `{{.DefaultURL}}` template variable usage

### Add

Three sections:
1. **HIFI Providers** — add/remove form + reorderable URL list (`id="hifi-provider-list"`)
2. **QOBUZ Providers** — add/remove form + reorderable URL list (`id="qobuz-provider-list"`)
3. **Default API Selectors** — three dropdowns for metadata/download/streaming

### JavaScript

- Per-type load/add/move functions
- `loadDefaultAPIs()` fetches current settings and sets dropdowns
- `saveDefaultAPI(key, value)` saves on dropdown change
- No `data-default-url` reference — removed

---

## 11. Files Created

| File | Purpose |
|------|---------|
| `internal/catalog/qobuz.go` | QobuzProvider stub |

## 12. Files Modified

| File | Changes |
|------|---------|
| `internal/store/schema.go` | `type` column in providers table |
| `internal/store/providers.go` | `Type` field, typed CRUD methods |
| `internal/catalog/provider.go` | `ProviderType` type |
| `internal/catalog/fallback.go` | Type-aware fallback |
| `internal/catalog/manager.go` | Multi-chain, setting-aware |
| `internal/config/config.go` | Remove ProviderURL |
| `cmd/server/main.go` | New constructor, settings repo |
| `internal/http/handler.go` | SettingsPage no longer passes DefaultURL |
| `internal/http/routes.go` | Typed provider endpoints, caller updates, new settings endpoints |
| `internal/http/stream.go` | Use streaming provider |
| `internal/app/downloader.go` | Use download provider |
| `internal/app/enricher.go` | Use metadata provider |
| `web/templates/settings.html` | Multi-section UI with type selectors |

---

## 13. Edge Cases

- **Empty provider list for type**: operations return errors (user must configure providers first)
- **Existing providers**: auto-migrate to type `'hifi'` via SQL DEFAULT
- **URL uniqueness**: remains global (across both types) — same URL can't be both HIFI and QOBUZ
- **No system default**: removes the dependency on `PROVIDER_URL` env var entirely
- **QOBUZ without implementation**: selecting QOBUZ as default without adding any QOBUZ URLs results in "qobuz provider not yet implemented" errors
