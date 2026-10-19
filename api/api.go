package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"url-shortener/middleware"
	"url-shortener/services/rerouter"
	"url-shortener/services/shortener"
)

func New(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	rerouter.New(&rerouter.RerouterHandlerParams{
		DBTX: db,
	}).Mount(mux)

	shortener.New(&shortener.ShortenerHandlerParams{
		DBTX: db,
	}).Mount(mux)

	handler := middleware.LoggingMiddleware(mux)
	slog.Info("Mounted the API")
	return handler
}
