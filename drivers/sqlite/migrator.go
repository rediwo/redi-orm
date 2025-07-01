package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// SQLiteMigrator implements database-specific migration operations for SQLite
type SQLiteMigrator struct {
	db *sql.DB
}

// NewSQLiteMigrator creates a new SQLite migrator
func NewSQLiteMigrator(db *sql.DB) *SQLiteMigrator {
	return &SQLiteMigrator{db: db}
}

// GetDatabaseType returns the database type
func (m *SQLiteMigrator) GetDatabaseType() string {
	return "sqlite"
}

// CompareSchema compares existing table with desired schema (placeholder implementation)
func (m *SQLiteMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema *schema.Schema) (*types.MigrationPlan, error) {
	plan := &types.MigrationPlan{
		CreateTables:  []string{},
		AddColumns:    []types.ColumnChange{},
		ModifyColumns: []types.ColumnChange{},
		DropColumns:   []types.ColumnChange{},
		AddIndexes:    []types.IndexChange{},
		DropIndexes:   []types.IndexChange{},
	}

	// For now, this is a placeholder - full schema comparison would be implemented here
	// This would compare each field, index, etc. and determine what needs to be changed
	
	return plan, nil
}

// GenerateMigrationSQL generates SQL statements for a migration plan (placeholder implementation)
func (m *SQLiteMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	var sqlStatements []string

	// Generate CREATE TABLE statements
	for _, table := range plan.CreateTables {
		sqlStatements = append(sqlStatements, fmt.Sprintf("CREATE TABLE %s (...)", table))
	}

	// Generate ADD COLUMN statements
	for _, change := range plan.AddColumns {
		sqlStatements = append(sqlStatements, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", change.TableName, change.ColumnName))
	}

	// Note: SQLite doesn't support DROP COLUMN or MODIFY COLUMN directly
	// These would require recreating the table

	return sqlStatements, nil
}

// GetTables returns all tables in the database
func (m *SQLiteMigrator) GetTables() ([]string, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'redi_migrations'`

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
func (m *SQLiteMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
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

	// SQLite foreign keys are more complex to introspect, skip for now
	info.ForeignKeys = []types.ForeignKeyInfo{}

	return info, nil
}

// getColumns returns column information for a table
func (m *SQLiteMigrator) getColumns(tableName string) ([]types.ColumnInfo, error) {
	rows, err := m.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.ColumnInfo
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &primaryKey)
		if err != nil {
			return nil, err
		}

		col := types.ColumnInfo{
			Name:       name,
			Type:       dataType,
			Nullable:   notNull == 0,
			PrimaryKey: primaryKey > 0,
		}

		if defaultVal.Valid {
			col.Default = defaultVal.String
		}

		// Check for autoincrement
		if primaryKey > 0 && strings.Contains(strings.ToUpper(dataType), "INTEGER") {
			col.AutoIncrement = true
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// getIndexes returns index information for a table
func (m *SQLiteMigrator) getIndexes(tableName string) ([]types.IndexInfo, error) {
	query := `SELECT name FROM sqlite_master WHERE type='index' AND tbl_name=? AND name NOT LIKE 'sqlite_%'`

	rows, err := m.db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []types.IndexInfo
	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			return nil, err
		}

		// Get index info
		infoRows, err := m.db.Query(fmt.Sprintf("PRAGMA index_info(%s)", indexName))
		if err != nil {
			return nil, err
		}

		var columns []string
		for infoRows.Next() {
			var (
				seqno int
				cid   int
				name  string
			)
			if err := infoRows.Scan(&seqno, &cid, &name); err != nil {
				infoRows.Close()
				return nil, err
			}
			columns = append(columns, name)
		}
		infoRows.Close()

		if len(columns) > 0 {
			indexes = append(indexes, types.IndexInfo{
				Name:    indexName,
				Columns: columns,
				// TODO: Determine if unique
			})
		}
	}

	return indexes, rows.Err()
}

// GenerateCreateTableSQL generates CREATE TABLE SQL for SQLite
func (m *SQLiteMigrator) GenerateCreateTableSQL(s *schema.Schema) (string, error) {

	var columns []string
	var primaryKeys []string

	for _, field := range s.Fields {
		col := m.generateColumnDefinition(field)
		columns = append(columns, col)

		if field.PrimaryKey && len(s.CompositeKey) == 0 {
			// Single primary key is handled in column definition
		} else if field.PrimaryKey {
			primaryKeys = append(primaryKeys, field.Name)
		}
	}

	// Handle composite primary key
	if len(s.CompositeKey) > 0 {
		// Convert field names to column names for composite key
		compositeColumns := m.mapFieldNamesToColumns(s, s.CompositeKey)
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(compositeColumns, ", ")))
	} else if len(primaryKeys) > 1 {
		// Convert field names to column names for regular composite key
		compositeColumns := m.mapFieldNamesToColumns(s, primaryKeys)
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(compositeColumns, ", ")))
	}

	// Add indexes
	for _, idx := range s.Indexes {
		// Convert field names to column names for indexes
		indexColumns := m.mapFieldNamesToColumns(s, idx.Fields)
		if idx.Unique {
			columns = append(columns, fmt.Sprintf("UNIQUE (%s)", strings.Join(indexColumns, ", ")))
		}
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		s.TableName, strings.Join(columns, ",\n  ")), nil
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (m *SQLiteMigrator) GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
}

// GenerateAddColumnSQL generates ALTER TABLE ADD COLUMN SQL
func (m *SQLiteMigrator) GenerateAddColumnSQL(tableName string, fieldInterface interface{}) (string, error) {
	field, ok := fieldInterface.(schema.Field)
	if !ok {
		return "", fmt.Errorf("expected schema.Field, got %T", fieldInterface)
	}

	columnDef := m.generateColumnDefinition(field)
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef), nil
}

// GenerateDropColumnSQL generates ALTER TABLE DROP COLUMN SQL
func (m *SQLiteMigrator) GenerateDropColumnSQL(tableName, columnName string) string {
	// SQLite doesn't support DROP COLUMN directly
	return fmt.Sprintf("-- SQLite: DROP COLUMN %s requires table recreation", columnName)
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (m *SQLiteMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	indexType := ""
	if unique {
		indexType = "UNIQUE "
	}

	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
		indexType, indexName, tableName, strings.Join(columns, ", "))
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (m *SQLiteMigrator) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
}

// ApplyMigration executes a migration SQL statement
func (m *SQLiteMigrator) ApplyMigration(sql string) error {
	_, err := m.db.Exec(sql)
	return err
}

// mapFieldNamesToColumns converts field names to actual column names
func (m *SQLiteMigrator) mapFieldNamesToColumns(s *schema.Schema, fieldNames []string) []string {
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
func (m *SQLiteMigrator) generateColumnDefinition(field schema.Field) string {
	parts := []string{
		field.GetColumnName(), // Use mapped column name if available
		m.mapFieldType(field),
	}

	// Handle single primary key in column definition
	if field.PrimaryKey && field.AutoIncrement {
		parts = append(parts, "PRIMARY KEY AUTOINCREMENT")
	} else if field.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	if !field.Nullable && !field.PrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	if field.Unique && !field.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if field.Default != nil && !field.AutoIncrement {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", m.formatDefaultValue(field.Default)))
	}

	return strings.Join(parts, " ")
}

// mapFieldType maps schema field types to SQLite types
func (m *SQLiteMigrator) mapFieldType(field schema.Field) string {
	// Use database-specific type if provided
	if field.DbType != "" {
		return strings.TrimPrefix(field.DbType, "@db.")
	}

	switch field.Type {
	case schema.FieldTypeString:
		return "TEXT"
	case schema.FieldTypeInt, schema.FieldTypeInt64:
		return "INTEGER"
	case schema.FieldTypeFloat:
		return "REAL"
	case schema.FieldTypeBool:
		return "INTEGER"
	case schema.FieldTypeDateTime:
		return "DATETIME"
	case schema.FieldTypeJSON:
		return "TEXT"
	case schema.FieldTypeDecimal:
		return "DECIMAL"
	case schema.FieldTypeStringArray, schema.FieldTypeIntArray,
		schema.FieldTypeFloatArray, schema.FieldTypeBoolArray:
		return "TEXT" // Store as JSON
	default:
		return "TEXT"
	}
}

// formatDefaultValue formats a default value for SQL
func (m *SQLiteMigrator) formatDefaultValue(value interface{}) string {
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
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}
