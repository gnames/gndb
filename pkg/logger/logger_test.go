package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/gnames/gndb/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_TextFormat(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "text",
	}

	logger := New(cfg)
	logger.Info("test message", "key", "value")

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	assert.Nil(t, err)
	output := buf.String()

	// Verify text format characteristics
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
	assert.Contains(t, output, "level=INFO")
}

func TestNew_JSONFormat(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "json",
	}

	logger := New(cfg)
	logger.Info("test message", "key", "value")

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	assert.Nil(t, err)
	output := buf.String()
	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err, "Output should be valid JSON")

	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, "INFO", logEntry["level"])
	assert.Contains(t, logEntry, "time") // Should have timestamp
}

func TestNew_LogLevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		configLevel   string
		logFunc       func(*slog.Logger)
		shouldContain string
		shouldLog     bool
	}{
		{
			name:          "info level shows info messages",
			configLevel:   "info",
			logFunc:       func(l *slog.Logger) { l.Info("info message") },
			shouldContain: "info message",
			shouldLog:     true,
		},
		{
			name:          "info level hides debug messages",
			configLevel:   "info",
			logFunc:       func(l *slog.Logger) { l.Debug("debug message") },
			shouldContain: "debug message",
			shouldLog:     false,
		},
		{
			name:          "debug level shows debug messages",
			configLevel:   "debug",
			logFunc:       func(l *slog.Logger) { l.Debug("debug message") },
			shouldContain: "debug message",
			shouldLog:     true,
		},
		{
			name:          "warn level hides info messages",
			configLevel:   "warn",
			logFunc:       func(l *slog.Logger) { l.Info("info message") },
			shouldContain: "info message",
			shouldLog:     false,
		},
		{
			name:          "error level only shows errors",
			configLevel:   "error",
			logFunc:       func(l *slog.Logger) { l.Warn("warn message") },
			shouldContain: "warn message",
			shouldLog:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			cfg := &config.LoggingConfig{
				Level:  tt.configLevel,
				Format: "text",
			}

			logger := New(cfg)
			tt.logFunc(logger)

			// Restore stdout
			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			assert.Nil(t, err)
			output := buf.String()

			if tt.shouldLog {
				assert.Contains(t, output, tt.shouldContain)
			} else {
				assert.NotContains(t, output, tt.shouldContain)
			}
		})
	}
}

func TestNew_InvalidLevelDefaultsToInfo(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.LoggingConfig{
		Level:  "invalid",
		Format: "text",
	}

	logger := New(cfg)

	// Debug should be hidden at default Info level
	logger.Debug("debug message")
	// Info should be shown
	logger.Info("info message")

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	assert.Nil(t, err)
	output := buf.String()

	assert.NotContains(t, output, "debug message", "Debug should be hidden at default Info level")
	assert.Contains(t, output, "info message", "Info should be shown at default Info level")
}

func TestNew_InvalidFormatDefaultsToText(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "invalid",
	}

	logger := New(cfg)
	logger.Info("test message")

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	assert.Nil(t, err)
	output := buf.String()

	// Should be text format, not JSON
	assert.Contains(t, output, "level=INFO")
	assert.Contains(t, output, "test message")

	// Should NOT be valid JSON
	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(output), &logEntry)
	assert.Error(t, err, "Output should not be valid JSON when format is invalid")
}

func TestNew_EmptyFormatDefaultsToText(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "",
	}

	logger := New(cfg)
	logger.Info("test message")

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	assert.Nil(t, err)
	output := buf.String()

	// Should be text format
	assert.Contains(t, output, "level=INFO")
	assert.Contains(t, output, "test message")
}

func TestNew_LoggerIncludesTimestamp(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "json",
	}

	logger := New(cfg)
	logger.Info("test message")

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	assert.Nil(t, err)
	output := buf.String()

	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)

	// Verify timestamp exists
	_, hasTime := logEntry["time"]
	assert.True(t, hasTime, "Log entry should include timestamp")
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"", slog.LevelInfo},        // Empty defaults to Info
		{"invalid", slog.LevelInfo}, // Invalid defaults to Info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNew_CaseInsensitiveFormat(t *testing.T) {
	formats := []string{"JSON", "Json", "json", "TEXT", "Text", "text"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			cfg := &config.LoggingConfig{
				Level:  "info",
				Format: format,
			}

			// Should not panic
			logger := New(cfg)
			assert.NotNil(t, logger)
		})
	}
}
