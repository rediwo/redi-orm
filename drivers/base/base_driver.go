package base

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Driver provides common functionality for all database drivers
type Driver struct {
	DB          *sql.DB
	Config      types.Config
	FieldMapper types.FieldMapper
	Schemas     map[string]*schema.Schema
	SchemasMu   sync.RWMutex
}

// NewDriver creates a new base driver instance
func NewDriver(config types.Config) *Driver {
	return &Driver{
		Config:      config,
		FieldMapper: types.NewDefaultFieldMapper(),
		Schemas:     make(map[string]*schema.Schema),
	}
}

// SetDB sets the database connection
func (b *Driver) SetDB(db *sql.DB) {
	b.DB = db
}

// GetDB returns the database connection
func (b *Driver) GetDB() *sql.DB {
	return b.DB
}

// RegisterSchema registers a schema with the database
func (b *Driver) RegisterSchema(modelName string, schema *schema.Schema) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if err := schema.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	b.SchemasMu.Lock()
	defer b.SchemasMu.Unlock()

	b.Schemas[modelName] = schema

	// Register with field mapper
	if mapper, ok := b.FieldMapper.(*types.DefaultFieldMapper); ok {
		mapper.RegisterSchema(modelName, schema)
	}

	return nil
}

// GetSchema returns a registered schema
func (b *Driver) GetSchema(modelName string) (*schema.Schema, error) {
	b.SchemasMu.RLock()
	defer b.SchemasMu.RUnlock()

	schema, exists := b.Schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("schema for model '%s' not registered", modelName)
	}
	return schema, nil
}

// GetModels returns all registered model names
func (b *Driver) GetModels() []string {
	b.SchemasMu.RLock()
	defer b.SchemasMu.RUnlock()

	models := make([]string, 0, len(b.Schemas))
	for modelName := range b.Schemas {
		models = append(models, modelName)
	}
	return models
}

// GetModelSchema returns schema for a model (alias for GetSchema)
func (b *Driver) GetModelSchema(modelName string) (*schema.Schema, error) {
	return b.GetSchema(modelName)
}

// LoadSchema loads schema from content string (accumulates schemas)
func (b *Driver) LoadSchema(ctx context.Context, schemaContent string) error {
	// Parse schema content
	schemas, err := prisma.ParseSchema(schemaContent)
	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Register all schemas
	for modelName, schema := range schemas {
		if err := b.RegisterSchema(modelName, schema); err != nil {
			return fmt.Errorf("failed to register schema for model %s: %w", modelName, err)
		}
	}

	return nil
}

// LoadSchemaFrom loads schema from file (accumulates schemas)
func (b *Driver) LoadSchemaFrom(ctx context.Context, filename string) error {
	// Parse schema file
	schemas, err := prisma.ParseSchemaFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse schema file: %w", err)
	}

	// Register all schemas
	for modelName, schema := range schemas {
		if err := b.RegisterSchema(modelName, schema); err != nil {
			return fmt.Errorf("failed to register schema for model %s: %w", modelName, err)
		}
	}

	return nil
}

// SyncSchemas synchronizes all loaded schemas with the database
// This must be called on the actual database driver implementation, not the base driver
func (b *Driver) SyncSchemas(ctx context.Context, db types.Database) error {
	b.SchemasMu.RLock()
	schemas := make(map[string]*schema.Schema)
	for k, v := range b.Schemas {
		schemas[k] = v
	}
	b.SchemasMu.RUnlock()

	if len(schemas) == 0 {
		return fmt.Errorf("no schemas loaded")
	}

	// Create migration manager and run migration
	migrationManager, err := migration.NewManager(db, migration.MigrationOptions{
		AutoMigrate: true,
		DryRun:      false,
		Force:       true,
	})
	if err != nil {
		return fmt.Errorf("failed to create migration manager: %w", err)
	}

	if err := migrationManager.Migrate(schemas); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// ResolveTableName resolves model name to table name
func (b *Driver) ResolveTableName(modelName string) (string, error) {
	return b.FieldMapper.ModelToTable(modelName)
}

// ResolveFieldName resolves schema field name to column name
func (b *Driver) ResolveFieldName(modelName, fieldName string) (string, error) {
	return b.FieldMapper.SchemaToColumn(modelName, fieldName)
}

// ResolveFieldNames resolves multiple schema field names to column names
func (b *Driver) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return b.FieldMapper.SchemaFieldsToColumns(modelName, fieldNames)
}

// GetFieldMapper returns the field mapper
func (b *Driver) GetFieldMapper() types.FieldMapper {
	return b.FieldMapper
}

// Ping checks if the database connection is alive
func (b *Driver) Ping(ctx context.Context) error {
	if b.DB == nil {
		return fmt.Errorf("database not connected")
	}
	return b.DB.PingContext(ctx)
}

// Close closes the database connection
func (b *Driver) Close() error {
	if b.DB != nil {
		return b.DB.Close()
	}
	return nil
}

// Exec executes a raw SQL statement
func (b *Driver) Exec(query string, args ...any) (sql.Result, error) {
	return b.DB.Exec(query, args...)
}

// Query executes a raw SQL query that returns rows
func (b *Driver) Query(query string, args ...any) (*sql.Rows, error) {
	return b.DB.Query(query, args...)
}

// QueryRow executes a raw SQL query that returns a single row
func (b *Driver) QueryRow(query string, args ...any) *sql.Row {
	return b.DB.QueryRow(query, args...)
}
