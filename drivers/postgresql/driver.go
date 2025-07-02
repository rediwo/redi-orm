package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/rediwo/redi-orm/drivers/base"
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
	*base.Driver
}

// NewPostgreSQLDB creates a new PostgreSQL database instance
func NewPostgreSQLDB(config types.Config) (*PostgreSQLDB, error) {
	return &PostgreSQLDB{
		Driver: base.NewDriver(config),
	}, nil
}

// GetDriverType returns the database driver type
func (p *PostgreSQLDB) GetDriverType() string {
	return "postgresql"
}

// Connect establishes a connection to the PostgreSQL database
func (p *PostgreSQLDB) Connect(ctx context.Context) error {
	if p.DB != nil {
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

	p.SetDB(db)
	return nil
}

// buildDSN builds the PostgreSQL connection string
func (p *PostgreSQLDB) buildDSN() string {
	var parts []string

	if p.Config.Host != "" {
		parts = append(parts, fmt.Sprintf("host=%s", p.Config.Host))
	}
	if p.Config.Port > 0 {
		parts = append(parts, fmt.Sprintf("port=%d", p.Config.Port))
	}
	if p.Config.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", p.Config.User))
	}
	if p.Config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", p.Config.Password))
	}
	if p.Config.Database != "" {
		parts = append(parts, fmt.Sprintf("dbname=%s", p.Config.Database))
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

// SyncSchemas synchronizes all loaded schemas with the database
func (p *PostgreSQLDB) SyncSchemas(ctx context.Context) error {
	return p.Driver.SyncSchemas(ctx, p)
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

	_, err = p.DB.ExecContext(ctx, sql)
	return err
}

// DropModel drops a table
func (p *PostgreSQLDB) DropModel(ctx context.Context, modelName string) error {
	tableName, err := p.ResolveTableName(modelName)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", p.quoteIdentifier(tableName))
	_, err = p.DB.ExecContext(ctx, sql)
	return err
}

// Model creates a new model query
func (p *PostgreSQLDB) Model(modelName string) types.ModelQuery {
	return query.NewModelQuery(modelName, p, p.GetFieldMapper())
}

// Raw creates a raw query
func (p *PostgreSQLDB) Raw(query string, args ...any) types.RawQuery {
	return &PostgreSQLRawQuery{
		db:   p.DB,
		sql:  query,
		args: args,
	}
}

// Transaction executes a function within a transaction
func (p *PostgreSQLDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	sqlTx, err := p.DB.BeginTx(ctx, nil)
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
	sqlTx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgreSQLTransaction{
		tx: sqlTx,
		db: p,
	}, nil
}

// Exec executes a raw SQL query
func (p *PostgreSQLDB) Exec(query string, args ...any) (sql.Result, error) {
	return p.DB.Exec(query, args...)
}

// Query executes a raw SQL query and returns rows
func (p *PostgreSQLDB) Query(query string, args ...any) (*sql.Rows, error) {
	return p.DB.Query(query, args...)
}

// QueryRow executes a raw SQL query and returns a single row
func (p *PostgreSQLDB) QueryRow(query string, args ...any) *sql.Row {
	return p.DB.QueryRow(query, args...)
}

// GetMigrator returns the migrator for this database
func (p *PostgreSQLDB) GetMigrator() types.DatabaseMigrator {
	return NewPostgreSQLMigrator(p.DB, p)
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
func (p *PostgreSQLDB) formatDefaultValue(value any, fieldType schema.FieldType) string {
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
