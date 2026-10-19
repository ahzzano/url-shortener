package rerouter

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"time"
	"url-shortener/database/cache"
	queries "url-shortener/database/postgres"
)

type RerouterService struct {
	database queries.Queries
	cache    cache.RedisCache
}

func (s *RerouterService) GetURL(ctx context.Context, id string) (string, error) {
	var cachedURL string
	hit, err := s.cache.Get(ctx, id, &cachedURL)

	if err != nil && !errors.Is(err, cache.CacheMissError) {
		slog.Warn("Cache error", "error", err.Error())
	}

	if hit {
		slog.Info("Cache hit", "url", cachedURL)
		return cachedURL, nil
	}

	idInt, err := strconv.ParseInt(id, 36, 64)
	if err != nil {
		return "", err
	}

	url, err := s.database.GetURLByID(ctx, idInt)
	if err != nil {
		return "", err
	}

	s.cache.Set(ctx, id, url, time.Minute*3)

	return url, nil
}
