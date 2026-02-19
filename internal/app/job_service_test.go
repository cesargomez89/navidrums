package app

import (
	"os"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/store"
)

func setupTestDB(t *testing.T) (*store.DB, func()) {
	tmpFile := "test_app.db"
	db, err := store.NewSQLiteDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	cleanup := func() {
		db.Close()
		os.Remove(tmpFile)
	}
	return db, cleanup
}

func TestJobService_EnqueueJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewJobService(db, log)

	// Test enqueue new job
	job, err := svc.EnqueueJob("track_123", domain.JobTypeTrack)
	if err != nil {
		t.Fatalf("EnqueueJob failed: %v", err)
	}
	if job == nil {
		t.Fatal("Expected job to be returned")
	}
	if job.Status != domain.JobStatusQueued {
		t.Errorf("Expected status queued, got %s", job.Status)
	}

	// Test deduplication - enqueue same job again
	existingJob, err := svc.EnqueueJob("track_123", domain.JobTypeTrack)
	if err != nil {
		t.Fatalf("EnqueueJob failed: %v", err)
	}
	if existingJob.ID != job.ID {
		t.Errorf("Expected same job ID %s, got %s", job.ID, existingJob.ID)
	}

	// Test different job type - should create new job
	differentType, err := svc.EnqueueJob("track_123", domain.JobTypeAlbum)
	if err != nil {
		t.Fatalf("EnqueueJob failed: %v", err)
	}
	if differentType.ID == job.ID {
		t.Error("Expected different job ID for different type")
	}
}

func TestJobService_CancelJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewJobService(db, log)

	// Create a running job
	job := &domain.Job{
		ID:        "cancel_test",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusRunning,
		SourceID:  "track_456",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateJob(job)

	// Cancel the job
	err := svc.CancelJob("cancel_test")
	if err != nil {
		t.Errorf("CancelJob failed: %v", err)
	}

	// Verify cancellation
	fetched, _ := db.GetJob("cancel_test")
	if fetched.Status != domain.JobStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", fetched.Status)
	}
}

func TestJobService_RetryJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewJobService(db, log)

	// Create a failed job
	job := &domain.Job{
		ID:        "retry_test",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusFailed,
		SourceID:  "track_789",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateJob(job)

	// Retry the job
	err := svc.RetryJob("retry_test")
	if err != nil {
		t.Errorf("RetryJob failed: %v", err)
	}

	// Verify job is queued again
	fetched, _ := db.GetJob("retry_test")
	if fetched.Status != domain.JobStatusQueued {
		t.Errorf("Expected status queued, got %s", fetched.Status)
	}

	// Test retry non-existent job
	err = svc.RetryJob("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestJobService_RetryJobClearsError(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewJobService(db, log)

	// Create a failed job with error message
	job := &domain.Job{
		ID:        "retry_error_test",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusFailed,
		SourceID:  "track_error",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateJob(job)

	errMsg := "download failed: network error"
	err := db.UpdateJobError("retry_error_test", errMsg)
	if err != nil {
		t.Fatalf("UpdateJobError failed: %v", err)
	}

	// Verify error is set
	fetched, _ := db.GetJob("retry_error_test")
	if fetched.Error == nil || *fetched.Error != errMsg {
		t.Errorf("Expected error %q, got %v", errMsg, fetched.Error)
	}

	// Retry the job
	err = svc.RetryJob("retry_error_test")
	if err != nil {
		t.Errorf("RetryJob failed: %v", err)
	}

	// Verify error is cleared
	fetched, _ = db.GetJob("retry_error_test")
	if fetched.Status != domain.JobStatusQueued {
		t.Errorf("Expected status queued, got %s", fetched.Status)
	}
	if fetched.Error != nil {
		t.Errorf("Expected error to be cleared, got %v", fetched.Error)
	}
	if fetched.Progress != 0 {
		t.Errorf("Expected progress to be 0, got %f", fetched.Progress)
	}
}

func TestJobService_ListJobs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewJobService(db, log)

	// Create multiple jobs
	jobs := []*domain.Job{
		{ID: "job_1", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "s1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job_2", Type: domain.JobTypeAlbum, Status: domain.JobStatusCompleted, SourceID: "s2", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job_3", Type: domain.JobTypeTrack, Status: domain.JobStatusFailed, SourceID: "s3", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, j := range jobs {
		db.CreateJob(j)
	}

	// Test ListJobs
	listed, err := svc.ListJobs()
	if err != nil {
		t.Errorf("ListJobs failed: %v", err)
	}
	if len(listed) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(listed))
	}

	// Test ListActiveJobs
	active, err := svc.ListActiveJobs()
	if err != nil {
		t.Errorf("ListActiveJobs failed: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("Expected 1 active job, got %d", len(active))
	}

	// Test ListFinishedJobs
	finished, err := svc.ListFinishedJobs(10)
	if err != nil {
		t.Errorf("ListFinishedJobs failed: %v", err)
	}
	if len(finished) != 2 {
		t.Errorf("Expected 2 finished jobs, got %d", len(finished))
	}
}

func TestJobService_GetJobStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewJobService(db, log)

	// Create jobs with different statuses
	jobs := []*domain.Job{
		{ID: "stat_1", Type: domain.JobTypeTrack, Status: domain.JobStatusCompleted, SourceID: "s1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "stat_2", Type: domain.JobTypeTrack, Status: domain.JobStatusCompleted, SourceID: "s2", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "stat_3", Type: domain.JobTypeTrack, Status: domain.JobStatusFailed, SourceID: "s3", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "stat_4", Type: domain.JobTypeTrack, Status: domain.JobStatusCancelled, SourceID: "s4", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, j := range jobs {
		db.CreateJob(j)
	}

	// Test GetJobStats
	stats, err := svc.GetJobStats()
	if err != nil {
		t.Errorf("GetJobStats failed: %v", err)
	}
	if stats.Total != 4 {
		t.Errorf("Expected total 4, got %d", stats.Total)
	}
	if stats.Completed != 2 {
		t.Errorf("Expected completed 2, got %d", stats.Completed)
	}

	// Test ClearFinishedJobs
	err = svc.ClearFinishedJobs()
	if err != nil {
		t.Errorf("ClearFinishedJobs failed: %v", err)
	}

	stats, _ = svc.GetJobStats()
	if stats.Total != 0 {
		t.Errorf("Expected total 0 after clear, got %d", stats.Total)
	}
}

func TestJobService_GetJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewJobService(db, log)

	// Create a job
	job := &domain.Job{
		ID:        "get_test",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusQueued,
		SourceID:  "track_get",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateJob(job)

	// Test GetJob
	fetched, err := svc.GetJob("get_test")
	if err != nil {
		t.Errorf("GetJob failed: %v", err)
	}
	if fetched.ID != job.ID {
		t.Errorf("Expected ID %s, got %s", job.ID, fetched.ID)
	}

	// Test GetJob - not found
	_, err = svc.GetJob("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}
