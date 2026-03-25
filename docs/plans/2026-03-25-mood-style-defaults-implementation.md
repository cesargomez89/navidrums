# Mood and Style Defaults Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add editable default mood/style lists in Settings, with multi-select dropdowns in track view and bulk actions. Values stored semicolon-separated, written to file tags.

**Architecture:** Add new settings keys (mood_list, style_list) stored as JSON arrays in settings table, similar to genre_map. Update UI to use multi-select dropdowns.

**Tech Stack:** Go (backend), SQLite (settings), HTML/JS (frontend)

---

### Task 1: Create default constants

**Files:**
- Create: `internal/app/defaults.go`

**Step 1: Create the file with default lists**

```go
package app

var DefaultMoods = []string{
	"Aggressive",
	"Atmospheric",
	"Chill",
	"Dark",
	"Energetic",
	"Melancholic",
	"Mystical",
	"Romantic",
	"Sophisticated",
	"Uplifting",
}

var DefaultStyles = []string{
	"Acoustic",
	"Alternative",
	"Cinematic",
	"Electronic",
	"Hardcore",
	"Lyricist",
	"Pop",
	"Rock",
	"Traditional",
	"Urban",
	"Crossover",
}
```

**Step 2: Add setting constants**

**Files:**
- Modify: `internal/store/settings.go` - add SettingMoodList and SettingStyleList constants

Add to const block:
```go
SettingMoodList  = "mood_list"
SettingStyleList = "style_list"
```

**Step 3: Commit**

```bash
git add internal/app/defaults.go internal/store/settings.go
git commit -m "feat: add default mood and style lists"
```

---

### Task 2: Add HTMX endpoints for mood list

**Files:**
- Modify: `internal/http/handler.go` - register routes
- Modify: `internal/http/routes.go` - implement handlers

**Step 1: Add route registrations in handler.go**

Find where genre-map routes are registered and add similar ones:
```go
r.Get("/htmx/mood-list", h.GetMoodListHTMX)
r.Post("/htmx/mood-list", h.SetMoodListHTMX)
r.Post("/htmx/mood-list/reset", h.ResetMoodListHTMX)
```

**Step 2: Add handler functions in routes.go**

Add after genre-map handlers (~line 467):
```go
func (h *Handler) GetMoodListHTMX(w http.ResponseWriter, r *http.Request) {
	custom, _ := h.SettingsRepo.Get(store.SettingMoodList)
	
	result := map[string]interface{}{
		"default": app.DefaultMoods,
	}
	if custom != "" {
		var list []string
		if err := json.Unmarshal([]byte(custom), &list); err == nil {
			result["custom"] = list
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) SetMoodListHTMX(w http.ResponseWriter, r *http.Request) {
	var body struct {
		MoodList []string `json:"moodList"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		json.NewEncoder(w).Encode(map[string]bool{"success": false})
		return
	}
	
	data, _ := json.Marshal(body.MoodList)
	h.SettingsRepo.Set(store.SettingMoodList, string(data))
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) ResetMoodListHTMX(w http.ResponseWriter, r *http.Request) {
	h.SettingsRepo.Delete(store.SettingMoodList)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
```

**Step 3: Import app package if not already**

Add to imports in routes.go:
```go
"navidrums/internal/app"
```

**Step 4: Commit**

```bash
git add internal/http/handler.go internal/http/routes.go
git commit -m "feat: add mood list HTMX endpoints"
```

---

### Task 3: Add HTMX endpoints for style list

**Files:**
- Modify: `internal/http/handler.go` - register routes
- Modify: `internal/http/routes.go` - implement handlers

**Step 1: Add route registrations in handler.go**

Add:
```go
r.Get("/htmx/style-list", h.GetStyleListHTMX)
r.Post("/htmx/style-list", h.SetStyleListHTMX)
r.Post("/htmx/style-list/reset", h.ResetStyleListHTMX)
```

**Step 2: Add handler functions in routes.go**

Add after mood-list handlers:
```go
func (h *Handler) GetStyleListHTMX(w http.ResponseWriter, r *http.Request) {
	custom, _ := h.SettingsRepo.Get(store.SettingStyleList)
	
	result := map[string]interface{}{
		"default": app.DefaultStyles,
	}
	if custom != "" {
		var list []string
		if err := json.Unmarshal([]byte(custom), &list); err == nil {
			result["custom"] = list
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) SetStyleListHTMX(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StyleList []string `json:"styleList"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		json.NewEncoder(w).Encode(map[string]bool{"success": false})
		return
	}
	
	data, _ := json.Marshal(body.StyleList)
	h.SettingsRepo.Set(store.SettingStyleList, string(data))
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) ResetStyleListHTMX(w http.ResponseWriter, r *http.Request) {
	h.SettingsRepo.Delete(store.SettingStyleList)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
```

**Step 3: Commit**

```bash
git add internal/http/handler.go internal/http/routes.go
git commit -m "feat: add style list HTMX endpoints"
```

---

### Task 4: Add Settings UI for mood list

**Files:**
- Modify: `web/templates/settings.html` - add mood list section

**Step 1: Add HTML section after Genre Mapping section**

Add after line 59 (after Genre Mapping section):
```html
<div class="section">
    <h2>Mood List</h2>
    <p class="hint">Default moods available for selection. Stored as JSON array.</p>
    <div id="mood-list-status"></div>
    <textarea id="mood-list-input" rows="8" class="w-full" style="font-family: monospace; font-size: 12px;"
        placeholder='["Aggressive", "Atmospheric", "Chill", ...]'></textarea>
    <div class="mt-2">
        <button onclick="saveMoodList()" class="btn-lg btn-primary">Save</button>
        <button onclick="resetMoodList()" class="btn-lg btn-secondary">Reset to Default</button>
    </div>
</div>
```

**Step 2: Add JavaScript functions**

Add after loadGenreMap() function (~line 197):
```javascript
function loadMoodList() {
    fetch('/htmx/mood-list', { cache: 'no-store' })
        .then(r => r.json())
        .then(data => {
            const textarea = document.getElementById('mood-list-input');
            const statusDiv = document.getElementById('mood-list-status');

            if (data.custom) {
                textarea.value = JSON.stringify(data.custom, null, 2);
                statusDiv.innerHTML = '<span class="badge badge-custom">Using custom list</span>';
            } else {
                textarea.value = JSON.stringify(data.default, null, 2);
                statusDiv.innerHTML = '<span class="badge badge-default">Using default list</span>';
            }
        });
}

function saveMoodList() {
    const textarea = document.getElementById('mood-list-input');
    let moodList;

    try {
        moodList = JSON.parse(textarea.value);
    } catch (e) {
        alert('Invalid JSON: ' + e.message);
        return;
    }

    fetch('/htmx/mood-list', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ moodList: moodList })
    })
        .then(r => r.json())
        .then(data => {
            if (data.success) {
                loadMoodList();
                alert('Mood list saved');
            }
        });
}

function resetMoodList() {
    if (!confirm('Reset to default mood list?')) return;

    fetch('/htmx/mood-list/reset', { method: 'POST' })
        .then(r => r.json())
        .then(data => {
            if (data.success) {
                loadMoodList();
            }
        });
}
```

**Step 3: Add loadMoodList() to init**

Add to the script's init section (before `</script>`):
```javascript
loadMoodList();
```

**Step 4: Commit**

```bash
git add web/templates/settings.html
git commit -m "feat: add mood list settings UI"
```

---

### Task 5: Add Settings UI for style list

**Files:**
- Modify: `web/templates/settings.html` - add style list section

**Step 1: Add HTML section after Mood List**

Add after mood list section:
```html
<div class="section">
    <h2>Style List</h2>
    <p class="hint">Default styles available for selection. Stored as JSON array.</p>
    <div id="style-list-status"></div>
    <textarea id="style-list-input" rows="8" class="w-full" style="font-family: monospace; font-size: 12px;"
        placeholder='["Acoustic", "Alternative", "Cinematic", ...]'></textarea>
    <div class="mt-2">
        <button onclick="saveStyleList()" class="btn-lg btn-primary">Save</button>
        <button onclick="resetStyleList()" class="btn-lg btn-secondary">Reset to Default</button>
    </div>
</div>
```

**Step 2: Add JavaScript functions**

Add after loadMoodList():
```javascript
function loadStyleList() {
    fetch('/htmx/style-list', { cache: 'no-store' })
        .then(r => r.json())
        .then(data => {
            const textarea = document.getElementById('style-list-input');
            const statusDiv = document.getElementById('style-list-status');

            if (data.custom) {
                textarea.value = JSON.stringify(data.custom, null, 2);
                statusDiv.innerHTML = '<span class="badge badge-custom">Using custom list</span>';
            } else {
                textarea.value = JSON.stringify(data.default, null, 2);
                statusDiv.innerHTML = '<span class="badge badge-default">Using default list</span>';
            }
        });
}

function saveStyleList() {
    const textarea = document.getElementById('style-list-input');
    let styleList;

    try {
        styleList = JSON.parse(textarea.value);
    } catch (e) {
        alert('Invalid JSON: ' + e.message);
        return;
    }

    fetch('/htmx/style-list', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ styleList: styleList })
    })
        .then(r => r.json())
        .then(data => {
            if (data.success) {
                loadStyleList();
                alert('Style list saved');
            }
        });
}

function resetStyleList() {
    if (!confirm('Reset to default style list?')) return;

    fetch('/htmx/style-list/reset', { method: 'POST' })
        .then(r => r.json())
        .then(data => {
            if (data.success) {
                loadStyleList();
            }
        });
}
```

**Step 3: Add loadStyleList() to init**

Add:
```javascript
loadStyleList();
```

**Step 4: Commit**

```bash
git add web/templates/settings.html
git commit -m "feat: add style list settings UI"
```

---

### Task 6: Add helper function to get merged mood/style lists

**Files:**
- Modify: `internal/app/defaults.go` - add GetMoods and GetStyles functions

**Step 1: Add functions to merge default and custom**

```go
func GetMoods(settingsRepo *store.SettingsRepo) []string {
	custom, err := settingsRepo.Get(store.SettingMoodList)
	if err != nil || custom == "" {
		return DefaultMoods
	}
	var list []string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultMoods
	}
	if len(list) == 0 {
		return DefaultMoods
	}
	return list
}

func GetStyles(settingsRepo *store.SettingsRepo) []string {
	custom, err := settingsRepo.Get(store.SettingStyleList)
	if err != nil || custom == "" {
		return DefaultStyles
	}
	var list []string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultStyles
	}
	if len(list) == 0 {
		return DefaultStyles
	}
	return list
}
```

**Step 2: Add imports**

```go
import (
	"encoding/json"
	"navidrums/internal/store"
)
```

**Step 3: Commit**

```bash
git add internal/app/defaults.go
git commit -m "feat: add GetMoods and GetStyles helper functions"
```

---

### Task 7: Add HTMX endpoints to serve mood/style lists to frontend

**Files:**
- Modify: `internal/http/routes.go` - add endpoint to get merged lists

**Step 1: Add GET endpoints for frontend dropdowns**

Add new routes in handler.go:
```go
r.Get("/htmx/moods", h.GetMoodsHTMX)
r.Get("/htmx/styles", h.GetStylesHTMX)
```

Add handlers in routes.go:
```go
func (h *Handler) GetMoodsHTMX(w http.ResponseWriter, r *http.Request) {
	moods := app.GetMoods(h.SettingsRepo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"moods": moods})
}

func (h *Handler) GetStylesHTMX(w http.ResponseWriter, r *http.Request) {
	styles := app.GetStyles(h.SettingsRepo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"styles": styles})
}
```

**Step 2: Commit**

```bash
git add internal/http/handler.go internal/http/routes.go
git commit -m "feat: add moods and styles endpoints for frontend"
```

---

### Task 8: Update track form with multi-select dropdowns

**Files:**
- Modify: `web/templates/components/track_form.html` - replace text inputs

**Step 1: Replace mood input with multi-select**

Current (lines 121-122):
```html
<label for="mood">Mood</label>
<input type="text" id="mood" name="mood" value="{{.Track.Mood}}">
```

Replace with:
```html
<label for="mood">Mood</label>
<select id="mood" name="mood" multiple class="multi-select" data-placeholder="Select moods...">
</select>
<div class="hint">Hold Ctrl/Cmd to select multiple</div>
```

**Step 2: Replace style input with multi-select**

Current (lines 126-127):
```html
<label for="style">Style</label>
<input type="text" id="style" name="style" value="{{.Track.Style}}">
```

Replace with:
```html
<label for="style">Style</label>
<select id="style" name="style" multiple class="multi-select" data-placeholder="Select styles...">
</select>
<div class="hint">Hold Ctrl/Cmd to select multiple</div>
```

**Step 3: Add JavaScript to populate dropdowns**

Add at end of file (before `{{end}}`):
```html
<script>
function loadMoodStyleOptions() {
    Promise.all([
        fetch('/htmx/moods', { cache: 'no-store' }).then(r => r.json()),
        fetch('/htmx/styles', { cache: 'no-store' }).then(r => r.json())
    ]).then(([moodData, styleData]) => {
        const moodSelect = document.getElementById('mood');
        const styleSelect = document.getElementById('style');
        
        const currentMood = "{{.Track.Mood}}";
        const currentStyle = "{{.Track.Style}}";
        
        // Populate moods
        moodData.moods.forEach(mood => {
            const option = document.createElement('option');
            option.value = mood;
            option.textContent = mood;
            if (currentMood.split(';').includes(mood)) {
                option.selected = true;
            }
            moodSelect.appendChild(option);
        });
        
        // Populate styles
        styleData.styles.forEach(style => {
            const option = document.createElement('option');
            option.value = style;
            option.textContent = style;
            if (currentStyle.split(';').includes(style)) {
                option.selected = true;
            }
            styleSelect.appendChild(option);
        });
    });
}
loadMoodStyleOptions();
</script>
```

**Step 4: Commit**

```bash
git add web/templates/components/track_form.html
git commit -m "feat: add multi-select dropdowns for mood and style in track form"
```

---

### Task 9: Update bulk action modal with multi-select dropdowns

**Files:**
- Modify: `web/templates/downloads.html` - replace text inputs in modal

**Step 1: Replace mood input**

Current (lines 55-56):
```html
<input type="text" id="mood-input" placeholder="Mood (e.g. Energetic, Chill, Melancholic...)"
    onkeydown="if(event.key==='Enter') applyMetadata()">
```

Replace with:
```html
<select id="mood-input" multiple class="multi-select-bulk" data-placeholder="Select moods...">
</select>
```

**Step 2: Replace style input**

Current (lines 57-58):
```html
<input type="text" id="style-input" placeholder="Style (e.g. Ambient, Techno, House...)"
    onkeydown="if(event.key==='Enter') applyMetadata()">
```

Replace with:
```html
<select id="style-input" multiple class="multi-select-bulk" data-placeholder="Select styles...">
</select>
```

**Step 3: Add CSS for multi-select styling**

Add to style.css or in `<style>` block in downloads.html:
```css
.multi-select-bulk {
    width: 100%;
    min-height: 80px;
    padding: 8px;
    background: var(--bg-secondary, #1e1e1e);
    border: 1px solid var(--border-color, #333);
    border-radius: 4px;
    color: var(--text-primary, #eee);
}
.multi-select-bulk option {
    padding: 4px 8px;
}
.multi-select-bulk option:checked {
    background: var(--accent, #646cff);
    color: white;
}
```

**Step 4: Add JavaScript to load options and handle selection**

Add to script section (before `</script>`):
```javascript
function loadBulkMoodStyleOptions() {
    Promise.all([
        fetch('/htmx/moods', { cache: 'no-store' }).then(r => r.json()),
        fetch('/htmx/styles', { cache: 'no-store' }).then(r => r.json())
    ]).then(([moodData, styleData]) => {
        const moodSelect = document.getElementById('mood-input');
        const styleSelect = document.getElementById('style-input');
        
        moodData.moods.forEach(mood => {
            const option = document.createElement('option');
            option.value = mood;
            option.textContent = mood;
            moodSelect.appendChild(option);
        });
        
        styleData.styles.forEach(style => {
            const option = document.createElement('option');
            option.value = style;
            option.textContent = style;
            styleSelect.appendChild(option);
        });
    });
}

function getMultiSelectValues(id) {
    const select = document.getElementById(id);
    return Array.from(select.selectedOptions).map(opt => opt.value).join(';');
}
```

**Step 5: Update applyMetadata to use getMultiSelectValues**

Modify applyMetadata function (line 208-209):
```javascript
var mood = getMultiSelectValues('mood-input');
var style = getMultiSelectValues('style-input');
```

**Step 6: Add loadBulkMoodStyleOptions() call**

Add to init section (find where other load functions are called):
```javascript
loadBulkMoodStyleOptions();
```

**Step 7: Commit**

```bash
git add web/templates/downloads.html
git commit -m "feat: add multi-select dropdowns for mood and style in bulk actions"
```

---

### Task 10: Run linter and tests

**Step 1: Run linter**

```bash
golangci-lint run
```

**Step 2: Run tests**

```bash
go test ./...
```

**Step 3: Commit any fixes**

```bash
git add -A && git commit -m "fix: linter/test fixes"
```

---

### Task 11: Verify in browser

**Step 1: Start the server**

```bash
go run ./cmd/server
```

**Step 2: Verify settings UI**

- Navigate to Settings page
- Verify Mood List and Style List sections appear
- Verify default values are shown
- Test Save and Reset buttons

**Step 3: Verify track form**

- Navigate to a track's edit page
- Verify mood/style dropdowns populate with options
- Verify selecting multiple values works

**Step 4: Verify bulk action**

- Select tracks in Downloads
- Open bulk action modal
- Verify mood/style dropdowns populate

---

**Plan complete and saved to `docs/plans/2026-03-25-mood-style-defaults-implementation.md`. Two execution options:**

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
