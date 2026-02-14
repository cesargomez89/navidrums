# Bug Fix Prompts for Navidrums

> **Context Required:** @AGENTS.md
> **Optional:** @DOMAIN.md for job lifecycle details, @ARCHITECTURE.md for file locations

## Job Stuck in Downloading State

A job is stuck in `downloading` status and never completes or fails.

Reproduction steps:
1. Start download of {album/playlist/track}
2. {specific action that triggers the bug}
3. Job remains in `downloading` state indefinitely

Expected behavior:
- Job should transition to `completed` or `failed`
- If download fails, should be retryable via `/htmx/retry/{id}`

Investigation areas:
- `internal/downloader/worker.go` - job state machine
- `internal/app/job_service.go` - status transitions
- `internal/store/job_repository.go` - persistence logic

Check:
1. Is the worker recovering from panics properly?
2. Are context cancellations being handled?
3. Is the state transition being persisted to database?

---

## Duplicate Downloads

The same track is being downloaded multiple times even though it exists in the database.

Expected behavior:
- Check `downloads` table before downloading
- Skip already downloaded tracks
- Log "already downloaded, skipping"

Actual behavior:
- Track is downloaded again
- Duplicate files created (possibly with different filenames)

Investigation:
- `internal/app/downloader.go` - `checkDownloaded()` method
- `internal/store/download_repository.go` - `GetByTrackID()` or similar
- Provider ID matching logic

Database query to check:
```sql
SELECT * FROM downloads WHERE provider_id = '{track_id}';
```

---

## Container Job Not Creating Child Jobs

When downloading an album/playlist/artist, the container job completes but no track jobs are created.

Expected flow:
```
queued → resolving_tracks → (create child track jobs) → completed
```

Actual flow:
```
queued → resolving_tracks → completed (no children created)
```

Files to check:
- `internal/downloader/worker.go` - `processContainerJob()`
- `internal/app/job_service.go` - `CreateTrackJobs()`
- `internal/catalog/provider.go` - track resolution

Debug questions:
1. Is the provider returning the track list?
2. Are we logging the number of tracks found?
3. Is there a transaction rollback happening?

---

## Database Locked Error

Getting "database is locked" errors under concurrent load.

Error:
```
database is locked (5) (SQLITE_BUSY)
```

Context:
- Multiple workers polling and updating jobs simultaneously
- SQLite with default configuration

Solutions to investigate:
1. Add `?cache=shared&mode=rwc` to connection string
2. Enable WAL mode: `PRAGMA journal_mode=WAL;`
3. Add busy timeout: `PRAGMA busy_timeout=5000;`
4. Implement connection pooling or mutex for writes

Files:
- `internal/store/db.go` - connection setup
- `internal/store/job_repository.go` - update operations

---

## Tagging Fails But Job Reports Success

Audio file is downloaded but metadata tags are not written, yet job shows as `completed`.

Expected:
- Track file downloaded
- Metadata tags written to file
- Album art embedded

Actual:
- File exists but has no metadata
- Job status is `completed`

Investigation:
- `internal/tagging/tag_writer.go` - error handling
- `internal/downloader/worker.go` - tagging step in workflow
- Check if tagging errors are being caught and logged but not failing the job

Fix requirements:
1. Tagging failure should fail the job
2. Or tagging failure should be logged as warning but job continues
3. Must be consistent - decide and implement
