package logger

import (
	"testing"
)

func TestNew(t *testing.T) {
	// Test with text format
	cfg := Config{
		Level:  "info",
		Format: "text",
	}
	logger := New(cfg)
	if logger == nil {
		t.Error("Expected logger to not be nil")
	}

	// Test with json format
	cfg.Format = "json"
	logger = New(cfg)
	if logger == nil {
		t.Error("Expected logger to not be nil")
	}

	// Test with debug level
	cfg.Level = "debug"
	logger = New(cfg)
	if logger == nil {
		t.Error("Expected logger to not be nil")
	}

	// Test with invalid level (should default to info)
	cfg.Level = "invalid"
	logger = New(cfg)
	if logger == nil {
		t.Error("Expected logger to not be nil")
	}
}

func TestWithComponent(t *testing.T) {
	logger := Default()
	componentLogger := logger.WithComponent("test-component")

	if componentLogger == nil {
		t.Error("Expected component logger to not be nil")
	}

	// Test chaining
	componentLogger2 := componentLogger.WithComponent("nested-component")
	if componentLogger2 == nil {
		t.Error("Expected nested component logger to not be nil")
	}
}

func TestWithJob(t *testing.T) {
	logger := Default()
	jobLogger := logger.WithJob("job-123", "track")

	if jobLogger == nil {
		t.Error("Expected job logger to not be nil")
	}
}

func TestWithTrack(t *testing.T) {
	logger := Default()
	trackLogger := logger.WithTrack("track-456", "Test Song")

	if trackLogger == nil {
		t.Error("Expected track logger to not be nil")
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Error("Expected default logger to not be nil")
	}
}

func TestLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		cfg := Config{
			Level:  level,
			Format: "text",
		}
		logger := New(cfg)
		if logger == nil {
			t.Errorf("Expected logger to not be nil for level %s", level)
		}
	}
}

func TestLogFormats(t *testing.T) {
	formats := []string{"text", "json"}

	for _, format := range formats {
		cfg := Config{
			Level:  "info",
			Format: format,
		}
		logger := New(cfg)
		if logger == nil {
			t.Errorf("Expected logger to not be nil for format %s", format)
		}
	}
}
