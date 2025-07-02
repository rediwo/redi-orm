package mysql

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/migration"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMySQLProductionMigrationWorkflow tests the complete production migration workflow for MySQL
func TestMySQLProductionMigrationWorkflow(t *testing.T) {
	// Skip if MySQL is not available
	mysqlHost := os.Getenv("MYSQL_HOST")
	if mysqlHost == "" {
		t.Skip("MYSQL_HOST not set, skipping MySQL migration tests")
	}

	mysqlUser := os.Getenv("MYSQL_USER")
	if mysqlUser == "" {
		mysqlUser = "root"
	}

	mysqlPass := os.Getenv("MYSQL_PASSWORD")
	mysqlDB := os.Getenv("MYSQL_DATABASE")
	if mysqlDB == "" {
		mysqlDB = "test_migrations"
	}

	dbURI := fmt.Sprintf("mysql://%s:%s@%s/%s", mysqlUser, mysqlPass, mysqlHost, mysqlDB)
	ctx := context.Background()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "mysql-migration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	migrationsDir := filepath.Join(tempDir, "migrations")
	err = os.MkdirAll(migrationsDir, 0755)
	require.NoError(t, err)

	// Test 1: Initial migration generation
	t.Run("GenerateInitialMigration", func(t *testing.T) {
		// Create database connection
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

		// MySQL-specific checks
		assert.Contains(t, string(upContent), "INT AUTO_INCREMENT", "MySQL should use INT AUTO_INCREMENT")
		assert.Contains(t, string(upContent), "DEFAULT CURRENT_TIMESTAMP", "MySQL should use CURRENT_TIMESTAMP")
		assert.Contains(t, string(upContent), "ENGINE=InnoDB", "MySQL should specify InnoDB engine")

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
		err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN ('users', 'posts')").Scan(&tableCount)
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
		
		// MySQL-specific checks for boolean fields
		assert.Contains(t, string(upContent), "BOOLEAN", "MySQL should use BOOLEAN type")
		assert.Contains(t, string(upContent), "DEFAULT TRUE", "MySQL should use TRUE for boolean default")
		assert.Contains(t, string(upContent), "DEFAULT FALSE", "MySQL should use FALSE for boolean default")
	})

	// Test 4: Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)

		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Clean up test tables
		tables := []string{"redi_migrations", "users", "posts", "comments"}
		for _, table := range tables {
			_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		}
	})
}

// TestMySQLMigrationEdgeCases tests edge cases specific to MySQL
func TestMySQLMigrationEdgeCases(t *testing.T) {
	// Skip if MySQL is not available
	mysqlHost := os.Getenv("MYSQL_HOST")
	if mysqlHost == "" {
		t.Skip("MYSQL_HOST not set, skipping MySQL migration tests")
	}

	mysqlUser := os.Getenv("MYSQL_USER")
	if mysqlUser == "" {
		mysqlUser = "root"
	}

	mysqlPass := os.Getenv("MYSQL_PASSWORD")
	mysqlDB := os.Getenv("MYSQL_DATABASE")
	if mysqlDB == "" {
		mysqlDB = "test_migrations"
	}

	dbURI := fmt.Sprintf("mysql://%s:%s@%s/%s", mysqlUser, mysqlPass, mysqlHost, mysqlDB)
	ctx := context.Background()

	t.Run("MySQLCharsetAndCollation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "mysql-charset-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Create schema with string fields
		testSchema := schema.New("Test").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("description").String().Nullable().Build())

		schemas := map[string]*schema.Schema{
			"Test": testSchema,
		}

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)

		// Generate migration
		err = manager.GenerateMigration("charset_test", schemas)
		require.NoError(t, err)

		// Apply migration
		err = manager.Migrate(nil)
		require.NoError(t, err)

		// Check table charset
		var charset, collation string
		err = db.QueryRow(`
			SELECT CHARACTER_SET_NAME, COLLATION_NAME 
			FROM information_schema.TABLES 
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'tests'
		`).Scan(&charset, &collation)
		require.NoError(t, err)
		assert.Equal(t, "utf8mb4", charset, "Table should use utf8mb4 charset")
		assert.Contains(t, collation, "utf8mb4", "Table should use utf8mb4 collation")

		// Cleanup
		_, _ = db.Exec("DROP TABLE IF EXISTS tests")
		_, _ = db.Exec("DROP TABLE IF EXISTS redi_migrations")
	})

	t.Run("MySQLIndexNameLength", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "mysql-index-length-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Create schema with very long field names
		testSchema := schema.New("VeryLongTableNameForTestingIndexNameLimits").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("veryLongFieldNameThatMightCauseIssuesWithIndexNaming").String().Build()).
			AddIndex(schema.Index{
				Fields: []string{"veryLongFieldNameThatMightCauseIssuesWithIndexNaming"},
			})

		schemas := map[string]*schema.Schema{
			"VeryLongTableNameForTestingIndexNameLimits": testSchema,
		}

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)

		// Generate and apply migration
		err = manager.GenerateMigration("long_names_test", schemas)
		require.NoError(t, err)

		err = manager.Migrate(nil)
		require.NoError(t, err)

		// Verify index was created (MySQL will truncate long names)
		var indexCount int
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM information_schema.STATISTICS 
			WHERE TABLE_SCHEMA = DATABASE() 
			AND TABLE_NAME = 'very_long_table_name_for_testing_index_name_limits'
			AND INDEX_NAME != 'PRIMARY'
		`).Scan(&indexCount)
		require.NoError(t, err)
		assert.Greater(t, indexCount, 0, "Should have created non-primary index")

		// Cleanup
		_, _ = db.Exec("DROP TABLE IF EXISTS very_long_table_name_for_testing_index_name_limits")
		_, _ = db.Exec("DROP TABLE IF EXISTS redi_migrations")
	})
}