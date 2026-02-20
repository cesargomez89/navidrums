package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cesargomez89/navidrums/internal/constants"
)

// Config holds all application configuration
type Config struct {
	Port           string
	DBPath         string
	DownloadsDir   string
	IncomingDir    string
	ProviderURL    string
	Quality        string
	LogLevel       string
	LogFormat      string
	Username       string
	Password       string
	SubdirTemplate string
	MusicBrainzURL string
	CacheTTL       time.Duration
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	home, _ := os.UserHomeDir()
	defaultDownload := filepath.Join(home, "Downloads/navidrums")
	defaultIncoming := filepath.Join(home, "Downloads/incoming")

	return &Config{
		Port:           getEnv("PORT", constants.DefaultPort),
		DBPath:         getEnv("DB_PATH", constants.DefaultDBPath),
		DownloadsDir:   getEnv("DOWNLOADS_DIR", defaultDownload),
		IncomingDir:    getEnv("INCOMING_DIR", defaultIncoming),
		ProviderURL:    getEnv("PROVIDER_URL", constants.DefaultProviderURL),
		Quality:        getEnv("QUALITY", constants.DefaultQuality),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		LogFormat:      getEnv("LOG_FORMAT", "text"),
		Username:       getEnv("NAVIDRUMS_USERNAME", constants.DefaultUsername),
		Password:       getEnv("NAVIDRUMS_PASSWORD", ""),
		SubdirTemplate: getEnv("SUBDIR_TEMPLATE", constants.DefaultSubdirTemplate),
		CacheTTL:       getEnvDuration("CACHE_TTL", constants.DefaultCacheTTL),
		MusicBrainzURL: getEnv("MUSICBRAINZ_URL", "https://musicbrainz.org/ws/2"),
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

	// Validate IncomingDir
	if c.IncomingDir == "" {
		errors = append(errors, "INCOMING_DIR cannot be empty")
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

	// Validate Username (optional - only required if password is set)
	if c.Password != "" && c.Username == "" {
		errors = append(errors, "USERNAME cannot be empty when PASSWORD is set")
	}

	// Password is optional - empty password disables basic auth

	// Validate SubdirTemplate
	if c.SubdirTemplate == "" {
		errors = append(errors, "SUBDIR_TEMPLATE cannot be empty")
	} else {
		if _, err := template.New("subdir").Parse(c.SubdirTemplate); err != nil {
			errors = append(errors, fmt.Sprintf("SUBDIR_TEMPLATE is invalid: %v", err))
		}
	}

	// Validate CacheTTL
	if c.CacheTTL <= 0 {
		errors = append(errors, "CACHE_TTL must be greater than 0")
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

// getEnvDuration retrieves an environment variable as time.Duration with a fallback default
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
