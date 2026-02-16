# Navidrums Refactoring Plan

## Overview
This document outlines areas for refactoring, removing duplication, and simplifying the codebase.

---

## Priority 1: SQL Query Duplication (High Impact)

### Issue
The same massive SELECT statement (~45 columns) is duplicated across 8+ methods in `internal/store/tracks.go`:
- `GetTrackByID`
- `GetTrackByProviderID`
- `ListTracks`
- `ListTracksByStatus`
- `ListTracksByParentJobID`
- `SearchTracks`
- `GetDownloadedTrack`
- `FindInterruptedTracks`

### Impact
~400 lines of duplicated SQL, fragile to schema changes.

### Solution
Extract to a helper function that returns column list, reuse in all methods.

---

## Priority 2: Duplicate Job Processing Logic (High Impact)

### Issue
`processAlbumJob`, `processPlaylistJob`, `processArtistJob` in `internal/downloader/worker.go` share ~80% identical code:
- Checking if track already downloaded
- Checking if track already active
- Creating track records
- Creating child track jobs

### Impact
~200 duplicate lines across three functions.

### Solution
Extract common logic to helper function `createTrackAndJob(ctx, catalogTrack, parentJobID)`.

---

## Priority 3: Unused Constants (Easy Cleanup)

### Issue
Many constants in `internal/constants/constants.go` are defined but never used:
- `Endpoint*` constants (lines 43-56) - API endpoints not used
- `Param*` constants (lines 59-72) - query params not used
- `Status*`/`Type*` Job constants (lines 84-100) - domain types used instead
- `TidalImageBaseURL`, `TokenFileName`, `CoverFileName`

### Solution
Remove unused constants.

---

## Priority 4: Redundant ProviderManager Wrapper

### Issue
`internal/catalog/manager.go` - `ProviderManager` wraps `Provider` interface with identical method signatures, adding indirection without value.

### Solution
Consider simplifying or removing this wrapper layer.

---

## Priority 5: Duplicate Schema Comment

### Issue
`internal/store/schema.go:61-62` has duplicate `-- Processing` comment.

### Solution
Remove duplicate line.

---

## Priority 6: Overkill Fields (Requires Migration)

### Issue
Some fields in domain models are rarely populated:
- `Track.Description`, `Track.URL`, `Track.AudioModes`
- `Track.ArtistIDs`, `Track.AlbumArtistIDs` (JSON arrays add complexity)
- `Track.ReleaseDate` (redundant with `Track.Year`)
- `CatalogTrack.ExplicitLyrics` (vs `Explicit`)

### Impact
Requires database migration, lower ROI.

### Solution
Consider removing after Priority 1-4 are complete.

---

## Implementation Order

1. **Extract SQL queries** - biggest code smell, highest impact
2. **Extract job processing helper** - reduces ~200 duplicate lines
3. **Remove unused constants** - easy cleanup
4. **Remove ProviderManager wrapper** - simplify architecture
5. **Consider field removals** - requires migration, lower priority
