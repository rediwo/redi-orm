package middleware

import (
	"net/http"
	"time"

	"github.com/rediwo/redi-orm/logger"
)

// Logging logs HTTP requests
func Logging(l logger.Logger) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next(wrapped, r)

			duration := time.Since(start)
			l.Info("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
