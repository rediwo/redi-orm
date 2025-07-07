package base

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// Driver provides common functionality for all database drivers
type Driver struct {
	DB          *sql.DB
	URI         string
	DriverType  types.DriverType
	FieldMapper types.FieldMapper
	Schemas     map[string]*schema.Schema
	SchemasMu   sync.RWMutex
	Logger      any // Using any to avoid circular dependency
}

// NewDriver creates a new base driver instance
func NewDriver(uri string, driverType types.DriverType) *Driver {
	return &Driver{
		URI:         uri,
		DriverType:  driverType,
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
func (b *Driver) RegisterSchema(modelName string, sch *schema.Schema) error {
	if sch == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if err := sch.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	b.SchemasMu.Lock()
	defer b.SchemasMu.Unlock()

	b.Schemas[modelName] = sch

	// Register with field mapper
	// First try direct cast to DefaultFieldMapper
	if mapper, ok := b.FieldMapper.(*types.DefaultFieldMapper); ok {
		mapper.RegisterSchema(modelName, sch)
	} else {
		// For wrapped field mappers (like MongoDB), we need to register with the underlying mapper
		// MongoDB's field mapper embeds the FieldMapper interface, which might contain DefaultFieldMapper
		type schemaRegistrar interface {
			RegisterSchema(modelName string, s *schema.Schema)
		}
		if registrar, ok := b.FieldMapper.(schemaRegistrar); ok {
			registrar.RegisterSchema(modelName, sch)
		}
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

	// Get the migrator from the database
	migrator := db.GetMigrator()
	if migrator == nil {
		return fmt.Errorf("database does not support migrations")
	}

	// Get current tables
	currentTables, err := migrator.GetTables()
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	// Create a map for quick lookup
	currentTableMap := make(map[string]bool)
	for _, table := range currentTables {
		currentTableMap[table] = true
	}

	// Analyze dependencies and get sorted order
	sortedModels, err := AnalyzeSchemasDependencies(schemas)
	if err != nil {
		// Circular dependency detected, use deferred constraint creation
		return b.syncSchemasWithDeferredConstraints(ctx, db, schemas, currentTableMap)
	}

	// Process schemas in dependency order
	for _, modelName := range sortedModels {
		sch, exists := schemas[modelName]
		if !exists || sch.TableName == "" {
			continue
		}

		if !currentTableMap[sch.TableName] {
			// Table doesn't exist, create it
			sql, err := migrator.GenerateCreateTableSQL(sch)
			if err != nil {
				return fmt.Errorf("failed to generate CREATE TABLE SQL for %s: %w", sch.TableName, err)
			}

			if err := migrator.ApplyMigration(sql); err != nil {
				return fmt.Errorf("failed to create table %s: %w", sch.TableName, err)
			}

			// Create indexes for the new table
			for _, index := range sch.Indexes {
				// Convert field names to column names
				columnNames := make([]string, len(index.Fields))
				for i, fieldName := range index.Fields {
					if field := sch.GetFieldByName(fieldName); field != nil {
						columnNames[i] = field.GetColumnName()
					} else {
						columnNames[i] = fieldName // Fallback to field name
					}
				}

				indexSQL := migrator.GenerateCreateIndexSQL(sch.TableName, index.Name, columnNames, index.Unique)
				if err := migrator.ApplyMigration(indexSQL); err != nil {
					return fmt.Errorf("failed to create index %s on table %s: %w", index.Name, sch.TableName, err)
				}
			}
		} else {
			// Table exists, check for differences
			tableInfo, err := migrator.GetTableInfo(sch.TableName)
			if err != nil {
				return fmt.Errorf("failed to get table info for %s: %w", sch.TableName, err)
			}

			// Compare schema with existing table
			plan, err := migrator.CompareSchema(tableInfo, sch)
			if err != nil {
				return fmt.Errorf("failed to compare schema for %s: %w", sch.TableName, err)
			}

			// Generate and apply migration SQL
			if plan != nil && (len(plan.AddColumns) > 0 || len(plan.ModifyColumns) > 0 ||
				len(plan.DropColumns) > 0 || len(plan.AddIndexes) > 0 || len(plan.DropIndexes) > 0) {

				// Log migration plan details
				if logger, ok := b.Logger.(utils.Logger); ok && logger != nil {
					logger.Warn("Table '%s' needs migration:", sch.TableName)

					// Log column changes
					for _, change := range plan.AddColumns {
						logger.Warn("  - Adding column: %s", change.ColumnName)
					}
					for _, change := range plan.ModifyColumns {
						if change.OldColumn != nil && change.NewColumn != nil {
							var changes []string
							if change.OldColumn.Type != change.NewColumn.Type {
								changes = append(changes, fmt.Sprintf("type: %s -> %s", change.OldColumn.Type, change.NewColumn.Type))
							}
							if change.OldColumn.Nullable != change.NewColumn.Nullable {
								changes = append(changes, fmt.Sprintf("nullable: %v -> %v", change.OldColumn.Nullable, change.NewColumn.Nullable))
							}
							if change.OldColumn.Default != change.NewColumn.Default {
								changes = append(changes, fmt.Sprintf("default: %v -> %v", change.OldColumn.Default, change.NewColumn.Default))
							}
							logger.Warn("  - Modifying column '%s': %s", change.ColumnName, strings.Join(changes, ", "))
						}
					}
					for _, change := range plan.DropColumns {
						logger.Warn("  - Dropping column: %s", change.ColumnName)
					}

					// Log index changes
					for _, change := range plan.AddIndexes {
						logger.Warn("  - Adding index: %s", change.IndexName)
					}
					for _, change := range plan.DropIndexes {
						logger.Warn("  - Dropping index: %s", change.IndexName)
					}
				}

				sqlStatements, err := migrator.GenerateMigrationSQL(plan)
				if err != nil {
					return fmt.Errorf("failed to generate migration SQL for %s: %w", sch.TableName, err)
				}

				// Apply each SQL statement
				for _, sql := range sqlStatements {
					if err := migrator.ApplyMigration(sql); err != nil {
						return fmt.Errorf("failed to apply migration for %s: %w", sch.TableName, err)
					}
				}
			}
		}
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
	start := time.Now()
	result, err := b.DB.Exec(query, args...)
	duration := time.Since(start)

	if logger, ok := b.Logger.(utils.Logger); ok && logger != nil {
		logger.LogSQL(query, args, duration)
	}

	return result, err
}

// Query executes a raw SQL query that returns rows
func (b *Driver) Query(query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := b.DB.Query(query, args...)
	duration := time.Since(start)

	if logger, ok := b.Logger.(utils.Logger); ok && logger != nil {
		logger.LogSQL(query, args, duration)
	}

	return rows, err
}

// QueryRow executes a raw SQL query that returns a single row
func (b *Driver) QueryRow(query string, args ...any) *sql.Row {
	start := time.Now()
	row := b.DB.QueryRow(query, args...)
	duration := time.Since(start)

	if logger, ok := b.Logger.(utils.Logger); ok && logger != nil {
		logger.LogSQL(query, args, duration)
	}

	return row
}

// SetLogger sets the logger for the driver
func (b *Driver) SetLogger(logger any) {
	b.Logger = logger
}

// GetLogger returns the logger for the driver
func (b *Driver) GetLogger() any {
	return b.Logger
}

// syncSchemasWithDeferredConstraints handles circular dependencies by creating tables without FK first
func (b *Driver) syncSchemasWithDeferredConstraints(ctx context.Context, db types.Database, schemas map[string]*schema.Schema, currentTableMap map[string]bool) error {
	migrator := db.GetMigrator()

	// Phase 1: Create all tables without foreign keys
	for _, sch := range schemas {
		if sch.TableName == "" || currentTableMap[sch.TableName] {
			continue
		}

		// For now, we'll create tables with foreign keys and let the database handle the order
		// In a more complete implementation, we would:
		// 1. Generate CREATE TABLE without FK constraints
		// 2. After all tables are created, use ALTER TABLE to add FK constraints
		sql, err := migrator.GenerateCreateTableSQL(sch)
		if err != nil {
			return fmt.Errorf("failed to generate CREATE TABLE SQL for %s: %w", sch.TableName, err)
		}

		if err := migrator.ApplyMigration(sql); err != nil {
			// Try to create without the schema reference to handle circular deps
			// This is a simplified approach - a full implementation would parse and modify the SQL
			return fmt.Errorf("failed to create table %s: %w (possible circular dependency)", sch.TableName, err)
		}

		// Create indexes for the new table
		for _, index := range sch.Indexes {
			// Convert field names to column names
			columnNames := make([]string, len(index.Fields))
			for i, fieldName := range index.Fields {
				if field := sch.GetFieldByName(fieldName); field != nil {
					columnNames[i] = field.GetColumnName()
				} else {
					columnNames[i] = fieldName // Fallback to field name
				}
			}

			indexSQL := migrator.GenerateCreateIndexSQL(sch.TableName, index.Name, columnNames, index.Unique)
			if err := migrator.ApplyMigration(indexSQL); err != nil {
				return fmt.Errorf("failed to create index %s on table %s: %w", index.Name, sch.TableName, err)
			}
		}
	}

	// Phase 2: Handle existing tables (updates)
	for _, sch := range schemas {
		if sch.TableName == "" || !currentTableMap[sch.TableName] {
			continue
		}

		// Table exists, check for differences
		tableInfo, err := migrator.GetTableInfo(sch.TableName)
		if err != nil {
			return fmt.Errorf("failed to get table info for %s: %w", sch.TableName, err)
		}

		// Compare schema with existing table
		plan, err := migrator.CompareSchema(tableInfo, sch)
		if err != nil {
			return fmt.Errorf("failed to compare schema for %s: %w", sch.TableName, err)
		}

		// Debug: Check if plan has any changes
		if logger, ok := b.Logger.(utils.Logger); ok && logger != nil {
			hasChanges := plan != nil && (len(plan.AddColumns) > 0 || len(plan.ModifyColumns) > 0 ||
				len(plan.DropColumns) > 0 || len(plan.AddIndexes) > 0 || len(plan.DropIndexes) > 0)
			if hasChanges && plan != nil {
				logger.Debug("Table '%s' migration plan details:", sch.TableName)
				logger.Debug("  - AddColumns: %d", len(plan.AddColumns))
				logger.Debug("  - ModifyColumns: %d", len(plan.ModifyColumns))
				logger.Debug("  - DropColumns: %d", len(plan.DropColumns))
				logger.Debug("  - AddIndexes: %d", len(plan.AddIndexes))
				logger.Debug("  - DropIndexes: %d", len(plan.DropIndexes))
			} else {
				logger.Debug("Table '%s' has no migration changes", sch.TableName)
			}
		}

		// Generate and apply migration SQL
		if plan != nil && (len(plan.AddColumns) > 0 || len(plan.ModifyColumns) > 0 ||
			len(plan.DropColumns) > 0 || len(plan.AddIndexes) > 0 || len(plan.DropIndexes) > 0) {

			// Log migration plan details
			if logger, ok := b.Logger.(utils.Logger); ok && logger != nil {
				logger.Warn("Table '%s' needs migration:", sch.TableName)

				// Log column changes
				for _, change := range plan.AddColumns {
					logger.Warn("  - Adding column: %s", change.ColumnName)
				}
				for _, change := range plan.ModifyColumns {
					if change.OldColumn != nil && change.NewColumn != nil {
						var changes []string
						if change.OldColumn.Type != change.NewColumn.Type {
							changes = append(changes, fmt.Sprintf("type: %s -> %s", change.OldColumn.Type, change.NewColumn.Type))
						}
						if change.OldColumn.Nullable != change.NewColumn.Nullable {
							changes = append(changes, fmt.Sprintf("nullable: %v -> %v", change.OldColumn.Nullable, change.NewColumn.Nullable))
						}
						if change.OldColumn.Default != change.NewColumn.Default {
							changes = append(changes, fmt.Sprintf("default: %v -> %v", change.OldColumn.Default, change.NewColumn.Default))
						}
						logger.Warn("  - Modifying column '%s': %s", change.ColumnName, strings.Join(changes, ", "))
					}
				}
				for _, change := range plan.DropColumns {
					logger.Warn("  - Dropping column: %s", change.ColumnName)
				}

				// Log index changes
				for _, change := range plan.AddIndexes {
					logger.Warn("  - Adding index: %s", change.IndexName)
				}
				for _, change := range plan.DropIndexes {
					logger.Warn("  - Dropping index: %s", change.IndexName)
				}
			}

			sqlStatements, err := migrator.GenerateMigrationSQL(plan)
			if err != nil {
				return fmt.Errorf("failed to generate migration SQL for %s: %w", sch.TableName, err)
			}

			// Apply each SQL statement
			for _, sql := range sqlStatements {
				// Skip empty SQL statements
				if strings.TrimSpace(sql) == "" {
					continue
				}
				if err := migrator.ApplyMigration(sql); err != nil {
					return fmt.Errorf("failed to apply migration for %s: %w", sch.TableName, err)
				}
			}
		}
	}

	return nil
}
