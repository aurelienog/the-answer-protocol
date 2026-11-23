package logger

import (
    "log/slog"
    "os"
)

func New() *slog.Logger {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level:     slog.LevelInfo,
        AddSource: true,
    })

    return slog.New(handler)
}