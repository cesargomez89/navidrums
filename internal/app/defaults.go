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

var DefaultLanguages = map[string]string{
	"ar":     "Arabic",
	"zh":     "Chinese",
	"en":     "English",
	"fr":     "French",
	"de":     "German",
	"hi":     "Hindi",
	"it":     "Italian",
	"ja":     "Japanese",
	"ko":     "Korean",
	"pt":     "Portuguese",
	"es":     "Spanish",
	"es-419": "Spanish (Latin America)",
}

var DefaultCountries = map[string]string{
	"ar": "Argentina",
	"br": "Brazil",
	"ca": "Canada",
	"cl": "Chile",
	"cn": "China",
	"co": "Colombia",
	"cu": "Cuba",
	"fr": "France",
	"de": "Germany",
	"in": "India",
	"it": "Italy",
	"jp": "Japan",
	"mx": "Mexico",
	"pr": "Puerto Rico",
	"kr": "South Korea",
	"es": "Spain",
	"gb": "United Kingdom",
	"us": "United States",
	"ve": "Venezuela",
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

func GetCountries(settingsRepo *store.SettingsRepo) map[string]string {
	custom, err := settingsRepo.Get(store.SettingCountryList)
	if err != nil || custom == "" {
		return DefaultCountries
	}
	var list map[string]string
	if err := json.Unmarshal([]byte(custom), &list); err != nil {
		return DefaultCountries
	}
	if len(list) == 0 {
		return DefaultCountries
	}
	return list
}
