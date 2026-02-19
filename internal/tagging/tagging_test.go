package tagging

import (
	"fmt"
	"testing"

	"github.com/cesargomez89/navidrums/internal/domain"
)

func TestFormatToLRC(t *testing.T) {
	input := "[00:10.00] Line 1\n[00:20.00] Line 2\n  \n [00:30.00] Line 3 "
	expected := "[00:10.00] Line 1\n[00:20.00] Line 2\n[00:30.00] Line 3\n"
	result := formatToLRC(input)
	if result != expected {
		t.Errorf("formatToLRC mismatch.\nGot: %q\nWant: %q", result, expected)
	}
}

func TestNewVorbisComment(t *testing.T) {
	track := &domain.Track{
		Title:       "Test Title",
		Artist:      "Solo Artist",
		Album:       "Test Album",
		Year:        2023,
		TrackNumber: 5,
		Genre:       "Rock",
		BPM:         120,
		Compilation: true,
		ArtistIDs:   []string{"id1", "id2"},
	}

	vc := newVorbisComment(track)

	check := func(name, expected string) {
		t.Helper()
		found := false
		target := fmt.Sprintf("%s=%s", name, expected)
		for _, entry := range vc.Comments {
			if entry == target {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Field %s not found in VorbisComment", target)
		}
	}

	check("TITLE", "Test Title")
	check("ARTIST", "Solo Artist")
	check("ALBUM", "Test Album")
	check("DATE", "2023")
	check("TRACKNUMBER", "5")
	check("GENRE", "Rock")
	check("BPM", "120")
	check("COMPILATION", "1")
	check("MUSICBRAINZ_ARTISTID", "id1")
	check("MUSICBRAINZ_ARTISTID", "id2")
}

func TestNewVorbisComment_MultiArtist(t *testing.T) {
	track := &domain.Track{
		Artists:      []string{"Artist A", "Artist B"},
		AlbumArtists: []string{"Album Artist 1"},
	}

	vc := newVorbisComment(track)

	artists := 0
	albumArtists := 0
	for _, entry := range vc.Comments {
		if entry == "ARTIST=Artist A" || entry == "ARTIST=Artist B" {
			artists++
		}
		if entry == "ALBUMARTIST=Album Artist 1" {
			albumArtists++
		}
	}

	if artists != 2 {
		t.Errorf("Expected 2 ARTIST fields, got %d", artists)
	}
	if albumArtists != 1 {
		t.Errorf("Expected 1 ALBUMARTIST field, got %d", albumArtists)
	}
}
