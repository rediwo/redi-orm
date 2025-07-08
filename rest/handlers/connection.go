package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/rest/types"
	"github.com/rediwo/redi-orm/utils"
)

// ConnectionHandler handles database connections
type ConnectionHandler struct {
	connections map[string]database.Database
	mu          sync.RWMutex
	logger      utils.Logger
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(logger utils.Logger) *ConnectionHandler {
	return &ConnectionHandler{
		connections: make(map[string]database.Database),
		logger:      logger,
	}
}

// ConnectRequest represents a database connection request
type ConnectRequest struct {
	URI    string `json:"uri"`
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

// Connect handles database connection requests
func (h *ConnectionHandler) Connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only POST method is allowed"))
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("INVALID_REQUEST", "Invalid request body", err.Error()))
		return
	}

	if req.URI == "" {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("MISSING_URI", "Database URI is required"))
		return
	}

	if req.Name == "" {
		req.Name = "default"
	}

	// Check if connection already exists
	h.mu.RLock()
	if _, exists := h.connections[req.Name]; exists {
		h.mu.RUnlock()
		writeJSON(w, http.StatusConflict, types.NewErrorResponse("CONNECTION_EXISTS", "Connection with this name already exists"))
		return
	}
	h.mu.RUnlock()

	// Create new database connection
	db, err := database.NewFromURI(req.URI)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("CONNECTION_FAILED", "Failed to create database connection", err.Error()))
		return
	}

	// Set logger
	db.SetLogger(h.logger)

	// Connect to database
	ctx := r.Context()
	if err := db.Connect(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("CONNECTION_FAILED", "Failed to connect to database", err.Error()))
		return
	}

	// Load schema if provided
	if req.Schema != "" {
		if err := db.LoadSchema(ctx, req.Schema); err != nil {
			db.Close()
			writeJSON(w, http.StatusBadRequest, types.NewErrorResponse("SCHEMA_LOAD_FAILED", "Failed to load schema", err.Error()))
			return
		}

		// Sync schemas
		if err := db.SyncSchemas(ctx); err != nil {
			db.Close()
			writeJSON(w, http.StatusInternalServerError, types.NewErrorResponse("SCHEMA_SYNC_FAILED", "Failed to sync schemas", err.Error()))
			return
		}
	}

	// Store connection
	h.mu.Lock()
	h.connections[req.Name] = db
	h.mu.Unlock()

	response := types.NewSuccessResponse(map[string]string{
		"name":   req.Name,
		"driver": db.GetDriverType(),
		"status": "connected",
	})

	writeJSON(w, http.StatusOK, response)
}

// Disconnect handles database disconnection requests
func (h *ConnectionHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only DELETE method is allowed"))
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		name = "default"
	}

	h.mu.Lock()
	db, exists := h.connections[name]
	if !exists {
		h.mu.Unlock()
		writeJSON(w, http.StatusNotFound, types.NewErrorResponse("CONNECTION_NOT_FOUND", "Connection not found"))
		return
	}

	delete(h.connections, name)
	h.mu.Unlock()

	// Close database connection
	if err := db.Close(); err != nil {
		h.logger.Error("Failed to close database connection: %v", err)
	}

	response := types.NewSuccessResponse(map[string]string{
		"name":   name,
		"status": "disconnected",
	})

	writeJSON(w, http.StatusOK, response)
}

// List returns all active connections
func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, types.NewErrorResponse("METHOD_NOT_ALLOWED", "Only GET method is allowed"))
		return
	}

	h.mu.RLock()
	connections := make([]map[string]string, 0, len(h.connections))
	for name, db := range h.connections {
		connections = append(connections, map[string]string{
			"name":   name,
			"driver": db.GetDriverType(),
			"status": "connected",
		})
	}
	h.mu.RUnlock()

	writeJSON(w, http.StatusOK, types.NewSuccessResponse(connections))
}

// GetConnection returns a database connection by name
func (h *ConnectionHandler) GetConnection(name string) (database.Database, error) {
	if name == "" {
		name = "default"
	}

	h.mu.RLock()
	db, exists := h.connections[name]
	h.mu.RUnlock()

	if !exists {
		return nil, errors.New("no database connection")
	}

	return db, nil
}

// AddConnection adds a database connection with a given name
func (h *ConnectionHandler) AddConnection(name string, db database.Database) {
	h.mu.Lock()
	h.connections[name] = db
	h.mu.Unlock()
}

// writeJSON writes JSON response
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
