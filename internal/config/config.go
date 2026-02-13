package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cesargomez89/navidrums/internal/constants"
)

// Config holds all application configuration
type Config struct {
	Port         string
	DBPath       string
	DownloadsDir string
	ProviderURL  string
	Quality      string
	LogLevel     string
	LogFormat    string
	Username     string
	Password     string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	home, _ := os.UserHomeDir()
	defaultDownload := filepath.Join(home, "Downloads/navidrums")

	return &Config{
		Port:         getEnv("PORT", constants.DefaultPort),
		DBPath:       getEnv("DB_PATH", constants.DefaultDBPath),
		DownloadsDir: getEnv("DOWNLOADS_DIR", defaultDownload),
		ProviderURL:  getEnv("PROVIDER_URL", constants.DefaultProviderURL),
		Quality:      getEnv("QUALITY", constants.DefaultQuality),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		LogFormat:    getEnv("LOG_FORMAT", "text"),
		Username:     getEnv("NAVIDRUMS_USERNAME", constants.DefaultUsername),
		Password:     getEnv("NAVIDRUMS_PASSWORD", ""),
	}
}

// Validate validates the configuration and returns detailed errors
func (c *Config) Validate() error {
	var errors []string

	// Validate Port
	if c.Port == "" {
		errors = append(errors, "PORT cannot be empty")
	} else {
		port, err := strconv.Atoi(c.Port)
		if err != nil {
			errors = append(errors, fmt.Sprintf("PORT must be a valid number, got: %s", c.Port))
		} else if port < 1 || port > 65535 {
			errors = append(errors, fmt.Sprintf("PORT must be between 1 and 65535, got: %d", port))
		}
	}

	// Validate DBPath
	if c.DBPath == "" {
		errors = append(errors, "DB_PATH cannot be empty")
	}

	// Validate DownloadsDir
	if c.DownloadsDir == "" {
		errors = append(errors, "DOWNLOADS_DIR cannot be empty")
	}

	// Validate ProviderURL
	if c.ProviderURL == "" {
		errors = append(errors, "PROVIDER_URL cannot be empty")
	} else {
		if _, err := url.Parse(c.ProviderURL); err != nil {
			errors = append(errors, fmt.Sprintf("PROVIDER_URL is not a valid URL: %s", c.ProviderURL))
		}
	}

	// Validate Quality
	validQualities := map[string]bool{
		constants.QualityLossless:      true,
		constants.QualityHiResLossless: true,
		constants.QualityHigh:          true,
		constants.QualityLow:           true,
	}
	if !validQualities[c.Quality] {
		errors = append(errors, fmt.Sprintf("QUALITY must be one of: LOSSLESS, HI_RES_LOSSLESS, HIGH, LOW, got: %s", c.Quality))
	}

	// Validate LogLevel
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		errors = append(errors, fmt.Sprintf("LOG_LEVEL must be one of: debug, info, warn, error, got: %s", c.LogLevel))
	}

	// Validate LogFormat
	validLogFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validLogFormats[c.LogFormat] {
		errors = append(errors, fmt.Sprintf("LOG_FORMAT must be one of: text, json, got: %s", c.LogFormat))
	}

	// Validate Username
	if c.Username == "" {
		errors = append(errors, "USERNAME cannot be empty")
	}

	// Validate Password
	if c.Password == "" {
		errors = append(errors, "PASSWORD cannot be empty")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// getEnv retrieves an environment variable with a fallback default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
