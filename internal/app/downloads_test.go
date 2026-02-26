package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
)

func TestDownloadsService_EnqueueSyncFileJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	track := &domain.Track{
		ProviderID: "sync_file_test",
		Title:      "Track",
		Artist:     "Artist",
		Album:      "Album",
		Status:     domain.TrackStatusCompleted,
		FilePath:   "/path/track.flac",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.CreateTrack(track); err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	err := svc.EnqueueSyncFileJob("sync_file_test")
	if err != nil {
		t.Fatalf("EnqueueSyncFileJob failed: %v", err)
	}

	job, err := db.GetActiveJobBySourceID("sync_file_test", domain.JobTypeSyncFile)
	if err != nil {
		t.Fatalf("GetActiveJobBySourceID failed: %v", err)
	}
	if job == nil {
		t.Fatal("Expected job to be created")
	}
	if job.Type != domain.JobTypeSyncFile {
		t.Errorf("Expected job type %s, got %s", domain.JobTypeSyncFile, job.Type)
	}
	if job.Status != domain.JobStatusQueued {
		t.Errorf("Expected status %s, got %s", domain.JobStatusQueued, job.Status)
	}
}

func TestDownloadsService_EnqueueSyncMetadataJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	track := &domain.Track{
		ProviderID: "sync_metadata_test",
		Title:      "Track",
		Artist:     "Artist",
		Album:      "Album",
		Status:     domain.TrackStatusCompleted,
		FilePath:   "/path/track.flac",
		ISRC:       "USABC1234567",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.CreateTrack(track); err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	err := svc.EnqueueSyncMetadataJob("sync_metadata_test")
	if err != nil {
		t.Fatalf("EnqueueSyncMetadataJob failed: %v", err)
	}

	job, err := db.GetActiveJobBySourceID("sync_metadata_test", domain.JobTypeSyncMusicBrainz)
	if err != nil {
		t.Fatalf("GetActiveJobBySourceID failed: %v", err)
	}
	if job == nil {
		t.Fatal("Expected job to be created")
	}
	if job.Type != domain.JobTypeSyncMusicBrainz {
		t.Errorf("Expected job type %s, got %s", domain.JobTypeSyncMusicBrainz, job.Type)
	}
	if job.Status != domain.JobStatusQueued {
		t.Errorf("Expected status %s, got %s", domain.JobStatusQueued, job.Status)
	}
}

func TestDownloadsService_ListDownloads(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	// Create completed tracks
	tracks := []*domain.Track{
		{ProviderID: "dl_1", Title: "Download 1", Artist: "Artist", Album: "Album", Status: domain.TrackStatusCompleted, FilePath: "/path/1.flac", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "dl_2", Title: "Download 2", Artist: "Artist", Album: "Album", Status: domain.TrackStatusCompleted, FilePath: "/path/2.flac", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "dl_3", Title: "Download 3", Artist: "Artist", Album: "Album", Status: domain.TrackStatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, tr := range tracks {
		if err := db.CreateTrack(tr); err != nil {
			t.Fatalf("CreateTrack failed: %v", err)
		}
	}

	// Test ListDownloads - should only return completed
	downloads, _, err := svc.ListDownloads(1, 10)
	if err != nil {
		t.Fatalf("ListDownloads failed: %v", err)
	}
	if len(downloads) != 2 {
		t.Errorf("Expected 2 downloads, got %d", len(downloads))
	}
}

func TestDownloadsService_SearchDownloads(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	// Create completed tracks
	tracks := []*domain.Track{
		{ProviderID: "search_1", Title: "Hello World", Artist: "Artist A", Album: "Album One", Status: domain.TrackStatusCompleted, FilePath: "/path/1.flac", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "search_2", Title: "Goodbye", Artist: "Artist B", Album: "Album Two", Status: domain.TrackStatusCompleted, FilePath: "/path/2.flac", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "search_3", Title: "Hello Again", Artist: "Artist A", Album: "Album Three", Status: domain.TrackStatusCompleted, FilePath: "/path/3.flac", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, tr := range tracks {
		if err := db.CreateTrack(tr); err != nil {
			t.Fatalf("CreateTrack failed: %v", err)
		}
	}

	// Search by title
	results, _, err := svc.SearchDownloads("Hello", 1, 10)
	if err != nil {
		t.Fatalf("SearchDownloads failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'Hello', got %d", len(results))
	}

	// Search by artist
	results, _, err = svc.SearchDownloads("Artist B", 1, 10)
	if err != nil {
		t.Fatalf("SearchDownloads failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Artist B', got %d", len(results))
	}

	// Search by album
	results, _, err = svc.SearchDownloads("Album Two", 1, 10)
	if err != nil {
		t.Fatalf("SearchDownloads failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Album Two', got %d", len(results))
	}

	// No results
	results, _, err = svc.SearchDownloads("Nonexistent", 1, 10)
	if err != nil {
		t.Fatalf("SearchDownloads failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestDownloadsService_DeleteDownload(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create temp directory for test
	tmpDir := t.TempDir()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.flac")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create track in DB pointing to the file
	track := &domain.Track{
		ProviderID: "delete_test",
		Title:      "Delete Me",
		Artist:     "Artist",
		Album:      "Album",
		Status:     domain.TrackStatusCompleted,
		FilePath:   testFile,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.CreateTrack(track); err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	// Create a folder with a file in it (to test empty folder deletion)
	folderPath := filepath.Join(tmpDir, "Album Artist", "2020 - Album")
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	folderFile := filepath.Join(folderPath, "track.flac")
	if err := os.WriteFile(folderFile, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Update track with new path
	track.FilePath = folderFile
	if err := db.UpdateTrack(track); err != nil {
		t.Fatalf("UpdateTrack failed: %v", err)
	}

	// Test DeleteDownload
	err := svc.DeleteDownload("delete_test")
	if err != nil {
		t.Fatalf("DeleteDownload failed: %v", err)
	}

	// Verify track is deleted
	deletedTrack, _ := db.GetTrackByProviderID("delete_test")
	if deletedTrack != nil {
		t.Error("Expected track to be deleted")
	}

	// Test deleting non-existent provider - returns error from DB
	err = svc.DeleteDownload("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent provider")
	}
}

func TestDownloadsService_DeleteDownload_CascadeCleanup(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tmpDir := t.TempDir()
	log := logger.Default()
	svc := NewDownloadsService(db, log)

	artistDir := filepath.Join(tmpDir, "Artist", "2020 - Album")
	if err := os.MkdirAll(artistDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	trackFile := filepath.Join(artistDir, "1-01 Track.flac")
	coverFile := filepath.Join(artistDir, "cover.jpg")

	if err := os.WriteFile(trackFile, []byte("audio"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(coverFile, []byte("image"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	track := &domain.Track{
		ProviderID: "cascade_test",
		Title:      "Track",
		Artist:     "Artist",
		Album:      "Album",
		Status:     domain.TrackStatusCompleted,
		FilePath:   trackFile,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := db.CreateTrack(track); err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	err := svc.DeleteDownload("cascade_test")
	if err != nil {
		t.Fatalf("DeleteDownload failed: %v", err)
	}

	if _, err := os.Stat(trackFile); !os.IsNotExist(err) {
		t.Error("Expected track file to be deleted")
	}
	if _, err := os.Stat(coverFile); !os.IsNotExist(err) {
		t.Error("Expected cover.jpg to be deleted")
	}
	if _, err := os.Stat(artistDir); !os.IsNotExist(err) {
		t.Error("Expected album folder to be deleted")
	}
	artistParent := filepath.Join(tmpDir, "Artist")
	if _, err := os.Stat(artistParent); !os.IsNotExist(err) {
		t.Error("Expected artist folder to be deleted")
	}
}

func TestDownloadsService_EnqueueSyncHiFiJob(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	err := svc.EnqueueSyncHiFiJob("hifi_test")
	if err != nil {
		t.Fatalf("EnqueueSyncHiFiJob failed: %v", err)
	}

	job, err := db.GetActiveJobBySourceID("hifi_test", domain.JobTypeSyncHiFi)
	if err != nil {
		t.Fatalf("GetActiveJobBySourceID failed: %v", err)
	}
	if job == nil {
		t.Fatal("Expected job to be created")
	}
	if job.Type != domain.JobTypeSyncHiFi {
		t.Errorf("Expected job type %s, got %s", domain.JobTypeSyncHiFi, job.Type)
	}
}

func TestDownloadsService_EnqueueSyncJobs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	// Create some completed tracks
	tracks := []*domain.Track{
		{ProviderID: "t1", Title: "T1", Artist: "A", Album: "A", Status: domain.TrackStatusCompleted, FilePath: "/p/1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "t2", Title: "T2", Artist: "A", Album: "A", Status: domain.TrackStatusCompleted, FilePath: "/p/2", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "t3", Title: "T3", Artist: "A", Album: "A", Status: domain.TrackStatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()}, // Should be skipped
	}

	for _, tr := range tracks {
		if err := db.CreateTrack(tr); err != nil {
			t.Fatalf("CreateTrack failed: %v", err)
		}
	}

	// Create an active job for t1 already
	existingJob := &domain.Job{
		ID:       "existing",
		Type:     domain.JobTypeSyncHiFi,
		Status:   domain.JobStatusRunning,
		SourceID: "t1",
	}
	if err := db.CreateJob(existingJob); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Run EnqueueSyncJobs
	count, err := svc.EnqueueSyncJobs()
	if err != nil {
		t.Fatalf("EnqueueSyncJobs failed: %v", err)
	}

	// Should only enqueue for t2 (t1 has active job, t3 is not completed)
	if count != 1 {
		t.Errorf("Expected 1 job enqueued, got %d", count)
	}

	// Verify t2 job exists
	job, _ := db.GetActiveJobBySourceID("t2", domain.JobTypeSyncHiFi)
	if job == nil {
		t.Fatal("Expected sync job for t2")
	}
}
