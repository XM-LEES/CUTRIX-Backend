package logger

import (
    "log/slog"
    "os"
)

// L is the global slog logger used across the application.
// It outputs JSON logs; level can be controlled via LOG_LEVEL env: debug|info|warn|error.
var L *slog.Logger

func init() {
    // Configure handler: JSON with levels
    level := new(slog.LevelVar)
    switch os.Getenv("LOG_LEVEL") {
    case "debug": level.Set(slog.LevelDebug)
    case "warn":  level.Set(slog.LevelWarn)
    case "error": level.Set(slog.LevelError)
    default:       level.Set(slog.LevelInfo)
    }
    L = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}