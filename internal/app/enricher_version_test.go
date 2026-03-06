package app_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/domain"
)

func TestMetadataEnricher_UpdateTrackFromCatalog_Version(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	enricher := app.NewMetadataEnricher(nil, nil)

	tests := []struct {
		name      string
		ctTitle   string
		ctVersion string
		expected  string
	}{
		{
			name:      "no_version",
			ctTitle:   "Song Title",
			ctVersion: "",
			expected:  "Song Title",
		},
		{
			name:      "with_version",
			ctTitle:   "Song Title",
			ctVersion: "Remastered 2023",
			expected:  "Song Title (Remastered 2023)",
		},
		{
			name:      "version_already_in_title",
			ctTitle:   "Song Title (Remastered 2023)",
			ctVersion: "Remastered 2023",
			expected:  "Song Title (Remastered 2023)",
		},
		{
			name:      "version_already_in_title_case_insensitive",
			ctTitle:   "Song Title (remastered 2023)",
			ctVersion: "Remastered 2023",
			expected:  "Song Title (remastered 2023)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := &domain.Track{}
			ct := &domain.CatalogTrack{
				Title:   tt.ctTitle,
				Version: tt.ctVersion,
			}

			enricher.UpdateTrackFromCatalog(track, ct, logger)

			if track.Title != tt.expected {
				t.Errorf("Title mismatch. Got: %q, Want: %q", track.Title, tt.expected)
			}
		})
	}
}
