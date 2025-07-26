package rest

import (
	"net/http"
	"strings"

	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/rest/handlers"
	"github.com/rediwo/redi-orm/rest/middleware"
)

// Router handles REST API routing
type Router struct {
	mux         *http.ServeMux
	connHandler *handlers.ConnectionHandler
	dataHandler *handlers.DataHandler
	logger      logger.Logger
}

// NewRouter creates a new REST API router
func NewRouter(l logger.Logger) *Router {
	if l == nil {
		l = logger.NewDefaultLogger("REST")
	}

	connHandler := handlers.NewConnectionHandler(l)
	dataHandler := handlers.NewDataHandler(connHandler, l)

	router := &Router{
		mux:         http.NewServeMux(),
		connHandler: connHandler,
		dataHandler: dataHandler,
		logger:      l,
	}

	router.setupRoutes()
	return router
}

// setupRoutes configures all API routes
func (r *Router) setupRoutes() {
	// Connection management
	r.mux.HandleFunc("/api/connections", r.withMiddleware(r.connHandler.List))
	r.mux.HandleFunc("/api/connections/connect", r.withMiddleware(r.connHandler.Connect))
	r.mux.HandleFunc("/api/connections/disconnect", r.withMiddleware(r.connHandler.Disconnect))

	// Data operations - using a pattern that matches model names
	r.mux.HandleFunc("/api/", r.withMiddleware(r.handleDataOperations))
}

// handleDataOperations routes data operations based on URL pattern
func (r *Router) handleDataOperations(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method

	// Check if it's a batch operation
	if isBatchOperation(path) {
		if method == http.MethodPost && endsWith(path, "/batch") {
			r.dataHandler.BatchCreate(w, req)
		} else {
			http.NotFound(w, req)
		}
		return
	}

	// Check if it has an ID (single record operation)
	if hasID(path) {
		switch method {
		case http.MethodGet:
			r.dataHandler.FindOne(w, req)
		case http.MethodPut, http.MethodPatch:
			r.dataHandler.Update(w, req)
		case http.MethodDelete:
			r.dataHandler.Delete(w, req)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Collection operations
	switch method {
	case http.MethodGet:
		r.dataHandler.Find(w, req)
	case http.MethodPost:
		r.dataHandler.Create(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// withMiddleware wraps a handler with middleware
func (r *Router) withMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return middleware.Chain(
		handler,
		middleware.CORS(),
		middleware.JSON(),
		middleware.Logging(r.logger),
	)
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Helper functions
func isBatchOperation(path string) bool {
	return endsWith(path, "/batch")
}

func hasID(path string) bool {
	// Remove /api/ prefix and check if there's an ID component
	trimmed := path[5:] // Remove "/api/"
	parts := splitPath(trimmed)
	return len(parts) >= 2 && parts[1] != ""
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range strings.Split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
