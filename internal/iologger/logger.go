// Package iologger provides slog-based logging initialization and configuration.
package iologger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gnames/gndb/pkg/config"
)

// Init initializes the global slog logger with the given configuration.
// Creates log file in logDir if destination is "file".
// Overwrites existing log file on each run (starts fresh).
func Init(logDir string, cfg config.LogConfig) error {
	var writer io.Writer

	// Determine output destination
	switch cfg.Destination {
	case "stdout":
		writer = os.Stdout
	case "stderr", "stdin": // stdin is treated as stderr (typo compatibility)
		writer = os.Stderr
	case "file":
		// Create log file (overwrite if exists)
		logPath := filepath.Join(logDir, "gndb.log")
		file, err := os.Create(logPath)
		if err != nil {
			return CreateLogFileError(logPath, err)
		}
		writer = file
	default:
		writer = os.Stderr
	}

	// Parse log level
	level := parseLevel(cfg.Level)

	// Create handler based on format
	var handler slog.Handler
	switch cfg.Format {
	case "text":
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: level,
		})
	case "tint":
		// For now, treat tint same as text
		// TODO: Use github.com/lmittmann/tint if desired
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: level,
		})
	case "json":
		fallthrough
	default:
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: level,
		})
	}

	// Set as default logger
	slog.SetDefault(slog.New(handler))

	return nil
}

// parseLevel converts string level to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
