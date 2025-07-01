package migration

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Manager coordinates the migration process
type Manager struct {
	db       *sql.DB
	migrator types.DatabaseMigrator
	history  *HistoryManager
	differ   *Differ
	options  MigrationOptions
}

// NewManager creates a new migration manager
func NewManager(db types.Database, options MigrationOptions) (*Manager, error) {
	migrator := db.GetMigrator()
	if migrator == nil {
		return nil, fmt.Errorf("database does not support migrations")
	}

	// Extract sql.DB for history manager
	sqlDB, ok := db.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("cannot access underlying sql.DB")
	}

	history := NewHistoryManager(sqlDB.GetDB())
	differ := NewDiffer(migrator)

	return &Manager{
		db:       sqlDB.GetDB(),
		migrator: migrator,
		history:  history,
		differ:   differ,
		options:  options,
	}, nil
}

// Migrate performs automatic migration based on schemas
func (m *Manager) Migrate(schemas map[string]*schema.Schema) error {
	log.Printf("Starting migration process...")

	// Ensure migration history table exists
	if err := m.history.EnsureMigrationTable(); err != nil {
		return fmt.Errorf("failed to ensure migration table: %w", err)
	}

	// Compute schema differences
	changes, err := m.differ.ComputeDiff(schemas)
	if err != nil {
		return fmt.Errorf("failed to compute schema diff: %w", err)
	}

	if len(changes) == 0 {
		log.Printf("No schema changes detected.")
		return nil
	}

	// Create migration plan
	version := GenerateVersion()
	checksum := ComputeChecksum(changes)

	log.Printf("Generated migration plan with %d changes (version: %s)", len(changes), version)

	// Check if this is a dry run
	if m.options.DryRun {
		m.printMigrationPlan(changes)
		return nil
	}

	// Check for destructive changes
	if m.hasDestructiveChanges(changes) && !m.options.Force {
		log.Printf("Warning: Migration contains destructive changes:")
		for _, change := range changes {
			if m.isDestructive(change) {
				log.Printf("  - %s: %s.%s", change.Type, change.TableName, change.ColumnName)
			}
		}

		if !m.options.Force {
			return fmt.Errorf("migration contains destructive changes. Use --force to proceed")
		}
	}

	// Execute migration
	if err := m.executeMigration(version, changes, checksum); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Printf("Migration completed successfully (version: %s)", version)
	return nil
}

// executeMigration executes a migration plan
func (m *Manager) executeMigration(version string, changes []SchemaChange, checksum string) error {
	// Begin transaction
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute each change
	for i, change := range changes {
		log.Printf("Executing change %d/%d: %s", i+1, len(changes), change.Type)

		if change.SQL == "" || strings.HasPrefix(change.SQL, "--") {
			log.Printf("Skipping: %s", change.SQL)
			continue
		}

		_, err := tx.Exec(change.SQL)
		if err != nil {
			return fmt.Errorf("failed to execute SQL: %s, error: %w", change.SQL, err)
		}
	}

	// Record migration in history
	_, err = tx.Exec(
		"INSERT INTO redi_migrations (version, name, checksum) VALUES (?, ?, ?)",
		version, "auto-migration", checksum,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	return tx.Commit()
}

// GetMigrationStatus returns the current migration status
func (m *Manager) GetMigrationStatus() (*MigrationStatus, error) {
	// Ensure migration history table exists
	if err := m.history.EnsureMigrationTable(); err != nil {
		return nil, fmt.Errorf("failed to ensure migration table: %w", err)
	}

	migrations, err := m.history.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	var lastMigration *Migration
	if len(migrations) > 0 {
		lastMigration = &migrations[len(migrations)-1]
	}

	tables, err := m.migrator.GetTables()
	if err != nil {
		return nil, err
	}

	return &MigrationStatus{
		AppliedMigrations: migrations,
		LastMigration:     lastMigration,
		TableCount:        len(tables),
		Tables:            tables,
	}, nil
}

// ResetMigrations drops all tables and clears migration history
func (m *Manager) ResetMigrations() error {
	log.Printf("Resetting all migrations...")

	// Get all tables
	tables, err := m.migrator.GetTables()
	if err != nil {
		return err
	}

	// Drop all tables
	for _, table := range tables {
		sql := m.migrator.GenerateDropTableSQL(table)
		_, err := m.db.Exec(sql)
		if err != nil {
			log.Printf("Warning: Failed to drop table %s: %v", table, err)
		} else {
			log.Printf("Dropped table: %s", table)
		}
	}

	// Clear migration history
	_, err = m.db.Exec("DELETE FROM redi_migrations")
	if err != nil {
		log.Printf("Warning: Failed to clear migration history: %v", err)
	}

	log.Printf("Migration reset completed.")
	return nil
}

// printMigrationPlan prints the migration plan for dry run
func (m *Manager) printMigrationPlan(changes []SchemaChange) {
	fmt.Println("\n=== MIGRATION PLAN (DRY RUN) ===")

	for i, change := range changes {
		fmt.Printf("\n%d. %s", i+1, change.Type)
		if change.TableName != "" {
			fmt.Printf(" - Table: %s", change.TableName)
		}
		if change.ColumnName != "" {
			fmt.Printf(" - Column: %s", change.ColumnName)
		}
		fmt.Printf("\n   SQL: %s\n", change.SQL)
	}

	fmt.Println("\n=== END MIGRATION PLAN ===")
}

// hasDestructiveChanges checks if the migration contains destructive changes
func (m *Manager) hasDestructiveChanges(changes []SchemaChange) bool {
	for _, change := range changes {
		if m.isDestructive(change) {
			return true
		}
	}
	return false
}

// isDestructive checks if a change is destructive
func (m *Manager) isDestructive(change SchemaChange) bool {
	switch change.Type {
	case ChangeTypeDropTable, ChangeTypeDropColumn:
		return true
	case ChangeTypeAlterColumn:
		// Column type changes can be destructive
		return true
	default:
		return false
	}
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	AppliedMigrations []Migration
	LastMigration     *Migration
	TableCount        int
	Tables            []string
}
