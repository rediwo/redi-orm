package orm

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Global ORM instance
var globalORM *ORM

type ORM struct {
	schemas  map[string]*schema.Schema
	database database.Database
	migrator *migration.Manager
}

// ORMOptions contains options for ORM initialization
type ORMOptions struct {
	AutoMigrate bool
	DryRun      bool
	Force       bool
}

// InitializeORM scans the schema directory for *.prisma files and initializes the ORM
func InitializeORM(schemaDir string, options ...ORMOptions) error {
	// Parse options
	opts := ORMOptions{AutoMigrate: true} // Default to auto-migrate
	if len(options) > 0 {
		opts = options[0]
	}
	orm := &ORM{
		schemas: make(map[string]*schema.Schema),
	}

	var datasourceInfo *prisma.DatasourceStatement

	// Scan for .prisma files
	err := filepath.Walk(schemaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".prisma") {
			return nil
		}

		fmt.Printf("Processing schema file: %s\n", path)

		// Read the schema file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", path, err)
		}

		// Parse the schema
		lexer := prisma.NewLexer(string(content))
		parser := prisma.NewParser(lexer)
		ast := parser.ParseSchema()
		if len(parser.Errors()) > 0 {
			return fmt.Errorf("failed to parse schema file %s: %v", path, parser.Errors())
		}

		// Extract datasource if found
		for _, stmt := range ast.Statements {
			if ds, ok := stmt.(*prisma.DatasourceStatement); ok {
				datasourceInfo = ds
				fmt.Printf("Found datasource: %s\n", ds.Name)
			}
		}

		// Convert AST to schema and create models
		converter := prisma.NewConverter()
		schemas, err := converter.Convert(ast)
		if err != nil {
			return fmt.Errorf("failed to convert schema file %s: %w", path, err)
		}

		// Add schemas to ORM
		for _, s := range schemas {
			orm.schemas[s.Name] = s
			fmt.Printf("Registered model: %s\n", s.Name)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan schema directory: %w", err)
	}

	if len(orm.schemas) == 0 {
		return fmt.Errorf("no models found in schema directory")
	}

	// Initialize database connection if datasource is found
	if datasourceInfo != nil {
		// Extract provider and URL from properties
		provider := ""
		dbURL := ""

		for _, prop := range datasourceInfo.Properties {
			if prop.Name == "provider" {
				// Extract string value from expression
				if str, ok := prop.Value.(*prisma.StringLiteral); ok {
					provider = str.Value
				}
			} else if prop.Name == "url" {
				// Check if it's a function call (env()) or a string literal
				if fn, ok := prop.Value.(*prisma.FunctionCall); ok && fn.Name == "env" {
					// Extract environment variable name from env() function
					if len(fn.Args) > 0 {
						if str, ok := fn.Args[0].(*prisma.StringLiteral); ok {
							envVar := str.Value
							dbURL = os.Getenv(envVar)
							if dbURL == "" {
								return fmt.Errorf("environment variable %s not set", envVar)
							}
						}
					}
				} else if str, ok := prop.Value.(*prisma.StringLiteral); ok {
					dbURL = str.Value
				}
			}
		}

		fmt.Printf("Database Provider: %s\n", provider)
		fmt.Printf("Database URL: %s\n", dbURL)

		// Connect to database
		db, err := database.NewFromURI(dbURL)
		if err != nil {
			return fmt.Errorf("failed to create database connection: %w", err)
		}

		if err := db.Connect(context.Background()); err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}

		orm.database = db
		fmt.Printf("Connected to %s database successfully\n", provider)

		// Initialize migration manager
		migrationOpts := migration.MigrationOptions{
			AutoMigrate: opts.AutoMigrate,
			DryRun:      opts.DryRun,
			Force:       opts.Force,
		}

		migrator, err := migration.NewManager(db, migrationOpts)
		if err != nil {
			return fmt.Errorf("failed to create migration manager: %w", err)
		}
		orm.migrator = migrator

		// Perform migration if auto-migrate is enabled
		if opts.AutoMigrate {
			fmt.Printf("Starting automatic migration...\n")
			if err := orm.migrator.Migrate(orm.schemas); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
		}
	}

	// Set global ORM instance
	globalORM = orm

	return nil
}

// GetSchemas returns all registered schemas
func GetSchemas() map[string]*schema.Schema {
	if globalORM == nil {
		return nil
	}
	return globalORM.schemas
}

// GetSchema returns a specific schema by name
func GetSchema(name string) *schema.Schema {
	if globalORM == nil {
		return nil
	}
	return globalORM.schemas[name]
}

// GetDatabase returns the database instance
func GetDatabase() types.Database {
	if globalORM == nil {
		return nil
	}
	return globalORM.database
}

// GetMigrator returns the migration manager
func GetMigrator() *migration.Manager {
	if globalORM == nil {
		return nil
	}
	return globalORM.migrator
}

// extractSQLDB extracts the underlying sql.DB from a Database interface
func extractSQLDB(db types.Database) (*sql.DB, error) {
	// This is a bit hacky, but we need to access the underlying sql.DB
	// for the migration manager. In a real implementation, we might want
	// to add this method to the Database interface.

	switch d := db.(type) {
	case interface{ GetDB() *sql.DB }:
		return d.GetDB(), nil
	default:
		// Try to use reflection or add a method to access sql.DB
		return nil, fmt.Errorf("cannot extract sql.DB from database type %T", db)
	}
}
