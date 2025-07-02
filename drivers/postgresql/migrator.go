package postgresql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// PostgreSQLMigrator implements database-specific migration logic for PostgreSQL
type PostgreSQLMigrator struct {
	db           *sql.DB
	postgresqlDB *PostgreSQLDB
}

// PostgreSQLMigratorWrapper wraps PostgreSQLMigrator with BaseMigrator to implement types.DatabaseMigrator
type PostgreSQLMigratorWrapper struct {
	*migration.BaseMigrator
	specific *PostgreSQLMigrator
}

// NewPostgreSQLMigrator creates a new PostgreSQL migrator that implements types.DatabaseMigrator
func NewPostgreSQLMigrator(db *sql.DB, postgresqlDB *PostgreSQLDB) types.DatabaseMigrator {
	specific := &PostgreSQLMigrator{
		db:           db,
		postgresqlDB: postgresqlDB,
	}
	wrapper := &PostgreSQLMigratorWrapper{
		specific: specific,
	}
	wrapper.BaseMigrator = migration.NewBaseMigrator(specific)
	return wrapper
}

// GetDatabaseType returns the database type
func (m *PostgreSQLMigrator) GetDatabaseType() string {
	return "postgresql"
}

// GetTables returns all table names in the database
func (m *PostgreSQLMigrator) GetTables() ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// GetTableInfo returns information about a specific table
func (m *PostgreSQLMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	tableInfo := &types.TableInfo{
		Name:    tableName,
		Columns: []types.ColumnInfo{},
		Indexes: []types.IndexInfo{},
	}

	// Get column information
	columnQuery := `
		SELECT 
			column_name,
			data_type,
			character_maximum_length,
			numeric_precision,
			numeric_scale,
			is_nullable,
			column_default,
			CASE 
				WHEN column_default LIKE 'nextval%' THEN true
				ELSE false
			END as is_auto_increment
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := m.db.Query(columnQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var colInfo types.ColumnInfo
		var dataType string
		var charMaxLength, numericPrecision, numericScale sql.NullInt64
		var isNullable string
		var columnDefault sql.NullString
		var isAutoIncrement bool

		err := rows.Scan(
			&colInfo.Name,
			&dataType,
			&charMaxLength,
			&numericPrecision,
			&numericScale,
			&isNullable,
			&columnDefault,
			&isAutoIncrement,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		// Build full type string
		colInfo.Type = m.buildColumnType(dataType, charMaxLength, numericPrecision, numericScale)
		colInfo.Nullable = isNullable == "YES"
		colInfo.AutoIncrement = isAutoIncrement

		if columnDefault.Valid {
			// Clean up PostgreSQL default values
			defaultValue := columnDefault.String
			defaultValue = strings.TrimPrefix(defaultValue, "'")
			defaultValue = strings.TrimSuffix(defaultValue, "'::character varying")
			defaultValue = strings.TrimSuffix(defaultValue, "'::text")

			// Handle boolean defaults
			if dataType == "boolean" {
				if defaultValue == "true" {
					defaultValue = "TRUE"
				} else if defaultValue == "false" {
					defaultValue = "FALSE"
				}
			}

			colInfo.Default = defaultValue
		}

		tableInfo.Columns = append(tableInfo.Columns, colInfo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate columns: %w", err)
	}

	// Get primary key information
	pkQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
		ORDER BY kcu.ordinal_position
	`

	pkRows, err := m.db.Query(pkQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query primary keys: %w", err)
	}
	defer pkRows.Close()

	primaryKeys := make(map[string]bool)
	for pkRows.Next() {
		var columnName string
		if err := pkRows.Scan(&columnName); err != nil {
			return nil, fmt.Errorf("failed to scan primary key: %w", err)
		}
		primaryKeys[columnName] = true
	}

	// Update column info with primary key status
	for i := range tableInfo.Columns {
		if primaryKeys[tableInfo.Columns[i].Name] {
			tableInfo.Columns[i].PrimaryKey = true
		}
	}

	// Get unique constraint information
	uniqueQuery := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'UNIQUE'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
	`

	uniqueRows, err := m.db.Query(uniqueQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query unique constraints: %w", err)
	}
	defer uniqueRows.Close()

	uniqueColumns := make(map[string]bool)
	for uniqueRows.Next() {
		var columnName string
		if err := uniqueRows.Scan(&columnName); err != nil {
			return nil, fmt.Errorf("failed to scan unique constraint: %w", err)
		}
		uniqueColumns[columnName] = true
	}

	// Update column info with unique status
	for i := range tableInfo.Columns {
		if uniqueColumns[tableInfo.Columns[i].Name] {
			tableInfo.Columns[i].Unique = true
		}
	}

	// Get index information
	indexQuery := `
		SELECT 
			i.relname as index_name,
			idx.indisunique as is_unique,
			array_agg(a.attname ORDER BY array_position(idx.indkey, a.attnum)) as column_names
		FROM pg_index idx
		JOIN pg_class t ON t.oid = idx.indrelid
		JOIN pg_class i ON i.oid = idx.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(idx.indkey)
		WHERE t.relname = $1
			AND t.relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
			AND NOT idx.indisprimary
		GROUP BY i.relname, idx.indisunique
	`

	indexRows, err := m.db.Query(indexQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes: %w", err)
	}
	defer indexRows.Close()

	for indexRows.Next() {
		var indexInfo types.IndexInfo
		var columnNames string

		err := indexRows.Scan(&indexInfo.Name, &indexInfo.Unique, &columnNames)
		if err != nil {
			return nil, fmt.Errorf("failed to scan index info: %w", err)
		}

		// Parse column names from PostgreSQL array format
		columnNames = strings.TrimPrefix(columnNames, "{")
		columnNames = strings.TrimSuffix(columnNames, "}")
		indexInfo.Columns = strings.Split(columnNames, ",")

		tableInfo.Indexes = append(tableInfo.Indexes, indexInfo)
	}

	return tableInfo, nil
}

// buildColumnType builds the full column type string
func (m *PostgreSQLMigrator) buildColumnType(dataType string, charMaxLength, numericPrecision, numericScale sql.NullInt64) string {
	switch dataType {
	case "character varying":
		if charMaxLength.Valid {
			return fmt.Sprintf("VARCHAR(%d)", charMaxLength.Int64)
		}
		return "VARCHAR"
	case "character":
		if charMaxLength.Valid {
			return fmt.Sprintf("CHAR(%d)", charMaxLength.Int64)
		}
		return "CHAR"
	case "numeric":
		if numericPrecision.Valid && numericScale.Valid {
			return fmt.Sprintf("DECIMAL(%d,%d)", numericPrecision.Int64, numericScale.Int64)
		}
		return "DECIMAL"
	case "integer":
		return "INTEGER"
	case "bigint":
		return "BIGINT"
	case "smallint":
		return "SMALLINT"
	case "double precision":
		return "DOUBLE PRECISION"
	case "real":
		return "REAL"
	case "boolean":
		return "BOOLEAN"
	case "timestamp without time zone":
		return "TIMESTAMP"
	case "timestamp with time zone":
		return "TIMESTAMPTZ"
	case "date":
		return "DATE"
	case "time without time zone":
		return "TIME"
	case "time with time zone":
		return "TIMETZ"
	case "jsonb":
		return "JSONB"
	case "json":
		return "JSON"
	case "text":
		return "TEXT"
	case "bytea":
		return "BYTEA"
	default:
		return strings.ToUpper(dataType)
	}
}

// GenerateCreateTableSQL generates CREATE TABLE SQL
func (m *PostgreSQLMigrator) GenerateCreateTableSQL(schema *schema.Schema) (string, error) {
	return m.postgresqlDB.generateCreateTableSQL(schema)
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (m *PostgreSQLMigrator) GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", m.quote(tableName))
}

// GenerateAddColumnSQL generates ALTER TABLE ADD COLUMN SQL
func (m *PostgreSQLMigrator) GenerateAddColumnSQL(tableName string, column any) (string, error) {
	switch col := column.(type) {
	case types.ColumnInfo:
		columnDef := m.GenerateColumnDefinitionFromColumnInfo(col)
		sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", m.quote(tableName), columnDef)
		return sql, nil
	case schema.Field:
		columnDef := m.GenerateColumnDefinition(col)
		sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", m.quote(tableName), columnDef)
		return sql, nil
	default:
		return "", fmt.Errorf("unsupported column type: %T", column)
	}
}

// GenerateModifyColumnSQL generates ALTER TABLE ALTER COLUMN SQL for PostgreSQL
func (m *PostgreSQLMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	if change.NewColumn == nil {
		return nil, fmt.Errorf("new column definition is required")
	}

	var sqls []string
	tableName := m.quote(change.TableName)
	columnName := m.quote(change.NewColumn.Name)

	// Handle column rename
	if change.OldColumn != nil && change.OldColumn.Name != change.NewColumn.Name {
		sql := fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s",
			tableName, m.quote(change.OldColumn.Name), columnName)
		sqls = append(sqls, sql)
	}

	// Change column type
	newType := change.NewColumn.Type
	sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", tableName, columnName, newType)
	sqls = append(sqls, sql)

	// Change NULL/NOT NULL
	if change.NewColumn.Nullable {
		sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", tableName, columnName)
	} else {
		sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", tableName, columnName)
	}
	sqls = append(sqls, sql)

	// Change default value
	if change.NewColumn.Default != "" {
		sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s",
			tableName, columnName, change.NewColumn.Default)
	} else {
		sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", tableName, columnName)
	}
	sqls = append(sqls, sql)

	// Handle UNIQUE constraint
	if change.OldColumn != nil && change.OldColumn.Unique != change.NewColumn.Unique {
		if change.NewColumn.Unique {
			// Add unique constraint
			constraintName := fmt.Sprintf("uk_%s_%s", change.TableName, change.NewColumn.Name)
			sql = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s)",
				tableName, m.quote(constraintName), columnName)
		} else {
			// Drop unique constraint - need to find the constraint name
			sql = fmt.Sprintf(`
				DO $$
				DECLARE
					constraint_name text;
				BEGIN
					SELECT con.conname INTO constraint_name
					FROM pg_constraint con
					JOIN pg_class rel ON rel.oid = con.conrelid
					JOIN pg_namespace nsp ON nsp.oid = con.connamespace
					WHERE nsp.nspname = 'public'
					AND rel.relname = '%s'
					AND con.contype = 'u'
					AND EXISTS (
						SELECT 1 FROM unnest(con.conkey) WITH ORDINALITY AS c(attnum, ordinality)
						JOIN pg_attribute att ON att.attrelid = rel.oid AND att.attnum = c.attnum
						WHERE att.attname = '%s'
					);
					IF constraint_name IS NOT NULL THEN
						EXECUTE 'ALTER TABLE %s DROP CONSTRAINT ' || quote_ident(constraint_name);
					END IF;
				END $$;
			`, change.TableName, change.NewColumn.Name, tableName)
		}
		sqls = append(sqls, sql)
	}

	return sqls, nil
}

// GenerateDropColumnSQL generates ALTER TABLE DROP COLUMN SQL
func (m *PostgreSQLMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", m.quote(tableName), m.quote(columnName))
	return []string{sql}, nil
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (m *PostgreSQLMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = m.quote(col)
	}

	indexType := "INDEX"
	if unique {
		indexType = "UNIQUE INDEX"
	}

	return fmt.Sprintf("CREATE %s %s ON %s (%s)",
		indexType, m.quote(indexName), m.quote(tableName), strings.Join(quotedColumns, ", "))
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (m *PostgreSQLMigrator) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", m.quote(indexName))
}

// GenerateColumnDefinitionFromColumnInfo generates column definition from ColumnInfo
func (m *PostgreSQLMigrator) GenerateColumnDefinitionFromColumnInfo(column types.ColumnInfo) string {
	parts := []string{m.quote(column.Name), column.Type}

	if column.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	if !column.Nullable && !column.AutoIncrement {
		parts = append(parts, "NOT NULL")
	}

	if column.Unique && !column.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if column.Default != "" && !column.AutoIncrement {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", column.Default))
	}

	return strings.Join(parts, " ")
}

// GenerateColumnDefinition generates column definition from schema field
func (m *PostgreSQLMigrator) GenerateColumnDefinition(field schema.Field) string {
	column := types.ColumnInfo{
		Name:          field.GetColumnName(),
		Type:          m.postgresqlDB.mapFieldTypeToSQL(field.Type),
		Nullable:      field.Nullable,
		PrimaryKey:    field.PrimaryKey,
		Unique:        field.Unique,
		AutoIncrement: field.AutoIncrement,
	}

	if field.Default != nil {
		column.Default = m.postgresqlDB.formatDefaultValue(field.Default, field.Type)
	}

	// Handle SERIAL types for auto increment
	if field.AutoIncrement && field.PrimaryKey {
		if field.Type == schema.FieldTypeInt {
			column.Type = "SERIAL"
		} else if field.Type == schema.FieldTypeInt64 {
			column.Type = "BIGSERIAL"
		}
	}

	return m.GenerateColumnDefinitionFromColumnInfo(column)
}

// ApplyMigration applies a migration SQL statement
func (m *PostgreSQLMigrator) ApplyMigration(sql string) error {
	_, err := m.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}
	return nil
}

// quote quotes an identifier for PostgreSQL
func (m *PostgreSQLMigrator) quote(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

// MapFieldType maps a schema field to PostgreSQL column type
func (m *PostgreSQLMigrator) MapFieldType(field schema.Field) string {
	// Handle SERIAL types for auto increment
	if field.AutoIncrement && field.PrimaryKey {
		if field.Type == schema.FieldTypeInt {
			return "SERIAL"
		} else if field.Type == schema.FieldTypeInt64 {
			return "BIGSERIAL"
		}
	}
	return m.postgresqlDB.mapFieldTypeToSQL(field.Type)
}

// FormatDefaultValue formats a default value for PostgreSQL
func (m *PostgreSQLMigrator) FormatDefaultValue(value any) string {
	return m.postgresqlDB.formatDefaultValue(value, schema.FieldTypeString)
}

// ConvertFieldToColumnInfo converts a schema field to column info
func (m *PostgreSQLMigrator) ConvertFieldToColumnInfo(field schema.Field) *types.ColumnInfo {
	return &types.ColumnInfo{
		Name:          field.GetColumnName(),
		Type:          m.MapFieldType(field),
		Nullable:      field.Nullable,
		Default:       field.Default,
		PrimaryKey:    field.PrimaryKey,
		AutoIncrement: field.AutoIncrement,
		Unique:        field.Unique,
	}
}

// Additional wrapper methods to satisfy the types.DatabaseMigrator interface

// GenerateCreateTableSQL wraps to match the DatabaseMigrator interface
func (w *PostgreSQLMigratorWrapper) GenerateCreateTableSQL(schemaInterface any) (string, error) {
	s, ok := schemaInterface.(*schema.Schema)
	if !ok {
		return "", fmt.Errorf("expected *schema.Schema, got %T", schemaInterface)
	}
	return w.specific.GenerateCreateTableSQL(s)
}

// GenerateAddColumnSQL wraps to match the DatabaseMigrator interface
func (w *PostgreSQLMigratorWrapper) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return w.specific.GenerateAddColumnSQL(tableName, field)
}

// CompareSchema wraps the BaseMigrator's CompareSchema to match the DatabaseMigrator interface
func (w *PostgreSQLMigratorWrapper) CompareSchema(existingTable *types.TableInfo, desiredSchema any) (*types.MigrationPlan, error) {
	s, ok := desiredSchema.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("expected *schema.Schema, got %T", desiredSchema)
	}
	return w.BaseMigrator.CompareSchema(existingTable, s)
}
