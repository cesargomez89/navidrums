package dto

import (
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{Field: "title", Message: "is required"}
	if err.Error() != "title: is required" {
		t.Errorf("Error() = %q, want %q", err.Error(), "title: is required")
	}
}

func TestValidationError_ToMap(t *testing.T) {
	err := ValidationError{Field: "title", Message: "is required"}
	m := err.ToMap()
	if m["title"] != "is required" {
		t.Errorf("ToMap() = %v, want {title: is required}", m)
	}
}

func TestToMap(t *testing.T) {
	errs := []ValidationError{
		{Field: "title", Message: "is required"},
		{Field: "year", Message: "must be between 1900 and 2100"},
	}
	m := ToMap(errs)
	if len(m) != 2 {
		t.Errorf("ToMap() returned %d items, want 2", len(m))
	}
	if m["title"] != "is required" {
		t.Errorf("ToMap()[title] = %q, want %q", m["title"], "is required")
	}
	if m["year"] != "must be between 1900 and 2100" {
		t.Errorf("ToMap()[year] = %q, want %q", m["year"], "must be between 1900 and 2100")
	}
}

func TestToResponse(t *testing.T) {
	errs := []ValidationError{
		{Field: "title", Message: "is required"},
		{Field: "year", Message: "invalid"},
	}
	resp := ToResponse(errs)
	expected := "title: is required; year: invalid"
	if resp != expected {
		t.Errorf("ToResponse() = %q, want %q", resp, expected)
	}
}

func TestValidateISRC(t *testing.T) {
	tests := []struct {
		isrc     *string
		name     string
		wantErrs int
	}{
		{nil, "nil isrc", 0},
		{strPtr(""), "empty isrc", 0},
		{strPtr("USRC17607839"), "valid isrc uppercase", 0},
		{strPtr("usrc17607839"), "valid isrc lowercase", 0},
		{strPtr("USRC123"), "invalid isrc - too short", 1},
		{strPtr("INVALID-ISRC"), "invalid isrc - bad format", 1},
		{strPtr("GBAHT2300001"), "valid isrc - typical format", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateISRC(tt.isrc)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateISRC() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateReleaseDate(t *testing.T) {
	tests := []struct {
		date     *string
		name     string
		wantErrs int
	}{
		{nil, "nil date", 0},
		{strPtr(""), "empty date", 0},
		{strPtr("2023"), "valid YYYY", 0},
		{strPtr("2023-05"), "valid YYYY-MM", 0},
		{strPtr("2023-05-15"), "valid YYYY-MM-DD", 0},
		{strPtr("2023-05-15T10:30:00Z"), "invalid - full ISO", 1},
		{strPtr("15-05-2023"), "invalid - wrong format", 1},
		{strPtr("2023-0"), "invalid - partial", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateReleaseDate(tt.date)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateReleaseDate() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		url      *string
		name     string
		wantErrs int
	}{
		{nil, "nil url", 0},
		{strPtr(""), "empty url", 0},
		{strPtr("http://example.com"), "valid http", 0},
		{strPtr("https://example.com/path?query=1"), "valid https", 0},
		{strPtr("not a url"), "invalid url", 1},
		{strPtr("example.com"), "invalid - missing scheme", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateURL(tt.url)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateURL() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateKeyScale(t *testing.T) {
	tests := []struct {
		keyScale *string
		name     string
		wantErrs int
	}{
		{nil, "nil keyScale", 0},
		{strPtr(""), "empty keyScale", 0},
		{strPtr("major"), "valid major", 0},
		{strPtr("minor"), "valid minor", 0},
		{strPtr("Major"), "invalid - Major", 1},
		{strPtr("MINOR"), "invalid - MINOR", 1},
		{strPtr("dorian"), "invalid - other", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateKeyScale(tt.keyScale)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateKeyScale() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateYear(t *testing.T) {
	tests := []struct {
		year     *int
		name     string
		wantErrs int
	}{
		{nil, "nil year", 0},
		{intPtr(1900), "valid 1900", 0},
		{intPtr(2023), "valid 2023", 0},
		{intPtr(2100), "valid 2100", 0},
		{intPtr(1899), "invalid - too low", 1},
		{intPtr(2101), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateYear(tt.year)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateYear() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateBPM(t *testing.T) {
	tests := []struct {
		bpm      *int
		name     string
		wantErrs int
	}{
		{nil, "nil bpm", 0},
		{intPtr(1), "valid 1", 0},
		{intPtr(120), "valid 120", 0},
		{intPtr(999), "valid 999", 0},
		{intPtr(0), "invalid - zero", 1},
		{intPtr(-1), "invalid - negative", 1},
		{intPtr(1000), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateBPM(tt.bpm)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateBPM() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateTrackNumber(t *testing.T) {
	tests := []struct {
		num      *int
		name     string
		wantErrs int
	}{
		{nil, "nil track number", 0},
		{intPtr(0), "valid 0", 0},
		{intPtr(1), "valid 1", 0},
		{intPtr(9999), "valid 9999", 0},
		{intPtr(-1), "invalid - negative", 1},
		{intPtr(10000), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateTrackNumber(tt.num)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateTrackNumber() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateDiscNumber(t *testing.T) {
	tests := []struct {
		num      *int
		name     string
		wantErrs int
	}{
		{nil, "nil disc number", 0},
		{intPtr(0), "valid 0", 0},
		{intPtr(1), "valid 1", 0},
		{intPtr(99), "valid 99", 0},
		{intPtr(-1), "invalid - negative", 1},
		{intPtr(100), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateDiscNumber(tt.num)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateDiscNumber() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateTotalTracks(t *testing.T) {
	tests := []struct {
		num      *int
		name     string
		wantErrs int
	}{
		{nil, "nil total tracks", 0},
		{intPtr(0), "valid 0", 0},
		{intPtr(10), "valid 10", 0},
		{intPtr(9999), "valid 9999", 0},
		{intPtr(-1), "invalid - negative", 1},
		{intPtr(10000), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateTotalTracks(tt.num)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateTotalTracks() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateTotalDiscs(t *testing.T) {
	tests := []struct {
		num      *int
		name     string
		wantErrs int
	}{
		{nil, "nil total discs", 0},
		{intPtr(0), "valid 0", 0},
		{intPtr(1), "valid 1", 0},
		{intPtr(99), "valid 99", 0},
		{intPtr(-1), "invalid - negative", 1},
		{intPtr(100), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateTotalDiscs(tt.num)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateTotalDiscs() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidateReplayGain(t *testing.T) {
	tests := []struct {
		val      *float64
		name     string
		wantErrs int
	}{
		{nil, "nil replay gain", 0},
		{floatPtr(-30.0), "valid -30", 0},
		{floatPtr(0.0), "valid 0", 0},
		{floatPtr(30.0), "valid 30", 0},
		{floatPtr(-6.5), "valid -6.5", 0},
		{floatPtr(-30.1), "invalid - too low", 1},
		{floatPtr(30.1), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateReplayGain(tt.val)
			if len(errs) != tt.wantErrs {
				t.Errorf("validateReplayGain() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestValidatePeak(t *testing.T) {
	tests := []struct {
		val      *float64
		name     string
		wantErrs int
	}{
		{nil, "nil peak", 0},
		{floatPtr(0.0), "valid 0", 0},
		{floatPtr(1.0), "valid 1.0", 0},
		{floatPtr(2.0), "valid 2.0", 0},
		{floatPtr(0.95), "valid 0.95", 0},
		{floatPtr(-0.1), "invalid - negative", 1},
		{floatPtr(2.1), "invalid - too high", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validatePeak(tt.val)
			if len(errs) != tt.wantErrs {
				t.Errorf("validatePeak() returned %d errors, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestTrackUpdateRequest_Validate(t *testing.T) {
	tests := []struct { //nolint:govet // test struct, fieldalignment not critical
		wantErrs int
		req      TrackUpdateRequest
		name     string
	}{
		{
			wantErrs: 0,
			req:      TrackUpdateRequest{},
			name:     "empty request",
		},
		{
			wantErrs: 0,
			req: TrackUpdateRequest{
				Title:       strPtr("Test Track"),
				Year:        intPtr(2023),
				BPM:         intPtr(120),
				TrackNumber: intPtr(1),
				KeyScale:    strPtr("major"),
				ISRC:        strPtr("USRC17607839"),
			},
			name: "valid request",
		},
		{
			wantErrs: 4,
			req: TrackUpdateRequest{
				Year:     intPtr(1800),
				BPM:      intPtr(0),
				KeyScale: strPtr("invalid"),
				ISRC:     strPtr("bad"),
			},
			name: "multiple errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.req.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("Validate() returned %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestTrackUpdateRequest_ToUpdates(t *testing.T) {
	req := TrackUpdateRequest{
		Title:       strPtr("New Title"),
		Artist:      strPtr(""),
		Year:        intPtr(2023),
		BPM:         intPtr(120),
		Compilation: boolPtr(true),
	}

	updates := req.ToUpdates()

	if len(updates) != 4 {
		t.Errorf("ToUpdates() returned %d items, want 4", len(updates))
	}
	if updates["title"] != "New Title" {
		t.Errorf("ToUpdates()[title] = %v, want 'New Title'", updates["title"])
	}
	if updates["year"] != 2023 {
		t.Errorf("ToUpdates()[year] = %v, want 2023", updates["year"])
	}
	if updates["bpm"] != 120 {
		t.Errorf("ToUpdates()[bpm] = %v, want 120", updates["bpm"])
	}
	if updates["compilation"] != true {
		t.Errorf("ToUpdates()[compilation] = %v, want true", updates["compilation"])
	}

	if _, ok := updates["artist"]; ok {
		t.Error("ToUpdates() should not include empty string fields")
	}
}

func TestJobResponse_NewJobResponse(t *testing.T) {
	now := parseTime("2023-06-15T10:30:00Z")
	errMsg := "download failed"
	job := &domain.Job{
		ID:        "job_123",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusFailed,
		SourceID:  "track_456",
		Progress:  75.5,
		CreatedAt: now,
		UpdatedAt: now,
		Error:     &errMsg,
	}

	resp := NewJobResponse(job)

	if resp.ID != "job_123" {
		t.Errorf("ID = %q, want %q", resp.ID, "job_123")
	}
	if resp.Type != "track" {
		t.Errorf("Type = %q, want %q", resp.Type, "track")
	}
	if resp.Status != "failed" {
		t.Errorf("Status = %q, want %q", resp.Status, "failed")
	}
	if resp.SourceID != "track_456" {
		t.Errorf("SourceID = %q, want %q", resp.SourceID, "track_456")
	}
	if resp.Progress != 75.5 {
		t.Errorf("Progress = %f, want 75.5", resp.Progress)
	}
	if resp.Error != "download failed" {
		t.Errorf("Error = %q, want %q", resp.Error, "download failed")
	}
}

func TestJobResponse_NewJobResponse_NilError(t *testing.T) {
	now := parseTime("2023-06-15T10:30:00Z")
	job := &domain.Job{
		ID:        "job_123",
		Type:      domain.JobTypeTrack,
		Status:    domain.JobStatusCompleted,
		SourceID:  "track_456",
		Progress:  100,
		CreatedAt: now,
		UpdatedAt: now,
		Error:     nil,
	}

	resp := NewJobResponse(job)

	if resp.Error != "" {
		t.Errorf("Error = %q, want empty string", resp.Error)
	}
}

func TestNewTrackResponse(t *testing.T) {
	now := parseTime("2023-06-15T10:30:00Z")
	completed := parseTime("2023-06-15T11:00:00Z")
	track := &domain.Track{
		ID:            1,
		ProviderID:    "track_123",
		Title:         "Test Track",
		Artist:        "Test Artist",
		Album:         "Test Album",
		AlbumArtist:   "Test Album Artist",
		AlbumID:       "album_456",
		ReleaseID:     "release_789",
		Genre:         "Rock",
		Label:         "Test Label",
		TrackNumber:   1,
		DiscNumber:    1,
		TotalTracks:   10,
		TotalDiscs:    1,
		Year:          2023,
		Duration:      180,
		BPM:           120,
		ReplayGain:    -6.5,
		Peak:          0.95,
		Key:           "C",
		KeyScale:      "major",
		Composer:      "Test Composer",
		Copyright:     "2023 Test",
		ISRC:          "USRC17607839",
		AudioQuality:  "LOSSLESS",
		AudioModes:    "STEREO",
		FilePath:      "/path/to/track.flac",
		FileExtension: ".flac",
		Status:        domain.TrackStatusCompleted,
		Artists:       []string{"Artist 1", "Artist 2"},
		AlbumArtists:  []string{"Album Artist"},
		CreatedAt:     now,
		UpdatedAt:     now,
		CompletedAt:   &completed,
	}

	resp := NewTrackResponse(track)

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Title != "Test Track" {
		t.Errorf("Title = %q, want %q", resp.Title, "Test Track")
	}
	if resp.Artist != "Test Artist" {
		t.Errorf("Artist = %q, want %q", resp.Artist, "Test Artist")
	}
	if len(resp.Artists) != 2 {
		t.Errorf("Artists length = %d, want 2", len(resp.Artists))
	}
	if resp.Status != "completed" {
		t.Errorf("Status = %q, want %q", resp.Status, "completed")
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
