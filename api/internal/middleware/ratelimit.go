package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DagMT/client-portal/internal/utils"
	"github.com/redis/go-redis/v9"
)

// RateLimit is a sliding-window rate limiter keyed by keyPrefix + client IP.
// Fails open on Redis errors so a Redis outage never blocks requests.
func RateLimit(rdb *redis.Client, keyPrefix string, requests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			key := fmt.Sprintf("rl:%s:%s", keyPrefix, ip)
			now := time.Now()
			cutoff := now.Add(-window)

			ctx := context.Background()
			pipe := rdb.Pipeline()
			pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", cutoff.UnixMilli()))
			pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMilli()), Member: now.UnixNano()})
			countCmd := pipe.ZCard(ctx, key)
			pipe.Expire(ctx, key, window)

			if _, err := pipe.Exec(ctx); err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if countCmd.Val() > int64(requests) {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				utils.RespondError(w, http.StatusTooManyRequests, "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func realIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	return r.RemoteAddr
}
