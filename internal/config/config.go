package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port         string
	DBPath       string
	DownloadsDir string
	ProviderURL  string
	Quality      string
}

func Load() *Config {
	home, _ := os.UserHomeDir()
	defaultDownload := filepath.Join(home, "Downloads/navidrums")

	return &Config{
		Port:         getEnv("PORT", "8080"),
		DBPath:       getEnv("DB_PATH", "navidrums.db"),
		DownloadsDir: getEnv("DOWNLOADS_DIR", defaultDownload),
		ProviderURL:  getEnv("PROVIDER_URL", "http://127.0.0.1:8000"),
		Quality:      getEnv("QUALITY", "LOSSLESS"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
