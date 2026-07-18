package logger

import (
	"log/slog"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func New() fx.Option {
	level := slog.LevelInfo
	// UnmarshalText leaves level untouched on an empty or unrecognized value, so this
	// falls back to info without any extra handling.
	_ = level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)
	return fx.WithLogger(withLogger)
}

func withLogger() fxevent.Logger {
	return &fxevent.SlogLogger{Logger: slog.Default()}
}
