package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// MySQLMigrator implements database-specific migration operations for MySQL
type MySQLMigrator struct {
	db   *sql.DB
	base *migration.BaseMigrator
}

// NewMySQLMigrator creates a new MySQL migrator
func NewMySQLMigrator(db *sql.DB) *MySQLMigrator {
	migrator := &MySQLMigrator{db: db}
	migrator.base = migration.NewBaseMigrator(migrator)
	return migrator
}

// GetDatabaseType returns the database type
func (m *MySQLMigrator) GetDatabaseType() string {
	return "mysql"
}

// CompareSchema uses the base migrator's shared logic
func (m *MySQLMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema *schema.Schema) (*types.MigrationPlan, error) {
	return m.base.CompareSchema(existingTable, desiredSchema)
}

// ConvertFieldToColumnInfo converts a schema field to ColumnInfo (DatabaseSpecificMigrator interface)
func (m *MySQLMigrator) ConvertFieldToColumnInfo(field schema.Field) *types.ColumnInfo {
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
func (m *MySQLMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	return m.base.GenerateMigrationSQL(plan)
}

// GenerateColumnDefinitionFromColumnInfo generates column definition SQL from ColumnInfo (DatabaseSpecificMigrator interface)
func (m *MySQLMigrator) GenerateColumnDefinitionFromColumnInfo(col types.ColumnInfo) string {
	parts := []string{col.Name, col.Type}

	// Handle nullable
	if !col.Nullable {
		parts = append(parts, "NOT NULL")
	}

	// Handle auto increment (must come before primary key in MySQL)
	if col.AutoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
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
func (m *MySQLMigrator) GetTables() ([]string, error) {
	query := "SHOW TABLES"
	
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
func (m *MySQLMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
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
func (m *MySQLMigrator) getColumns(tableName string) ([]types.ColumnInfo, error) {
	query := "DESCRIBE " + tableName
	
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []types.ColumnInfo
	for rows.Next() {
		var (
			field    string
			dataType string
			null     string
			key      string
			defaultVal sql.NullString
			extra    string
		)

		err := rows.Scan(&field, &dataType, &null, &key, &defaultVal, &extra)
		if err != nil {
			return nil, err
		}

		col := types.ColumnInfo{
			Name:          field,
			Type:          dataType,
			Nullable:      null == "YES",
			PrimaryKey:    key == "PRI",
			Unique:        key == "UNI",
			AutoIncrement: strings.Contains(extra, "auto_increment"),
		}

		if defaultVal.Valid {
			col.Default = defaultVal.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// getIndexes returns index information for a table
func (m *MySQLMigrator) getIndexes(tableName string) ([]types.IndexInfo, error) {
	query := "SHOW INDEX FROM " + tableName
	
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	indexMap := make(map[string]*types.IndexInfo)
	for rows.Next() {
		var (
			table      string
			nonUnique  int
			keyName    string
			seqInIndex int
			columnName string
			collation  sql.NullString
			cardinality sql.NullInt64
			subPart    sql.NullInt64
			packed     sql.NullString
			null       string
			indexType  string
			comment    string
			indexComment string
		)

		err := rows.Scan(&table, &nonUnique, &keyName, &seqInIndex, &columnName, 
			&collation, &cardinality, &subPart, &packed, &null, &indexType, &comment, &indexComment)
		if err != nil {
			return nil, err
		}

		// Skip primary key index as it's handled separately
		if keyName == "PRIMARY" {
			continue
		}

		if index, exists := indexMap[keyName]; exists {
			index.Columns = append(index.Columns, columnName)
		} else {
			indexMap[keyName] = &types.IndexInfo{
				Name:    keyName,
				Columns: []string{columnName},
				Unique:  nonUnique == 0,
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
func (m *MySQLMigrator) GenerateCreateTableSQL(s *schema.Schema) (string, error) {
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
		} else {
			columns = append(columns, fmt.Sprintf("INDEX (%s)", strings.Join(indexColumns, ", ")))
		}
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		s.TableName, strings.Join(columns, ",\n  ")), nil
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (m *MySQLMigrator) GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
}

// GenerateAddColumnSQL generates ALTER TABLE ADD COLUMN SQL
func (m *MySQLMigrator) GenerateAddColumnSQL(tableName string, fieldInterface interface{}) (string, error) {
	field, ok := fieldInterface.(schema.Field)
	if !ok {
		return "", fmt.Errorf("expected schema.Field, got %T", fieldInterface)
	}

	columnDef := m.generateColumnDefinition(field)
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef), nil
}

// GenerateModifyColumnSQL generates ALTER TABLE MODIFY COLUMN SQL for MySQL
func (m *MySQLMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	if change.NewColumn == nil {
		return nil, fmt.Errorf("new column info is required for modify operation")
	}
	
	columnDef := m.GenerateColumnDefinitionFromColumnInfo(*change.NewColumn)
	sql := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s", change.TableName, columnDef)
	return []string{sql}, nil
}

// GenerateDropColumnSQL generates ALTER TABLE DROP COLUMN SQL
func (m *MySQLMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName)
	return []string{sql}, nil
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (m *MySQLMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	indexType := ""
	if unique {
		indexType = "UNIQUE "
	}

	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
		indexType, indexName, tableName, strings.Join(columns, ", "))
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (m *MySQLMigrator) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX %s", indexName)
}

// ApplyMigration executes a migration SQL statement
func (m *MySQLMigrator) ApplyMigration(sql string) error {
	_, err := m.db.Exec(sql)
	return err
}

// mapFieldNamesToColumns converts field names to actual column names
func (m *MySQLMigrator) mapFieldNamesToColumns(s *schema.Schema, fieldNames []string) []string {
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
func (m *MySQLMigrator) generateColumnDefinition(field schema.Field) string {
	parts := []string{
		field.GetColumnName(), // Use mapped column name if available
		m.MapFieldType(field),
	}

	// Handle nullable
	if !field.Nullable && !field.PrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	// Handle auto increment (must come before PRIMARY KEY in MySQL)
	if field.AutoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
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

// MapFieldType maps schema field types to MySQL types (DatabaseSpecificMigrator interface)
func (m *MySQLMigrator) MapFieldType(field schema.Field) string {
	// Use database-specific type if provided
	if field.DbType != "" {
		return strings.TrimPrefix(field.DbType, "@db.")
	}

	switch field.Type {
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
	case schema.FieldTypeStringArray, schema.FieldTypeIntArray,
		schema.FieldTypeFloatArray, schema.FieldTypeBoolArray:
		return "JSON" // Store as JSON
	default:
		return "VARCHAR(255)"
	}
}

// FormatDefaultValue formats a default value for SQL (DatabaseSpecificMigrator interface)
func (m *MySQLMigrator) FormatDefaultValue(value interface{}) string {
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