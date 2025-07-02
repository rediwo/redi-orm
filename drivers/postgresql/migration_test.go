package postgresql

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

// TestPostgreSQLProductionMigrationWorkflow tests the complete production migration workflow for PostgreSQL
func TestPostgreSQLProductionMigrationWorkflow(t *testing.T) {
	// Skip if PostgreSQL is not available
	pgHost := os.Getenv("POSTGRES_HOST")
	if pgHost == "" {
		t.Skip("POSTGRES_HOST not set, skipping PostgreSQL migration tests")
	}

	pgUser := os.Getenv("POSTGRES_USER")
	if pgUser == "" {
		pgUser = "postgres"
	}

	pgPass := os.Getenv("POSTGRES_PASSWORD")
	pgDB := os.Getenv("POSTGRES_DB")
	if pgDB == "" {
		pgDB = "test_migrations"
	}

	dbURI := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgDB)
	ctx := context.Background()

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "pg-migration-test-*")
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

		// PostgreSQL-specific checks
		assert.Contains(t, string(upContent), "SERIAL PRIMARY KEY", "PostgreSQL should use SERIAL for auto-increment")
		assert.Contains(t, string(upContent), "DEFAULT now()", "PostgreSQL should use now() function")
		assert.Contains(t, string(upContent), "VARCHAR", "PostgreSQL should use VARCHAR for strings")

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
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name IN ('users', 'posts')
		`).Scan(&tableCount)
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
		
		// PostgreSQL-specific checks for boolean fields
		assert.Contains(t, string(upContent), "BOOLEAN", "PostgreSQL should use BOOLEAN type")
		assert.Contains(t, string(upContent), "DEFAULT true", "PostgreSQL should use lowercase true")
		assert.Contains(t, string(upContent), "DEFAULT false", "PostgreSQL should use lowercase false")
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
			_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		}
	})
}

// TestPostgreSQLMigrationEdgeCases tests edge cases specific to PostgreSQL
func TestPostgreSQLMigrationEdgeCases(t *testing.T) {
	// Skip if PostgreSQL is not available
	pgHost := os.Getenv("POSTGRES_HOST")
	if pgHost == "" {
		t.Skip("POSTGRES_HOST not set, skipping PostgreSQL migration tests")
	}

	pgUser := os.Getenv("POSTGRES_USER")
	if pgUser == "" {
		pgUser = "postgres"
	}

	pgPass := os.Getenv("POSTGRES_PASSWORD")
	pgDB := os.Getenv("POSTGRES_DB")
	if pgDB == "" {
		pgDB = "test_migrations"
	}

	dbURI := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgDB)
	ctx := context.Background()

	t.Run("PostgreSQLSerialSequences", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "pg-serial-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Create schema with multiple auto-increment fields
		testSchema := schema.New("Test").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("sequenceNum").Int().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build())

		schemas := map[string]*schema.Schema{
			"Test": testSchema,
		}

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)

		// Generate migration
		err = manager.GenerateMigration("serial_test", schemas)
		require.NoError(t, err)

		// Apply migration
		err = manager.Migrate(nil)
		require.NoError(t, err)

		// Check that sequences were created
		var sequenceCount int
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM pg_sequences 
			WHERE schemaname = 'public' 
			AND sequencename LIKE 'tests_%_seq'
		`).Scan(&sequenceCount)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, sequenceCount, 1, "Should have created at least one sequence")

		// Cleanup
		_, _ = db.Exec("DROP TABLE IF EXISTS tests CASCADE")
		_, _ = db.Exec("DROP TABLE IF EXISTS redi_migrations CASCADE")
	})

	t.Run("PostgreSQLCaseInsensitivity", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "pg-case-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Create schema with mixed case field names
		testSchema := schema.New("MixedCaseTable").
			AddField(schema.NewField("ID").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("FirstName").String().Build()).
			AddField(schema.NewField("LastName").String().Build()).
			AddField(schema.NewField("EmailAddress").String().Unique().Build())

		schemas := map[string]*schema.Schema{
			"MixedCaseTable": testSchema,
		}

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)

		// Generate and apply migration
		err = manager.GenerateMigration("case_test", schemas)
		require.NoError(t, err)

		err = manager.Migrate(nil)
		require.NoError(t, err)

		// Verify table and columns were created with lowercase names
		var tableName string
		err = db.QueryRow(`
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND lower(table_name) = 'mixed_case_tables'
		`).Scan(&tableName)
		require.NoError(t, err)
		assert.Equal(t, "mixed_case_tables", tableName, "Table name should be lowercase with underscores")

		// Check column names
		var columnCount int
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'mixed_case_tables'
			AND column_name IN ('id', 'first_name', 'last_name', 'email_address')
		`).Scan(&columnCount)
		require.NoError(t, err)
		assert.Equal(t, 4, columnCount, "All columns should exist with snake_case names")

		// Cleanup
		_, _ = db.Exec("DROP TABLE IF EXISTS mixed_case_tables CASCADE")
		_, _ = db.Exec("DROP TABLE IF EXISTS redi_migrations CASCADE")
	})

	t.Run("PostgreSQLArrayTypes", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "pg-array-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		db, err := database.NewFromURI(dbURI)
		require.NoError(t, err)
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Create schema with JSON field (which could be used for arrays)
		testSchema := schema.New("Test").
			AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("tags").JSON().Build()).
			AddField(schema.NewField("metadata").JSON().Nullable().Build())

		schemas := map[string]*schema.Schema{
			"Test": testSchema,
		}

		manager, err := migration.NewManager(db, types.MigrationOptions{
			Mode:          types.MigrationModeFile,
			MigrationsDir: tempDir,
		})
		require.NoError(t, err)

		// Generate migration
		err = manager.GenerateMigration("json_test", schemas)
		require.NoError(t, err)

		// Read migration to verify JSON type usage
		dirs, err := os.ReadDir(tempDir)
		require.NoError(t, err)
		var migrationDir string
		for _, d := range dirs {
			if d.IsDir() && strings.Contains(d.Name(), "json_test") {
				migrationDir = d.Name()
				break
			}
		}
		require.NotEmpty(t, migrationDir)

		upContent, err := os.ReadFile(filepath.Join(tempDir, migrationDir, "up.sql"))
		require.NoError(t, err)
		assert.Contains(t, string(upContent), "JSONB", "PostgreSQL should use JSONB for JSON fields")

		// Apply migration
		err = manager.Migrate(nil)
		require.NoError(t, err)

		// Verify column types
		var dataType string
		err = db.QueryRow(`
			SELECT data_type 
			FROM information_schema.columns 
			WHERE table_schema = 'public' 
			AND table_name = 'tests'
			AND column_name = 'tags'
		`).Scan(&dataType)
		require.NoError(t, err)
		assert.Equal(t, "jsonb", dataType, "Should use JSONB type for JSON fields")

		// Cleanup
		_, _ = db.Exec("DROP TABLE IF EXISTS tests CASCADE")
		_, _ = db.Exec("DROP TABLE IF EXISTS redi_migrations CASCADE")
	})
}