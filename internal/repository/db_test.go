package repository

import (
	"os"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/models"
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
	job := &models.Job{
		ID:        "123",
		Type:      models.JobTypeTrack,
		Status:    models.JobStatusQueued,
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
	if fetched.Status != models.JobStatusQueued {
		t.Errorf("Expected status %s, got %s", models.JobStatusQueued, fetched.Status)
	}

	// Test UpdateJobStatus
	err = db.UpdateJobStatus("123", models.JobStatusDownloading, 50.0)
	if err != nil {
		t.Errorf("UpdateJobStatus failed: %v", err)
	}

	fetched, _ = db.GetJob("123")
	if fetched.Status != models.JobStatusDownloading {
		t.Errorf("Expected status %s, got %s", models.JobStatusDownloading, fetched.Status)
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

func TestDB_JobItems(t *testing.T) {
	// Setup
	tmpFile := "test_items.db"
	db, err := NewSQLiteDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer func() {
		db.Close()
		os.Remove(tmpFile)
	}()

	// Create Parent Job
	job := &models.Job{ID: "job1", Type: "album", Status: "queued", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.CreateJob(job); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Create Item
	item := &models.JobItem{
		JobID:    "job1",
		TrackID:  "t1",
		Status:   models.JobItemStatusPending,
		Progress: 0,
		Title:    "Track 1",
	}
	err = db.CreateJobItem(item)
	if err != nil {
		t.Errorf("CreateJobItem failed: %v", err)
	}

	// Get Items
	items, err := db.GetJobItems("job1")
	if err != nil {
		t.Errorf("GetJobItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}
	if items[0].TrackID != "t1" {
		t.Errorf("Expected track t1, got %s", items[0].TrackID)
	}
}
