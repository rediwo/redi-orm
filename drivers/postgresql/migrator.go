package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// PostgreSQLMigrator implements database-specific migration operations for PostgreSQL
type PostgreSQLMigrator struct {
	db   *sql.DB
	base *migration.BaseMigrator
}

// NewPostgreSQLMigrator creates a new PostgreSQL migrator
func NewPostgreSQLMigrator(db *sql.DB) *PostgreSQLMigrator {
	migrator := &PostgreSQLMigrator{db: db}
	migrator.base = migration.NewBaseMigrator(migrator)
	return migrator
}

// GetDatabaseType returns the database type
func (m *PostgreSQLMigrator) GetDatabaseType() string {
	return "postgresql"
}

// CompareSchema uses the base migrator's shared logic
func (m *PostgreSQLMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema *schema.Schema) (*types.MigrationPlan, error) {
	return m.base.CompareSchema(existingTable, desiredSchema)
}

// ConvertFieldToColumnInfo converts a schema field to ColumnInfo (DatabaseSpecificMigrator interface)
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

// GenerateMigrationSQL uses the base migrator's shared logic
func (m *PostgreSQLMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	return m.base.GenerateMigrationSQL(plan)
}

// GenerateColumnDefinitionFromColumnInfo generates column definition SQL from ColumnInfo (DatabaseSpecificMigrator interface)
func (m *PostgreSQLMigrator) GenerateColumnDefinitionFromColumnInfo(col types.ColumnInfo) string {
	parts := []string{col.Name, col.Type}

	// Handle nullable
	if !col.Nullable && !col.PrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	// Handle primary key
	if col.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	// Handle unique
	if col.Unique && !col.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	// Handle default value
	if col.Default != nil && !col.AutoIncrement {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", m.FormatDefaultValue(col.Default)))
	}

	return strings.Join(parts, " ")
}

// GetTables returns all tables in the database
func (m *PostgreSQLMigrator) GetTables() ([]string, error) {
	query := "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
	
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// GetTableInfo returns detailed information about a table
func (m *PostgreSQLMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	info := &types.TableInfo{
		Name: tableName,
	}

	// Get columns
	columns, err := m.getColumns(tableName)
	if err != nil {
		return nil, err
	}
	info.Columns = columns

	// Get indexes
	indexes, err := m.getIndexes(tableName)
	if err != nil {
		return nil, err
	}
	info.Indexes = indexes

	// Foreign keys are more complex, skip for now
	info.ForeignKeys = []types.ForeignKeyInfo{}

	return info, nil
}

// getColumns returns column information for a table
func (m *PostgreSQLMigrator) getColumns(tableName string) ([]types.ColumnInfo, error) {
	query := `
		SELECT 
			column_name, 
			data_type, 
			is_nullable, 
			column_default,
			CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN true ELSE false END as is_primary_key,
			CASE WHEN tc.constraint_type = 'UNIQUE' THEN true ELSE false END as is_unique
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu 
			ON c.table_name = kcu.table_name AND c.column_name = kcu.column_name
		LEFT JOIN information_schema.table_constraints tc 
			ON kcu.constraint_name = tc.constraint_name AND kcu.table_name = tc.table_name
		WHERE c.table_name = $1 AND c.table_schema = 'public'
		ORDER BY c.ordinal_position`
	
	rows, err := m.db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.ColumnInfo
	for rows.Next() {
		var (
			columnName   string
			dataType     string
			isNullable   string
			defaultVal   sql.NullString
			isPrimaryKey bool
			isUnique     bool
		)

		err := rows.Scan(&columnName, &dataType, &isNullable, &defaultVal, &isPrimaryKey, &isUnique)
		if err != nil {
			return nil, err
		}

		col := types.ColumnInfo{
			Name:       columnName,
			Type:       dataType,
			Nullable:   isNullable == "YES",
			PrimaryKey: isPrimaryKey,
			Unique:     isUnique,
		}

		if defaultVal.Valid {
			col.Default = defaultVal.String
			// Check for auto increment (PostgreSQL uses sequences)
			if strings.Contains(defaultVal.String, "nextval") {
				col.AutoIncrement = true
			}
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// getIndexes returns index information for a table
func (m *PostgreSQLMigrator) getIndexes(tableName string) ([]types.IndexInfo, error) {
	query := `
		SELECT 
			i.relname as index_name,
			ix.indisunique as is_unique,
			a.attname as column_name
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1 
		AND t.relkind = 'r'
		AND i.relname NOT LIKE 'pg_%'
		ORDER BY i.relname, a.attnum`
	
	rows, err := m.db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*types.IndexInfo)
	for rows.Next() {
		var (
			indexName  string
			isUnique   bool
			columnName string
		)

		err := rows.Scan(&indexName, &isUnique, &columnName)
		if err != nil {
			return nil, err
		}

		// Skip primary key indexes
		if strings.HasSuffix(indexName, "_pkey") {
			continue
		}

		if index, exists := indexMap[indexName]; exists {
			index.Columns = append(index.Columns, columnName)
		} else {
			indexMap[indexName] = &types.IndexInfo{
				Name:    indexName,
				Columns: []string{columnName},
				Unique:  isUnique,
			}
		}
	}

	var indexes []types.IndexInfo
	for _, index := range indexMap {
		indexes = append(indexes, *index)
	}

	return indexes, rows.Err()
}

// GenerateCreateTableSQL generates CREATE TABLE SQL
func (m *PostgreSQLMigrator) GenerateCreateTableSQL(s *schema.Schema) (string, error) {
	var columns []string

	// Get composite primary key fields
	primaryKeys := s.CompositeKey
	hasSinglePrimaryKey := false

	// Generate column definitions
	for _, field := range s.Fields {
		columnDef := m.generateColumnDefinition(field)
		columns = append(columns, columnDef)
		
		if field.PrimaryKey {
			hasSinglePrimaryKey = true
		}
	}

	// Add composite primary key if no single primary key exists
	if len(primaryKeys) > 0 && !hasSinglePrimaryKey {
		compositeColumns := m.mapFieldNamesToColumns(s, primaryKeys)
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(compositeColumns, ", ")))
	}

	// Add indexes
	for _, idx := range s.Indexes {
		indexColumns := m.mapFieldNamesToColumns(s, idx.Fields)
		if idx.Unique {
			columns = append(columns, fmt.Sprintf("UNIQUE (%s)", strings.Join(indexColumns, ", ")))
		}
		// Note: PostgreSQL non-unique indexes are usually created separately
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		s.TableName, strings.Join(columns, ",\n  ")), nil
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (m *PostgreSQLMigrator) GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
}

// GenerateAddColumnSQL generates ALTER TABLE ADD COLUMN SQL
func (m *PostgreSQLMigrator) GenerateAddColumnSQL(tableName string, fieldInterface interface{}) (string, error) {
	field, ok := fieldInterface.(schema.Field)
	if !ok {
		return "", fmt.Errorf("expected schema.Field, got %T", fieldInterface)
	}

	columnDef := m.generateColumnDefinition(field)
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef), nil
}

// GenerateModifyColumnSQL generates ALTER TABLE ALTER COLUMN SQL for PostgreSQL
func (m *PostgreSQLMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	if change.NewColumn == nil {
		return nil, fmt.Errorf("new column info is required for modify operation")
	}
	
	var sqls []string
	
	// PostgreSQL requires separate statements for different changes
	// 1. Change type
	if change.OldColumn != nil && change.OldColumn.Type != change.NewColumn.Type {
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", 
			change.TableName, change.ColumnName, change.NewColumn.Type)
		sqls = append(sqls, sql)
	}
	
	// 2. Change nullability
	if change.OldColumn != nil && change.OldColumn.Nullable != change.NewColumn.Nullable {
		if change.NewColumn.Nullable {
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", 
				change.TableName, change.ColumnName)
			sqls = append(sqls, sql)
		} else {
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", 
				change.TableName, change.ColumnName)
			sqls = append(sqls, sql)
		}
	}
	
	// 3. Change default
	if change.NewColumn.Default != nil {
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", 
			change.TableName, change.ColumnName, m.FormatDefaultValue(change.NewColumn.Default))
		sqls = append(sqls, sql)
	} else if change.OldColumn != nil && change.OldColumn.Default != nil {
		// Remove default if it was removed
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", 
			change.TableName, change.ColumnName)
		sqls = append(sqls, sql)
	}
	
	// If no specific changes were generated, create a comment
	if len(sqls) == 0 {
		comment := fmt.Sprintf("-- PostgreSQL: No changes detected for column %s in table %s", change.ColumnName, change.TableName)
		sqls = append(sqls, comment)
	}
	
	return sqls, nil
}

// GenerateDropColumnSQL generates ALTER TABLE DROP COLUMN SQL
func (m *PostgreSQLMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
	return []string{sql}, nil
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (m *PostgreSQLMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	indexType := ""
	if unique {
		indexType = "UNIQUE "
	}

	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
		indexType, indexName, tableName, strings.Join(columns, ", "))
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (m *PostgreSQLMigrator) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
}

// ApplyMigration executes a migration SQL statement
func (m *PostgreSQLMigrator) ApplyMigration(sql string) error {
	_, err := m.db.Exec(sql)
	return err
}

// mapFieldNamesToColumns converts field names to actual column names
func (m *PostgreSQLMigrator) mapFieldNamesToColumns(s *schema.Schema, fieldNames []string) []string {
	columnNames := make([]string, len(fieldNames))
	fieldMap := make(map[string]string)

	// Create field name to column name mapping
	for _, field := range s.Fields {
		fieldMap[field.Name] = field.GetColumnName()
	}

	// Convert field names to column names
	for i, fieldName := range fieldNames {
		if columnName, exists := fieldMap[fieldName]; exists {
			columnNames[i] = columnName
		} else {
			columnNames[i] = fieldName // Fallback to field name
		}
	}

	return columnNames
}

// generateColumnDefinition generates column definition SQL
func (m *PostgreSQLMigrator) generateColumnDefinition(field schema.Field) string {
	parts := []string{
		field.GetColumnName(), // Use mapped column name if available
		m.MapFieldType(field),
	}

	// Handle nullable
	if !field.Nullable && !field.PrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	// Handle single primary key in column definition
	if field.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	// Handle unique
	if field.Unique && !field.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	// Handle default value
	if field.Default != nil && !field.AutoIncrement {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", m.FormatDefaultValue(field.Default)))
	}

	return strings.Join(parts, " ")
}

// MapFieldType maps schema field types to PostgreSQL types (DatabaseSpecificMigrator interface)
func (m *PostgreSQLMigrator) MapFieldType(field schema.Field) string {
	// Use database-specific type if provided
	if field.DbType != "" {
		return strings.TrimPrefix(field.DbType, "@db.")
	}

	switch field.Type {
	case schema.FieldTypeString:
		return "VARCHAR(255)"
	case schema.FieldTypeInt:
		return "INTEGER"
	case schema.FieldTypeInt64:
		if field.AutoIncrement {
			return "BIGSERIAL"
		}
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
	case schema.FieldTypeStringArray, schema.FieldTypeIntArray,
		schema.FieldTypeFloatArray, schema.FieldTypeBoolArray:
		return "JSONB" // Store as JSONB
	default:
		return "VARCHAR(255)"
	}
}

// FormatDefaultValue formats a default value for SQL (DatabaseSpecificMigrator interface)
func (m *PostgreSQLMigrator) FormatDefaultValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Check for special values
		if strings.ToLower(v) == "now()" || strings.ToLower(v) == "current_timestamp" {
			return "CURRENT_TIMESTAMP"
		}
		// Check for functions
		if strings.Contains(v, "(") && strings.Contains(v, ")") {
			return v
		}
		return fmt.Sprintf("'%s'", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}