package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// PathTemplateData holds the data for path template execution
type PathTemplateData struct {
	AlbumArtist  string
	Album        string
	Disc         string
	Track        string
	Title        string
	OriginalYear int
}

// BuildPath executes the template and returns the full path (without extension)
func BuildPath(templateStr string, data *PathTemplateData) (string, error) {
	tmpl, err := template.New("subdir").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// BuildPathTemplateData creates PathTemplateData from track metadata
func BuildPathTemplateData(albumArtist string, year int, album string, discNum, trackNum int, title string) *PathTemplateData {
	// Sanitize all string values
	sanitizedAlbumArtist := Sanitize(albumArtist)
	sanitizedAlbum := Sanitize(album)
	sanitizedTitle := Sanitize(title)

	// Format disc and track numbers with zero-padding
	discStr := fmt.Sprintf("%02d", discNum)
	trackStr := fmt.Sprintf("%02d", trackNum)

	return &PathTemplateData{
		AlbumArtist:  sanitizedAlbumArtist,
		OriginalYear: year,
		Album:        sanitizedAlbum,
		Disc:         discStr,
		Track:        trackStr,
		Title:        sanitizedTitle,
	}
}

// BuildFullPath constructs the complete file path with extension
// If hidden is true, prepends a dot to the filename to make it hidden
func BuildFullPath(downloadsDir, templateStr string, data *PathTemplateData, ext string, hidden bool) (string, error) {
	relPath, err := BuildPath(templateStr, data)
	if err != nil {
		return "", err
	}

	// Ensure extension starts with a dot
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Prepend dot for hidden files
	if hidden {
		relPath = "." + relPath
	}

	// Join downloads dir with relative path and extension
	fullPath := filepath.Join(downloadsDir, relPath+ext)

	// Clean the path to remove any ".." or redundant separators
	fullPath = filepath.Clean(fullPath)

	return fullPath, nil
}

// GetDirectoryAndFilename splits a full path into directory and filename (without ext)
func GetDirectoryAndFilename(fullPath string) (dir, filename string) {
	dir = filepath.Dir(fullPath)
	filename = filepath.Base(fullPath)
	// Remove extension if present
	if ext := filepath.Ext(filename); ext != "" {
		filename = strings.TrimSuffix(filename, ext)
	}
	return dir, filename
}

// ParseExtension parses an extension string, ensuring it starts with a dot
func ParseExtension(ext string) string {
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		return "." + ext
	}
	return ext
}

// FormatTrackNumber formats a track number with zero-padding
func FormatTrackNumber(n int) string {
	return fmt.Sprintf("%02d", n)
}

// FormatDiscNumber formats a disc number with zero-padding
func FormatDiscNumber(n int) string {
	return fmt.Sprintf("%02d", n)
}

// SafeAtoi converts string to int, returns 0 on error
func SafeAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// ToggleHiddenFile renames a file to add or remove the leading dot
func ToggleHiddenFile(path string, hidden bool) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	if hidden {
		if !strings.HasPrefix(base, ".") {
			base = "." + base
		}
	} else {
		base = strings.TrimPrefix(base, ".")
	}

	newPath := filepath.Join(dir, base)
	if newPath == path {
		return path, nil
	}

	if err := os.Rename(path, newPath); err != nil {
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	return newPath, nil
}

// IsHiddenFile checks if a file has a leading dot
func IsHiddenFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".")
}
