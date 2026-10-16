package shortener

import "net/http"

type ShortenerHandler struct {
}

func New() *ShortenerHandler {
	return &ShortenerHandler{}
}

func (s *ShortenerHandler) Mount() error {
	http.HandleFunc("POST /shorten", s.shortenUrl)
	return nil
}

func (s *ShortenerHandler) shortenUrl(w http.ResponseWriter, r *http.Request) {
    form := r.Form.Get("url")
    if form == "" {
        http.Error(w, "No URL found", http.StatusBadRequest)
        return
    }
}
