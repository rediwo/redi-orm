package mysql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLCaseSensitivity(t *testing.T) {
	config := test.GetTestConfig("mysql")
	if config.Host == "" {
		t.Skip("MySQL test connection not configured")
	}

	db, err := NewMySQLDB(config)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)

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
	// MySQL is case-insensitive by default
	User := db.Model("User")
	var users []test.TestUser
	err = User.Select().
		WhereCondition(User.Where("name").Equals("alice")). // lowercase
		FindMany(ctx, &users)
	
	// MySQL should find the user despite case difference
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].Name) // Actual name is capitalized

	// Test with email as well
	users = []test.TestUser{}
	err = User.Select().
		WhereCondition(User.Where("email").Equals("ALICE@EXAMPLE.COM")). // uppercase
		FindMany(ctx, &users)
	
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice@example.com", users[0].Email) // Actual email is lowercase

	// Test LIKE is also case-insensitive
	users = []test.TestUser{}
	err = User.Select().
		WhereCondition(User.Where("name").Contains("LIC")). // uppercase
		FindMany(ctx, &users)
	
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].Name)
}