package migration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// Runner executes file-based migrations
type Runner struct {
	db          types.Database
	migrator    types.DatabaseMigrator
	fileManager *FileManager
}

// NewRunner creates a new migration runner
func NewRunner(db types.Database, fileManager *FileManager) (*Runner, error) {
	migrator := db.GetMigrator()

	return &Runner{
		db:          db,
		migrator:    migrator,
		fileManager: fileManager,
	}, nil
}

// RunMigrations applies all pending migrations
func (r *Runner) RunMigrations(ctx context.Context) error {
	// Ensure migrations table exists
	if err := r.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := r.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get pending migrations
	pending, err := r.fileManager.GetPendingMigrations(applied)
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(pending) == 0 {
		utils.LogInfo("No pending migrations.")
		return nil
	}

	utils.LogInfo("Found %d pending migration(s):", len(pending))
	for _, m := range pending {
		utils.LogInfo("  - %s: %s", m.Version, m.Name)
	}

	// Apply each migration
	for _, migration := range pending {
		if err := r.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}
	}

	utils.LogInfo("Successfully applied %d migration(s).", len(pending))
	return nil
}

// RollbackMigration rolls back the last applied migration
func (r *Runner) RollbackMigration(ctx context.Context) error {
	// Ensure migrations table exists
	if err := r.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Get last applied migration
	lastMigration, err := r.getLastMigration()
	if err != nil {
		return fmt.Errorf("failed to get last migration: %w", err)
	}

	if lastMigration == nil {
		return fmt.Errorf("no migrations to rollback")
	}

	// Read migration file
	migration, err := r.fileManager.ReadMigration(lastMigration.Version)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	utils.LogInfo("Rolling back migration %s: %s", migration.Version, migration.Name)

	// Execute down SQL
	if err := r.executeSQLScript(ctx, migration.DownSQL); err != nil {
		return fmt.Errorf("failed to execute down SQL: %w", err)
	}

	// Remove from migrations table
	if err := r.removeMigrationRecord(migration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	utils.LogInfo("Rollback completed successfully.")
	return nil
}

// applyMigration applies a single migration
func (r *Runner) applyMigration(ctx context.Context, migration *types.MigrationFile) error {
	utils.LogInfo("Applying migration %s: %s", migration.Version, migration.Name)

	// Verify checksum
	if migration.Metadata.Checksum == "" {
		return fmt.Errorf("migration has no checksum")
	}

	// Execute up SQL
	if err := r.executeSQLScript(ctx, migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute up SQL: %w", err)
	}

	// Record migration
	if err := r.recordMigration(migration); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// executeSQLScript executes a SQL script
func (r *Runner) executeSQLScript(ctx context.Context, script string) error {
	// Split script into individual statements
	statements := r.splitSQLStatements(script)

	// Execute each statement
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if err := r.migrator.ApplyMigration(stmt); err != nil {
			return fmt.Errorf("failed to execute SQL: %w\nStatement: %s", err, stmt)
		}
	}

	return nil
}

// splitSQLStatements splits a SQL script into individual statements
func (r *Runner) splitSQLStatements(script string) []string {
	var statements []string
	var current strings.Builder

	lines := strings.Split(script, "\n")
	for _, line := range lines {
		// Skip comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		current.WriteString(line)
		current.WriteString("\n")

		// Check for statement terminator
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && stmt != ";" {
				statements = append(statements, strings.TrimSuffix(stmt, ";"))
			}
			current.Reset()
		}
	}

	// Add any remaining statement
	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// ensureMigrationsTable creates the migrations table if it doesn't exist
func (r *Runner) ensureMigrationsTable() error {
	sql := `CREATE TABLE IF NOT EXISTS ` + MigrationsTableName + ` (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version VARCHAR(255) NOT NULL UNIQUE,
		name VARCHAR(255) NOT NULL,
		checksum VARCHAR(255) NOT NULL,
		applied_at TIMESTAMP NOT NULL
	)`

	return r.migrator.ApplyMigration(sql)
}

// getAppliedMigrations returns a map of applied migration versions
func (r *Runner) getAppliedMigrations() (map[string]bool, error) {
	query := `SELECT version FROM ` + MigrationsTableName + ` ORDER BY version`

	rows, err := r.db.Query(query)
	if err != nil {
		// Table might not exist yet
		if strings.Contains(err.Error(), "no such table") {
			return make(map[string]bool), nil
		}
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// getLastMigration returns the last applied migration
func (r *Runner) getLastMigration() (*types.Migration, error) {
	query := `SELECT version, name, checksum, applied_at FROM ` + MigrationsTableName +
		` ORDER BY version DESC LIMIT 1`

	var m types.Migration
	err := r.db.QueryRow(query).Scan(&m.Version, &m.Name, &m.Checksum, &m.AppliedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &m, nil
}

// recordMigration records a migration in the migrations table
func (r *Runner) recordMigration(migration *types.MigrationFile) error {
	query := fmt.Sprintf(`INSERT INTO %s (version, name, checksum, applied_at) VALUES ('%s', '%s', '%s', '%s')`,
		MigrationsTableName,
		migration.Version,
		migration.Name,
		migration.Metadata.Checksum,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	return r.migrator.ApplyMigration(query)
}

// removeMigrationRecord removes a migration record
func (r *Runner) removeMigrationRecord(version string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE version = '%s'`, MigrationsTableName, version)

	return r.migrator.ApplyMigration(query)
}
