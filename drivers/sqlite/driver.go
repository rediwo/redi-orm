package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
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
	db          *sql.DB
	config      types.Config
	fieldMapper types.FieldMapper
	schemas     map[string]*schema.Schema
}

// NewSQLiteDB creates a new SQLite database instance
func NewSQLiteDB(config types.Config) (*SQLiteDB, error) {
	return &SQLiteDB{
		config:      config,
		fieldMapper: types.NewDefaultFieldMapper(),
		schemas:     make(map[string]*schema.Schema),
	}, nil
}

// Connect establishes connection to SQLite database
func (s *SQLiteDB) Connect(ctx context.Context) error {
	db, err := sql.Open("sqlite3", s.config.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	s.db = db
	return nil
}

// Close closes the database connection
func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (s *SQLiteDB) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}
	return s.db.PingContext(ctx)
}

// RegisterSchema registers a schema with the database
func (s *SQLiteDB) RegisterSchema(modelName string, schema *schema.Schema) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if err := schema.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	s.schemas[modelName] = schema

	// Register with field mapper
	if mapper, ok := s.fieldMapper.(*types.DefaultFieldMapper); ok {
		mapper.RegisterSchema(modelName, schema)
	}

	return nil
}

// GetSchema returns a registered schema
func (s *SQLiteDB) GetSchema(modelName string) (*schema.Schema, error) {
	schema, exists := s.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("schema for model '%s' not registered", modelName)
	}
	return schema, nil
}

// GetModels returns all registered model names
func (s *SQLiteDB) GetModels() []string {
	models := make([]string, 0, len(s.schemas))
	for modelName := range s.schemas {
		models = append(models, modelName)
	}
	return models
}

// GetModelSchema returns schema for a model (alias for GetSchema)
func (s *SQLiteDB) GetModelSchema(modelName string) (*schema.Schema, error) {
	return s.GetSchema(modelName)
}

// ResolveTableName resolves model name to table name
func (s *SQLiteDB) ResolveTableName(modelName string) (string, error) {
	return s.fieldMapper.ModelToTable(modelName)
}

// ResolveFieldName resolves schema field name to column name
func (s *SQLiteDB) ResolveFieldName(modelName, fieldName string) (string, error) {
	return s.fieldMapper.SchemaToColumn(modelName, fieldName)
}

// ResolveFieldNames resolves multiple schema field names to column names
func (s *SQLiteDB) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return s.fieldMapper.SchemaFieldsToColumns(modelName, fieldNames)
}

// GetFieldMapper returns the field mapper
func (s *SQLiteDB) GetFieldMapper() types.FieldMapper {
	return s.fieldMapper
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

	_, err = s.db.ExecContext(ctx, sql)
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
	_, err = s.db.ExecContext(ctx, sql)
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
func (s *SQLiteDB) Raw(sql string, args ...interface{}) types.RawQuery {
	return NewSQLiteRawQuery(s.db, sql, args...)
}

// Begin starts a new transaction
func (s *SQLiteDB) Begin(ctx context.Context) (types.Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return NewSQLiteTransaction(tx, s), nil
}

// Exec executes a raw SQL statement
func (s *SQLiteDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

// Query executes a raw SQL query that returns rows
func (s *SQLiteDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

// QueryRow executes a raw SQL query that returns a single row
func (s *SQLiteDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.db.QueryRow(query, args...)
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

// GetMigrator returns a migrator for SQLite
func (s *SQLiteDB) GetMigrator() types.DatabaseMigrator {
	return NewSQLiteMigrator(s.db, s)
}

// generateCreateTableSQL generates CREATE TABLE SQL for a schema
func (s *SQLiteDB) generateCreateTableSQL(schema *schema.Schema) (string, error) {
	var columns []string

	for _, field := range schema.Fields {
		column, err := s.generateColumnSQL(field)
		if err != nil {
			return "", fmt.Errorf("failed to generate column SQL for field %s: %w", field.Name, err)
		}
		columns = append(columns, column)
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		schema.GetTableName(),
		fmt.Sprintf("%s", columns[0]))

	if len(columns) > 1 {
		for _, column := range columns[1:] {
			sql = sql[:len(sql)-2] + ",\n  " + column + "\n)"
		}
	}

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
		parts = append(parts, fmt.Sprintf("DEFAULT %v", field.Default))
	}

	return strings.Join(parts, " "), nil
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
