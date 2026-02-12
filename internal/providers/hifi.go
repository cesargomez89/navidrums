package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cesargomez89/navidrums/internal/models"
)

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
	// It's a Tidal ID. Use Tidal's official CDN for images.
	// We replace dashes with slashes if they are present, as many Tidal clients do.
	path := strings.ReplaceAll(urlOrID, "-", "/")
	return fmt.Sprintf("https://resources.tidal.com/images/%s/%s.jpg", path, imgSize)
}

func (p *HifiProvider) GetArtist(ctx context.Context, id string) (*models.Artist, error) {
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

	artist := &models.Artist{
		ID:         formatID(resp.Artist.ID),
		Name:       resp.Artist.Name,
		PictureURL: p.ensureAbsoluteURL(resp.Artist.Picture, "320x320"),
	}

	// Fetch Albums and Top Tracks using the aggregate endpoint
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
			Album       struct {
				Title string `json:"title"`
				Cover string `json:"cover"`
			} `json:"album"`
		} `json:"tracks"`
	}

	if err := p.get(ctx, aggUrl, &aggResp); err == nil {
		for _, item := range aggResp.Albums.Items {
			artist.Albums = append(artist.Albums, models.Album{
				ID:          formatID(item.ID),
				Title:       item.Title,
				Artist:      artist.Name,
				AlbumArtURL: p.ensureAbsoluteURL(item.Cover, "640x640"),
			})
		}
		for _, item := range aggResp.Tracks {
			artist.TopTracks = append(artist.TopTracks, models.Track{
				ID:          formatID(item.ID),
				Title:       item.Title,
				Artist:      artist.Name,
				Album:       item.Album.Title,
				TrackNumber: item.TrackNumber,
				Duration:    item.Duration,
				AlbumArtURL: p.ensureAbsoluteURL(item.Album.Cover, "640x640"),
			})
		}
	}

	return artist, nil
}

func (p *HifiProvider) GetAlbum(ctx context.Context, id string) (*models.Album, error) {
	u := fmt.Sprintf("%s/album/?id=%s", p.BaseURL, id)
	var resp struct {
		Data struct {
			ID              json.Number `json:"id"`
			Title           string      `json:"title"`
			ReleaseDate     string      `json:"releaseDate"`
			Copyright       string      `json:"copyright"`
			NumberOfTracks  int         `json:"numberOfTracks"`
			NumberOfVolumes int         `json:"numberOfVolumes"`
			Artist          struct {
				Name string `json:"name"`
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
					Artists      []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"item"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	// Extract year from release date
	year := 0
	if len(resp.Data.ReleaseDate) >= 4 {
		fmt.Sscanf(resp.Data.ReleaseDate[:4], "%d", &year)
	}

	// Get album art URL
	albumArtURL := ""
	if len(resp.Data.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(resp.Data.Cover[0], "640x640")
	}

	album := &models.Album{
		ID:          formatID(resp.Data.ID),
		Title:       resp.Data.Title,
		Artist:      resp.Data.Artist.Name,
		Year:        year,
		Copyright:   resp.Data.Copyright,
		TotalTracks: resp.Data.NumberOfTracks,
		TotalDiscs:  resp.Data.NumberOfVolumes,
		AlbumArtURL: albumArtURL,
	}

	for _, wrapped := range resp.Data.Items {
		item := wrapped.Item
		tArtist := album.Artist
		if len(item.Artists) > 0 {
			tArtist = item.Artists[0].Name
		}

		album.Tracks = append(album.Tracks, models.Track{
			ID:             formatID(item.ID),
			Title:          item.Title,
			Artist:         tArtist,
			AlbumArtist:    album.Artist,
			Album:          album.Title,
			TrackNumber:    item.TrackNumber,
			DiscNumber:     item.VolumeNumber,
			TotalTracks:    album.TotalTracks,
			TotalDiscs:     album.TotalDiscs,
			Duration:       item.Duration,
			Year:           album.Year,
			Copyright:      album.Copyright,
			ISRC:           item.ISRC,
			AlbumArtURL:    album.AlbumArtURL,
			ExplicitLyrics: item.Explicit,
		})
	}
	return album, nil
}

func (p *HifiProvider) GetPlaylist(ctx context.Context, id string) (*models.Playlist, error) {
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
					Title string    `json:"title"`
					Cover FlexCover `json:"cover"`
				} `json:"album"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"item"`
		} `json:"items"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	pl := &models.Playlist{
		ID:          resp.Playlist.Uuid,
		Title:       resp.Playlist.Title,
		Description: resp.Playlist.Description,
		ImageURL:    p.ensureAbsoluteURL(resp.Playlist.SquareImage, "640x640"),
	}

	for _, wrapped := range resp.Items {
		item := wrapped.Item
		artist := "Unknown"
		if len(item.Artists) > 0 {
			artist = item.Artists[0].Name
		}

		albumArtURL := ""
		if len(item.Album.Cover) > 0 {
			albumArtURL = p.ensureAbsoluteURL(item.Album.Cover[0], "640x640")
		}

		pl.Tracks = append(pl.Tracks, models.Track{
			ID:             formatID(item.ID),
			Title:          item.Title,
			Artist:         artist,
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

func (p *HifiProvider) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	u := fmt.Sprintf("%s/info/?id=%s", p.BaseURL, id)
	var resp struct {
		Data struct {
			ID           json.Number `json:"id"`
			Title        string      `json:"title"`
			Duration     int         `json:"duration"`
			TrackNumber  int         `json:"trackNumber"`
			VolumeNumber int         `json:"volumeNumber"`
			ISRC         string      `json:"isrc"`
			Explicit     bool        `json:"explicit"`
			Copyright    string      `json:"copyright"`
			Album        struct {
				Title           string    `json:"title"`
				ReleaseDate     string    `json:"releaseDate"`
				NumberOfTracks  int       `json:"numberOfTracks"`
				NumberOfVolumes int       `json:"numberOfVolumes"`
				Cover           FlexCover `json:"cover"`
			} `json:"album"`
			Artist struct {
				Name string `json:"name"`
			} `json:"artist"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"data"`
	}
	if err := p.get(ctx, u, &resp); err != nil {
		return nil, err
	}

	// Extract year from release date
	year := 0
	if len(resp.Data.Album.ReleaseDate) >= 4 {
		fmt.Sscanf(resp.Data.Album.ReleaseDate[:4], "%d", &year)
	}

	// Get album art URL
	albumArtURL := ""
	if len(resp.Data.Album.Cover) > 0 {
		albumArtURL = p.ensureAbsoluteURL(resp.Data.Album.Cover[0], "640x640")
	}

	// Get album artist (use first artist or main artist)
	albumArtist := resp.Data.Artist.Name

	return &models.Track{
		ID:             formatID(resp.Data.ID),
		Title:          resp.Data.Title,
		Artist:         resp.Data.Artist.Name,
		AlbumArtist:    albumArtist,
		Album:          resp.Data.Album.Title,
		TrackNumber:    resp.Data.TrackNumber,
		DiscNumber:     resp.Data.VolumeNumber,
		TotalTracks:    resp.Data.Album.NumberOfTracks,
		TotalDiscs:     resp.Data.Album.NumberOfVolumes,
		Duration:       resp.Data.Duration,
		Year:           year,
		ISRC:           resp.Data.ISRC,
		Copyright:      resp.Data.Copyright,
		AlbumArtURL:    albumArtURL,
		ExplicitLyrics: resp.Data.Explicit,
	}, nil
}

func (p *HifiProvider) GetStream(ctx context.Context, trackID string, quality string) (io.ReadCloser, string, error) {
	u := fmt.Sprintf("%s/track/?id=%s&quality=%s", p.BaseURL, trackID, quality)

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

		// Fetch stream
		streamUrl := manifest.Urls[0]
		sResp, err := p.Client.Get(streamUrl)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != 200 {
			sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil // Assuming FLAC for now
	}

	if resp.Data.ManifestMimeType == "application/dash+xml" {
		s := string(decoded)

		// Check for SegmentTemplate (segmented DASH)
		if strings.Contains(s, "<SegmentTemplate") {
			return p.handleSegmentedDash(ctx, s)
		}

		// Regex to find BaseURL content regardless of namespaces or attributes
		re := regexp.MustCompile(`(?is)<BaseURL[^>]*>(.*?)</BaseURL>`)
		match := re.FindStringSubmatch(s)
		streamUrl := ""
		if len(match) > 1 {
			streamUrl = strings.TrimSpace(match[1])
		}

		if streamUrl == "" {
			fmt.Printf("[DEBUG] FAILED DASH MANIFEST: %s\n", s)
			return nil, "", fmt.Errorf("no BaseURL found in DASH manifest")
		}

		// Fetch stream
		sResp, err := p.Client.Get(streamUrl)
		if err != nil {
			return nil, "", err
		}
		if sResp.StatusCode != 200 {
			sResp.Body.Close()
			return nil, "", fmt.Errorf("stream fetch failed: %s", sResp.Status)
		}
		return sResp.Body, "audio/flac", nil
	}

	return nil, "", fmt.Errorf("unsupported manifest type: %s", resp.Data.ManifestMimeType)
}

func (p *HifiProvider) handleSegmentedDash(ctx context.Context, manifest string) (io.ReadCloser, string, error) {
	// Simple regex/string scanning for SegmentTemplate attributes
	// <SegmentTemplate timescale="48000" initialization=".../0.mp4?..." media=".../$Number$.mp4?..." startNumber="1">
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

	// Parse SegmentTimeline to get total count
	// <S d="188416" r="38"/> -> 1 + 38 = 39 segments
	// <S d="118208"/> -> 1 segment
	count := 0
	sRe := regexp.MustCompile(`<S\s+[^>]*?r="(\d+)"[^>]*/>`)
	matches := sRe.FindAllStringSubmatch(manifest, -1)
	for _, m := range matches {
		r := 0
		fmt.Sscanf(m[1], "%d", &r)
		count += 1 + r
	}
	// Also count segments without 'r'
	sSimpleRe := regexp.MustCompile(`<S\s+[^>]*?d="\d+"(?:\s+[^>]*?)?(?:\s+r="\d+")?[^>]*?/>`)
	_ = sSimpleRe.FindAllString(manifest, -1)
	// Actually, the logic above covers 'r' ones. Let's just count total <S> tags minus the ones with 'r'
	// No, easier: iterate through all <S> tags and parse 'r' if present.

	count = 0
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

func (p *HifiProvider) get(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API request failed: %s", resp.Status)
	}

	// For debugging, peek at the body if it fails to decode?
	// or just decode.
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	err = decoder.Decode(target)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to decode response from %s: %v\n", url, err)
	}
	return err
}
