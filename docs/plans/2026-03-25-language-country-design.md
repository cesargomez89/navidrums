# Design: Language and Country Fields for Tracks

## Overview

Add language and country fields to tracks with editable default lists, editable from settings UI, and usable in bulk metadata operations and single track editing.

## Data Model

### Domain Model
Add to `internal/domain/models.go` Track struct:
- `Language` - string, ISO 639-2 code (e.g., "en", "es", "pt-BR")
- `Country` - string, ISO 3166-1 alpha-2 code (e.g., "US", "BR", "GB")

### Database Schema
Add columns to `internal/store/schema.go`:
```sql
language TEXT,
country TEXT,
```

## Settings Constants & Defaults

### Store
Add to `internal/store/settings.go`:
```go
SettingLanguageList = "language_list"
SettingCountryList = "country_list"
```

### Defaults
Add to `internal/app/defaults.go`:
```go
var DefaultLanguages = map[string]string{
    "ar":    "Arabic",
    "zh":    "Chinese",
    "en":    "English",
    "fr":    "French",
    "de":    "German",
    "hi":    "Hindi",
    "it":    "Italian",
    "ja":    "Japanese",
    "ko":    "Korean",
    "pt":    "Portuguese",
    "es":    "Spanish",
    "es-419": "Spanish (Latin America)",
}

var DefaultCountries = map[string]string{
    "ar": "Argentina",
    "br": "Brazil",
    "ca": "Canada",
    "cl": "Chile",
    "cn": "China",
    "co": "Colombia",
    "cu": "Cuba",
    "fr": "France",
    "de": "Germany",
    "in": "India",
    "it": "Italy",
    "jp": "Japan",
    "mx": "Mexico",
    "pr": "Puerto Rico",
    "kr": "South Korea",
    "es": "Spain",
    "gb": "United Kingdom",
    "us": "United States",
    "ve": "Venezuela",
}
```

Add getter functions `GetLanguages()` and `GetCountries()` following existing pattern for moods/styles.

## Audio Tagging

### TagMap
Add to `internal/tagging/tagging.go` TagMap struct:
```go
Language string
Country  string
```

### Tag Writing
Write to standard tags:
- **MP3**: `TLAN` frame for language
- **FLAC/Vorbis**: `LANGUAGE` comment
- **MP4**: Via ffmpeg metadata

Also add as custom tags for compatibility:
- `LANGUAGE` (custom)
- `COUNTRY` (custom)

## Settings UI

Add language/country list editors to settings page, following the existing mood/style pattern (JSON array storage).

## Bulk Metadata Modal

Extend `web/templates/downloads.html`:
- Add language dropdown (populated from `/htmx/languages`)
- Add country dropdown (populated from `/htmx/countries`)

Add HTMX endpoints:
- `GET /htmx/languages` - returns JSON
- `GET /htmx/countries` - returns JSON
- `POST /htmx/downloads/bulk-genre` - already handles arbitrary fields, just need to add language/country to request

## Single Track Edit

Add to `web/templates/track.html`:
- New "Language & Region" section with dropdown fields
- Inline editing via HTMX handler `PUT /htmx/track/{id}`

## Implementation Order

1. Add constants to `internal/store/settings.go`
2. Add defaults to `internal/app/defaults.go` with getter functions
3. Add fields to `internal/domain/models.go` Track struct
4. Add columns to `internal/store/schema.go`
5. Update repository methods for Track (if needed)
6. Add to `internal/tagging/tagging.go` TagMap and tag writers
7. Add HTMX endpoints for languages/countries
8. Add settings page editors
9. Update bulk metadata modal
10. Update single track page

## Testing

- Unit tests for defaults getters
- Manual UI testing for dropdown population
- Verify tags written correctly to MP3/FLAC/MP4