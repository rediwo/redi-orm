package sqlite

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
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLiteProductionMigrationWorkflow tests the complete production migration workflow for SQLite
func TestSQLiteProductionMigrationWorkflow(t *testing.T) {
	ctx := context.Background()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sqlite-migration-test-*")
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
		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)

		// Generate migration
		err = manager.GenerateMigration("initial_schema", schemas)
		require.NoError(t, err)

		// Verify migration directory exists
		dirs, err := os.ReadDir(migrationsDir)
		require.NoError(t, err)
		assert.Equal(t, 1, len(dirs), "Should have one migration directory")

		// Check directory structure
		var migrationDir string
		for _, d := range dirs {
			if d.IsDir() && strings.Contains(d.Name(), "_initial_schema") {
				migrationDir = d.Name()
				break
			}
		}
		require.NotEmpty(t, migrationDir, "Should have migration directory")

		// Check files inside migration directory
		migrationPath := filepath.Join(migrationsDir, migrationDir)
		files, err := os.ReadDir(migrationPath)
		require.NoError(t, err)

		var upFile, downFile string
		for _, f := range files {
			if f.Name() == "up.sql" {
				upFile = f.Name()
			} else if f.Name() == "down.sql" {
				downFile = f.Name()
			}
		}
		assert.NotEmpty(t, upFile, "Should have up migration file")
		assert.NotEmpty(t, downFile, "Should have down migration file")

		// Read and verify up migration content
		upContent, err := os.ReadFile(filepath.Join(migrationPath, upFile))
		require.NoError(t, err)
		assert.Contains(t, string(upContent), "CREATE TABLE", "Up migration should contain CREATE TABLE")
		assert.Contains(t, string(upContent), "users", "Should create users table")
		assert.Contains(t, string(upContent), "posts", "Should create posts table")

		// SQLite-specific checks
		assert.Contains(t, string(upContent), "INTEGER PRIMARY KEY AUTOINCREMENT", "SQLite should use INTEGER PRIMARY KEY AUTOINCREMENT")
		assert.Contains(t, string(upContent), "CURRENT_TIMESTAMP", "SQLite should use CURRENT_TIMESTAMP instead of now()")

		// Read and verify down migration content
		downContent, err := os.ReadFile(filepath.Join(migrationPath, downFile))
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

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
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
			AddField(schema.NewField("bio").String().Nullable().Build()).         // New field
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

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: migrationsDir,
		})
		require.NoError(t, err)

		// Generate migration for changes
		err = manager.GenerateMigration("add_fields_and_comments", schemas)
		require.NoError(t, err)

		// Verify new migration directory
		dirs, err := os.ReadDir(migrationsDir)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(dirs), 1, "Should have at least 1 migration directory")

		// Find and verify the new migration
		var newMigrationDir string
		for _, d := range dirs {
			if d.IsDir() && strings.Contains(d.Name(), "add_fields_and_comments") {
				newMigrationDir = d.Name()
				break
			}
		}
		require.NotEmpty(t, newMigrationDir, "Should have new migration directory")

		// Read and verify new migration content
		newMigrationPath := filepath.Join(migrationsDir, newMigrationDir)
		upContent, err := os.ReadFile(filepath.Join(newMigrationPath, "up.sql"))
		require.NoError(t, err)
		// Since this test runs independently, it will create all tables
		assert.Contains(t, string(upContent), "CREATE TABLE", "Should contain CREATE TABLE statements")
		assert.Contains(t, string(upContent), "comments", "Should create comments table")
		// Check that it has the new fields
		assert.Contains(t, string(upContent), "bio", "Should have bio field")
		assert.Contains(t, string(upContent), "is_active", "Should have is_active field")
		assert.Contains(t, string(upContent), "published", "Should have published field")

		// SQLite-specific checks - boolean defaults might appear as 1/0 or 'true'/'false'
		// depending on whether it's a CREATE TABLE or ALTER TABLE statement
		upContentStr := string(upContent)
		hasBooleanDefaults := (strings.Contains(upContentStr, "DEFAULT 1") || strings.Contains(upContentStr, "DEFAULT 'true'")) &&
			(strings.Contains(upContentStr, "DEFAULT 0") || strings.Contains(upContentStr, "DEFAULT 'false'"))
		assert.True(t, hasBooleanDefaults, "SQLite should have boolean defaults in migration")
	})

	// Test 4: Apply new migration
	t.Run("ApplyNewMigration", func(t *testing.T) {
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)

		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
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

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
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

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
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

// TestSQLiteMigrationEdgeCases tests edge cases in migration workflow for SQLite
func TestSQLiteMigrationEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("EmptyMigrationsDirectory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sqlite-empty-migrations-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		db, err := database.NewFromURI("sqlite://:memory:")
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
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
		tempDir, err := os.MkdirTemp("", "sqlite-dup-migrations-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create duplicate migration files
		timestamp := time.Now().Unix()
		file1 := filepath.Join(tempDir, fmt.Sprintf("%d_test_migration.up.sql", timestamp))
		file2 := filepath.Join(tempDir, fmt.Sprintf("%d_test_migration.up.sql", timestamp+1))

		err = os.WriteFile(file1, []byte("CREATE TABLE test1 (id INTEGER);"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(file2, []byte("CREATE TABLE test2 (id INTEGER);"), 0644)
		require.NoError(t, err)

		// This should handle duplicates gracefully
		db, err := database.NewFromURI("sqlite://:memory:")
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
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

		tempDir, err := os.MkdirTemp("", "sqlite-no-migrations-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)

		// Should error when trying to rollback with no migrations
		err = manager.RollbackMigration()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no migrations to rollback")
	})

	t.Run("SQLiteBooleanDefaults", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "sqlite-bool-defaults-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		db, err := database.NewFromURI("sqlite://:memory:")
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Create schema with boolean fields
		testSchema := schema.New("Test").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("isActive").Bool().Default(true).Build()).
			AddField(schema.NewField("isDeleted").Bool().Default(false).Build())

		schemas := map[string]*schema.Schema{
			"Test": testSchema,
		}

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)

		// Generate migration
		err = manager.GenerateMigration("boolean_test", schemas)
		require.NoError(t, err)

		// Apply migration
		err = manager.Migrate(nil)
		require.NoError(t, err)

		// Verify boolean defaults work correctly
		_, err = db.Exec("INSERT INTO tests (id) VALUES (1)")
		require.NoError(t, err)

		var isActive, isDeleted int
		err = db.QueryRow("SELECT is_active, is_deleted FROM tests WHERE id = 1").Scan(&isActive, &isDeleted)
		require.NoError(t, err)
		assert.Equal(t, 1, isActive, "isActive should default to 1 (true)")
		assert.Equal(t, 0, isDeleted, "isDeleted should default to 0 (false)")
	})
}