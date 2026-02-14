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
| GET | `/history` | Download history page |
| GET | `/settings` | Settings page |

### HTMX Fragments

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/htmx/search?q={query}&type={type}` | Search results fragment |
| GET | `/htmx/album/{id}/similar` | Similar albums fragment |
| POST | `/htmx/download/{type}/{id}` | Enqueue download job |
| GET | `/htmx/queue` | Queue list fragment |
| POST | `/htmx/cancel/{id}` | Cancel a job |
| POST | `/htmx/retry/{id}` | Retry a failed job |
| POST | `/htmx/history/clear` | Clear finished jobs |
| GET | `/htmx/providers` | Get provider configuration |
| POST | `/htmx/provider/set?url={url}` | Set active provider |
| POST | `/htmx/provider/add?name={name}&url={url}` | Add custom provider |
| POST | `/htmx/provider/remove?url={url}` | Remove custom provider |

---

## Behavior

All POST endpoints enqueue jobs asynchronously.
They never block waiting for downloads.
Jobs are processed by background workers.

Download types accepted:
- `track` - Single track
- `album` - Full album (decomposes into tracks)
- `playlist` - Playlist (decomposes into tracks)
- `artist` - Artist top tracks (decomposes into tracks)

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
