package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rediwo/redi-orm/base"
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
	*base.Driver
}

// NewMySQLDB creates a new MySQL database instance
func NewMySQLDB(config types.Config) (*MySQLDB, error) {
	return &MySQLDB{
		Driver: base.NewDriver(config),
	}, nil
}

// Connect establishes connection to MySQL database
func (m *MySQLDB) Connect(ctx context.Context) error {
	dsn := m.buildDSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	m.SetDB(db)
	return nil
}

// buildDSN builds the MySQL connection string
func (m *MySQLDB) buildDSN() string {
	// Basic DSN format: user:password@tcp(host:port)/database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		m.Config.User,
		m.Config.Password,
		m.Config.Host,
		m.Config.Port,
		m.Config.Database,
	)

	// Add query parameters from Options
	var params []string

	// Add options from Config.Options
	if m.Config.Options != nil {
		for key, value := range m.Config.Options {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Ensure parseTime and charset have defaults if not specified
	if m.Config.Options == nil || m.Config.Options["parseTime"] == "" {
		hasParseTime := false
		for _, p := range params {
			if strings.HasPrefix(p, "parseTime=") {
				hasParseTime = true
				break
			}
		}
		if !hasParseTime {
			params = append(params, "parseTime=true")
		}
	}

	if m.Config.Options == nil || m.Config.Options["charset"] == "" {
		hasCharset := false
		for _, p := range params {
			if strings.HasPrefix(p, "charset=") {
				hasCharset = true
				break
			}
		}
		if !hasCharset {
			params = append(params, "charset=utf8mb4")
		}
	}

	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	return dsn
}

// SyncSchemas synchronizes all loaded schemas with the database
func (m *MySQLDB) SyncSchemas(ctx context.Context) error {
	return m.Driver.SyncSchemas(ctx, m)
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

	_, err = m.DB.ExecContext(ctx, sql)
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
	_, err = m.DB.ExecContext(ctx, sql)
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
func (m *MySQLDB) Raw(sql string, args ...any) types.RawQuery {
	return NewMySQLRawQuery(m.DB, sql, args...)
}

// Begin starts a new transaction
func (m *MySQLDB) Begin(ctx context.Context) (types.Transaction, error) {
	tx, err := m.DB.BeginTx(ctx, nil)
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
func (m *MySQLDB) Exec(query string, args ...any) (sql.Result, error) {
	return m.DB.Exec(query, args...)
}

// Query executes a raw SQL query that returns rows
func (m *MySQLDB) Query(query string, args ...any) (*sql.Rows, error) {
	return m.DB.Query(query, args...)
}

// QueryRow executes a raw SQL query that returns a single row
func (m *MySQLDB) QueryRow(query string, args ...any) *sql.Row {
	return m.DB.QueryRow(query, args...)
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

	// Add foreign key constraints
	for _, relation := range schema.Relations {
		if relation.Type == "manyToOne" ||
			(relation.Type == "oneToOne" && relation.ForeignKey != "") {
			// Get the referenced table name
			referencedSchema, err := m.GetSchema(relation.Model)
			if err != nil {
				// If we can't find the schema, skip this foreign key
				// This might happen during circular dependencies
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
				// If field not found, use the relation.ForeignKey as is
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
				"CONSTRAINT `fk_%s_%s` FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`)",
				strings.ReplaceAll(schema.GetTableName(), ".", "_"),
				foreignKeyColumn,
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
func (m *MySQLDB) formatDefaultValue(value any) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		// Special handling for MySQL functions
		if v == "CURRENT_TIMESTAMP" || v == "NOW()" {
			return v
		}
		// Convert "now()" to MySQL's CURRENT_TIMESTAMP
		if v == "now()" {
			return "CURRENT_TIMESTAMP"
		}
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
// GetDriverType returns the database driver type
func (m *MySQLDB) GetDriverType() string {
	return "mysql"
}

// QuoteIdentifier quotes an identifier for MySQL using backticks
func (m *MySQLDB) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

// SupportsDefaultValues returns false for MySQL as it doesn't support DEFAULT VALUES syntax
func (m *MySQLDB) SupportsDefaultValues() bool {
	return false
}

// GetNullsOrderingSQL returns the SQL clause for NULL ordering
// MySQL doesn't support NULLS FIRST/LAST syntax
func (m *MySQLDB) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	return "" // MySQL doesn't support NULLS FIRST/LAST
}

// RequiresLimitForOffset returns true if the database requires LIMIT when using OFFSET
// MySQL requires LIMIT when using OFFSET
func (m *MySQLDB) RequiresLimitForOffset() bool {
	return true
}

func (m *MySQLDB) GetMigrator() types.DatabaseMigrator {
	return NewMySQLMigrator(m.DB, m)
}
