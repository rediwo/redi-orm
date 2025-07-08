package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/utils"
)

// Server represents a REST API server
type Server struct {
	db     database.Database
	Router *Router // Exported for testing
	port   int
	logger utils.Logger
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Database   database.Database
	Port       int
	LogLevel   string
	SchemaFile string
}

// NewServer creates a new REST API server
func NewServer(config ServerConfig) (*Server, error) {
	// Set up logger
	logLevel := utils.LogLevelInfo
	if config.LogLevel != "" {
		switch config.LogLevel {
		case "debug":
			logLevel = utils.LogLevelDebug
		case "info":
			logLevel = utils.LogLevelInfo
		case "warn":
			logLevel = utils.LogLevelWarn
		case "error":
			logLevel = utils.LogLevelError
		case "none":
			logLevel = utils.LogLevelNone
		}
	}

	logger := utils.NewDefaultLogger("REST")
	logger.SetLevel(logLevel)

	// Set logger on database if provided
	if config.Database != nil {
		config.Database.SetLogger(logger)

		// Only connect if not already connected
		ctx := context.Background()
		if err := config.Database.Ping(ctx); err != nil {
			if err := config.Database.Connect(ctx); err != nil {
				return nil, fmt.Errorf("failed to connect to database: %w", err)
			}
		}
	}

	// Load schema if provided
	if config.SchemaFile != "" && config.Database != nil {
		ctx := context.Background()
		if err := config.Database.LoadSchemaFrom(ctx, config.SchemaFile); err != nil {
			return nil, fmt.Errorf("failed to load schema: %w", err)
		}

		// Sync schemas
		if err := config.Database.SyncSchemas(ctx); err != nil {
			return nil, fmt.Errorf("failed to sync schemas: %w", err)
		}
	}

	// Create router
	router := NewRouter(logger)

	// Add default database connection if provided
	if config.Database != nil {
		router.connHandler.AddConnection("default", config.Database)
	}

	// Set default port if not provided
	if config.Port == 0 {
		config.Port = 8080
	}

	return &Server{
		db:     config.Database,
		Router: router,
		port:   config.Port,
		logger: logger,
	}, nil
}

// Start starts the REST API server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.logger.Info("Starting REST API server on http://localhost%s", addr)
	s.logger.Info("Available endpoints:")
	s.logger.Info("  - POST   /api/connections/connect")
	s.logger.Info("  - DELETE /api/connections/disconnect")
	s.logger.Info("  - GET    /api/connections")
	s.logger.Info("  - GET    /api/{model}")
	s.logger.Info("  - GET    /api/{model}/{id}")
	s.logger.Info("  - POST   /api/{model}")
	s.logger.Info("  - PUT    /api/{model}/{id}")
	s.logger.Info("  - DELETE /api/{model}/{id}")
	s.logger.Info("  - POST   /api/{model}/batch")

	// Default database is already added in NewServer

	return http.ListenAndServe(addr, s.Router)
}

// Stop stops the server (cleanup)
func (s *Server) Stop() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
