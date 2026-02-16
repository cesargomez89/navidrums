package catalog

import (
	"context"
	"io"

	"github.com/cesargomez89/navidrums/internal/domain"
)

type Provider interface {
	Search(ctx context.Context, query string, searchType string) (*domain.SearchResult, error)
	GetArtist(ctx context.Context, id string) (*domain.Artist, error)
	GetAlbum(ctx context.Context, id string) (*domain.Album, error)
	GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error)
	GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error)
	GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error)
	GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error)
	GetLyrics(ctx context.Context, trackID string) (string, string, error)
}
