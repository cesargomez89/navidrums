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
	tests := []struct {
		genreMap      map[string]string
		name          string
		wantMainGenre string
		recordings    []recording
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
			wantMainGenre: "metal",
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
			wantMainGenre: "pop",
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
			wantMainGenre: "metal",
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
			wantMainGenre: "metal",
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
				"synthwave": "electronic",
				"vaporwave": "electronic",
			},
			wantMainGenre: "electronic",
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
			wantMainGenre: "rock",
		},
		{
			name: "uses highest count tag after genre mapping",
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
			wantMainGenre: "hip-hop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainGenre := extractMainGenre(tt.recordings, tt.genreMap)
			if mainGenre != tt.wantMainGenre {
				t.Errorf("mainGenre = %q, want %q", mainGenre, tt.wantMainGenre)
			}
		})
	}
}

func TestDefaultGenreMapContainsExpectedMappings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"death metal", "metal"},
		{"indie pop", "pop"},
		{"hip hop", "hip-hop"},
		{"drill", "hip-hop"},
		{"corridos tumbados", "regional mexican"},
		{"norteño", "regional mexican"},
		{"reggaeton", "latin"},
		{"dubstep", "electronic"},
		{"neo soul", "r&b"},
		{"soundtrack", "soundtrack"},
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
		// 1100ms is the interval. We accept anything >= 1050ms to avoid flakes on busy CI runners
		if diff < minRequestInterval-50*time.Millisecond {
			t.Errorf("Requests too close! Request %d and %d separated by %v, expected >= ~%v", i-1, i, diff, minRequestInterval)
		}
	}
}

func TestClient_RetryAfterHeader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting test in short mode")
	}

	var mu sync.Mutex
	var requests int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()

		// Always return 429 to test immediate failure behavior
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)

	start := time.Now()
	resp, err := client.doGet(context.Background(), ts.URL)
	elapsed := time.Since(start)

	// Should return response (not error) with 429 status - caller handles status check
	if err != nil {
		t.Errorf("doGet should not return error, got: %v", err)
	}
	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}
	defer resp.Body.Close()

	// Verify it's actually 429
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", resp.StatusCode)
	}

	mu.Lock()
	totalReqs := requests
	mu.Unlock()

	// Only 1 request should be made (no retries)
	if totalReqs != 1 {
		t.Errorf("Expected 1 request (no retries), got %d", totalReqs)
	}

	// Should be fast (no Retry-After waiting, no retries)
	if elapsed > 500*time.Millisecond {
		t.Errorf("Expected fast response, got %v", elapsed)
	}
}
