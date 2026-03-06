package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/logger"
)

func TestDownloadsService_GetRecommendationSeeds(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	log := logger.Default()
	svc := NewDownloadsService(db, log)

	seeds, err := svc.GetRecommendationSeeds()
	if err != nil {
		t.Fatalf("GetRecommendationSeeds failed with no tracks: %v", err)
	}
	if seeds != nil {
		t.Errorf("Expected nil seeds when no tracks exist, got %v", seeds)
	}

	artistID := "artist_1"
	for i := 1; i <= 5; i++ {
		track := &domain.Track{
			ProviderID:  fmt.Sprintf("track_%d", i),
			Title:       fmt.Sprintf("Track %d", i),
			Artist:      "Artist 1",
			ArtistIDs:   []string{artistID},
			Album:       "Album 1",
			AlbumID:     "album_1",
			Status:      domain.TrackStatusCompleted,
			ReleaseType: "album",
			CreatedAt:   time.Now().Add(time.Duration(i) * time.Minute),
			UpdatedAt:   time.Now(),
		}
		if createErr := db.CreateTrack(track); createErr != nil {
			t.Fatalf("Failed to create track %d: %v", i, createErr)
		}
	}

	seeds, err = svc.GetRecommendationSeeds()
	if err != nil {
		t.Fatalf("GetRecommendationSeeds failed: %v", err)
	}
	if seeds == nil {
		t.Fatal("Expected seeds, got nil")
	}
	if seeds.TrackID == "" {
		t.Error("Expected TrackID to be set")
	}

	track2 := &domain.Track{
		ProviderID:  "track_6",
		Title:       "Track 2-1",
		Artist:      "Artist 2",
		ArtistIDs:   []string{"artist_2"},
		Album:       "Album 2",
		AlbumID:     "album_2",
		Status:      domain.TrackStatusCompleted,
		ReleaseType: "album",
		CreatedAt:   time.Now().Add(10 * time.Minute),
		UpdatedAt:   time.Now(),
	}
	if createErr := db.CreateTrack(track2); createErr != nil {
		t.Fatalf("Failed to create track 6: %v", createErr)
	}

	track3 := &domain.Track{
		ProviderID:  "track_7",
		Title:       "Track 3-1",
		Artist:      "Artist 3",
		ArtistIDs:   []string{"artist_3"},
		Album:       "Album 3",
		AlbumID:     "album_3",
		Status:      domain.TrackStatusCompleted,
		ReleaseType: "album",
		CreatedAt:   time.Now().Add(15 * time.Minute),
		UpdatedAt:   time.Now(),
	}
	if createErr := db.CreateTrack(track3); createErr != nil {
		t.Fatalf("Failed to create track 7: %v", createErr)
	}

	seeds, err = svc.GetRecommendationSeeds()
	if err != nil {
		t.Fatalf("GetRecommendationSeeds failed: %v", err)
	}
	if seeds == nil {
		t.Fatal("Expected seeds, got nil")
	}
	if seeds.TrackID == "" {
		t.Error("Expected TrackID to be set")
	}
	if seeds.AlbumID == "" {
		t.Error("Expected AlbumID to be set")
	}
	if seeds.ArtistID == "" {
		t.Error("Expected ArtistID to be set")
	}

	artistMap := make(map[string]bool)
	if seeds.Track != nil && len(seeds.Track.ArtistIDs) > 0 {
		artistMap[seeds.Track.ArtistIDs[0]] = true
	}
	if seeds.Album != nil && len(seeds.Album.ArtistIDs) > 0 {
		artistMap[seeds.Album.ArtistIDs[0]] = true
	}
	if seeds.Artist != nil && len(seeds.Artist.ArtistIDs) > 0 {
		artistMap[seeds.Artist.ArtistIDs[0]] = true
	}

	if len(artistMap) < 2 {
		t.Errorf("Expected at least 2 distinct artists, got %d", len(artistMap))
	}
}
