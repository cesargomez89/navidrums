package catalog

import (
	"strconv"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func resolveQobuzAudioQuality(hires bool, bitDepth int) string {
	if hires && bitDepth >= 24 {
		return "HI_RES_LOSSLESS"
	}
	if bitDepth >= 16 {
		return "LOSSLESS"
	}
	return "LOW"
}

func (r *QobuzSearchData) ToDomain() *domain.SearchResult {
	result := &domain.SearchResult{
		Albums:    make([]domain.Album, 0),
		Tracks:    make([]domain.CatalogTrack, 0),
		Artists:   make([]domain.Artist, 0),
		Playlists: make([]domain.Playlist, 0),
	}

	for _, item := range r.Albums.Items {
		result.Albums = append(result.Albums, item.ToDomain())
	}

	for _, item := range r.Tracks.Items {
		result.Tracks = append(result.Tracks, item.ToDomain())
	}

	for _, item := range r.Artists.Items {
		result.Artists = append(result.Artists, item.ToDomain())
	}

	for _, item := range r.Playlists.Items {
		result.Playlists = append(result.Playlists, item.ToDomain())
	}

	return result
}

func (item *QobuzSearchAlbumItem) ToDomain() domain.Album {
	return domain.Album{
		ID:           item.ID,
		Title:        item.Title,
		ArtistID:     strconv.Itoa(item.Artist.ID),
		Artist:       item.Artist.Name,
		AlbumArtURL:  item.Image.Large,
		URL:          item.URL,
		Genre:        item.Genre.Name,
		Label:        item.Label.Name,
		UPC:          item.UPC,
		Year:         parseYear(item.ReleaseDateOriginal),
		TotalTracks:  item.TracksCount,
	}
}

func (item *QobuzSearchArtistItem) ToDomain() domain.Artist {
	picURL := ""
	if item.Image != nil {
		picURL = "https://static.qobuz.com/images/artists/" + item.Image.Hash + "." + item.Image.Format
	}
	return domain.Artist{
		ID:         strconv.Itoa(item.ID),
		Name:       item.Name,
		PictureURL: picURL,
	}
}

func (item *QobuzSearchPlaylistItem) ToDomain() domain.Playlist {
	return domain.Playlist{
		ProviderID:  strconv.FormatInt(item.ID, 10),
		Title:       item.Title,
		Description: item.Description,
		ImageURL:    item.Image.Large,
	}
}

func (resp *QobuzAlbumResponse) ToDomain() *domain.Album {
	tracks := make([]domain.CatalogTrack, 0)
	for _, t := range resp.Tracks.Items {
		tracks = append(tracks, t.ToDomain())
	}

	var artistIDs []string
	var artists []string
	for _, a := range resp.Artists {
		artistIDs = append(artistIDs, strconv.Itoa(a.ID))
		artists = append(artists, a.Name)
	}

	return &domain.Album{
		ID:           resp.ID,
		Title:        resp.Title,
		ArtistID:     strconv.Itoa(resp.Artist.ID),
		Artist:       resp.Artist.Name,
		AlbumArtURL:  resp.Image.Large,
		Genre:        resp.Genre.Name,
		Label:        resp.Label.Name,
		UPC:          resp.UPC,
		Year:         parseYear(resp.ReleaseDateOriginal),
		TotalTracks:  resp.TracksCount,
		TotalDiscs:   resp.MediaCount,
		Copyright:    resp.Copyright,
		Tracks:       tracks,
		ArtistIDs:    artistIDs,
		Artists:      artists,
	}
}

func (item *QobuzTrackItem) ToDomain() domain.CatalogTrack {
	var replayGain float64
	var peak float64
	if item.AudioInfo != nil {
		replayGain = item.AudioInfo.ReplayGainTrackGain
		peak = item.AudioInfo.ReplayGainTrackPeak
	}

	albumID := ""
	albumTitle := ""
	albumArtist := ""
	albumArtURL := ""
	if item.Album != nil {
		albumID = item.Album.ID
		albumTitle = item.Album.Title
		albumArtist = item.Album.Artist.Name
		albumArtURL = item.Album.Image.Large
	}

	artists := []string{item.Performer.Name}
	artistIDs := []string{strconv.Itoa(item.Performer.ID)}

	return domain.CatalogTrack{
		ID:             strconv.Itoa(item.ID),
		Title:          item.Title,
		Artist:         item.Performer.Name,
		ArtistID:       strconv.Itoa(item.Performer.ID),
		Album:          albumTitle,
		AlbumID:        albumID,
		AlbumArtist:    albumArtist,
		AlbumArtURL:    albumArtURL,
		TrackNumber:    item.TrackNumber,
		DiscNumber:     item.MediaNumber,
		Year:           parseYear(item.ReleaseDateOriginal),
		Duration:       item.Duration,
		ISRC:           item.ISRC,
		Genre:          "",
		Copyright:      item.Copyright,
		ReplayGain:     replayGain,
		Peak:           peak,
		ExplicitLyrics: item.ParentalWarning,
		TotalTracks:    0,
		TotalDiscs:     0,
		Artists:        artists,
		ArtistIDs:      artistIDs,
		AudioQuality:   resolveQobuzAudioQuality(item.Hires, item.MaximumBitDepth),
	}
}

func (item *QobuzTrackResponse) ToDomain() domain.CatalogTrack {
	var replayGain float64
	var peak float64
	if item.AudioInfo != nil {
		replayGain = item.AudioInfo.ReplayGainTrackGain
		peak = item.AudioInfo.ReplayGainTrackPeak
	}

	albumID := ""
	albumTitle := ""
	albumArtist := ""
	albumArtURL := ""
	if item.Album != nil {
		albumID = item.Album.ID
		albumTitle = item.Album.Title
		albumArtist = item.Album.Artist.Name
		albumArtURL = item.Album.Image.Large
	}

	artists := []string{item.Performer.Name}
	artistIDs := []string{strconv.Itoa(item.Performer.ID)}

	return domain.CatalogTrack{
		ID:             strconv.Itoa(item.ID),
		Title:          item.Title,
		Artist:         item.Performer.Name,
		ArtistID:       strconv.Itoa(item.Performer.ID),
		Album:          albumTitle,
		AlbumID:        albumID,
		AlbumArtist:    albumArtist,
		AlbumArtURL:    albumArtURL,
		TrackNumber:    item.TrackNumber,
		DiscNumber:     item.MediaNumber,
		Year:           parseYear(item.ReleaseDateOriginal),
		Duration:       item.Duration,
		ISRC:           item.ISRC,
		Genre:          "",
		Copyright:      item.Copyright,
		ReplayGain:     replayGain,
		Peak:           peak,
		ExplicitLyrics: item.ParentalWarning,
		TotalTracks:    0,
		TotalDiscs:     0,
		Artists:        artists,
		ArtistIDs:      artistIDs,
		AudioQuality:   resolveQobuzAudioQuality(item.Hires, item.MaximumBitDepth),
	}
}

func (data *QobuzArtistData) ToDomain() *domain.Artist {
	picURL := ""
	if data.Artist.Images.Portrait != nil {
		picURL = "https://static.qobuz.com/images/artists/" + data.Artist.Images.Portrait.Hash + "." + data.Artist.Images.Portrait.Format
	}

	albums := make([]domain.Album, 0)
	for _, a := range data.Artist.Albums.Items {
		albums = append(albums, domain.Album{
			ID:         a.ID,
			Title:      a.Title,
			AlbumArtURL: a.Image.Large,
			Genre:      a.Genre.Name,
			Year:       parseYear(a.ReleaseDateOriginal),
		})
	}

	topTracks := make([]domain.CatalogTrack, 0)
	for _, t := range data.Artist.TopTracks {
		topTracks = append(topTracks, t.ToDomain())
	}

	return &domain.Artist{
		ID:        strconv.Itoa(data.Artist.ID),
		Name:      data.Artist.Name.Display,
		PictureURL: picURL,
		Albums:    albums,
		TopTracks: topTracks,
	}
}

func (item *QobuzTopTrackItem) ToDomain() domain.CatalogTrack {
	bitDepth := 16
	hires := false
	if item.AudioInfo.MaximumBitDepth > 0 {
		bitDepth = item.AudioInfo.MaximumBitDepth
		hires = item.Rights.HiresStreamable
	}

	albumID := ""
	albumTitle := ""
	albumArtURL := ""
	if item.Album != nil {
		albumID = item.Album.ID
		albumTitle = item.Album.Title
		albumArtURL = item.Album.Image.Large
	}

	return domain.CatalogTrack{
		ID:             strconv.Itoa(item.ID),
		Title:          item.Title,
		Artist:         item.Artist.Display,
		Album:          albumTitle,
		AlbumID:        albumID,
		AlbumArtURL:    albumArtURL,
		TrackNumber:    item.PhysicalSupport.TrackNumber,
		DiscNumber:     item.PhysicalSupport.MediaNumber,
		Duration:       item.Duration,
		ISRC:           item.ISRC,
		ExplicitLyrics: item.ParentalWarning,
		Artists:        []string{item.Artist.Display},
		AudioQuality:   resolveQobuzAudioQuality(hires, bitDepth),
	}
}

func (item *QobuzSimilarArtistItem) ToDomain() domain.Artist {
	picURL := ""
	if item.Images.Portrait != nil {
		picURL = "https://static.qobuz.com/images/artists/" + item.Images.Portrait.Hash + "." + item.Images.Portrait.Format
	}
	return domain.Artist{
		ID:         strconv.Itoa(item.ID),
		Name:       item.Name.Display,
		PictureURL: picURL,
	}
}