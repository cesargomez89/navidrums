package config

import (
	"os"
	"path/filepath"
	"testing"

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
	os.Setenv("PORT", "9090")
	os.Setenv("DB_PATH", "/tmp/test.db")
	os.Setenv("PROVIDER_URL", "http://example.com:8000")
	os.Setenv("QUALITY", "HIGH")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_PATH")
		os.Unsetenv("PROVIDER_URL")
		os.Unsetenv("QUALITY")
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
				Port:         "8080",
				DBPath:       "test.db",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "LOSSLESS",
				LogLevel:     "info",
				LogFormat:    "text",
				Username:     "navidrums",
				Password:     "testpass",
			},
			wantErr: false,
		},
		{
			name: "invalid port - not a number",
			config: Config{
				Port:         "abc",
				DBPath:       "test.db",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "LOSSLESS",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid port - out of range",
			config: Config{
				Port:         "99999",
				DBPath:       "test.db",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "LOSSLESS",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "empty port",
			config: Config{
				Port:         "",
				DBPath:       "test.db",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "LOSSLESS",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "empty db path",
			config: Config{
				Port:         "8080",
				DBPath:       "",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "LOSSLESS",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid quality",
			config: Config{
				Port:         "8080",
				DBPath:       "test.db",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "INVALID",
				LogLevel:     "info",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Port:         "8080",
				DBPath:       "test.db",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "LOSSLESS",
				LogLevel:     "invalid",
				LogFormat:    "text",
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			config: Config{
				Port:         "8080",
				DBPath:       "test.db",
				DownloadsDir: "/tmp/downloads",
				ProviderURL:  "http://localhost:8000",
				Quality:      "LOSSLESS",
				LogLevel:     "info",
				LogFormat:    "xml",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

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
