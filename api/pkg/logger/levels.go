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

// Trace logs a message at LevelTrace. Only visible in development.
func Trace(msg string, args ...any) {
	Log.Log(context.Background(), LevelTrace, msg, args...)
}

// Fatal logs a message at LevelFatal then exits the process with status 1.
func Fatal(msg string, args ...any) {
	Log.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}

// replaceAttr maps custom level integers to human-readable names, instead of printing them as `DEBUG-4` or `ERROR+4` — ugly and confusing.
// Used by both the tint (dev) and JSON (prod) handlers in logger.go. In dev trace and Fatal have colors the string value accepts any string hard coded like "\033[31mFATAL\033[0m"
func replaceAttrDev(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		switch a.Value.Any().(slog.Level) {
		case LevelTrace:
			a.Value = slog.StringValue("\033[33mTRC\033[0m") // yellow
		case LevelFatal:
			a.Value = slog.StringValue("\033[31mFATAL\033[0m") // red
		}
	}
	return a
}
func replaceAttrProd(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		switch a.Value.Any().(slog.Level) {
		case LevelTrace:
			a.Value = slog.StringValue("TRC")
		case LevelFatal:
			a.Value = slog.StringValue("FATAL")
		}
	}
	return a
}
