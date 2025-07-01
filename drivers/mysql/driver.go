package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	// Register MySQL driver
	registry.Register("mysql", func(config types.Config) (types.Database, error) {
		return NewMySQLDB(config)
	})
	
	// Register MySQL URI parser
	registry.RegisterURIParser("mysql", NewMySQLURIParser())
}

// MySQLDB implements the Database interface for MySQL
type MySQLDB struct {
	db          *sql.DB
	config      types.Config
	fieldMapper types.FieldMapper
	schemas     map[string]*schema.Schema
}

// NewMySQLDB creates a new MySQL database instance
func NewMySQLDB(config types.Config) (*MySQLDB, error) {
	return &MySQLDB{
		config:      config,
		fieldMapper: types.NewDefaultFieldMapper(),
		schemas:     make(map[string]*schema.Schema),
	}, nil
}

// Connect establishes connection to MySQL database
func (m *MySQLDB) Connect(ctx context.Context) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		m.config.User,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	m.db = db
	return nil
}

// Close closes the database connection
func (m *MySQLDB) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (m *MySQLDB) Ping(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}
	return m.db.PingContext(ctx)
}

// RegisterSchema registers a schema with the database
func (m *MySQLDB) RegisterSchema(modelName string, schema *schema.Schema) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if err := schema.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	m.schemas[modelName] = schema

	// Register with field mapper
	if mapper, ok := m.fieldMapper.(*types.DefaultFieldMapper); ok {
		mapper.RegisterSchema(modelName, schema)
	}

	return nil
}

// GetSchema returns a registered schema
func (m *MySQLDB) GetSchema(modelName string) (*schema.Schema, error) {
	schema, exists := m.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("schema for model '%s' not registered", modelName)
	}
	return schema, nil
}

// GetModels returns all registered model names
func (m *MySQLDB) GetModels() []string {
	models := make([]string, 0, len(m.schemas))
	for modelName := range m.schemas {
		models = append(models, modelName)
	}
	return models
}

// GetModelSchema returns schema for a model (alias for GetSchema)
func (m *MySQLDB) GetModelSchema(modelName string) (*schema.Schema, error) {
	return m.GetSchema(modelName)
}

// ResolveTableName resolves model name to table name
func (m *MySQLDB) ResolveTableName(modelName string) (string, error) {
	return m.fieldMapper.ModelToTable(modelName)
}

// ResolveFieldName resolves schema field name to column name
func (m *MySQLDB) ResolveFieldName(modelName, fieldName string) (string, error) {
	return m.fieldMapper.SchemaToColumn(modelName, fieldName)
}

// ResolveFieldNames resolves multiple schema field names to column names
func (m *MySQLDB) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return m.fieldMapper.SchemaFieldsToColumns(modelName, fieldNames)
}

// GetFieldMapper returns the field mapper
func (m *MySQLDB) GetFieldMapper() types.FieldMapper {
	return m.fieldMapper
}

// CreateModel creates a table for the given model
func (m *MySQLDB) CreateModel(ctx context.Context, modelName string) error {
	schema, err := m.GetSchema(modelName)
	if err != nil {
		return fmt.Errorf("failed to get schema for model %s: %w", modelName, err)
	}

	sql, err := m.generateCreateTableSQL(schema)
	if err != nil {
		return fmt.Errorf("failed to generate CREATE TABLE SQL: %w", err)
	}

	_, err = m.db.ExecContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// DropModel drops the table for the given model
func (m *MySQLDB) DropModel(ctx context.Context, modelName string) error {
	tableName, err := m.ResolveTableName(modelName)
	if err != nil {
		return fmt.Errorf("failed to resolve table name: %w", err)
	}

	sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
	_, err = m.db.ExecContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	return nil
}

// Model creates a new model query
func (m *MySQLDB) Model(modelName string) types.ModelQuery {
	return query.NewModelQuery(modelName, m, m.GetFieldMapper())
}

// Raw creates a new raw query
func (m *MySQLDB) Raw(sql string, args ...interface{}) types.RawQuery {
	return NewMySQLRawQuery(m.db, sql, args...)
}

// Begin starts a new transaction
func (m *MySQLDB) Begin(ctx context.Context) (types.Transaction, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return NewMySQLTransaction(tx, m), nil
}

// Transaction executes a function within a transaction
func (m *MySQLDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	tx, err := m.Begin(ctx)
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

// Exec executes a raw SQL statement
func (m *MySQLDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.db.Exec(query, args...)
}

// Query executes a raw SQL query that returns rows
func (m *MySQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.Query(query, args...)
}

// QueryRow executes a raw SQL query that returns a single row
func (m *MySQLDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return m.db.QueryRow(query, args...)
}

// generateCreateTableSQL generates CREATE TABLE SQL for MySQL
func (m *MySQLDB) generateCreateTableSQL(schema *schema.Schema) (string, error) {
	var columns []string
	var primaryKeys []string

	for _, field := range schema.Fields {
		column, err := m.generateColumnSQL(field)
		if err != nil {
			return "", fmt.Errorf("failed to generate column SQL for field %s: %w", field.Name, err)
		}
		columns = append(columns, column)
		
		if field.PrimaryKey {
			primaryKeys = append(primaryKeys, fmt.Sprintf("`%s`", field.GetColumnName()))
		}
	}

	// Add primary key constraint if we have primary keys
	if len(primaryKeys) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n  %s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",
		schema.GetTableName(),
		strings.Join(columns, ",\n  "))

	return sql, nil
}

// generateColumnSQL generates SQL for a single column
func (m *MySQLDB) generateColumnSQL(field schema.Field) (string, error) {
	columnName := field.GetColumnName()
	sqlType := m.mapFieldTypeToSQL(field.Type)

	var parts []string
	parts = append(parts, fmt.Sprintf("`%s` %s", columnName, sqlType))

	if !field.Nullable {
		parts = append(parts, "NOT NULL")
	}

	if field.AutoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
	}

	if field.Unique && !field.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if field.Default != nil {
		defaultValue := m.formatDefaultValue(field.Default)
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
	}

	return strings.Join(parts, " "), nil
}

// mapFieldTypeToSQL maps schema field types to MySQL SQL types
func (m *MySQLDB) mapFieldTypeToSQL(fieldType schema.FieldType) string {
	switch fieldType {
	case schema.FieldTypeString:
		return "VARCHAR(255)"
	case schema.FieldTypeInt:
		return "INT"
	case schema.FieldTypeInt64:
		return "BIGINT"
	case schema.FieldTypeFloat:
		return "DOUBLE"
	case schema.FieldTypeBool:
		return "BOOLEAN"
	case schema.FieldTypeDateTime:
		return "DATETIME"
	case schema.FieldTypeJSON:
		return "JSON"
	case schema.FieldTypeDecimal:
		return "DECIMAL(10,2)"
	default:
		return "VARCHAR(255)"
	}
}

// formatDefaultValue formats a default value for MySQL
func (m *MySQLDB) formatDefaultValue(value interface{}) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", value)
	}
}




// GetMigrator returns a migrator for MySQL
func (m *MySQLDB) GetMigrator() types.DatabaseMigrator {
	return NewMySQLMigrator(m.db, m)
}
