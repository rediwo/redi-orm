package graphql

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// Server represents a GraphQL server
type Server struct {
	db      types.Database
	schemas map[string]*schema.Schema
	handler *Handler
	port    int
	cors    bool
}

// ServerConfig contains configuration for the GraphQL server
type ServerConfig struct {
	DatabaseURI string
	SchemaPath  string
	Port        int
	Playground  bool
	CORS        bool
	LogLevel    string
}

// NewServer creates a new GraphQL server
func NewServer(config ServerConfig) (*Server, error) {
	// Create database connection
	db, err := database.NewFromURI(config.DatabaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Connect to database
	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set up database logger based on log level
	if config.LogLevel != "" {
		dbLogger := utils.NewDefaultLogger("RediORM")
		// Map GraphQL log level to utils log level
		switch config.LogLevel {
		case "debug", "DEBUG":
			dbLogger.SetLevel(utils.LogLevelDebug)
		case "info", "INFO":
			dbLogger.SetLevel(utils.LogLevelInfo)
		case "warn", "WARN", "warning", "WARNING":
			dbLogger.SetLevel(utils.LogLevelWarn)
		case "error", "ERROR":
			dbLogger.SetLevel(utils.LogLevelError)
		case "none", "NONE", "off", "OFF":
			dbLogger.SetLevel(utils.LogLevelNone)
		default:
			dbLogger.SetLevel(utils.LogLevelInfo)
		}
		db.SetLogger(dbLogger)
	}

	// Load schemas from file
	content, err := loadSchemaFile(config.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema file: %w", err)
	}

	// Parse schemas
	schemas, err := prisma.ParseSchema(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Register schemas with database
	for modelName, schema := range schemas {
		if err := db.RegisterSchema(modelName, schema); err != nil {
			return nil, fmt.Errorf("failed to register schema %s: %w", modelName, err)
		}
	}

	// Sync schemas with database
	if err := db.SyncSchemas(ctx); err != nil {
		return nil, fmt.Errorf("failed to sync schemas: %w", err)
	}

	// Generate GraphQL schema
	generator := NewSchemaGenerator(db, schemas)
	graphqlSchema, err := generator.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate GraphQL schema: %w", err)
	}

	// Create handler
	handler := NewHandler(graphqlSchema)
	if config.Playground {
		handler.EnablePlayground()
	}

	// Set log level
	if config.LogLevel != "" {
		logLevel := parseLogLevel(config.LogLevel)
		handler.SetLogLevel(logLevel)
	}

	return &Server{
		db:      db,
		schemas: schemas,
		handler: handler,
		port:    config.Port,
		cors:    config.CORS,
	}, nil
}

// Start starts the GraphQL server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// GraphQL endpoint
	mux.Handle("/graphql", s.corsMiddleware(s.handler))

	// Playground endpoint (same as GraphQL endpoint)
	mux.Handle("/playground", s.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect to /graphql which serves both API and playground
		http.Redirect(w, r, "/graphql", http.StatusMovedPermanently)
	})))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprintf(w, "RediORM GraphQL Server\n\nEndpoints:\n- /graphql - GraphQL API and Playground\n- /health - Health check\n")
			return
		}
		http.NotFound(w, r)
	})

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("🚀 GraphQL server ready at http://localhost%s/graphql", addr)
	log.Printf("🎮 GraphQL Playground available at http://localhost%s/graphql", addr)

	return http.ListenAndServe(addr, mux)
}

// Stop stops the GraphQL server
func (s *Server) Stop() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Handler returns the GraphQL handler
func (s *Server) Handler() http.Handler {
	return s.handler
}

// corsMiddleware adds CORS headers if enabled
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// loadSchemaFile loads schema content from a file
func loadSchemaFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// parseLogLevel parses a log level string
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug", "DEBUG":
		return LogLevelDebug
	case "info", "INFO":
		return LogLevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return LogLevelWarn
	case "error", "ERROR":
		return LogLevelError
	case "none", "NONE", "off", "OFF":
		return LogLevelNone
	default:
		return LogLevelInfo
	}
}
