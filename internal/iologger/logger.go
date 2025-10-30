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
// If append is true, appends to existing log file; otherwise creates fresh file.
func Init(logDir string, cfg config.LogConfig, append bool) error {
	var writer io.Writer

	// Determine output destination
	switch cfg.Destination {
	case "stdout":
		writer = os.Stdout
	case "stderr", "stdin": // stdin is treated as stderr (typo compatibility)
		writer = os.Stderr
	case "file":
		logPath := filepath.Join(logDir, "gndb.log")
		var file *os.File
		var err error

		if append {
			// Append to existing log file (preserve previous logs)
			file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		} else {
			// Create fresh log file (truncate if exists)
			file, err = os.Create(logPath)
		}

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
	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, handlerOpts)
	case "text":
		handler = slog.NewTextHandler(writer, handlerOpts)
	case "tint":
		// For now, treat tint same as text
		// TODO: Use github.com/lmittmann/tint if desired
		handler = slog.NewTextHandler(writer, handlerOpts)
	default:
		// Default to JSON format for any unrecognized format
		handler = slog.NewJSONHandler(writer, handlerOpts)
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
