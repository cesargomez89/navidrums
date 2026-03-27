package app

import (
	"encoding/json"

	"github.com/cesargomez89/navidrums/internal/store"
)

var DefaultMoods = []string{
	"Aggressive",
	"Atmospheric",
	"Chill",
	"Dark",
	"Energetic",
	"Melancholic",
	"Mystical",
	"Nostalgic",
	"Sophisticated",
	"Uplifting",
}

var DefaultLanguages = map[string]string{
	"ara": "Arabic",
	"deu": "German",
	"eng": "English",
	"spa": "Spanish",
	"fra": "French",
	"hin": "Hindi",
	"ita": "Italian",
	"jpn": "Japanese",
	"kor": "Korean",
	"por": "Portuguese",
	"zho": "Chinese",
}

func GetMoods(settingsRepo *store.SettingsRepo) []string {
	custom, err := settingsRepo.Get(store.SettingMoodList)
	if err != nil || custom == "" {
		return DefaultMoods
	}
	var list []string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultMoods
	}
	if len(list) == 0 {
		return DefaultMoods
	}
	return list
}

func GetLanguages(settingsRepo *store.SettingsRepo) map[string]string {
	custom, err := settingsRepo.Get(store.SettingLanguageList)
	if err != nil || custom == "" {
		return DefaultLanguages
	}
	var list map[string]string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultLanguages
	}
	if len(list) == 0 {
		return DefaultLanguages
	}
	return list
}
