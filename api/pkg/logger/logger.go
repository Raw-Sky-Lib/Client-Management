package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

var Log *slog.Logger

func InitLogger(environment string) {
	if environment == "production" {
		Log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		Log = slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level:      LevelTrace,
			TimeFormat: time.Kitchen,
		}))
	}
}
