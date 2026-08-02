// Package logger provides structured logging using log/slog.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Log is the package-level default logger var
var Log *slog.Logger

// New creates and configures a new slog instance.
func New(level string, format string, writer io.Writer) *slog.Logger {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	var handler slog.Handler
	if strings.ToLower(format) == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	return slog.New(handler)
}

// validLevels are the only --log-level values New will honor; anything else
// silently became "info" before this check existed.
var validLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

// validFormats are the only --log-format values New will honor.
var validFormats = map[string]bool{"text": true, "json": true}

// Init initializes the global package-level logger. Logs go to stderr by
// default so stdout stays clean for `--json | jq`.
func Init(logFilePath, format, level string) error {
	normLevel := strings.ToLower(strings.TrimSpace(level))
	if !validLevels[normLevel] {
		return fmt.Errorf("invalid --log-level %q: must be one of debug, info, warn, error", level)
	}

	normFormat := strings.ToLower(strings.TrimSpace(format))
	if !validFormats[normFormat] {
		return fmt.Errorf("invalid --log-format %q: must be one of text, json", format)
	}

	var writer io.Writer = os.Stderr

	// Wire --log-file flag to write to file
	if logFilePath != "" {
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		writer = file
	}

	Log = New(normLevel, normFormat, writer)
	slog.SetDefault(Log)
	return nil
}
