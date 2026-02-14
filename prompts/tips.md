# General Tips Prompts for Navidrums

> **Context Required:** @AGENTS.md
> **Optional:** @ARCHITECTURE.md for detailed layer info

## Optimize Database Queries

Identify and optimize slow database queries.

Common issues to look for:
- Missing indexes on foreign keys (job.status, download.provider_id)
- N+1 queries in loops
- Full table scans
- SELECT * when only need specific columns

Tools:
```bash
# Enable query logging
sqlite3 navidrums.db ".trace stdout"

# Analyze query plan
EXPLAIN QUERY PLAN SELECT * FROM jobs WHERE status = 'downloading';
```

Add indexes:
```sql
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_downloads_provider ON downloads(provider_id);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);
```

Verify with benchmarks before/after.

---

## Add Structured Logging

Improve logging throughout the codebase.

Requirements:
- Use `internal/logger` package
- Contextual fields (job_id, track_id, etc.)
- Log levels: debug, info, warn, error
- Structured format (JSON in production)

Patterns:
```go
// Good
logger.Info("starting download",
    "job_id", job.ID,
    "track", track.Title,
    "provider", provider.Name)

// Bad
log.Printf("Starting download of %s", track.Title)
```

Add logging to:
- Job state transitions
- Provider API calls (with timing)
- File operations
- Configuration loading

---

## Implement Graceful Shutdown

Add graceful shutdown handling for clean exits.

Requirements:
- Stop accepting new jobs
- Wait for active downloads to complete (with timeout)
- Close database connections
- Flush logs
- Exit code 0 on clean shutdown

Implementation in `cmd/server/main.go`:
```go
func main() {
    // ... setup
    
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    
    <-quit
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        logger.Error("server shutdown error", "error", err)
    }
    
    worker.Stop()
    db.Close()
}
```

---

## Add Configuration Validation

Validate configuration on startup and fail fast.

Check:
- Required fields are set
- URLs are valid
- Paths exist and are writable
- Database connection works
- Provider is reachable

Pattern:
```go
func (c *Config) Validate() error {
    if c.DownloadDir == "" {
        return errors.New("DOWNLOADS_DIR is required")
    }
    
    if _, err := url.Parse(c.ProviderURL); err != nil {
        return fmt.Errorf("invalid PROVIDER_URL: %w", err)
    }
    
    // Test write permissions
    testFile := filepath.Join(c.DownloadDir, ".write_test")
    if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
        return fmt.Errorf("downloads directory not writable: %w", err)
    }
    os.Remove(testFile)
    
    return nil
}
```

Call in `main()` before starting server.

---

## Document Public APIs

Add Go doc comments to all exported types and functions.

Standard format:
```go
// JobService manages the lifecycle of download jobs.
// It handles job creation, status updates, and querying.
type JobService struct {
    repo JobRepository
    // ...
}

// EnqueueJob creates a new download job and adds it to the queue.
// Returns the created job or an error if validation fails.
// The job will be in "queued" status initially.
func (s *JobService) EnqueueJob(ctx context.Context, req EnqueueRequest) (*Job, error) {
    // ...
}
```

Generate docs:
```bash
go doc -all ./internal/app
```

---

## Add Health Check Endpoint

Implement health check for monitoring and load balancers.

Endpoint: `GET /health`

Response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "checks": {
    "database": "ok",
    "provider": "ok",
    "disk_space": "ok"
  }
}
```

Implementation:
```go
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
    checks := map[string]string{}
    
    // Check database
    if err := h.db.Ping(); err != nil {
        checks["database"] = "error: " + err.Error()
        w.WriteHeader(http.StatusServiceUnavailable)
    } else {
        checks["database"] = "ok"
    }
    
    // Check disk space
    // ...
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "healthy",
        "checks": checks,
    })
}
```

---

## Implement Request Logging Middleware

Add middleware to log all HTTP requests.

Requirements:
- Log method, path, status code, duration
- Log user agent and IP
- Skip logging for health checks (optional)
- Structured format

Implementation:
```go
func LoggingMiddleware(logger *logger.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
            next.ServeHTTP(wrapped, r)
            
            logger.Info("http request",
                "method", r.Method,
                "path", r.URL.Path,
                "status", wrapped.statusCode,
                "duration", time.Since(start),
                "ip", r.RemoteAddr,
            )
        })
    }
}
```

Apply to all routes in router setup.
