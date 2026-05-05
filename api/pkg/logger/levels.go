package logger

import (
	"context"
	"log/slog"
	"os"
)

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
)

func Trace(msg string, args ...any) {
	Log.Log(context.Background(), LevelTrace, msg, args...)
}

func Fatal(msg string, args ...any) {
	Log.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}
