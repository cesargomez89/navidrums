package store

import (
	"os"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func TestDB_Jobs(t *testing.T) {
	// Setup
	tmpFile := "test.db"
	db, err := NewSQLiteDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(tmpFile)
	}()

	// Test CreateJob
	job := &domain.Job{
		ID:        "123",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusQueued,
		SourceID:  "track_123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Title:     "Test Job",
		Artist:    "Test Artist",
	}

	err = db.CreateJob(job)
	if err != nil {
		t.Errorf("CreateJob failed: %v", err)
	}

	// Test GetJob
	fetched, err := db.GetJob("123")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if fetched.ID != job.ID {
		t.Errorf("Expected ID %s, got %s", job.ID, fetched.ID)
	}
	if fetched.Status != domain.JobStatusQueued {
		t.Errorf("Expected status %s, got %s", domain.JobStatusQueued, fetched.Status)
	}

	// Test UpdateJobStatus
	err = db.UpdateJobStatus("123", domain.JobStatusDownloading, 50.0)
	if err != nil {
		t.Errorf("UpdateJobStatus failed: %v", err)
	}

	fetched, _ = db.GetJob("123")
	if fetched.Status != domain.JobStatusDownloading {
		t.Errorf("Expected status %s, got %s", domain.JobStatusDownloading, fetched.Status)
	}
	if fetched.Progress != 50.0 {
		t.Errorf("Expected progress 50.0, got %f", fetched.Progress)
	}

	// Test ListActiveJobs
	list, err := db.ListActiveJobs()
	if err != nil {
		t.Errorf("ListActiveJobs failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 active job, got %d", len(list))
	}
}
