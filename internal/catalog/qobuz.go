package catalog

import (
	"context"
	"errors"
	"io"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type QobuzProvider struct {
	BaseURL string
}

func NewQobuzProvider(baseURL string) *QobuzProvider {
	return &QobuzProvider{BaseURL: baseURL}
}

var errQobuzNotImplemented = errors.New("qobuz provider not yet implemented")

func (p *QobuzProvider) Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	return nil, "", errQobuzNotImplemented
}

func (p *QobuzProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetSimilarArtists(ctx context.Context, id string) ([]domain.Artist, error) {
	return nil, errQobuzNotImplemented
}

func (p *QobuzProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	return "", "", errQobuzNotImplemented
}

func (p *QobuzProvider) GetRecommendations(ctx context.Context, id string) ([]domain.CatalogTrack, error) {
	return nil, errQobuzNotImplemented
}

var _ Provider = (*QobuzProvider)(nil)