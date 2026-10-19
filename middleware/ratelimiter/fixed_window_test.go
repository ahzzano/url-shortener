package ratelimiter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// Run with RATE_LIMIT_TEST_REDIS_ADDR=localhost:6379 against a development Redis.
func TestFixedWindowRedis(t *testing.T) {
	addr := os.Getenv("RATE_LIMIT_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set RATE_LIMIT_TEST_REDIS_ADDR to run Redis integration tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr, ContextTimeoutEnabled: true})
	t.Cleanup(func() { client.Close() })
	secondClient := redis.NewClient(&redis.Options{Addr: addr, ContextTimeoutEnabled: true})
	t.Cleanup(func() { secondClient.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	key := fmt.Sprintf("ratelimit:test:%d", time.Now().UnixNano())
	otherKey := key + ":other"
	t.Cleanup(func() { client.Del(context.Background(), key, otherKey) })

	const limit = 7
	algorithm := FixedWindow(limit, 5*time.Second)
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := range 40 {
		requestClient := client
		if i%2 == 0 {
			requestClient = secondClient
		}
		wg.Go(func() {
			decision, err := algorithm(ctx, requestClient, key)
			if err != nil {
				t.Error(err)
				return
			}
			if decision.Allowed {
				allowed.Add(1)
			}
			if decision.RetryAfter <= 0 || decision.RetryAfter > 5*time.Second {
				t.Errorf("unexpected retry duration: %v", decision.RetryAfter)
			}
		})
	}
	wg.Wait()
	if got := allowed.Load(); got != limit {
		t.Fatalf("allowed %d concurrent requests, want %d", got, limit)
	}
	decision, err := algorithm(ctx, client, otherKey)
	if err != nil || !decision.Allowed {
		t.Fatalf("independent key: decision=%+v err=%v", decision, err)
	}

	// Shorten only our test key's TTL to check denials do not reset it.
	if err := client.PExpire(ctx, key, 150*time.Millisecond).Err(); err != nil {
		t.Fatal(err)
	}
	decision, err = algorithm(ctx, client, key)
	if err != nil || decision.Allowed || decision.RetryAfter > 150*time.Millisecond {
		t.Fatalf("denial extended expiry: decision=%+v err=%v", decision, err)
	}
	time.Sleep(200 * time.Millisecond)
	decision, err = algorithm(ctx, client, key)
	if err != nil || !decision.Allowed {
		t.Fatalf("window did not reset: decision=%+v err=%v", decision, err)
	}
	if count := client.Get(ctx, key).Val(); count != "1" {
		t.Fatalf("reset count=%q, want 1", count)
	}

	closedClient := redis.NewClient(&redis.Options{Addr: addr})
	if err := closedClient.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := algorithm(ctx, closedClient, key); err == nil {
		t.Fatal("expected error from closed Redis client")
	}
}
