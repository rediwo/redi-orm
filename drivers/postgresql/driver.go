package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Add init function to register the driver
func init() {
	// Register PostgreSQL driver
	registry.Register("postgresql", func(config types.Config) (types.Database, error) {
		return NewPostgreSQLDB(config)
	})
	registry.Register("postgres", func(config types.Config) (types.Database, error) {
		return NewPostgreSQLDB(config)
	})
	
	// Register PostgreSQL URI parser
	registry.RegisterURIParser("postgresql", NewPostgreSQLURIParser())
	registry.RegisterURIParser("postgres", NewPostgreSQLURIParser())
}

// PostgreSQLDB implements the Database interface for PostgreSQL
type PostgreSQLDB struct {
	config      types.Config
	db          *sql.DB
	fieldMapper types.FieldMapper
	schemas     map[string]*schema.Schema
}

// NewPostgreSQLDB creates a new PostgreSQL database instance
func NewPostgreSQLDB(config types.Config) (*PostgreSQLDB, error) {
	return &PostgreSQLDB{
		config:      config,
		fieldMapper: types.NewDefaultFieldMapper(),
		schemas:     make(map[string]*schema.Schema),
	}, nil
}

// GetDriverType returns the database driver type
func (p *PostgreSQLDB) GetDriverType() string {
	return "postgresql"
}

// Connect establishes a connection to the PostgreSQL database
func (p *PostgreSQLDB) Connect(ctx context.Context) error {
	if p.db != nil {
		return nil
	}

	dsn := p.buildDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	return nil
}

// buildDSN builds the PostgreSQL connection string
func (p *PostgreSQLDB) buildDSN() string {
	var parts []string

	if p.config.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", p.config.Host))
	}
	if p.config.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", p.config.Port))
	}
	if p.config.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", p.config.User))
	}
	if p.config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", p.config.Password))
	}
	if p.config.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", p.config.Database))
	}
	
	// Add sslmode if not specified
	hasSSLMode := false
	for _, part := range parts {
		if strings.HasPrefix(part, "sslmode=") {
			hasSSLMode = true
			break
		}
	}
	if !hasSSLMode {
		parts = append(parts, "sslmode=disable")
	}

	return strings.Join(parts, " ")
}

// Close closes the database connection
func (p *PostgreSQLDB) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Ping pings the database
func (p *PostgreSQLDB) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.PingContext(ctx)
}

// RegisterSchema registers a schema with the database
func (p *PostgreSQLDB) RegisterSchema(modelName string, s *schema.Schema) error {
	if s == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if err := s.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	p.schemas[modelName] = s

	// Register with field mapper
	if mapper, ok := p.fieldMapper.(*types.DefaultFieldMapper); ok {
		mapper.RegisterSchema(modelName, s)
	}

	return nil
}

// GetSchema returns the schema for a model
func (p *PostgreSQLDB) GetSchema(modelName string) (*schema.Schema, error) {
	s, exists := p.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("schema not found for model: %s", modelName)
	}
	return s, nil
}

// GetModelSchema returns the schema for a model (alias for GetSchema)
func (p *PostgreSQLDB) GetModelSchema(modelName string) (*schema.Schema, error) {
	return p.GetSchema(modelName)
}

// GetModels returns all registered model names
func (p *PostgreSQLDB) GetModels() []string {
	models := make([]string, 0, len(p.schemas))
	for modelName := range p.schemas {
		models = append(models, modelName)
	}
	return models
}

// ResolveTableName resolves model name to table name
func (p *PostgreSQLDB) ResolveTableName(modelName string) (string, error) {
	return p.fieldMapper.ModelToTable(modelName)
}

// ResolveFieldName resolves field name to column name
func (p *PostgreSQLDB) ResolveFieldName(modelName, fieldName string) (string, error) {
	return p.fieldMapper.SchemaToColumn(modelName, fieldName)
}

// ResolveFieldNames resolves multiple field names to column names
func (p *PostgreSQLDB) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return p.fieldMapper.SchemaFieldsToColumns(modelName, fieldNames)
}

// GetFieldMapper returns the field mapper
func (p *PostgreSQLDB) GetFieldMapper() types.FieldMapper {
	return p.fieldMapper
}

// CreateModel creates a table from the registered schema
func (p *PostgreSQLDB) CreateModel(ctx context.Context, modelName string) error {
	s, err := p.GetSchema(modelName)
	if err != nil {
		return err
	}

	sql, err := p.generateCreateTableSQL(s)
	if err != nil {
		return err
	}

	_, err = p.db.ExecContext(ctx, sql)
	return err
}

// DropModel drops a table
func (p *PostgreSQLDB) DropModel(ctx context.Context, modelName string) error {
	tableName, err := p.ResolveTableName(modelName)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", p.quoteIdentifier(tableName))
	_, err = p.db.ExecContext(ctx, sql)
	return err
}

// Model creates a new model query
func (p *PostgreSQLDB) Model(modelName string) types.ModelQuery {
	return query.NewModelQuery(modelName, p, p.GetFieldMapper())
}

// Raw creates a raw query
func (p *PostgreSQLDB) Raw(query string, args ...interface{}) types.RawQuery {
	return &PostgreSQLRawQuery{
		db:   p.db,
		sql:  query,
		args: args,
	}
}

// Transaction executes a function within a transaction
func (p *PostgreSQLDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	sqlTx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	tx := &PostgreSQLTransaction{
		tx: sqlTx,
		db: p,
	}

	if err := fn(tx); err != nil {
		if rbErr := sqlTx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Begin starts a new transaction
func (p *PostgreSQLDB) Begin(ctx context.Context) (types.Transaction, error) {
	sqlTx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgreSQLTransaction{
		tx: sqlTx,
		db: p,
	}, nil
}

// Exec executes a raw SQL query
func (p *PostgreSQLDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return p.db.Exec(query, args...)
}

// Query executes a raw SQL query and returns rows
func (p *PostgreSQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.Query(query, args...)
}

// QueryRow executes a raw SQL query and returns a single row
func (p *PostgreSQLDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return p.db.QueryRow(query, args...)
}

// GetMigrator returns the migrator for this database
func (p *PostgreSQLDB) GetMigrator() types.DatabaseMigrator {
	return NewPostgreSQLMigrator(p.db, p)
}

// generateCreateTableSQL generates CREATE TABLE SQL for PostgreSQL
func (p *PostgreSQLDB) generateCreateTableSQL(schema *schema.Schema) (string, error) {
	var columns []string
	var primaryKeys []string
	
	for _, field := range schema.Fields {
		column, err := p.generateColumnSQL(field)
		if err != nil {
			return "", fmt.Errorf("failed to generate column SQL for field %s: %w", field.Name, err)
		}
		columns = append(columns, column)
		
		if field.PrimaryKey {
			primaryKeys = append(primaryKeys, p.quoteIdentifier(field.GetColumnName()))
		}
	}
	
	if len(primaryKeys) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}
	
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		p.quoteIdentifier(schema.GetTableName()),
		strings.Join(columns, ",\n  "))
	
	return sql, nil
}

// generateColumnSQL generates column definition SQL
func (p *PostgreSQLDB) generateColumnSQL(field schema.Field) (string, error) {
	var parts []string
	
	columnName := p.quoteIdentifier(field.GetColumnName())
	columnType := p.mapFieldTypeToSQL(field.Type)
	
	// Handle SERIAL for auto increment primary keys
	if field.PrimaryKey && field.AutoIncrement {
		if field.Type == schema.FieldTypeInt {
			columnType = "SERIAL"
		} else if field.Type == schema.FieldTypeInt64 {
			columnType = "BIGSERIAL"
		}
	}
	
	parts = append(parts, columnName, columnType)
	
	// Add NOT NULL constraint
	if !field.Nullable && !field.AutoIncrement {
		parts = append(parts, "NOT NULL")
	}
	
	// Add UNIQUE constraint
	if field.Unique && !field.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}
	
	// Add DEFAULT value
	if field.Default != nil && !field.AutoIncrement {
		defaultValue := p.formatDefaultValue(field.Default, field.Type)
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
	}
	
	return strings.Join(parts, " "), nil
}

// mapFieldTypeToSQL maps schema field types to PostgreSQL data types
func (p *PostgreSQLDB) mapFieldTypeToSQL(fieldType schema.FieldType) string {
	switch fieldType {
	case schema.FieldTypeString:
		return "VARCHAR(255)"
	case schema.FieldTypeInt:
		return "INTEGER"
	case schema.FieldTypeInt64:
		return "BIGINT"
	case schema.FieldTypeFloat:
		return "DOUBLE PRECISION"
	case schema.FieldTypeBool:
		return "BOOLEAN"
	case schema.FieldTypeDateTime:
		return "TIMESTAMP"
	case schema.FieldTypeJSON:
		return "JSONB"
	case schema.FieldTypeDecimal:
		return "DECIMAL(10,2)"
	default:
		return "TEXT"
	}
}

// formatDefaultValue formats a default value for SQL
func (p *PostgreSQLDB) formatDefaultValue(value interface{}, fieldType schema.FieldType) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case nil:
		return "NULL"
	default:
		return fmt.Sprintf("%v", value)
	}
}

// quoteIdentifier quotes an identifier for PostgreSQL
func (p *PostgreSQLDB) quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}