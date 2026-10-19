package ratelimiter

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var fixedWindowScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return {count, redis.call('PTTL', KEYS[1])}
`)

// FixedWindow returns an algorithm whose window starts with the first request.
// shortcut: fixed windows allow boundary bursts; use a token bucket if smoothing is needed.
// It panics if limit is not positive or window is less than one millisecond.
func FixedWindow(limit int, window time.Duration) Algorithm {
	if limit <= 0 || window < time.Millisecond {
		panic("fixed window requires a positive limit and a window of at least one millisecond")
	}
	return func(ctx context.Context, client *redis.Client, key string) (Decision, error) {
		values, err := fixedWindowScript.Run(ctx, client, []string{key}, window.Milliseconds()).Int64Slice()
		if err != nil {
			return Decision{}, err
		}
		if len(values) != 2 || values[1] < 0 {
			return Decision{}, errors.New("invalid fixed-window counter or expiry")
		}
		return Decision{
			Allowed:    values[0] <= int64(limit),
			RetryAfter: time.Duration(values[1]) * time.Millisecond,
		}, nil
	}
}
