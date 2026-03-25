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

var DefaultStyles = []string{
	"Acoustic",
	"Cinematic",
	"Experimental",
	"Hardcore",
	"Lo-Fi",
	"Lyricist",
	"Minimalist",
	"Organic",
	"Polished",
	"Synthetic",
	"Urban",
	"Crossover",
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

func GetStyles(settingsRepo *store.SettingsRepo) []string {
	custom, err := settingsRepo.Get(store.SettingStyleList)
	if err != nil || custom == "" {
		return DefaultStyles
	}
	var list []string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultStyles
	}
	if len(list) == 0 {
		return DefaultStyles
	}
	return list
}
