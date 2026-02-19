package catalog

import (
	"encoding/json"
	"testing"
)

func TestFormatID(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string input", "12345", "12345"},
		{"float64 input", float64(12345), "12345"},
		{"float64 with decimals", float64(12345.67), "12346"},
		{"json.Number input", json.Number("12345"), "12345"},
		{"int input", 12345, "12345"},
		{"empty string", "", ""},
		{"nil input", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatID(tt.input)
			if result != tt.expected {
				t.Errorf("formatID(%v) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseYear(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid date", "2023-05-15", 2023},
		{"valid year only", "2023", 2023},
		{"short year", "23", 0},
		{"empty string", "", 0},
		{"invalid format", "not-a-date", 0},
		{"date with time", "2023-12-31T23:59:59Z", 2023},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseYear(tt.input)
			if result != tt.expected {
				t.Errorf("parseYear(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFlexCover_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "string format",
			input:    `"image.jpg"`,
			expected: []string{"image.jpg"},
		},
		{
			name:     "array of objects",
			input:    `[{"url": "img1.jpg"}, {"url": "img2.jpg"}]`,
			expected: []string{"img1.jpg", "img2.jpg"},
		},
		{
			name:     "object format",
			input:    `{"url": "single.jpg"}`,
			expected: []string{"single.jpg"},
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cover FlexCover
			err := json.Unmarshal([]byte(tt.input), &cover)
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}
			if len(cover) != len(tt.expected) {
				t.Errorf("UnmarshalJSON got %d elements, want %d", len(cover), len(tt.expected))
				return
			}
			for i := range cover {
				if cover[i] != tt.expected[i] {
					t.Errorf("cover[%d] = %s, want %s", i, cover[i], tt.expected[i])
				}
			}
		})
	}
}
