package migration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProductionMigrationWorkflow tests the complete production migration workflow
func TestProductionMigrationWorkflow(t *testing.T) {
	ctx := context.Background()
	
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "migration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	migrationsDir := filepath.Join(tempDir, "migrations")
	err = os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)
	
	dbPath := filepath.Join(tempDir, "test.db")
	dbURI := "sqlite://" + dbPath
	
	// Test 1: Initial migration generation
	t.Run("GenerateInitialMigration", func(t *testing.T) {
		// Create database
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		// Define initial schema
		userSchema := schema.New("User").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("email").String().Unique().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("createdAt").DateTime().Default("now()").Build())
		
		postSchema := schema.New("Post").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("title").String().Build()).
			AddField(schema.NewField("content").String().Nullable().Build()).
			AddField(schema.NewField("authorId").Int().Build()).
			AddField(schema.NewField("createdAt").DateTime().Default("now()").Build())
		
		schemas := map[string]*schema.Schema{
			"User": userSchema,
			"Post": postSchema,
		}
		
		// Create migration manager
		manager, err := migration.NewManager(db, migration.MigrationOptions{
			Mode:          migration.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)
		
		// Generate migration
		err = manager.GenerateMigration("initial_schema", schemas)
		require.NoError(t, err)
		
		// Verify migration files exist
		files, err := os.ReadDir(migrationsDir)
		require.NoError(t, err)
		assert.Equal(t, 2, len(files), "Should have up and down migration files")
		
		// Check file names
		var upFile, downFile string
		for _, f := range files {
			if strings.HasSuffix(f.Name(), "_initial_schema.up.sql") {
				upFile = f.Name()
			} else if strings.HasSuffix(f.Name(), "_initial_schema.down.sql") {
				downFile = f.Name()
			}
		}
		assert.NotEmpty(t, upFile, "Should have up migration file")
		assert.NotEmpty(t, downFile, "Should have down migration file")
		
		// Read and verify up migration content
		upContent, err := os.ReadFile(filepath.Join(migrationsDir, upFile))
		require.NoError(t, err)
		assert.Contains(t, string(upContent), "CREATE TABLE", "Up migration should contain CREATE TABLE")
		assert.Contains(t, string(upContent), "users", "Should create users table")
		assert.Contains(t, string(upContent), "posts", "Should create posts table")
		
		// Read and verify down migration content
		downContent, err := os.ReadFile(filepath.Join(migrationsDir, downFile))
		require.NoError(t, err)
		assert.Contains(t, string(downContent), "DROP TABLE", "Down migration should contain DROP TABLE")
	})
	
	// Test 2: Apply migrations
	t.Run("ApplyMigrations", func(t *testing.T) {
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		manager, err := migration.NewManager(db, migration.MigrationOptions{
			Mode:          migration.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)
		
		// Apply migrations
		err = manager.Migrate(nil)
		require.NoError(t, err)
		
		// Verify tables exist
		var tableCount int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('users', 'posts')").Scan(&tableCount)
		require.NoError(t, err)
		assert.Equal(t, 2, tableCount, "Should have created 2 tables")
		
		// Check migration history
		status, err := manager.GetMigrationStatus()
		require.NoError(t, err)
		assert.Equal(t, 1, len(status.AppliedMigrations), "Should have 1 applied migration")
		assert.Contains(t, status.AppliedMigrations[0].Name, "initial_schema")
	})
	
	// Test 3: Generate migration for schema changes
	t.Run("GenerateMigrationForChanges", func(t *testing.T) {
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		// Updated schema with new fields and model
		userSchema := schema.New("User").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("email").String().Unique().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("bio").String().Nullable().Build()). // New field
			AddField(schema.NewField("isActive").Bool().Default("true").Build()). // New field
			AddField(schema.NewField("createdAt").DateTime().Default("now()").Build()).
			AddField(schema.NewField("updatedAt").DateTime().Default("now()").Build()) // New field
		
		postSchema := schema.New("Post").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("title").String().Build()).
			AddField(schema.NewField("content").String().Nullable().Build()).
			AddField(schema.NewField("published").Bool().Default("false").Build()). // New field
			AddField(schema.NewField("authorId").Int().Build()).
			AddField(schema.NewField("createdAt").DateTime().Default("now()").Build())
		
		commentSchema := schema.New("Comment"). // New model
						AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
						AddField(schema.NewField("content").String().Build()).
						AddField(schema.NewField("postId").Int().Build()).
						AddField(schema.NewField("authorId").Int().Build()).
						AddField(schema.NewField("createdAt").DateTime().Default("now()").Build())
		
		schemas := map[string]*schema.Schema{
			"User":    userSchema,
			"Post":    postSchema,
			"Comment": commentSchema,
		}
		
		manager, err := migration.NewManager(db, migration.MigrationOptions{
			Mode:          migration.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)
		
		// Generate migration for changes
		err = manager.GenerateMigration("add_fields_and_comments", schemas)
		require.NoError(t, err)
		
		// Verify new migration files
		files, err := os.ReadDir(migrationsDir)
		require.NoError(t, err)
		assert.Equal(t, 4, len(files), "Should have 4 migration files (2 pairs)")
		
		// Find and verify the new migration
		var newUpFile string
		for _, f := range files {
			if strings.Contains(f.Name(), "add_fields_and_comments") && strings.HasSuffix(f.Name(), ".up.sql") {
				newUpFile = f.Name()
				break
			}
		}
		require.NotEmpty(t, newUpFile, "Should have new up migration file")
		
		// Read and verify new migration content
		upContent, err := os.ReadFile(filepath.Join(migrationsDir, newUpFile))
		require.NoError(t, err)
		assert.Contains(t, string(upContent), "ALTER TABLE", "Should contain ALTER TABLE statements")
		assert.Contains(t, string(upContent), "CREATE TABLE comments", "Should create comments table")
	})
	
	// Test 4: Apply new migration
	t.Run("ApplyNewMigration", func(t *testing.T) {
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		manager, err := migration.NewManager(db, migration.MigrationOptions{
			Mode:          migration.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)
		
		// Apply new migrations
		err = manager.Migrate(nil)
		require.NoError(t, err)
		
		// Verify comments table exists
		var exists bool
		err = db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='comments'").Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Comments table should exist")
		
		// Check migration history
		status, err := manager.GetMigrationStatus()
		require.NoError(t, err)
		assert.Equal(t, 2, len(status.AppliedMigrations), "Should have 2 applied migrations")
	})
	
	// Test 5: Rollback migration
	t.Run("RollbackMigration", func(t *testing.T) {
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		manager, err := migration.NewManager(db, migration.MigrationOptions{
			Mode:          migration.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)
		
		// Rollback last migration
		err = manager.RollbackMigration()
		require.NoError(t, err)
		
		// Verify comments table no longer exists
		var exists bool
		err = db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='comments'").Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists, "Comments table should not exist after rollback")
		
		// Check migration history
		status, err := manager.GetMigrationStatus()
		require.NoError(t, err)
		assert.Equal(t, 1, len(status.AppliedMigrations), "Should have 1 applied migration after rollback")
	})
	
	// Test 6: Re-apply rolled back migration
	t.Run("ReApplyMigration", func(t *testing.T) {
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		manager, err := migration.NewManager(db, migration.MigrationOptions{
			Mode:          migration.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)
		
		// Re-apply migrations
		err = manager.Migrate(nil)
		require.NoError(t, err)
		
		// Verify comments table exists again
		var exists bool
		err = db.QueryRow("SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='comments'").Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Comments table should exist after re-applying")
		
		// Final status check
		status, err := manager.GetMigrationStatus()
		require.NoError(t, err)
		assert.Equal(t, 2, len(status.AppliedMigrations), "Should have 2 applied migrations again")
		
		// Verify both migrations are tracked
		migrationNames := make([]string, 0, len(status.AppliedMigrations))
		for _, m := range status.AppliedMigrations {
			migrationNames = append(migrationNames, m.Name)
		}
		assert.Contains(t, migrationNames, "initial_schema")
		assert.Contains(t, migrationNames, "add_fields_and_comments")
	})
}

// TestMigrationEdgeCases tests edge cases in migration workflow
func TestMigrationEdgeCases(t *testing.T) {
	ctx := context.Background()
	
	t.Run("EmptyMigrationsDirectory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "empty-migrations-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)
		
		db, err := database.NewFromURI("sqlite://:memory:")
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		manager, err := NewManager(db, MigrationOptions{
			Mode:          MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)
		
		// Should not error on empty directory
		err = manager.Migrate(nil)
		assert.NoError(t, err)
		
		status, err := manager.GetMigrationStatus()
		require.NoError(t, err)
		assert.Equal(t, 0, len(status.AppliedMigrations))
	})
	
	t.Run("DuplicateMigrationNames", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "dup-migrations-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)
		
		// Create duplicate migration files
		timestamp := time.Now().Unix()
		file1 := filepath.Join(tempDir, fmt.Sprintf("%d_test_migration.up.sql", timestamp))
		file2 := filepath.Join(tempDir, fmt.Sprintf("%d_test_migration.up.sql", timestamp+1))
		
		err = os.WriteFile(file1, []byte("CREATE TABLE test1 (id INT);"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("CREATE TABLE test2 (id INT);"), 0644)
		require.NoError(t, err)
		
		// This should handle duplicates gracefully
		db, err := database.NewFromURI("sqlite://:memory:")
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		manager, err := NewManager(db, MigrationOptions{
			Mode:          MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)
		
		// Should apply both migrations
		err = manager.Migrate(nil)
		assert.NoError(t, err)
	})
	
	t.Run("RollbackWithNoMigrations", func(t *testing.T) {
		db, err := database.NewFromURI("sqlite://:memory:")
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()
		
		tempDir, err := os.MkdirTemp("", "no-migrations-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)
		
		manager, err := NewManager(db, MigrationOptions{
			Mode:          MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)
		
		// Should error when trying to rollback with no migrations
		err = manager.RollbackMigration()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no migrations to rollback")
	})
}

// TestMigrationFileNaming tests migration file naming conventions
func TestMigrationFileNaming(t *testing.T) {
	t.Run("ValidMigrationFileNames", func(t *testing.T) {
		validNames := []string{
			"20240101120000_initial_schema.up.sql",
			"20240101120001_add_users_table.down.sql",
			"20240101120002_update_products.up.sql",
			"1704110400_create_indexes.up.sql",
		}
		
		for _, name := range validNames {
			version, migrationName, direction := parseMigrationFileName(name)
			assert.NotEmpty(t, version, "Should extract version from %s", name)
			assert.NotEmpty(t, migrationName, "Should extract name from %s", name)
			assert.Contains(t, []string{"up", "down"}, direction, "Should extract direction from %s", name)
		}
	})
	
	t.Run("InvalidMigrationFileNames", func(t *testing.T) {
		invalidNames := []string{
			"migration.sql",
			"add_users.up.sql",
			"20240101_missing_extension",
			"not_a_migration.txt",
		}
		
		for _, name := range invalidNames {
			version, _, _ := parseMigrationFileName(name)
			assert.Empty(t, version, "Should not extract version from invalid name %s", name)
		}
	})
}

// Helper function to parse migration file names
func parseMigrationFileName(filename string) (version, name, direction string) {
	// Expected format: <timestamp>_<name>.<direction>.sql
	parts := strings.Split(filename, "_")
	if len(parts) < 2 {
		return "", "", ""
	}
	
	version = parts[0]
	
	// Extract name and direction
	remaining := strings.Join(parts[1:], "_")
	if strings.HasSuffix(remaining, ".up.sql") {
		name = strings.TrimSuffix(remaining, ".up.sql")
		direction = "up"
	} else if strings.HasSuffix(remaining, ".down.sql") {
		name = strings.TrimSuffix(remaining, ".down.sql")
		direction = "down"
	}
	
	return version, name, direction
}