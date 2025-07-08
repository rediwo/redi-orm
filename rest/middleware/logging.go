package middleware

import (
	"net/http"
	"time"

	"github.com/rediwo/redi-orm/utils"
)

// Logging logs HTTP requests
func Logging(logger utils.Logger) Middleware {
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
			logger.Info("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
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
