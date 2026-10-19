package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"url-shortener/api"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("API initialization failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	db, err := setupDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	rdb := setupRedis()

	handler := api.New(&api.APIConfig{
		Database: db,
		RedisDb:  rdb,
	})
	slog.Info("Running the API")
	return http.ListenAndServe(":8080", handler)
}

func setupRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
}

func setupDatabase() (*sql.DB, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, errors.New("invalid DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		db.Close()
		// Driver errors may contain credentials from the connection string.
		return nil, errors.New("cannot connect to PostgreSQL; check DATABASE_URL and database availability")
	}

	return db, nil
}
