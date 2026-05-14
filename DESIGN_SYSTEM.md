# Design System Guide

CSS/HTML patterns for navidrums.

## Principles

1. **Utility-first** — use existing classes before custom CSS
2. **Generic components** — reusable patterns, not one-offs
3. **No inline styles** — except JS visibility (`display:none`)

---

## CSS Variables

Colors: `--bg`, `--card`, `--accent`, `--text-dim`, `--danger`, `--success`, `--warning`
Spacing: `--space-1` through `--space-8` (4px base)
Typography: `--text-xs` through `--text-3xl`
Radius: `--radius-sm`, `--radius-md`, `--radius-lg`, `--radius-xl`

## Utility Classes

| Category | Classes |
|----------|---------|
| Layout | `.flex`, `.flex-col`, `.flex-wrap`, `.flex-1`, `.grid` |
| Alignment | `.items-center`, `.items-start`, `.justify-between`, `.justify-center`, `.justify-end` |
| Spacing | `.gap-1` to `.gap-6`, `.p-1` to `.p-6`, `.mt-2` to `.mt-6`, `.mb-2` to `.mb-6` |
| Typography | `.text-xs` to `.text-3xl`, `.font-medium`, `.font-bold`, `.text-center`, `.text-dim`, `.text-accent`, `.text-danger`, `.text-success`, `.uppercase`, `.truncate` |
| Other | `.rounded-sm` to `.rounded-xl`, `.border`, `.w-full`, `.relative`, `.sticky`, `.cursor-pointer`, `.transition` |

---

## Components

### Item (`.item`)

Image + content + actions for list rows.

```html
<div class="item">
  <img class="item-img" src="..." alt="...">
  <div class="item-body">
    <div class="item-title"><a href="...">Title</a></div>
    <div class="item-subtitle">Subtitle</div>
  </div>
  <div class="item-actions"><button class="btn btn-sm">Action</button></div>
</div>
```

### Info Grid (`.info-grid`)

Key-value data display.

```html
<div class="info-grid">
  <div class="data-item"><span class="data-label">Label</span><span class="data-value">Value</span></div>
  <div class="data-item data-item--full">Full width</div>
</div>
```

Variants: `.info-grid--full`, `.data-item--full`, `.data-value--mono`

### List Grid (`.list-grid`)

Wide items on large screens — 400px min per item.

### Toolbar (`.toolbar`)

Search/filter bars. `.toolbar-row`, `.toolbar-section`, `.toolbar-section--grow`, `.toolbar-section--end`

### Stats Grid (`.stats-grid`)

Label/value pairs.

```html
<div class="stats-grid">
  <div class="stat-item"><span class="stat-label">Label</span><span class="stat-value">Value</span></div>
</div>
```

### Section Header (`.section-header`)

Title + action button.

---

## Buttons

| Class | Description |
|-------|-------------|
| `.btn` | Default yellow accent |
| `.btn-secondary` | Outline |
| `.btn-outline` | Transparent with border |
| `.btn-outline-danger` | Red outline |
| `.btn-danger` | Red background |
| `.btn-success` | Green background |
| `.btn-warning` | Orange background |

Sizes: `.btn-sm`, `.btn-lg`

---

## Cards (`.card`)

```html
<div class="card">
  <a href="..." class="card-img-wrapper"><img src="..." alt="..." loading="lazy"></a>
  <span class="quality-badge quality-badge--lossless">Lossless</span>
  <div class="card-title"><a href="...">Title</a></div>
  <div class="card-sub">Subtitle</div>
</div>
```

Quality badges: `.quality-badge--hires`, `.quality-badge--lossless`, `.quality-badge--high`, `.quality-badge--low`

---

## Forms

```html
<div class="form-group"><label>Label</label><input type="text"></div>
<div class="form-grid"><div class="form-group">...</div></div>
<select class="form-select">...</select>
```

---

## Alerts & Status

```html
<div class="alert alert-success">Success</div>
<div class="alert alert-error">Error</div>

<span class="job-status status-queued">Queued</span>
<span class="job-status status-processing">Processing</span>
<span class="job-status status-completed">Completed</span>
<span class="job-status status-failed">Failed</span>
<span class="job-status status-cancelled">Cancelled</span>
```