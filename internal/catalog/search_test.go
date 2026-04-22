package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHifiProviderSearch_ReturnsErrorOnUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failure", http.StatusBadGateway)
	}))
	defer srv.Close()

	provider := NewHifiProvider(srv.URL)
	types := []string{"artist", "album", "track", "playlist", "unexpected"}

	for _, searchType := range types {
		t.Run(searchType, func(t *testing.T) {
			res, err := provider.Search(context.Background(), "test", searchType)
			if err == nil {
				t.Fatalf("expected error for search type %q, got nil", searchType)
			}
			if res != nil {
				t.Fatalf("expected nil result on error for search type %q", searchType)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("API request failed: %d", http.StatusBadGateway)) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
