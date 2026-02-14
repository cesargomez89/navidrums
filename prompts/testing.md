# Testing Prompts for Navidrums

> **Context Required:** @AGENTS.md
> **Optional:** @DOMAIN.md for test data structures

## Unit Test for Service Method

Write unit tests for the `{MethodName}` method in `internal/app/{service}.go`.

Requirements:
- Test happy path (success case)
- Test error cases (not found, validation error, repository error)
- Use mock repository (create if needed)
- Follow Go testing conventions
- Run with: `go test ./internal/app/...`

Test structure:
```go
func Test{Service}_{Method}_Success(t *testing.T) {
    // Arrange
    mockRepo := &mockRepository{}
    service := New{Service}(mockRepo)
    
    // Act
    result, err := service.Method(ctx, input)
    
    // Assert
    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
    // ... more assertions
}
```

Coverage target: {X}%

---

## Integration Test for Download Flow

Create an integration test for the full download job lifecycle.

Test scenario:
1. Enqueue a track download job
2. Worker picks up the job
3. Verify job transitions: `queued → resolving_tracks → downloading → completed`
4. Verify file exists on disk
5. Verify download recorded in database

Setup requirements:
- Use temp directory for downloads (`t.TempDir()`)
- Use in-memory or temp file SQLite database
- Mock provider responses (don't hit real API)

Files involved:
- `internal/app/job_service.go`
- `internal/downloader/worker.go`
- `internal/store/job_repository.go`
- `internal/storage/file_storage.go`

Run:
```bash
go test -v ./internal/downloader/... -run TestDownloadFlow
```

---

## Add Race Detection Tests

Add tests with race detector for concurrent job processing.

Scenarios to test:
1. Multiple workers picking up same job (should not happen with proper locking)
2. Cancelling job while downloading
3. Retrying failed job while another worker processing

Run with race detector:
```bash
go test -race ./internal/downloader/...
go test -race ./internal/app/...
```

Fix any races found:
- Job status updates should be atomic
- Download tracking needs mutex or DB constraints
- Provider cache (if any) needs synchronization

---

## Test Handler Endpoints

Write HTTP handler tests for `{Endpoint}` in `internal/http/{handler}.go`.

Test cases:
1. Valid request returns expected HTML/JSON
2. Invalid input returns 400 Bad Request
3. Not found returns 404
4. Service error returns 500
5. HTMX headers handled correctly

Use `httptest` package:
```go
func Test{Handler}_{Endpoint}(t *testing.T) {
    req := httptest.NewRequest("POST", "/htmx/download/track/123", nil)
    rr := httptest.NewRecorder()
    
    handler.ServeHTTP(rr, req)
    
    if rr.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rr.Code)
    }
    
    // Check response body contains expected HTML
}
```

Mock service layer - don't use real services.

---

## Test Database Constraints

Add tests to verify database constraints prevent invalid data.

Test cases:
1. Duplicate provider_id in downloads table should fail
2. Job status must be valid value
3. Foreign key constraints work (if enabled)
4. Required fields cannot be null

Use `internal/store` test helpers:
```go
func TestDownloadRepository_DuplicatePrevention(t *testing.T) {
    db := setupTestDB(t)
    repo := NewDownloadRepository(db)
    
    // Insert first download
    err := repo.Save(&domain.Download{ProviderID: "track123"})
    if err != nil {
        t.Fatal(err)
    }
    
    // Try to insert duplicate - should fail
    err = repo.Save(&domain.Download{ProviderID: "track123"})
    if err == nil {
        t.Error("expected error for duplicate download")
    }
}
```

---

## Benchmark Critical Path

Add benchmark tests for performance-critical operations.

Target areas:
- Job polling in worker
- Provider search queries
- Database queries with many jobs

Example:
```go
func BenchmarkWorker_PollJobs(b *testing.B) {
    db := setupBenchmarkDB(b)
    // Insert 1000 queued jobs
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = worker.PollJobs(ctx, 10)
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./internal/...
```

Identify bottlenecks:
- Missing database indexes
- N+1 queries
- Inefficient in-memory operations
