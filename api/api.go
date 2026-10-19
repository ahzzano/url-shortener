package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"url-shortener/database/cache"
	"url-shortener/middleware"
	"url-shortener/services/rerouter"
	"url-shortener/services/shortener"

	"github.com/redis/go-redis/v9"
)

type APIConfig struct {
	Database          *sql.DB
	RedisDb           *redis.Client
	ShortenMiddleware middleware.Middleware
}

func New(config *APIConfig) http.Handler {
	mux := http.NewServeMux()

	rerouter.New(&rerouter.RerouterHandlerParams{
		DBTX:  config.Database,
		Cache: cache.New(config.RedisDb),
	}).Mount(mux)

	shortener.New(&shortener.ShortenerHandlerParams{
		DBTX: config.Database,
	}).Mount(mux, config.ShortenMiddleware)

	handler := middleware.LoggingMiddleware(mux)
	slog.Info("Mounted the API")
	return handler
}
