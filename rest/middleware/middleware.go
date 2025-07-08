package middleware

import "net/http"

// Middleware represents a middleware function
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain applies multiple middleware in sequence
func Chain(handler http.HandlerFunc, middleware ...Middleware) http.HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}
