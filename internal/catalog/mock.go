package catalog

import (
	"context"
	"io"
	"strings"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	res := &domain.SearchResult{
		Artists: []domain.Artist{{ID: "1", Name: "Mock Artist"}},
		Albums:  []domain.Album{{ID: "1", Title: "Mock Album", Artist: "Mock Artist"}},
		Tracks:  []domain.Track{{ID: "1", Title: "Mock Track", Artist: "Mock Artist", Album: "Mock Album", TrackNumber: 1, Duration: 180}},
	}

	if searchType == "" {
		searchType = "album"
	}

	resFiltered := &domain.SearchResult{}
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

func (p *MockProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	return &domain.Artist{ID: id, Name: "Mock Artist"}, nil
}

func (p *MockProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	return &domain.Album{
		ID:     id,
		Title:  "Mock Album",
		Artist: "Mock Artist",
		Tracks: []domain.Track{
			{ID: "1", Title: "Track 1", Artist: "Mock Artist", TrackNumber: 1, Duration: 180},
			{ID: "2", Title: "Track 2", Artist: "Mock Artist", TrackNumber: 2, Duration: 200},
		},
	}, nil
}

func (p *MockProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	return &domain.Playlist{
		ID:    id,
		Title: "Mock Playlist",
		Tracks: []domain.Track{
			{ID: "3", Title: "Track 3", Artist: "Unknown", TrackNumber: 1},
		},
	}, nil
}

func (p *MockProvider) GetTrack(ctx context.Context, id string) (*domain.Track, error) {
	return &domain.Track{ID: id, Title: "Mock Track", Artist: "Mock Artist", Album: "Mock Album", TrackNumber: 1}, nil
}

func (p *MockProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader("dummy audio content")), "audio/flac", nil
}

func (p *MockProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	return []domain.Album{
		{ID: "101", Title: "Similar Mock Album 1", Artist: "Mock Artist"},
		{ID: "102", Title: "Similar Mock Album 2", Artist: "Mock Artist"},
		{ID: "103", Title: "Similar Mock Album 3", Artist: "Mock Artist"},
		{ID: "104", Title: "Similar Mock Album 4", Artist: "Mock Artist"},
		{ID: "105", Title: "Similar Mock Album 5", Artist: "Mock Artist"},
		{ID: "106", Title: "Similar Mock Album 6", Artist: "Mock Artist"},
		{ID: "107", Title: "Similar Mock Album 7", Artist: "Mock Artist"},
		{ID: "108", Title: "Similar Mock Album 8", Artist: "Mock Artist"},
	}, nil
}

func (p *MockProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	return "Mock lyrics for testing", "[00:00.00] Mock lyrics for testing", nil
}
