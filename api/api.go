package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"url-shortener/middleware"
	"url-shortener/services"
	"url-shortener/services/rerouter"
	"url-shortener/services/shortener"
)

type App struct {
	services    []services.Handler
	middleware  []middleware.Middleware
	mux         *http.ServeMux
	mainHandler http.Handler
}

func New(db *sql.DB) App {
	return App{
		services: []services.Handler{
			rerouter.New(),
			shortener.New(&shortener.ShortenerHandlerParams{
				DBTX: db,
			}),
		},
		middleware: []middleware.Middleware{
			middleware.LoggingMiddleware,
		},
	}
}

func (s *App) mountMiddlewares() http.Handler {
	var handler http.Handler = s.mux
	for _, m := range s.middleware {
		handler = m(handler)
	}

	return handler
}

func (s *App) Mount() {
	s.mux = http.NewServeMux()

	for _, service := range s.services {
		service.Mount(s.mux)
	}

	s.mainHandler = s.mountMiddlewares()
	slog.Info("Mounted the API")
}

func (s *App) Run() {
	slog.Info("Running the API")
	http.ListenAndServe(":8080", s.mainHandler)
}
