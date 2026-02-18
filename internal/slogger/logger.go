package slogger

import (
	"log/slog"
	"os"
)

// Setup creates and configures a slog.Logger and sets it as the default.
// format should be "json" (production) or "text" (development).
func Setup(format string) *slog.Logger {
	level := slog.LevelInfo
	if os.Getenv("PXBIN_LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
