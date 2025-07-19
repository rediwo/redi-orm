package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/mcp/transport"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"

	// Import all drivers for CLI usage
	_ "github.com/rediwo/redi-orm/drivers/mongodb"
	_ "github.com/rediwo/redi-orm/drivers/mysql"
	_ "github.com/rediwo/redi-orm/drivers/postgresql"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
)

// NewServer creates a new MCP server instance
func NewServer(config ServerConfig) (*Server, error) {
	// Set defaults
	if config.MaxQueryRows == 0 {
		config.MaxQueryRows = 1000
	}
	if config.Transport == "" {
		config.Transport = "http"
	}

	// Create logger
	logger := utils.NewDefaultLogger("MCP")
	if config.LogLevel != "" {
		switch config.LogLevel {
		case "debug":
			logger.SetLevel(utils.LogLevelDebug)
		case "info":
			logger.SetLevel(utils.LogLevelInfo)
		case "warn":
			logger.SetLevel(utils.LogLevelWarn)
		case "error":
			logger.SetLevel(utils.LogLevelError)
		case "none":
			logger.SetLevel(utils.LogLevelNone)
		}
	}

	// Create server instance
	server := &Server{
		config:   config,
		logger:   logger,
		schemas:  make(map[string]*schema.Schema),
		security: NewSecurityManager(config.Security),
	}

	// Initialize database if URI provided
	if config.DatabaseURI != "" {
		db, err := database.NewFromURI(config.DatabaseURI)
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
		db.SetLogger(logger)
		
		if err := db.Connect(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		
		server.db = db
		logger.Info("Connected to database: %s", config.DatabaseURI)
	}

	// Load schema if provided
	if config.SchemaPath != "" {
		if err := server.loadSchemaFromFile(config.SchemaPath); err != nil {
			return nil, fmt.Errorf("failed to load schema: %w", err)
		}
		logger.Info("Loaded schema from: %s", config.SchemaPath)
	}

	// Create handler
	server.handler = NewHandler(server)

	// Create transport
	switch config.Transport {
	case "stdio":
		server.transport = transport.NewStdioTransport(server.logger)
	case "http":
		if config.Port == 0 {
			config.Port = 3000 // Default port for HTTP transport
		}
		httpTransport := transport.NewHTTPTransport(config.Port, server.logger)
		// Set security handler before starting
		httpTransport.SetSecurityHandler(server.security)
		server.transport = httpTransport
	default:
		return nil, fmt.Errorf("unknown transport: %s", config.Transport)
	}

	return server, nil
}

// Start starts the MCP server
func (s *Server) Start() error {
	s.logger.Info("Starting MCP server with transport: %s", s.config.Transport)

	// Start transport
	if err := s.transport.Start(); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	// Handle different transport types
	if s.config.Transport == "http" {
		// For HTTP transport, the server runs in background
		// We need to integrate the handler with the HTTP transport
		if httpTransport, ok := s.transport.(*transport.HTTPTransport); ok {
			httpTransport.SetHandler(s.handler)
		}
		
		// Block indefinitely for HTTP transport
		select {}
	} else {
		// Main message loop for stdio transport
		ctx := context.Background()
		for {
			// Receive message
			message, err := s.transport.Receive()
			if err != nil {
				// Check if it's a normal shutdown
				if err.Error() == "EOF" || err.Error() == "closed" {
					s.logger.Info("MCP server shutting down")
					break
				}
				s.logger.Error("Failed to receive message: %v", err)
				continue
			}

			// Handle message
			response := s.handler.Handle(ctx, message)

			// Send response
			if err := s.transport.Send(response); err != nil {
				s.logger.Error("Failed to send response: %v", err)
				continue
			}
		}
	}

	return s.Stop()
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.logger.Info("Stopping MCP server")

	if s.transport != nil {
		if err := s.transport.Stop(); err != nil {
			return fmt.Errorf("failed to stop transport: %w", err)
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}

	return nil
}

// loadSchemaFromFile loads a Prisma schema from file
func (s *Server) loadSchemaFromFile(path string) error {
	// Read schema file
	schemaBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}
	schemaContent := string(schemaBytes)

	// Parse Prisma schema
	schemas, err := prisma.ParseSchema(schemaContent)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Store the parsed schemas
	s.schemas = schemas

	// If database is connected, sync schemas
	if s.db != nil {
		// Register schemas to database
		for modelName, schema := range s.schemas {
			if err := s.db.RegisterSchema(modelName, schema); err != nil {
				return fmt.Errorf("failed to register schema %s: %w", modelName, err)
			}
		}

		// Sync with database
		if err := s.db.SyncSchemas(context.Background()); err != nil {
			return fmt.Errorf("failed to sync schemas: %w", err)
		}
	}

	return nil
}


// GetDatabase returns the database instance
func (s *Server) GetDatabase() types.Database {
	return s.db
}

// GetSchemas returns all loaded schemas
func (s *Server) GetSchemas() map[string]*schema.Schema {
	return s.schemas
}

// RegisterSchema registers a schema with the server
func (s *Server) RegisterSchema(name string, sch *schema.Schema) {
	s.schemas[name] = sch
}

// SetDatabase sets the database instance (for testing)
func (s *Server) SetDatabase(db types.Database) {
	s.db = db
}

