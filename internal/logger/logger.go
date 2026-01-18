package logger

import (
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

// Init initializes the global logger with JSON output.
// Call this early in main() before any logging occurs.
func Init() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// parseLevel converts string to slog.Level
func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Default returns the configured default logger
func Default() *slog.Logger {
	if defaultLogger == nil {
		Init()
	}
	return defaultLogger
}
