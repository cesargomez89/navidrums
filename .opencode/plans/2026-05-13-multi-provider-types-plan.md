# Multi-Provider Type Support — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add provider types (HIFI, QOBUZ) with separate ordered URL lists and per-operation default API type selectors (metadata, download, streaming).

**Architecture:** Add `type` column to providers table; create `ProviderType` domain type; make `FallbackProvider` type-aware with separate provider lists per type; give `ProviderManager` three chains (metadata/download/streaming) selected via settings; create `QobuzProvider` stub; update UI with per-type sections and default API selectors.

**Tech Stack:** Go 1.21+, SQLite, Chi router, HTMX, vanilla JS

---

### Task 1: Add ProviderType domain type

**Files:**
- Modify: `internal/catalog/provider.go`

- [ ] **Step 1: Add ProviderType to provider.go**

Add after the `Provider` interface definition (after line ~21):

```go
type ProviderType string

const (
	ProviderTypeHifi  ProviderType = "hifi"
	ProviderTypeQobuz ProviderType = "qobuz"
)
```

- [ ] **Step 2: Verify build compiles**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/catalog/provider.go
git commit -m "feat: add ProviderType domain type (hifi, qobuz)"
```

---

### Task 2: Create QobuzProvider stub

**Files:**
- Create: `internal/catalog/qobuz.go`

- [ ] **Step 1: Create the stub file**

```go
package catalog

import (
	"context"
	"errors"
	"io"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type QobuzProvider struct {
	BaseURL string
}

func NewQobuzProvider(baseURL string) *QobuzProvider {
	return &QobuzProvider{BaseURL: baseURL}
}

var errQobuzNotImplemented = errors.New("qobuz provider not yet implemented")

func (p *QobuzProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	return nil, "", errQobuzNotImplemented
}

func (p *QobuzProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetSimilarArtists(ctx context.Context, id string) ([]domain.Artist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	return "", "", errQobuzNotImplemented
}

func (p *QobuzProvider) GetRecommendations(ctx context.Context, id string) ([]domain.CatalogTrack, error) {
	return nil, errQobuzNotImplemented
}

var _ Provider = (*QobuzProvider)(nil)
```

- [ ] **Step 2: Verify build compiles**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/catalog/qobuz.go
git commit -m "feat: add QobuzProvider stub"
```

---

### Task 3: Add ProviderFactory

**Files:**
- Modify: `internal/catalog/hifi.go` (add NewProvider factory function at the bottom)

- [ ] **Step 1: Add factory function**

Add at the end of `internal/catalog/hifi.go`:

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

- [ ] **Step 2: Verify build compiles**

Run: `go build ./...`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add internal/catalog/hifi.go
git commit -m "feat: add NewProvider factory dispatching by ProviderType"
```

---

### Task 4: Update database schema with type column

**Files:**
- Modify: `internal/store/schema.go`

- [ ] **Step 1: Add type column to providers table definition**

In `internal/store/schema.go`, replace the `providers` table definition (lines 144-151):

Replace:
```sql
CREATE TABLE IF NOT EXISTS providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT UNIQUE NOT NULL,
	name TEXT,
	position INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

With:
```sql
CREATE TABLE IF NOT EXISTS providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL DEFAULT 'hifi',
	url TEXT UNIQUE NOT NULL,
	name TEXT,
	position INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

- [ ] **Step 2: Verify build compiles**

Run: `go build ./...`
Expected: No errors (the column is only in SQL — Go code changed next).

- [ ] **Step 3: Commit**

```bash
git add internal/store/schema.go
git commit -m "feat: add type column to providers table"
```

---

### Task 5: Update ProviderRecord and ProvidersRepo

**Files:**
- Modify: `internal/store/providers.go`
- Modify: `internal/store/providers_test.go`

- [ ] **Step 1: Add Type field to ProviderRecord**

In `internal/store/providers.go`, add `Type` field:

```go
type ProviderRecord struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Name     string `json:"name"`
}
```

- [ ] **Step 2: Update Create to accept and store providerType**

Replace:
```go
func (r *ProvidersRepo) Create(url, name string) (int64, error) {
	var id int64
	err := r.db.RunInTx(func(txDB *DB) error {
		var maxPos int
		err := txDB.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM providers`).Scan(&maxPos)
		if err != nil {
			return err
		}
		query := `INSERT INTO providers (url, name, position) VALUES (?, ?, ?) RETURNING id`
		row := txDB.QueryRowx(query, url, name, maxPos+1)
		return row.Scan(&id)
	})
	return id, err
}
```

With:
```go
func (r *ProvidersRepo) Create(providerType, url, name string) (int64, error) {
	var id int64
	err := r.db.RunInTx(func(txDB *DB) error {
		var maxPos int
		err := txDB.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM providers WHERE type = ?`, providerType).Scan(&maxPos)
		if err != nil {
			return err
		}
		query := `INSERT INTO providers (type, url, name, position) VALUES (?, ?, ?, ?) RETURNING id`
		row := txDB.QueryRowx(query, providerType, url, name, maxPos+1)
		return row.Scan(&id)
	})
	return id, err
}
```

- [ ] **Step 3: Rename ListOrdered to ListByType with type filtering**

Replace:
```go
func (r *ProvidersRepo) ListOrdered() ([]ProviderRecord, error) {
	var providers []ProviderRecord
	query := `SELECT id, url, name, position FROM providers ORDER BY position ASC`
	err := r.db.Select(&providers, query)
	return providers, err
}
```

With:
```go
func (r *ProvidersRepo) ListByType(providerType string) ([]ProviderRecord, error) {
	var providers []ProviderRecord
	query := `SELECT id, type, url, name, position FROM providers WHERE type = ? ORDER BY position ASC`
	err := r.db.Select(&providers, query, providerType)
	return providers, err
}
```

- [ ] **Step 4: Update GetByPosition to accept providerType**

Replace:
```go
func (r *ProvidersRepo) GetByPosition(pos int) (*ProviderRecord, error) {
	query := `SELECT id, url, name, position FROM providers WHERE position = ?`
	var provider ProviderRecord
	err := r.db.Get(&provider, query, pos)
	if err != nil {
		return nil, err
	}
	return &provider, nil
}
```

With:
```go
func (r *ProvidersRepo) GetByPosition(providerType string, pos int) (*ProviderRecord, error) {
	query := `SELECT id, type, url, name, position FROM providers WHERE type = ? AND position = ?`
	var provider ProviderRecord
	err := r.db.Get(&provider, query, providerType, pos)
	if err != nil {
		return nil, err
	}
	return &provider, nil
}
```

- [ ] **Step 5: Also keep ListOrdered for backward compat (remove later)**

At this point also update Reorder to be type-scoped. The current Reorder updates all positions with +1000, then reassigns. We need to scope it to type:

Replace the Reorder method entirely:
```go
func (r *ProvidersRepo) Reorder(ids []int64) error {
	return r.db.RunInTx(func(txDB *DB) error {
		if len(ids) == 0 {
			return nil
		}
		_, err := txDB.Exec(`UPDATE providers SET position = position + 1000`)
		if err != nil {
			return err
		}
		for i, id := range ids {
			_, err := txDB.Exec(`UPDATE providers SET position = ? WHERE id = ?`, i, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
```

With (type-scoped mass update):
```go
func (r *ProvidersRepo) Reorder(ids []int64) error {
	return r.db.RunInTx(func(txDB *DB) error {
		if len(ids) == 0 {
			return nil
		}
		// shift positions within the same type as the first provider
		_, err := txDB.Exec(`
			UPDATE providers SET position = position + 1000
			WHERE type = (SELECT type FROM providers WHERE id = ?)`, ids[0])
		if err != nil {
			return err
		}
		for i, id := range ids {
			_, err := txDB.Exec(`UPDATE providers SET position = ? WHERE id = ?`, i, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
```

- [ ] **Step 6: Update providers_test.go**

In `internal/store/providers_test.go`, update all references:

- `Create(url, name)` → `Create("hifi", url, name)`
- `ListOrdered()` → `ListByType("hifi")`
- `GetByPosition(pos)` → `GetByPosition("hifi", pos)`
- No change to `Delete`, `Exists`, `Update`
- Add test for QOBUZ type creation and listing: `Create("qobuz", url, name)` → `ListByType("qobuz")` returns only qobuz providers
- Verify that HIFI and QOBUZ positions are independent (each starts at 0)

- [ ] **Step 7: Run tests to verify**

Run: `go test ./internal/store/ -v -run TestProvider`
Expected: All provider tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/store/providers.go internal/store/providers_test.go
git commit -m "feat: add type-aware provider CRUD (type column, ListByType, scoped Reorder)"
```

---

### Task 6: Update FallbackProvider to be type-aware

**Files:**
- Modify: `internal/catalog/fallback.go`

- [ ] **Step 1: Add providerType field to FallbackProvider**

```go
type FallbackProvider struct {
	manager         *ProviderManager
	providerType    ProviderType
	cachedProviders []Provider
	cacheExpiry     time.Time
	cacheMu         sync.Mutex
}
```

- [ ] **Step 2: Update getProviders() to query by type, remove system default**

Replace the entire `getProviders()` method (lines 22-47):

```go
func (f *FallbackProvider) getProviders() []Provider {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	if f.cachedProviders != nil && time.Now().Before(f.cacheExpiry) {
		return f.cachedProviders
	}

	var providers []Provider

	if f.manager != nil && f.manager.providers != nil {
		storeProviders, _ := f.manager.providers.ListByType(string(f.providerType))
		for _, p := range storeProviders {
			providers = append(providers, NewProvider(f.providerType, p.URL))
		}
	}

	f.cachedProviders = providers
	f.cacheExpiry = time.Now().Add(providerCacheTTL)
	return providers
}
```

- [ ] **Step 3: Verify build compiles**

Run: `go build ./...`
Expected: No errors yet, even though ProviderManager hasn't been updated (the fallback provider doesn't create itself).

- [ ] **Step 4: Commit**

```bash
git add internal/catalog/fallback.go
git commit -m "feat: make FallbackProvider type-aware with per-type provider lists"
```

---

### Task 7: Rewrite ProviderManager with multi-chain support

**Files:**
- Modify: `internal/catalog/manager.go`

- [ ] **Step 1: Rewrite the entire manager.go file**

```go
package catalog

import (
	"log/slog"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/store"
)

type Logger interface {
	With(keyValues ...any) *slog.Logger
	Info(msg string, keyValues ...any)
	Error(msg string, keyValues ...any)
}

type ProviderManager struct {
	logger    Logger
	providers *store.ProvidersRepo
	settings  *store.SettingsRepo
	cacheTTL  time.Duration
	db        *store.DB

	metadataChain  *CachedProvider
	downloadChain  *CachedProvider
	streamingChain *CachedProvider

	mu sync.RWMutex
}

func NewProviderManager(db *store.DB, settings *store.SettingsRepo, cacheTTL time.Duration, logger Logger) *ProviderManager {
	var providersRepo *store.ProvidersRepo
	if db != nil {
		providersRepo = store.NewProvidersRepo(db)
	}

	return &ProviderManager{
		logger:    logger,
		providers: providersRepo,
		settings:  settings,
		cacheTTL:  cacheTTL,
		db:        db,
	}
}

func (m *ProviderManager) readSetting(key string) ProviderType {
	if m.settings == nil {
		return ProviderTypeHifi
	}
	val, err := m.settings.Get(key)
	if err != nil || val == "" {
		return ProviderTypeHifi
	}
	pt := ProviderType(val)
	if pt != ProviderTypeHifi && pt != ProviderTypeQobuz {
		return ProviderTypeHifi
	}
	return pt
}

func (m *ProviderManager) getOrCreateChain(pt ProviderType) Provider {
	switch pt {
	case ProviderTypeQobuz:
		if m.streamingChain == nil { // placeholder — we use a map approach instead
			// The actual implementation uses a different pattern — see below
		}
	}
	// We use a map-based approach for simplicity
	return m.getChain(pt)
}

func (m *ProviderManager) getChain(pt ProviderType) Provider {
	// Create a new FallbackProvider per type with caching
	fb := &FallbackProvider{manager: m, providerType: pt}
	if m.db != nil {
		return NewCachedProvider(fb, &storeCache{store: m.db}, m.cacheTTL)
	}
	return fb
}

func (m *ProviderManager) GetProvider(pt ProviderType) Provider {
	return m.getChain(pt)
}

func (m *ProviderManager) GetMetadataProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveMetadataProvider))
}

func (m *ProviderManager) GetDownloadProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveDownloadProvider))
}

func (m *ProviderManager) GetStreamingProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveStreamingProvider))
}

func (m *ProviderManager) InvalidateAllCaches() {
	// Since chains are created per-call via getChain, the FallbackProvider
	// cache (30s TTL) handles staleness. But we force-invalidate the
	// fallback provider caches by resetting their expiry.
	// In practice, after a setting change or provider add/remove, callers
	// just need to know that the next getProviders() call will re-query the DB.
	// The providerCacheTTL of 30 seconds means worst case 30s delay.
	// For immediate effect, we could store and invalidate chains, but
	// for simplicity we just log the event.
	if m.logger != nil {
		m.logger.Info("Provider caches marked for invalidation")
	}
}
```

Wait, this approach doesn't properly cache the chains. Let me think again...

Actually, the simplest correct approach: create new FallbackProvider chains on each call. The FallbackProvider already has a 30-second internal cache (`cachedProviders`). So `getChain()` creating a new CachedProvider each time is wasteful but correct — the inner FallbackProvider cache handles it. But creating a new CachedProvider invalidates the CachedProvider's own cache, which is wasteful.

Better approach: cache the three chains lazily. Let me rewrite:

```go
package catalog

import (
	"log/slog"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/store"
)

type Logger interface {
	With(keyValues ...any) *slog.Logger
	Info(msg string, keyValues ...any)
	Error(msg string, keyValues ...any)
}

type ProviderManager struct {
	logger    Logger
	providers *store.ProvidersRepo
	settings  *store.SettingsRepo
	cacheTTL  time.Duration
	db        *store.DB

	metadataChain  *CachedProvider
	downloadChain  *CachedProvider
	streamingChain *CachedProvider

	mu sync.RWMutex
}

func NewProviderManager(db *store.DB, settings *store.SettingsRepo, cacheTTL time.Duration, logger Logger) *ProviderManager {
	var providersRepo *store.ProvidersRepo
	if db != nil {
		providersRepo = store.NewProvidersRepo(db)
	}

	return &ProviderManager{
		logger:    logger,
		providers: providersRepo,
		settings:  settings,
		cacheTTL:  cacheTTL,
		db:        db,
	}
}

func (m *ProviderManager) readSetting(key string) ProviderType {
	if m.settings == nil {
		return ProviderTypeHifi
	}
	val, err := m.settings.Get(key)
	if err != nil || val == "" {
		return ProviderTypeHifi
	}
	pt := ProviderType(val)
	if pt != ProviderTypeHifi && pt != ProviderTypeQobuz {
		return ProviderTypeHifi
	}
	return pt
}

func (m *ProviderManager) getOrCreateChain(pt ProviderType) *CachedProvider {
	fb := &FallbackProvider{manager: m, providerType: pt}
	if m.db != nil {
		return NewCachedProvider(fb, &storeCache{store: m.db}, m.cacheTTL)
	}
	// wrap in a simple pass-through cached provider even without DB
	return NewCachedProvider(fb, nil, m.cacheTTL)
}

func (m *ProviderManager) GetProvider(pt ProviderType) Provider {
	m.mu.RLock()
	var chain *CachedProvider
	switch pt {
	case ProviderTypeQobuz:
		// for QOBUZ, we don't cache the chain since it's less frequently used
		m.mu.RUnlock()
		return m.getOrCreateChain(pt)
	default:
		chain = m.metadataChain
	}
	m.mu.RUnlock()

	if chain != nil {
		return chain
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock
	switch pt {
	case ProviderTypeQobuz:
		return m.getOrCreateChain(pt)
	default:
		if m.metadataChain == nil {
			m.metadataChain = m.getOrCreateChain(pt)
		}
		return m.metadataChain
	}
}
```

Hmm, this is getting complicated. Let me simplify. The simplest approach that works:

**Approach**: `GetProvider(pt)` always creates a new chain. The inner FallbackProvider has a 30s cache, so repeated calls within 30s return the same provider list. After a provider change, we call `InvalidateAllCaches()` which... hmm, we'd need a way to clear FallbackProvider caches. Since FallbackProvider is per-chain and the chain is recreated each call, the cache is always fresh.

Wait, if we recreate the chain each call, the FallbackProvider is fresh and its cache is empty. So there's no caching at the FallbackProvider level either. That means every call hits the DB.

Better approach: cache the three chains in the ProviderManager. On `InvalidateAllCaches()`, set them to nil so they're recreated on next use.

Let me just go with a clean, simple implementation:

```go
package catalog

import (
	"log/slog"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/store"
)

type Logger interface {
	With(keyValues ...any) *slog.Logger
	Info(msg string, keyValues ...any)
	Error(msg string, keyValues ...any)
}

type ProviderManager struct {
	logger    Logger
	providers *store.ProvidersRepo
	settings  *store.SettingsRepo
	cacheTTL  time.Duration
	db        *store.DB

	metadataChain  *CachedProvider
	downloadChain  *CachedProvider
	streamingChain *CachedProvider

	mu sync.RWMutex
}

func NewProviderManager(db *store.DB, settings *store.SettingsRepo, cacheTTL time.Duration, logger Logger) *ProviderManager {
	var providersRepo *store.ProvidersRepo
	if db != nil {
		providersRepo = store.NewProvidersRepo(db)
	}

	return &ProviderManager{
		logger:    logger,
		providers: providersRepo,
		settings:  settings,
		cacheTTL:  cacheTTL,
		db:        db,
	}
}

func (m *ProviderManager) readSetting(key string) ProviderType {
	if m.settings == nil {
		return ProviderTypeHifi
	}
	val, err := m.settings.Get(key)
	if err != nil || val == "" {
		return ProviderTypeHifi
	}
	pt := ProviderType(val)
	if pt != ProviderTypeHifi && pt != ProviderTypeQobuz {
		return ProviderTypeHifi
	}
	return pt
}

func (m *ProviderManager) buildChain(pt ProviderType) *CachedProvider {
	fb := &FallbackProvider{manager: m, providerType: pt}
	var cacheStore *storeCache
	if m.db != nil {
		cacheStore = &storeCache{store: m.db}
	}
	return NewCachedProvider(fb, cacheStore, m.cacheTTL)
}

func (m *ProviderManager) GetProvider(pt ProviderType) Provider {
	m.mu.RLock()
	var chain *CachedProvider
	switch pt {
	case ProviderTypeHifi:
		chain = m.metadataChain
	case ProviderTypeQobuz:
		chain = m.downloadChain
	}
	m.mu.RUnlock()

	if chain != nil {
		return chain
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var chainPtr **CachedProvider
	switch pt {
	case ProviderTypeHifi:
		chainPtr = &m.metadataChain
	case ProviderTypeQobuz:
		chainPtr = &m.downloadChain
	}

	if *chainPtr == nil {
		*chainPtr = m.buildChain(pt)
	}
	return *chainPtr
}

func (m *ProviderManager) GetMetadataProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveMetadataProvider))
}

func (m *ProviderManager) GetDownloadProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveDownloadProvider))
}

func (m *ProviderManager) GetStreamingProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveStreamingProvider))
}

func (m *ProviderManager) InvalidateAllCaches() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadataChain = nil
	m.downloadChain = nil
	m.streamingChain = nil
}
```

Hmm wait, this has a problem. `GetMetadataProvider()` reads the setting to determine the type, then calls `GetProvider()`. But `GetProvider()` maps HIFI to `metadataChain` and QOBUZ to `downloadChain`. This is wrong — the mapping is by the ProviderType, not by the operation. If metadata is QOBUZ, it should use the QOBUZ chain, not some "downloadChain".

Let me fix: use a map instead of named fields:

```go
type ProviderManager struct {
	logger    Logger
	providers *store.ProvidersRepo
	settings  *store.SettingsRepo
	cacheTTL  time.Duration
	db        *store.DB

	chains map[ProviderType]*CachedProvider
	mu     sync.RWMutex
}
```

Then `GetProvider(pt)` looks up the chain in the map. Much cleaner.

Actually, even simpler — use just the two types as map keys:

```go
func (m *ProviderManager) GetProvider(pt ProviderType) Provider {
	m.mu.RLock()
	chain := m.chains[pt]
	m.mu.RUnlock()
	if chain != nil {
		return chain
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chains[pt] == nil {
		if m.chains == nil {
			m.chains = make(map[ProviderType]*CachedProvider)
		}
		m.chains[pt] = m.buildChain(pt)
	}
	return m.chains[pt]
}
```

And `InvalidateAllCaches()`:

```go
func (m *ProviderManager) InvalidateAllCaches() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chains = nil
}
```

This is clean and simple. Let me write the full file this way.

OK I need to be more careful. Let me re-think the full implementation and present it cleanly in the plan.

The plan file is getting large. Let me write it in the planner's output.<｜end▁of▁thinking｜>The plan is complex — let me write all of it now:

<｜｜DSML｜｜tool_calls>
<｜｜DSML｜｜invoke name="write">
<｜｜DSML｜｜parameter name="content" string="true"># Multi-Provider Type Support — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add provider types (HIFI, QOBUZ) with separate ordered URL lists and per-operation default API type selectors (metadata, download, streaming).

**Architecture:** Add `type` column to providers table; create `ProviderType` domain type; make `FallbackProvider` type-aware; give `ProviderManager` three chains (metadata/download/streaming) read from settings; create `QobuzProvider` stub; update UI with per-type sections and default API selectors.

**Tech Stack:** Go 1.21+, SQLite, Chi router, HTMX, vanilla JS

---

### Task 1: Add ProviderType domain type

**Files:** Modify: `internal/catalog/provider.go`

- [ ] **Step 1: Add after the Provider interface (after line ~21)**

```go
type ProviderType string

const (
	ProviderTypeHifi  ProviderType = "hifi"
	ProviderTypeQobuz ProviderType = "qobuz"
)
```

- [ ] **Step 2: Verify compiles**
```
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**
```
git add internal/catalog/provider.go
git commit -m "feat: add ProviderType domain type (hifi, qobuz)"
```

---

### Task 2: Create QobuzProvider stub

**Files:** Create: `internal/catalog/qobuz.go`

- [ ] **Step 1: Write the file**

```go
package catalog

import (
	"context"
	"errors"
	"io"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type QobuzProvider struct {
	BaseURL string
}

func NewQobuzProvider(baseURL string) *QobuzProvider {
	return &QobuzProvider{BaseURL: baseURL}
}

var errQobuzNotImplemented = errors.New("qobuz provider not yet implemented")

func (p *QobuzProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	return nil, "", errQobuzNotImplemented
}

func (p *QobuzProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetSimilarArtists(ctx context.Context, id string) ([]domain.Artist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	return "", "", errQobuzNotImplemented
}

func (p *QobuzProvider) GetRecommendations(ctx context.Context, id string) ([]domain.CatalogTrack, error) {
	return nil, errQobuzNotImplemented
}

var _ Provider = (*QobuzProvider)(nil)
```

- [ ] **Step 2: Verify compiles**
```
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**
```
git add internal/catalog/qobuz.go
git commit -m "feat: add QobuzProvider stub"
```

---

### Task 3: Add NewProvider factory function

**Files:** Modify: `internal/catalog/hifi.go` (add at end)

- [ ] **Step 1: Add factory at end of file**

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

- [ ] **Step 2: Verify compiles**
```
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**
```
git add internal/catalog/hifi.go
git commit -m "feat: add NewProvider factory dispatching by ProviderType"
```

---

### Task 4: Update DB schema — add type column

**Files:** Modify: `internal/store/schema.go`

- [ ] **Step 1: In the providers CREATE TABLE (~line 144), add type column**

Replace:
```sql
CREATE TABLE IF NOT EXISTS providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT UNIQUE NOT NULL,
	name TEXT,
	position INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

With:
```sql
CREATE TABLE IF NOT EXISTS providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL DEFAULT 'hifi',
	url TEXT UNIQUE NOT NULL,
	name TEXT,
	position INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

- [ ] **Step 2: Verify compiles**
```
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**
```
git add internal/store/schema.go
git commit -m "feat: add type column to providers table schema"
```

---

### Task 5: Update ProviderRecord and ProvidersRepo

**Files:** 
- Modify: `internal/store/providers.go`
- Modify: `internal/store/providers_test.go`

- [ ] **Step 1: Add Type field to ProviderRecord**

```go
type ProviderRecord struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Name     string `json:"name"`
}
```

- [ ] **Step 2: Update Create signature and query**

Replace the entire Create method:
```go
func (r *ProvidersRepo) Create(providerType, url, name string) (int64, error) {
	var id int64
	err := r.db.RunInTx(func(txDB *DB) error {
		var maxPos int
		err := txDB.QueryRow(`SELECT COALESCE(MAX(position), -1) FROM providers WHERE type = ?`, providerType).Scan(&maxPos)
		if err != nil {
			return err
		}
		query := `INSERT INTO providers (type, url, name, position) VALUES (?, ?, ?, ?) RETURNING id`
		row := txDB.QueryRowx(query, providerType, url, name, maxPos+1)
		return row.Scan(&id)
	})
	return id, err
}
```

- [ ] **Step 3: Replace ListOrdered with ListByType**

```go
func (r *ProvidersRepo) ListByType(providerType string) ([]ProviderRecord, error) {
	var providers []ProviderRecord
	query := `SELECT id, type, url, name, position FROM providers WHERE type = ? ORDER BY position ASC`
	err := r.db.Select(&providers, query, providerType)
	return providers, err
}
```

- [ ] **Step 4: Update GetByPosition signature**

```go
func (r *ProvidersRepo) GetByPosition(providerType string, pos int) (*ProviderRecord, error) {
	query := `SELECT id, type, url, name, position FROM providers WHERE type = ? AND position = ?`
	var provider ProviderRecord
	err := r.db.Get(&provider, query, providerType, pos)
	if err != nil {
		return nil, err
	}
	return &provider, nil
}
```

- [ ] **Step 5: Update Reorder to scope mass-update by type**

```go
func (r *ProvidersRepo) Reorder(ids []int64) error {
	return r.db.RunInTx(func(txDB *DB) error {
		if len(ids) == 0 {
			return nil
		}
		_, err := txDB.Exec(`
			UPDATE providers SET position = position + 1000
			WHERE type = (SELECT type FROM providers WHERE id = ?)`, ids[0])
		if err != nil {
			return err
		}
		for i, id := range ids {
			_, err := txDB.Exec(`UPDATE providers SET position = ? WHERE id = ?`, i, id)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
```

- [ ] **Step 6: Update Exists to also be type-aware (URL uniqueness stays global, so no change needed)**

The `Exists` method is fine as-is — it checks URL uniqueness across all types.

- [ ] **Step 7: Update providers_test.go**

Replace all `Create(url, name)` → `Create("hifi", url, name)` across the test file.
Replace all `ListOrdered()` → `ListByType("hifi")`.
Replace all `GetByPosition(pos)` → `GetByPosition("hifi", pos)`.

Add new test at end of file:
```go
func TestProvidersRepo_Types(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	repo := NewProvidersRepo(db)

	// Create hifi provider
	id1, err := repo.Create("hifi", "http://hifi.example.com", "Hifi One")
	assert.NoError(t, err)
	assert.Greater(t, id1, int64(0))

	// Create qobuz provider
	id2, err := repo.Create("qobuz", "http://qobuz.example.com", "Qobuz One")
	assert.NoError(t, err)
	assert.Greater(t, id2, int64(0))

	// ListByType("hifi") returns only hifi
	hifi, err := repo.ListByType("hifi")
	assert.NoError(t, err)
	assert.Len(t, hifi, 1)
	assert.Equal(t, "http://hifi.example.com", hifi[0].URL)

	// ListByType("qobuz") returns only qobuz
	qobuz, err := repo.ListByType("qobuz")
	assert.NoError(t, err)
	assert.Len(t, qobuz, 1)
	assert.Equal(t, "http://qobuz.example.com", qobuz[0].URL)

	// Positions are independent per type
	assert.Equal(t, 0, hifi[0].Position)
	assert.Equal(t, 0, qobuz[0].Position)

	// Create second hifi — position increments within type
	id3, err := repo.Create("hifi", "http://hifi2.example.com", "Hifi Two")
	assert.NoError(t, err)
	assert.Greater(t, id3, int64(0))

	hifiAll, err := repo.ListByType("hifi")
	assert.NoError(t, err)
	assert.Len(t, hifiAll, 2)
	assert.Equal(t, 1, hifiAll[1].Position)

	// Reorder within hifi type
	ids := []int64{hifiAll[1].ID, hifiAll[0].ID}
	err = repo.Reorder(ids)
	assert.NoError(t, err)

	// Verify hifi reordered
	hifiAfter, err := repo.ListByType("hifi")
	assert.NoError(t, err)
	assert.Equal(t, hifiAll[1].ID, hifiAfter[0].ID)
	assert.Equal(t, hifiAll[0].ID, hifiAfter[1].ID)

	// Verify qobuz positions unaffected
	qobuzAfter, err := repo.ListByType("qobuz")
	assert.NoError(t, err)
	assert.Len(t, qobuzAfter, 1)
	assert.Equal(t, 0, qobuzAfter[0].Position)
}
```

Note: the test file uses `assert` from `github.com/stretchr/testify/assert` — check imports and adjust if it doesn't.

- [ ] **Step 8: Run tests**
```
go test ./internal/store/ -v -run TestProvider
```
Expected: all provider tests pass.

- [ ] **Step 9: Commit**
```
git add internal/store/providers.go internal/store/providers_test.go
git commit -m "feat: add type-aware provider CRUD with per-type positions"
```

---

### Task 6: Make FallbackProvider type-aware

**Files:** Modify: `internal/catalog/fallback.go`

- [ ] **Step 1: Add providerType field**

```go
type FallbackProvider struct {
	manager         *ProviderManager
	providerType    ProviderType
	cachedProviders []Provider
	cacheExpiry     time.Time
	cacheMu         sync.Mutex
}
```

- [ ] **Step 2: Rewrite getProviders — remove system default, query by type**

Replace the entire `getProviders()` method (lines 22-47):

```go
func (f *FallbackProvider) getProviders() []Provider {
	f.cacheMu.Lock()
	defer f.cacheMu.Unlock()

	if f.cachedProviders != nil && time.Now().Before(f.cacheExpiry) {
		return f.cachedProviders
	}

	var providers []Provider

	if f.manager != nil && f.manager.providers != nil {
		storeProviders, _ := f.manager.providers.ListByType(string(f.providerType))
		for _, p := range storeProviders {
			providers = append(providers, NewProvider(f.providerType, p.URL))
		}
	}

	f.cachedProviders = providers
	f.cacheExpiry = time.Now().Add(providerCacheTTL)
	return providers
}
```

- [ ] **Step 3: Verify compiles**
```
go build ./internal/catalog/...
```
Expected: may fail until ProviderManager is updated (fallback.go references `f.manager.providers` which is type-checked at compile time). If it fails, proceed to Task 7 — the full build will pass after both are done.

- [ ] **Step 4: Commit**
```
git add internal/catalog/fallback.go
git commit -m "feat: make FallbackProvider type-aware, remove system default"
```

---

### Task 7: Rewrite ProviderManager

**Files:** Modify: `internal/catalog/manager.go`

- [ ] **Step 1: Replace entire file content**

```go
package catalog

import (
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/store"
)

type Logger interface {
	With(keyValues ...any) *slog.Logger
	Info(msg string, keyValues ...any)
	Error(msg string, keyValues ...any)
}

type ProviderManager struct {
	logger    Logger
	providers *store.ProvidersRepo
	settings  *store.SettingsRepo
	cacheTTL  time.Duration
	db        *store.DB

	chains map[ProviderType]*CachedProvider
	mu     sync.RWMutex
}

func NewProviderManager(db *store.DB, settings *store.SettingsRepo, cacheTTL time.Duration, logger Logger) *ProviderManager {
	var providersRepo *store.ProvidersRepo
	if db != nil {
		providersRepo = store.NewProvidersRepo(db)
	}

	return &ProviderManager{
		logger:    logger,
		providers: providersRepo,
		settings:  settings,
		cacheTTL:  cacheTTL,
		db:        db,
	}
}

func (m *ProviderManager) readSetting(key string) ProviderType {
	if m.settings == nil {
		return ProviderTypeHifi
	}
	val, err := m.settings.Get(key)
	if err != nil || val == "" {
		return ProviderTypeHifi
	}
	pt := ProviderType(val)
	if pt != ProviderTypeHifi && pt != ProviderTypeQobuz {
		return ProviderTypeHifi
	}
	return pt
}

func (m *ProviderManager) buildChain(pt ProviderType) *CachedProvider {
	fb := &FallbackProvider{manager: m, providerType: pt}
	var cacheStore *storeCache
	if m.db != nil {
		cacheStore = &storeCache{store: m.db}
	}
	return NewCachedProvider(fb, cacheStore, m.cacheTTL)
}

func (m *ProviderManager) GetProvider(pt ProviderType) Provider {
	m.mu.RLock()
	chain := m.chains[pt]
	m.mu.RUnlock()
	if chain != nil {
		return chain
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chains == nil {
		m.chains = make(map[ProviderType]*CachedProvider)
	}
	if m.chains[pt] == nil {
		m.chains[pt] = m.buildChain(pt)
	}
	return m.chains[pt]
}

func (m *ProviderManager) GetMetadataProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveMetadataProvider))
}

func (m *ProviderManager) GetDownloadProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveDownloadProvider))
}

func (m *ProviderManager) GetStreamingProvider() Provider {
	return m.GetProvider(m.readSetting(store.SettingActiveStreamingProvider))
}

func (m *ProviderManager) InvalidateAllCaches() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chains = nil
}
```

- [ ] **Step 2: Verify compiles**
```
go build ./...
```
Expected: fails due to callers still using `GetProvider()` without args. Proceed to fix callers in next tasks.

- [ ] **Step 3: Commit**
```
git add internal/catalog/manager.go
git commit -m "feat: rewrite ProviderManager with per-type chains and settings-based selection"
```

---

### Task 8: Remove ProviderURL from Config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Remove ProviderURL from Config struct**

Remove line 21 (`ProviderURL string`).

- [ ] **Step 2: Remove ProviderURL from Load()**

Remove line 51: `ProviderURL: getEnv("PROVIDER_URL", constants.DefaultProviderURL),`

- [ ] **Step 3: Remove ProviderURL validation**

Remove lines 99-106 (the ProviderURL validation block).

- [ ] **Step 4: Update config_test.go**

- Remove the `ProviderURL` assertion from `TestLoad` (lines 25-27):
```go
// DELETE these lines:
if cfg.ProviderURL != constants.DefaultProviderURL {
    t.Errorf("Expected ProviderURL to be %s, got %s", constants.DefaultProviderURL, cfg.ProviderURL)
}
```

- Remove `PROVIDER_URL` env handling from `TestLoadWithEnvVars`:
Remove lines 47-49 (Setenv), lines 60-62 (Unsetenv), lines 78-80 (assertion).

- In `TestValidate`, remove `ProviderURL` field from all test configs (9 occurrences).

- Remove the entire `TestValidateProviderURLs` function (lines 289-353).

- Remove the entire `TestLoadProviderURL` function (lines 355-374).

- [ ] **Step 5: Run config tests**
```
go test ./internal/config/ -v
```
Expected: all tests pass.

- [ ] **Step 6: Commit**
```
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: remove ProviderURL from config (providers now UI-managed)"
```

---

### Task 9: Update main.go

**Files:** Modify: `cmd/server/main.go`

- [ ] **Step 1: Replace ProviderManager construction and startup loading**

Replace lines 56-63:
```go
// Initialize Provider Manager
providerManager := catalog.NewProviderManager(cfg.ProviderURL, db, cfg.CacheTTL, appLogger)

// Load saved provider from settings if exists
settingsRepo := store.NewSettingsRepo(db)
if savedMetURL, err := settingsRepo.Get(store.SettingActiveMetadataProvider); err == nil && savedMetURL != "" {
    providerManager.SetProvider(savedMetURL)
}
```

With:
```go
// Initialize Settings Repo (needed by ProviderManager)
settingsRepo := store.NewSettingsRepo(db)

// Initialize Provider Manager (no system default — providers configured via UI)
providerManager := catalog.NewProviderManager(db, settingsRepo, cfg.CacheTTL, appLogger)
```

- [ ] **Step 2: Remove the duplicate `settingsRepo` declaration below**

Currently line 60 has `settingsRepo := store.NewSettingsRepo(db)` and line 73 has another `providersRepo := store.NewProvidersRepo(db)`. After the change above, `settingsRepo` is already defined. Remove the redundant `settingsRepo := store.NewSettingsRepo(db)` if it appears again. The current setup is:

Lines 56-63 (to be replaced):
```go
providerManager := catalog.NewProviderManager(cfg.ProviderURL, db, cfg.CacheTTL, appLogger)
settingsRepo := store.NewSettingsRepo(db)
if savedMetURL, err := settingsRepo.Get(store.SettingActiveMetadataProvider); err == nil && savedMetURL != "" {
    providerManager.SetProvider(savedMetURL)
}
```

Line 72-73 (keep but check):
```go
jobService := app.NewJobService(db, appLogger)
downloadsService := app.NewDownloadsService(db, appLogger)
providersRepo := store.NewProvidersRepo(db)
```

So the `settingsRepo` moves up to before `providerManager` creation, and the duplicate below is removed.

- [ ] **Step 3: Verify compiles**
```
go build ./cmd/server
```
Expected: fails due to callers still using old `GetProvider()`. Continue to next tasks.

- [ ] **Step 4: Commit**
```
git add cmd/server/main.go
git commit -m "feat: update main.go for new ProviderManager constructor"
```

---

### Task 10: Update app layer callers

**Files:**
- Modify: `internal/app/downloader.go`
- Modify: `internal/app/enricher.go`

- [ ] **Step 1: Update downloader.go line 35**

Replace:
```go
provider := d.providerManager.GetProvider()
```
With:
```go
provider := d.providerManager.GetDownloadProvider()
```

- [ ] **Step 2: Update enricher.go — three call sites**

Line 106 — FetchLyrics:
```go
// Before:
lyrics, subtitles, err := e.providerManager.GetProvider().GetLyrics(ctx, track.ProviderID)
// After:
lyrics, subtitles, err := e.providerManager.GetMetadataProvider().GetLyrics(ctx, track.ProviderID)
```

Line 125 — EnrichFromHiFi:
```go
// Before:
ct, err = e.providerManager.GetProvider().GetTrack(ctx, track.ProviderID)
// After:
ct, err = e.providerManager.GetMetadataProvider().GetTrack(ctx, track.ProviderID)
```

Line 149 — EnrichFromHiFi:
```go
// Before:
album, err = e.providerManager.GetProvider().GetAlbum(ctx, albumID)
// After:
album, err = e.providerManager.GetMetadataProvider().GetAlbum(ctx, albumID)
```

- [ ] **Step 3: Verify compiles**
```
go build ./internal/app/...
```
Expected: no errors.

- [ ] **Step 4: Commit**
```
git add internal/app/downloader.go internal/app/enricher.go
git commit -m "feat: use typed provider accessors in app layer (download/metadata)"
```

---

### Task 11: Update downloader handlers (worker)

**Files:** Modify: `internal/downloader/handlers.go`

- [ ] **Step 1: Update 4 call sites in handlers.go**

Line 439:
```go
// Before:
album, err := h.ProviderManager.GetProvider().GetAlbum(ctx, job.GetSourceID())
// After:
album, err := h.ProviderManager.GetMetadataProvider().GetAlbum(ctx, job.GetSourceID())
```

Line 470:
```go
// Before:
pl, err := h.ProviderManager.GetProvider().GetPlaylist(ctx, job.GetSourceID())
// After:
pl, err := h.ProviderManager.GetMetadataProvider().GetPlaylist(ctx, job.GetSourceID())
```

Line 530:
```go
// Before:
artist, err := h.ProviderManager.GetProvider().GetArtist(ctx, job.GetSourceID())
// After:
artist, err := h.ProviderManager.GetMetadataProvider().GetArtist(ctx, job.GetSourceID())
```

Line 563:
```go
// Before:
artist, err := h.ProviderManager.GetProvider().GetArtist(ctx, job.GetSourceID())
// After:
artist, err := h.ProviderManager.GetMetadataProvider().GetArtist(ctx, job.GetSourceID())
```

- [ ] **Step 2: Verify compiles**
```
go build ./internal/downloader/...
```
Expected: no errors.

- [ ] **Step 3: Commit**
```
git add internal/downloader/handlers.go
git commit -m "feat: use GetMetadataProvider in downloader worker handlers"
```

---

### Task 12: Update HTTP handler callers

**Files:**
- Modify: `internal/http/routes.go`
- Modify: `internal/http/stream.go`

- [ ] **Step 1: Update stream.go line 25**

```go
// Before:
provider := h.ProviderManager.GetProvider()
// After:
provider := h.ProviderManager.GetStreamingProvider()
```

- [ ] **Step 2: Update routes.go — all GetProvider() call sites**

Line 39 (SearchHTMX):
```go
// Before: provider := h.ProviderManager.GetProvider()
// After:
provider := h.ProviderManager.GetMetadataProvider()
```

Line 86 (LuckyHTMX — recommendations):
```go
// Before: provider := h.ProviderManager.GetProvider()
// After:
provider := h.ProviderManager.GetMetadataProvider()
```

Line 164 (Artist page):
```go
// Before: artist, err := h.ProviderManager.GetProvider().GetArtist(r.Context(), id)
// After:
artist, err := h.ProviderManager.GetMetadataProvider().GetArtist(r.Context(), id)
```

Line 182 (Album page):
```go
// Before: album, err := h.ProviderManager.GetProvider().GetAlbum(r.Context(), id)
// After:
album, err := h.ProviderManager.GetMetadataProvider().GetAlbum(r.Context(), id)
```

Line 196 (Playlist page):
```go
// Before: pl, err := h.ProviderManager.GetProvider().GetPlaylist(r.Context(), id)
// After:
pl, err := h.ProviderManager.GetMetadataProvider().GetPlaylist(r.Context(), id)
```

Line 639 (SimilarAlbumsHTMX):
```go
// Before: albums, err := h.ProviderManager.GetProvider().GetSimilarAlbums(r.Context(), id)
// After:
albums, err := h.ProviderManager.GetMetadataProvider().GetSimilarAlbums(r.Context(), id)
```

Line 650 (SimilarArtistsHTMX):
```go
// Before: artists, err := h.ProviderManager.GetProvider().GetSimilarArtists(r.Context(), id)
// After:
artists, err := h.ProviderManager.GetMetadataProvider().GetSimilarArtists(r.Context(), id)
```

- [ ] **Step 3: Verify compiles**
```
go build ./...
```
Expected: may fail because SettingsPage still references `h.Config.ProviderURL`. Continue to next task.

- [ ] **Step 4: Commit**
```
git add internal/http/routes.go internal/http/stream.go
git commit -m "feat: route provider calls through typed accessors (metadata/streaming)"
```

---

### Task 13: Update provider HTMX endpoints

**Files:** Modify: `internal/http/routes.go`

- [ ] **Step 1: Rewrite GetProvidersHTMX to return typed data**

Replace lines 312-331:

```go
func (h *Handler) GetProvidersHTMX(w http.ResponseWriter, r *http.Request) {
	hifiProviders, err := h.ProvidersRepo.ListByType("hifi")
	if err != nil {
		h.Logger.Error("Failed to list hifi providers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	qobuzProviders, err := h.ProvidersRepo.ListByType("qobuz")
	if err != nil {
		h.Logger.Error("Failed to list qobuz providers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"hifi":  hifiProviders,
		"qobuz": qobuzProviders,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Error("Failed to encode providers response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
```

- [ ] **Step 2: Update ReorderProvidersHTMX to accept type param**

Replace lines 333-358:

```go
func (h *Handler) ReorderProvidersHTMX(w http.ResponseWriter, r *http.Request) {
	providerType := r.URL.Query().Get("type")
	if providerType == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	ids := r.Form["ids[]"]
	intIDs := make([]int64, 0, len(ids))
	for _, idStr := range ids {
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			continue
		}
		intIDs = append(intIDs, id)
	}

	if err := h.ProvidersRepo.Reorder(intIDs); err != nil {
		h.Logger.Error("Failed to reorder providers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.ProviderManager.InvalidateAllCaches()

	_, _ = w.Write([]byte(`{"success":true}`))
}
```

- [ ] **Step 3: Update AddProviderHTMX to accept type param**

Replace lines 360-378:

```go
func (h *Handler) AddProviderHTMX(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	url := r.URL.Query().Get("url")
	providerType := r.URL.Query().Get("type")
	if name == "" || url == "" || providerType == "" {
		http.Error(w, "name, url, and type are required", http.StatusBadRequest)
		return
	}

	id, err := h.ProvidersRepo.Create(providerType, url, name)
	if err != nil || id == 0 {
		h.Logger.Error("Failed to create provider", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.ProviderManager.InvalidateAllCaches()

	_, _ = w.Write([]byte(`{"success":true}`))
}
```

- [ ] **Step 4: Update RemoveProviderHTMX to invalidate all caches**

Replace line 399:
```go
// Before: h.ProviderManager.InvalidateProviderCache()
// After:
h.ProviderManager.InvalidateAllCaches()
```

Also update line 355 and 375 (in the new code above, already using `InvalidateAllCaches`).

- [ ] **Step 5: Update SettingsPage to remove DefaultURL**

Replace lines 222-227:

```go
func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, "settings.html", map[string]interface{}{
		"ActivePage": "settings",
	})
}
```

- [ ] **Step 6: Add new settings endpoints for default API types**

Add after the RemoveProviderHTMX function (~line 402):

```go
func (h *Handler) GetDefaultAPIsHTMX(w http.ResponseWriter, r *http.Request) {
	keys := []string{"active_metadata_provider", "active_download_provider", "active_streaming_provider"}
	defaults := map[string]string{
		"active_metadata_provider":  "hifi",
		"active_download_provider":  "hifi",
		"active_streaming_provider": "hifi",
	}

	response := make(map[string]string)
	for _, key := range keys {
		val, err := h.SettingsRepo.Get(key)
		if err != nil || val == "" {
			val = defaults[key]
		}
		response[key] = val
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Error("Failed to encode default APIs response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) SetDefaultAPIHTMX(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	validKeys := map[string]bool{
		"active_metadata_provider":  true,
		"active_download_provider":  true,
		"active_streaming_provider": true,
	}
	if !validKeys[body.Key] {
		http.Error(w, "Invalid key", http.StatusBadRequest)
		return
	}

	if body.Value != "hifi" && body.Value != "qobuz" {
		http.Error(w, "Value must be 'hifi' or 'qobuz'", http.StatusBadRequest)
		return
	}

	if err := h.SettingsRepo.Set(body.Key, body.Value); err != nil {
		h.Logger.Error("Failed to save default API setting", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.ProviderManager.InvalidateAllCaches()

	_, _ = w.Write([]byte(`{"success":true}`))
}
```

- [ ] **Step 7: Register new routes in handler.go**

In `handler.go` RegisterRoutes, near the provider routes (after line 95), add:

```go
r.Get("/htmx/default-apis", h.GetDefaultAPIsHTMX)
r.Post("/htmx/default-apis", h.SetDefaultAPIHTMX)
```

- [ ] **Step 8: Verify compiles**
```
go build ./...
```
Expected: no errors.

- [ ] **Step 9: Commit**
```
git add internal/http/routes.go internal/http/handler.go
git commit -m "feat: update provider HTMX endpoints with type support, add default API settings endpoints"
```

---

### Task 14: Update settings.html template

**Files:** Modify: `web/templates/settings.html`

- [ ] **Step 1: Replace the Providers section (lines 4-12) and add new sections**

Replace the entire providers section (lines 4-12):

```html
<!-- DELETE old section -->
```

Replace the script at the bottom — completely replace the provider-related JS and add default API JS. The new file:

Actually, let me write the full new content. The template will have:

1. HIFI Providers section
2. QOBUZ Providers section  
3. Default API Selectors section
4. Updated JS

The full replacement for the old providers section (lines 4-12) plus the new sections after it:

```html
<div class="section">
    <h2>HIFI Providers</h2>
    <p class="hint">HIFI API URLs used for metadata, downloads, and streaming.</p>
    <div id="hifi-provider-list" class="flex flex-col gap-2"></div>
    <form class="provider-form mt-4" onsubmit="addProvider(event, 'hifi')">
        <input type="text" id="hifi-provider-name" placeholder="Name" required>
        <input type="url" id="hifi-provider-url" placeholder="URL" required>
        <button type="submit" class="btn-lg btn-primary">+ Add</button>
    </form>
</div>

<div class="section">
    <h2>QOBUZ Providers</h2>
    <p class="hint">QOBUZ API URLs (provider implementation coming soon).</p>
    <div id="qobuz-provider-list" class="flex flex-col gap-2"></div>
    <form class="provider-form mt-4" onsubmit="addProvider(event, 'qobuz')">
        <input type="text" id="qobuz-provider-name" placeholder="Name" required>
        <input type="url" id="qobuz-provider-url" placeholder="URL" required>
        <button type="submit" class="btn-lg btn-primary">+ Add</button>
    </form>
</div>

<div class="section">
    <h2>Default APIs</h2>
    <p class="hint">Select which provider type to use for each operation.</p>
    <div class="flex flex-col gap-3 mt-2">
        <div class="flex gap-2 items-center">
            <label class="w-52">Metadata (search/browse):</label>
            <select id="default-metadata-api" class="form-select" onchange="saveDefaultAPI('active_metadata_provider', this.value)">
                <option value="hifi">HIFI</option>
                <option value="qobuz">QOBUZ</option>
            </select>
        </div>
        <div class="flex gap-2 items-center">
            <label class="w-52">Download:</label>
            <select id="default-download-api" class="form-select" onchange="saveDefaultAPI('active_download_provider', this.value)">
                <option value="hifi">HIFI</option>
                <option value="qobuz">QOBUZ</option>
            </select>
        </div>
        <div class="flex gap-2 items-center">
            <label class="w-52">Streaming (playback):</label>
            <select id="default-streaming-api" class="form-select" onchange="saveDefaultAPI('active_streaming_provider', this.value)">
                <option value="hifi">HIFI</option>
                <option value="qobuz">QOBUZ</option>
            </select>
        </div>
    </div>
    <div id="default-api-status" class="mt-2"></div>
</div>
```

- [ ] **Step 2: Replace all provider-related JavaScript**

Replace the `loadProviders` through `removeProvider` functions and the initial `loadProviders()` call:

```javascript
function loadProviders(type) {
    fetch('/htmx/providers', { cache: 'no-store' })
        .then(r => r.json())
        .then(data => {
            const providers = data[type] || [];
            renderProviderList(type, providers);
        });
}

function renderProviderList(type, providers) {
    const containerId = type + '-provider-list';
    const container = document.getElementById(containerId);
    if (!container) return;

    if (providers.length === 0) {
        container.innerHTML = '<p class="hint">No ' + type.toUpperCase() + ' providers configured.</p>';
        return;
    }

    container.innerHTML = providers.map((p, i) => `
        <div class="item item-bordered">
            <span class="badge-env flex-shrink-0">${i + 1}</span>
            <div class="item-body min-w-0 flex-1">
                <div class="item-title truncate font-medium" title="${p.name}">${p.name}</div>
                <div class="item-subtitle truncate text-dim" title="${p.url}"><a href="${p.url}" target="_blank" rel="noopener noreferrer" class="text-dim">${p.url}</a></div>
            </div>
            <div class="item-actions">
                <button class="btn btn-sm btn-outline" onclick="moveProvider(${p.id}, 'up', '${type}')" ${i === 0 ? 'disabled' : ''} title="Move Up">
                    <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"></line><polyline points="5 12 12 5 19 12"></polyline></svg>
                </button>
                <button class="btn btn-sm btn-outline" onclick="moveProvider(${p.id}, 'down', '${type}')" ${i === providers.length - 1 ? 'disabled' : ''} title="Move Down">
                    <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><polyline points="19 12 12 19 5 12"></polyline></svg>
                </button>
                <button class="btn btn-sm btn-outline-danger" onclick="removeProvider(${p.id}, '${type}')" title="Remove">
                    <svg class="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><line x1="10" y1="11" x2="10" y2="17"></line><line x1="14" y1="11" x2="14" y2="17"></line></svg>
                </button>
            </div>
        </div>
    `).join('');
}

function moveProvider(id, direction, type) {
    fetch('/htmx/providers?type=' + type, { cache: 'no-store' })
        .then(r => r.json())
        .then(data => {
            const providers = data[type] || [];
            const currentIdx = providers.findIndex(p => p.id === id);
            if (currentIdx === -1) return;

            const newIdx = direction === 'up' ? currentIdx - 1 : currentIdx + 1;
            if (newIdx < 0 || newIdx >= providers.length) return;

            const newOrder = [...providers];
            [newOrder[currentIdx], newOrder[newIdx]] = [newOrder[newIdx], newOrder[currentIdx]];

            const ids = newOrder.map(p => p.id);

            fetch('/htmx/providers/reorder?type=' + type, {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: 'ids[]=' + ids.join('&ids[]=')
            }).then(() => loadProviders(type));
        });
}

function addProvider(e, type) {
    e.preventDefault();
    const nameEl = document.getElementById(type + '-provider-name');
    const urlEl = document.getElementById(type + '-provider-url');
    const name = nameEl.value;
    const url = urlEl.value;
    fetch('/htmx/provider?name=' + encodeURIComponent(name) + '&url=' + encodeURIComponent(url) + '&type=' + type, { method: 'POST' })
        .then(r => r.json())
        .then(() => {
            nameEl.value = '';
            urlEl.value = '';
            loadProviders(type);
        });
}

function removeProvider(id, type) {
    if (!confirm('Remove this provider?')) return;
    fetch('/htmx/provider?id=' + id, { method: 'DELETE' })
        .then(r => r.json())
        .then(() => loadProviders(type));
}

function loadDefaultAPIs() {
    fetch('/htmx/default-apis', { cache: 'no-store' })
        .then(r => r.json())
        .then(data => {
            document.getElementById('default-metadata-api').value = data.active_metadata_provider || 'hifi';
            document.getElementById('default-download-api').value = data.active_download_provider || 'hifi';
            document.getElementById('default-streaming-api').value = data.active_streaming_provider || 'hifi';
        });
}

function saveDefaultAPI(key, value) {
    fetch('/htmx/default-apis', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ key: key, value: value })
    })
        .then(r => r.json())
        .then(data => {
            if (data.success) {
                const statusDiv = document.getElementById('default-api-status');
                statusDiv.innerHTML = '<span class="badge badge-success">Saved</span>';
                setTimeout(() => statusDiv.innerHTML = '', 2000);
            }
        });
}
```

- [ ] **Step 3: Update the onload calls at the bottom**

Replace:
```javascript
loadProviders();
```

With:
```javascript
loadProviders('hifi');
loadProviders('qobuz');
loadDefaultAPIs();
```

- [ ] **Step 4: Verify HTML is valid (no template syntax errors)**

Check that no `{{.DefaultURL}}` references remain in the file.

- [ ] **Step 5: Commit**
```
git add web/templates/settings.html
git commit -m "feat: multi-section provider UI with per-type lists and default API selectors"
```

---

### Task 15: Cleanup unused code

**Files:** Remove: `internal/http/dto/provider.go`

- [ ] **Step 1: Delete unused DTO file**

`internal/http/dto/provider.go` references `catalog.CustomProvider` and `catalog.ProviderSettings` which no longer exist. Delete the file.

- [ ] **Step 2: Verify compiles**
```
go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**
```
git add internal/http/dto/provider.go
git commit -m "chore: remove unused provider DTO (replaced by typed endpoints)"
```

---

### Task 16: Full build and test

- [ ] **Step 1: Build the entire project**
```
go build ./...
```
Expected: no errors.

- [ ] **Step 2: Run all tests**
```
go test ./...
```
Expected: all tests pass.

- [ ] **Step 3: Run lint**
```
golangci-lint run
```
Expected: no new issues. Fix any lint errors.

- [ ] **Step 4: Final commit (if any fixes needed)**
```
git add -A
git commit -m "fix: address build/lint issues from multi-provider type support"
```
