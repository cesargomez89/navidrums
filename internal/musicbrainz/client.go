package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/constants"
)

const (
	DefaultUserAgent   = "navidrums/1.0 (https://github.com/cesargomez89/navidrums)"
	requestTimeout     = 10 * time.Second
	minRequestInterval = 1050 * time.Millisecond
)

var DefaultGenreMap = map[string]string{
	"rock":                "Rock",
	"alternative rock":    "Rock",
	"indie rock":          "Rock",
	"hard rock":           "Rock",
	"punk":                "Rock",
	"punk rock":           "Rock",
	"post-punk":           "Rock",
	"garage rock":         "Rock",
	"grunge":              "Rock",
	"emo":                 "Rock",
	"soft rock":           "Rock",
	"industrial rock":     "Rock",
	"metal":               "Metal",
	"heavy metal":         "Metal",
	"nu metal":            "Metal",
	"death metal":         "Metal",
	"black metal":         "Metal",
	"thrash metal":        "Metal",
	"metalcore":           "Metal",
	"progressive metal":   "Metal",
	"rap metal":           "Metal",
	"alternative metal":   "Metal",
	"industrial metal":    "Metal",
	"neue deutsche härte": "Metal",
	"funk metal":          "Metal",
	"pop":                 "Pop",
	"indie pop":           "Pop",
	"synthpop":            "Pop",
	"dance pop":           "Pop",
	"electropop":          "Pop",
	"latin pop":           "Pop",
	"art pop":             "Pop",
	"noise pop":           "Pop",
	"synth-pop":           "Pop",
	"dance-pop":           "Pop",
	"hip hop":             "Hip-Hop",
	"rap":                 "Hip-Hop",
	"trap":                "Hip-Hop",
	"drill":               "Hip-Hop",
	"boom bap":            "Hip-Hop",
	"latin trap":          "Hip-Hop",
	"abstract hip hop":    "Hip-Hop",
	"hardcore rap":        "Hip-Hop",
	"r&b":                 "R&B",
	"rnb":                 "R&B",
	"contemporary r&b":    "R&B",
	"soul":                "R&B",
	"neo soul":            "R&B",
	"funk":                "R&B",
	"rhythm and blues":    "R&B",
	"electronic":          "Electronic",
	"edm":                 "Electronic",
	"house":               "Electronic",
	"techno":              "Electronic",
	"trance":              "Electronic",
	"dubstep":             "Electronic",
	"drum and bass":       "Electronic",
	"dnb":                 "Electronic",
	"trip hop":            "Electronic",
	"alternative dance":   "Electronic",
	"chillwave":           "Electronic",
	"industrial":          "Electronic",
	"club/dance":          "Electronic",
	"disco":               "Electronic",
	"folktronica":         "Electronic",
	"microhouse":          "Electronic",
	"ambient house":       "Electronic",
	"electronica":         "Electronic",
	"dub":                 "Electronic",
	"latin":               "Latin",
	"reggaeton":           "Latin",
	"dembow":              "Latin",
	"salsa":               "Latin",
	"bachata":             "Latin",
	"merengue":            "Latin",
	"cumbia":              "Latin",
	"bomba":               "Latin",
	"bossa nova":          "Latin",
	"regional mexican":    "Regional Mexican",
	"banda":               "Regional Mexican",
	"norteño":             "Regional Mexican",
	"norteno":             "Regional Mexican",
	"corridos":            "Regional Mexican",
	"corridos tumbados":   "Regional Mexican",
	"mariachi":            "Regional Mexican",
	"grupero":             "Regional Mexican",
	"sierreño":            "Regional Mexican",
	"sierreño urbano":     "Regional Mexican",
	"country":             "Country",
	"americana":           "Country",
	"alt-country":         "Country",
	"jazz":                "Jazz",
	"smooth jazz":         "Jazz",
	"bebop":               "Jazz",
	"classical":           "Classical",
	"opera":               "Classical",
	"baroque":             "Classical",
	"romantic":            "Classical",
	"orchestral":          "Classical",
	"classical crossover": "Classical",
	"folk":                "Folk",
	"indie folk":          "Folk",
	"acoustic":            "Folk",
	"reggae":              "Reggae",
	"dancehall":           "Reggae",
	"ska":                 "Reggae",
	"blues":               "Blues",
	"soundtrack":          "Soundtrack",
	"film score":          "Soundtrack",
}

type Client struct {
	httpClient  *http.Client
	genreMap    map[string]string
	lastRequest time.Time
	baseURL     string
	userAgent   string
	mu          sync.Mutex
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		userAgent: DefaultUserAgent,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		genreMap: DefaultGenreMap,
	}
}

func (c *Client) SetGenreMap(m map[string]string) {
	if m != nil {
		c.genreMap = m
	}
}

func (c *Client) GetGenreMap() map[string]string {
	return c.genreMap
}

func (c *Client) GetRecording(ctx context.Context, recordingID, isrc, albumName string) (*RecordingMetadata, error) {
	if recordingID != "" {
		return c.GetRecordingByMBID(ctx, recordingID, albumName)
	}
	return c.GetRecordingByISRC(ctx, isrc, albumName)
}

func (c *Client) GetGenres(ctx context.Context, recordingID, isrc string) (GenreResult, error) {
	if recordingID != "" {
		return c.GetGenresByMBID(ctx, recordingID)
	}
	return c.GetGenresByISRC(ctx, isrc)
}

type GenreResult struct {
	MainGenre string
	SubGenre  string
}

func (c *Client) GetGenresByISRC(ctx context.Context, isrc string) (GenreResult, error) {
	if isrc == "" {
		return GenreResult{}, nil
	}

	u := fmt.Sprintf("%s/recording?query=isrc:%s&inc=tags&fmt=json", c.baseURL, url.QueryEscape(isrc))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return GenreResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return GenreResult{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return GenreResult{}, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return GenreResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	mainGenre, subGenre := extractMainGenre(result.Recordings, c.genreMap)
	return GenreResult{MainGenre: mainGenre, SubGenre: subGenre}, nil
}

func (c *Client) GetRecordingByISRC(ctx context.Context, isrc string, albumName string) (*RecordingMetadata, error) {
	if isrc == "" {
		return nil, nil
	}

	u := fmt.Sprintf("%s/recording?query=isrc:%s&inc=artists+releases+release-artists+tags+isrcs&fmt=json&limit=1", c.baseURL, url.QueryEscape(isrc))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Recordings) == 0 {
		return nil, nil
	}

	rec := result.Recordings[0]
	mainGenre, subGenre := extractMainGenre(result.Recordings, c.genreMap)
	meta := &RecordingMetadata{
		RecordingID: rec.ID,
		Title:       rec.Title,
		Duration:    rec.Length,
		Genre:       mainGenre,
		SubGenre:    subGenre,
		Tags:        extractTags(result.Recordings),
		ISRC:        isrc,
	}

	if meta.ISRC == "" && len(rec.ISRCs) > 0 {
		meta.ISRC = rec.ISRCs[0]
	}

	if len(rec.ArtistCredit) > 0 {
		meta.Artist = rec.ArtistCredit[0].Artist.Name
		meta.Artists = make([]string, len(rec.ArtistCredit))
		meta.ArtistIDs = make([]string, len(rec.ArtistCredit))
		for i, ac := range rec.ArtistCredit {
			meta.Artists[i] = ac.Artist.Name
			meta.ArtistIDs[i] = ac.Artist.ID
			if ac.Type == "composer" && meta.Composer == "" {
				meta.Composer = ac.Artist.Name
			}
		}
	}

	rel := selectBestRelease(rec.Releases, albumName)
	if rel == nil {
		return meta, nil
	}

	meta.Album = rel.Title
	meta.ReleaseDate = rel.Date
	meta.ReleaseID = rel.ReleaseGroup.ID
	meta.Barcode = rel.Barcode
	meta.CatalogNumber = rel.CatalogNumber
	meta.ReleaseType = rel.ReleaseGroup.PrimaryType
	if len(rel.LabelInfo) > 0 {
		meta.Label = rel.LabelInfo[0].Label.Name
	}
	if rel.Date != "" && len(rel.Date) >= 4 {
		_, _ = fmt.Sscanf(rel.Date, "%d", &meta.Year)
	}
	if len(rel.ArtistCredit) > 0 {
		meta.AlbumArtists = make([]string, len(rel.ArtistCredit))
		meta.AlbumArtistIDs = make([]string, len(rel.ArtistCredit))
		for i, ac := range rel.ArtistCredit {
			meta.AlbumArtists[i] = ac.Artist.Name
			meta.AlbumArtistIDs[i] = ac.Artist.ID
		}
		if len(meta.AlbumArtists) > 0 {
			meta.AlbumArtist = meta.AlbumArtists[0]
		}
	}

	return meta, nil
}

func (c *Client) GetRecordingByMBID(ctx context.Context, mbid string, albumName string) (*RecordingMetadata, error) {
	if mbid == "" {
		return nil, nil
	}

	u := fmt.Sprintf("%s/recording/%s?inc=artists+releases+release-groups+artist-credits+tags+isrcs&fmt=json", c.baseURL, url.PathEscape(mbid))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var rec recording
	if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	mainGenre, subGenre := extractMainGenre([]recording{rec}, c.genreMap)
	meta := &RecordingMetadata{
		RecordingID: rec.ID,
		Title:       rec.Title,
		Duration:    rec.Length,
		Genre:       mainGenre,
		SubGenre:    subGenre,
		Tags:        extractTags([]recording{rec}),
	}

	if len(rec.ISRCs) > 0 {
		meta.ISRC = rec.ISRCs[0]
	}

	if len(rec.ArtistCredit) > 0 {
		meta.Artist = rec.ArtistCredit[0].Artist.Name
		meta.Artists = make([]string, len(rec.ArtistCredit))
		meta.ArtistIDs = make([]string, len(rec.ArtistCredit))
		for i, ac := range rec.ArtistCredit {
			meta.Artists[i] = ac.Artist.Name
			meta.ArtistIDs[i] = ac.Artist.ID
			if ac.Type == "composer" && meta.Composer == "" {
				meta.Composer = ac.Artist.Name
			}
		}
	}

	rel := selectBestRelease(rec.Releases, albumName)
	if rel == nil {
		return meta, nil
	}

	meta.Album = rel.Title
	meta.ReleaseDate = rel.Date
	meta.ReleaseID = rel.ReleaseGroup.ID
	meta.Barcode = rel.Barcode
	meta.CatalogNumber = rel.CatalogNumber
	meta.ReleaseType = rel.ReleaseGroup.PrimaryType
	if len(rel.LabelInfo) > 0 {
		meta.Label = rel.LabelInfo[0].Label.Name
	}
	if rel.Date != "" && len(rel.Date) >= 4 {
		_, _ = fmt.Sscanf(rel.Date, "%d", &meta.Year)
	}
	if len(rel.ArtistCredit) > 0 {
		meta.AlbumArtists = make([]string, len(rel.ArtistCredit))
		meta.AlbumArtistIDs = make([]string, len(rel.ArtistCredit))
		for i, ac := range rel.ArtistCredit {
			meta.AlbumArtists[i] = ac.Artist.Name
			meta.AlbumArtistIDs[i] = ac.Artist.ID
		}
		if len(meta.AlbumArtists) > 0 {
			meta.AlbumArtist = meta.AlbumArtists[0]
		}
	}

	return meta, nil
}

func (c *Client) GetGenresByMBID(ctx context.Context, mbid string) (GenreResult, error) {
	if mbid == "" {
		return GenreResult{}, nil
	}

	u := fmt.Sprintf("%s/recording/%s?inc=tags&fmt=json", c.baseURL, url.PathEscape(mbid))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return GenreResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return GenreResult{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return GenreResult{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return GenreResult{}, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var rec recording
	if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
		return GenreResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	mainGenre, subGenre := extractMainGenre([]recording{rec}, c.genreMap)
	return GenreResult{MainGenre: mainGenre, SubGenre: subGenre}, nil
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt < constants.DefaultRetryCount; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if elapsed := time.Since(c.lastRequest); elapsed < minRequestInterval {
			time.Sleep(minRequestInterval - elapsed)
		}
		c.lastRequest = time.Now()

		resp, err := c.httpClient.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * constants.DefaultRetryBase)
	}
	return nil, lastErr
}

func selectBestRelease(releases []release, albumName string) *release {
	if len(releases) == 0 {
		return nil
	}

	normalize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, "_", "")
		s = strings.ReplaceAll(s, ",", "")
		s = strings.ReplaceAll(s, "(", "")
		s = strings.ReplaceAll(s, ")", "")
		return s
	}

	albumNorm := normalize(albumName)

	for i := range releases {
		r := &releases[i]
		releaseNorm := normalize(r.Title)
		if albumNorm != "" && releaseNorm != "" &&
			(strings.Contains(releaseNorm, albumNorm) || strings.Contains(albumNorm, releaseNorm)) {
			return r
		}
	}

	return &releases[0]
}

func extractMainGenre(recordings []recording, genreMap map[string]string) (mainGenre string, subGenre string) {
	genreCounts := make(map[string]int)
	var highestOriginalTag string
	var highestOriginalCount int

	for _, rec := range recordings {
		for _, t := range rec.Tags {
			if t.Count <= 0 {
				continue
			}

			normalized := strings.ToLower(strings.TrimSpace(t.Name))
			if normalized == "" {
				continue
			}

			if mapped, ok := genreMap[normalized]; ok {
				genreCounts[mapped] += t.Count
			} else {
				genreCounts[t.Name] += t.Count
			}

			if t.Count > highestOriginalCount {
				highestOriginalCount = t.Count
				highestOriginalTag = t.Name
			}
		}
	}

	if len(genreCounts) == 0 {
		return "", ""
	}

	var maxGenre string
	var maxCount int
	for genre, count := range genreCounts {
		if count > maxCount {
			maxCount = count
			maxGenre = genre
		}
	}

	if highestOriginalTag != "" && !strings.EqualFold(highestOriginalTag, maxGenre) {
		return maxGenre, highestOriginalTag
	}
	return maxGenre, ""
}

func extractTags(recordings []recording) []string {
	tagsSet := make(map[string]struct{})
	for _, rec := range recordings {
		for _, t := range rec.Tags {
			if t.Count > 0 {
				name := strings.TrimSpace(t.Name)
				if name != "" {
					tagsSet[name] = struct{}{}
				}
			}
		}
	}
	if len(tagsSet) == 0 {
		return nil
	}
	tags := make([]string, 0, len(tagsSet))
	for t := range tagsSet {
		tags = append(tags, t)
	}
	return tags
}

type searchResponse struct {
	Recordings []recording `json:"recordings"`
}

type recording struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Tags         []tag          `json:"tags"`
	Releases     []release      `json:"releases"`
	ArtistCredit []artistCredit `json:"artist-credit"`
	ISRCs        []string       `json:"isrcs"`
	Length       int            `json:"length"`
}

type release struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Status        string         `json:"status"`
	Date          string         `json:"date"`
	Country       string         `json:"country"`
	Barcode       string         `json:"barcode"`
	CatalogNumber string         `json:"catalognumber"`
	LabelInfo     []labelInfo    `json:"label-info"`
	ReleaseGroup  releaseGroup   `json:"release-group"`
	Media         []media        `json:"media"`
	ArtistCredit  []artistCredit `json:"artist-credit"`
}

type artistCredit struct {
	Name       string `json:"name"`
	Artist     artist `json:"artist"`
	JoinPhrase string `json:"joinphrase"`
	Type       string `json:"type"`
}

type releaseGroup struct {
	ID          string `json:"id"`
	PrimaryType string `json:"primary-type"`
}

type media struct {
	TrackCount int `json:"trackCount"`
}

type artist struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	SortName string `json:"sort-name"`
}

type tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type labelInfo struct {
	Label label `json:"label"`
}

type label struct {
	Name string `json:"name"`
}

type RecordingMetadata struct {
	RecordingID    string
	ReleaseID      string
	Composer       string
	Album          string
	AlbumArtist    string
	ReleaseDate    string
	Barcode        string
	Artist         string
	CatalogNumber  string
	Title          string
	ReleaseType    string
	Genre          string
	SubGenre       string
	Label          string
	ISRC           string
	Tags           []string
	Artists        []string
	ArtistIDs      []string
	AlbumArtists   []string
	AlbumArtistIDs []string
	Year           int
	Duration       int
}
