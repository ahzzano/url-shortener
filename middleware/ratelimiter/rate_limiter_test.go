package ratelimiter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRateLimiterDecisions(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { _ = client.Close() })
	for _, tt := range []struct {
		name       string
		decision   Decision
		err        error
		wantStatus int
		wantRetry  string
		wantNext   bool
	}{
		{name: "allowed", decision: Decision{Allowed: true}, wantStatus: http.StatusNoContent, wantNext: true},
		{name: "denied exact second", decision: Decision{RetryAfter: time.Second}, wantStatus: http.StatusTooManyRequests, wantRetry: "1"},
		{name: "denied fractional second", decision: Decision{RetryAfter: 1500 * time.Millisecond}, wantStatus: http.StatusTooManyRequests, wantRetry: "2"},
		{name: "denied subsecond", decision: Decision{RetryAfter: time.Nanosecond}, wantStatus: http.StatusTooManyRequests, wantRetry: "1"},
		{name: "denied zero", wantStatus: http.StatusTooManyRequests, wantRetry: "1"},
		{name: "denied negative", decision: Decision{RetryAfter: -time.Second}, wantStatus: http.StatusTooManyRequests, wantRetry: "1"},
		{name: "algorithm error", decision: Decision{Allowed: true}, err: errors.New("Redis unavailable"), wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var algorithmContext context.Context
			calls := 0
			limiter := RateLimiter{Algorithm: func(ctx context.Context, gotClient *redis.Client, key string) (Decision, error) {
				calls++
				algorithmContext = ctx
				if gotClient != client || key != "ratelimit:shorten:192.0.2.1" {
					t.Errorf("algorithm received client %p and key %q", gotClient, key)
				}
				return tt.decision, tt.err
			}}
			nextCalled := false
			handler := limiter.New(client)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				if algorithmContext.Err() != context.Canceled {
					t.Error("algorithm context was not canceled before downstream handler")
				}
				if r.Context().Err() != nil {
					t.Errorf("request context canceled: %v", r.Context().Err())
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodPost, "/shorten", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)

			if response.Code != tt.wantStatus || response.Header().Get("Retry-After") != tt.wantRetry {
				t.Errorf("response = (%d, Retry-After %q), want (%d, %q)", response.Code, response.Header().Get("Retry-After"), tt.wantStatus, tt.wantRetry)
			}
			if nextCalled != tt.wantNext || calls != 1 {
				t.Errorf("next called = %v, algorithm calls = %d; want %v, 1", nextCalled, calls, tt.wantNext)
			}
			if algorithmContext == nil || algorithmContext.Err() != context.Canceled {
				t.Error("algorithm context not canceled after algorithm returned")
			}
		})
	}
}

func TestRateLimiterClientIP(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { _ = client.Close() })
	for _, tt := range []struct {
		name, remote, ip string
	}{
		{"IPv4", "192.0.2.1:1234", "192.0.2.1"},
		{"different port", "192.0.2.1:5678", "192.0.2.1"},
		{"IPv6", "[2001:0db8:0:0:0:0:0:1]:1234", "2001:db8::1"},
		{"mapped IPv4", "[::ffff:192.0.2.1]:1234", "192.0.2.1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var gotKey string
			limiter := RateLimiter{Algorithm: func(_ context.Context, _ *redis.Client, key string) (Decision, error) {
				gotKey = key
				return Decision{Allowed: true}, nil
			}}
			handler := limiter.New(client)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodPost, "/shorten", nil)
			req.RemoteAddr = tt.remote
			req.Header.Set("X-Forwarded-For", "203.0.113.99")
			req.Header.Set("Forwarded", "for=203.0.113.98")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)

			if want := "ratelimit:shorten:" + tt.ip; gotKey != want {
				t.Errorf("key = %q, want %q", gotKey, want)
			}
			if response.Code != http.StatusNoContent {
				t.Errorf("status = %d, want 204", response.Code)
			}
		})
	}
}

func TestRateLimiterInvalidRemoteAddr(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { _ = client.Close() })
	for _, remote := range []string{"", "192.0.2.1", "2001:db8::1", "example.com:1234", "192.0.2.1:not-a-port", "192.0.2.1:65536"} {
		t.Run(remote, func(t *testing.T) {
			limiter := RateLimiter{Algorithm: func(context.Context, *redis.Client, string) (Decision, error) {
				t.Error("algorithm called for invalid RemoteAddr")
				return Decision{Allowed: true}, nil
			}}
			handler := limiter.New(client)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("downstream called for invalid RemoteAddr")
			}))
			req := httptest.NewRequest(http.MethodPost, "/shorten", nil)
			req.RemoteAddr = remote
			req.Header.Set("X-Forwarded-For", "192.0.2.1")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, req)

			if response.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", response.Code)
			}
		})
	}
}

func TestRateLimiterContext(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { _ = client.Close() })
	type contextKey struct{}
	for _, tt := range []struct {
		name    string
		timeout time.Duration
	}{
		{"default deadline", 0},
		{"earlier parent deadline", 100 * time.Millisecond},
		{"later parent deadline", time.Hour},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parent := context.WithValue(context.Background(), contextKey{}, "request value")
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				parent, cancel = context.WithTimeout(parent, tt.timeout)
				defer cancel()
			}
			before := time.Now()
			limiter := RateLimiter{Algorithm: func(ctx context.Context, _ *redis.Client, _ string) (Decision, error) {
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Fatal("algorithm context has no deadline")
				}
				if tt.timeout > 0 && tt.timeout < 500*time.Millisecond {
					want, _ := parent.Deadline()
					if !deadline.Equal(want) {
						t.Errorf("deadline = %v, want parent deadline %v", deadline, want)
					}
				} else if deadline.Before(before.Add(500*time.Millisecond)) || deadline.After(time.Now().Add(500*time.Millisecond)) {
					t.Errorf("deadline %v not within 500ms of middleware invocation", deadline)
				}
				if ctx.Value(contextKey{}) != "request value" {
					t.Error("request context value not propagated")
				}
				return Decision{Allowed: true}, nil
			}}
			req := httptest.NewRequest(http.MethodPost, "/shorten", nil).WithContext(parent)
			req.RemoteAddr = "192.0.2.1:1234"
			handler := limiter.New(client)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				if r.Context() != parent {
					t.Error("downstream request context was replaced")
				}
			}))

			handler.ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}

func TestRateLimiterParentCancellation(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { _ = client.Close() })
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	limiter := RateLimiter{Algorithm: func(ctx context.Context, _ *redis.Client, _ string) (Decision, error) {
		cancel()
		if ctx.Err() != context.Canceled {
			t.Errorf("algorithm context error = %v, want context.Canceled", ctx.Err())
		}
		return Decision{}, ctx.Err()
	}}
	req := httptest.NewRequest(http.MethodPost, "/shorten", nil).WithContext(parent)
	req.RemoteAddr = "192.0.2.1:1234"
	handler := limiter.New(client)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("downstream called after algorithm returned cancellation error")
	}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, req)

	if response.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", response.Code)
	}
}

func TestRateLimiterInvalidConfig(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { _ = client.Close() })
	algorithm := func(context.Context, *redis.Client, string) (Decision, error) { return Decision{}, nil }
	for _, tt := range []struct {
		name    string
		limiter RateLimiter
		client  *redis.Client
	}{
		{"nil algorithm", RateLimiter{}, client},
		{"nil client", RateLimiter{Algorithm: algorithm}, nil},
		{"both nil", RateLimiter{}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("New did not panic for invalid configuration")
				}
			}()
			tt.limiter.New(tt.client)
		})
	}
}

func TestFixedWindowInvalid(t *testing.T) {
	for _, tt := range []struct {
		name string
		new  func()
	}{
		{"zero limit", func() { FixedWindow(0, time.Second) }},
		{"negative limit", func() { FixedWindow(-1, time.Second) }},
		{"zero window", func() { FixedWindow(1, 0) }},
		{"negative window", func() { FixedWindow(1, -time.Second) }},
		{"submillisecond window", func() { FixedWindow(1, time.Millisecond-time.Nanosecond) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("FixedWindow did not panic for invalid configuration")
				}
			}()
			tt.new()
		})
	}
}
