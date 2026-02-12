package constants

import (
	"testing"
	"time"
)

func TestDefaultValues(t *testing.T) {
	// Test that default values are set correctly
	if DefaultPort != "8080" {
		t.Errorf("Expected DefaultPort to be '8080', got '%s'", DefaultPort)
	}

	if DefaultDBPath != "navidrums.db" {
		t.Errorf("Expected DefaultDBPath to be 'navidrums.db', got '%s'", DefaultDBPath)
	}

	if DefaultQuality != "LOSSLESS" {
		t.Errorf("Expected DefaultQuality to be 'LOSSLESS', got '%s'", DefaultQuality)
	}

	if DefaultProviderURL != "http://127.0.0.1:8000" {
		t.Errorf("Expected DefaultProviderURL to be 'http://127.0.0.1:8000', got '%s'", DefaultProviderURL)
	}
}

func TestQualityLevels(t *testing.T) {
	qualities := []string{
		QualityLossless,
		QualityHiResLossless,
		QualityHigh,
		QualityLow,
	}

	for _, q := range qualities {
		if q == "" {
			t.Error("Quality constant should not be empty")
		}
	}
}

func TestImageSizes(t *testing.T) {
	sizes := []string{
		ImageSizeSmall,
		ImageSizeMedium,
		ImageSizeLarge,
	}

	for _, s := range sizes {
		if s == "" {
			t.Error("Image size constant should not be empty")
		}
	}
}

func TestTimeouts(t *testing.T) {
	if DefaultHTTPTimeout != 5*time.Minute {
		t.Errorf("Expected DefaultHTTPTimeout to be 5 minutes, got %v", DefaultHTTPTimeout)
	}

	if DefaultPollInterval != 2*time.Second {
		t.Errorf("Expected DefaultPollInterval to be 2 seconds, got %v", DefaultPollInterval)
	}

	if DefaultRetryBase != 1*time.Second {
		t.Errorf("Expected DefaultRetryBase to be 1 second, got %v", DefaultRetryBase)
	}
}

func TestRetryCount(t *testing.T) {
	if DefaultRetryCount != 3 {
		t.Errorf("Expected DefaultRetryCount to be 3, got %d", DefaultRetryCount)
	}
}

func TestConcurrency(t *testing.T) {
	if DefaultConcurrency != 2 {
		t.Errorf("Expected DefaultConcurrency to be 2, got %d", DefaultConcurrency)
	}
}

func TestEndpoints(t *testing.T) {
	endpoints := []string{
		EndpointSearch,
		EndpointInfo,
		EndpointTrack,
		EndpointArtist,
		EndpointAlbum,
		EndpointPlaylist,
		EndpointRecommendations,
		EndpointLyrics,
		EndpointArtistSimilar,
		EndpointAlbumSimilar,
		EndpointMix,
		EndpointCover,
	}

	for _, e := range endpoints {
		if e == "" {
			t.Error("Endpoint constant should not be empty")
		}
		// Should start with /
		if e[0] != '/' {
			t.Errorf("Endpoint %s should start with /", e)
		}
	}
}

func TestMimeTypes(t *testing.T) {
	types := []string{
		MimeTypeBTS,
		MimeTypeDashXML,
		MimeTypeFLAC,
		MimeTypeMP3,
		MimeTypeMP4,
		MimeTypeJPEG,
	}

	for _, m := range types {
		if m == "" {
			t.Error("MIME type constant should not be empty")
		}
	}
}

func TestStatusValues(t *testing.T) {
	statuses := []string{
		StatusQueued,
		StatusResolve,
		StatusDownloading,
		StatusCompleted,
		StatusFailed,
		StatusCancelled,
	}

	for _, s := range statuses {
		if s == "" {
			t.Error("Status constant should not be empty")
		}
	}
}

func TestJobTypes(t *testing.T) {
	types := []string{
		TypeTrack,
		TypeAlbum,
		TypePlaylist,
		TypeArtist,
	}

	for _, jt := range types {
		if jt == "" {
			t.Error("Job type constant should not be empty")
		}
	}
}

func TestFileExtensions(t *testing.T) {
	extensions := []string{
		ExtFLAC,
		ExtMP3,
		ExtMP4,
		ExtM4A,
		ExtM3U,
		ExtJPG,
	}

	for _, ext := range extensions {
		if ext == "" {
			t.Error("File extension constant should not be empty")
		}
		// Should start with .
		if ext[0] != '.' {
			t.Errorf("File extension %s should start with .", ext)
		}
	}
}

func TestHTTPStatusCodes(t *testing.T) {
	if StatusOK != 200 {
		t.Errorf("Expected StatusOK to be 200, got %d", StatusOK)
	}

	if StatusBadRequest != 400 {
		t.Errorf("Expected StatusBadRequest to be 400, got %d", StatusBadRequest)
	}

	if StatusNotFound != 404 {
		t.Errorf("Expected StatusNotFound to be 404, got %d", StatusNotFound)
	}

	if StatusInternalError != 500 {
		t.Errorf("Expected StatusInternalError to be 500, got %d", StatusInternalError)
	}
}

func TestInvalidPathChars(t *testing.T) {
	if InvalidPathChars == "" {
		t.Error("InvalidPathChars should not be empty")
	}
}
