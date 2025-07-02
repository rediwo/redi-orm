package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// FileManager handles migration files on disk
type FileManager struct {
	baseDir string
}

// NewFileManager creates a new file manager
func NewFileManager(baseDir string) *FileManager {
	return &FileManager{
		baseDir: baseDir,
	}
}

// EnsureDirectory ensures the migrations directory exists
func (f *FileManager) EnsureDirectory() error {
	return os.MkdirAll(f.baseDir, 0755)
}

// WriteMigration writes a migration to disk
func (f *FileManager) WriteMigration(migration *types.MigrationFile) error {
	// Ensure base directory exists
	if err := f.EnsureDirectory(); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Create migration directory
	dirName := fmt.Sprintf("%s_%s", migration.Version, sanitizeName(migration.Name))
	migrationDir := filepath.Join(f.baseDir, dirName)

	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory: %w", err)
	}

	// Write up.sql
	upPath := filepath.Join(migrationDir, "up.sql")
	if err := os.WriteFile(upPath, []byte(migration.UpSQL), 0644); err != nil {
		return fmt.Errorf("failed to write up.sql: %w", err)
	}

	// Write down.sql
	downPath := filepath.Join(migrationDir, "down.sql")
	if err := os.WriteFile(downPath, []byte(migration.DownSQL), 0644); err != nil {
		return fmt.Errorf("failed to write down.sql: %w", err)
	}

	// Write metadata.json
	metadataPath := filepath.Join(migrationDir, "metadata.json")
	metadataJSON, err := json.MarshalIndent(migration.Metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("failed to write metadata.json: %w", err)
	}

	return nil
}

// ReadMigration reads a migration from disk
func (f *FileManager) ReadMigration(version string) (*types.MigrationFile, error) {
	// Find migration directory
	entries, err := os.ReadDir(f.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationDir string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), version+"_") {
			migrationDir = filepath.Join(f.baseDir, entry.Name())
			break
		}
	}

	if migrationDir == "" {
		return nil, fmt.Errorf("migration %s not found", version)
	}

	// Read files
	upSQL, err := os.ReadFile(filepath.Join(migrationDir, "up.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read up.sql: %w", err)
	}

	downSQL, err := os.ReadFile(filepath.Join(migrationDir, "down.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read down.sql: %w", err)
	}

	metadataJSON, err := os.ReadFile(filepath.Join(migrationDir, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.json: %w", err)
	}

	var metadata types.MigrationMetadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Extract name from directory
	parts := strings.SplitN(filepath.Base(migrationDir), "_", 2)
	name := ""
	if len(parts) > 1 {
		name = parts[1]
	}

	return &types.MigrationFile{
		Version:  version,
		Name:     name,
		UpSQL:    string(upSQL),
		DownSQL:  string(downSQL),
		Metadata: metadata,
	}, nil
}

// ListMigrations returns all migrations sorted by version
func (f *FileManager) ListMigrations() ([]*types.MigrationFile, error) {
	entries, err := os.ReadDir(f.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.MigrationFile{}, nil
		}
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []*types.MigrationFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Extract version from directory name
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) == 0 {
			continue
		}

		version := parts[0]
		migration, err := f.ReadMigration(version)
		if err != nil {
			// Log but don't fail on individual migration read errors
			fmt.Printf("Warning: failed to read migration %s: %v\n", version, err)
			continue
		}

		migrations = append(migrations, migration)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// GetPendingMigrations returns migrations that haven't been applied yet
func (f *FileManager) GetPendingMigrations(appliedVersions map[string]bool) ([]*types.MigrationFile, error) {
	allMigrations, err := f.ListMigrations()
	if err != nil {
		return nil, err
	}

	var pending []*types.MigrationFile
	for _, migration := range allMigrations {
		if !appliedVersions[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// sanitizeName removes special characters from migration name
func sanitizeName(name string) string {
	// Replace spaces and special characters with underscores
	replacer := strings.NewReplacer(
		" ", "_",
		"-", "_",
		".", "_",
		"/", "_",
		"\\", "_",
	)

	sanitized := replacer.Replace(name)

	// Remove any remaining non-alphanumeric characters except underscores
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}

	return result.String()
}
