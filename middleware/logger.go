package middleware

import (
	"log/slog"
	"net/http"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	logger := slog.Default()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request", "method", r.Method, "url", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
