package store

import (
	"os"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	tmpFile := "test.db"
	db, err := NewSQLiteDB(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	cleanup := func() {
		if cErr := db.Close(); cErr != nil {
			t.Logf("db.Close error: %v", cErr)
		}
		if rErr := os.Remove(tmpFile); rErr != nil {
			t.Logf("os.Remove error: %v", rErr)
		}
	}
	return db, cleanup
}

func TestDB_Jobs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test CreateJob
	job := &domain.Job{
		ID:        "123",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusQueued,
		SourceID:  "track_123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := db.CreateJob(job)
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
	err = db.UpdateJobStatus("123", domain.JobStatusRunning, 50.0)
	if err != nil {
		t.Errorf("UpdateJobStatus failed: %v", err)
	}

	fetched, _ = db.GetJob("123")
	if fetched.Status != domain.JobStatusRunning {
		t.Errorf("Expected status %s, got %s", domain.JobStatusRunning, fetched.Status)
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

func TestDB_Tracks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	track := &domain.Track{
		ProviderID:   "track_123",
		Title:        "Test Track",
		Artist:       "Test Artist",
		Album:        "Test Album",
		AlbumID:      "album_456",
		AlbumArtist:  "Test Album Artist",
		TrackNumber:  1,
		DiscNumber:   1,
		TotalTracks:  10,
		Year:         2023,
		Duration:     180,
		Status:       domain.TrackStatusMissing,
		ParentJobID:  "job_789",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Artists:      []string{"Test Artist", "Featuring Artist"},
		AlbumArtists: []string{"Test Album Artist"},
	}

	// Test CreateTrack
	err := db.CreateTrack(track)
	if err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}
	if track.ID == 0 {
		t.Error("Expected track ID to be set")
	}

	// Test GetTrackByID
	fetched, err := db.GetTrackByID(track.ID)
	if err != nil {
		t.Fatalf("GetTrackByID failed: %v", err)
	}
	if fetched.Title != track.Title {
		t.Errorf("Expected title %s, got %s", track.Title, fetched.Title)
	}
	if len(fetched.Artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(fetched.Artists))
	}
	if fetched.Artists[0] != "Test Artist" {
		t.Errorf("Expected artist[0] 'Test Artist', got %s", fetched.Artists[0])
	}

	// Test GetTrackByProviderID
	byProvider, err := db.GetTrackByProviderID("track_123")
	if err != nil {
		t.Fatalf("GetTrackByProviderID failed: %v", err)
	}
	if byProvider.ID != track.ID {
		t.Errorf("Expected track ID %d, got %d", track.ID, byProvider.ID)
	}

	// Test UpdateTrackStatus
	err = db.UpdateTrackStatus(track.ID, domain.TrackStatusDownloading, "/path/to/file")
	if err != nil {
		t.Errorf("UpdateTrackStatus failed: %v", err)
	}

	fetched, _ = db.GetTrackByID(track.ID)
	if fetched.Status != domain.TrackStatusDownloading {
		t.Errorf("Expected status %s, got %s", domain.TrackStatusDownloading, fetched.Status)
	}
	if fetched.FilePath != "/path/to/file" {
		t.Errorf("Expected file path '/path/to/file', got %s", fetched.FilePath)
	}

	// Test MarkTrackCompleted
	err = db.MarkTrackCompleted(track.ID, "/path/to/completed.flac", "abc123hash")
	if err != nil {
		t.Errorf("MarkTrackCompleted failed: %v", err)
	}

	fetched, _ = db.GetTrackByID(track.ID)
	if fetched.Status != domain.TrackStatusCompleted {
		t.Errorf("Expected status %s, got %s", domain.TrackStatusCompleted, fetched.Status)
	}
	if fetched.FileHash != "abc123hash" {
		t.Errorf("Expected hash 'abc123hash', got %s", fetched.FileHash)
	}
	if fetched.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set")
	}

	// Test IsTrackDownloaded
	downloaded, err := db.IsTrackDownloaded("track_123")
	if err != nil {
		t.Errorf("IsTrackDownloaded failed: %v", err)
	}
	if !downloaded {
		t.Error("Expected track to be marked as downloaded")
	}

	// Test GetDownloadedTrack
	downloadedTrack, err := db.GetDownloadedTrack("track_123")
	if err != nil {
		t.Errorf("GetDownloadedTrack failed: %v", err)
	}
	if downloadedTrack.Title != "Test Track" {
		t.Errorf("Expected title 'Test Track', got %s", downloadedTrack.Title)
	}

	// Test SearchTracks
	results, err := db.SearchTracks("Test", 10)
	if err != nil {
		t.Errorf("SearchTracks failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	results, err = db.SearchTracks("Nonexistent", 10)
	if err != nil {
		t.Errorf("SearchTracks failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}

	// Test MarkTrackFailed
	anotherTrack := &domain.Track{
		ProviderID: "track_fail",
		Title:      "Fail Track",
		Artist:     "Artist",
		Album:      "Album",
		Status:     domain.TrackStatusMissing,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = db.CreateTrack(anotherTrack)
	if err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	err = db.MarkTrackFailed(anotherTrack.ID, "Download failed")
	if err != nil {
		t.Errorf("MarkTrackFailed failed: %v", err)
	}

	failedTrack, _ := db.GetTrackByID(anotherTrack.ID)
	if failedTrack.Status != domain.TrackStatusFailed {
		t.Errorf("Expected status %s, got %s", domain.TrackStatusFailed, failedTrack.Status)
	}
	if failedTrack.Error != "Download failed" {
		t.Errorf("Expected error 'Download failed', got %s", failedTrack.Error)
	}
}

func TestDB_TrackListOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create tracks with different statuses
	tracks := []*domain.Track{
		{ProviderID: "track_1", Title: "Track 1", Artist: "Artist", Album: "Album", Status: domain.TrackStatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "track_2", Title: "Track 2", Artist: "Artist", Album: "Album", Status: domain.TrackStatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "track_3", Title: "Track 3", Artist: "Artist", Album: "Album", Status: domain.TrackStatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "track_4", Title: "Track 4", Artist: "Artist", Album: "Album", Status: domain.TrackStatusFailed, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, tr := range tracks {
		if err := db.CreateTrack(tr); err != nil {
			t.Fatalf("CreateTrack failed: %v", err)
		}
	}

	// Test ListTracks
	allTracks, err := db.ListTracks(10)
	if err != nil {
		t.Errorf("ListTracks failed: %v", err)
	}
	if len(allTracks) != 4 {
		t.Errorf("Expected 4 tracks, got %d", len(allTracks))
	}

	// Test ListCompletedTracks
	completed, err := db.ListCompletedTracks(10)
	if err != nil {
		t.Errorf("ListCompletedTracks failed: %v", err)
	}
	if len(completed) != 2 {
		t.Errorf("Expected 2 completed tracks, got %d", len(completed))
	}

	// Test ListTracksByStatus
	queued, err := db.ListTracksByStatus(domain.TrackStatusQueued, 10)
	if err != nil {
		t.Errorf("ListTracksByStatus failed: %v", err)
	}
	if len(queued) != 1 {
		t.Errorf("Expected 1 queued track, got %d", len(queued))
	}

	// Test DeleteTrack
	err = db.DeleteTrack(tracks[3].ID)
	if err != nil {
		t.Errorf("DeleteTrack failed: %v", err)
	}

	allTracks, _ = db.ListTracks(10)
	if len(allTracks) != 3 {
		t.Errorf("Expected 3 tracks after delete, got %d", len(allTracks))
	}
}

func TestDB_RecomputeAlbumState(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create tracks for the same album
	albumID := "album_test_123"
	tracks := []*domain.Track{
		{ProviderID: "track_1", Title: "Track 1", Artist: "Artist", Album: "Album", AlbumID: albumID, Status: domain.TrackStatusCompleted, FilePath: "/path/1.flac", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "track_2", Title: "Track 2", Artist: "Artist", Album: "Album", AlbumID: albumID, Status: domain.TrackStatusCompleted, FilePath: "/path/2.flac", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ProviderID: "track_3", Title: "Track 3", Artist: "Artist", Album: "Album", AlbumID: albumID, Status: domain.TrackStatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, tr := range tracks {
		if err := db.CreateTrack(tr); err != nil {
			t.Fatalf("CreateTrack failed: %v", err)
		}
	}

	// Test RecomputeAlbumState - partial
	state, err := db.RecomputeAlbumState(albumID)
	if err != nil {
		t.Errorf("RecomputeAlbumState failed: %v", err)
	}
	if state != "partial" {
		t.Errorf("Expected state 'partial', got %s", state)
	}

	// Update all to completed
	for _, tr := range tracks {
		err = db.UpdateTrackStatus(tr.ID, domain.TrackStatusCompleted, tr.FilePath)
		if err != nil {
			t.Fatalf("UpdateTrackStatus failed: %v", err)
		}
	}

	state, err = db.RecomputeAlbumState(albumID)
	if err != nil {
		t.Errorf("RecomputeAlbumState failed: %v", err)
	}
	if state != "completed" {
		t.Errorf("Expected state 'completed', got %s", state)
	}
}

func TestDB_JobStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create jobs with different statuses
	jobs := []*domain.Job{
		{ID: "job_1", Type: domain.JobTypeTrack, Status: domain.JobStatusCompleted, SourceID: "s1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job_2", Type: domain.JobTypeTrack, Status: domain.JobStatusCompleted, SourceID: "s2", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job_3", Type: domain.JobTypeTrack, Status: domain.JobStatusFailed, SourceID: "s3", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job_4", Type: domain.JobTypeTrack, Status: domain.JobStatusCancelled, SourceID: "s4", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job_5", Type: domain.JobTypeTrack, Status: domain.JobStatusQueued, SourceID: "s5", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, j := range jobs {
		if err := db.CreateJob(j); err != nil {
			t.Fatalf("CreateJob failed: %v", err)
		}
	}

	// Test GetJobStats
	stats, err := db.GetJobStats()
	if err != nil {
		t.Errorf("GetJobStats failed: %v", err)
	}
	if stats.Total != 4 {
		t.Errorf("Expected total 4, got %d", stats.Total)
	}
	if stats.Completed != 2 {
		t.Errorf("Expected completed 2, got %d", stats.Completed)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected failed 1, got %d", stats.Failed)
	}
	if stats.Cancelled != 1 {
		t.Errorf("Expected cancelled 1, got %d", stats.Cancelled)
	}

	// Test ClearFinishedJobs
	err = db.ClearFinishedJobs()
	if err != nil {
		t.Errorf("ClearFinishedJobs failed: %v", err)
	}

	finished, err := db.ListFinishedJobs(10)
	if err != nil {
		t.Errorf("ListFinishedJobs failed: %v", err)
	}
	if len(finished) != 0 {
		t.Errorf("Expected 0 finished jobs, got %d", len(finished))
	}
}

func TestDB_GetActiveJobBySourceID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	job := &domain.Job{
		ID:        "active_job",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusRunning,
		SourceID:  "track_123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.CreateJob(job); err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Test GetActiveJobBySourceID - finds running job
	active, err := db.GetActiveJobBySourceID("track_123", domain.JobTypeTrack)
	if err != nil {
		t.Errorf("GetActiveJobBySourceID failed: %v", err)
	}
	if active == nil {
		t.Error("Expected to find active job")
	} else if active.ID != "active_job" {
		t.Errorf("Expected job ID 'active_job', got %s", active.ID)
	}

	// Test GetActiveJobBySourceID - non-existent returns nil
	nonexistent, err := db.GetActiveJobBySourceID("nonexistent", domain.JobTypeTrack)
	if err != nil {
		t.Errorf("GetActiveJobBySourceID failed: %v", err)
	}
	if nonexistent != nil {
		t.Error("Expected nil for non-existent job")
	}

	// Test IsTrackActive
	isActive, err := db.IsTrackActive("track_123")
	if err != nil {
		t.Errorf("IsTrackActive failed: %v", err)
	}
	if !isActive {
		t.Error("Expected track to be active")
	}

	isActive, err = db.IsTrackActive("nonexistent")
	if err != nil {
		t.Errorf("IsTrackActive failed: %v", err)
	}
	if isActive {
		t.Error("Expected track to not be active")
	}
}

func TestDB_TrackWithJsonFields(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test track with Artists and AlbumArtists stored as JSON arrays
	track := &domain.Track{
		ProviderID:   "json_test",
		Title:        "Test Track",
		Artist:       "Primary Artist",
		Artists:      []string{"Primary Artist", "Featuring 1", "Featuring 2"},
		AlbumArtist:  "Album Artist",
		AlbumArtists: []string{"Album Artist", "Guest Album Artist"},
		Album:        "Test Album",
		AlbumID:      "album_123",
		Status:       domain.TrackStatusMissing,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := db.CreateTrack(track)
	if err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	// Retrieve and verify JSON fields are correctly deserialized
	fetched, err := db.GetTrackByID(track.ID)
	if err != nil {
		t.Fatalf("GetTrackByID failed: %v", err)
	}

	// Verify Artists array
	if len(fetched.Artists) != 3 {
		t.Errorf("Expected 3 artists, got %d", len(fetched.Artists))
	}
	if fetched.Artists[0] != "Primary Artist" {
		t.Errorf("Artists[0] = %s, want 'Primary Artist'", fetched.Artists[0])
	}
	if fetched.Artists[1] != "Featuring 1" {
		t.Errorf("Artists[1] = %s, want 'Featuring 1'", fetched.Artists[1])
	}

	// Verify AlbumArtists array
	if len(fetched.AlbumArtists) != 2 {
		t.Errorf("Expected 2 album artists, got %d", len(fetched.AlbumArtists))
	}
	if fetched.AlbumArtists[0] != "Album Artist" {
		t.Errorf("AlbumArtists[0] = %s, want 'Album Artist'", fetched.AlbumArtists[0])
	}

	// Test track with empty Artists JSON
	emptyTrack := &domain.Track{
		ProviderID: "empty_json_test",
		Title:      "Empty Track",
		Artist:     "Solo Artist",
		Album:      "Album",
		Status:     domain.TrackStatusMissing,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		// Artists and AlbumArtists left as nil
	}

	err = db.CreateTrack(emptyTrack)
	if err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	fetchedEmpty, _ := db.GetTrackByID(emptyTrack.ID)
	if len(fetchedEmpty.Artists) != 0 {
		t.Errorf("Expected empty Artists slice, got %d elements", len(fetchedEmpty.Artists))
	}
}
