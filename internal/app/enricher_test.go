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

	t.Run("success_all_fields", func(t *testing.T) {
		mbID := "mb-recording-124"
		mockClient := &mockMBClient{
			recording: &musicbrainz.RecordingMetadata{
				RecordingID:    mbID,
				Artist:         "MB Artist",
				Artists:        []string{"MB Artist", "MB Featuring"},
				ArtistIDs:      []string{"a1", "a2"},
				Title:          "MB Title",
				Duration:       180,
				Year:           2023,
				Genre:          "Alternative Rock",
				SubGenre:       "Indie",
				Barcode:        "1234567890",
				CatalogNumber:  "CAT123",
				ReleaseID:      "rel-123",
				ReleaseType:    "album",
				Label:          "MB Label",
				AlbumArtists:   []string{"MB Album Artist"},
				AlbumArtistIDs: []string{"aa1"},
				Composer:       "MB Composer",
				Tags:           []string{"tag1", "tag2"},
			},
		}

		enricher := app.NewMetadataEnricher(mockClient, nil)
		track := &domain.Track{
			ISRC: "USABC1234567",
		}

		err := enricher.EnrichTrack(context.Background(), track, logger)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify fields
		if *track.RecordingID != mbID {
			t.Errorf("RecordingID mismatch")
		}
		if track.Artist != "MB Artist" {
			t.Errorf("Artist mismatch")
		}
		if len(track.Artists) != 2 {
			t.Errorf("Artists length mismatch")
		}
		if track.Title != "MB Title" {
			t.Errorf("Title mismatch")
		}
		if track.Year != 2023 {
			t.Errorf("Year mismatch")
		}
		// SubGenre is now embedded in Genre as "Genre; subgenre"
		if track.Genre != "Alternative Rock; Indie" {
			t.Errorf("Genre mismatch, got %q, want %q", track.Genre, "Alternative Rock; Indie")
		}
		if track.ReleaseID != "rel-123" {
			t.Errorf("ReleaseID mismatch")
		}
	})

	t.Run("no_id_isrc_skips", func(t *testing.T) {
		mockClient := &mockMBClient{}
		enricher := app.NewMetadataEnricher(mockClient, nil)
		track := &domain.Track{}

		err := enricher.EnrichTrack(context.Background(), track, logger)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("dont_overwrite_existing", func(t *testing.T) {
		mockClient := &mockMBClient{
			recording: &musicbrainz.RecordingMetadata{
				Artist: "MB Artist",
				Year:   2023,
				Genre:  "Metal",
			},
		}

		enricher := app.NewMetadataEnricher(mockClient, nil)
		track := &domain.Track{
			ISRC:   "USABC1234567",
			Artist: "Keep Me",
			Year:   2000,
			Genre:  "Keep Me Too",
		}

		_ = enricher.EnrichTrack(context.Background(), track, logger)

		if track.Artist != "Keep Me" {
			t.Errorf("Overwrote Artist")
		}
		if track.Year != 2000 {
			t.Errorf("Overwrote Year")
		}
		if track.Genre != "Keep Me Too" {
			t.Errorf("Overwrote Genre")
		}
	})

	t.Run("genre_no_subgenre_no_semicolon", func(t *testing.T) {
		mockClient := &mockMBClient{
			recording: &musicbrainz.RecordingMetadata{
				Genre:    "Alternative Rock",
				SubGenre: "", // no sub-genre
			},
		}

		enricher := app.NewMetadataEnricher(mockClient, nil)
		track := &domain.Track{
			ISRC: "USABC1234567",
		}

		_ = enricher.EnrichTrack(context.Background(), track, logger)

		if track.Genre != "Alternative Rock" {
			t.Errorf("Genre mismatch, got %q, want %q", track.Genre, "Alternative Rock")
		}
	})
}
