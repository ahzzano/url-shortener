package rerouter

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"url-shortener/database/cache"
	queries "url-shortener/database/postgres"
)

type Rerouter struct {
	service RerouterService
}

type RerouterHandlerParams struct {
	DBTX  queries.DBTX
	Cache cache.RedisCache
}

func New(params *RerouterHandlerParams) *Rerouter {
	return &Rerouter{
		RerouterService{
			database: *queries.New(params.DBTX),
			cache:    params.Cache,
		},
	}
}

func (s *Rerouter) Mount(mux *http.ServeMux) error {
	mux.HandleFunc("GET /{id}", s.redirect)
	return nil
}

func (s *Rerouter) redirect(w http.ResponseWriter, r *http.Request) {
	urlId := r.PathValue("id")
	if urlId == "" {
		http.Error(w, "URL ID not provided", http.StatusBadRequest)
		return
	}

	url, err := s.service.GetURL(r.Context(), urlId)
	if err != nil {
		var numErr *strconv.NumError
		switch {
		case errors.As(err, &numErr):
			http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		case errors.Is(err, sql.ErrNoRows):
			http.Error(w, "URL not found", http.StatusNotFound)
		default:
			slog.Error("Unable to find URL ID", "error", err.Error())
			http.Error(w, "Unable to find url ID", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}
