// Package logger provides structured logging using log/slog.
package logger

import (
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

// Init initializes the global package-level logger.
func Init(logFilePath string, format string) error {
	var writer io.Writer = os.Stderr // Default to stderr

	// Wire --log-file flag to write to file
	if logFilePath != "" {
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writer = file
	}

	Log = New("info", format, writer)
	slog.SetDefault(Log)
	return nil
}
