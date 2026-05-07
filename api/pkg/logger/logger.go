package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

var Log *slog.Logger

func InitLogger(env string) {
	var handler slog.Handler //setting the slog handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceAttrProd})
	} else {
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:       LevelTrace,
			TimeFormat:  time.Kitchen,
			ReplaceAttr: replaceAttrDev,
		})
	}

	Log = slog.New(handler)

}
