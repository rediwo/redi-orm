package migration

import (
	"crypto/sha256"
	"fmt"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Differ compares schemas and generates migration plans
type Differ struct {
	migrator types.DatabaseMigrator
}

// NewDiffer creates a new schema differ
func NewDiffer(migrator types.DatabaseMigrator) *Differ {
	return &Differ{
		migrator: migrator,
	}
}

// ComputeDiff compares desired schemas with current database state
func (d *Differ) ComputeDiff(schemas map[string]*schema.Schema) ([]SchemaChange, error) {
	var changes []SchemaChange

	// Get current tables from database
	currentTables, err := d.migrator.GetTables()
	if err != nil {
		return nil, err
	}

	currentTableMap := make(map[string]bool)
	for _, table := range currentTables {
		currentTableMap[table] = true
	}

	// Check for new tables to create
	for _, s := range schemas {
		if !currentTableMap[s.TableName] {
			// Table doesn't exist, create it
			sql, err := d.migrator.GenerateCreateTableSQL(s)
			if err != nil {
				return nil, err
			}
			changes = append(changes, SchemaChange{
				Type:      ChangeTypeCreateTable,
				TableName: s.TableName,
				SQL:       sql,
			})
		} else {
			// Table exists, check for column changes
			tableChanges, err := d.computeTableDiff(s)
			if err != nil {
				return nil, err
			}
			changes = append(changes, tableChanges...)
		}
	}

	// Check for tables to drop (tables in DB but not in schema)
	desiredTableMap := make(map[string]bool)
	for _, s := range schemas {
		desiredTableMap[s.TableName] = true
	}

	for _, table := range currentTables {
		// Skip system tables (like migrations table)
		if table == MigrationsTableName {
			continue
		}

		if !desiredTableMap[table] {
			changes = append(changes, SchemaChange{
				Type:      ChangeTypeDropTable,
				TableName: table,
				SQL:       d.migrator.GenerateDropTableSQL(table),
			})
		}
	}

	return changes, nil
}

// computeTableDiff compares a single table's schema with database
func (d *Differ) computeTableDiff(s *schema.Schema) ([]SchemaChange, error) {
	var changes []SchemaChange

	// Get current table info
	tableInfo, err := d.migrator.GetTableInfo(s.TableName)
	if err != nil {
		return nil, err
	}

	// Use the migrator's CompareSchema method to get detailed changes
	plan, err := d.migrator.CompareSchema(tableInfo, s)
	if err != nil {
		return nil, err
	}

	// Generate SQL for the migration plan
	sqlStatements, err := d.migrator.GenerateMigrationSQL(plan)
	if err != nil {
		return nil, err
	}

	// Convert MigrationPlan to SchemaChange array
	// Process columns
	for i, change := range plan.AddColumns {
		if i < len(sqlStatements) {
			changes = append(changes, SchemaChange{
				Type:       ChangeTypeAddColumn,
				TableName:  change.TableName,
				ColumnName: change.ColumnName,
				SQL:        sqlStatements[i],
			})
		}
	}

	for _, change := range plan.ModifyColumns {
		// Modify columns may generate multiple SQL statements
		changes = append(changes, SchemaChange{
			Type:       ChangeTypeAlterColumn,
			TableName:  change.TableName,
			ColumnName: change.ColumnName,
			SQL:        fmt.Sprintf("-- Modify column %s.%s", change.TableName, change.ColumnName),
		})
	}

	for _, change := range plan.DropColumns {
		changes = append(changes, SchemaChange{
			Type:       ChangeTypeDropColumn,
			TableName:  change.TableName,
			ColumnName: change.ColumnName,
			SQL:        fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", change.TableName, change.ColumnName),
		})
	}

	// Process indexes
	for _, change := range plan.AddIndexes {
		changes = append(changes, SchemaChange{
			Type:      ChangeTypeAddIndex,
			TableName: change.TableName,
			IndexName: change.IndexName,
			SQL:       d.migrator.GenerateCreateIndexSQL(change.TableName, change.IndexName, change.NewIndex.Columns, change.NewIndex.Unique),
		})
	}

	for _, change := range plan.DropIndexes {
		changes = append(changes, SchemaChange{
			Type:      ChangeTypeDropIndex,
			TableName: change.TableName,
			IndexName: change.IndexName,
			SQL:       d.migrator.GenerateDropIndexSQL(change.IndexName),
		})
	}

	return changes, nil
}

// ComputeChecksum computes a checksum for a migration plan
func ComputeChecksum(changes []SchemaChange) string {
	h := sha256.New()

	for _, change := range changes {
		h.Write([]byte(change.Type))
		h.Write([]byte(change.TableName))
		h.Write([]byte(change.SQL))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
