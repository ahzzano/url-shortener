package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"url-shortener/middleware/ratelimiter"

	"github.com/redis/go-redis/v9"
)

func TestShortenMiddlewareScope(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { client.Close() })
	calls := 0
	limiter := ratelimiter.RateLimiter{Algorithm: func(context.Context, *redis.Client, string) (ratelimiter.Decision, error) {
		calls++
		return ratelimiter.Decision{RetryAfter: time.Second}, nil
	}}
	handler := New(&APIConfig{RedisDb: client, ShortenMiddleware: limiter.New(client)})
	for _, tc := range []struct {
		method, path string
		status       int
		calls        int
	}{
		{http.MethodPost, "/shorten", http.StatusTooManyRequests, 1},
		// Invalid IDs exercise the redirect handler without a live database.
		{http.MethodGet, "/!", http.StatusBadRequest, 1},
		{http.MethodHead, "/!", http.StatusBadRequest, 1},
		{http.MethodPost, "/other", http.StatusMethodNotAllowed, 1},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.RemoteAddr = "192.0.2.1:1234"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != tc.status || calls != tc.calls {
			t.Errorf("%s %s: status=%d calls=%d, want status=%d calls=%d", tc.method, tc.path, response.Code, calls, tc.status, tc.calls)
		}
	}
}

func TestShortenWithoutMiddleware(t *testing.T) {
	handler := New(&APIConfig{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/shorten", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 from the shortening handler", response.Code)
	}
}
