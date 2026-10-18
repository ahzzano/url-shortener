package services

import (
	"encoding/json"
	"io"
	"net/http"
)

type Handler interface {
	Mount(*http.ServeMux) error
}

func ToJSONResponse(w io.Writer, value any) {
	encoder := json.NewEncoder(w)
	encoder.Encode(value)
}
