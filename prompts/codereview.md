# Code Review Prompts for Navidrums

> **Context Required:** @AGENTS.md
> **Optional:** @ARCHITECTURE.md for layer responsibilities

## Review Service Layer Changes

Review the changes in `internal/app/{service_file}.go` for compliance with architecture rules.

Checklist:
- [ ] No direct database access (must use repository)
- [ ] No filesystem writes (must use `internal/storage`)
- [ ] Error handling follows: `fmt.Errorf("failed to X: %w", err)`
- [ ] Business logic properly encapsulated
- [ ] No provider calls mixed with persistence logic
- [ ] Exported errors use `Err` prefix

Architecture verification:
```
✓ services → repository (allowed)
✓ services → providers (allowed)
✓ services → storage (allowed)
✗ services → database directly (forbidden)
```

Questions to answer:
1. Can this logic be tested without a real database?
2. Are there any goroutines spawned that should be in workers?
3. Does the error chain provide enough context?

---

## Review Handler Changes

Review the changes in `internal/http/{handler_file}.go` for compliance with handler rules.

Critical checks (any violation = reject):
- [ ] No downloading in handlers
- [ ] No goroutine spawning
- [ ] No direct repository access
- [ ] No provider calls
- [ ] No filesystem writes

Handler responsibilities only:
- [ ] Request parsing and validation
- [ ] Calling service methods
- [ ] HTML/JSON response formatting
- [ ] Template rendering

Verify:
```go
// BAD - direct DB access
job, _ := store.GetJob(id)

// GOOD - through service
job, _ := jobService.GetJob(id)
```

---

## Review Worker Changes

Review changes in `internal/downloader/worker.go` or related worker files.

Worker rules:
- [ ] Recovers from panics with `defer recover()`
- [ ] Respects context cancellation
- [ ] Uses services for business logic (doesn't implement it)
- [ ] Properly updates job status through service layer
- [ ] Handles `resolving_tracks → downloading → completed/failed` lifecycle

Check for:
1. Race conditions in job status updates
2. Proper cleanup on cancellation
3. Logging of state transitions
4. Container job decomposition logic

Example of good worker pattern:
```go
func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
    defer func() {
        if r := recover(); r != nil {
            w.logger.Error("job panic", "error", r)
            w.jobService.FailJob(job.ID, fmt.Errorf("panic: %v", r))
        }
    }()
    
    // Check cancellation
    if err := ctx.Err(); err != nil {
        return
    }
    
    // Delegate to services
    if err := w.downloader.Download(ctx, job); err != nil {
        w.jobService.FailJob(job.ID, err)
    }
}
```

---

## Review Database Schema Changes

Review migration or schema changes in `internal/store/`.

Checklist:
- [ ] Backward compatible (or migration provided)
- [ ] Indexes added for query patterns
- [ ] Foreign keys defined where appropriate
- [ ] Proper column types (TEXT, INTEGER, etc.)
- [ ] NOT NULL constraints where applicable

For Navidrums domain:
- [ ] Job table has proper status enum/check
- [ ] Download table prevents duplicates (unique constraint on provider_id?)
- [ ] Settings table for key-value config

Verify queries:
```sql
-- Check for N+1 queries
-- Check for missing indexes on foreign keys
-- Check for transactions where needed
```

---

## Review Import Organization

Verify imports follow the project standard:

Order:
1. Standard library imports
2. Third-party imports  
3. Internal project imports (github.com/.../internal/...)

With blank lines between groups.

Example:
```go
import (
	"context"
	"fmt"
	"time"

	"github.com/some/lib"

	"github.com/navidrums/internal/app"
	"github.com/navidrums/internal/domain"
)
```

Check:
```bash
go fmt ./...
```

Should produce no changes.
