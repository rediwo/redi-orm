package migration

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// DatabaseSpecificMigrator defines database-specific operations that each driver must implement
type DatabaseSpecificMigrator interface {
	// Database introspection
	GetTables() ([]string, error)
	GetTableInfo(tableName string) (*types.TableInfo, error)

	// SQL generation
	GenerateCreateTableSQL(s *schema.Schema) (string, error)
	GenerateDropTableSQL(tableName string) string
	GenerateAddColumnSQL(tableName string, field interface{}) (string, error)
	GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error)
	GenerateDropColumnSQL(tableName, columnName string) ([]string, error)
	GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string
	GenerateDropIndexSQL(indexName string) string

	// Migration execution
	ApplyMigration(sql string) error
	GetDatabaseType() string

	// Type mapping and column generation
	MapFieldType(field schema.Field) string
	FormatDefaultValue(value interface{}) string
	GenerateColumnDefinitionFromColumnInfo(col types.ColumnInfo) string
	ConvertFieldToColumnInfo(field schema.Field) *types.ColumnInfo
}

// BaseMigrator provides common migration functionality that all database drivers can use
type BaseMigrator struct {
	specific DatabaseSpecificMigrator
}

// NewBaseMigrator creates a new base migrator with database-specific implementation
func NewBaseMigrator(specific DatabaseSpecificMigrator) *BaseMigrator {
	return &BaseMigrator{
		specific: specific,
	}
}

// GetDatabaseType returns the database type
func (b *BaseMigrator) GetDatabaseType() string {
	return b.specific.GetDatabaseType()
}

// GetTables returns all tables in the database
func (b *BaseMigrator) GetTables() ([]string, error) {
	return b.specific.GetTables()
}

// GetTableInfo returns detailed information about a table
func (b *BaseMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	return b.specific.GetTableInfo(tableName)
}

// GenerateCreateTableSQL generates CREATE TABLE SQL
func (b *BaseMigrator) GenerateCreateTableSQL(s *schema.Schema) (string, error) {
	return b.specific.GenerateCreateTableSQL(s)
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (b *BaseMigrator) GenerateDropTableSQL(tableName string) string {
	return b.specific.GenerateDropTableSQL(tableName)
}

// GenerateAddColumnSQL generates ALTER TABLE ADD COLUMN SQL
func (b *BaseMigrator) GenerateAddColumnSQL(tableName string, field interface{}) (string, error) {
	return b.specific.GenerateAddColumnSQL(tableName, field)
}

// GenerateModifyColumnSQL generates ALTER TABLE MODIFY COLUMN SQL
func (b *BaseMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	return b.specific.GenerateModifyColumnSQL(change)
}

// GenerateDropColumnSQL generates ALTER TABLE DROP COLUMN SQL
func (b *BaseMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	return b.specific.GenerateDropColumnSQL(tableName, columnName)
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (b *BaseMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	return b.specific.GenerateCreateIndexSQL(tableName, indexName, columns, unique)
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (b *BaseMigrator) GenerateDropIndexSQL(indexName string) string {
	return b.specific.GenerateDropIndexSQL(indexName)
}

// ApplyMigration executes a migration SQL statement
func (b *BaseMigrator) ApplyMigration(sql string) error {
	return b.specific.ApplyMigration(sql)
}

// CompareSchema compares existing table with desired schema and creates migration plan
// This is the shared logic that works for all databases
func (b *BaseMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema *schema.Schema) (*types.MigrationPlan, error) {
	plan := &types.MigrationPlan{
		CreateTables:  []string{},
		AddColumns:    []types.ColumnChange{},
		ModifyColumns: []types.ColumnChange{},
		DropColumns:   []types.ColumnChange{},
		AddIndexes:    []types.IndexChange{},
		DropIndexes:   []types.IndexChange{},
	}

	// Create maps for efficient lookup
	existingColumnMap := make(map[string]*types.ColumnInfo)
	for i := range existingTable.Columns {
		col := &existingTable.Columns[i]
		existingColumnMap[col.Name] = col
	}

	desiredColumnMap := make(map[string]*schema.Field)
	for i := range desiredSchema.Fields {
		field := &desiredSchema.Fields[i]
		columnName := field.GetColumnName()
		desiredColumnMap[columnName] = field
	}

	// Check for new columns (ADD)
	for _, field := range desiredSchema.Fields {
		columnName := field.GetColumnName()
		if _, exists := existingColumnMap[columnName]; !exists {
			// This is a new column
			newColumn := b.specific.ConvertFieldToColumnInfo(field)
			plan.AddColumns = append(plan.AddColumns, types.ColumnChange{
				TableName:  existingTable.Name,
				ColumnName: columnName,
				OldColumn:  nil,
				NewColumn:  newColumn,
			})
		}
	}

	// Check for modified columns (MODIFY) and removed columns (DROP)
	for _, existingCol := range existingTable.Columns {
		if desiredField, exists := desiredColumnMap[existingCol.Name]; exists {
			// Column exists in both, check if it needs modification
			newColumn := b.specific.ConvertFieldToColumnInfo(*desiredField)
			if b.columnsNeedModification(existingCol, newColumn) {
				plan.ModifyColumns = append(plan.ModifyColumns, types.ColumnChange{
					TableName:  existingTable.Name,
					ColumnName: existingCol.Name,
					OldColumn:  &existingCol,
					NewColumn:  newColumn,
				})
			}
		} else {
			// Column exists in database but not in schema (DROP)
			plan.DropColumns = append(plan.DropColumns, types.ColumnChange{
				TableName:  existingTable.Name,
				ColumnName: existingCol.Name,
				OldColumn:  &existingCol,
				NewColumn:  nil,
			})
		}
	}

	// TODO: Compare indexes (AddIndexes, DropIndexes)
	// For now, we'll focus on column changes

	return plan, nil
}

// GenerateMigrationSQL generates SQL statements for a migration plan
// This provides a common implementation that databases can override if needed
func (b *BaseMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	var sqlStatements []string

	// Generate CREATE TABLE statements
	for _, table := range plan.CreateTables {
		sqlStatements = append(sqlStatements, fmt.Sprintf("CREATE TABLE %s (...)", table))
	}

	// Generate ADD COLUMN statements
	for _, change := range plan.AddColumns {
		if change.NewColumn == nil {
			continue
		}

		columnDef := b.specific.GenerateColumnDefinitionFromColumnInfo(*change.NewColumn)
		sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", change.TableName, columnDef)
		sqlStatements = append(sqlStatements, sql)
	}

	// Generate MODIFY COLUMN statements (database-specific handling)
	for _, change := range plan.ModifyColumns {
		if change.NewColumn == nil {
			continue
		}

		// Delegate to database-specific implementation
		sqls, err := b.specific.GenerateModifyColumnSQL(change)
		if err != nil {
			return nil, fmt.Errorf("failed to generate modify column SQL: %w", err)
		}
		sqlStatements = append(sqlStatements, sqls...)
	}

	// Generate DROP COLUMN statements (database-specific handling)
	for _, change := range plan.DropColumns {
		// Delegate to database-specific implementation
		sqls, err := b.specific.GenerateDropColumnSQL(change.TableName, change.ColumnName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate drop column SQL: %w", err)
		}
		sqlStatements = append(sqlStatements, sqls...)
	}

	// Generate index changes
	for _, change := range plan.AddIndexes {
		if change.NewIndex != nil {
			sql := b.GenerateCreateIndexSQL(change.TableName, change.IndexName,
				change.NewIndex.Columns, change.NewIndex.Unique)
			sqlStatements = append(sqlStatements, sql)
		}
	}

	for _, change := range plan.DropIndexes {
		sql := b.GenerateDropIndexSQL(change.IndexName)
		sqlStatements = append(sqlStatements, sql)
	}

	return sqlStatements, nil
}

// columnsNeedModification checks if two columns differ and need modification
// This is shared logic that works for all databases
func (b *BaseMigrator) columnsNeedModification(existing types.ColumnInfo, desired *types.ColumnInfo) bool {
	// Normalize types for comparison
	existingType := strings.ToUpper(strings.TrimSpace(existing.Type))
	desiredType := strings.ToUpper(strings.TrimSpace(desired.Type))

	// Check each property
	if existingType != desiredType {
		return true
	}
	if existing.Nullable != desired.Nullable {
		return true
	}
	if existing.PrimaryKey != desired.PrimaryKey {
		return true
	}
	if existing.AutoIncrement != desired.AutoIncrement {
		return true
	}
	if existing.Unique != desired.Unique {
		return true
	}

	// Compare defaults (simplified)
	if !b.defaultValuesEqual(existing.Default, desired.Default) {
		return true
	}

	return false
}

// defaultValuesEqual compares two default values
// This is shared logic that works for all databases
func (b *BaseMigrator) defaultValuesEqual(existing, desired interface{}) bool {
	// Handle nil cases
	if existing == nil && desired == nil {
		return true
	}
	if existing == nil || desired == nil {
		return false
	}

	// Convert to strings for comparison (simplified)
	existingStr := fmt.Sprintf("%v", existing)
	desiredStr := fmt.Sprintf("%v", desired)

	return existingStr == desiredStr
}

// EnsureSchemaForRegisteredSchemas provides common EnsureSchema logic for all drivers
func (b *BaseMigrator) EnsureSchemaForRegisteredSchemas(schemas map[string]interface{}, createTableFunc func(*schema.Schema) error) error {
	// Get list of existing tables
	existingTables, err := b.GetTables()
	if err != nil {
		return fmt.Errorf("failed to get existing tables: %w", err)
	}

	// Create a map for quick lookup
	existingTableMap := make(map[string]bool)
	for _, table := range existingTables {
		existingTableMap[table] = true
	}

	// Process each registered schema
	for _, schemaInterface := range schemas {
		schemaObj, ok := schemaInterface.(*schema.Schema)
		if !ok {
			continue
		}

		tableName := schemaObj.TableName

		if !existingTableMap[tableName] {
			// Table doesn't exist, create it
			if err := createTableFunc(schemaObj); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		} else {
			// Table exists, check for schema changes and apply migrations
			existingTableInfo, err := b.GetTableInfo(tableName)
			if err != nil {
				return fmt.Errorf("failed to get table info for %s: %w", tableName, err)
			}

			// Compare existing table with desired schema
			migrationPlan, err := b.CompareSchema(existingTableInfo, schemaObj)
			if err != nil {
				return fmt.Errorf("failed to compare schema for %s: %w", tableName, err)
			}

			// Apply migrations if needed
			if b.hasMigrations(migrationPlan) {
				sqlStatements, err := b.GenerateMigrationSQL(migrationPlan)
				if err != nil {
					return fmt.Errorf("failed to generate migration SQL for %s: %w", tableName, err)
				}

				// Execute migration statements
				for _, sql := range sqlStatements {
					// Skip comment-only statements
					if strings.HasPrefix(strings.TrimSpace(sql), "--") {
						continue
					}

					if err := b.ApplyMigration(sql); err != nil {
						return fmt.Errorf("failed to apply migration for %s: %s, error: %w", tableName, sql, err)
					}
				}
			}
		}
	}

	return nil
}

// hasMigrations checks if a migration plan has any actual changes
func (b *BaseMigrator) hasMigrations(plan *types.MigrationPlan) bool {
	return len(plan.CreateTables) > 0 ||
		len(plan.AddColumns) > 0 ||
		len(plan.ModifyColumns) > 0 ||
		len(plan.DropColumns) > 0 ||
		len(plan.AddIndexes) > 0 ||
		len(plan.DropIndexes) > 0
}
