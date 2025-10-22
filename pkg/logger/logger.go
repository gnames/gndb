package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gnames/gndb/pkg/config"
	"github.com/lmittmann/tint"
)

// New creates a new slog.Logger based on the provided configuration.
// It respects the logging level and format from the config.
// Invalid values default to Info level and Text format.
func New(cfg *config.LoggingConfig) *slog.Logger {
	// Parse log level
	level := ParseLevel(cfg.Level)

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	case "tint", "": // Default to tint if empty or invalid
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: time.TimeOnly, // "3:04PM" - compact time format
		})
	default:
		// Invalid format, default to tint
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: time.TimeOnly,
		})
	}

	return slog.New(handler)
}

// ParseLevel converts a string log level to slog.Level.
// Valid levels: "debug", "info", "warn", "error" (case-insensitive).
// Invalid levels default to Info.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info", "": // Default to info if empty
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		// Invalid level, default to info
		return slog.LevelInfo
	}
}
