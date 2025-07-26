package migration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Manager coordinates the migration process
type Manager struct {
	db          *sql.DB
	database    types.Database // Keep reference to database for logging
	migrator    types.DatabaseMigrator
	history     *HistoryManager
	differ      *Differ
	options     types.MigrationOptions
	fileManager *FileManager
	generator   *Generator
	runner      *Runner
}

// NewManager creates a new migration manager
func NewManager(db types.Database, options types.MigrationOptions) (*Manager, error) {
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

	manager := &Manager{
		db:       sqlDB.GetDB(),
		database: db,
		migrator: migrator,
		history:  history,
		differ:   differ,
		options:  options,
	}

	// Initialize file-based migration components if in file mode
	if options.Mode == types.MigrationModeFile && options.MigrationsDir != "" {
		manager.fileManager = NewFileManager(options.MigrationsDir)
		manager.generator = NewGenerator(migrator, manager.fileManager)

		runner, err := NewRunner(db, manager.fileManager)
		if err != nil {
			return nil, fmt.Errorf("failed to create migration runner: %w", err)
		}
		manager.runner = runner
	}

	return manager, nil
}

// Migrate performs migration based on schemas and configured mode
func (m *Manager) Migrate(schemas map[string]*schema.Schema) error {
	// Handle file-based migrations
	if m.options.Mode == types.MigrationModeFile {
		return m.runFileMigrations()
	}

	// Default to auto-migration mode
	return m.autoMigrate(schemas)
}

// autoMigrate performs automatic migration based on schemas
func (m *Manager) autoMigrate(schemas map[string]*schema.Schema) error {
	log.Printf("Starting auto-migration process...")

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

// runFileMigrations runs file-based migrations from the configured directory
func (m *Manager) runFileMigrations() error {
	if m.runner == nil {
		return fmt.Errorf("file-based migrations not configured")
	}

	ctx := context.Background()
	return m.runner.RunMigrations(ctx)
}

// GenerateMigration generates a new migration file
func (m *Manager) GenerateMigration(name string, schemas map[string]*schema.Schema) error {
	if m.generator == nil {
		return fmt.Errorf("file-based migrations not configured")
	}

	migration, err := m.generator.GenerateMigration(name, schemas)
	if err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	// Write migration to disk
	if err := m.fileManager.WriteMigration(migration); err != nil {
		return fmt.Errorf("failed to write migration: %w", err)
	}

	log.Printf("Generated migration: %s_%s", migration.Version, migration.Name)
	log.Printf("  Up SQL: %s/up.sql", migration.Version+"_"+migration.Name)
	log.Printf("  Down SQL: %s/down.sql", migration.Version+"_"+migration.Name)

	return nil
}

// RollbackMigration rolls back the last applied migration
func (m *Manager) RollbackMigration() error {
	if m.runner == nil {
		return fmt.Errorf("file-based migrations not configured")
	}

	ctx := context.Background()
	return m.runner.RollbackMigration(ctx)
}

// executeMigration executes a migration plan
func (m *Manager) executeMigration(version string, changes []types.SchemaChange, checksum string) error {
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
		"INSERT INTO "+MigrationsTableName+" (version, name, checksum) VALUES (?, ?, ?)",
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

	var lastMigration *types.Migration
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
	_, err = m.db.Exec("DELETE FROM " + MigrationsTableName)
	if err != nil {
		log.Printf("Warning: Failed to clear migration history: %v", err)
	}

	log.Printf("Migration reset completed.")
	return nil
}

// printMigrationPlan prints the migration plan for dry run
func (m *Manager) printMigrationPlan(changes []types.SchemaChange) {
	if m.database.GetLogger() != nil {
		m.database.GetLogger().Info("\n=== MIGRATION PLAN (DRY RUN) ===")

		for i, change := range changes {
			m.database.GetLogger().Info("%d. %s", i+1, change.Type)
			if change.TableName != "" {
				m.database.GetLogger().Info("   Table: %s", change.TableName)
			}
			if change.ColumnName != "" {
				m.database.GetLogger().Info("   Column: %s", change.ColumnName)
			}
			m.database.GetLogger().Info("   SQL: %s", change.SQL)
		}

		m.database.GetLogger().Info("\n=== END MIGRATION PLAN ===")
	}
}

// hasDestructiveChanges checks if the migration contains destructive changes
func (m *Manager) hasDestructiveChanges(changes []types.SchemaChange) bool {
	for _, change := range changes {
		if m.isDestructive(change) {
			return true
		}
	}
	return false
}

// isDestructive checks if a change is destructive
func (m *Manager) isDestructive(change types.SchemaChange) bool {
	switch change.Type {
	case types.ChangeTypeDropTable, types.ChangeTypeDropColumn:
		return true
	case types.ChangeTypeAlterColumn:
		// Column type changes can be destructive
		return true
	default:
		return false
	}
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	AppliedMigrations []types.Migration
	LastMigration     *types.Migration
	TableCount        int
	Tables            []string
}
