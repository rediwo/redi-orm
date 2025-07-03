package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	// Register SQLite driver
	registry.Register("sqlite", func(config types.Config) (types.Database, error) {
		return NewSQLiteDB(config)
	})

	// Register SQLite URI parser
	registry.RegisterURIParser("sqlite", NewSQLiteURIParser())
}

// SQLiteDB implements the Database interface for SQLite
type SQLiteDB struct {
	*base.Driver
}

// NewSQLiteDB creates a new SQLite database instance
func NewSQLiteDB(config types.Config) (*SQLiteDB, error) {
	return &SQLiteDB{
		Driver: base.NewDriver(config),
	}, nil
}

// Connect establishes connection to SQLite database
func (s *SQLiteDB) Connect(ctx context.Context) error {
	db, err := sql.Open("sqlite3", s.Config.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	s.SetDB(db)
	
	// Enable foreign key constraints in SQLite
	_, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("failed to enable foreign key constraints: %w", err)
	}
	
	return nil
}

// SyncSchemas synchronizes all loaded schemas with the database
func (s *SQLiteDB) SyncSchemas(ctx context.Context) error {
	return s.Driver.SyncSchemas(ctx, s)
}

// CreateModel creates a table for the given model
func (s *SQLiteDB) CreateModel(ctx context.Context, modelName string) error {
	schema, err := s.GetSchema(modelName)
	if err != nil {
		return fmt.Errorf("failed to get schema for model %s: %w", modelName, err)
	}

	sql, err := s.generateCreateTableSQL(schema)
	if err != nil {
		return fmt.Errorf("failed to generate CREATE TABLE SQL: %w", err)
	}

	_, err = s.DB.ExecContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// DropModel drops the table for the given model
func (s *SQLiteDB) DropModel(ctx context.Context, modelName string) error {
	tableName, err := s.ResolveTableName(modelName)
	if err != nil {
		return fmt.Errorf("failed to resolve table name: %w", err)
	}

	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err = s.DB.ExecContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	return nil
}

// Model creates a new model query
func (s *SQLiteDB) Model(modelName string) types.ModelQuery {
	return query.NewModelQuery(modelName, s, s.GetFieldMapper())
}

// Raw creates a new raw query
func (s *SQLiteDB) Raw(sql string, args ...any) types.RawQuery {
	return NewSQLiteRawQuery(s.DB, sql, args...)
}

// Begin starts a new transaction
func (s *SQLiteDB) Begin(ctx context.Context) (types.Transaction, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return NewSQLiteTransaction(tx, s), nil
}

// Transaction executes a function within a transaction
func (s *SQLiteDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	tx, err := s.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDriverType returns the database driver type
func (s *SQLiteDB) GetDriverType() string {
	return "sqlite"
}

// GetMigrator returns a migrator for SQLite
func (s *SQLiteDB) GetMigrator() types.DatabaseMigrator {
	return NewSQLiteMigrator(s.DB, s)
}

// generateCreateTableSQL generates CREATE TABLE SQL for a schema
func (s *SQLiteDB) generateCreateTableSQL(schema *schema.Schema) (string, error) {
	var columns []string
	var primaryKeys []string

	for _, field := range schema.Fields {
		column, err := s.generateColumnSQL(field)
		if err != nil {
			return "", fmt.Errorf("failed to generate column SQL for field %s: %w", field.Name, err)
		}
		columns = append(columns, column)
		
		if field.PrimaryKey && !field.AutoIncrement {
			// For composite primary keys (non-autoincrement)
			primaryKeys = append(primaryKeys, field.GetColumnName())
		}
	}
	
	// Add composite primary key if needed
	if len(primaryKeys) > 1 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	// Add foreign key constraints
	for _, relation := range schema.Relations {
		if relation.Type == "manyToOne" || 
		   (relation.Type == "oneToOne" && relation.ForeignKey != "") {
			// Get the referenced table name
			referencedSchema, err := s.GetSchema(relation.Model)
			if err != nil {
				// If we can't find the schema, skip this foreign key
				continue
			}
			
			// Find the actual field to get the column name
			var foreignKeyColumn string
			for _, field := range schema.Fields {
				if field.Name == relation.ForeignKey {
					foreignKeyColumn = field.GetColumnName()
					break
				}
			}
			if foreignKeyColumn == "" {
				foreignKeyColumn = relation.ForeignKey
			}
			
			// Find the referenced column name
			var referencesColumn string
			for _, field := range referencedSchema.Fields {
				if field.Name == relation.References {
					referencesColumn = field.GetColumnName()
					break
				}
			}
			if referencesColumn == "" {
				referencesColumn = relation.References
			}
			
			fkConstraint := fmt.Sprintf(
				"FOREIGN KEY (%s) REFERENCES %s(%s)",
				foreignKeyColumn,
				referencedSchema.GetTableName(),
				referencesColumn,
			)
			
			// Add ON DELETE/UPDATE rules if specified
			if relation.OnDelete != "" {
				fkConstraint += " ON DELETE " + relation.OnDelete
			}
			if relation.OnUpdate != "" {
				fkConstraint += " ON UPDATE " + relation.OnUpdate
			}
			
			columns = append(columns, fkConstraint)
		}
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		schema.GetTableName(),
		strings.Join(columns, ",\n  "))

	return sql, nil
}

// generateColumnSQL generates SQL for a single column
func (s *SQLiteDB) generateColumnSQL(field schema.Field) (string, error) {
	columnName := field.GetColumnName()
	sqlType := s.mapFieldTypeToSQL(field.Type)

	var parts []string
	parts = append(parts, fmt.Sprintf("%s %s", columnName, sqlType))

	if field.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
		if field.AutoIncrement {
			parts = append(parts, "AUTOINCREMENT")
		}
	}

	if !field.Nullable && !field.PrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	if field.Unique && !field.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if field.Default != nil {
		defaultValue := s.formatDefaultValue(field.Default)
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
	}

	return strings.Join(parts, " "), nil
}

// formatDefaultValue formats a default value for SQLite
func (s *SQLiteDB) formatDefaultValue(value any) string {
	switch v := value.(type) {
	case string:
		// Handle special SQLite functions
		if v == "now()" || v == "CURRENT_TIMESTAMP" {
			return "CURRENT_TIMESTAMP"
		}
		// Handle boolean strings
		if v == "true" {
			return "1"
		}
		if v == "false" {
			return "0"
		}
		// Quote string values
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "1"
		}
		return "0"
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("%v", value)
	}
}

// mapFieldTypeToSQL maps schema field types to SQLite SQL types
func (s *SQLiteDB) mapFieldTypeToSQL(fieldType schema.FieldType) string {
	switch fieldType {
	case schema.FieldTypeString:
		return "TEXT"
	case schema.FieldTypeInt:
		return "INTEGER"
	case schema.FieldTypeInt64:
		return "INTEGER"
	case schema.FieldTypeFloat:
		return "REAL"
	case schema.FieldTypeBool:
		return "INTEGER" // SQLite doesn't have native boolean
	case schema.FieldTypeDateTime:
		return "DATETIME"
	case schema.FieldTypeJSON:
		return "TEXT" // Store JSON as text in SQLite
	case schema.FieldTypeDecimal:
		return "DECIMAL"
	default:
		return "TEXT"
	}
}
