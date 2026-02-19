package musicbrainz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/cesargomez89/navidrums/internal/httpclient"
)

const (
	DefaultUserAgent   = "navidrums/1.0 (https://github.com/cesargomez89/navidrums)"
	requestTimeout     = 10 * time.Second
	minRequestInterval = 1100 * time.Millisecond
)

// --------------------------------------------------------------------------
// Client
// --------------------------------------------------------------------------

type Client struct {
	httpClient *httpclient.Client
	genreMap   map[string]string
	baseURL    string
	userAgent  string
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		userAgent: DefaultUserAgent,
		httpClient: httpclient.NewClient(&http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		}, minRequestInterval),
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

// --------------------------------------------------------------------------
// Public API
// --------------------------------------------------------------------------

// GetRecording fetches full recording metadata, preferring lookup by MBID.
func (c *Client) GetRecording(ctx context.Context, recordingID, isrc, albumName string) (*RecordingMetadata, error) {
	if recordingID != "" {
		return c.GetRecordingByMBID(ctx, recordingID, albumName)
	}
	return c.GetRecordingByISRC(ctx, isrc, albumName)
}

// GetGenres fetches genre information only, preferring lookup by MBID.
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

// GetGenresByISRC fetches genre data for a recording identified by ISRC.
func (c *Client) GetGenresByISRC(ctx context.Context, isrc string) (GenreResult, error) {
	if isrc == "" {
		return GenreResult{}, nil
	}
	u := fmt.Sprintf("%s/recording?query=isrc:%s&inc=tags&fmt=json", c.baseURL, url.QueryEscape(isrc))
	resp, err := c.doGet(ctx, u)
	if err != nil {
		return GenreResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusBadRequest {
		return GenreResult{}, nil
	}
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

// GetGenresByMBID fetches genre data for a recording identified by MusicBrainz ID.
func (c *Client) GetGenresByMBID(ctx context.Context, mbid string) (GenreResult, error) {
	if mbid == "" {
		return GenreResult{}, nil
	}
	u := fmt.Sprintf("%s/recording/%s?inc=tags&fmt=json", c.baseURL, url.PathEscape(mbid))
	resp, err := c.doGet(ctx, u)
	if err != nil {
		return GenreResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
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

// GetRecordingByISRC fetches full metadata for a recording identified by ISRC.
func (c *Client) GetRecordingByISRC(ctx context.Context, isrc string, albumName string) (*RecordingMetadata, error) {
	if isrc == "" {
		return nil, nil
	}
	u := fmt.Sprintf("%s/recording?query=isrc:%s&inc=artists+releases+release-artists+tags+isrcs&fmt=json&limit=1", c.baseURL, url.QueryEscape(isrc))
	resp, err := c.doGet(ctx, u)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusBadRequest {
		return nil, nil
	}
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

	return buildMetadata(result.Recordings[0], result.Recordings, c.genreMap, albumName, isrc), nil
}

// GetRecordingByMBID fetches full metadata for a recording identified by MusicBrainz ID.
func (c *Client) GetRecordingByMBID(ctx context.Context, mbid string, albumName string) (*RecordingMetadata, error) {
	if mbid == "" {
		return nil, nil
	}
	u := fmt.Sprintf("%s/recording/%s?inc=artists+releases+release-groups+artist-credits+tags+isrcs&fmt=json", c.baseURL, url.PathEscape(mbid))
	resp, err := c.doGet(ctx, u)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz returned status %d", resp.StatusCode)
	}

	var rec recording
	if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return buildMetadata(rec, []recording{rec}, c.genreMap, albumName, ""), nil
}

// --------------------------------------------------------------------------
// HTTP helpers
// --------------------------------------------------------------------------

// doGet creates a GET request with standard headers and executes it with retry/rate-limit logic.
func (c *Client) doGet(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	return c.httpClient.Do(ctx, req)
}

// --------------------------------------------------------------------------
// Metadata builders
// --------------------------------------------------------------------------

// buildMetadata constructs a RecordingMetadata from a decoded recording and its sibling
// recordings (used for tag aggregation). Pass the known ISRC when available (ISRC search);
// leave empty when doing an MBID lookup (it will be read from the recording itself).
func buildMetadata(rec recording, recordings []recording, genreMap map[string]string, albumName, isrc string) *RecordingMetadata {
	mainGenre, subGenre := extractMainGenre(recordings, genreMap)
	meta := &RecordingMetadata{
		RecordingID: rec.ID,
		Title:       rec.Title,
		Duration:    rec.Length,
		Genre:       mainGenre,
		SubGenre:    subGenre,
		Tags:        extractTags(recordings),
		ISRC:        isrc,
	}

	if meta.ISRC == "" && len(rec.ISRCs) > 0 {
		meta.ISRC = rec.ISRCs[0]
	}

	populateArtists(meta, rec.ArtistCredit)
	populateRelease(meta, selectBestRelease(rec.Releases, albumName))
	return meta
}

// populateArtists fills artist-related fields on meta from a list of artist credits.
func populateArtists(meta *RecordingMetadata, credits []artistCredit) {
	if len(credits) == 0 {
		return
	}
	meta.Artist = credits[0].Artist.Name
	meta.Artists = make([]string, len(credits))
	meta.ArtistIDs = make([]string, len(credits))
	for i, ac := range credits {
		meta.Artists[i] = ac.Artist.Name
		meta.ArtistIDs[i] = ac.Artist.ID
		if ac.Type == "composer" && meta.Composer == "" {
			meta.Composer = ac.Artist.Name
		}
	}
}

// populateRelease fills release-related fields on meta. No-ops when rel is nil.
func populateRelease(meta *RecordingMetadata, rel *release) {
	if rel == nil {
		return
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
		meta.AlbumArtist = meta.AlbumArtists[0]
	}
}

// --------------------------------------------------------------------------
// Release selection
// --------------------------------------------------------------------------

func selectBestRelease(releases []release, albumName string) *release {
	if len(releases) == 0 {
		return nil
	}
	albumNorm := normalizeString(albumName)
	for i := range releases {
		r := &releases[i]
		releaseNorm := normalizeString(r.Title)
		if albumNorm != "" && releaseNorm != "" &&
			(strings.Contains(releaseNorm, albumNorm) || strings.Contains(albumNorm, releaseNorm)) {
			return r
		}
	}
	return &releases[0]
}

// --------------------------------------------------------------------------
// String normalization
// --------------------------------------------------------------------------

// normalizeString lowercases and strips spaces, hyphens, underscores, and common punctuation.
// Used for fuzzy release and genre matching.
func normalizeString(s string) string {
	s = strings.ToLower(s)
	for _, ch := range []string{" ", "-", "_", ",", "(", ")"} {
		s = strings.ReplaceAll(s, ch, "")
	}
	return s
}

// normalizeGenreKey is a lighter variant that only strips spaces, hyphens, and underscores.
func normalizeGenreKey(s string) string {
	s = strings.ToLower(s)
	for _, ch := range []string{" ", "-", "_"} {
		s = strings.ReplaceAll(s, ch, "")
	}
	return s
}

// --------------------------------------------------------------------------
// Genre / tag extraction
// --------------------------------------------------------------------------

func extractMainGenre(recordings []recording, genreMap map[string]string) (mainGenre string, subGenre string) {
	tagCounts := make(map[string]int)
	for _, rec := range recordings {
		for _, t := range rec.Tags {
			if t.Count <= 0 {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(t.Name))
			if name != "" {
				tagCounts[name] += t.Count
			}
		}
	}
	if len(tagCounts) == 0 {
		return "", ""
	}

	type tagInfo struct {
		name  string
		count int
	}
	tags := make([]tagInfo, 0, len(tagCounts))
	for name, count := range tagCounts {
		tags = append(tags, tagInfo{name: name, count: count})
	}
	sort.SliceStable(tags, func(i, j int) bool {
		if tags[i].count == tags[j].count {
			return tags[i].name < tags[j].name
		}
		return tags[i].count > tags[j].count
	})

	highestTag := tags[0].name
	var maxGenre string
	for _, t := range tags {
		if mapped, ok := genreMap[strings.ToLower(t.name)]; ok {
			maxGenre = mapped
			break
		}
	}
	if maxGenre == "" {
		maxGenre = highestTag
	}

	// Suppress sub_genre when it's just a differently-formatted version of maxGenre.
	if normalizeGenreKey(highestTag) == normalizeGenreKey(maxGenre) {
		return maxGenre, ""
	}
	return maxGenre, highestTag
}

func extractTags(recordings []recording) []string {
	tagsSet := make(map[string]struct{})
	for _, rec := range recordings {
		for _, t := range rec.Tags {
			if t.Count > 0 {
				if name := strings.TrimSpace(t.Name); name != "" {
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

// --------------------------------------------------------------------------
// API response types
// --------------------------------------------------------------------------

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

// --------------------------------------------------------------------------
// Public output type
// --------------------------------------------------------------------------

type RecordingMetadata struct {
	ReleaseType    string
	AlbumArtist    string
	ISRC           string
	Title          string
	SubGenre       string
	Genre          string
	Artist         string
	CatalogNumber  string
	Barcode        string
	Label          string
	ReleaseID      string
	Album          string
	Composer       string
	RecordingID    string
	ReleaseDate    string
	AlbumArtistIDs []string
	AlbumArtists   []string
	ArtistIDs      []string
	Artists        []string
	Tags           []string
	Year           int
	Duration       int
}
