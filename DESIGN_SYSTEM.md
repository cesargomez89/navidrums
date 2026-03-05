# Design System Guide

This document outlines the CSS/HTML approach for navidrums to maintain consistency and minimize code.

## Principles

1. **Utility-first** - Use existing utility classes before writing custom CSS
2. **Generic components** - Create reusable patterns, not one-off styles
3. **No inline styles** - Except for JS-controlled state (`display:none`) or component-specific tweaks

---

## CSS Organization

### Variables (Lines 1-71)

All values should use CSS variables defined in `:root`:
- Colors: `--bg`, `--card`, `--accent`, `--text-dim`, `--danger`, `--success`, `--warning`
- Spacing: `--space-1` through `--space-8` (4px base)
- Typography: `--text-xs` through `--text-3xl`
- Radius: `--radius-sm`, `--radius-md`, `--radius-lg`, `--radius-xl`
- Shadows: `--shadow-*`, `--gradient-*`

### Utility Classes (Lines 144-261)

**Display & Layout**
```css
.flex { display: flex; }
.flex-col { flex-direction: column; }
.flex-wrap { flex-wrap: wrap; }
.flex-1 { flex: 1; }
.grid { display: grid; }

.items-center { align-items: center; }
.items-start { align-items: flex-start; }
.justify-between { justify-content: space-between; }
.justify-center { justify-content: center; }
.justify-end { justify-content: flex-end; }
```

**Spacing**
```css
.gap-1 through .gap-6     /* gap: var(--space-N) */
.p-1 through .p-6         /* padding: var(--space-N) */
.px-2 through .px-5       /* padding-left/right */
.py-1 through .py-4       /* padding-top/bottom */
.mt-2 through .mt-6       /* margin-top */
.mb-2 through .mb-6       /* margin-bottom */
.mx-auto                  /* margin-left/right: auto */
.ml-auto, .mr-auto
.ml-4                     /* margin-left */
```

**Typography**
```css
.text-xs, .text-sm, .text-base, .text-md, .text-lg, .text-xl, .text-2xl, .text-3xl
.font-medium, .font-bold
.text-center, .text-dim, .text-accent, .text-danger, .text-success
.uppercase, .truncate
```

**Borders & Radius**
```css
.rounded-sm, .rounded-md, .rounded-lg, .rounded-xl
.border, .border-t, .border-b
```

**Other**
```css
.w-full { width: 100%; }
.min-w-0 { min-width: 0; }
.relative { position: relative; }
.sticky { position: sticky; }
.overflow-hidden, .overflow-x-auto
.cursor-pointer
.transition
```

---

## Generic Components

### Item (`.item`)

For list rows with image + content + actions.

```html
<div class="item">
    <img class="item-img" src="..." alt="...">
    <div class="item-body">
        <div class="item-title"><a href="...">Title</a></div>
        <div class="item-subtitle">Subtitle text</div>
    </div>
    <div class="item-actions">
        <button class="btn btn-sm">Action</button>
    </div>
</div>
```

**Variants:**
- `.item-actions--col` - vertical alignment for actions

### Info Grid (`.info-grid`)

For key-value data display.

```html
<div class="info-grid">
    <div class="data-item">
        <span class="data-label">Label</span>
        <span class="data-value">Value</span>
    </div>
    <div class="data-item data-item--full">...</div>
</div>
```

**Variants:**
- `.info-grid--full` - single column
- `.data-item--full` - spans full width
- `.data-value--mono` - monospace font

### List Grid (`.list-grid`)

For displaying lists of wide items (like tracks or downloads) on larger screens. Uses a grid that switches to multiple columns on wide screens, minimum 400px per item.

```html
<div class="list-grid">
    <div class="item">...</div>
    <div class="item">...</div>
</div>
```

### Toolbar (`.toolbar`)

For search/filter bars or group buttons.

```html
<div class="toolbar">
    <div class="toolbar-row">
        <div class="toolbar-section toolbar-section--grow">
            <input ...>
            <button>Search</button>
        </div>
        <div class="toolbar-section">
            <select>...</select>
        </div>
    </div>
    <div class="toolbar-row">
        <div class="toolbar-section">...</div>
    </div>
</div>
```

**Classes:**
- `.toolbar-row` - horizontal row
- `.toolbar-section` - grouping
- `.toolbar-section--grow` - flex: 1
- `.toolbar-section--end` - margin-left: auto

### Stats Grid (`.stats-grid`)

```html
<div class="stats-grid">
    <div class="stat-item">
        <span class="stat-label">Label</span>
        <span class="stat-value">Value</span>
    </div>
</div>
```

### Section Header (`.section-header`)

```html
<div class="section-header">
    <h2>Title</h2>
    <button>Action</button>
</div>
```

---

## Buttons

### Variants

| Class | Description |
|-------|-------------|
| `.btn` | Default yellow accent |
| `.btn-primary` | Yellow accent (same as `.btn`) |
| `.btn-secondary` | Outline |
| `.btn-outline` | Transparent with border |
| `.btn-outline-danger` | Red outline |
| `.btn-danger` | Red background |
| `.btn-success` | Green background |
| `.btn-warning` | Orange background |

### Sizes

```html
<button class="btn">Default</button>
<button class="btn btn-sm">Small</button>
<button class="btn btn-lg">Large</button>
```

---

## Cards (`.card`)

For grid-based content (albums, artists).

```html
<div class="card">
    <a href="..." class="card-img-wrapper">
        <img src="..." alt="..." loading="lazy">
    </a>
    <!-- optional quality badge -->
    <span class="quality-badge quality-badge--lossless">Lossless</span>
    <div class="card-title"><a href="...">Title</a></div>
    <div class="card-sub">Subtitle</div>
</div>
```

### Quality Badges

```html
<span class="quality-badge">Default</span>
<span class="quality-badge quality-badge--hires">Hi-Res</span>
<span class="quality-badge quality-badge--lossless">Lossless</span>
<span class="quality-badge quality-badge--high">High</span>
<span class="quality-badge quality-badge--low">Low</span>
```

---

## Forms

### Form Group

```html
<div class="form-group">
    <label for="...">Label</label>
    <input type="text" id="...">
</div>
<div class="form-group form-group--full">Full width</div>
```

### Form Grid

```html
<div class="form-grid">
    <div class="form-group">...</div>
    <div class="form-group">...</div>
</div>
```

### Select

```html
<select class="form-select">...</select>
```

---

## Alerts

```html
<div class="alert alert-success">Success message</div>
<div class="alert alert-error">Error message</div>
```

---

## Status Badges

```html
<span class="job-status status-queued">Queued</span>
<span class="job-status status-processing">Processing</span>
<span class="job-status status-completed">Completed</span>
<span class="job-status status-failed">Failed</span>
<span class="job-status status-cancelled">Cancelled</span>
```

---

## When to Add New CSS

Before adding a new rule:

1. **Check existing utilities** - Can you combine `.flex`, `.items-center`, `.gap-3`, `.mt-4`?
2. **Check existing components** - Is this an `.item`, `.toolbar`, or `.info-grid`?
3. **Make it generic** - If you need a new pattern, add it to `style.css` with a reusable class

### Where to Add

| Type | Location |
|------|----------|
| Utility class | After line 261 (after existing utilities) |
| Button variant | After `.btn-outline` section |
| Generic component | After line 720 (after `.section`) |
| Page-specific | Avoid - extend generic components |

---

## HTML Guidelines

1. **No inline styles** except:
   - JS-controlled visibility: `style="display:none;"`
   - Component-specific width: `style="width:80px;"`

2. **Use consistent class order**:
   ```
   wrapper class
   structural (flex, grid)
   spacing (gap, p, m)
   typography
   component-specific
   ```

3. **Prefer semantic HTML**:
   - Use `<button>` for actions
   - Use `<a>` for navigation
   - Use `<label>` for form fields

4. **Accessibility**:
   - Always include `alt` for images
   - Use `loading="lazy"` for below-fold images
   - Include `type` on buttons when needed
