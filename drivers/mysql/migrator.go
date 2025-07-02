package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// MySQLMigrator implements DatabaseSpecificMigrator for MySQL
type MySQLMigrator struct {
	db      *sql.DB
	mysqlDB *MySQLDB
}

// MySQLMigratorWrapper wraps MySQLMigrator with BaseMigrator to implement types.DatabaseMigrator
type MySQLMigratorWrapper struct {
	*migration.BaseMigrator
	specific *MySQLMigrator
}

// NewMySQLMigrator creates a new MySQL migrator that implements types.DatabaseMigrator
func NewMySQLMigrator(db *sql.DB, mysqlDB *MySQLDB) types.DatabaseMigrator {
	specific := &MySQLMigrator{
		db:      db,
		mysqlDB: mysqlDB,
	}
	wrapper := &MySQLMigratorWrapper{
		specific: specific,
	}
	wrapper.BaseMigrator = migration.NewBaseMigrator(specific)
	return wrapper
}

// GetTables returns all table names
func (m *MySQLMigrator) GetTables() ([]string, error) {
	query := `
		SELECT TABLE_NAME 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME
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
func (m *MySQLMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	tableInfo := &types.TableInfo{
		Name:        tableName,
		Columns:     []types.ColumnInfo{},
		Indexes:     []types.IndexInfo{},
		ForeignKeys: []types.ForeignKeyInfo{},
	}

	// Get column information
	query := `
		SELECT 
			COLUMN_NAME,
			DATA_TYPE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			COLUMN_KEY,
			EXTRA,
			CHARACTER_MAXIMUM_LENGTH,
			NUMERIC_PRECISION,
			NUMERIC_SCALE
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := m.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get column info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var columnName string
		var dataType string
		var isNullable string
		var columnDefault sql.NullString
		var columnKey string
		var extra string
		var charMaxLength sql.NullInt64
		var numPrecision sql.NullInt64
		var numScale sql.NullInt64

		if err := rows.Scan(
			&columnName,
			&dataType,
			&isNullable,
			&columnDefault,
			&columnKey,
			&extra,
			&charMaxLength,
			&numPrecision,
			&numScale,
		); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}

		column := types.ColumnInfo{
			Name:          columnName,
			Type:          m.formatColumnType(dataType, charMaxLength, numPrecision, numScale),
			Nullable:      isNullable == "YES",
			PrimaryKey:    columnKey == "PRI",
			AutoIncrement: strings.Contains(extra, "auto_increment"),
			Unique:        columnKey == "UNI",
		}

		if columnDefault.Valid {
			column.Default = columnDefault.String
		}

		tableInfo.Columns = append(tableInfo.Columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %w", err)
	}

	// Get index information
	indexQuery := `
		SELECT 
			INDEX_NAME,
			GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) as COLUMNS,
			NON_UNIQUE
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		GROUP BY INDEX_NAME, NON_UNIQUE
	`

	indexRows, err := m.db.Query(indexQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get index info: %w", err)
	}
	defer indexRows.Close()

	for indexRows.Next() {
		var indexName string
		var columnsStr string
		var nonUnique int

		if err := indexRows.Scan(&indexName, &columnsStr, &nonUnique); err != nil {
			return nil, fmt.Errorf("failed to scan index info: %w", err)
		}

		// Skip primary key index
		if indexName == "PRIMARY" {
			continue
		}

		index := types.IndexInfo{
			Name:    indexName,
			Columns: strings.Split(columnsStr, ","),
			Unique:  nonUnique == 0,
		}
		tableInfo.Indexes = append(tableInfo.Indexes, index)
	}

	if err := indexRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating indexes: %w", err)
	}

	// Get foreign key information
	fkQuery := `
		SELECT 
			CONSTRAINT_NAME,
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = ?
			AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY CONSTRAINT_NAME, ORDINAL_POSITION
	`

	fkRows, err := m.db.Query(fkQuery, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var constraintName string
		var columnName string
		var referencedTable string
		var referencedColumn string

		if err := fkRows.Scan(&constraintName, &columnName, &referencedTable, &referencedColumn); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		// Get ON UPDATE and ON DELETE actions
		var updateRule, deleteRule string
		actionQuery := `
			SELECT UPDATE_RULE, DELETE_RULE
			FROM information_schema.REFERENTIAL_CONSTRAINTS
			WHERE CONSTRAINT_SCHEMA = DATABASE() AND CONSTRAINT_NAME = ?
		`
		err := m.db.QueryRow(actionQuery, constraintName).Scan(&updateRule, &deleteRule)
		if err != nil {
			// Default to RESTRICT if we can't get the rules
			updateRule = "RESTRICT"
			deleteRule = "RESTRICT"
		}

		fk := types.ForeignKeyInfo{
			Name:             constraintName,
			Column:           columnName,
			ReferencedTable:  referencedTable,
			ReferencedColumn: referencedColumn,
			OnUpdate:         updateRule,
			OnDelete:         deleteRule,
		}
		tableInfo.ForeignKeys = append(tableInfo.ForeignKeys, fk)
	}

	if err := fkRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating foreign keys: %w", err)
	}

	return tableInfo, nil
}

// formatColumnType formats MySQL column type with size/precision
func (m *MySQLMigrator) formatColumnType(dataType string, charMaxLength, numPrecision, numScale sql.NullInt64) string {
	upperType := strings.ToUpper(dataType)

	switch upperType {
	case "VARCHAR":
		if charMaxLength.Valid {
			return fmt.Sprintf("VARCHAR(%d)", charMaxLength.Int64)
		}
		return "VARCHAR(255)"
	case "CHAR":
		if charMaxLength.Valid {
			return fmt.Sprintf("CHAR(%d)", charMaxLength.Int64)
		}
		return "CHAR(1)"
	case "DECIMAL":
		if numPrecision.Valid && numScale.Valid {
			return fmt.Sprintf("DECIMAL(%d,%d)", numPrecision.Int64, numScale.Int64)
		}
		return "DECIMAL(10,2)"
	default:
		return upperType
	}
}

// GenerateCreateTableSQL generates CREATE TABLE SQL from schema
func (m *MySQLMigrator) GenerateCreateTableSQL(s *schema.Schema) (string, error) {
	return m.mysqlDB.generateCreateTableSQL(s)
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (m *MySQLMigrator) GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
}

// GenerateAddColumnSQL generates ADD COLUMN SQL
func (m *MySQLMigrator) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	f, ok := field.(schema.Field)
	if !ok {
		return "", fmt.Errorf("expected schema.Field, got %T", field)
	}

	columnDef, err := m.mysqlDB.generateColumnSQL(f)
	if err != nil {
		return "", fmt.Errorf("failed to generate column definition: %w", err)
	}

	return fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", tableName, columnDef), nil
}

// GenerateModifyColumnSQL generates MODIFY COLUMN SQL
func (m *MySQLMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	// MySQL supports direct column modification with ALTER TABLE
	if change.NewColumn == nil {
		return nil, fmt.Errorf("new column definition is required")
	}

	columnDef := m.GenerateColumnDefinitionFromColumnInfo(*change.NewColumn)

	sql := fmt.Sprintf("ALTER TABLE `%s` MODIFY COLUMN %s", change.TableName, columnDef)

	// If renaming column, use CHANGE instead of MODIFY
	if change.OldColumn != nil && change.OldColumn.Name != change.NewColumn.Name {
		sql = fmt.Sprintf("ALTER TABLE `%s` CHANGE COLUMN `%s` %s",
			change.TableName, change.OldColumn.Name, columnDef)
	}

	return []string{sql}, nil
}

// GenerateDropColumnSQL generates DROP COLUMN SQL
func (m *MySQLMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	return []string{
		fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", tableName, columnName),
	}, nil
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (m *MySQLMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	uniqueStr := ""
	if unique {
		uniqueStr = "UNIQUE "
	}

	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = fmt.Sprintf("`%s`", col)
	}

	return fmt.Sprintf("CREATE %sINDEX `%s` ON `%s` (%s)",
		uniqueStr, indexName, tableName, strings.Join(quotedColumns, ", "))
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (m *MySQLMigrator) GenerateDropIndexSQL(indexName string) string {
	// In MySQL, indexes are dropped with table name
	// This is a simplified version - in practice, you'd need the table name
	return fmt.Sprintf("DROP INDEX `%s`", indexName)
}

// ApplyMigration executes a migration SQL
func (m *MySQLMigrator) ApplyMigration(sql string) error {
	_, err := m.db.Exec(sql)
	return err
}

// GetDatabaseType returns the database type
func (m *MySQLMigrator) GetDatabaseType() string {
	return "mysql"
}

// MapFieldType maps schema field types to MySQL types
func (m *MySQLMigrator) MapFieldType(field schema.Field) string {
	return m.mysqlDB.mapFieldTypeToSQL(field.Type)
}

// FormatDefaultValue formats a default value for MySQL
func (m *MySQLMigrator) FormatDefaultValue(value any) string {
	return m.mysqlDB.formatDefaultValue(value)
}

// GenerateColumnDefinitionFromColumnInfo generates column definition from ColumnInfo
func (m *MySQLMigrator) GenerateColumnDefinitionFromColumnInfo(col types.ColumnInfo) string {
	parts := []string{fmt.Sprintf("`%s`", col.Name), col.Type}

	if !col.Nullable {
		parts = append(parts, "NOT NULL")
	}

	if col.AutoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
	}

	if col.Unique && !col.PrimaryKey {
		parts = append(parts, "UNIQUE")
	}

	if col.Default != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", m.FormatDefaultValue(col.Default)))
	}

	if col.PrimaryKey {
		parts = append(parts, "PRIMARY KEY")
	}

	return strings.Join(parts, " ")
}

// ConvertFieldToColumnInfo converts a schema field to column info
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

// IsSystemIndex checks if an index is a system-generated index in MySQL
func (m *MySQLMigrator) IsSystemIndex(indexName string) bool {
	lower := strings.ToLower(indexName)
	
	// MySQL system index patterns:
	// - Primary key: PRIMARY
	// - Foreign key constraints: fk_*
	// - System internal: mysql_*
	return lower == "primary" ||
		strings.HasPrefix(lower, "fk_") ||
		strings.HasPrefix(lower, "mysql_") ||
		strings.Contains(lower, "primary_key")
}

// Additional wrapper methods to satisfy the types.DatabaseMigrator interface

// GenerateCreateTableSQL wraps to match the DatabaseMigrator interface
func (w *MySQLMigratorWrapper) GenerateCreateTableSQL(schemaInterface any) (string, error) {
	s, ok := schemaInterface.(*schema.Schema)
	if !ok {
		return "", fmt.Errorf("expected *schema.Schema, got %T", schemaInterface)
	}
	return w.specific.GenerateCreateTableSQL(s)
}

// GenerateAddColumnSQL wraps to match the DatabaseMigrator interface
func (w *MySQLMigratorWrapper) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return w.specific.GenerateAddColumnSQL(tableName, field)
}

// CompareSchema wraps the BaseMigrator's CompareSchema to match the DatabaseMigrator interface
func (w *MySQLMigratorWrapper) CompareSchema(existingTable *types.TableInfo, desiredSchema any) (*types.MigrationPlan, error) {
	s, ok := desiredSchema.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("expected *schema.Schema, got %T", desiredSchema)
	}
	return w.BaseMigrator.CompareSchema(existingTable, s)
}
