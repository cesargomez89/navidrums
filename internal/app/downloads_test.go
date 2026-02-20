package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
)

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
	downloads, err := svc.ListDownloads()
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
	results, err := svc.SearchDownloads("Hello")
	if err != nil {
		t.Fatalf("SearchDownloads failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'Hello', got %d", len(results))
	}

	// Search by artist
	results, err = svc.SearchDownloads("Artist B")
	if err != nil {
		t.Fatalf("SearchDownloads failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Artist B', got %d", len(results))
	}

	// Search by album
	results, err = svc.SearchDownloads("Album Two")
	if err != nil {
		t.Fatalf("SearchDownloads failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Album Two', got %d", len(results))
	}

	// No results
	results, err = svc.SearchDownloads("Nonexistent")
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
