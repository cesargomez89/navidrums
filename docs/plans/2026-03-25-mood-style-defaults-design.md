# Design: Default Mood and Style Lists

## Overview

Add editable default lists for moods and styles in Settings, similar to genre mapping. Users can select multiple values from dropdowns in track view and bulk actions. Values are stored semicolon-separated and written to file tags using the genre separator.

## Data Flow

1. **Defaults** → Hardcoded Go constants (mood list, style list)
2. **Settings** → Stored as JSON arrays in `settings` table (like genre_map)
3. **UI** → Multi-select dropdowns populated from defaults
4. **Database** → Semicolon-separated string (e.g., "Chill;Dark;Uplifting")
5. **Tagging** → Split by separator and write multiple custom frames

## Components

### 1. Backend Constants (`internal/app/defaults.go` - new file)

```go
var DefaultMoods = []string{
    "Aggressive", "Atmospheric", "Chill", "Dark", "Energetic",
    "Melancholic", "Mystical", "Romantic", "Sophisticated", "Uplifting",
}

var DefaultStyles = []string{
    "Acoustic", "Alternative", "Cinematic", "Electronic", "Hardcore",
    "Lyricist", "Pop", "Rock", "Traditional", "Urban", "Crossover",
}
```

### 2. Settings Storage

- New settings keys: `mood_list`, `style_list`
- Same pattern as genre_map: GET returns both default + custom, POST saves custom, RESET deletes custom

### 3. HTMX Endpoints (in `internal/http/routes.go`)

- `GET /htmx/mood-list` → returns default + custom moods as JSON
- `POST /htmx/mood-list` → saves custom mood list
- `POST /htmx/mood-list/reset` → resets to default
- Same for `style-list`

### 4. Settings UI (`web/templates/settings.html`)

- Add sections for "Mood List" and "Style List" (similar to Genre Mapping)
- JSON textarea for editing
- Save/Reset buttons

### 5. Track Form Multi-Select (`web/templates/components/track_form.html`)

- Replace text input with multi-select dropdown
- Populated from mood_list/style_list settings
- Stores semicolon-separated values

### 6. Bulk Action Modal (`web/templates/downloads.html`)

- Replace text inputs with multi-select dropdowns
- Same pattern as track form

### 7. Tagging (`internal/tagging/tagging.go`)

- Already handles mood/style via `addCustom("MOOD", ...)` and `addCustom("STYLE", ...)`
- Modify to split by separator (like genre) - already uses GenreSeparator variable

## Edge Cases

- Empty custom list → fall back to defaults
- Values not in default list → allow (user can add custom)
- Existing tracks with plain strings → continue to work (no migration needed)

## Testing Strategy

- Unit tests for default lists in new `defaults.go`
- HTMX endpoint tests for mood-list/style-list (GET, POST, RESET)
- Frontend: verify dropdowns populate correctly, multi-select stores semicolon-separated values

## Migration & Compatibility

- No database migration needed - `mood` and `style` columns already exist
- Existing tracks with single values continue to work
- Tagging already uses `addCustom()` which handles multiple values
