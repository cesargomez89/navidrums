package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cesargomez89/navidrums/internal/constants"
)

func TestLoad(t *testing.T) {
	// Test default values
	cfg := Load()

	if cfg.Port != constants.DefaultPort {
		t.Errorf("Expected Port to be %s, got %s", constants.DefaultPort, cfg.Port)
	}

	if cfg.DBPath != constants.DefaultDBPath {
		t.Errorf("Expected DBPath to be %s, got %s", constants.DefaultDBPath, cfg.DBPath)
	}

	if cfg.ProviderURL != constants.DefaultProviderURL {
		t.Errorf("Expected ProviderURL to be %s, got %s", constants.DefaultProviderURL, cfg.ProviderURL)
	}

	if cfg.Quality != constants.DefaultQuality {
		t.Errorf("Expected Quality to be %s, got %s", constants.DefaultQuality, cfg.Quality)
	}

	// Check DownloadsDir is not empty (depends on user's home dir)
	if cfg.DownloadsDir == "" {
		t.Error("Expected DownloadsDir to not be empty")
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("PORT", "9090"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	if err := os.Setenv("DB_PATH", "/tmp/test.db"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	if err := os.Setenv("PROVIDER_URL", "http://example.com:8000"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	if err := os.Setenv("QUALITY", "HIGH"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("PORT"); err != nil {
			t.Logf("Unsetenv error: %v", err)
		}
		if err := os.Unsetenv("DB_PATH"); err != nil {
			t.Logf("Unsetenv error: %v", err)
		}
		if err := os.Unsetenv("PROVIDER_URL"); err != nil {
			t.Logf("Unsetenv error: %v", err)
		}
		if err := os.Unsetenv("QUALITY"); err != nil {
			t.Logf("Unsetenv error: %v", err)
		}
	}()

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Expected Port to be 9090, got %s", cfg.Port)
	}

	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("Expected DBPath to be /tmp/test.db, got %s", cfg.DBPath)
	}

	if cfg.ProviderURL != "http://example.com:8000" {
		t.Errorf("Expected ProviderURL to be http://example.com:8000, got %s", cfg.ProviderURL)
	}

	if cfg.Quality != "HIGH" {
		t.Errorf("Expected Quality to be HIGH, got %s", cfg.Quality)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Port:              "8080",
				DBPath:            "test.db",
				DownloadsDir:      "/tmp/downloads",
				ProviderURL:       "http://localhost:8000",
				Quality:           "LOSSLESS",
				LogLevel:          "info",
				LogFormat:         "text",
				Username:          "navidrums",
				Password:          "testpass",
				SubdirTemplate:    "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
				CacheTTL:          12 * time.Hour,
				RateLimitRequests: 60,
				RateLimitWindow:   time.Minute,
				RateLimitBurst:    10,
			},
			wantErr: false,
		},
		{
			name: "invalid port - not a number",
			config: Config{
				Port:           "abc",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "info",
				LogFormat:      "text",
				SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
			},
			wantErr: true,
		},
		{
			name: "invalid port - out of range",
			config: Config{
				Port:           "99999",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "info",
				LogFormat:      "text",
				SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
			},
			wantErr: true,
		},
		{
			name: "empty port",
			config: Config{
				Port:           "",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "info",
				LogFormat:      "text",
				SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
			},
			wantErr: true,
		},
		{
			name: "empty db path",
			config: Config{
				Port:           "8080",
				DBPath:         "",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "info",
				LogFormat:      "text",
				SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
			},
			wantErr: true,
		},
		{
			name: "invalid quality",
			config: Config{
				Port:           "8080",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "INVALID",
				LogLevel:       "info",
				LogFormat:      "text",
				SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Port:           "8080",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "invalid",
				LogFormat:      "text",
				SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: Config{
				Port:           "8080",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "info",
				LogFormat:      "xml",
				SubdirTemplate: "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
			},
			wantErr: true,
		},
		{
			name: "empty subdir template",
			config: Config{
				Port:           "8080",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "info",
				LogFormat:      "text",
				SubdirTemplate: "",
			},
			wantErr: true,
		},
		{
			name: "invalid subdir template syntax",
			config: Config{
				Port:           "8080",
				DBPath:         "test.db",
				DownloadsDir:   "/tmp/downloads",
				ProviderURL:    "http://localhost:8000",
				Quality:        "LOSSLESS",
				LogLevel:       "info",
				LogFormat:      "text",
				SubdirTemplate: "{{.InvalidField",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	if err := os.Setenv("TEST_VAR", "test_value"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_VAR"); err != nil {
			t.Logf("Unsetenv error: %v", err)
		}
	}()

	value := getEnv("TEST_VAR", "default")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}

	// Test with non-existing env var
	value = getEnv("NON_EXISTENT_VAR", "default")
	if value != "default" {
		t.Errorf("Expected 'default', got '%s'", value)
	}
}

func TestDownloadsDirDefault(t *testing.T) {
	// Ensure HOME is set
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME environment variable not set")
	}

	cfg := Load()
	expectedDir := filepath.Join(home, "Downloads/navidrums")
	if cfg.DownloadsDir != expectedDir {
		t.Errorf("Expected DownloadsDir to be %s, got %s", expectedDir, cfg.DownloadsDir)
	}
}

func TestValidateProviderURLs(t *testing.T) {
	baseConfig := Config{
		Port:              "8080",
		DBPath:            "test.db",
		DownloadsDir:      "/tmp/downloads",
		Quality:           "LOSSLESS",
		LogLevel:          "info",
		LogFormat:         "text",
		SubdirTemplate:    "{{.AlbumArtist}}/{{.Album}}/{{.Title}}",
		CacheTTL:          12 * time.Hour,
		RateLimitRequests: 60,
		RateLimitWindow:   time.Minute,
		RateLimitBurst:    10,
	}

	//nolint:govet // test struct, optimization not needed
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid legacy PROVIDER_URL",
			config: Config{
				ProviderURL: "http://localhost:8000",
			},
			wantErr: false,
		},
		{
			name: "empty legacy PROVIDER_URL",
			config: Config{
				ProviderURL: "",
			},
			wantErr: true,
			errMsg:  "PROVIDER_URL cannot be empty",
		},
		{
			name: "both new URLs valid",
			config: Config{
				ProviderMetadataURL: "http://localhost:8000",
				ProviderDownloadURL: "http://localhost:8001",
			},
			wantErr: false,
		},
		{
			name: "metadata URL invalid",
			config: Config{
				ProviderMetadataURL: "not-a-valid-url",
				ProviderDownloadURL: "http://localhost:8001",
			},
			wantErr: true,
			errMsg:  "PROVIDER_METADATA_URL is not a valid absolute URL",
		},
		{
			name: "download URL invalid",
			config: Config{
				ProviderMetadataURL: "http://localhost:8000",
				ProviderDownloadURL: "not-a-valid-url",
			},
			wantErr: true,
			errMsg:  "PROVIDER_DOWNLOAD_URL is not a valid absolute URL",
		},
		{
			name: "only metadata URL set",
			config: Config{
				ProviderMetadataURL: "http://localhost:8000",
			},
			wantErr: true,
			errMsg:  "PROVIDER_DOWNLOAD_URL is required when PROVIDER_METADATA_URL is set",
		},
		{
			name: "only download URL set",
			config: Config{
				ProviderDownloadURL: "http://localhost:8001",
			},
			wantErr: true,
			errMsg:  "PROVIDER_METADATA_URL is required when PROVIDER_DOWNLOAD_URL is set",
		},
		{
			name: "legacy URL with new URLs (new takes precedence)",
			config: Config{
				ProviderURL:         "http://localhost:9999",
				ProviderMetadataURL: "http://localhost:8000",
				ProviderDownloadURL: "http://localhost:8001",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig
			cfg.ProviderURL = tt.config.ProviderURL
			cfg.ProviderMetadataURL = tt.config.ProviderMetadataURL
			cfg.ProviderDownloadURL = tt.config.ProviderDownloadURL

			err := cfg.Validate(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestLoadProviderURLs(t *testing.T) {
	originalProviderURL := os.Getenv("PROVIDER_URL")
	originalMetadataURL := os.Getenv("PROVIDER_METADATA_URL")
	originalDownloadURL := os.Getenv("PROVIDER_DOWNLOAD_URL")
	defer func() {
		if originalProviderURL != "" {
			_ = os.Setenv("PROVIDER_URL", originalProviderURL)
		} else {
			_ = os.Unsetenv("PROVIDER_URL")
		}
		if originalMetadataURL != "" {
			_ = os.Setenv("PROVIDER_METADATA_URL", originalMetadataURL)
		} else {
			_ = os.Unsetenv("PROVIDER_METADATA_URL")
		}
		if originalDownloadURL != "" {
			_ = os.Setenv("PROVIDER_DOWNLOAD_URL", originalDownloadURL)
		} else {
			_ = os.Unsetenv("PROVIDER_DOWNLOAD_URL")
		}
	}()

	_ = os.Unsetenv("PROVIDER_URL")
	_ = os.Unsetenv("PROVIDER_METADATA_URL")
	_ = os.Unsetenv("PROVIDER_DOWNLOAD_URL")

	_ = os.Setenv("PROVIDER_METADATA_URL", "http://metadata.example.com")
	_ = os.Setenv("PROVIDER_DOWNLOAD_URL", "http://download.example.com")

	cfg := Load()

	if cfg.ProviderMetadataURL != "http://metadata.example.com" {
		t.Errorf("Expected ProviderMetadataURL to be 'http://metadata.example.com', got '%s'", cfg.ProviderMetadataURL)
	}
	if cfg.ProviderDownloadURL != "http://download.example.com" {
		t.Errorf("Expected ProviderDownloadURL to be 'http://download.example.com', got '%s'", cfg.ProviderDownloadURL)
	}
}
