package logging

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger builds a JSON logger tuned for production use.
// We prefer slog here because it keeps the standard library feel
// while still emitting structured logs we can ship to any backend.
func NewLogger(level string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     levelFromString(level),
		AddSource: true,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

func levelFromString(level string) slog.Leveler {
	switch strings.ToLower(strings.TrimSpace(level)) {
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
