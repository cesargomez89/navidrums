package catalog

import (
	"testing"

	"github.com/cesargomez89/navidrums/internal/constants"
)

func TestResolveAudioQuality(t *testing.T) {
	tests := []struct {
		name         string
		audioQuality string
		want         string
		tags         []string
	}{
		{
			name:         "HIRES_LOSSLESS tag",
			audioQuality: constants.QualityLossless,
			tags:         []string{"LOSSLESS", "HIRES_LOSSLESS"},
			want:         constants.QualityHiResLossless,
		},
		{
			name:         "HI_RES_LOSSLESS tag",
			audioQuality: constants.QualityLossless,
			tags:         []string{"LOSSLESS", "HI_RES_LOSSLESS"},
			want:         constants.QualityHiResLossless,
		},
		{
			name:         "No HIRES tag",
			audioQuality: constants.QualityLossless,
			tags:         []string{"LOSSLESS"},
			want:         constants.QualityLossless,
		},
		{
			name:         "Empty tags",
			audioQuality: constants.QualityHiResLossless,
			tags:         []string{},
			want:         constants.QualityHiResLossless,
		},
		{
			name:         "Empty tags with HIGH",
			audioQuality: constants.QualityHigh,
			tags:         []string{},
			want:         constants.QualityHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveAudioQuality(tt.audioQuality, tt.tags)
			if got != tt.want {
				t.Errorf("resolveAudioQuality(%q, %v) = %q, want %q", tt.audioQuality, tt.tags, got, tt.want)
			}
		})
	}
}
