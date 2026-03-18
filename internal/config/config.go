package config

import (
	"fmt"
	"log/slog"
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
	Port                string
	DBPath              string
	DownloadsDir        string
	ProviderURL         string
	ProviderMetadataURL string
	ProviderDownloadURL string
	Quality             string
	PlayQuality         string
	LogLevel            string
	LogFormat           string
	Username            string
	Password            string
	SubdirTemplate      string
	MusicBrainzURL      string
	FFmpegPath          string
	FFprobePath         string
	Theme               string
	CacheTTL            time.Duration
	MusicBrainzCacheTTL time.Duration
	RateLimitWindow     time.Duration
	RateLimitRequests   int
	RateLimitBurst      int
	SkipAuth            bool
	DisableRateLimit    bool
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	home, _ := os.UserHomeDir()
	defaultDownload := filepath.Join(home, "Downloads/navidrums")

	return &Config{
		Port:                getEnv("PORT", constants.DefaultPort),
		DBPath:              getEnv("DB_PATH", constants.DefaultDBPath),
		DownloadsDir:        getEnv("DOWNLOADS_DIR", defaultDownload),
		ProviderURL:         getEnv("PROVIDER_URL", constants.DefaultProviderURL),
		ProviderMetadataURL: getEnv("PROVIDER_METADATA_URL", ""),
		ProviderDownloadURL: getEnv("PROVIDER_DOWNLOAD_URL", ""),
		Quality:             getEnv("QUALITY", constants.DefaultQuality),
		PlayQuality:         getEnv("PLAY_QUALITY", "HIGH"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		LogFormat:           getEnv("LOG_FORMAT", "text"),
		Username:            getEnv("NAVIDRUMS_USERNAME", constants.DefaultUsername),
		Password:            getEnv("NAVIDRUMS_PASSWORD", ""),
		SubdirTemplate:      getEnv("SUBDIR_TEMPLATE", constants.DefaultSubdirTemplate),
		CacheTTL:            getEnvDuration("CACHE_TTL", constants.DefaultCacheTTL),
		MusicBrainzCacheTTL: getEnvDuration("MUSICBRAINZ_CACHE_TTL", constants.DefaultMusicBrainzCacheTTL),
		MusicBrainzURL:      getEnv("MUSICBRAINZ_URL", "https://musicbrainz.org/ws/2"),
		RateLimitRequests:   getEnvInt("RATE_LIMIT_REQUESTS", 200),
		RateLimitWindow:     getEnvDuration("RATE_LIMIT_WINDOW", time.Minute),
		RateLimitBurst:      getEnvInt("RATE_LIMIT_BURST", 10),
		SkipAuth:            getEnvBool("SKIP_AUTH", false),
		DisableRateLimit:    getEnvBool("DISABLE_RATE_LIMIT", false),
		Theme:               getEnv("THEME", "golden"),
		FFmpegPath:          getEnv("FFMPEG_PATH", ""),
		FFprobePath:         getEnv("FFPROBE_PATH", ""),
	}
}

// Validate validates the configuration and returns detailed errors.
// If a logger is provided, deprecation warnings will be logged for legacy configuration.
func (c *Config) Validate(log *slog.Logger) error {
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

	// Validate Provider URLs based on configuration
	providerErrors, hasNewConfig, hasLegacyConfig := c.validateProviderURLs(log)
	errors = append(errors, providerErrors...)

	// Emit deprecation warning if using legacy PROVIDER_URL
	if hasLegacyConfig && log != nil {
		if hasNewConfig {
			log.Warn("PROVIDER_URL is deprecated and ignored when PROVIDER_METADATA_URL or PROVIDER_DOWNLOAD_URL are set")
		} else {
			log.Warn("PROVIDER_URL is deprecated, use PROVIDER_METADATA_URL and PROVIDER_DOWNLOAD_URL instead")
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
		errors = append(errors, fmt.Sprintf("QUALITY must be one of: %s, %s, %s, %s, got: %s",
			constants.QualityLossless, constants.QualityHiResLossless, constants.QualityHigh, constants.QualityLow, c.Quality))
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

	// Validate MusicBrainzCacheTTL
	if c.MusicBrainzCacheTTL <= 0 {
		errors = append(errors, "MUSICBRAINZ_CACHE_TTL must be greater than 0")
	}

	// Validate RateLimitRequests
	if c.RateLimitRequests <= 0 {
		errors = append(errors, "RATE_LIMIT_REQUESTS must be greater than 0")
	}

	// Validate RateLimitWindow
	if c.RateLimitWindow <= 0 {
		errors = append(errors, "RATE_LIMIT_WINDOW must be greater than 0")
	}

	// Validate RateLimitBurst
	if c.RateLimitBurst <= 0 {
		errors = append(errors, "RATE_LIMIT_BURST must be greater than 0")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateProviderURLs validates the provider URL configuration.
// Returns: (errors, hasNewConfig, hasLegacyConfig)
func (c *Config) validateProviderURLs(log *slog.Logger) ([]string, bool, bool) {
	var errors []string
	hasMetadataURL := c.ProviderMetadataURL != ""
	hasDownloadURL := c.ProviderDownloadURL != ""
	hasLegacyURL := c.ProviderURL != ""

	hasNewConfig := hasMetadataURL || hasDownloadURL
	hasLegacyConfig := hasLegacyURL && !hasNewConfig

	if hasNewConfig {
		// Both new vars set: validate both
		if hasMetadataURL {
			if u, err := url.Parse(c.ProviderMetadataURL); err != nil || u.Scheme == "" || u.Host == "" {
				errors = append(errors, fmt.Sprintf("PROVIDER_METADATA_URL is not a valid absolute URL: %s", c.ProviderMetadataURL))
			}
		} else {
			errors = append(errors, "PROVIDER_METADATA_URL is required when PROVIDER_DOWNLOAD_URL is set")
		}

		if hasDownloadURL {
			if u, err := url.Parse(c.ProviderDownloadURL); err != nil || u.Scheme == "" || u.Host == "" {
				errors = append(errors, fmt.Sprintf("PROVIDER_DOWNLOAD_URL is not a valid absolute URL: %s", c.ProviderDownloadURL))
			}
		} else {
			errors = append(errors, "PROVIDER_DOWNLOAD_URL is required when PROVIDER_METADATA_URL is set")
		}
	} else {
		// Only legacy PROVIDER_URL set: validate it, use it for both (backward compat)
		if c.ProviderURL == "" {
			errors = append(errors, "PROVIDER_URL cannot be empty")
		} else {
			if _, err := url.Parse(c.ProviderURL); err != nil {
				errors = append(errors, fmt.Sprintf("PROVIDER_URL is not a valid URL: %s", c.ProviderURL))
			}
		}
	}

	return errors, hasNewConfig, hasLegacyConfig
}

// getEnv retrieves an environment variable with a fallback default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// getEnvInt retrieves an environment variable as int with a fallback default
func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return fallback
}

// getEnvBool retrieves an environment variable as bool with a fallback default
func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return strings.ToLower(value) == "true" || value == "1"
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
