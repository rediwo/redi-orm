package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

func TestNewFileManager(t *testing.T) {
	fm := NewFileManager("/tmp/migrations")
	if fm == nil {
		t.Error("NewFileManager returned nil")
	}
	if fm.baseDir != "/tmp/migrations" {
		t.Errorf("NewFileManager baseDir = %v, want /tmp/migrations", fm.baseDir)
	}
}

func TestFileManager_EnsureDirectory(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "migrations")

	fm := NewFileManager(testDir)

	// Directory shouldn't exist yet
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("Directory already exists")
	}

	// Ensure directory
	err := fm.EnsureDirectory()
	if err != nil {
		t.Errorf("EnsureDirectory() error = %v", err)
	}

	// Directory should exist now
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}

	// Ensure again should work (idempotent)
	err = fm.EnsureDirectory()
	if err != nil {
		t.Errorf("EnsureDirectory() second call error = %v", err)
	}
}

func TestFileManager_WriteMigration(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	migration := &types.MigrationFile{
		Version: "20240101120000",
		Name:    "create_users_table",
		UpSQL:   "CREATE TABLE users (id INT PRIMARY KEY);",
		DownSQL: "DROP TABLE users;",
		Metadata: types.MigrationMetadata{
			Description: "Create users table",
			Checksum:    "abc123",
		},
	}

	err := fm.WriteMigration(migration)
	if err != nil {
		t.Fatalf("WriteMigration() error = %v", err)
	}

	// Check that directory was created
	migrationDir := filepath.Join(tmpDir, "20240101120000_create_users_table")
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		t.Error("Migration directory was not created")
	}

	// Check up.sql
	upContent, err := os.ReadFile(filepath.Join(migrationDir, "up.sql"))
	if err != nil {
		t.Errorf("Failed to read up.sql: %v", err)
	}
	if string(upContent) != migration.UpSQL {
		t.Errorf("up.sql content = %v, want %v", string(upContent), migration.UpSQL)
	}

	// Check down.sql
	downContent, err := os.ReadFile(filepath.Join(migrationDir, "down.sql"))
	if err != nil {
		t.Errorf("Failed to read down.sql: %v", err)
	}
	if string(downContent) != migration.DownSQL {
		t.Errorf("down.sql content = %v, want %v", string(downContent), migration.DownSQL)
	}

	// Check metadata.json exists
	if _, err := os.Stat(filepath.Join(migrationDir, "metadata.json")); os.IsNotExist(err) {
		t.Error("metadata.json was not created")
	}
}

func TestFileManager_ReadMigration(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	// First write a migration
	original := &types.MigrationFile{
		Version: "20240101120000",
		Name:    "create_users_table",
		UpSQL:   "CREATE TABLE users (id INT PRIMARY KEY);",
		DownSQL: "DROP TABLE users;",
		Metadata: types.MigrationMetadata{
			Description: "Create users table",
			Checksum:    "abc123",
		},
	}

	err := fm.WriteMigration(original)
	if err != nil {
		t.Fatalf("WriteMigration() error = %v", err)
	}

	// Now read it back
	read, err := fm.ReadMigration("20240101120000")
	if err != nil {
		t.Fatalf("ReadMigration() error = %v", err)
	}

	if read.Version != original.Version {
		t.Errorf("ReadMigration() Version = %v, want %v", read.Version, original.Version)
	}
	if read.Name != original.Name {
		t.Errorf("ReadMigration() Name = %v, want %v", read.Name, original.Name)
	}
	if read.UpSQL != original.UpSQL {
		t.Errorf("ReadMigration() UpSQL = %v, want %v", read.UpSQL, original.UpSQL)
	}
	if read.DownSQL != original.DownSQL {
		t.Errorf("ReadMigration() DownSQL = %v, want %v", read.DownSQL, original.DownSQL)
	}

	// Test reading non-existent migration
	_, err = fm.ReadMigration("99999999999999")
	if err == nil {
		t.Error("ReadMigration() should error for non-existent migration")
	}
}

func TestFileManager_ListMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	// Write multiple migrations
	migrations := []*types.MigrationFile{
		{
			Version: "20240101120000",
			Name:    "create_users",
			UpSQL:   "CREATE TABLE users;",
			DownSQL: "DROP TABLE users;",
		},
		{
			Version: "20240102120000",
			Name:    "create_posts",
			UpSQL:   "CREATE TABLE posts;",
			DownSQL: "DROP TABLE posts;",
		},
		{
			Version: "20240103120000",
			Name:    "add_email",
			UpSQL:   "ALTER TABLE users ADD email;",
			DownSQL: "ALTER TABLE users DROP email;",
		},
	}

	for _, m := range migrations {
		err := fm.WriteMigration(m)
		if err != nil {
			t.Fatalf("WriteMigration() error = %v", err)
		}
	}

	// List migrations
	listed, err := fm.ListMigrations()
	if err != nil {
		t.Fatalf("ListMigrations() error = %v", err)
	}

	if len(listed) != len(migrations) {
		t.Errorf("ListMigrations() returned %d migrations, want %d", len(listed), len(migrations))
	}

	// Check that migrations are sorted by version
	for i := 0; i < len(listed)-1; i++ {
		if listed[i].Version >= listed[i+1].Version {
			t.Error("ListMigrations() not sorted by version")
		}
	}
}

func TestFileManager_GetPendingMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	// Write some migrations
	allMigrations := []*types.MigrationFile{
		{Version: "20240101120000", Name: "create_users"},
		{Version: "20240102120000", Name: "create_posts"},
		{Version: "20240103120000", Name: "add_email"},
	}

	for _, m := range allMigrations {
		m.UpSQL = "CREATE TABLE test;"
		m.DownSQL = "DROP TABLE test;"
		err := fm.WriteMigration(m)
		if err != nil {
			t.Fatalf("WriteMigration() error = %v", err)
		}
	}

	// Test with some applied migrations
	appliedVersions := map[string]bool{
		"20240101120000": true,
		"20240102120000": true,
	}

	pending, err := fm.GetPendingMigrations(appliedVersions)
	if err != nil {
		t.Fatalf("GetPendingMigrations() error = %v", err)
	}

	if len(pending) != 1 {
		t.Errorf("GetPendingMigrations() returned %d, want 1", len(pending))
	}

	if len(pending) > 0 && pending[0].Version != "20240103120000" {
		t.Errorf("GetPendingMigrations() returned wrong version: %v", pending[0].Version)
	}

	// Test with all applied
	appliedVersions = map[string]bool{
		"20240101120000": true,
		"20240102120000": true,
		"20240103120000": true,
	}

	pending, err = fm.GetPendingMigrations(appliedVersions)
	if err != nil {
		t.Fatalf("GetPendingMigrations() error = %v", err)
	}

	if len(pending) != 0 {
		t.Errorf("GetPendingMigrations() returned %d, want 0", len(pending))
	}
}
