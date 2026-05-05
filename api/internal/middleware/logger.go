package middleware

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/DagMT/client-portal/pkg/logger"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(status int) {
	sr.status = status
	sr.ResponseWriter.WriteHeader(status)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := fmt.Sprintf("%016x", rand.Int63())

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		w.Header().Set("X-Request-ID", reqID)

		next.ServeHTTP(rec, r)

		logger.Log.Info("request",
			slog.String("RequestID", reqID),
			slog.String("Method", r.Method),
			slog.String("Path", r.URL.Path),
			slog.Int("Status", rec.status),
			slog.Duration("Elapsed", time.Since(start)),
		)
	})
}
