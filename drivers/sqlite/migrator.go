package sqlite

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// SQLiteMigrator implements types.DatabaseSpecificMigrator for SQLite
type SQLiteMigrator struct {
	db       *sql.DB
	sqliteDB *SQLiteDB
}

// normalizeDefaultValue normalizes a default value for comparison
// This handles SQLite's quirks with default values
func (m *SQLiteMigrator) normalizeDefaultValue(value string) string {
	// Remove surrounding quotes and parentheses
	value = strings.TrimSpace(value)

	// Handle CURRENT_TIMESTAMP variations
	if strings.Contains(strings.ToUpper(value), "CURRENT_TIMESTAMP") {
		return "CURRENT_TIMESTAMP"
	}

	// Remove excessive quotes that may have accumulated
	for strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) > 2 {
		// Check if it's a properly quoted string (not accumulated quotes)
		inner := value[1 : len(value)-1]
		if !strings.Contains(inner, "'") || strings.Count(inner, "''")%2 == 0 {
			break
		}
		value = inner
	}

	// Handle boolean values
	if value == "'0'" || value == "0" || value == "'false'" || value == "false" {
		return "0"
	}
	if value == "'1'" || value == "1" || value == "'true'" || value == "true" {
		return "1"
	}

	return value
}

// SQLiteMigratorWrapper wraps SQLiteMigrator with BaseMigrator to implement types.DatabaseMigrator
type SQLiteMigratorWrapper struct {
	*base.BaseMigrator
	specific *SQLiteMigrator
}

// NewSQLiteMigrator creates a new SQLite migrator that implements types.DatabaseMigrator
func NewSQLiteMigrator(db *sql.DB, sqliteDB *SQLiteDB) types.DatabaseMigrator {
	specific := &SQLiteMigrator{
		db:       db,
		sqliteDB: sqliteDB,
	}
	wrapper := &SQLiteMigratorWrapper{
		specific: specific,
	}
	wrapper.BaseMigrator = base.NewBaseMigrator(specific)
	return wrapper
}

// GetTables returns all table names
func (m *SQLiteMigrator) GetTables() ([]string, error) {
	query := `
		SELECT name 
		FROM sqlite_master 
		WHERE type='table' 
			AND name NOT LIKE 'sqlite_%'
		ORDER BY name
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	return tables, nil
}

// GetTableInfo returns table information
func (m *SQLiteMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	tableInfo := &types.TableInfo{
		Name:        tableName,
		Columns:     []types.ColumnInfo{},
		Indexes:     []types.IndexInfo{},
		ForeignKeys: []types.ForeignKeyInfo{},
	}

	// Get column information using PRAGMA table_info
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dtype string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dtype, &notNull, &defaultValue, &pk); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		column := types.ColumnInfo{
			Name:          name,
			Type:          dtype,
			Nullable:      notNull == 0 && pk == 0, // PRIMARY KEY is implicitly NOT NULL
			PrimaryKey:    pk > 0,
			AutoIncrement: false, // Will be determined below
			Unique:        false, // Will be determined from indexes
		}

		if defaultValue.Valid {
			// Normalize the default value to prevent accumulation of quotes
			column.Default = m.normalizeDefaultValue(defaultValue.String)
		}

		// Check for AUTOINCREMENT
		// In SQLite, INTEGER PRIMARY KEY columns are automatically ROWID aliases
		// and behave like AUTOINCREMENT even without the keyword
		if pk > 0 && strings.Contains(strings.ToUpper(dtype), "INTEGER") {
			column.AutoIncrement = true
		}

		tableInfo.Columns = append(tableInfo.Columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %w", err)
	}

	// Get index information
	indexRows, err := m.db.Query(fmt.Sprintf("PRAGMA index_list(%s)", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to get index list: %w", err)
	}
	defer indexRows.Close()

	// Collect index names first to avoid nested queries
	type indexData struct {
		name   string
		unique bool
		origin string
	}
	var indexes []indexData

	for indexRows.Next() {
		var seq int
		var indexName string
		var unique int
		var origin string
		var partial int

		if err := indexRows.Scan(&seq, &indexName, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("failed to scan index info: %w", err)
		}

		// Skip auto-generated indexes for primary keys only
		// origin="c" means created by CREATE INDEX
		// origin="u" means created by UNIQUE constraint
		// origin="pk" means created by PRIMARY KEY
		if origin == "pk" {
			continue
		}

		indexes = append(indexes, indexData{
			name:   indexName,
			unique: unique == 1,
			origin: origin,
		})
	}

	if err := indexRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating indexes: %w", err)
	}

	// Now get columns for each index
	for _, idx := range indexes {
		// Get columns for this index
		columns := []string{}
		indexColRows, err := m.db.Query("PRAGMA index_info(" + idx.name + ")")
		if err != nil {
			return nil, fmt.Errorf("failed to get index columns for %s: %w", idx.name, err)
		}

		for indexColRows.Next() {
			var seqno int
			var cid int
			var colName string
			if err := indexColRows.Scan(&seqno, &cid, &colName); err != nil {
				indexColRows.Close()
				return nil, fmt.Errorf("failed to scan index column for %s: %w", idx.name, err)
			}
			columns = append(columns, colName)
		}
		indexColRows.Close()

		if err := indexColRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating index columns for %s: %w", idx.name, err)
		}

		// Update column Unique flag if this is a unique index with one column
		if idx.unique && len(columns) == 1 {
			for i := range tableInfo.Columns {
				if tableInfo.Columns[i].Name == columns[0] {
					tableInfo.Columns[i].Unique = true
					break
				}
			}
		}

		index := types.IndexInfo{
			Name:    idx.name,
			Columns: columns,
			Unique:  idx.unique,
		}
		tableInfo.Indexes = append(tableInfo.Indexes, index)
	}

	// Get foreign key information
	fkRows, err := m.db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var id int
		var seq int
		var table string
		var from string
		var to string
		var onUpdate string
		var onDelete string
		var match string

		if err := fkRows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		fk := types.ForeignKeyInfo{
			Name:             fmt.Sprintf("fk_%s_%s_%s", tableName, from, table),
			Column:           from,
			ReferencedTable:  table,
			ReferencedColumn: to,
			OnUpdate:         onUpdate,
			OnDelete:         onDelete,
		}
		tableInfo.ForeignKeys = append(tableInfo.ForeignKeys, fk)
	}

	if err := fkRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating foreign keys: %w", err)
	}

	return tableInfo, nil
}

// GenerateCreateTableSQL generates CREATE TABLE SQL from schema (for DatabaseSpecificMigrator)
func (m *SQLiteMigrator) GenerateCreateTableSQL(s *schema.Schema) (string, error) {
	// Reuse the existing implementation from SQLiteDB
	return m.sqliteDB.generateCreateTableSQL(s)
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (m *SQLiteMigrator) GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
}

// GenerateAddColumnSQL generates ADD COLUMN SQL
func (m *SQLiteMigrator) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	f, ok := field.(schema.Field)
	if !ok {
		return "", fmt.Errorf("expected schema.Field, got %T", field)
	}

	columnDef, err := m.sqliteDB.generateColumnSQL(f)
	if err != nil {
		return "", fmt.Errorf("failed to generate column definition: %w", err)
	}

	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef), nil
}

// GenerateModifyColumnSQL generates SQL to modify a column (not directly supported in SQLite)
func (m *SQLiteMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	// SQLite doesn't support direct column modification
	// We need to recreate the table with the new schema

	// Get current table info
	tableInfo, err := m.GetTableInfo(change.TableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}

	var sqls []string

	// Generate temporary table name
	tempTableName := fmt.Sprintf("%s_temp_%d", change.TableName, time.Now().Unix())

	// Build CREATE TABLE statement for the temporary table
	var columnDefs []string
	for _, col := range tableInfo.Columns {
		if col.Name == change.ColumnName {
			// Use the new column definition
			if change.NewColumn != nil {
				columnDef := m.GenerateColumnDefinitionFromColumnInfo(*change.NewColumn)
				columnDefs = append(columnDefs, columnDef)
			}
		} else {
			// Keep existing column definition
			columnDef := m.GenerateColumnDefinitionFromColumnInfo(col)
			columnDefs = append(columnDefs, columnDef)
		}
	}

	// Create temporary table with new schema
	createSQL := fmt.Sprintf("CREATE TABLE %s (\n  %s\n)",
		tempTableName,
		strings.Join(columnDefs, ",\n  "))
	sqls = append(sqls, createSQL)

	// Copy data from old table to new table
	// Build column lists for the INSERT
	var selectColumns []string
	var insertColumns []string

	for _, col := range tableInfo.Columns {
		if col.Name == change.ColumnName {
			// Handle column name changes or type conversions
			if change.NewColumn != nil && change.NewColumn.Name != "" {
				insertColumns = append(insertColumns, change.NewColumn.Name)
				// SQLite will attempt automatic type conversion
				selectColumns = append(selectColumns, col.Name)
			}
		} else {
			insertColumns = append(insertColumns, col.Name)
			selectColumns = append(selectColumns, col.Name)
		}
	}

	copySQL := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s",
		tempTableName,
		strings.Join(insertColumns, ", "),
		strings.Join(selectColumns, ", "),
		change.TableName)
	sqls = append(sqls, copySQL)

	// Drop the old table
	dropSQL := fmt.Sprintf("DROP TABLE %s", change.TableName)
	sqls = append(sqls, dropSQL)

	// Rename temporary table to original name
	renameSQL := fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tempTableName, change.TableName)
	sqls = append(sqls, renameSQL)

	// Recreate indexes
	for _, idx := range tableInfo.Indexes {
		// Skip auto-generated indexes as they'll be recreated automatically
		if strings.HasPrefix(idx.Name, "sqlite_autoindex_") {
			continue
		}

		// Update column names in index if the modified column is part of it
		var indexColumns []string
		for _, col := range idx.Columns {
			if col == change.ColumnName && change.NewColumn != nil && change.NewColumn.Name != "" {
				indexColumns = append(indexColumns, change.NewColumn.Name)
			} else {
				indexColumns = append(indexColumns, col)
			}
		}

		indexSQL := m.GenerateCreateIndexSQL(change.TableName, idx.Name, indexColumns, idx.Unique)
		sqls = append(sqls, indexSQL)
	}

	// Recreate foreign keys (if any)
	if len(tableInfo.ForeignKeys) > 0 {
		// SQLite doesn't support adding foreign keys after table creation
		// They need to be included in the CREATE TABLE statement
		// This is a limitation we'll document
		sqls = append(sqls, fmt.Sprintf("-- Note: Foreign keys need to be manually recreated for table %s", change.TableName))
	}

	return sqls, nil
}

// GenerateDropColumnSQL generates DROP COLUMN SQL
func (m *SQLiteMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	// SQLite 3.35.0+ supports DROP COLUMN
	return []string{
		fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, columnName),
	}, nil
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (m *SQLiteMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	uniqueStr := ""
	if unique {
		uniqueStr = "UNIQUE "
	}
	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
		uniqueStr, indexName, tableName, strings.Join(columns, ", "))
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (m *SQLiteMigrator) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
}

// ApplyMigration executes a migration SQL
func (m *SQLiteMigrator) ApplyMigration(sql string) error {
	if m.sqliteDB != nil {
		_, err := m.sqliteDB.Exec(sql)
		return err
	}
	_, err := m.db.Exec(sql)
	return err
}

// GetDatabaseType returns the database type
func (m *SQLiteMigrator) GetDatabaseType() string {
	return "sqlite"
}

// MapFieldType maps schema field types to SQLite types
func (m *SQLiteMigrator) MapFieldType(field schema.Field) string {
	return m.sqliteDB.mapFieldTypeToSQL(field.Type)
}

// FormatDefaultValue formats a default value for SQLite
func (m *SQLiteMigrator) FormatDefaultValue(value any) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", value)
	}
}

// GenerateColumnDefinitionFromColumnInfo generates column definition from ColumnInfo
func (m *SQLiteMigrator) GenerateColumnDefinitionFromColumnInfo(col types.ColumnInfo) string {
	parts := []string{col.Name, col.Type}

	if col.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
		if col.AutoIncrement {
			parts = append(parts, "AUTOINCREMENT")
		}
	}

	if !col.Nullable && !col.PrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	if col.Unique && !col.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if col.Default != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", m.FormatDefaultValue(col.Default)))
	}

	return strings.Join(parts, " ")
}

// ConvertFieldToColumnInfo converts a schema field to column info
func (m *SQLiteMigrator) ConvertFieldToColumnInfo(field schema.Field) *types.ColumnInfo {
	// Normalize the default value for SQLite
	var normalizedDefault any = field.Default
	if field.Default != nil {
		if defaultStr, ok := field.Default.(string); ok {
			normalizedDefault = m.normalizeDefaultValue(defaultStr)
		} else if defaultBool, ok := field.Default.(bool); ok {
			// Convert boolean to SQLite representation
			if defaultBool {
				normalizedDefault = "1"
			} else {
				normalizedDefault = "0"
			}
		}
	}

	return &types.ColumnInfo{
		Name:          field.GetColumnName(),
		Type:          m.MapFieldType(field),
		Nullable:      field.Nullable,
		Default:       normalizedDefault,
		PrimaryKey:    field.PrimaryKey,
		AutoIncrement: field.AutoIncrement,
		Unique:        field.Unique,
	}
}

// IsSystemIndex checks if an index is a system-generated index in SQLite
func (m *SQLiteMigrator) IsSystemIndex(indexName string) bool {
	lower := strings.ToLower(indexName)

	// SQLite system index patterns:
	// - sqlite_autoindex_*: automatically created indexes for UNIQUE and PRIMARY KEY
	// - sqlite_*: other system indexes
	// - pk_*: primary key indexes
	return strings.HasPrefix(lower, "sqlite_") ||
		strings.HasPrefix(lower, "pk_") ||
		strings.Contains(lower, "primary")
}

// IsSystemTable checks if a table is a system table in SQLite
func (m *SQLiteMigrator) IsSystemTable(tableName string) bool {
	lower := strings.ToLower(tableName)

	// SQLite system table patterns:
	// - sqlite_* tables are system tables
	// - Tables that store SQLite's internal data
	return strings.HasPrefix(lower, "sqlite_") ||
		lower == "sqlite_sequence" ||
		lower == "sqlite_stat1" ||
		lower == "sqlite_stat2" ||
		lower == "sqlite_stat3" ||
		lower == "sqlite_stat4"
}

// ParseDefaultValue parses SQLite-specific default values to normalized form
func (m *SQLiteMigrator) ParseDefaultValue(value any, fieldType schema.FieldType) any {
	if value == nil {
		return nil
	}

	// Convert to string for processing
	strVal := fmt.Sprintf("%v", value)

	// Strip surrounding quotes that SQLite might add
	strVal = strings.Trim(strVal, `'"`)

	// Normalize common SQLite default functions
	upperVal := strings.ToUpper(strVal)
	switch {
	case upperVal == "CURRENT_TIMESTAMP" || upperVal == "DATETIME('NOW')" || upperVal == "DATETIME(NOW)" || upperVal == "NOW()":
		return "CURRENT_TIMESTAMP"
	case upperVal == "NULL":
		return nil
	}

	// Type-specific parsing
	switch fieldType {
	case schema.FieldTypeBool:
		// SQLite stores booleans as 0/1
		if strVal == "0" || strVal == "false" || strVal == "FALSE" {
			return false
		} else if strVal == "1" || strVal == "true" || strVal == "TRUE" {
			return true
		}
	case schema.FieldTypeInt, schema.FieldTypeInt64:
		// Try to parse as integer
		if i, err := strconv.ParseInt(strVal, 10, 64); err == nil {
			return i
		}
	case schema.FieldTypeFloat, schema.FieldTypeDecimal:
		// Try to parse as float
		if f, err := strconv.ParseFloat(strVal, 64); err == nil {
			return f
		}
	}

	// Return as string for other types
	return strVal
}

// NormalizeDefaultToPrismaFunction converts normalized defaults to Prisma function names
func (m *SQLiteMigrator) NormalizeDefaultToPrismaFunction(value any, fieldType schema.FieldType) (string, bool) {
	if value == nil {
		return "", false
	}

	strVal, ok := value.(string)
	if !ok {
		return "", false
	}

	switch strVal {
	case "CURRENT_TIMESTAMP":
		return "now", true
		// SQLite doesn't have built-in UUID generation
		// Auto-increment is handled differently in SQLite
	}

	return "", false
}

// MapDatabaseTypeToFieldType converts SQLite column types to schema field types
func (m *SQLiteMigrator) MapDatabaseTypeToFieldType(dbType string) schema.FieldType {
	// Normalize the type to lowercase and remove size specifications
	dbType = strings.ToLower(dbType)

	// Remove size specifications like (255) or (10,2)
	if idx := strings.Index(dbType, "("); idx != -1 {
		dbType = dbType[:idx]
	}

	// Trim any whitespace
	dbType = strings.TrimSpace(dbType)

	// SQLite type mappings
	switch dbType {
	// Integer types
	case "integer", "int", "tinyint", "smallint", "mediumint", "bigint":
		return schema.FieldTypeInt
	case "int2", "int8":
		return schema.FieldTypeInt

	// Floating point types
	case "real", "double", "double precision", "float":
		return schema.FieldTypeFloat
	case "decimal", "numeric":
		return schema.FieldTypeDecimal

	// String types
	case "text", "varchar", "character", "char", "clob":
		return schema.FieldTypeString
	case "nvarchar", "nchar":
		return schema.FieldTypeString

	// Boolean types
	case "boolean", "bool":
		return schema.FieldTypeBool

	// Date/Time types
	case "date", "datetime", "timestamp":
		return schema.FieldTypeDateTime

	// Binary types
	case "blob":
		return schema.FieldTypeBinary

	// Default to string for unknown types
	default:
		return schema.FieldTypeString
	}
}

// IsPrimaryKeyIndex checks if an index name indicates it's a primary key index
func (m *SQLiteMigrator) IsPrimaryKeyIndex(indexName string) bool {
	lower := strings.ToLower(indexName)
	return strings.HasPrefix(lower, "sqlite_autoindex_") && strings.Contains(lower, "_pk") ||
		strings.Contains(lower, "primary") ||
		strings.HasPrefix(lower, "pk_")
}

// Additional wrapper methods to satisfy the types.DatabaseMigrator interface

// GenerateCreateTableSQL wraps to match the DatabaseMigrator interface
func (w *SQLiteMigratorWrapper) GenerateCreateTableSQL(schemaInterface any) (string, error) {
	s, ok := schemaInterface.(*schema.Schema)
	if !ok {
		return "", fmt.Errorf("expected *schema.Schema, got %T", schemaInterface)
	}
	return w.specific.GenerateCreateTableSQL(s)
}

// GenerateAddColumnSQL wraps to match the DatabaseMigrator interface
func (w *SQLiteMigratorWrapper) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return w.specific.GenerateAddColumnSQL(tableName, field)
}

// CompareSchema wraps the BaseMigrator's CompareSchema to match the DatabaseMigrator interface
func (w *SQLiteMigratorWrapper) CompareSchema(existingTable *types.TableInfo, desiredSchema any) (*types.MigrationPlan, error) {
	s, ok := desiredSchema.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("expected *schema.Schema, got %T", desiredSchema)
	}
	return w.BaseMigrator.CompareSchema(existingTable, s)
}

// IsSystemTable delegates to the specific implementation
func (w *SQLiteMigratorWrapper) IsSystemTable(tableName string) bool {
	return w.specific.IsSystemTable(tableName)
}

// GetSpecific returns the specific migrator implementation
func (w *SQLiteMigratorWrapper) GetSpecific() types.DatabaseSpecificMigrator {
	return w.specific
}
