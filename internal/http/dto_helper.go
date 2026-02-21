package httpapp

import (
	"reflect"
)

func DTOToUpdates(dtoVal interface{}) map[string]interface{} {
	updates := make(map[string]interface{})
	dtoValReflect := reflect.ValueOf(dtoVal)
	if dtoValReflect.Kind() == reflect.Ptr {
		dtoValReflect = dtoValReflect.Elem()
	}
	dtoType := dtoValReflect.Type()

	for i := 0; i < dtoType.NumField(); i++ {
		field := dtoType.Field(i)
		fieldVal := dtoValReflect.Field(i)

		if fieldVal.Kind() != reflect.Ptr || fieldVal.IsNil() {
			continue
		}

		colName := field.Tag.Get("form")
		if colName == "" {
			continue
		}

		dbCol := formToDBColumn(colName)

		if !fieldVal.Elem().IsValid() {
			continue
		}

		actualVal := fieldVal.Elem().Interface()

		if strVal, ok := actualVal.(string); ok && strVal == "" {
			continue
		}

		updates[dbCol] = actualVal
	}

	return updates
}

func formToDBColumn(formName string) string {
	mapping := map[string]string{
		"title":          "title",
		"artist":         "artist",
		"album":          "album",
		"album_artist":   "album_artist",
		"genre":          "genre",
		"label":          "label",
		"composer":       "composer",
		"copyright":      "copyright",
		"isrc":           "isrc",
		"version":        "version",
		"description":    "description",
		"url":            "url",
		"audio_quality":  "audio_quality",
		"audio_modes":    "audio_modes",
		"lyrics":         "lyrics",
		"subtitles":      "subtitles",
		"barcode":        "barcode",
		"catalog_number": "catalog_number",
		"release_type":   "release_type",
		"release_date":   "release_date",
		"key":            "key_name",
		"key_scale":      "key_scale",
		"track_number":   "track_number",
		"disc_number":    "disc_number",
		"total_tracks":   "total_tracks",
		"total_discs":    "total_discs",
		"year":           "year",
		"bpm":            "bpm",
		"replay_gain":    "replay_gain",
		"peak":           "peak",
		"compilation":    "compilation",
		"explicit":       "explicit",
	}

	if col, ok := mapping[formName]; ok {
		return col
	}
	return formName
}
