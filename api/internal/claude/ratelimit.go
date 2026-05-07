package claude

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrMinuteLimitExceeded = errors.New("minute_limit")
	ErrHourLimitExceeded   = errors.New("hour_limit")
)

const (
	minuteLimit = 5
	hourLimit   = 20
)

// slidingWindowScript atomically checks and records a request in a sorted-set
// sliding window.
//
// KEYS[1] = redis key
// ARGV[1] = current time in nanoseconds (score + unique member)
// ARGV[2] = window duration in nanoseconds
// ARGV[3] = limit (integer)
// ARGV[4] = TTL in seconds for the key
//
// Returns 1 if the request is allowed, 0 if the limit is exceeded.
var slidingWindowScript = redis.NewScript(`
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit  = tonumber(ARGV[3])
local ttl    = tonumber(ARGV[4])

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

local count = redis.call('ZCARD', key)
if count < limit then
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, ttl)
    return 1
end
return 0
`)

type RateLimiter struct {
	rdb *redis.Client
}

func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// Check enforces the per-tenant sliding window limits.
// Returns nil if allowed, ErrMinuteLimitExceeded or ErrHourLimitExceeded if not.
func (rl *RateLimiter) Check(ctx context.Context, tenantID string) error {
	now := time.Now().UnixNano()

	type window struct {
		key      string
		duration int64 // nanoseconds
		ttl      int   // seconds
		limit    int
		limitErr error
	}

	windows := []window{
		{
			key:      fmt.Sprintf("claude_rl:%s:minute", tenantID),
			duration: int64(time.Minute),
			ttl:      60,
			limit:    minuteLimit,
			limitErr: ErrMinuteLimitExceeded,
		},
		{
			key:      fmt.Sprintf("claude_rl:%s:hour", tenantID),
			duration: int64(time.Hour),
			ttl:      3600,
			limit:    hourLimit,
			limitErr: ErrHourLimitExceeded,
		},
	}

	for _, w := range windows {
		allowed, err := slidingWindowScript.Run(ctx, rl.rdb,
			[]string{w.key},
			now, w.duration, w.limit, w.ttl,
		).Int()
		if err != nil {
			return fmt.Errorf("rate limit check (%s): %w", w.key, err)
		}
		if allowed == 0 {
			return w.limitErr
		}
	}

	return nil
}
