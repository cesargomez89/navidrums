package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// FlexCover handles flexible cover image formats from the API
type FlexCover []string

// UnmarshalJSON implements custom JSON unmarshaling for FlexCover
func (f *FlexCover) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Handle string format
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = []string{s}
		return nil
	}

	// Handle array format with objects
	if data[0] == '[' {
		var items []struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		var urls []string
		for _, item := range items {
			urls = append(urls, item.URL)
		}
		*f = urls
		return nil
	}

	// Handle object format
	if data[0] == '{' {
		var item struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		*f = []string{item.URL}
		return nil
	}

	return nil
}

// formatID converts various ID types to string
func formatID(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case json.Number:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// manifestResponse represents the manifest data from the streaming endpoint
type manifestResponse struct {
	Data struct {
		Manifest         string `json:"manifest"`
		ManifestMimeType string `json:"manifestMimeType"`
	} `json:"data"`
}

// btsManifest represents the BTS manifest format
type btsManifest struct {
	Urls []string `json:"urls"`
}

// multiSegmentReader implements io.ReadCloser for segmented DASH streams
type multiSegmentReader struct {
	urls     []string
	client   *http.Client
	ctx      context.Context
	currIdx  int
	currBody io.ReadCloser
}

func (r *multiSegmentReader) Read(p []byte) (n int, err error) {
	if r.currBody == nil {
		if r.currIdx >= len(r.urls) {
			return 0, io.EOF
		}
		// Fetch next segment
		req, err := http.NewRequestWithContext(r.ctx, "GET", r.urls[r.currIdx], nil)
		if err != nil {
			return 0, err
		}
		resp, err := r.client.Do(req)
		if err != nil {
			return 0, err
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return 0, fmt.Errorf("segment fetch failed (%d): %s", r.currIdx, resp.Status)
		}
		r.currBody = resp.Body
		r.currIdx++
	}

	n, err = r.currBody.Read(p)
	if err == io.EOF {
		r.currBody.Close()
		r.currBody = nil
		return r.Read(p) // recursive call to next segment
	}
	return n, err
}

func (r *multiSegmentReader) Close() error {
	if r.currBody != nil {
		return r.currBody.Close()
	}
	return nil
}
