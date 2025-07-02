package migration

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Generator generates migration files
type Generator struct {
	differ      *Differ
	fileManager *FileManager
}

// NewGenerator creates a new migration generator
func NewGenerator(migrator types.DatabaseMigrator, fileManager *FileManager) *Generator {
	return &Generator{
		differ:      NewDiffer(migrator),
		fileManager: fileManager,
	}
}

// GenerateMigration generates a new migration by comparing schemas with database
func (g *Generator) GenerateMigration(name string, schemas map[string]*schema.Schema) (*MigrationFile, error) {
	// Compute differences
	changes, err := g.differ.ComputeDiff(schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes detected")
	}

	// Generate version
	version := GenerateVersion()

	// Generate up and down SQL
	upSQL, downSQL, err := g.generateSQL(changes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Compute checksum
	checksum := ComputeChecksum(changes)

	// Compute schema hashes
	schemaHashes := make(map[string]string)
	for modelName, s := range schemas {
		schemaHashes[modelName] = g.computeSchemaHash(s)
	}

	// Create migration file
	migration := &MigrationFile{
		Version: version,
		Name:    name,
		UpSQL:   upSQL,
		DownSQL: downSQL,
		Metadata: MigrationMetadata{
			Version:     version,
			Name:        name,
			Checksum:    checksum,
			CreatedAt:   time.Now(),
			Description: g.generateDescription(changes),
			Changes:     changes,
			Schemas:     schemaHashes,
		},
	}

	return migration, nil
}

// generateSQL generates up and down SQL from changes
func (g *Generator) generateSQL(changes []SchemaChange) (upSQL, downSQL string, err error) {
	var upStatements []string
	var downStatements []string

	// Group changes by table for better organization
	tableChanges := g.groupChangesByTable(changes)

	// Generate SQL for each table
	for tableName, changes := range tableChanges {
		// Process changes in order: indexes first (drop), columns, then indexes (add)
		// This ensures we don't try to add indexes on columns that don't exist yet

		// Drop indexes first (in down SQL, these become creates)
		for _, change := range changes {
			if change.Type == ChangeTypeDropIndex {
				upStatements = append(upStatements, change.SQL)
				// For down SQL, we need to recreate the index
				// This would need to be stored in metadata
				downStatements = append([]string{g.generateCreateIndexSQL(change)}, downStatements...)
			}
		}

		// Handle column changes
		for _, change := range changes {
			switch change.Type {
			case ChangeTypeCreateTable:
				upStatements = append(upStatements, change.SQL)
				downStatements = append([]string{fmt.Sprintf("DROP TABLE %s", tableName)}, downStatements...)

			case ChangeTypeDropTable:
				upStatements = append(upStatements, change.SQL)
				// For down SQL, we would need the full table definition stored
				// This is a limitation - we can't recreate dropped tables without storing their schema
				downStatements = append([]string{
					fmt.Sprintf("-- Cannot recreate table %s without stored schema", tableName),
				}, downStatements...)

			case ChangeTypeAddColumn:
				upStatements = append(upStatements, change.SQL)
				downStatements = append([]string{
					fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", tableName, change.ColumnName),
				}, downStatements...)

			case ChangeTypeDropColumn:
				upStatements = append(upStatements, change.SQL)
				// For down SQL, we would need the column definition stored
				downStatements = append([]string{
					fmt.Sprintf("-- Cannot recreate column %s.%s without stored definition", tableName, change.ColumnName),
				}, downStatements...)

			case ChangeTypeAlterColumn:
				upStatements = append(upStatements, change.SQL)
				// For down SQL, we would need the old column definition
				downStatements = append([]string{
					fmt.Sprintf("-- Cannot revert column %s.%s without stored old definition", tableName, change.ColumnName),
				}, downStatements...)
			}
		}

		// Add indexes last (in down SQL, these become drops)
		for _, change := range changes {
			if change.Type == ChangeTypeAddIndex {
				upStatements = append(upStatements, change.SQL)
				downStatements = append([]string{
					fmt.Sprintf("DROP INDEX %s", change.IndexName),
				}, downStatements...)
			}
		}
	}

	// Build final SQL with headers
	upSQL = g.buildSQLScript(upStatements, "UP", "")
	downSQL = g.buildSQLScript(downStatements, "DOWN", "")

	return upSQL, downSQL, nil
}

// groupChangesByTable groups changes by table name
func (g *Generator) groupChangesByTable(changes []SchemaChange) map[string][]SchemaChange {
	grouped := make(map[string][]SchemaChange)
	for _, change := range changes {
		grouped[change.TableName] = append(grouped[change.TableName], change)
	}
	return grouped
}

// buildSQLScript builds a SQL script with comments
func (g *Generator) buildSQLScript(statements []string, direction, version string) string {
	var lines []string

	// Add header
	lines = append(lines, fmt.Sprintf("-- RediORM Migration %s", version))
	lines = append(lines, fmt.Sprintf("-- Direction: %s", direction))
	lines = append(lines, fmt.Sprintf("-- Generated at: %s", time.Now().Format(time.RFC3339)))
	lines = append(lines, "")

	// Add statements
	for _, stmt := range statements {
		lines = append(lines, stmt+";")
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// generateDescription generates a human-readable description of changes
func (g *Generator) generateDescription(changes []SchemaChange) string {
	summary := make(map[ChangeType]int)
	for _, change := range changes {
		summary[change.Type]++
	}

	var parts []string
	if count := summary[ChangeTypeCreateTable]; count > 0 {
		parts = append(parts, fmt.Sprintf("Create %d table(s)", count))
	}
	if count := summary[ChangeTypeDropTable]; count > 0 {
		parts = append(parts, fmt.Sprintf("Drop %d table(s)", count))
	}
	if count := summary[ChangeTypeAddColumn]; count > 0 {
		parts = append(parts, fmt.Sprintf("Add %d column(s)", count))
	}
	if count := summary[ChangeTypeDropColumn]; count > 0 {
		parts = append(parts, fmt.Sprintf("Drop %d column(s)", count))
	}
	if count := summary[ChangeTypeAlterColumn]; count > 0 {
		parts = append(parts, fmt.Sprintf("Alter %d column(s)", count))
	}
	if count := summary[ChangeTypeAddIndex]; count > 0 {
		parts = append(parts, fmt.Sprintf("Add %d index(es)", count))
	}
	if count := summary[ChangeTypeDropIndex]; count > 0 {
		parts = append(parts, fmt.Sprintf("Drop %d index(es)", count))
	}

	return strings.Join(parts, ", ")
}

// computeSchemaHash computes a hash of a schema for change detection
func (g *Generator) computeSchemaHash(s *schema.Schema) string {
	h := sha256.New()

	// Hash schema name and table name
	h.Write([]byte(s.Name))
	h.Write([]byte(s.TableName))

	// Hash fields
	for _, field := range s.Fields {
		h.Write([]byte(field.Name))
		h.Write([]byte(field.Type))
		h.Write([]byte(fmt.Sprintf("%v", field.PrimaryKey)))
		h.Write([]byte(fmt.Sprintf("%v", field.Nullable)))
		h.Write([]byte(fmt.Sprintf("%v", field.Unique)))
		h.Write([]byte(fmt.Sprintf("%v", field.Default)))
		h.Write([]byte(field.Map))
	}

	// Hash indexes
	for _, index := range s.Indexes {
		h.Write([]byte(index.Name))
		h.Write([]byte(strings.Join(index.Fields, ",")))
		h.Write([]byte(fmt.Sprintf("%v", index.Unique)))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// generateCreateIndexSQL generates SQL to recreate a dropped index
// This is a placeholder - in a real implementation, we'd need to store
// the index definition in the metadata
func (g *Generator) generateCreateIndexSQL(change SchemaChange) string {
	return fmt.Sprintf("-- Recreate index %s (definition needed from metadata)", change.IndexName)
}
