package logger

import (
	"log/slog"
	"os"
)

// Init sets the default slog logger (JSON on stdout). Call once from main.
func Init() {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
}
