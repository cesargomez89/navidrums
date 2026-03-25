# Language and Country Fields Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add language and country fields to tracks with editable default lists, bulk metadata editing, and single track editing with audio tagging.

**Architecture:** Extend Track model with Language/Country fields, store in SQLite, expose via HTMX endpoints, edit via settings/bulk/single-track UIs, write to audio file tags.

**Tech Stack:** Go, SQLite, HTMX, id3v2 (MP3), go-flac (FLAC), ffmpeg (MP4)

---

## Task 1: Add Settings Constants

**Files:**
- Modify: `internal/store/settings.go:49-51`

**Step 1: Add constants**

Add after line 50:
```go
SettingLanguageList               = "language_list"
SettingCountryList                = "country_list"
```

**Step 2: Commit**
```bash
git add internal/store/settings.go
git commit -m "feat: add language and country list setting keys"
```

---

## Task 2: Add Default Maps and Getters

**Files:**
- Modify: `internal/app/defaults.go`

**Step 1: Add default maps**

After line 35 (after DefaultStyles closing brace):
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

**Step 2: Add getter functions**

After line 65 (end of file):
```go
func GetLanguages(settingsRepo *store.SettingsRepo) map[string]string {
	custom, err := settingsRepo.Get(store.SettingLanguageList)
	if err != nil || custom == "" {
		return DefaultLanguages
	}
	var list map[string]string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultLanguages
	}
	if len(list) == 0 {
		return DefaultLanguages
	}
	return list
}

func GetCountries(settingsRepo *store.SettingsRepo) map[string]string {
	custom, err := settingsRepo.Get(store.SettingCountryList)
	if err != nil || custom == "" {
		return DefaultCountries
	}
	var list map[string]string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultCountries
	}
	if len(list) == 0 {
		return DefaultCountries
	}
	return list
}
```

**Step 3: Commit**
```bash
git add internal/app/defaults.go
git commit -m "feat: add default languages and countries with getters"
```

---

## Task 3: Add Fields to Track Model

**Files:**
- Modify: `internal/domain/models.go:164-165`

**Step 1: Add fields**

After line 164 (`AlbumArtistIDs StringSlice`):
```go
	Language string `json:"language,omitempty" db:"language"`
	Country  string `json:"country,omitempty" db:"country"`
```

**Step 2: Commit**
```bash
git add internal/domain/models.go
git commit -m "feat: add Language and Country fields to Track model"
```

---

## Task 4: Add Columns to Database Schema

**Files:**
- Modify: `internal/store/schema.go`

**Step 1: Add columns**

Find the tracks table definition. Add after `album_artist_ids TEXT,`:
```sql
	language TEXT,
	country TEXT,
```

**Step 2: Add migration (optional - for existing databases)**

If there's a migration system, add migration. Otherwise, new installs will get columns automatically.

**Step 3: Commit**
```bash
git add internal/store/schema.go
git commit -m "feat: add language and country columns to tracks table"
```

---

## Task 5: Add Tagging Support

**Files:**
- Modify: `internal/tagging/tagging.go`

**Step 1: Add to TagMap struct**

After line 56 (`BPM int`):
```go
	Language string
	Country  string
```

**Step 2: Add to buildTagMap function**

After line 109 (after `Custom: make(map[string]string),`):
```go
		Language: track.Language,
		Country:  track.Country,
```

**Step 3: Add custom tags for compatibility**

After line 147 (after `addCustom("URL", track.URL)`):
```go
	addCustom("LANGUAGE", track.Language)
	addCustom("COUNTRY", track.Country)
```

**Step 4: Add to MP3Tagger newVorbisComment**

Add after line 560 (after UNSYNCEDLYRICS):
```go
	if tags.Language != "" {
		add("TLAN", tags.Language)
	}
```

**Step 5: Add to MP3Tagger WriteTags (TLAN frame)**

After line 391 (before closing brace of custom loop):
```go
	case "LANGUAGE":
		tag.AddTextFrame("TLAN", tag.DefaultEncoding(), v)
	case "COUNTRY":
		tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding:    id3v2.EncodingUTF8,
			Description: "COUNTRY",
			Value:       v,
		})
```

**Step 6: Add to FLACTagger newVorbisComment**

Add after line 560 (after UNSYNCEDLYRICS loop):
```go
	if tags.Language != "" {
		add("LANGUAGE", tags.Language)
	}
	if tags.Country != "" {
		add("COUNTRY", tags.Country)
	}
```

**Step 7: Add to MP4Tagger WriteTags**

Add to Metadata struct in lines 192-212:
```go
Language: tags.Language,
Country:  tags.Country,
```

**Step 8: Commit**
```bash
git add internal/tagging/tagging.go
git commit -m "feat: add language and country tagging support"
```

---

## Task 6: Add HTMX Endpoints

**Files:**
- Modify: `internal/http/handler.go`
- Modify: `internal/http/routes.go`

**Step 1: Add handler functions**

In handler.go, add after GetMoodsHTMX/GetStylesHTMX:
```go
func (h *Handler) GetLanguagesHTMX(w http.ResponseWriter, r *http.Request) {
	list := app.GetLanguages(h.SettingsRepo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"languages": list})
}

func (h *Handler) GetCountriesHTMX(w http.ResponseWriter, r *http.Request) {
	list := app.GetCountries(h.SettingsRepo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"countries": list})
}
```

**Step 2: Add routes**

In routes.go, after the moods/styles routes:
```go
r.Get("/htmx/languages", h.GetLanguagesHTMX)
r.Get("/htmx/countries", h.GetCountriesHTMX)
```

**Step 3: Commit**
```bash
git add internal/http/handler.go internal/http/routes.go
git commit -m "feat: add HTMX endpoints for languages and countries"
```

---

## Task 7: Add Settings Page Editors

**Files:**
- Modify: `web/templates/settings.html` (find existing mood/style editors)
- May need: HTMX handler for saving language/country lists

**Step 1: Find existing settings pattern**

Look at how mood/style list editors are implemented in settings.html.

**Step 2: Add language/country list editors**

Add similar editors for language and country lists following the same pattern.

**Step 3: Add save handlers if needed**

Check if existing save pattern works or needs extension.

**Step 4: Commit**
```bash
git add web/templates/settings.html
git commit -m "feat: add language and country list editors to settings"
```

---

## Task 8: Update Bulk Metadata Modal

**Files:**
- Modify: `web/templates/downloads.html:50-67`

**Step 1: Add dropdown fields**

After the existing form fields (line 66), add:
```html
<select id="language-input" class="form-select">
    <option value="">Select language...</option>
</select>
<select id="country-input" class="form-select">
    <option value="">Select country...</option>
</select>
```

**Step 2: Add JavaScript to populate**

After line 305 (after loadBulkMoodStyleOptions call):
```javascript
function loadBulkLanguageCountryOptions() {
    Promise.all([
        fetch('/htmx/languages', { cache: 'no-store' }).then(r => r.json()),
        fetch('/htmx/countries', { cache: 'no-store' }).then(r => r.json())
    ]).then(([langData, countryData]) => {
        const langSelect = document.getElementById('language-input');
        const countrySelect = document.getElementById('country-input');

        Object.entries(langData.languages).forEach(([code, name]) => {
            const option = document.createElement('option');
            option.value = code;
            option.textContent = name;
            langSelect.appendChild(option);
        });

        Object.entries(countryData.countries).forEach(([code, name]) => {
            const option = document.createElement('option');
            option.value = code;
            option.textContent = name;
            countrySelect.appendChild(option);
        });
    });
}
loadBulkLanguageCountryOptions();
```

**Step 3: Update applyMetadata function**

Add language and country to the params:
```javascript
var language = document.getElementById('language-input').value;
var country = document.getElementById('country-input').value;

// In the condition check:
if (!pathArtist && !artists && !albumArtists && !year && !genre && !mood && !style && !language && !country) {

// In the params:
if (language) params.set('language', language);
if (country) params.set('country', country);
```

**Step 4: Update openGenreModal**

Reset the new fields:
```javascript
document.getElementById('language-input').value = '';
document.getElementById('country-input').value = '';
```

**Step 5: Commit**
```bash
git add web/templates/downloads.html
git commit -m "feat: add language and country to bulk metadata modal"
```

---

## Task 9: Update Bulk Genre Handler

**Files:**
- Modify: `internal/http/routes.go` (find bulk-genre handler)

**Step 1: Find handler**

Look for `DownloadsBulkGenre` or similar handler.

**Step 2: Add language/country fields**

Add parsing for `language` and `country` form parameters and update the track fields.

**Step 3: Commit**
```bash
git commit -m "feat: handle language and country in bulk metadata handler"
```

---

## Task 10: Add to Single Track Page

**Files:**
- Modify: `web/templates/track.html`

**Step 1: Add section**

After line 104 (before `{{end}}`), add:
```html
<div class="section">
    <h2>Language & Region</h2>
    <div class="info-grid">
        <div class="data-item">
            <span class="data-label">Language</span>
            <span class="data-value">{{if .Track.Language}}{{.Track.Language}}{{else}}—{{end}}</span>
        </div>
        <div class="data-item">
            <span class="data-label">Country</span>
            <span class="data-value">{{if .Track.Country}}{{.Track.Country}}{{else}}—{{end}}</span>
        </div>
    </div>
</div>
```

For editing, add inline edit capability or a modal trigger similar to other fields.

**Step 2: Commit**
```bash
git add web/templates/track.html
git commit -m "feat: add language and country display to track page"
```

---

## Task 11: Run Tests and Verify

**Step 1: Run tests**
```bash
go test ./...
go build -o navidrums ./cmd/server
```

**Step 2: Run linter**
```bash
golangci-lint run
```

**Step 3: Commit any fixes**
```bash
git commit -m "fix: test/lint fixes"
```

---

## Plan Complete

The implementation plan has 11 tasks covering:
1. Settings constants
2. Default maps and getters
3. Track model fields
4. Database schema
5. Audio tagging support
6. HTMX endpoints
7. Settings UI editors
8. Bulk metadata modal
9. Bulk handler update
10. Single track page
11. Tests and verification

Each task is designed for 2-5 minute execution with immediate commits.