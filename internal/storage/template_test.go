package storage

import (
	"path/filepath"
	"testing"
)

func TestBuildPath(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		data       *PathTemplateData
		want       string
		wantErr    bool
		errContain string
	}{
		{
			name:     "default template",
			template: "{{.AlbumArtist}}/{{.OriginalYear}} - {{.Album}}/{{.Disc}}-{{.Track}} {{.Title}}",
			data: &PathTemplateData{
				AlbumArtist:  "Pink Floyd",
				OriginalYear: 1973,
				Album:        "The Dark Side of the Moon",
				Disc:         "01",
				Track:        "01",
				Title:        "Speak to Me",
			},
			want:    "Pink Floyd/1973 - The Dark Side of the Moon/01-01 Speak to Me",
			wantErr: false,
		},
		{
			name:     "custom template",
			template: "{{.AlbumArtist}} - {{.Album}}/{{.Track}}. {{.Title}}",
			data: &PathTemplateData{
				AlbumArtist:  "The Beatles",
				OriginalYear: 1969,
				Album:        "Abbey Road",
				Disc:         "01",
				Track:        "05",
				Title:        "Something",
			},
			want:    "The Beatles - Abbey Road/05. Something",
			wantErr: false,
		},
		{
			name:     "template with only filename",
			template: "{{.Track}} - {{.Title}}",
			data: &PathTemplateData{
				AlbumArtist:  "Artist",
				OriginalYear: 2020,
				Album:        "Album",
				Disc:         "01",
				Track:        "10",
				Title:        "Song Title",
			},
			want:    "10 - Song Title",
			wantErr: false,
		},
		{
			name:     "invalid template syntax",
			template: "{{.AlbumArtist",
			data: &PathTemplateData{
				AlbumArtist: "Test",
			},
			wantErr:    true,
			errContain: "failed to parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildPath(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContain != "" {
				if err == nil || !contains(err.Error(), tt.errContain) {
					t.Errorf("BuildPath() error = %v, should contain %v", err, tt.errContain)
				}
				return
			}
			if got != tt.want {
				t.Errorf("BuildPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildPathTemplateData(t *testing.T) {
	data := BuildPathTemplateData(
		"Pink Floyd",
		1973,
		"The Dark Side of the Moon",
		1,
		5,
		"Money",
	)

	if data.AlbumArtist != "Pink Floyd" {
		t.Errorf("AlbumArtist = %v, want Pink Floyd", data.AlbumArtist)
	}
	if data.OriginalYear != 1973 {
		t.Errorf("OriginalYear = %v, want 1973", data.OriginalYear)
	}
	if data.Album != "The Dark Side of the Moon" {
		t.Errorf("Album = %v, want The Dark Side of the Moon", data.Album)
	}
	if data.Disc != "01" {
		t.Errorf("Disc = %v, want 01", data.Disc)
	}
	if data.Track != "05" {
		t.Errorf("Track = %v, want 05", data.Track)
	}
	if data.Title != "Money" {
		t.Errorf("Title = %v, want Money", data.Title)
	}
}

func TestBuildPathTemplateData_Sanitization(t *testing.T) {
	data := BuildPathTemplateData(
		"AC/DC",
		1980,
		"Back In Black<>:\"/\\|?*",
		1,
		1,
		"Hells Bells.. ",
	)

	// Album should have invalid chars removed
	if data.Album != "Back In Black" {
		t.Errorf("Album sanitization failed, got %v", data.Album)
	}

	// Title should have trailing dots/spaces removed
	if data.Title != "Hells Bells" {
		t.Errorf("Title sanitization failed, got %v", data.Title)
	}

	// AlbumArtist should have / removed
	if data.AlbumArtist != "ACDC" {
		t.Errorf("AlbumArtist sanitization failed, got %v", data.AlbumArtist)
	}
}

func TestBuildFullPath(t *testing.T) {
	data := &PathTemplateData{
		AlbumArtist:  "Artist",
		OriginalYear: 2020,
		Album:        "Album",
		Disc:         "01",
		Track:        "01",
		Title:        "Song",
	}

	tests := []struct {
		name         string
		downloadsDir string
		template     string
		ext          string
		want         string
		wantErr      bool
	}{
		{
			name:         "with dot extension",
			downloadsDir: "/home/user/Music",
			template:     "{{.AlbumArtist}}/{{.Title}}",
			ext:          ".flac",
			want:         filepath.Join("/home/user/Music", "Artist/Song.flac"),
			wantErr:      false,
		},
		{
			name:         "without dot extension",
			downloadsDir: "/home/user/Music",
			template:     "{{.AlbumArtist}}/{{.Title}}",
			ext:          "mp3",
			want:         filepath.Join("/home/user/Music", "Artist/Song.mp3"),
			wantErr:      false,
		},
		{
			name:         "empty extension",
			downloadsDir: "/home/user/Music",
			template:     "{{.AlbumArtist}}/{{.Title}}",
			ext:          "",
			want:         filepath.Join("/home/user/Music", "Artist/Song"),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildFullPath(tt.downloadsDir, tt.template, data, tt.ext)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildFullPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuildFullPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatTrackNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{1, "01"},
		{5, "05"},
		{10, "10"},
		{99, "99"},
		{100, "100"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatTrackNumber(tt.input)
			if got != tt.want {
				t.Errorf("FormatTrackNumber(%d) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDiscNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{1, "01"},
		{2, "02"},
		{10, "10"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDiscNumber(tt.input)
			if got != tt.want {
				t.Errorf("FormatDiscNumber(%d) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseExtension(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"flac", ".flac"},
		{".flac", ".flac"},
		{"", ""},
		{"mp3", ".mp3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseExtension(tt.input)
			if got != tt.want {
				t.Errorf("ParseExtension(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetDirectoryAndFilename(t *testing.T) {
	tests := []struct {
		path         string
		wantDir      string
		wantFilename string
	}{
		{
			path:         "/home/user/Music/Artist/Album/song.flac",
			wantDir:      filepath.Dir("/home/user/Music/Artist/Album/song.flac"),
			wantFilename: "song",
		},
		{
			path:         "/home/user/Music/Artist/Album/song",
			wantDir:      filepath.Dir("/home/user/Music/Artist/Album/song"),
			wantFilename: "song",
		},
		{
			path:         "relative/path/to/song.mp3",
			wantDir:      filepath.Dir("relative/path/to/song.mp3"),
			wantFilename: "song",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotDir, gotFilename := GetDirectoryAndFilename(tt.path)
			if gotDir != tt.wantDir {
				t.Errorf("GetDirectoryAndFilename() dir = %v, want %v", gotDir, tt.wantDir)
			}
			if gotFilename != tt.wantFilename {
				t.Errorf("GetDirectoryAndFilename() filename = %v, want %v", gotFilename, tt.wantFilename)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if len(s)-start < len(substr) {
		return containsAt(s, substr, start+1)
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}
