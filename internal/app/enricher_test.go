package app_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/musicbrainz"
)

// mockMBClient is a simple mock for musicbrainz.ClientInterface
type mockMBClient struct {
	recording *musicbrainz.RecordingMetadata
	genres    *musicbrainz.GenreResult
	err       error
}

func (m *mockMBClient) GetRecording(ctx context.Context, recordingID, isrc, fallbackTitle string) (*musicbrainz.RecordingMetadata, error) {
	return m.recording, m.err
}

func (m *mockMBClient) GetGenres(ctx context.Context, recordingID, isrc string) (musicbrainz.GenreResult, error) {
	if m.genres != nil {
		return *m.genres, m.err
	}
	return musicbrainz.GenreResult{}, m.err
}

func (m *mockMBClient) SetGenreMap(genreMap map[string]string) {}

func (m *mockMBClient) GetGenreMap() map[string]string {
	return nil
}

func TestMetadataEnricher_EnrichTrack(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mockClient := &mockMBClient{
		recording: &musicbrainz.RecordingMetadata{
			Artist:   "MB Artist",
			Title:    "MB Title",
			Year:     2023,
			Genre:    "Alternative Rock",
			SubGenre: "Indie",
		},
		genres: &musicbrainz.GenreResult{
			MainGenre: "Alternative Rock",
			SubGenre:  "Indie",
		},
	}

	enricher := app.NewMetadataEnricher(mockClient)

	track := &domain.Track{
		ISRC:   "USABC1234567",
		Artist: "Original Artist", // Should not be overwritten
		Title:  "",                // Should be filled
	}

	err := enricher.EnrichTrack(context.Background(), track, logger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if track.Artist != "Original Artist" {
		t.Errorf("Expected Artist to be Original Artist, got %s", track.Artist)
	}

	if track.Title != "MB Title" {
		t.Errorf("Expected Title to be filled from MB, got %s", track.Title)
	}

	if track.Year != 2023 {
		t.Errorf("Expected Year to be filled from MB, got %d", track.Year)
	}

	if track.Genre != "Alternative Rock" {
		t.Errorf("Expected Genre to be filled from MB genres, got %s", track.Genre)
	}
}
