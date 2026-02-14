package storage

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal Name", "Normal Name"},
		{"Slash/Name", "SlashName"},
		{"Colon:Name", "ColonName"},
		{"Trailing Dot.", "Trailing Dot"},
		{"AC/DC", "ACDC"},
		{"<Invalid>", "Invalid"},
	}

	for _, tt := range tests {
		got := Sanitize(tt.input)
		if got != tt.expected {
			t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
