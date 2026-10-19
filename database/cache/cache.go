package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	CacheMissError = errors.New("Cache missed")
)

type RedisCache struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) RedisCache {
	return RedisCache{
		rdb: rdb,
	}
}

func (s *RedisCache) Get(ctx context.Context, key string, result interface{}) (bool, error) {
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, CacheMissError
	} else if err != nil {
		return false, errors.New("Redis error")
	}

	err = json.Unmarshal([]byte(val), result)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.rdb.Set(ctx, key, jsonData, ttl).Err()
}
