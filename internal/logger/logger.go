// Package logger provides structured logging functionality
package logger

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger for application-wide logging
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level  string // debug, info, warn, error
	Format string // text, json
}

// New creates a new structured logger
func New(cfg Config) *Logger {
	// Parse log level
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Create handler based on format
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithComponent returns a logger with a component attribute
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.With("component", component),
	}
}

// WithJob returns a logger with job context attributes
func (l *Logger) WithJob(jobID, jobType string) *Logger {
	return &Logger{
		Logger: l.With("job_id", jobID, "job_type", jobType),
	}
}

// WithTrack returns a logger with track context attributes
func (l *Logger) WithTrack(trackID, trackTitle string) *Logger {
	return &Logger{
		Logger: l.With("track_id", trackID, "track_title", trackTitle),
	}
}

// Default returns a default logger for quick usage
func Default() *Logger {
	return New(Config{
		Level:  "info",
		Format: "text",
	})
}
