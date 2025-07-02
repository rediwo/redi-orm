package base

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// BaseMigrator provides common migration functionality that all database drivers can use
type BaseMigrator struct {
	specific types.DatabaseSpecificMigrator
}

// NewBaseMigrator creates a new base migrator with database-specific implementation
func NewBaseMigrator(specific types.DatabaseSpecificMigrator) *BaseMigrator {
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
func (b *BaseMigrator) GenerateAddColumnSQL(tableName string, field any) (string, error) {
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

	// Compare indexes
	if err := b.compareIndexes(existingTable, desiredSchema, plan); err != nil {
		return nil, fmt.Errorf("failed to compare indexes: %w", err)
	}

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
func (b *BaseMigrator) defaultValuesEqual(existing, desired any) bool {
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
func (b *BaseMigrator) EnsureSchemaForRegisteredSchemas(schemas map[string]any, createTableFunc func(*schema.Schema) error) error {
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

// compareIndexes compares existing indexes with desired indexes
func (b *BaseMigrator) compareIndexes(existingTable *types.TableInfo, desiredSchema *schema.Schema, plan *types.MigrationPlan) error {
	// Create maps for efficient lookup
	existingIndexMap := make(map[string]*types.IndexInfo)
	for i := range existingTable.Indexes {
		idx := &existingTable.Indexes[i]
		// Normalize index name for comparison
		normalizedName := b.normalizeIndexName(idx.Name)
		existingIndexMap[normalizedName] = idx
	}

	desiredIndexMap := make(map[string]*schema.Index)
	processedIndexNames := make(map[string]bool)

	// Process explicitly defined indexes
	for i := range desiredSchema.Indexes {
		idx := &desiredSchema.Indexes[i]
		indexName := b.generateIndexName(existingTable.Name, idx)
		normalizedName := b.normalizeIndexName(indexName)
		desiredIndexMap[normalizedName] = idx
		processedIndexNames[normalizedName] = true
	}

	// Process field-level indexes
	for _, field := range desiredSchema.Fields {
		if field.Index && !field.PrimaryKey && !field.Unique {
			// Generate index name for field-level index
			indexName := b.generateFieldIndexName(existingTable.Name, field.GetColumnName())
			normalizedName := b.normalizeIndexName(indexName)

			// Skip if already processed (e.g., part of composite index)
			if !processedIndexNames[normalizedName] {
				idx := &schema.Index{
					Name:   indexName,
					Fields: []string{field.Name},
					Unique: false,
				}
				desiredIndexMap[normalizedName] = idx
				processedIndexNames[normalizedName] = true
			}
		}
	}

	// Check for new indexes (ADD)
	for normalizedName, desiredIdx := range desiredIndexMap {
		if _, exists := existingIndexMap[normalizedName]; !exists {
			// Convert field names to column names
			columnNames := make([]string, len(desiredIdx.Fields))
			for i, fieldName := range desiredIdx.Fields {
				if field := desiredSchema.GetFieldByName(fieldName); field != nil {
					columnNames[i] = field.GetColumnName()
				} else {
					columnNames[i] = fieldName // Fallback to field name
				}
			}

			newIndex := &types.IndexInfo{
				Name:    desiredIdx.Name,
				Columns: columnNames,
				Unique:  desiredIdx.Unique,
			}

			plan.AddIndexes = append(plan.AddIndexes, types.IndexChange{
				TableName: existingTable.Name,
				IndexName: desiredIdx.Name,
				OldIndex:  nil,
				NewIndex:  newIndex,
			})
		}
	}

	// Check for indexes to drop (DROP)
	for normalizedName, existingIdx := range existingIndexMap {
		// Skip primary key and unique constraint indexes
		if b.specific.IsSystemIndex(existingIdx.Name) {
			continue
		}

		if _, exists := desiredIndexMap[normalizedName]; !exists {
			plan.DropIndexes = append(plan.DropIndexes, types.IndexChange{
				TableName: existingTable.Name,
				IndexName: existingIdx.Name,
				OldIndex:  existingIdx,
				NewIndex:  nil,
			})
		}
	}

	// Check for modified indexes (DROP and recreate)
	for normalizedName, desiredIdx := range desiredIndexMap {
		if existingIdx, exists := existingIndexMap[normalizedName]; exists {
			// Convert field names to column names for comparison
			columnNames := make([]string, len(desiredIdx.Fields))
			for i, fieldName := range desiredIdx.Fields {
				if field := desiredSchema.GetFieldByName(fieldName); field != nil {
					columnNames[i] = field.GetColumnName()
				} else {
					columnNames[i] = fieldName
				}
			}

			// Check if index needs modification
			if b.indexNeedsModification(existingIdx, columnNames, desiredIdx.Unique) {
				// Drop old index
				plan.DropIndexes = append(plan.DropIndexes, types.IndexChange{
					TableName: existingTable.Name,
					IndexName: existingIdx.Name,
					OldIndex:  existingIdx,
					NewIndex:  nil,
				})

				// Add new index
				newIndex := &types.IndexInfo{
					Name:    desiredIdx.Name,
					Columns: columnNames,
					Unique:  desiredIdx.Unique,
				}

				plan.AddIndexes = append(plan.AddIndexes, types.IndexChange{
					TableName: existingTable.Name,
					IndexName: desiredIdx.Name,
					OldIndex:  nil,
					NewIndex:  newIndex,
				})
			}
		}
	}

	return nil
}

// normalizeIndexName normalizes index name for comparison
func (b *BaseMigrator) normalizeIndexName(name string) string {
	// Remove common prefixes and suffixes
	normalized := strings.ToLower(name)
	normalized = strings.TrimPrefix(normalized, "idx_")
	normalized = strings.TrimPrefix(normalized, "index_")
	normalized = strings.TrimSuffix(normalized, "_idx")
	normalized = strings.TrimSuffix(normalized, "_index")
	return normalized
}

// generateIndexName generates a consistent index name
func (b *BaseMigrator) generateIndexName(tableName string, idx *schema.Index) string {
	if idx.Name != "" {
		return idx.Name
	}

	// Generate name based on columns
	prefix := "idx"
	if idx.Unique {
		prefix = "uniq"
	}

	columnPart := strings.Join(idx.Fields, "_")
	return fmt.Sprintf("%s_%s_%s", prefix, tableName, columnPart)
}

// generateFieldIndexName generates index name for field-level index
func (b *BaseMigrator) generateFieldIndexName(tableName, columnName string) string {
	return fmt.Sprintf("idx_%s_%s", tableName, columnName)
}

// indexNeedsModification checks if an index needs to be modified
func (b *BaseMigrator) indexNeedsModification(existing *types.IndexInfo, desiredColumns []string, desiredUnique bool) bool {
	// Check unique flag
	if existing.Unique != desiredUnique {
		return true
	}

	// Check column count
	if len(existing.Columns) != len(desiredColumns) {
		return true
	}

	// Check column names and order
	for i, col := range existing.Columns {
		if !strings.EqualFold(col, desiredColumns[i]) {
			return true
		}
	}

	return false
}
