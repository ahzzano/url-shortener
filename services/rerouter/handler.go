package rerouter

import "net/http"

type Rerouter struct {
}

func New() *Rerouter {
    return &Rerouter{}
}

func (s *Rerouter) Mount() error {
	http.HandleFunc("GET /{id}", s.redirect)
	return nil
}

func (s *Rerouter) redirect(w http.ResponseWriter, r *http.Request) {
}
