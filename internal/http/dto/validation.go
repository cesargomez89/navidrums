package dto

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e *ValidationError) ToMap() map[string]string {
	return map[string]string{e.Field: e.Message}
}

func ToMap(errs []ValidationError) map[string]string {
	result := make(map[string]string)
	for _, e := range errs {
		result[e.Field] = e.Message
	}
	return result
}

func ToResponse(errs []ValidationError) string {
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

func validateISRC(isrc *string) []ValidationError {
	var errs []ValidationError
	if isrc != nil && *isrc != "" {
		isrcStr := strings.ToUpper(*isrc)
		isrcRegex := regexp.MustCompile(`^[A-Z]{2}[A-Z0-9]{3}[0-9]{7}$`)
		if !isrcRegex.MatchString(isrcStr) {
			errs = append(errs, ValidationError{Field: "isrc", Message: "invalid ISRC format (expected: CC-XXX-YY-NNNNN)"})
		}
	}
	return errs
}

func validateReleaseDate(releaseDate *string) []ValidationError {
	var errs []ValidationError
	if releaseDate != nil && *releaseDate != "" {
		dateRegex := regexp.MustCompile(`^\d{4}(-\d{2}(-\d{2})?)?$`)
		if !dateRegex.MatchString(*releaseDate) {
			errs = append(errs, ValidationError{Field: "release_date", Message: "invalid date format (expected: YYYY or YYYY-MM or YYYY-MM-DD)"})
		}
	}
	return errs
}

func validateURL(urlVal *string) []ValidationError {
	var errs []ValidationError
	if urlVal != nil && *urlVal != "" {
		_, err := url.ParseRequestURI(*urlVal)
		if err != nil {
			errs = append(errs, ValidationError{Field: "url", Message: "invalid URL format"})
		}
	}
	return errs
}

func validateKeyScale(keyScale *string) []ValidationError {
	var errs []ValidationError
	if keyScale != nil && *keyScale != "" {
		validScales := map[string]bool{"major": true, "minor": true}
		if !validScales[*keyScale] {
			errs = append(errs, ValidationError{Field: "key_scale", Message: "must be 'major' or 'minor'"})
		}
	}
	return errs
}

func validateYear(year *int) []ValidationError {
	var errs []ValidationError
	if year != nil {
		if *year < 1900 || *year > 2100 {
			errs = append(errs, ValidationError{Field: "year", Message: "must be between 1900 and 2100"})
		}
	}
	return errs
}

func validateBPM(bpm *int) []ValidationError {
	var errs []ValidationError
	if bpm != nil {
		if *bpm < 1 || *bpm > 999 {
			errs = append(errs, ValidationError{Field: "bpm", Message: "must be between 1 and 999"})
		}
	}
	return errs
}

func validateTrackNumber(trackNumber *int) []ValidationError {
	var errs []ValidationError
	if trackNumber != nil {
		if *trackNumber < 0 || *trackNumber > 9999 {
			errs = append(errs, ValidationError{Field: "track_number", Message: "must be between 0 and 9999"})
		}
	}
	return errs
}

func validateDiscNumber(discNumber *int) []ValidationError {
	var errs []ValidationError
	if discNumber != nil {
		if *discNumber < 0 || *discNumber > 99 {
			errs = append(errs, ValidationError{Field: "disc_number", Message: "must be between 0 and 99"})
		}
	}
	return errs
}

func validateTotalTracks(totalTracks *int) []ValidationError {
	var errs []ValidationError
	if totalTracks != nil {
		if *totalTracks < 0 || *totalTracks > 9999 {
			errs = append(errs, ValidationError{Field: "total_tracks", Message: "must be between 0 and 9999"})
		}
	}
	return errs
}

func validateTotalDiscs(totalDiscs *int) []ValidationError {
	var errs []ValidationError
	if totalDiscs != nil {
		if *totalDiscs < 0 || *totalDiscs > 99 {
			errs = append(errs, ValidationError{Field: "total_discs", Message: "must be between 0 and 99"})
		}
	}
	return errs
}

func validateReplayGain(replayGain *float64) []ValidationError {
	var errs []ValidationError
	if replayGain != nil {
		if *replayGain < -30 || *replayGain > 30 {
			errs = append(errs, ValidationError{Field: "replay_gain", Message: "must be between -30 and 30 dB"})
		}
	}
	return errs
}

func validatePeak(peak *float64) []ValidationError {
	var errs []ValidationError
	if peak != nil {
		if *peak < 0 || *peak > 2 {
			errs = append(errs, ValidationError{Field: "peak", Message: "must be between 0 and 2"})
		}
	}
	return errs
}
