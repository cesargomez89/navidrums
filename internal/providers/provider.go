package providers

import (
	"context"
	"io"

	"github.com/cesargomez89/navidrums/internal/models"
)

type Provider interface {
	Search(ctx context.Context, query string, searchType string) (*models.SearchResult, error)
	GetArtist(ctx context.Context, id string) (*models.Artist, error)
	GetAlbum(ctx context.Context, id string) (*models.Album, error)
	GetPlaylist(ctx context.Context, id string) (*models.Playlist, error)
	GetTrack(ctx context.Context, id string) (*models.Track, error)
	GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) // Returns stream, mimeType, error
}
