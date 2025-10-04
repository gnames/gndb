package logger

import (
	"log/slog"
	"os"
	"strings"

	"github.com/gnames/gndb/pkg/config"
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
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text", "": // Default to text if empty or invalid
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		// Invalid format, default to text
		handler = slog.NewTextHandler(os.Stdout, opts)
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
