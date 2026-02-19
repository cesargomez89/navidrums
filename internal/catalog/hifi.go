package catalog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

var hifiLogger = slog.Default().WithGroup("hifi")

type HifiProvider struct {
	BaseURL string
	Client  *http.Client
}

func NewHifiProvider(baseURL string) *HifiProvider {
	return &HifiProvider{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *HifiProvider) ensureAbsoluteURL(urlOrID string, size ...string) string {
	if urlOrID == "" {
		return ""
	}
	if strings.HasPrefix(urlOrID, "http://") || strings.HasPrefix(urlOrID, "https://") {
		return urlOrID
	}
	imgSize := "640x640"
	if len(size) > 0 {
		imgSize = size[0]
	}
	path := strings.ReplaceAll(urlOrID, "-", "/")
	return fmt.Sprintf("https://resources.tidal.com/images/%s/%s.jpg", path, imgSize)
}

func (p *HifiProvider) GetArtist(ctx context.Context, id string) (*domain.Artist, error) {
	u := fmt.Sprintf("%s/artist/?id=%s", p.BaseURL, id)
	var resp struct {
		Artist struct {
			ID      json.Number `json:"id"`
			Name    string      `json:"name"`
			Picture string      `json:"picture"`
		} `json:"artist"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	artist := &domain.Artist{
		ID:         formatID(resp.Artist.ID),
		Name:       resp.Artist.Name,
		PictureURL: p.ensureAbsoluteURL(resp.Artist.Picture, "320x320"),
	}

	aggUrl := fmt.Sprintf("%s/artist/?f=%s&skip_tracks=true", p.BaseURL, id)
	var aggResp struct {
		Albums struct {
			Items []struct {
				ID    json.Number `json:"id"`
				Title string      `json:"title"`
				Cover string      `json:"cover"`
			} `json:"items"`
		} `json:"albums"`
		Tracks []struct {
			ID          json.Number `json:"id"`
			Title       string      `json:"title"`
			TrackNumber int         `json:"trackNumber"`
			Duration    int         `json:"duration"`
			Artist      struct {
				ID   json.Number `json:"id"`
				Name string      `json:"name"`
			} `json:"artist"`
			Album struct {
				ID    json.Number `json:"id"`
				Title string      `json:"title"`
				Cover string      `json:"cover"`
			} `json:"album"`
		} `json:"tracks"`
	}

	if err := p.get(ctx, aggUrl, &aggResp); err == nil {
		for _, item := range aggResp.Albums.Items {
			album := domain.Album{
				ID:          formatID(item.ID),
				Title:       item.Title,
				Artist:      artist.Name,
				AlbumArtURL: p.ensureAbsoluteURL(item.Cover, "640x640"),
			}
			artist.Albums = append(artist.Albums, album)
		}
		for _, item := range aggResp.Tracks {
			artist.TopTracks = append(artist.TopTracks, domain.CatalogTrack{
				ID:          formatID(item.ID),
				Title:       item.Title,
				ArtistID:    formatID(item.Artist.ID),
				Artist:      item.Artist.Name,
				AlbumID:     formatID(item.Album.ID),
				Album:       item.Album.Title,
				TrackNumber: item.TrackNumber,
				Duration:    item.Duration,
				AlbumArtURL: p.ensureAbsoluteURL(item.Album.Cover, "640x640"),
			})
		}
	}

	return artist, nil
}

func (p *HifiProvider) GetAlbum(ctx context.Context, id string) (*domain.Album, error) {
	u := fmt.Sprintf("%s/album/?id=%s", p.BaseURL, id)
	var resp struct {
		Data struct {
			ID              json.Number `json:"id"`
			Title           string      `json:"title"`
			ReleaseDate     string      `json:"releaseDate"`
			Copyright       string      `json:"copyright"`
			NumberOfTracks  int         `json:"numberOfTracks"`
			NumberOfVolumes int         `json:"numberOfVolumes"`
			Type            string      `json:"type"`
			UPC             string      `json:"upc"`
			URL             string      `json:"url"`
			Explicit        bool        `json:"explicit"`
			Genre           string      `json:"genre"`
			Label           string      `json:"label"`
			Artist          struct {
				ID   json.Number `json:"id"`
				Name string      `json:"name"`
			} `json:"artist"`
			Cover FlexCover `json:"cover"`
			Items []struct {
				Item struct {
					Title        string      `json:"title"`
					Duration     int         `json:"duration"`
					TrackNumber  int         `json:"trackNumber"`
					VolumeNumber int         `json:"volumeNumber"`
					ID           json.Number `json:"id"`
					ISRC         string      `json:"isrc"`
					Explicit     bool        `json:"explicit"`
					BPM          int         `json:"bpm"`
					Key          string      `json:"key"`
					KeyScale     string      `json:"keyScale"`
					ReplayGain   float64     `json:"replayGain"`
					Peak         float64     `json:"peak"`
					Version      *string     `json:"version"`
					URL          string      `json:"url"`
					AudioQuality string      `json:"audioQuality"`
					Artists      []struct {
						ID   json.Number `json:"id"`
						Name string      `json:"name"`
					} `json:"artists"`
				} `json:"item"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	year := parseYear(resp.Data.ReleaseDate)

	albumArtURL := ""
	if len(resp.Data.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(resp.Data.Cover[0], "640x640")
	}

	album := &domain.Album{
		ID:          formatID(resp.Data.ID),
		Title:       resp.Data.Title,
		ArtistID:    formatID(resp.Data.Artist.ID),
		Artist:      resp.Data.Artist.Name,
		Artists:     []string{resp.Data.Artist.Name},
		ArtistIDs:   []string{formatID(resp.Data.Artist.ID)},
		Year:        year,
		ReleaseDate: resp.Data.ReleaseDate,
		Copyright:   resp.Data.Copyright,
		TotalTracks: resp.Data.NumberOfTracks,
		TotalDiscs:  resp.Data.NumberOfVolumes,
		AlbumArtURL: albumArtURL,
		UPC:         resp.Data.UPC,
		AlbumType:   resp.Data.Type,
		URL:         resp.Data.URL,
		Explicit:    resp.Data.Explicit,
		Genre:       resp.Data.Genre,
		Label:       resp.Data.Label,
	}

	for _, wrapped := range resp.Data.Items {
		item := wrapped.Item
		tArtist := album.Artist
		tArtistID := album.ArtistID

		var artists []string
		var artistIDs []string
		for _, a := range item.Artists {
			artists = append(artists, a.Name)
			artistIDs = append(artistIDs, formatID(a.ID))
		}
		if len(artists) > 0 {
			tArtist = artists[0]
			tArtistID = artistIDs[0]
		}

		track := domain.CatalogTrack{
			ID:             formatID(item.ID),
			Title:          item.Title,
			ArtistID:       tArtistID,
			Artist:         tArtist,
			Artists:        artists,
			ArtistIDs:      artistIDs,
			AlbumID:        album.ID,
			AlbumArtist:    album.Artist,
			AlbumArtists:   album.Artists,
			AlbumArtistIDs: album.ArtistIDs,
			Album:          album.Title,
			TrackNumber:    item.TrackNumber,
			DiscNumber:     item.VolumeNumber,
			TotalTracks:    album.TotalTracks,
			TotalDiscs:     album.TotalDiscs,
			Duration:       item.Duration,
			Year:           album.Year,
			ReleaseDate:    album.ReleaseDate,
			Copyright:      album.Copyright,
			ISRC:           item.ISRC,
			AlbumArtURL:    album.AlbumArtURL,
			ExplicitLyrics: item.Explicit,
			BPM:            item.BPM,
			Key:            item.Key,
			KeyScale:       item.KeyScale,
			ReplayGain:     item.ReplayGain,
			Peak:           item.Peak,
			URL:            item.URL,
			AudioQuality:   item.AudioQuality,
			Genre:          album.Genre,
			Label:          album.Label,
		}
		if item.Version != nil {
			track.Version = *item.Version
		}
		album.Tracks = append(album.Tracks, track)
	}
	return album, nil
}

func (p *HifiProvider) GetPlaylist(ctx context.Context, id string) (*domain.Playlist, error) {
	u := fmt.Sprintf("%s/playlist/?id=%s", p.BaseURL, id)
	var resp struct {
		Playlist struct {
			Uuid        string `json:"uuid"`
			Title       string `json:"title"`
			Description string `json:"description"`
			SquareImage string `json:"squareImage"`
		} `json:"playlist"`
		Items []struct {
			Item struct {
				ID          json.Number `json:"id"`
				Title       string      `json:"title"`
				TrackNumber int         `json:"trackNumber"`
				Duration    int         `json:"duration"`
				ISRC        string      `json:"isrc"`
				Explicit    bool        `json:"explicit"`
				Album       struct {
					ID    json.Number `json:"id"`
					Title string      `json:"title"`
					Cover FlexCover   `json:"cover"`
				} `json:"album"`
				Artists []struct {
					ID   json.Number `json:"id"`
					Name string      `json:"name"`
				} `json:"artists"`
			} `json:"item"`
		} `json:"items"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	pl := &domain.Playlist{
		ID:          resp.Playlist.Uuid,
		Title:       resp.Playlist.Title,
		Description: resp.Playlist.Description,
		ImageURL:    p.ensureAbsoluteURL(resp.Playlist.SquareImage, "640x640"),
	}

	for _, wrapped := range resp.Items {
		item := wrapped.Item

		var artists []string
		var artistIDs []string
		for _, a := range item.Artists {
			artists = append(artists, a.Name)
			artistIDs = append(artistIDs, formatID(a.ID))
		}
		if len(artists) == 0 {
			artists = []string{"Unknown"}
			artistIDs = []string{""}
		}

		albumArtURL := ""
		if len(item.Album.Cover) > 0 {
			albumArtURL = p.ensureAbsoluteURL(item.Album.Cover[0], "640x640")
		}

		pl.Tracks = append(pl.Tracks, domain.CatalogTrack{
			ID:             formatID(item.ID),
			Title:          item.Title,
			ArtistID:       artistIDs[0],
			Artist:         artists[0],
			Artists:        artists,
			ArtistIDs:      artistIDs,
			AlbumID:        formatID(item.Album.ID),
			Album:          item.Album.Title,
			TrackNumber:    item.TrackNumber,
			Duration:       item.Duration,
			ISRC:           item.ISRC,
			AlbumArtURL:    albumArtURL,
			ExplicitLyrics: item.Explicit,
		})
	}

	return pl, nil
}

func (p *HifiProvider) GetTrack(ctx context.Context, id string) (*domain.CatalogTrack, error) {
	u := fmt.Sprintf("%s/info/?id=%s", p.BaseURL, id)
	var resp struct {
		Data struct {
			ID              json.Number `json:"id"`
			Title           string      `json:"title"`
			Duration        int         `json:"duration"`
			TrackNumber     int         `json:"trackNumber"`
			VolumeNumber    int         `json:"volumeNumber"`
			ISRC            string      `json:"isrc"`
			Explicit        bool        `json:"explicit"`
			Copyright       string      `json:"copyright"`
			BPM             int         `json:"bpm"`
			Key             string      `json:"key"`
			KeyScale        string      `json:"keyScale"`
			ReplayGain      float64     `json:"replayGain"`
			Peak            float64     `json:"peak"`
			Version         *string     `json:"version"`
			URL             string      `json:"url"`
			StreamStartDate string      `json:"streamStartDate"`
			AudioQuality    string      `json:"audioQuality"`
			AudioModes      []string    `json:"audioModes"`
			Album           struct {
				ID              json.Number `json:"id"`
				Title           string      `json:"title"`
				ReleaseDate     string      `json:"releaseDate"`
				NumberOfTracks  int         `json:"numberOfTracks"`
				NumberOfVolumes int         `json:"numberOfVolumes"`
				Cover           FlexCover   `json:"cover"`
				UPC             string      `json:"upc"`
				Label           string      `json:"label"`
				Genre           string      `json:"genre"`
			} `json:"album"`
			Artist struct {
				ID   json.Number `json:"id"`
				Name string      `json:"name"`
			} `json:"artist"`
			Artists []struct {
				ID   json.Number `json:"id"`
				Name string      `json:"name"`
			} `json:"artists"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	year := parseYear(resp.Data.Album.ReleaseDate)
	if year == 0 {
		// Fallback to streamStartDate at track level
		year = parseYear(resp.Data.StreamStartDate)
	}

	albumArtURL := ""
	if len(resp.Data.Album.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(resp.Data.Album.Cover[0], "640x640")
	}

	albumArtist := resp.Data.Artist.Name

	audioModes := ""
	if len(resp.Data.AudioModes) > 0 {
		audioModes = resp.Data.AudioModes[0]
	}

	var artists []string
	var artistIDs []string
	for _, a := range resp.Data.Artists {
		artists = append(artists, a.Name)
		artistIDs = append(artistIDs, formatID(a.ID))
	}
	if len(artists) == 0 {
		artists = []string{resp.Data.Artist.Name}
		artistIDs = []string{formatID(resp.Data.Artist.ID)}
	}

	track := &domain.CatalogTrack{
		ID:             formatID(resp.Data.ID),
		Title:          resp.Data.Title,
		ArtistID:       artistIDs[0],
		Artist:         artists[0],
		Artists:        artists,
		ArtistIDs:      artistIDs,
		AlbumID:        formatID(resp.Data.Album.ID),
		AlbumArtist:    albumArtist,
		AlbumArtists:   []string{albumArtist},
		AlbumArtistIDs: []string{formatID(resp.Data.Artist.ID)},
		Album:          resp.Data.Album.Title,
		TrackNumber:    resp.Data.TrackNumber,
		DiscNumber:     resp.Data.VolumeNumber,
		TotalTracks:    resp.Data.Album.NumberOfTracks,
		TotalDiscs:     resp.Data.Album.NumberOfVolumes,
		Duration:       resp.Data.Duration,
		Year:           year,
		ReleaseDate:    resp.Data.Album.ReleaseDate,
		ISRC:           resp.Data.ISRC,
		Copyright:      resp.Data.Copyright,
		AlbumArtURL:    albumArtURL,
		ExplicitLyrics: resp.Data.Explicit,
		BPM:            resp.Data.BPM,
		Key:            resp.Data.Key,
		KeyScale:       resp.Data.KeyScale,
		ReplayGain:     resp.Data.ReplayGain,
		Peak:           resp.Data.Peak,
		URL:            resp.Data.URL,
		AudioQuality:   resp.Data.AudioQuality,
		AudioModes:     audioModes,
		Label:          resp.Data.Album.Label,
		Genre:          resp.Data.Album.Genre,
	}
	if resp.Data.Version != nil {
		track.Version = *resp.Data.Version
	}

	return track, nil
}

func (p *HifiProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	u := fmt.Sprintf("%s/track/?id=%s&quality=%s", p.BaseURL, trackID, quality)
	hifiLogger.Info("GetStream request", "base_url", p.BaseURL, "track_id", trackID, "quality", quality)

	var resp struct {
		Data struct {
			Manifest         string `json:"manifest"`
			ManifestMimeType string `json:"manifestMimeType"`
		} `json:"data"`
	}

	if err := p.get(ctx, u, &resp); err != nil {
		return nil, "", err
	}

	if resp.Data.Manifest == "" {
		return nil, "", fmt.Errorf("no manifest found")
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.Data.Manifest)
	if err != nil {
		return nil, "", err
	}

	if resp.Data.ManifestMimeType == "application/vnd.tidal.bts" {
		var manifest struct {
			Urls []string `json:"urls"`
		}
		if err := json.Unmarshal(decoded, &manifest); err != nil {
			return nil, "", err
		}
		if len(manifest.Urls) == 0 {
			return nil, "", fmt.Errorf("no urls in manifest")
		}

		streamUrl := manifest.Urls[0]
		sResp, err := p.Client.Get(streamUrl)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != http.StatusOK {
			sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil
	}

	if resp.Data.ManifestMimeType == "application/dash+xml" {
		s := string(decoded)

		if strings.Contains(s, "<SegmentTemplate") {
			return p.handleSegmentedDash(ctx, s)
		}

		re := regexp.MustCompile(`(?is)<BaseURL[^>]*>(.*?)</BaseURL>`)
		match := re.FindStringSubmatch(s)
		streamUrl := ""
		if len(match) > 1 {
			streamUrl = strings.TrimSpace(match[1])
		}

		if streamUrl == "" {
			return nil, "", fmt.Errorf("no BaseURL found in DASH manifest")
		}

		sResp, err := p.Client.Get(streamUrl)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != http.StatusOK {
			sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil
	}

	return nil, "", fmt.Errorf("unsupported manifest type: %s", resp.Data.ManifestMimeType)
}

func (p *HifiProvider) handleSegmentedDash(ctx context.Context, manifest string) (io.ReadCloser, string, error) {
	initRe := regexp.MustCompile(`initialization="([^"]+)"`)
	mediaRe := regexp.MustCompile(`media="([^"]+)"`)
	startNumRe := regexp.MustCompile(`startNumber="(\d+)"`)

	initMatch := initRe.FindStringSubmatch(manifest)
	mediaMatch := mediaRe.FindStringSubmatch(manifest)
	startNumMatch := startNumRe.FindStringSubmatch(manifest)

	if len(initMatch) < 2 || len(mediaMatch) < 2 {
		return nil, "", fmt.Errorf("failed to parse SegmentTemplate URLs")
	}

	initUrl := strings.ReplaceAll(initMatch[1], "&amp;", "&")
	mediaTemplate := strings.ReplaceAll(mediaMatch[1], "&amp;", "&")
	startNum := 1
	if len(startNumMatch) > 1 {
		fmt.Sscanf(startNumMatch[1], "%d", &startNum)
	}

	count := 0
	fullSRe := regexp.MustCompile(`<S\s+([^>]*?)/>`)
	sMatches := fullSRe.FindAllStringSubmatch(manifest, -1)
	for _, sm := range sMatches {
		attrs := sm[1]
		rMatch := regexp.MustCompile(`r="(\d+)"`).FindStringSubmatch(attrs)
		if len(rMatch) > 1 {
			r := 0
			fmt.Sscanf(rMatch[1], "%d", &r)
			count += 1 + r
		} else {
			count += 1
		}
	}

	urls := []string{initUrl}
	for i := 0; i < count; i++ {
		num := startNum + i
		segUrl := strings.ReplaceAll(mediaTemplate, "$Number$", fmt.Sprintf("%d", num))
		urls = append(urls, segUrl)
	}

	return &multiSegmentReader{
		urls:   urls,
		client: p.Client,
		ctx:    ctx,
	}, "audio/mp4", nil
}

func (p *HifiProvider) GetSimilarAlbums(ctx context.Context, id string) ([]domain.Album, error) {
	u := fmt.Sprintf("%s/album/similar/?id=%s&limit=8", p.BaseURL, id)

	var resp struct {
		Albums []struct {
			ID      int    `json:"id"`
			Title   string `json:"title"`
			Artists []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"artists"`
			Cover string `json:"cover"`
		} `json:"albums"`
	}

	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	var albums []domain.Album
	for _, item := range resp.Albums {
		artistName := ""
		if len(item.Artists) > 0 {
			artistName = item.Artists[0].Name
		}

		albums = append(albums, domain.Album{
			ID:          formatID(item.ID),
			Title:       item.Title,
			Artist:      artistName,
			AlbumArtURL: p.ensureAbsoluteURL(item.Cover, "640x640"),
		})
	}

	return albums, nil
}

func (p *HifiProvider) GetLyrics(ctx context.Context, trackID string) (string, string, error) {
	u := fmt.Sprintf("%s/lyrics/?id=%s", p.BaseURL, trackID)
	var resp struct {
		Lyrics struct {
			Lyrics    string `json:"lyrics"`
			Subtitles string `json:"subtitles"`
			Provider  string `json:"lyricsProvider"`
		} `json:"lyrics"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return "", "", err
	}
	return resp.Lyrics.Lyrics, resp.Lyrics.Subtitles, nil
}

func parseYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	var year int
	if _, err := fmt.Sscanf(date[:4], "%d", &year); err != nil {
		return 0
	}
	return year
}

func (p *HifiProvider) get(ctx context.Context, url string, target interface{}) error {
	hifiLogger.Debug("API request", "url", url, "base_url", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed: %s", resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	err = decoder.Decode(target)
	return err
}
