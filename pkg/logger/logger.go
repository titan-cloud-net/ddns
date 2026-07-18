package logger

import (
	"log/slog"
	"os"
)

func New() *slog.Logger {
	level := slog.LevelInfo
	// UnmarshalText leaves level untouched on an empty or unrecognized value, so this
	// falls back to info without any extra handling.
	_ = level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
