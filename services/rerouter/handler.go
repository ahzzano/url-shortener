package rerouter

import "net/http"

type Rerouter struct {
}

func New() *Rerouter {
	return &Rerouter{}
}

func (s *Rerouter) Mount(mux *http.ServeMux) error {
	mux.HandleFunc("GET /{id}", s.redirect)
	return nil
}

func (s *Rerouter) redirect(w http.ResponseWriter, r *http.Request) {
}
