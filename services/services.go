package services

import (
	"encoding/json"
	"net/http"
)

type Handler interface {
	Mount(*http.ServeMux) error
}

func ToJSONResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.Encode(value)
}
