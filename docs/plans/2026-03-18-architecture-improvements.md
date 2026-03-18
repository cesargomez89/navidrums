# Architecture Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**Goal:** Fix critical and medium-priority architecture issues in the navidrums download orchestrator: false progress reporting, race conditions in M3U generation, missing job hierarchy, and partial failure handling.

**Architecture:** 
- Add `ParentJobID` to Job struct for proper job hierarchy
- Track container job progress by aggregating child track completion counts
- Add database-level locking for M3U generation to prevent race conditions
- Wrap decomposition in database transactions for atomicity
- Introduce `decomposed` status for container jobs to avoid false completion signals

**Tech Stack:** Go, SQLite (existing), domain models, repository pattern

---

## Task 1: Add `ParentJobID` to Job Struct

**Files:**
- Modify: `internal/domain/models.go`
- Modify: `internal/store/schema.go`
- Modify: `internal/downloader/handlers.go:544-551`
- Test: `internal/store/jobs_test.go` (create if missing)
- Test: `internal/downloader/handlers_test.go` (create if missing)

**Step 1: Write failing test for Job.ParentJobID**

Create `internal/store/jobs_test.go`:
```go
func TestCreateJobWithParent(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    parentID := "parent-job-123"
    childJob := &domain.Job{
        ID:         "child-job-456",
        Type:       domain.JobTypeTrack,
        Status:     domain.JobStatusQueued,
        SourceID:   "track-789",
        ParentJobID: parentID,
    }
    err := db.CreateJob(childJob)
    require.NoError(t, err)

    retrieved, err := db.GetJob("child-job-456")
    require.NoError(t, err)
    assert.Equal(t, parentID, retrieved.ParentJobID)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store/... -run TestCreateJobWithParent -v`
Expected: FAIL - undefined function or missing field

**Step 3: Add ParentJobID to domain.Job struct**

In `internal/domain/models.go`, find the Job struct and add:
```go
type Job struct {
    // ... existing fields ...
    ParentJobID string `json:"parent_job_id" db:"parent_job_id"`
}
```

**Step 4: Add column to schema**

In `internal/store/schema.go`, add to `CREATE TABLE IF NOT EXISTS jobs`:
```go
parent_job_id TEXT,
```
Also add to the corresponding INSERT statements in `jobs.go`.

**Step 5: Update CreateJob to accept ParentJobID**

In `internal/downloader/handlers.go:544-551`, when creating child jobs:
```go
childJob := &domain.Job{
    ID:          uuid.New().String(),
    Type:        domain.JobTypeTrack,
    Status:      domain.JobStatusQueued,
    SourceID:    catalogTrack.ID,
    ParentJobID: job.ID, // ADD THIS LINE
    CreatedAt:   time.Now(),
    UpdatedAt:   time.Now(),
}
```

**Step 6: Run tests to verify they pass**

Run: `go test ./internal/store/... -run TestCreateJobWithParent -v && go test ./internal/downloader/... -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/domain/models.go internal/store/schema.go internal/store/jobs.go internal/downloader/handlers.go
git commit -m "feat: add ParentJobID to Job struct for proper hierarchy"
```

---

## Task 2: Fix False Progress - Track Real Container Job Status

**Files:**
- Modify: `internal/domain/models.go` - add `JobStatusDecomposed`
- Modify: `internal/downloader/handlers.go` - change container job status transitions
- Modify: `internal/downloader/handlers.go` - add progress aggregation in TrackJobHandler
- Modify: `internal/store/jobs.go` - add `CountChildJobsByParentJobID`
- Test: `internal/downloader/handlers_test.go`

**Step 1: Write failing test for decomposed status**

Add to `internal/downloader/handlers_test.go`:
```go
func TestContainerJobStatusTransitions(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Create album job
    job := &domain.Job{
        ID:       "album-job-1",
        Type:     domain.JobTypeAlbum,
        Status:   domain.JobStatusQueued,
        SourceID: "album-123",
    }
    db.CreateJob(job)

    // Process album job (simulate decomposition)
    handler := NewContainerJobHandler(db, provider, logger)
    err := handler.Handle(job)
    require.NoError(t, err)

    // Job should be decomposed, NOT completed immediately
    updated, _ := db.GetJob("album-job-1")
    assert.Equal(t, domain.JobStatusDecomposed, updated.Status)
    assert.Equal(t, 0, updated.Progress) // No actual work done yet
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/downloader/... -run TestContainerJobStatusTransitions -v`
Expected: FAIL - JobStatusDecomposed not defined

**Step 3: Add JobStatusDecomposed to domain constants**

In `internal/domain/models.go`, add to the JobStatus type:
```go
const (
    JobStatusQueued     JobStatus = "queued"
    JobStatusRunning    JobStatus = "running"
    JobStatusDecomposed JobStatus = "decomposed" // ADD: Container created children
    JobStatusCompleted  JobStatus = "completed"
    JobStatusFailed     JobStatus = "failed"
    JobStatusCancelled  JobStatus = "cancelled"
)
```

**Step 4: Change container job handlers to set Decomposed instead of Completed**

In `internal/downloader/handlers.go`, find all places where container jobs are marked completed:
- `processAlbumJob`: Change `h.Repo.UpdateJobStatus(job.ID, domain.JobStatusCompleted, 100)` to `h.Repo.UpdateJobStatus(job.ID, domain.JobStatusDecomposed, 0)`
- `processPlaylistJob`: Same change
- `processArtistJob`: Same change
- `processDiscographyJob`: Same change

**Step 5: Add progress aggregation in TrackJobHandler**

In `internal/downloader/handlers.go`, in the `finalizeTrackDownload()` function (or wherever track completion is handled), add progress update for parent job:

```go
func (h *TrackJobHandler) finalizeTrackDownload(track *domain.Track, logger *slog.Logger) error {
    // ... existing logic ...
    
    // Update parent container job progress
    if track.ParentJobID != "" {
        total, pending, err := h.Repo.CountJobsForParent(track.ParentJobID)
        if err == nil && total > 0 {
            progress := float64(total-pending) / float64(total) * 100
            h.Repo.UpdateJobProgress(track.ParentJobID, progress)
        }
        
        // Check if all children are done
        if pending == 0 {
            h.Repo.UpdateJobStatus(track.ParentJobID, domain.JobStatusCompleted, 100)
        }
    }
    
    // ... existing playlist generation trigger ...
}
```

**Step 6: Add CountJobsForParent to store**

In `internal/store/jobs.go`, add:
```go
func (db *DB) CountJobsForParent(parentID string) (total int, pending int, err error) {
    row := db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE parent_job_id = ?`, parentID)
    row.Scan(&total)
    
    row = db.QueryRow(`
        SELECT COUNT(*) FROM jobs 
        WHERE parent_job_id = ? AND status IN (?, ?)`, 
        parentID, JobStatusQueued, JobStatusRunning)
    row.Scan(&pending)
    return
}

func (db *DB) UpdateJobProgress(id string, progress float64) error {
    _, err := db.Exec(`UPDATE jobs SET progress = ?, updated_at = ? WHERE id = ?`, 
        progress, time.Now(), id)
    return err
}
```

**Step 7: Run tests to verify they pass**

Run: `go test ./internal/downloader/... -run TestContainerJobStatusTransitions -v`
Expected: PASS

**Step 8: Commit**

```bash
git add internal/domain/models.go internal/downloader/handlers.go internal/store/jobs.go
git commit -m "feat: track real container job progress via decomposed status and aggregation"
```

---

## Task 3: Fix Race Condition in M3U Generation

**Files:**
- Modify: `internal/store/schema.go` - add `m3u_generating` column to jobs table
- Modify: `internal/store/jobs.go` - add `TrySetM3UGenerating` with advisory lock
- Modify: `internal/downloader/handlers.go` - use atomic M3U generation check
- Test: `internal/downloader/handlers_test.go`

**Step 1: Write failing test for M3U generation race condition**

Add to `internal/downloader/handlers_test.go`:
```go
func TestM3UGenerationRaceCondition(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    parentID := "playlist-job-1"
    
    // First thread claims generation
    claimed, err := db.TrySetM3UGenerating(parentID)
    require.NoError(t, err)
    assert.True(t, claimed)
    
    // Second thread should NOT claim
    claimed2, err := db.TrySetM3UGenerating(parentID)
    require.NoError(t, err)
    assert.False(t, claimed2)
    
    // Clear flag
    db.ClearM3UGenerating(parentID)
    
    // Third thread can now claim
    claimed3, err := db.TrySetM3UGenerating(parentID)
    require.NoError(t, err)
    assert.True(t, claimed3)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/downloader/... -run TestM3UGenerationRaceCondition -v`
Expected: FAIL - TrySetM3UGenerating not defined

**Step 3: Add m3u_generating column to schema**

In `internal/store/schema.go`, add to `CREATE TABLE IF NOT EXISTS jobs`:
```go
m3u_generating INTEGER DEFAULT 0,
```

**Step 4: Implement TrySetM3UGenerating with advisory lock**

In `internal/store/jobs.go`, add:
```go
func (db *DB) TrySetM3UGenerating(parentID string) (bool, error) {
    result, err := db.Exec(`
        UPDATE jobs 
        SET m3u_generating = 1 
        WHERE id = ? AND m3u_generating = 0`,
        parentID)
    if err != nil {
        return false, err
    }
    affected, _ := result.RowsAffected()
    return affected > 0, nil
}

func (db *DB) ClearM3UGenerating(parentID string) error {
    _, err := db.Exec(`UPDATE jobs SET m3u_generating = 0 WHERE id = ?`, parentID)
    return err
}
```

**Step 5: Use atomic check in triggerPlaylistGenerationIfComplete**

In `internal/downloader/handlers.go`, modify `triggerPlaylistGenerationIfComplete()`:

```go
func (h *TrackJobHandler) triggerPlaylistGenerationIfComplete(parentJobID string, logger *slog.Logger) {
    parentJob, err := h.Repo.GetJob(parentJobID)
    if err != nil || parentJob.Status != domain.JobStatusCompleted {
        return
    }
    
    count, err := h.Repo.CountPendingTracksByParentJobID(parentJobID)
    if err != nil || count > 0 {
        return
    }
    
    // CRITICAL: Only one goroutine can generate M3U
    claimed, err := h.Repo.TrySetM3UGenerating(parentJobID)
    if err != nil || !claimed {
        return // Another goroutine is generating, skip
    }
    
    // Ensure we clear the flag even on error
    defer h.Repo.ClearM3UGenerating(parentJobID)
    
    // Generate M3U
    if parentJob.Type == domain.JobTypePlaylist {
        pl := h.buildPlaylistFromJob(parentJob)
        if err := h.PlaylistGenerator.Generate(pl, h.lookupTrack); err != nil {
            logger.Error("M3U generation failed", "job", parentJobID, "err", err)
        }
    }
}
```

**Step 6: Run tests to verify they pass**

Run: `go test ./internal/downloader/... -run TestM3UGenerationRaceCondition -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/store/schema.go internal/store/jobs.go internal/downloader/handlers.go
git commit -m "fix: prevent race condition in M3U generation with advisory lock"
```

---

## Task 4: Wrap Decomposition in Transaction

**Files:**
- Modify: `internal/store/jobs.go` - add `CreateJobBatch` with transaction
- Modify: `internal/store/tracks.go` - add `CreateTrackBatch` with transaction
- Modify: `internal/downloader/handlers.go` - use batch creation in `createTracksAndJobs`
- Test: `internal/store/jobs_test.go`

**Step 1: Write failing test for batch job creation**

Add to `internal/store/jobs_test.go`:
```go
func TestCreateJobBatch(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    jobs := []*domain.Job{
        {ID: "job-1", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "t1"},
        {ID: "job-2", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "t2"},
        {ID: "job-3", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "t3"},
    }

    err := db.CreateJobBatch(jobs)
    require.NoError(t, err)

    for _, j := range jobs {
        retrieved, err := db.GetJob(j.ID)
        require.NoError(t, err)
        assert.Equal(t, j.SourceID, retrieved.SourceID)
    }
}

func TestCreateJobBatchAtomicity(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Job-2 will fail (duplicate constraint on SourceID if we add unique)
    jobs := []*domain.Job{
        {ID: "job-a", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "unique1"},
        {ID: "job-b", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "unique1"}, // duplicate
    }

    err := db.CreateJobBatch(jobs)
    require.Error(t, err) // Should fail

    // Verify neither was created (atomicity)
    _, err1 := db.GetJob("job-a")
    _, err2 := db.GetJob("job-b")
    assert.Error(t, err1) // Should not exist
    assert.Error(t, err2)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/store/... -run TestCreateJobBatch -v`
Expected: FAIL - CreateJobBatch not defined

**Step 3: Implement CreateJobBatch with transaction**

In `internal/store/jobs.go`, add:
```go
func (db *DB) CreateJobBatch(jobs []*domain.Job) error {
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    for _, job := range jobs {
        if job.CreatedAt.IsZero() {
            job.CreatedAt = time.Now()
        }
        if job.UpdatedAt.IsZero() {
            job.UpdatedAt = time.Now()
        }

        _, err := tx.Exec(`
            INSERT INTO jobs (id, type, status, source_id, parent_job_id, progress, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
            job.ID, job.Type, job.Status, job.SourceID, job.ParentJobID, job.Progress,
            job.CreatedAt, job.UpdatedAt)
        if err != nil {
            return fmt.Errorf("failed to create job %s: %w", job.ID, err)
        }
    }

    return tx.Commit()
}
```

**Step 4: Implement CreateTrackBatch with transaction**

In `internal/store/tracks.go`, add similar function:
```go
func (db *DB) CreateTrackBatch(tracks []*domain.Track) error {
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()

    for _, track := range tracks {
        if track.CreatedAt.IsZero() {
            track.CreatedAt = time.Now()
        }
        if track.UpdatedAt.IsZero() {
            track.UpdatedAt = time.Now()
        }

        // Use existing CreateTrack logic but within transaction
        _, err := tx.Exec(`
            INSERT INTO tracks (id, provider_id, album_id, release_id, recording_id, title, artist, album, 
                album_artist, track_number, disc_number, year, duration, status, parent_job_id, 
                created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
            track.ID, track.ProviderID, track.AlbumID, track.ReleaseID, track.RecordingID,
            track.Title, track.Artist, track.Album, track.AlbumArtist, track.TrackNumber,
            track.DiscNumber, track.Year, track.Duration, track.Status, track.ParentJobID,
            track.CreatedAt, track.UpdatedAt)
        if err != nil {
            return fmt.Errorf("failed to create track %s: %w", track.ID, err)
        }
    }

    return tx.Commit()
}
```

**Step 5: Update createTracksAndJobs to use batch operations**

In `internal/downloader/handlers.go`, refactor `createTracksAndJobs()`:

```go
func (h *ContainerJobHandler) createTracksAndJobs(parentJobID string, catalogTracks []domain.CatalogTrack, logger *slog.Logger) int {
    var tracks []*domain.Track
    var jobs []*domain.Job
    created := 0

    for _, catalogTrack := range catalogTracks {
        if downloaded, _ := h.Repo.IsTrackDownloaded(catalogTrack.ID); downloaded && !forceDownload {
            continue
        }

        trackID := uuid.New().String()
        track := &domain.Track{
            ID:          trackID,
            ProviderID:  catalogTrack.ID,
            ParentJobID: parentJobID,
            Status:      domain.TrackStatusQueued,
            Title:       catalogTrack.Title,
            Artist:      catalogTrack.Artist,
            Album:       catalogTrack.Album,
        }
        tracks = append(tracks, track)

        job := &domain.Job{
            ID:          uuid.New().String(),
            Type:        domain.JobTypeTrack,
            Status:      domain.JobStatusQueued,
            SourceID:    catalogTrack.ID,
            ParentJobID: parentJobID,
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        }
        jobs = append(jobs, job)
    }

    // Batch insert within transaction
    if len(tracks) > 0 {
        if err := h.Repo.CreateTrackBatch(tracks); err != nil {
            logger.Error("failed to create tracks batch", "err", err)
        } else {
            created += len(tracks)
        }
    }

    if len(jobs) > 0 {
        if err := h.Repo.CreateJobBatch(jobs); err != nil {
            logger.Error("failed to create jobs batch", "err", err)
        } else {
            created += len(jobs)
        }
    }

    return created
}
```

**Step 6: Run tests to verify they pass**

Run: `go test ./internal/store/... -run TestCreateJobBatch -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/store/jobs.go internal/store/tracks.go internal/downloader/handlers.go
git commit -m "feat: add batch job/track creation with transaction for atomicity"
```

---

## Task 5: Add Cancellation Propagation

**Files:**
- Modify: `internal/app/job_service.go` - add `CancelContainerJob` that cancels children
- Modify: `internal/store/jobs.go` - add `CancelJobsByParentID`
- Test: `internal/app/job_service_test.go` (create if missing)

**Step 1: Write failing test for cancellation propagation**

Add to `internal/app/job_service_test.go`:
```go
func TestCancelContainerJobCancelsChildren(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    service := NewJobService(db, nil, nil)

    // Create container job
    parentID := "container-1"
    db.CreateJob(&domain.Job{ID: parentID, Type: domain.JobTypeAlbum, Status: domain.JobStatusDecomposed, SourceID: "album-1"})

    // Create child jobs
    db.CreateJob(&domain.Job{ID: "child-1", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "t1", ParentJobID: parentID})
    db.CreateJob(&domain.Job{ID: "child-2", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "t2", ParentJobID: parentID})

    // Cancel parent
    err := service.CancelJob(parentID)
    require.NoError(t, err)

    // Verify children are cancelled
    child1, _ := db.GetJob("child-1")
    child2, _ := db.GetJob("child-2")
    assert.Equal(t, domain.JobStatusCancelled, child1.Status)
    assert.Equal(t, domain.JobStatusCancelled, child2.Status)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app/... -run TestCancelContainerJobCancelsChildren -v`
Expected: FAIL

**Step 3: Add CancelJobsByParentID to store**

In `internal/store/jobs.go`, add:
```go
func (db *DB) CancelJobsByParentID(parentID string) error {
    _, err := db.Exec(`
        UPDATE jobs 
        SET status = ?, updated_at = ? 
        WHERE parent_job_id = ? AND status IN (?, ?)`,
        JobStatusCancelled, time.Now(), parentID, JobStatusQueued, JobStatusRunning)
    return err
}
```

**Step 4: Update CancelJob in job service**

In `internal/app/job_service.go`, modify `CancelJob()`:
```go
func (s *JobService) CancelJob(id string) error {
    job, err := s.Repo.GetJob(id)
    if err != nil {
        return fmt.Errorf("failed to get job: %w", err)
    }

    // If container job, cancel all children first
    if job.Type != domain.JobTypeTrack && job.Status != domain.JobStatusCompleted {
        if err := s.Repo.CancelJobsByParentID(id); err != nil {
            return fmt.Errorf("failed to cancel child jobs: %w", err)
        }
    }

    return s.Repo.UpdateJobStatus(id, domain.JobStatusCancelled, 0)
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/app/... -run TestCancelContainerJobCancelsChildren -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/app/job_service.go internal/store/jobs.go
git commit -m "feat: propagate cancellation to child jobs"
```

---

## Task 6: Update DOMAIN.md Documentation

**Files:**
- Modify: `DOMAIN.md`

**Step 1: Update DOMAIN.md with new job status**

In `DOMAIN.md`, update the Job status machine section:

```markdown
Status machine:
```
queued → running → decomposed → completed | failed | cancelled
```

Note: `decomposed` is set when a container job (album/playlist/artist) has created its child track jobs. The container job remains in `decomposed` status until all children complete, at which point it transitions to `completed`. Progress is aggregated from child job completion counts.
```

**Step 2: Add ParentJobID to Job structure**

Update the Job Structure section:
```markdown
Structure:
- Minimal fields: ID, Type, Status, SourceID, Progress, Error, timestamps, ParentJobID
- `SourceID` links to Track.ProviderID
- `ParentJobID` links container jobs to their child track jobs (for cancellation/progress tracking)
```

**Step 3: Add M3U Generation section**

Add new section before SearchResult:
```markdown
## Playlist Generation

M3U playlist files are generated after all tracks in a playlist download complete. Generation is protected by an advisory lock (`m3u_generating` flag on the job record) to prevent race conditions when multiple track jobs complete simultaneously.

Location: `<DownloadsDir>/playlists/<sanitized_title>_<id>.m3u`

Format: Extended M3U with `#EXTM3U`, `#PLAYLIST:`, and `#EXTINF:` tags containing track duration and metadata.
```

**Step 4: Update Worker section with new behavior**

```markdown
## Worker
Executes jobs asynchronously with concurrency control.

Workers:
- Poll for queued jobs at regular intervals
- Process jobs with configurable max concurrency
- Handle job lifecycle: running → download → tagging → completion
- Decompose container jobs (album/playlist/artist) into track records + child jobs atomically
- Use batch operations with transactions for decomposition
- Look up Track metadata for downloads (no duplicate provider calls)
- Update Track status throughout lifecycle (missing → queued → downloading → downloaded → processing → completed)
- Update parent container job progress based on child job completion
- Mark container job completed only when all children finish
- Recover interrupted tracks on startup (reset downloading/processing to queued)
- Verify file hash for idempotent downloads (skip if file exists and hash matches)
- Recompute album state after track completion
- Recover from panics gracefully

Workers never decide business rules. They only execute service instructions.
```

**Step 5: Commit**

```bash
git add DOMAIN.md
git commit -m "docs: update DOMAIN.md with decomposed status, M3U generation, and job hierarchy"
```

---

## Task 7: Final Verification

**Step 1: Run full test suite**

Run: `go test ./... -race`
Expected: All tests pass with no race conditions detected

**Step 2: Run linter**

Run: `golangci-lint run`
Expected: No errors or warnings

**Step 3: Build**

Run: `go build -o navidrums ./cmd/server`
Expected: Successful binary compilation

**Step 4: Final commit (if changes needed)**

```bash
git add -A
git commit -m "chore: verify full test suite passes after architecture improvements"
```

---

## Summary of Changes by File

| File | Changes |
|------|---------|
| `internal/domain/models.go` | Added `JobStatusDecomposed`, `ParentJobID` to Job |
| `internal/store/schema.go` | Added `parent_job_id`, `m3u_generating` columns |
| `internal/store/jobs.go` | Added `CountJobsForParent`, `UpdateJobProgress`, `TrySetM3UGenerating`, `ClearM3UGenerating`, `CancelJobsByParentID`, `CreateJobBatch` |
| `internal/store/tracks.go` | Added `CreateTrackBatch` |
| `internal/downloader/handlers.go` | Refactored container job handling, added progress aggregation, atomic M3U generation |
| `internal/app/job_service.go` | Added cancellation propagation |
| `DOMAIN.md` | Updated documentation |

---

## Execution Options

**Plan complete and saved to `docs/plans/2026-03-18-architecture-improvements.md`. Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
