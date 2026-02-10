# Metadata Tagging Implementation Summary

## Overview
Implemented comprehensive metadata tagging for downloaded audio tracks with support for FLAC, MP3, and MP4 formats. The system now automatically embeds rich metadata into audio files and saves album/playlist artwork.

## Changes Made

### 1. Enhanced Data Models (`internal/models/models.go`)
Added extensive metadata fields to Track, Album, and Playlist models:

**Track Model:**
- `AlbumArtist` - Album artist name
- `DiscNumber` - Disc number for multi-disc albums
- `TotalTracks` - Total tracks in album
- `TotalDiscs` - Total discs in album
- `Year` - Release year
- `Genre` - Music genre
- `Label` - Record label
- `ISRC` - International Standard Recording Code
- `Copyright` - Copyright information
- `Composer` - Composer name
- `AlbumArtURL` - URL to album artwork
- `ExplicitLyrics` - Explicit content flag

**Album Model:**
- `Year`, `Genre`, `Label`, `Copyright`
- `TotalTracks`, `TotalDiscs`
- `AlbumArtURL`

**Playlist Model:**
- `Description` - Playlist description
- `ImageURL` - Playlist cover image URL

### 2. Created Tagging Package (`internal/tagging/tagging.go`)
New package providing audio file tagging functionality:

**Functions:**
- `TagFile()` - Main function to tag audio files based on format
- `tagFLAC()` - FLAC metadata tagging using vorbis comments
- `tagMP3()` - MP3 ID3v2.4 tagging
- `tagMP4()` - MP4/M4A tagging (placeholder for future implementation)
- `DownloadImage()` - Downloads album/playlist artwork from URLs
- `SaveImageToFile()` - Saves image data to disk

**Supported Tags:**
- Title, Artist, Album Artist, Album
- Track Number, Disc Number, Total Tracks, Total Discs
- Year, Genre, Label, ISRC, Copyright, Composer
- Embedded album artwork (JPEG)

**Dependencies Added:**
- `github.com/bogem/id3v2/v2` - MP3 ID3v2 tagging
- `github.com/go-flac/go-flac` - FLAC file parsing
- `github.com/go-flac/flacvorbis` - FLAC vorbis comments
- `github.com/go-flac/flacpicture` - FLAC picture embedding

### 3. Enhanced HiFi Provider (`internal/providers/hifi.go`)
Updated API response parsing to fetch comprehensive metadata:

**GetAlbum():**
- Fetches release date, copyright, track/disc counts
- Retrieves album cover URL
- Parses volume numbers, ISRC codes, explicit flags

**GetTrack():**
- Fetches complete track metadata including album info
- Retrieves album artwork URL
- Parses release date for year extraction

**GetPlaylist():**
- Fetches playlist description and cover image
- Retrieves track-level album artwork URLs
- Parses ISRC and explicit flags

### 4. Enhanced Worker (`internal/worker/worker.go`)
Updated download workflow to include tagging and image saving:

**New Helper Methods:**
- `downloadImage()` - Downloads images from URLs
- `tagFile()` - Tags audio files with metadata
- `saveAlbumArt()` - Saves album artwork to album folder
- `savePlaylistImage()` - Saves playlist cover to playlists folder

**Updated runJob() Flow:**
1. Fetch metadata from provider (includes artwork URLs)
2. For albums: Save album art to `{Artist} - {Album}/cover.jpg`
3. For playlists: Save playlist image to `playlists/{Playlist Title}.jpg`
4. Download audio stream
5. Download album artwork for embedding
6. Tag audio file with all available metadata
7. Save album art to folder (if not already saved)
8. Record download completion

### 5. Updated Documentation (`README.md`)
Added comprehensive feature documentation highlighting:
- Metadata tagging capabilities
- Supported tag types
- Album/playlist artwork handling
- Supported audio formats

## File Organization

### Album Downloads
```
{Downloads Dir}/
  {Artist} - {Album}/
    cover.jpg                    # Album artwork
    01 - Track Title.flac        # Tagged audio file
    02 - Another Track.flac
    ...
```

### Playlist Downloads
```
{Downloads Dir}/
  playlists/
    {Playlist Title}.jpg         # Playlist cover image
    {Playlist Title}.m3u         # Playlist file
```

## Metadata Embedded in Files

### FLAC Files (Vorbis Comments)
- TITLE, ARTIST, ALBUMARTIST, ALBUM
- TRACKNUMBER, TRACKTOTAL, DISCNUMBER, DISCTOTAL
- DATE (year), GENRE, LABEL, ISRC
- COPYRIGHT, COMPOSER
- Embedded JPEG album artwork (PICTURE block)

### MP3 Files (ID3v2.4)
- Title, Artist, Album, Year, Genre
- Album Artist (TPE2 frame)
- Track Number/Total (TRCK frame)
- Disc Number/Total (TPOS frame)
- Composer, Publisher (Label), ISRC, Copyright
- Embedded JPEG album artwork (APIC frame)

### MP4 Files
- Placeholder for future implementation
- Requires atom-level manipulation

## Benefits

1. **Complete Metadata**: All downloaded files have comprehensive tags
2. **Album Artwork**: Embedded in files and saved separately for compatibility
3. **Playlist Images**: Visual representation of playlists
4. **Navidrome Ready**: Properly tagged files work seamlessly with Navidrome
5. **Future-Proof**: Extensible design allows easy addition of new metadata fields

## Testing Recommendations

1. Download a single track and verify tags with a metadata viewer
2. Download an album and check:
   - All tracks are properly tagged
   - Album art is embedded and saved as `cover.jpg`
   - Disc numbers are correct for multi-disc albums
3. Download a playlist and verify:
   - Playlist image is saved
   - M3U file is generated correctly
4. Test with different audio qualities (LOSSLESS, HI_RES_LOSSLESS)
5. Verify compatibility with Navidrome library scanning

## Known Limitations

1. MP4/M4A tagging not yet implemented (requires additional library)
2. Genre information may not always be available from the API
3. Composer information availability depends on API data
4. Image downloads may fail if URLs are invalid or inaccessible

## Future Enhancements

1. Implement MP4/M4A tagging using `github.com/abema/go-mp4` or similar
2. Add support for lyrics embedding
3. Add support for additional artwork types (back cover, artist photo)
4. Implement tag validation and correction
5. Add option to re-tag existing files
