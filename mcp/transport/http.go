package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rediwo/redi-orm/utils"
)

// HTTPTransport implements Transport interface for HTTP/SSE communication
type HTTPTransport struct {
	port            int
	server          *http.Server
	logger          utils.Logger
	mu              sync.RWMutex
	clients         map[string]*Client
	httpHandler     http.Handler
	mcpHandler      MCPHandler
	securityHandler SecurityHandler
	closed          bool
}

// MCPHandler interface for handling MCP requests
type MCPHandler interface {
	Handle(ctx context.Context, message json.RawMessage) json.RawMessage
}

// SecurityHandler interface for security middleware
type SecurityHandler interface {
	SecurityMiddleware(next http.Handler) http.Handler
}

// Client represents a connected SSE client
type Client struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	ctx      context.Context
	cancel   context.CancelFunc
	sendCh   chan json.RawMessage
	closeCh  chan struct{}
	lastSeen time.Time
}

// NewHTTPTransport creates a new HTTP/SSE transport
func NewHTTPTransport(port int, logger utils.Logger) *HTTPTransport {
	if port == 0 {
		port = 3000 // Default port
	}
	return &HTTPTransport{
		port:    port,
		logger:  logger,
		clients: make(map[string]*Client),
	}
}

// Start initializes the HTTP server
func (t *HTTPTransport) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	// Create HTTP multiplexer
	mux := http.NewServeMux()
	
	// Handle SSE connections
	mux.HandleFunc("/events", t.handleSSE)
	
	// Handle JSON-RPC over HTTP POST
	mux.HandleFunc("/", t.handleHTTP)
	
	// Apply security middleware if available
	var handler http.Handler = mux
	if t.securityHandler != nil {
		handler = t.securityHandler.SecurityMiddleware(handler)
	}
	
	// Add CORS middleware
	t.httpHandler = t.corsMiddleware(handler)

	// Create HTTP server
	t.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", t.port),
		Handler: t.httpHandler,
	}

	// Start server in background
	go func() {
		t.logger.Info("Starting HTTP transport on port %d", t.port)
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.logger.Error("HTTP server error: %v", err)
		}
	}()

	// Start client cleanup routine
	go t.cleanupClients()

	return nil
}

// Stop shuts down the HTTP server
func (t *HTTPTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true

	// Close all clients
	for _, client := range t.clients {
		t.closeClient(client)
	}
	t.clients = make(map[string]*Client)

	// Shutdown HTTP server
	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := t.server.Shutdown(ctx); err != nil {
			t.logger.Error("HTTP server shutdown error: %v", err)
			return err
		}
	}

	t.logger.Debug("HTTP transport stopped")
	return nil
}

// Send writes a message to all connected clients (SSE only)
func (t *HTTPTransport) Send(message json.RawMessage) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.closed {
		return fmt.Errorf("transport is closed")
	}

	// Send to all connected SSE clients
	for _, client := range t.clients {
		select {
		case client.sendCh <- message:
			client.lastSeen = time.Now()
		case <-time.After(1 * time.Second):
			t.logger.Warn("Client %s send timeout, removing", client.id)
			t.removeClient(client.id)
		}
	}

	t.logger.Debug("Sent message to %d clients", len(t.clients))
	return nil
}

// Receive is not applicable for HTTP transport (client-initiated)
func (t *HTTPTransport) Receive() (json.RawMessage, error) {
	return nil, fmt.Errorf("receive not supported for HTTP transport")
}

// handleSSE handles Server-Sent Events connections
func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create client
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(r.Context())
	
	client := &Client{
		id:       clientID,
		writer:   w,
		flusher:  flusher,
		ctx:      ctx,
		cancel:   cancel,
		sendCh:   make(chan json.RawMessage, 100),
		closeCh:  make(chan struct{}),
		lastSeen: time.Now(),
	}

	// Register client
	t.mu.Lock()
	t.clients[clientID] = client
	t.mu.Unlock()

	t.logger.Debug("SSE client connected: %s", clientID)

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"clientId\":\"%s\"}\n\n", clientID)
	flusher.Flush()

	// Handle client messages
	go t.handleSSEClient(client)

	// Wait for client to disconnect
	<-client.closeCh
	
	// Cleanup
	t.mu.Lock()
	delete(t.clients, clientID)
	t.mu.Unlock()
	
	t.logger.Debug("SSE client disconnected: %s", clientID)
}

// handleSSEClient handles messages for a specific SSE client
func (t *HTTPTransport) handleSSEClient(client *Client) {
	defer close(client.closeCh)
	defer client.cancel()

	for {
		select {
		case message := <-client.sendCh:
			// Send message as SSE event
			_, err := fmt.Fprintf(client.writer, "data: %s\n\n", string(message))
			if err != nil {
				t.logger.Debug("Failed to send to client %s: %v", client.id, err)
				return
			}
			client.flusher.Flush()
			client.lastSeen = time.Now()
			
		case <-client.ctx.Done():
			return
			
		case <-time.After(30 * time.Second):
			// Send keepalive
			_, err := fmt.Fprintf(client.writer, ": keepalive\n\n")
			if err != nil {
				t.logger.Debug("Keepalive failed for client %s: %v", client.id, err)
				return
			}
			client.flusher.Flush()
		}
	}
}

// handleHTTP handles regular HTTP POST requests for JSON-RPC
func (t *HTTPTransport) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate JSON
	if !json.Valid(body) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Check if MCP handler is available
	t.mu.RLock()
	handler := t.mcpHandler
	t.mu.RUnlock()

	if handler == nil {
		// No handler available
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"error": map[string]interface{}{
				"code":    -32603,
				"message": "Server not ready",
			},
			"id": nil,
		}
		responseJSON, _ := json.Marshal(response)
		w.Write(responseJSON)
		return
	}

	// Process JSON-RPC request through MCP handler
	response := handler.Handle(r.Context(), json.RawMessage(body))
	w.Write(response)

	t.logger.Debug("Handled HTTP request from %s", r.RemoteAddr)
}

// corsMiddleware adds CORS headers
func (t *HTTPTransport) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// cleanupClients removes stale clients
func (t *HTTPTransport) cleanupClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for id, client := range t.clients {
				if now.Sub(client.lastSeen) > 2*time.Minute {
					t.logger.Debug("Removing stale client: %s", id)
					t.closeClient(client)
					delete(t.clients, id)
				}
			}
			t.mu.Unlock()
		}
	}
}

// closeClient safely closes a client connection
func (t *HTTPTransport) closeClient(client *Client) {
	client.cancel()
	close(client.sendCh)
}

// removeClient removes a client (must be called with lock held)
func (t *HTTPTransport) removeClient(clientID string) {
	if client, exists := t.clients[clientID]; exists {
		t.closeClient(client)
		delete(t.clients, clientID)
	}
}

// SetHandler sets the MCP handler for processing requests
func (t *HTTPTransport) SetHandler(handler MCPHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.mcpHandler = handler
}

// SetSecurityHandler sets the security handler for middleware
func (t *HTTPTransport) SetSecurityHandler(handler SecurityHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.securityHandler = handler
	
	// Rebuild handler chain if server is already running
	if t.server != nil {
		t.rebuildHandlerChain()
	}
}

// rebuildHandlerChain rebuilds the HTTP handler chain with security middleware
func (t *HTTPTransport) rebuildHandlerChain() {
	// Create HTTP multiplexer
	mux := http.NewServeMux()
	
	// Handle SSE connections
	mux.HandleFunc("/events", t.handleSSE)
	
	// Handle JSON-RPC over HTTP POST
	mux.HandleFunc("/", t.handleHTTP)
	
	// Apply security middleware if available
	var handler http.Handler = mux
	if t.securityHandler != nil {
		handler = t.securityHandler.SecurityMiddleware(handler)
	}
	
	// Add CORS middleware
	t.httpHandler = t.corsMiddleware(handler)
	
	// Update server handler
	if t.server != nil {
		t.server.Handler = t.httpHandler
	}
}

// GetClientCount returns the number of connected clients
func (t *HTTPTransport) GetClientCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.clients)
}