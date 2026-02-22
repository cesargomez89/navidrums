# API

## UI Routes (HTMX)

All routes are server-rendered HTML endpoints using HTMX for partial updates.

### Pages

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/` | Search page (root) |
| GET | `/artist/{id}` | Artist detail page |
| GET | `/album/{id}` | Album detail page |
| GET | `/playlist/{id}` | Playlist detail page |
| GET | `/queue` | Download queue page |
| GET | `/downloads` | Downloads browser page |
| GET | `/settings` | Settings page |

### HTMX Fragments

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/htmx/search?q={query}&type={type}` | Search results fragment |
| GET | `/htmx/album/{id}/similar` | Similar albums fragment |
| POST | `/htmx/download/{type}/{id}` | Enqueue download job |
| GET | `/htmx/queue/active` | Active jobs fragment |
| GET | `/htmx/queue/history` | Job history fragment |
| POST | `/htmx/cancel/{id}` | Cancel a job |
| POST | `/htmx/retry/{id}` | Retry a failed job |
| POST | `/htmx/history/clear` | Clear finished jobs |
| GET | `/htmx/downloads?q={query}` | Downloads browser fragment |
| POST | `/htmx/downloads/sync` | Sync all completed tracks (enrich from Hi-Fi) |
| DELETE | `/htmx/download/{id}` | Delete a downloaded track |
| GET | `/htmx/track/{id}` | Track form fragment |
| POST | `/htmx/track/{id}/save` | Save track metadata |
| POST | `/htmx/track/{id}/sync` | Re-tag file with existing metadata |
| POST | `/htmx/track/{id}/enrich` | Enrich track from MusicBrainz |
| POST | `/htmx/track/{id}/enrich-hifi` | Enrich track from Hi-Fi + MusicBrainz |
| GET | `/htmx/providers` | Get provider configuration |
| POST | `/htmx/provider/set?url={url}` | Set active provider |
| POST | `/htmx/provider/add?name={name}&url={url}` | Add custom provider |
| POST | `/htmx/provider/remove?url={url}` | Remove custom provider |
| GET | `/htmx/genre-map` | Get genre map configuration (JSON) |
| POST | `/htmx/genre-map` | Save custom genre map |
| POST | `/htmx/genre-map/reset` | Reset genre map to default |

### Track Pages

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/track/{id}` | Track detail/edit page |

---

## Behavior

All POST endpoints enqueue jobs asynchronously.
They never block waiting for downloads.
Jobs are processed by background workers.

Download types accepted:
- `track` - Single track
- `album` - Full album (decomposes into tracks, saves cover.jpg)
- `playlist` - Playlist (decomposes into tracks, generates M3U file)
- `artist` - Artist top tracks (decomposes into tracks, generates M3U file)

---

## Response Formats

### HTML Fragments
HTMX endpoints return HTML fragments for DOM replacement.

### JSON (Providers)
Provider configuration endpoints return JSON:
```json
{
  "predefined": [{"name": "...", "url": "..."}],
  "custom": [{"name": "...", "url": "..."}],
  "active": "http://...",
  "default": "http://..."
}
```

### JSON (Genre Map)
Genre map endpoints return JSON:
```json
{
  "default": {"death metal": "Metal", "indie pop": "Pop", ...},
  "custom": {"ambient techno": "Electronic", ...}
}
```
`custom` is `null` if no custom map is set.
