package providers

import (
	"context"
	"io"
	"strings"

	"github.com/cesargomez89/navidrums/internal/models"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Search(ctx context.Context, query string, searchType string) (*models.SearchResult, error) {
	res := &models.SearchResult{
		Artists: []models.Artist{{ID: "1", Name: "Mock Artist"}},
		Albums:  []models.Album{{ID: "1", Title: "Mock Album", Artist: "Mock Artist"}},
		Tracks:  []models.Track{{ID: "1", Title: "Mock Track", Artist: "Mock Artist", Album: "Mock Album", TrackNumber: 1, Duration: 180}},
	}

	if searchType == "" {
		searchType = "album"
	}

	resFiltered := &models.SearchResult{}
	switch searchType {
	case "artist":
		resFiltered.Artists = res.Artists
	case "album":
		resFiltered.Albums = res.Albums
	case "track":
		resFiltered.Tracks = res.Tracks
	default:
		resFiltered.Albums = res.Albums
	}
	return resFiltered, nil
}

func (p *MockProvider) GetArtist(ctx context.Context, id string) (*models.Artist, error) {
	return &models.Artist{ID: id, Name: "Mock Artist"}, nil
}

func (p *MockProvider) GetAlbum(ctx context.Context, id string) (*models.Album, error) {
	return &models.Album{
		ID:     id,
		Title:  "Mock Album",
		Artist: "Mock Artist",
		Tracks: []models.Track{
			{ID: "1", Title: "Track 1", Artist: "Mock Artist", TrackNumber: 1, Duration: 180},
			{ID: "2", Title: "Track 2", Artist: "Mock Artist", TrackNumber: 2, Duration: 200},
		},
	}, nil
}

func (p *MockProvider) GetPlaylist(ctx context.Context, id string) (*models.Playlist, error) {
	return &models.Playlist{
		ID:    id,
		Title: "Mock Playlist",
		Tracks: []models.Track{
			{ID: "3", Title: "Track 3", Artist: "Unknown", TrackNumber: 1},
		},
	}, nil
}

func (p *MockProvider) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	return &models.Track{ID: id, Title: "Mock Track", Artist: "Mock Artist", Album: "Mock Album", TrackNumber: 1}, nil
}

func (p *MockProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	// Return a dummy stream
	return io.NopCloser(strings.NewReader("dummy audio content")), "audio/flac", nil
}
