package sqlite

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteCaseSensitivity(t *testing.T) {
	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("sqlite")
	config, err := NewSQLiteURIParser().ParseURI(uri)
	require.NoError(t, err)

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)

	// Clean up any existing tables first
	cleanupTables(t, db)

	// Create test database with cleanup
	td := test.NewTestDatabase(t, db, config, func() {
		cleanupTables(t, db)
		db.Close()
	})
	defer td.Cleanup()

	err = td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	// Test case sensitivity in string comparisons
	// First let's test with raw SQL to understand SQLite's behavior
	var rawResults []map[string]any
	err = db.Raw("SELECT * FROM users WHERE name = ?", "alice").Find(ctx, &rawResults)
	require.NoError(t, err)
	t.Logf("Raw SQL query for name='alice' found %d results", len(rawResults))

	// Now test with our query builder
	User := db.Model("User")
	var users []test.TestUser

	// Test with = operator
	err = User.Select().
		WhereCondition(User.Where("name").Equals("alice")). // lowercase
		FindMany(ctx, &users)

	require.NoError(t, err)
	t.Logf("Query builder for name='alice' found %d results", len(users))

	// SQLite's behavior depends on the column collation
	// If the query builder finds 0 results, it means it's case-sensitive
	// If it finds 1 result, it means it's case-insensitive
	if len(users) == 0 {
		// Case-sensitive behavior
		t.Log("SQLite is case-sensitive for = operator")
	} else {
		// Case-insensitive behavior
		assert.Len(t, users, 1)
		assert.Equal(t, "Alice", users[0].Name)
		t.Log("SQLite is case-insensitive for = operator")
	}

	// Test with LIKE operator - case-sensitive
	users = []test.TestUser{}
	err = User.Select().
		WhereCondition(User.Where("name").Like("alice%")). // lowercase
		FindMany(ctx, &users)

	require.NoError(t, err)
	t.Logf("LIKE 'alice%%' found %d results", len(users))

	// Now test with raw SQL LIKE
	rawResults = []map[string]any{}
	err = db.Raw("SELECT * FROM users WHERE name LIKE ?", "alice%").Find(ctx, &rawResults)
	require.NoError(t, err)
	t.Logf("Raw SQL LIKE 'alice%%' found %d results", len(rawResults))

	// SQLite LIKE is case-insensitive by default unless PRAGMA case_sensitive_like is ON
	// So we should handle both cases
	if len(users) == 0 {
		t.Log("LIKE is case-sensitive in this SQLite instance")
	} else {
		t.Log("LIKE is case-insensitive in this SQLite instance")
		assert.Len(t, users, 1)
		assert.Equal(t, "Alice", users[0].Name)
	}

	// Test with correct case
	users = []test.TestUser{}
	err = User.Select().
		WhereCondition(User.Where("name").Equals("Alice")). // correct case
		FindMany(ctx, &users)

	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].Name)

	// Test case-insensitive search with COLLATE NOCASE
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE name = ? COLLATE NOCASE", "alice").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Alice", results[0]["name"])
}
