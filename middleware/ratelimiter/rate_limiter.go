package ratelimiter

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"time"
	"url-shortener/middleware"

	"github.com/redis/go-redis/v9"
)

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Algorithm atomically checks and consumes a request for the supplied client key.
// A denied decision supplies the duration until another request may be attempted.
type Algorithm func(context.Context, *redis.Client, string) (Decision, error)

type RateLimiter struct {
	Algorithm Algorithm
}

// New panics on missing dependencies so configuration errors fail at startup.
func (r RateLimiter) New(client *redis.Client) middleware.Middleware {
	if client == nil || r.Algorithm == nil {
		panic("rate limiter requires a Redis client and algorithm")
	}
	algorithm := r.Algorithm
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			remote, err := netip.ParseAddrPort(req.RemoteAddr)
			if err != nil {
				http.Error(w, "Invalid client address", http.StatusBadRequest)
				return
			}
			key := "ratelimit:shorten:" + remote.Addr().Unmap().String()
			ctx, cancel := context.WithTimeout(req.Context(), 500*time.Millisecond)
			decision, err := algorithm(ctx, client, key)
			cancel()
			if err != nil {
				slog.Error("Rate limiter unavailable", "error", err)
				http.Error(w, "Rate limiter unavailable", http.StatusServiceUnavailable)
				return
			}
			if !decision.Allowed {
				seconds := decision.RetryAfter / time.Second
				if decision.RetryAfter%time.Second > 0 {
					seconds++
				}
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(int64(seconds), 10))
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}
