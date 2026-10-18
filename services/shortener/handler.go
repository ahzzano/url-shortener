package shortener

import (
	"log/slog"
	"net/http"
	queries "url-shortener/database/postgres"
	"url-shortener/services"
)

type ShortenerHandler struct {
	service ShortenerService
}

type ShortenerHandlerParams struct {
	DBTX queries.DBTX
}

type ShortenURLResponse struct {
	URL string `json:"url"`
}

func New(params *ShortenerHandlerParams) *ShortenerHandler {
	return &ShortenerHandler{
		service: ShortenerService{queries: queries.New(params.DBTX)},
	}
}

func (s *ShortenerHandler) Mount(mux *http.ServeMux) error {
	mux.HandleFunc("POST /shorten", s.shortenUrl)
	return nil
}

func (s *ShortenerHandler) shortenUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	url := r.Form.Get("url")
	if url == "" {
		http.Error(w, "No URL found", http.StatusBadRequest)
		slog.Error("Error shortening url", "error", "URL not provided")
		return
	}

	shortened, err := s.service.shortenURL(r.Context(), url)
	if err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		slog.Error("Error shortening url", "error", err.Error())
		return
	}

	services.ToJSONResponse(w, ShortenURLResponse{
		URL: shortened,
	})
}
