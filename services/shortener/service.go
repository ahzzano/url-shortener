package shortener

import "net/http"

type ShortenerService struct {
}

func New() *ShortenerService {
	return &ShortenerService{}
}

func (s *ShortenerService) Mount() error {
	http.HandleFunc("POST /shorten", s.shortenUrl)
	return nil
}

func (s *ShortenerService) shortenUrl(w http.ResponseWriter, r *http.Request) {
    form := r.Form.Get("url")
    if form == "" {
        http.Error(w, "No URL found", http.StatusBadRequest)
        return
    }
}
