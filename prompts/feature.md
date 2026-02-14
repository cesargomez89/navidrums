# Feature Implementation Prompts for Navidrums

> **Context Required:** @AGENTS.md, @ARCHITECTURE.md
> **Optional:** @DOMAIN.md for data models, @API.md for endpoints

## Implement New Download Type

Add support for downloading `{new_type}` (e.g., podcasts, audiobooks).

Implementation order (follow strictly):

1. **Domain** (`internal/domain/`)
   - Add `{NewType}` struct with fields
   - Add to `SearchResult` if searchable
   - Define constants for type

2. **Provider Interface** (`internal/catalog/`)
   - Add `Get{NewType}()` method to Provider interface
   - Implement in concrete provider
   - Add search support if applicable

3. **Service** (`internal/app/`)
   - Add `{NewType}Service` for business logic
   - Handle download orchestration
   - Add validation

4. **Repository** (`internal/store/`)
   - Add storage methods if new entities needed
   - Update job repository for new type

5. **Worker** (`internal/downloader/`)
   - Add handling for `{new_type}` job type
   - Decompose into track jobs if applicable
   - Or handle as atomic download

6. **Handler** (`internal/http/`)
   - Add endpoint: `GET /{new_type}/{id}`
   - Add HTMX fragment: `POST /htmx/download/{new_type}/{id}`
   - Add template rendering

7. **UI** (`web/`)
   - Add detail page template
   - Add search result template

---

## Add Retry with Exponential Backoff

Implement retry logic with exponential backoff for failed downloads.

Requirements:
- Max retries: 3
- Backoff: 1s, 2s, 4s (exponential)
- Only retry certain errors (network errors, 5xx from provider)
- Don't retry: 4xx errors, disk full, cancelled jobs
- Track retry count in job metadata

Implementation:
1. Add retry fields to `domain.Job`:
   ```go
   RetryCount int
   LastError  string
   ```

2. Update `internal/downloader/worker.go`:
   - Check retry count before failing
   - Reschedule job with backoff
   - Update retry metadata

3. Update `internal/store/job_repository.go`:
   - Add `IncrementRetryCount(jobID)`
   - Add `RescheduleJob(jobID, nextAttempt)`

4. UI updates:
   - Show retry count in queue
   - Show "will retry in X seconds"

---

## Implement Concurrent Download Limits

Add per-source concurrent download limits.

Requirements:
- Global limit: N concurrent downloads (existing)
- Per-source limit: M concurrent from same provider
- Track active downloads by source
- Queue jobs if source limit reached

Implementation:

1. **Config** (`internal/config/`)
   ```go
   MaxConcurrentDownloads int // global
   MaxConcurrentPerSource int // per provider
   ```

2. **Worker** (`internal/downloader/worker.go`)
   - Add `sourceSemaphores map[string]chan struct{}`
   - Check source limit before starting download
   - Release on completion/error/cancel

3. **Database** (`internal/store/`)
   - Track `source` field on active jobs
   - Query to count active by source

---

## Add Download Statistics Dashboard

Create a stats page showing download metrics.

Requirements:
- Total downloads (all time)
- Downloads today/this week/this month
- Success/failure rates
- Average download time
- Storage usage by artist/album

Implementation:

1. **Domain** (`internal/domain/`)
   ```go
   type DownloadStats struct {
       TotalCount      int
       SuccessCount    int
       FailedCount     int
       AverageDuration time.Duration
       ByPeriod        map[string]int
       StorageByArtist map[string]int64
   }
   ```

2. **Repository** (`internal/store/`)
   - `GetDownloadStats(since time.Time)`
   - `GetStorageStats()`

3. **Service** (`internal/app/`)
   - `StatsService` to aggregate data
   - Caching for expensive queries

4. **Handler** (`internal/http/`)
   - `GET /stats` - main page
   - `GET /htmx/stats/summary` - fragment for dashboard

5. **UI** (`web/`)
   - Stats page template with charts
   - HTMX polling for real-time updates

---

## Implement Job Priority Queue

Add priority levels to job processing.

Requirements:
- Priority levels: high, normal, low
- Higher priority jobs processed first
- Maintain FIFO within same priority
- Allow priority change via UI/API

Implementation:

1. **Domain** (`internal/domain/job.go`)
   ```go
   type JobPriority int
   const (
       PriorityLow JobPriority = iota
       PriorityNormal
       PriorityHigh
   )
   
   type Job struct {
       // ... existing fields
       Priority JobPriority
   }
   ```

2. **Repository** (`internal/store/job_repository.go`)
   - Update `GetQueuedJobs()` to order by priority DESC, created_at ASC
   - Add `UpdateJobPriority(jobID, priority)`

3. **Service** (`internal/app/job_service.go`)
   - Add priority parameter to `EnqueueJob()`
   - Default to normal priority

4. **Handler** (`internal/http/`)
   - Add priority to download POST endpoint
   - Add `POST /htmx/job/{id}/priority` to change priority

5. **UI** (`web/`)
   - Priority selector in download confirmation
   - Priority indicator in queue list

---

## Add Export/Import Functionality

Allow exporting download history and importing on new instance.

Requirements:
- Export: JSON with all downloads metadata
- Import: Restore downloads table from JSON
- Skip existing downloads (by provider_id)
- Progress indicator for large imports

Implementation:

1. **Service** (`internal/app/`)
   ```go
   type ExportService struct {
       downloadRepo DownloadRepository
   }
   
   func (s *ExportService) Export(w io.Writer) error
   func (s *ExportService) Import(r io.Reader) (ImportResult, error)
   ```

2. **Handler** (`internal/http/`)
   - `GET /api/export` - download JSON file
   - `POST /api/import` - upload and import
   - HTMX progress endpoint for import status

3. **Format**:
   ```json
   {
     "version": "1.0",
     "exported_at": "2024-01-15T10:30:00Z",
     "downloads": [
       {
         "provider_id": "track123",
         "title": "Song Name",
         "artist": "Artist Name",
         "downloaded_at": "2024-01-10T08:00:00Z",
         "file_path": "/downloads/Artist/Album/Track.flac"
       }
     ]
   }
   ```

---

## Implement Webhook Notifications

Send webhook notifications on job completion/failure.

Requirements:
- Configurable webhook URL
- Events: job_completed, job_failed
- Retry failed webhook deliveries
- Signature verification (HMAC)

Implementation:

1. **Config** (`internal/config/`)
   ```go
   WebhookURL      string
   WebhookSecret   string
   WebhookEvents   []string
   ```

2. **Domain** (`internal/domain/`)
   ```go
   type WebhookEvent struct {
       Type      string          // job.completed, job.failed
       Timestamp time.Time
       Data      json.RawMessage
       Signature string
   }
   ```

3. **Service** (`internal/app/`)
   ```go
   type WebhookService struct {
       url    string
       secret string
       client *http.Client
   }
   
   func (s *WebhookService) Send(event WebhookEvent)
   ```

4. **Worker** (`internal/downloader/worker.go`)
   - Call webhook service on job completion/failure
   - Don't block on webhook (async)
   - Log webhook errors

5. **Settings UI** (`web/`)
   - Webhook configuration form
   - Test webhook button
   - Event type selection
