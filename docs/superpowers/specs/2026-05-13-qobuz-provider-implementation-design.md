# QobuzProvider Implementation

**Date:** 2026-05-13

## Goal

Implement the `QobuzProvider` to connect to a Qobuz-compatible proxy server via HTTP, providing metadata, search, and streaming for the navidrums download orchestrator. The provider follows the same architectural pattern as the existing `HifiProvider`.

---

## Architecture

### New Files

| File | Purpose |
|------|---------|
| `internal/catalog/qobuz.go` | Provider struct, constructor, all 10 interface methods, HTTP helpers |
| `internal/catalog/qobuz_dto.go` | Qobuz API response structs for JSON deserialization |
| `internal/catalog/qobuz_convert.go` | `ToDomain()` methods mapping DTOs to domain types |

**Existing file modified:** `internal/catalog/qobuz.go` (the stub) is fully replaced.

### Alignment with Existing Code

The structure mirrors `HifiProvider`:
- `hifi.go` → `qobuz.go` — provider implementation
- `dto.go` → `qobuz_dto.go` — API response types
- `convert.go` → `qobuz_convert.go` — domain converters

---

## API Contract

Based on `QOBUZ_API.md` and `api-examples/qobuz-api/`.

### Base URL

Provided via constructor: `NewQobuzProvider(baseURL)`. The proxy handles Qobuz authentication.

### Endpoints

| Method | Endpoint | Query Params |
|--------|----------|--------------|
| `Search` | `GET /get-music` | `q` (string), `offset` (int, default 0) |
| `GetArtist` | `GET /get-artist` | `artist_id` (int) |
| `GetAlbum` | `GET /get-album` | `album_id` (string, UUID) |
| `GetTrack` | `GET /get-track` | `isrc` (int, misnamed — takes track ID) |
| `GetStream` | `GET /download-music` | `track_id` (int), `quality` (int) → then GET signed URL |

### Unsupported Operations

| Method | Behavior |
|--------|----------|
| `GetPlaylist` | Returns `ErrQobuzNotSupported` |
| `GetSimilarAlbums` | Returns `ErrQobuzNotSupported` |
| `GetLyrics` | Returns `ErrQobuzNotSupported` |
| `GetRecommendations` | Returns `ErrQobuzNotSupported` |

---

## Provider Struct

```go
type QobuzProvider struct {
    client  *httpclient.Client
    BaseURL string
}
```

### Constructor

```go
func NewQobuzProvider(baseURL string) *QobuzProvider {
    return &QobuzProvider{
        BaseURL: baseURL,
        client: httpclient.NewClient(&http.Client{
            Timeout: 20 * time.Second,
        }, 500*time.Millisecond),
    }
}
```

Matches HifiProvider pattern:
- 20s HTTP timeout
- 500ms rate limit between requests

### HTTP Helpers

```go
func (p *QobuzProvider) get(ctx context.Context, url string, target interface{}) error
func (p *QobuzProvider) setHeaders(req *http.Request)
func qobuzQualityCode(quality string) int  // navidrums constant → Qobuz numeric code
```

---

## DTOs — `qobuz_dto.go`

### Response Wrappers

```go
type QobuzSearchResponse  struct { Success bool; Data QobuzSearchData }
type QobuzArtistResponse  struct { Success bool; Data QobuzArtistData }
type QobuzAlbumResponse   struct { /* album fields */ }
type QobuzTrackResponse   struct { /* track fields */ }
type QobuzDownloadResponse struct { URL string }
```

### Shared Types

```go
type QobuzImage      struct { Small, Thumbnail, Large string }
type QobuzLabel      struct { Name string; ID int }
type QobuzGenre      struct { Name string; ID int; Path []int }
type QobuzComposer  struct { Name string; ID int }
type QobuzArtistRef struct { Name string; ID int; Slug string; Picture *string }
type QobuzArtistBrief struct { Name string; ID int }
```

### Album Track Shape

```go
type QobuzTrackItem struct {
    ID                int
    Title             string
    TrackNumber       int
    MediaNumber       int
    Duration          int
    ISRC              string
    Copyright         string
    Performers        string
    Version           *string
    MaximumBitDepth        int
    MaximumSamplingRate    float64
    MaximumChannelCount   int
    Hires, HiresStreamable  bool
    ParentalWarning  bool
    AudioInfo        *QobuzReplayGain
    Performer         QobuzPerformer
    Composer          *QobuzComposer
    Album             *QobuzTrackAlbum  // embedded in search/get-track
}

type QobuzReplayGain struct {
    ReplayGainTrackPeak float64 `json:"replaygain_track_peak"`
    ReplayGainTrackGain float64 `json:"replaygain_track_gain"`
}

type QobuzPerformer struct { Name string; ID int }

type QobuzTrackAlbum struct {
    ID                string
    Title             string
    QobuzID           int `json:"qobuz_id"`
    Artist            QobuzArtistRef
    Genre             QobuzGenre
    Image             QobuzImage
    MaximumBitDepth  int
    MaximumSamplingRate float64
}
```

### Album Response

```go
type QobuzAlbumResponse struct {
    ID                     string
    Title                  string
    QobuzID                int
    UPC                    string
    TracksCount            int
    MediaCount             int
    ParentalWarning        bool
    Copyright              string
    ReleaseDateOriginal    string `json:"release_date_original"`
    Image                  QobuzImage
    Artist                 QobuzArtistRef
    Artists                []QobuzAlbumArtist
    Label                  QobuzLabel
    Genre                  QobuzGenre
    Tracks                 QobuzTracksContainer
}

type QobuzAlbumArtist struct {
    ID   int
    Name string
    Roles []string
}

type QobuzTracksContainer struct {
    Total  int
    Offset int
    Limit  int
    Items  []QobuzTrackItem
}
```

### Artist Response

```go
type QobuzArtistData struct {
    Artist QobuzArtistFull `json:"artist"`
}

type QobuzArtistFull struct {
    ID           int
    Name         QobuzNameObject `json:"name"`
    Biography    *QobuzBiography
    Images       QobuzArtistImages
    Albums       QobuzArtistAlbums
    TopTracks    []QobuzTopTrackItem
    SimilarArtists QobuzSimilarArtists
}

type QobuzNameObject struct { Display string }

type QobuzBiography struct {
    Content  string
    Language string
}

type QobuzArtistImages struct {
    Portrait *QobuzImageHash
}

type QobuzImageHash struct {
    Hash   string
    Format string
}

type QobuzArtistAlbums struct {
    Items []QobuzArtistAlbumItem
}

type QobuzArtistAlbumItem struct {
    ID     string
    Title  string
    Image  QobuzImage
    Genre  QobuzGenre
    ReleaseDateOriginal string
    MaximumBitDepth    int
}

type QobuzTopTrackItem struct {
    ID              int
    Title           string
    Duration        int
    ISRC            string
    ParentalWarning bool
    Composer        QobuzNameObject
    Artist          QobuzNameObject
    PhysicalSupport QobuzPhysicalSupport
    Rights          QobuzTrackRights
    AudioInfo       QobuzTechAudioInfo
    Album           *QobuzTopTrackAlbum
}

type QobuzPhysicalSupport struct {
    MediaNumber   int `json:"media_number"`
    TrackNumber   int `json:"track_number"`
}

type QobuzTrackRights struct {
    Streamable       bool
    HiresStreamable  bool
    Purchasable      bool
    Downloadable     bool
    Previewable      bool
}

type QobuzTechAudioInfo struct {
    MaximumBitDepth      int `json:"maximum_bit_depth"`
    MaximumChannelCount int `json:"maximum_channel_count"`
    MaximumSamplingRate float64 `json:"maximum_sampling_rate"`
}

type QobuzTopTrackAlbum struct {
    ID     string
    Title  string
    Image  QobuzImage
    Label  QobuzLabel
    Genre  QobuzGenre
}

type QobuzSimilarArtists struct {
    Items []QobuzSimilarArtistItem
}

type QobuzSimilarArtistItem struct {
    ID   int
    Name QobuzNameObject
    Images QobuzArtistImages
}
```

### Search Response

```go
type QobuzSearchData struct {
    Query     string
    Albums    QobuzSearchAlbums
    Tracks    QobuzSearchTracks
    Artists   QobuzSearchArtists
    Playlists QobuzSearchPlaylists
}

type QobuzSearchAlbums struct {
    Total  int
    Items  []QobuzSearchAlbumItem
}

type QobuzSearchAlbumItem struct {
    ID                  string
    Title               string
    QobuzID             int
    Image               QobuzImage
    Artist              QobuzArtistRef
    Genre               QobuzGenre
    ReleaseDateOriginal string
    MaximumBitDepth     int
}

type QobuzSearchTracks struct {
    Total  int
    Items  []QobuzTrackItem  // same as album tracks
}

type QobuzSearchArtists struct {
    Total  int
    Items  []QobuzSearchArtistItem
}

type QobuzSearchArtistItem struct {
    ID    int
    Name  string
    Image *QobuzImageHash
}

type QobuzSearchPlaylists struct {
    Total  int
    Items  []QobuzSearchPlaylistItem
}

type QobuzSearchPlaylistItem struct {
    ID          int64
    Title       string
    Description string
    Image       QobuzImage
}
```

---

## Converters — `qobuz_convert.go`

### Quality Resolution

```go
func resolveAudioQuality(track *QobuzTrackItem) string {
    if track.Hires && track.MaximumBitDepth >= 24 {
        return "HI_RES_LOSSLESS"
    }
    if track.MaximumBitDepth >= 16 {
        return "LOSSLESS"
    }
    return "LOW"
}
```

### Error Definitions

```go
var (
    ErrQobuzNotSupported = errors.New("qobuz provider does not support this operation")
)
```

### ToDomain Methods

| Method | Behavior |
|--------|----------|
| `QobuzSearchResponse.ToDomain(p)` | Returns `domain.SearchResult` with albums, tracks, artists, playlists |
| `QobuzAlbumResponse.ToDomain(p)` | Returns `domain.Album` with title, artist, label, genre, UPC, image, year, tracks |
| `QobuzTrackItem.ToDomain(p)` | Returns `domain.CatalogTrack` with all metadata |
| `QobuzArtistData.ToDomain(p)` | Returns `domain.Artist` with name, bio, image, albums, top tracks, similar artists |
| `QobuzSearchAlbumItem.ToDomain(p)` | Returns `domain.Album` (summary form for search results) |
| `QobuzSearchArtistItem.ToDomain(p)` | Returns `domain.Artist` (summary form for search results) |
| `QobuzSearchPlaylistItem.ToDomain(p)` | Returns `domain.Playlist` (search result form) |
| `QobuzTopTrackItem.ToDomain(p)` | Returns `domain.CatalogTrack` (top tracks shape) |

### Field Mapping Notes

- Image URLs: Qobuz returns full `static.qobuz.com` URLs — no construction needed
- IDs: Qobuz uses `int` for track/artist IDs → `strconv.Itoa()` for domain string fields
- Album IDs: already strings (UUIDs) — use directly
- Year: parse from `release_date_original` (YYYY-MM-DD format)
- ReplayGain: same field names as HifiProvider (`replaygain_track_gain`, `replaygain_track_peak`)
- Artists: extract from `Performer.Name`, `Album.Artist.Name`, `Artists` array

---

## Method Implementations — `qobuz.go`

### Search

```go
func (p *QobuzProvider) Search(ctx context.Context, query, searchType string) (*domain.SearchResult, error) {
    url := fmt.Sprintf("%s/get-music?q=%s&offset=0", p.BaseURL, url.QueryEscape(query))
    var resp QobuzSearchResponse
    if err := p.get(ctx, url, &resp); err != nil {
        return nil, fmt.Errorf("qobuz search failed: %w", err)
    }

    result := resp.Data.ToDomain(p)

    // filter by searchType
    if searchType != "" && searchType != "all" {
        switch searchType {
        case "artist":
            result.Albums = nil
            result.Tracks = nil
            result.Playlists = nil
        case "album":
            result.Artists = nil
            result.Tracks = nil
            result.Playlists = nil
        case "track":
            result.Albums = nil
            result.Artists = nil
            result.Playlists = nil
        case "playlist":
            result.Albums = nil
            result.Artists = nil
            result.Tracks = nil
        }
    }
    return result, nil
}
```

### GetArtist

```go
func (p *QobuzProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
    artistID, err := strconv.Atoi(id)
    if err != nil {
        return nil, fmt.Errorf("invalid artist id: %w", err)
    }
    url := fmt.Sprintf("%s/get-artist?artist_id=%d", p.BaseURL, artistID)
    var resp QobuzArtistResponse
    if err := p.get(ctx, url, &resp); err != nil {
        return nil, fmt.Errorf("qobuz get artist failed: %w", err)
    }
    return resp.Data.ToDomain(p), nil
}
```

### GetAlbum

```go
func (p *QobuzProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
    url := fmt.Sprintf("%s/get-album?album_id=%s", p.BaseURL, url.PathEscape(id))
    var resp QobuzAlbumResponse
    if err := p.get(ctx, url, &resp); err != nil {
        return nil, fmt.Errorf("qobuz get album failed: %w", err)
    }
    return resp.ToDomain(p), nil
}
```

### GetTrack

```go
func (p *QobuzProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
    trackID, err := strconv.Atoi(id)
    if err != nil {
        return nil, fmt.Errorf("invalid track id: %w", err)
    }
    url := fmt.Sprintf("%s/get-track?isrc=%d", p.BaseURL, trackID)
    var resp QobuzTrackResponse  // same shape as QobuzTrackItem
    if err := p.get(ctx, url, &resp); err != nil {
        return nil, fmt.Errorf("qobuz get track failed: %w", err)
    }
    return resp.ToDomain(p), nil
}
```

### GetStream — Two-Step Process

```go
func (p *QobuzProvider) GetStream(ctx context.Context, trackID, quality string) (io.ReadCloser, string, error) {
    tid, err := strconv.Atoi(trackID)
    if err != nil {
        return nil, "", fmt.Errorf("invalid track id: %w", err)
    }
    q := qobuzQualityCode(quality)

    // Step 1: Get signed URL
    url := fmt.Sprintf("%s/download-music?track_id=%d&quality=%d", p.BaseURL, tid, q)
    var downloadResp QobuzDownloadResponse
    if err := p.get(ctx, url, &downloadResp); err != nil {
        return nil, "", fmt.Errorf("qobuz get stream failed: %w", err)
    }

    // Step 2: Fetch the actual audio stream
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadResp.URL, nil)
    if err != nil {
        return nil, "", fmt.Errorf("failed to create stream request: %w", err)
    }

    resp, err := p.client.GetUnderlyingClient().Do(req)
    if err != nil {
        return nil, "", fmt.Errorf("failed to fetch stream: %w", err)
    }

    // Determine content type
    mime := resp.Header.Get("Content-Type")
    if mime == "" {
        mime = "audio/flac"
    }

    return resp.Body, mime, nil
}
```

### GetSimilarArtists

```go
func (p *QobuzProvider) GetSimilarArtists(ctx context.Context, id string) ([]domain.Artist, error) {
    artist, err := p.GetArtist(ctx, id)
    if err != nil {
        return nil, err
    }
    return artist.SimilarArtists, nil
}
```

### Unsupported Methods

```go
func (p *QobuzProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
    return nil, ErrQobuzNotSupported
}

func (p *QobuzProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
    return nil, ErrQobuzNotSupported
}

func (p *QobuzProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
    return "", "", ErrQobuzNotSupported
}

func (p *QobuzProvider) GetRecommendations(ctx context.Context, id string) ([]domain.CatalogTrack, error) {
    return nil, ErrQobuzNotSupported
}
```

---

## Interface Compliance

```go
var _ catalog.Provider = (*QobuzProvider)(nil)
```

Compile-time check that all 10 methods are implemented.

---

## Quality Code Mapping

```go
func qobuzQualityCode(quality string) int {
    switch quality {
    case "HI_RES_LOSSLESS":
        return 27  // tentative, undocumented
    case "LOSSLESS":
        return 6
    case "HIGH":
        return 5  // 320kbps MP3 fallback
    case "LOW":
        return 1  // 128kbps MP3
    default:
        return 6  // default to lossless
    }
}
```

---

## Error Handling

| Error Type | Source | Message |
|------------|--------|---------|
| Network failure | `p.get()` | `"qobuz search/get artist/get album/get track failed: <underlying>"` |
| Invalid ID | `strconv.Atoi()` | `"invalid artist id: <underlying>"` or `"invalid track id: <underlying>"` |
| Stream fetch | `client.Do()` | `"failed to fetch stream: <underlying>"` |
| Unsupported | Direct return | `"qobuz provider does not support this operation"` |

Errors returned are wrapped with `fmt.Errorf` using `%w` for context, following project conventions.

---

## Rate Limiting

Uses `httpclient.Client` with 500ms minimum interval between requests, same as HifiProvider. The signed streaming URL fetch does not go through the rate-limited client — direct HTTP to CDN.

---

## Caching

Provider methods are wrapped by `CachedProvider` in the `ProviderManager` chain. The `CachedProvider` implementation is unchanged — QobuzProvider benefits from automatic caching of:
- Search results
- Artist metadata
- Album metadata
- Track metadata
- Similar artists

`GetStream` and `GetLyrics` are not cached (pass-through).

---

## Testing Strategy

1. **Unit tests for converters** — table-driven, using sample JSON from `api-examples/qobuz-api/`
2. **Provider method tests** — use `httptest` server with recorded responses
3. **Mock provider** — extend `MockProvider` if needed for integration tests

Test files:
- `qobuz_convert_test.go` — converter tests
- `qobuz_test.go` — method tests (if applicable)

---

## Edge Cases

| Case | Handling |
|------|----------|
| Empty search results | Return `SearchResult` with empty slices (not nil) |
| Album without tracks | `tracks.items` is empty but not nil — handle gracefully |
| Track without ISRC | ISRC field is empty string |
| Artist without image | `Images.Portrait` is nil — use placeholder |
| Stream URL expired | Return error from CDN fetch — caller can retry |
| Invalid quality parameter | Default to 6 (LOSSLESS) |

---

## Files Summary

| File | Lines (est) | Description |
|------|-------------|-------------|
| `internal/catalog/qobuz.go` | ~200 | Provider implementation |
| `internal/catalog/qobuz_dto.go` | ~300 | API response types |
| `internal/catalog/qobuz_convert.go` | ~350 | ToDomain converters |

Total: ~850 lines, similar to HifiProvider's ~900 lines across 4 files.

---

## Next Steps

1. Create `internal/catalog/qobuz_dto.go` — define all DTOs
2. Create `internal/catalog/qobuz_convert.go` — implement converters
3. Replace `internal/catalog/qobuz.go` — implement all methods
4. Add tests in `qobuz_convert_test.go` and optionally `qobuz_test.go`
5. Run `go build` and `go test ./internal/catalog/...`
6. Run `golangci-lint run`