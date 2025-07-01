package migration

import (
	"database/sql"
	"fmt"
	"time"
)

// HistoryManager manages migration history in the database
type HistoryManager struct {
	db *sql.DB
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(db *sql.DB) *HistoryManager {
	return &HistoryManager{db: db}
}

// EnsureMigrationTable creates the migrations table if it doesn't exist
func (h *HistoryManager) EnsureMigrationTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS ` + MigrationsTableName + ` (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version VARCHAR(255) NOT NULL UNIQUE,
		name VARCHAR(255),
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		checksum VARCHAR(64)
	)`

	_, err := h.db.Exec(query)
	return err
}

// GetAppliedMigrations returns all applied migrations
func (h *HistoryManager) GetAppliedMigrations() ([]Migration, error) {
	query := `SELECT id, version, name, applied_at, checksum FROM ` + MigrationsTableName + ` ORDER BY id`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var m Migration
		err := rows.Scan(&m.ID, &m.Version, &m.Name, &m.AppliedAt, &m.Checksum)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	return migrations, rows.Err()
}

// RecordMigration records a new migration in the history
func (h *HistoryManager) RecordMigration(version, name, checksum string) error {
	query := `INSERT INTO ` + MigrationsTableName + ` (version, name, checksum) VALUES (?, ?, ?)`
	_, err := h.db.Exec(query, version, name, checksum)
	return err
}

// IsMigrationApplied checks if a migration version has been applied
func (h *HistoryManager) IsMigrationApplied(version string) (bool, error) {
	query := `SELECT COUNT(*) FROM ` + MigrationsTableName + ` WHERE version = ?`

	var count int
	err := h.db.QueryRow(query, version).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetLastMigration returns the most recent migration
func (h *HistoryManager) GetLastMigration() (*Migration, error) {
	query := `SELECT id, version, name, applied_at, checksum FROM ` + MigrationsTableName + ` ORDER BY id DESC LIMIT 1`

	var m Migration
	err := h.db.QueryRow(query).Scan(&m.ID, &m.Version, &m.Name, &m.AppliedAt, &m.Checksum)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// RemoveMigration removes a migration from the history (for rollback)
func (h *HistoryManager) RemoveMigration(version string) error {
	query := `DELETE FROM ` + MigrationsTableName + ` WHERE version = ?`
	_, err := h.db.Exec(query, version)
	return err
}

// GenerateVersion generates a new migration version based on timestamp
func GenerateVersion() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
