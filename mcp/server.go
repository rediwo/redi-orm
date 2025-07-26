package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/schema/generator"
)

// ServerConfig holds the configuration for the MCP server
type ServerConfig struct {
	DatabaseURI string
	SchemaPath  string
	Transport   string
	Port        int
	LogLevel    string
	ReadOnly    bool
	Security    SecurityConfig
	Version     string // Version of the MCP server
}

// SDKServer wraps the MCP SDK server
type SDKServer struct {
	mcpServer            *mcp.Server
	config               ServerConfig
	db                   database.Database
	schemas              []*schema.Schema
	logger               logger.Logger
	security             *SecurityManager
	persistence          *generator.SchemaPersistence
	pendingSchemaManager *PendingSchemaManager
}

// NewSDKServer creates a new MCP server using the official SDK
func NewSDKServer(config ServerConfig) (*SDKServer, error) {
	// Create logger
	l := logger.NewDefaultLogger("MCP")
	l.SetLevel(logger.ParseLogLevel(config.LogLevel))

	// For stdio transport, output logs to stderr to avoid polluting JSON-RPC stream
	if config.Transport == "stdio" {
		l.SetOutput(os.Stderr)
	}

	// Create database logger with same configuration
	dbLogger := logger.NewDefaultLogger("RediORM")
	dbLogger.SetLevel(logger.ParseLogLevel(config.LogLevel))
	if config.Transport == "stdio" {
		dbLogger.SetOutput(os.Stderr)
	}

	// Create database instance
	db, err := database.NewFromURI(config.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Connect to database first
	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	db.SetLogger(dbLogger)

	// Get migrator after connection for schema persistence
	migrator := db.GetMigrator()
	if migrator == nil {
		return nil, fmt.Errorf("database does not support migrations")
	}

	// Create schema persistence manager with migrator
	persistence := generator.NewSchemaPersistence(config.SchemaPath, l, migrator)

	// Load schemas
	schemas, err := persistence.LoadSchemas()
	if err != nil {
		return nil, fmt.Errorf("failed to load schemas: %w", err)
	}

	// Register schemas with the database
	for _, s := range schemas {
		if err := db.RegisterSchema(s.Name, s); err != nil {
			return nil, fmt.Errorf("failed to register schema %s: %w", s.Name, err)
		}
	}
	l.Debug("Registered %d schemas with database", len(schemas))

	// Reverse engineer schemas from existing database tables
	l.Info("Checking for existing database tables to generate schemas...")

	// Track which models we already have schemas for
	existingModels := make(map[string]bool)
	for _, s := range schemas {
		existingModels[s.Name] = true
		existingModels[s.TableName] = true
	}

	// Generate schemas with relations
	l.Info("Generating schemas from existing database tables...")
	generatedSchemas, err := generator.GenerateSchemasFromTablesWithRelations(migrator)
	if err != nil {
		l.Warn("Failed to generate schemas from tables: %v", err)
	} else {
		generatedCount := 0
		for _, generatedSchema := range generatedSchemas {
			// Check if we already have a schema for this model
			if existingModels[generatedSchema.Name] || existingModels[generatedSchema.TableName] {
				continue
			}

			// Register the generated schema
			if err := db.RegisterSchema(generatedSchema.Name, generatedSchema); err != nil {
				l.Warn("Failed to register generated schema for model %s: %v", generatedSchema.Name, err)
				continue
			}

			// Add to our schemas list
			schemas = append(schemas, generatedSchema)
			generatedCount++

			l.Debug("Generated schema for table %s as model %s", generatedSchema.TableName, generatedSchema.Name)

			// Log relations if any
			if len(generatedSchema.Relations) > 0 {
				for relName, rel := range generatedSchema.Relations {
					l.Debug("  - Relation %s: %s to %s", relName, rel.Type, rel.Model)
				}
			}

			// Save the generated schema to file
			if err := persistence.SaveSchema(generatedSchema); err != nil {
				l.Warn("Failed to save generated schema for model %s to file: %v", generatedSchema.Name, err)
			} else {
				l.Debug("Saved generated schema for model %s to file", generatedSchema.Name)
			}
		}

		if generatedCount > 0 {
			l.Info("Generated %d schemas from existing database tables with relations", generatedCount)
			l.Info("Generated schemas have been saved to schema files")
		}
	}

	// Create security manager
	security := NewSecurityManager(config.Security)

	// Create server instance first (without mcpServer)
	server := &SDKServer{
		config:      config,
		db:          db,
		schemas:     schemas,
		logger:      l,
		security:    security,
		persistence: persistence,
	}

	// Create MCP server with comprehensive logging handlers
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "redi-mcp",
		Version: config.Version,
	}, &mcp.ServerOptions{
		Instructions: "RediORM MCP Server - A schema-driven ORM with Prisma-style query interface",
		PageSize:     100,
		InitializedHandler: func(ctx context.Context, session *mcp.ServerSession, params *mcp.InitializedParams) {
			sessionID := session.ID()
			if sessionID != "" {
				l.Info("Client initialized session: %s", sessionID)
			} else {
				l.Info("Client initialized (no session ID)")
			}
			// InitializedParams might not have ClientInfo in this version
		},
		RootsListChangedHandler: func(ctx context.Context, session *mcp.ServerSession, params *mcp.RootsListChangedParams) {
			sessionID := session.ID()
			if sessionID != "" {
				l.Info("Client roots changed for session: %s", sessionID)
			} else {
				l.Info("Client roots changed")
			}
		},
		ProgressNotificationHandler: func(ctx context.Context, session *mcp.ServerSession, params *mcp.ProgressNotificationParams) {
			// Progress is likely a float64, not a pointer
			sessionID := session.ID()
			if sessionID != "" {
				l.Debug("Progress notification from session %s: token=%v (%.2f%%)",
					sessionID, params.ProgressToken, params.Progress*100)
			} else {
				l.Debug("Progress notification: token=%v (%.2f%%)",
					params.ProgressToken, params.Progress*100)
			}
		},
		CompletionHandler: func(ctx context.Context, session *mcp.ServerSession, params *mcp.CompleteParams) (*mcp.CompleteResult, error) {
			sessionID := session.ID()
			if sessionID != "" {
				l.Debug("Completion request from session %s", sessionID)
			} else {
				l.Debug("Completion request")
			}
			// Return nil to indicate completion is not supported yet
			return nil, fmt.Errorf("completion not implemented")
		},
		// Note: Subscribe/Unsubscribe handlers may not be available in this SDK version
		// KeepAlive disabled - some clients may not handle it correctly
		// KeepAlive: 30 * time.Second,
	})

	// Set the mcpServer
	server.mcpServer = mcpServer

	// Register all tools
	server.registerTools()
	server.registerSchemaModificationTools()

	// Add receiving middleware to log all incoming method calls
	receivingMiddleware := func(next mcp.MethodHandler[*mcp.ServerSession]) mcp.MethodHandler[*mcp.ServerSession] {
		return func(ctx context.Context, session *mcp.ServerSession, method string, params mcp.Params) (mcp.Result, error) {
			// Log incoming method calls at debug level
			if session != nil {
				sessionID := session.ID()
				if sessionID != "" {
					l.Debug("Received method '%s' from session %s", method, sessionID)
				} else {
					l.Debug("Received method '%s' (no session ID)", method)
				}
			} else {
				l.Debug("Received method '%s' (no session)", method)
			}

			// Call the actual handler
			result, err := next(ctx, session, method, params)

			// Log errors
			if err != nil {
				l.Error("Method '%s' failed: %v", method, err)
			}

			return result, err
		}
	}

	// Add sending middleware to log all outgoing responses
	sendingMiddleware := func(next mcp.MethodHandler[*mcp.ServerSession]) mcp.MethodHandler[*mcp.ServerSession] {
		return func(ctx context.Context, session *mcp.ServerSession, method string, params mcp.Params) (mcp.Result, error) {
			// Call the actual handler
			result, err := next(ctx, session, method, params)

			// Log outgoing response
			if err != nil {
				l.Debug("Sending error response for method '%s': %v", method, err)
			} else if result != nil {
				l.Debug("Sending success response for method '%s'", method)
			}

			return result, err
		}
	}

	// Add middleware to the server
	mcpServer.AddReceivingMiddleware(receivingMiddleware)
	mcpServer.AddSendingMiddleware(sendingMiddleware)

	return server, nil
}

// GetLogger returns the server's logger
func (s *SDKServer) GetLogger() logger.Logger {
	return s.logger
}

// Start starts the MCP server with the configured transport
func (s *SDKServer) Start() error {
	s.logger.Info("Starting MCP server with transport: %s", s.config.Transport)

	ctx := context.Background()

	switch s.config.Transport {
	case "stdio":
		return s.startStdioServer(ctx)

	case "http":
		return s.startHTTPServer(ctx)

	default:
		return fmt.Errorf("unsupported transport: %s", s.config.Transport)
	}
}

// startStdioServer starts the MCP server with stdio transport
func (s *SDKServer) startStdioServer(ctx context.Context) error {
	// Create stdio transport
	var transport mcp.Transport
	transport = mcp.NewStdioTransport()

	// Add logging transport when debug level is enabled
	if s.logger.GetLevel() >= logger.LogLevelDebug {
		s.logger.Debug("Enabling MCP SDK debug logging for stdio transport")
		logWriter := NewLoggerWriter(s.logger, "MCP")
		transport = mcp.NewLoggingTransport(transport, logWriter)
	}

	// Log server lifecycle
	s.logger.Info("MCP stdio server starting...")

	// Run the MCP server with stdio transport
	err := s.mcpServer.Run(ctx, transport)

	if err != nil {
		s.logger.Error("MCP stdio server stopped with error: %v", err)
	} else {
		s.logger.Info("MCP stdio server stopped gracefully")
	}

	return err
}

// startHTTPServer starts an HTTP server with both SSE and Streamable transport support
func (s *SDKServer) startHTTPServer(ctx context.Context) error {
	// Create handlers for both transport types

	// SSE handler for the original transport
	sseHandler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		s.logger.Info("New SSE connection from %s", r.RemoteAddr)
		return s.mcpServer
	})

	// Streamable handler for the newer transport
	streamableHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		s.logger.Info("New streamable connection from %s", r.RemoteAddr)
		return s.mcpServer
	}, nil)

	// Create a router with intelligent transport detection
	mux := http.NewServeMux()

	// Main handler that routes based on request characteristics
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is an SSE session POST (has sessionid query parameter)
		if r.Method == http.MethodPost && r.URL.Query().Get("sessionid") != "" {
			s.logger.Debug("Routing POST with sessionid to SSE handler")
			sseHandler.ServeHTTP(w, r)
			return
		}

		// Check for streamable transport indicators
		sessionHeader := r.Header.Get("Mcp-Session-Id")
		acceptHeader := r.Header.Get("Accept")

		// Route to streamable if:
		// 1. Has Mcp-Session-Id header (definitive streamable indicator)
		// 2. POST with both application/json and text/event-stream in Accept
		if sessionHeader != "" ||
			(r.Method == http.MethodPost &&
				strings.Contains(acceptHeader, "application/json") &&
				strings.Contains(acceptHeader, "text/event-stream")) {
			s.logger.Debug("Routing to streamable handler")
			streamableHandler.ServeHTTP(w, r)
			return
		}

		// Default to streamable for GET requests without sessionid (new sessions)
		if r.Method == http.MethodGet && r.URL.Query().Get("sessionid") == "" {
			s.logger.Debug("Routing new GET session to streamable handler")
			streamableHandler.ServeHTTP(w, r)
			return
		}

		// Route GET with sessionid to SSE
		if r.Method == http.MethodGet && r.URL.Query().Get("sessionid") != "" {
			s.logger.Debug("Routing GET with sessionid to SSE handler")
			sseHandler.ServeHTTP(w, r)
			return
		}

		// Default fallback to streamable
		s.logger.Debug("Default routing to streamable handler")
		streamableHandler.ServeHTTP(w, r)
	})

	// Root path uses intelligent routing
	mux.Handle("/", mainHandler)

	// Dedicated SSE path for explicit SSE connections
	mux.Handle("/sse", sseHandler)

	// Add a helper endpoint that shows available transports
	mux.HandleFunc("/transports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"transports": {
				"/": "intelligent routing - supports both streamable and SSE transports",
				"/sse": "dedicated SSE endpoint - uses sessionid query parameter",
				"routing": {
					"streamable": "Mcp-Session-Id header or POST with application/json+text/event-stream",
					"sse": "sessionid query parameter in URL"
				}
			}
		}`))
	})

	// Build middleware chain
	var handler http.Handler = mux

	// Add CORS support
	handler = s.corsMiddleware(handler)

	// Add security middleware (authentication, rate limiting) if configured
	if s.config.Security.EnableAuth || s.config.Security.EnableRateLimit {
		handler = s.security.SecurityMiddleware(handler)
	}

	addr := fmt.Sprintf(":%d", s.config.Port)
	s.logger.Info("Starting HTTP server on %s", addr)
	s.logger.Info("Transport endpoints:")
	s.logger.Info("  - Intelligent routing: http://localhost%s/ (supports both streamable and SSE)", addr)
	s.logger.Info("  - Dedicated SSE: http://localhost%s/sse", addr)
	s.logger.Info("  - Transport info: http://localhost%s/transports", addr)

	return http.ListenAndServe(addr, handler)
}

// corsMiddleware adds CORS headers
func (s *SDKServer) corsMiddleware(next http.Handler) http.Handler {
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
