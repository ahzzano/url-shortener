package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"
	"url-shortener/api"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("API initialization failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return errors.New("invalid DATABASE_URL")
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		// Driver errors may contain credentials from the connection string.
		return errors.New("cannot connect to PostgreSQL; check DATABASE_URL and database availability")
	}

	app := api.New(db)
	app.Mount()
	app.Run()
	return nil
}
