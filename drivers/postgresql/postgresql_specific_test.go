package postgresql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLArrayTypes(t *testing.T) {
	uri := test.GetTestDatabaseUri("postgresql")

	db, err := database.NewFromURI(uri)
	if err != nil {
		t.Skipf("Failed to create PostgreSQL database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skip("PostgreSQL test connection not available")
	}

	// Clean up any existing tables first
	pgDB, _ := db.(*PostgreSQLDB)
	cleanupTables(t, pgDB)

	// Create test database with cleanup
	td := test.NewTestDatabase(t, db, uri, func() {
		cleanupTables(t, pgDB)
		db.Close()
	})
	defer td.Cleanup()

	err = td.CreateStandardSchemas()
	require.NoError(t, err)

	// PostgreSQL supports array types
	_, err = db.Exec("ALTER TABLE users ADD COLUMN tags TEXT[]")
	require.NoError(t, err)

	// Insert user with array (include active field to satisfy NOT NULL constraint)
	_, err = db.Exec("INSERT INTO users (name, email, age, active, tags) VALUES ($1, $2, $3, $4, $5)",
		"Henry", "henry@example.com", 28, true, "{developer,golang,postgres}")
	require.NoError(t, err)

	// Query array contains
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE 'golang' = ANY(tags)").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Henry", results[0]["name"])

	// Test array length
	results = []map[string]any{}
	err = db.Raw("SELECT * FROM users WHERE array_length(tags, 1) = 3").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Test array overlap
	results = []map[string]any{}
	err = db.Raw("SELECT * FROM users WHERE tags && ARRAY['developer', 'python']").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1) // Should find Henry because 'developer' matches
}

func TestPostgreSQLJSONTypes(t *testing.T) {
	uri := test.GetTestDatabaseUri("postgresql")

	db, err := database.NewFromURI(uri)
	if err != nil {
		t.Skipf("Failed to create PostgreSQL database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skip("PostgreSQL test connection not available")
	}

	// Create test database with cleanup
	pgDB, _ := db.(*PostgreSQLDB)
	td := test.NewTestDatabase(t, db, uri, func() {
		cleanupTables(t, pgDB)
		db.Close()
	})
	defer td.Cleanup()

	err = td.CreateStandardSchemas()
	require.NoError(t, err)

	// PostgreSQL supports JSON/JSONB types
	_, err = db.Exec("ALTER TABLE users ADD COLUMN metadata JSONB")
	require.NoError(t, err)

	// Insert user with JSON
	_, err = db.Exec(`INSERT INTO users (name, email, age, metadata) 
		VALUES ($1, $2, $3, $4)`,
		"Ivy", "ivy@example.com", 32,
		`{"role": "admin", "permissions": ["read", "write", "delete"]}`)
	require.NoError(t, err)

	// Query JSON field
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE metadata->>'role' = 'admin'").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Ivy", results[0]["name"])

	// Query JSON array contains - use @> operator
	err = db.Raw("SELECT * FROM users WHERE metadata->'permissions' @> '\"delete\"'::jsonb").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Test JSON path exists - use jsonb_exists for clarity
	err = db.Raw("SELECT * FROM users WHERE jsonb_exists(metadata, 'role')").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// Test nested JSON query
	_, err = db.Exec(`INSERT INTO users (name, email, age, metadata) 
		VALUES ($1, $2, $3, $4)`,
		"Jack", "jack@example.com", 28,
		`{"profile": {"department": "engineering", "level": 5}}`)
	require.NoError(t, err)

	results = []map[string]any{}
	err = db.Raw("SELECT * FROM users WHERE metadata->'profile'->>'department' = 'engineering'").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Jack", results[0]["name"])
}

func TestPostgreSQLCaseSensitivity(t *testing.T) {
	uri := test.GetTestDatabaseUri("postgresql")

	db, err := database.NewFromURI(uri)
	if err != nil {
		t.Skipf("Failed to create PostgreSQL database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skip("PostgreSQL test connection not available")
	}

	// Create test database with cleanup
	pgDB, _ := db.(*PostgreSQLDB)
	td := test.NewTestDatabase(t, db, uri, func() {
		cleanupTables(t, pgDB)
		db.Close()
	})
	defer td.Cleanup()

	err = td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	// Test case sensitivity in string comparisons
	// PostgreSQL is case-sensitive by default
	User := db.Model("User")
	var users []test.TestUser
	err = User.Select().
		WhereCondition(User.Where("name").Equals("alice")). // lowercase
		FindMany(ctx, &users)

	// PostgreSQL should NOT find the user due to case sensitivity
	require.NoError(t, err)
	assert.Len(t, users, 0)

	// Test with correct case
	users = []test.TestUser{}
	err = User.Select().
		WhereCondition(User.Where("name").Equals("Alice")). // correct case
		FindMany(ctx, &users)

	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].Name)

	// Test case-insensitive search with ILIKE
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE name ILIKE $1", "alice").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Alice", results[0]["name"])
}
