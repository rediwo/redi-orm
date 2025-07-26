package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Add init function to register the driver
func init() {
	driverType := types.DriverPostgreSQL

	// Register PostgreSQL driver
	registry.Register(string(driverType), func(uri string) (types.Database, error) {
		return NewPostgreSQLDB(uri)
	})
	// Also register with "postgres" alias for compatibility
	registry.Register("postgres", func(uri string) (types.Database, error) {
		return NewPostgreSQLDB(uri)
	})

	// Register PostgreSQL capabilities
	registry.RegisterCapabilities(driverType, NewPostgreSQLCapabilities())

	// Register PostgreSQL URI parser
	registry.RegisterURIParser(string(driverType), NewPostgreSQLURIParser())
	// Also register with "postgres" alias
	registry.RegisterURIParser("postgres", NewPostgreSQLURIParser())
}

// PostgreSQLDB implements the Database interface for PostgreSQL
type PostgreSQLDB struct {
	*base.Driver
	nativeURI string
}

// NewPostgreSQLDB creates a new PostgreSQL database instance
// The uri parameter should be a native PostgreSQL DSN (e.g., "host=localhost port=5432 user=user dbname=db")
func NewPostgreSQLDB(nativeURI string) (*PostgreSQLDB, error) {
	return &PostgreSQLDB{
		Driver:    base.NewDriver(nativeURI, types.DriverPostgreSQL),
		nativeURI: nativeURI,
	}, nil
}

// GetDriverType returns the database driver type
func (p *PostgreSQLDB) GetDriverType() string {
	return "postgresql"
}

// GetCapabilities returns driver capabilities
func (p *PostgreSQLDB) GetCapabilities() types.DriverCapabilities {
	return NewPostgreSQLCapabilities()
}

// GetBooleanLiteral returns PostgreSQL-specific boolean literal
func (p *PostgreSQLDB) GetBooleanLiteral(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// Connect establishes a connection to the PostgreSQL database
func (p *PostgreSQLDB) Connect(ctx context.Context) error {
	if p.DB != nil {
		return nil
	}

	db, err := sql.Open("postgres", p.nativeURI)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for better performance
	db.SetMaxOpenConns(25) // Maximum number of open connections
	db.SetMaxIdleConns(5)  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.SetDB(db)
	return nil
}

// SyncSchemas synchronizes all loaded schemas with the database
func (p *PostgreSQLDB) SyncSchemas(ctx context.Context) error {
	return p.Driver.SyncSchemas(ctx, p)
}

// CreateModel creates a table from the registered schema
func (p *PostgreSQLDB) CreateModel(ctx context.Context, modelName string) error {
	return p.Driver.CreateModel(ctx, p, modelName)
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
	// Convert ? placeholders to $1, $2, etc.
	query = convertPlaceholders(query)
	// Debug logging handled by base driver
	return p.DB.Exec(query, args...)
}

// Query executes a raw SQL query and returns rows
func (p *PostgreSQLDB) Query(query string, args ...any) (*sql.Rows, error) {
	// Convert ? placeholders to $1, $2, etc.
	query = convertPlaceholders(query)
	// Debug logging handled by base driver
	return p.DB.Query(query, args...)
}

// QueryRow executes a raw SQL query and returns a single row
func (p *PostgreSQLDB) QueryRow(query string, args ...any) *sql.Row {
	// Convert ? placeholders to $1, $2, etc.
	query = convertPlaceholders(query)
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

	// Add foreign key constraints
	for _, relation := range schema.Relations {
		if relation.Type == "manyToOne" ||
			(relation.Type == "oneToOne" && relation.ForeignKey != "") {
			// Get the referenced table name
			referencedSchema, err := p.GetSchema(relation.Model)
			if err != nil {
				// If we can't find the schema, use the model name as table name (pluralized and lowercased)
				// This might happen during circular dependencies
				continue
			}

			// Find the actual field to get the column name
			var foreignKeyColumn string
			var foreignKeyFieldExists bool
			for _, field := range schema.Fields {
				if field.Name == relation.ForeignKey {
					foreignKeyColumn = field.GetColumnName()
					foreignKeyFieldExists = true
					break
				}
			}

			// Skip if the foreign key field doesn't exist in this schema
			// This happens when the foreign key is on the other side of a one-to-one relation
			if !foreignKeyFieldExists {
				continue
			}

			if foreignKeyColumn == "" {
				// If field not found, use the relation.ForeignKey as is (might be already a column name)
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
				"CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
				strings.ReplaceAll(schema.GetTableName(), ".", "_"), // Handle schema prefixes
				foreignKeyColumn,
				p.quoteIdentifier(foreignKeyColumn),
				p.quoteIdentifier(referencedSchema.GetTableName()),
				p.quoteIdentifier(referencesColumn),
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
		// Special handling for PostgreSQL functions
		if v == "CURRENT_TIMESTAMP" || v == "NOW()" {
			return v
		}
		// Convert "now()" to PostgreSQL's NOW()
		if v == "now()" {
			return "NOW()"
		}
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

// convertPlaceholders converts ? placeholders to $1, $2, etc. for PostgreSQL
func convertPlaceholders(sql string) string {
	var result strings.Builder
	argIndex := 0
	inQuote := false
	escaped := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		// Handle escape sequences
		if escaped {
			result.WriteByte(ch)
			escaped = false
			continue
		}

		// Handle escape character
		if ch == '\\' {
			result.WriteByte(ch)
			escaped = true
			continue
		}

		// Handle quotes
		if ch == '\'' {
			result.WriteByte(ch)
			inQuote = !inQuote
			continue
		}

		// Replace ? with $N when not in quotes
		if ch == '?' && !inQuote {
			// Check if this is a PostgreSQL JSON operator
			isJSONOp := false

			// Look ahead for JSON operators: ?& ?| or single ? preceded by JSON ops
			if i < len(sql)-1 {
				nextChar := sql[i+1]
				if nextChar == '&' || nextChar == '|' {
					isJSONOp = true
				}
			}

			// Also check if preceded by JSON path operators -> or ->>
			if i >= 1 && !isJSONOp {
				prevChar := sql[i-1]
				if prevChar == ' ' && i >= 2 {
					// Check for -> or ->> operators specifically (not just >)
					if i >= 3 && sql[i-3] == '-' && sql[i-2] == '>' {
						// This is -> operator
						isJSONOp = true
					} else if i >= 4 && sql[i-4] == '-' && sql[i-3] == '>' && sql[i-2] == '>' {
						// This is ->> operator
						isJSONOp = true
					}
				}
			}

			if !isJSONOp {
				argIndex++
				result.WriteString("$" + strconv.Itoa(argIndex))
			} else {
				result.WriteByte(ch)
			}
		} else {
			result.WriteByte(ch)
		}
	}

	return result.String()
}
