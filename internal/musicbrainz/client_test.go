package musicbrainz

import (
	"testing"
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
			name: "aggregates counts for same main genre",
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
			wantMainGenre: "Metal",
			wantSubGenre:  "pop",
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
