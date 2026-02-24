package musicbrainz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestExtractMainGenre(t *testing.T) {
	tests := []struct { //nolint:govet
		recordings    []recording
		genreMap      map[string]string
		name          string
		wantMainGenre string
		wantSubGenre  string
	}{
		{
			name: "maps sub-genres to main genre",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "death metal", Count: 5},
						{Name: "thrash metal", Count: 3},
						{Name: "rock", Count: 2},
					},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "Metal",
			wantSubGenre:  "death metal",
		},
		{
			name: "uses original tag when no match",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "obscure genre", Count: 10},
						{Name: "another unknown", Count: 5},
					},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "obscure genre",
			wantSubGenre:  "",
		},
		{
			name: "selects highest tag ignoring category aggregation",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "death metal", Count: 5},
						{Name: "black metal", Count: 4},
						{Name: "pop", Count: 8},
					},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "Pop",
			wantSubGenre:  "",
		},
		{
			name: "returns empty when no tags",
			recordings: []recording{
				{
					Tags: []tag{},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "",
			wantSubGenre:  "",
		},
		{
			name: "ignores tags with zero count",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "rock", Count: 0},
						{Name: "metal", Count: 5},
					},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "Metal",
			wantSubGenre:  "",
		},
		{
			name: "handles case-insensitive matching",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "DEATH METAL", Count: 5},
						{Name: "Thrash Metal", Count: 3},
					},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "Metal",
			wantSubGenre:  "DEATH METAL",
		},
		{
			name: "custom genre map overrides default",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "synthwave", Count: 10},
						{Name: "vaporwave", Count: 5},
					},
				},
			},
			genreMap: map[string]string{
				"synthwave": "Electronic",
				"vaporwave": "Electronic",
			},
			wantMainGenre: "Electronic",
			wantSubGenre:  "synthwave",
		},
		{
			name: "multiple recordings aggregate",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "indie rock", Count: 3},
					},
				},
				{
					Tags: []tag{
						{Name: "alternative rock", Count: 5},
					},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "Rock",
			wantSubGenre:  "alternative rock",
		},
		{
			// Regression: "hip hop" (highest raw count) is equivalent to "Hip-Hop" after
			// normalisation — it must NOT leak through as sub_genre.
			name: "suppresses sub_genre when highest tag is same as canonical after normalisation",
			recordings: []recording{
				{
					Tags: []tag{
						{Name: "hip hop", Count: 10},
						{Name: "hip-hop/rap", Count: 8},
						{Name: "rap/hip hop", Count: 6},
						{Name: "melodic rap", Count: 4},
					},
				},
			},
			genreMap:      DefaultGenreMap,
			wantMainGenre: "Hip-Hop",
			wantSubGenre:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainGenre, subGenre := extractMainGenre(tt.recordings, tt.genreMap)
			if mainGenre != tt.wantMainGenre {
				t.Errorf("mainGenre = %q, want %q", mainGenre, tt.wantMainGenre)
			}
			if subGenre != tt.wantSubGenre {
				t.Errorf("subGenre = %q, want %q", subGenre, tt.wantSubGenre)
			}
		})
	}
}

func TestDefaultGenreMapContainsExpectedMappings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"death metal", "Metal"},
		{"indie pop", "Pop"},
		{"hip hop", "Hip-Hop"},
		{"drill", "Hip-Hop"},
		{"corridos tumbados", "Regional Mexican"},
		{"norteño", "Regional Mexican"},
		{"reggaeton", "Latin"},
		{"dubstep", "Electronic"},
		{"neo soul", "R&B"},
		{"soundtrack", "Soundtrack"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, ok := DefaultGenreMap[tt.input]
			if !ok {
				t.Errorf("DefaultGenreMap missing key %q", tt.input)
				return
			}
			if result != tt.expected {
				t.Errorf("DefaultGenreMap[%q] = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClient_ConcurrentRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting concurrency test in short mode")
	}

	var mu sync.Mutex
	var timestamps []time.Time

	// Mock server simply records the time it received the request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		timestamps = append(timestamps, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recordings":[]}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)

	// We'll spin up 10 goroutines that all try to make a request at the exact same time.
	numRequests := 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	// A channel to block all goroutines until ready, to maximize concurrency contention
	ready := make(chan struct{})

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()
			<-ready

			_, err := client.doGet(context.Background(), ts.URL)
			if err != nil {
				t.Errorf("Request failed: %v", err)
			}
		}()
	}

	// Release the hounds!
	close(ready)
	wg.Wait()

	if len(timestamps) != numRequests {
		t.Fatalf("Expected %d successful requests, got %d", numRequests, len(timestamps))
	}

	// Sort timestamps in case network delivery was slightly out of order
	sort.SliceStable(timestamps, func(i, j int) bool {
		return timestamps[i].Before(timestamps[j])
	})

	// Check intervals
	// We allow a tiny bit of leeway (e.g., 50ms) for timer overhead,
	// but the spacing must be >= minRequestInterval ideally, or very close to it.
	for i := 1; i < len(timestamps); i++ {
		diff := timestamps[i].Sub(timestamps[i-1])
		// 1550ms is the interval. We accept anything >= 1500ms to avoid flakes on busy CI runners
		if diff < minRequestInterval-50*time.Millisecond {
			t.Errorf("Requests too close! Request %d and %d separated by %v, expected >= ~%v", i-1, i, diff, minRequestInterval)
		}
	}
}

func TestClient_RetryAfterHeader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting concurrency test in short mode")
	}

	var mu sync.Mutex
	var requests int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqCount := requests
		requests++
		mu.Unlock()

		if reqCount == 0 {
			// First request, tell them to wait 2 seconds
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recordings":[]}`))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)

	// This should block for at least 2 seconds because of the Retry-After header
	start := time.Now()
	_, err := client.doGet(context.Background(), ts.URL)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Request failed: %v", err)
	}

	mu.Lock()
	totalReqs := requests
	mu.Unlock()

	if totalReqs != 2 {
		t.Errorf("Expected 2 requests total (1 rejected, 1 success), got %d", totalReqs)
	}

	if elapsed < 2*time.Second {
		t.Errorf("Expected request to block for at least 2 seconds due to Retry-After, got %v", elapsed)
	}
}
