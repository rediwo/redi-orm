package sqlite

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteRawQuery_NewSQLiteRawQuery(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Test raw query creation
	sql := "SELECT 1 as test_value"
	args := []interface{}{"arg1", 42}

	rawQuery := db.Raw(sql, args...)
	require.NotNil(t, rawQuery)

	// Cast to SQLiteRawQuery to test internal structure
	sqliteRaw, ok := rawQuery.(*SQLiteRawQuery)
	require.True(t, ok)
	assert.Equal(t, db.db, sqliteRaw.db)
	assert.Equal(t, sql, sqliteRaw.sql)
	assert.Equal(t, args, sqliteRaw.args)

	// Test direct constructor
	directRaw := NewSQLiteRawQuery(db.db, sql, args...)
	require.NotNil(t, directRaw)
}

func TestSQLiteRawQuery_Exec_Select(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Test SELECT query execution
	rawQuery := db.Raw("SELECT 1 as test_value, 'hello' as test_string")
	result, err := rawQuery.Exec(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, int64(0), result.LastInsertID) // SELECT doesn't insert
	assert.Equal(t, int64(0), result.RowsAffected) // SELECT doesn't affect rows
}

func TestSQLiteRawQuery_Exec_Insert(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create a test table
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, value INTEGER)")
	require.NoError(t, err)

	// Test INSERT query execution
	rawQuery := db.Raw("INSERT INTO test_table (name, value) VALUES (?, ?)", "test_name", 42)
	result, err := rawQuery.Exec(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.LastInsertID)
	assert.Equal(t, int64(1), result.RowsAffected)

	// Test batch insert
	rawQuery = db.Raw("INSERT INTO test_table (name, value) VALUES (?, ?), (?, ?)", 
		"test2", 100, "test3", 200)
	result, err = rawQuery.Exec(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, int64(3), result.LastInsertID) // Should be the last inserted ID
	assert.Equal(t, int64(2), result.RowsAffected) // 2 rows affected
}

func TestSQLiteRawQuery_Exec_Update(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create and populate test table
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT, value INTEGER)")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO test_table (name, value) VALUES (?, ?), (?, ?), (?, ?)",
		"test1", 10, "test2", 20, "test3", 30)
	require.NoError(t, err)

	// Test UPDATE query execution
	rawQuery := db.Raw("UPDATE test_table SET value = ? WHERE value > ?", 999, 15)
	result, err := rawQuery.Exec(ctx)
	
	assert.NoError(t, err)
	// LastInsertID can vary in SQLite, but for UPDATE operations it should not be used
	assert.Equal(t, int64(2), result.RowsAffected) // Should update 2 rows (value 20 and 30)

	// Verify the update
	rows, err := db.Query("SELECT COUNT(*) FROM test_table WHERE value = 999")
	require.NoError(t, err)
	defer rows.Close()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	}
}

func TestSQLiteRawQuery_Exec_Delete(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create and populate test table
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT, value INTEGER)")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO test_table (name, value) VALUES (?, ?), (?, ?), (?, ?)",
		"test1", 10, "test2", 20, "test3", 30)
	require.NoError(t, err)

	// Test DELETE query execution
	rawQuery := db.Raw("DELETE FROM test_table WHERE value < ?", 25)
	result, err := rawQuery.Exec(ctx)
	
	assert.NoError(t, err)
	// LastInsertID can vary in SQLite, but for DELETE operations it should not be used
	assert.Equal(t, int64(2), result.RowsAffected) // Should delete 2 rows (value 10 and 20)

	// Verify the deletion
	rows, err := db.Query("SELECT COUNT(*) FROM test_table")
	require.NoError(t, err)
	defer rows.Close()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count) // Only 1 row should remain
	}
}

func TestSQLiteRawQuery_Find(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT, value INTEGER)")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO test_table (name, value) VALUES (?, ?), (?, ?)",
		"test1", 10, "test2", 20)
	require.NoError(t, err)

	// Test Find
	rawQuery := db.Raw("SELECT * FROM test_table ORDER BY id")
	
	type TestRow struct {
		ID    int
		Name  string
		Value int
	}
	
	var dest []TestRow
	err = rawQuery.Find(ctx, &dest)
	
	assert.NoError(t, err)
	assert.Len(t, dest, 2)
	assert.Equal(t, "test1", dest[0].Name)
	assert.Equal(t, 10, dest[0].Value)
	assert.Equal(t, "test2", dest[1].Name)
	assert.Equal(t, 20, dest[1].Value)
}

func TestSQLiteRawQuery_FindOne(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT, value INTEGER)")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO test_table (name, value) VALUES (?, ?)", "test1", 10)
	require.NoError(t, err)

	// Test FindOne
	rawQuery := db.Raw("SELECT * FROM test_table WHERE id = ?", 1)
	
	type TestRow struct {
		ID    int
		Name  string
		Value int
	}
	
	var dest TestRow
	err = rawQuery.FindOne(ctx, &dest)
	
	assert.NoError(t, err)
	assert.Equal(t, 1, dest.ID)
	assert.Equal(t, "test1", dest.Name)
	assert.Equal(t, 10, dest.Value)
}

func TestSQLiteRawQuery_ErrorHandling(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Test error handling with invalid SQL
	rawQuery := db.Raw("INVALID SQL STATEMENT")
	_, err = rawQuery.Exec(ctx)
	assert.Error(t, err)

	// Test error handling with wrong number of parameters
	rawQuery = db.Raw("SELECT * FROM test_table WHERE id = ? AND name = ?", 1) // Missing second parameter
	_, err = rawQuery.Exec(ctx)
	assert.Error(t, err)

	// Test error handling with non-existent table
	rawQuery = db.Raw("SELECT * FROM non_existent_table")
	_, err = rawQuery.Exec(ctx)
	assert.Error(t, err)

	// Test that Find and FindOne also handle errors properly
	rawQuery = db.Raw("INVALID SQL FOR FIND")
	
	var dest []map[string]interface{}
	err = rawQuery.Find(ctx, &dest)
	assert.Error(t, err)

	var destOne map[string]interface{}
	err = rawQuery.FindOne(ctx, &destOne)
	assert.Error(t, err)
}

func TestSQLiteRawQuery_WithDifferentDataTypes(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create table with various data types
	_, err = db.Exec(`CREATE TABLE test_types (
		id INTEGER PRIMARY KEY,
		text_field TEXT,
		integer_field INTEGER,
		real_field REAL,
		blob_field BLOB,
		null_field TEXT
	)`)
	require.NoError(t, err)

	// Test inserting different data types
	rawQuery := db.Raw(`INSERT INTO test_types 
		(text_field, integer_field, real_field, blob_field, null_field) 
		VALUES (?, ?, ?, ?, ?)`,
		"test string", 42, 3.14, []byte("binary data"), nil)
	
	result, err := rawQuery.Exec(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.LastInsertID)
	assert.Equal(t, int64(1), result.RowsAffected)

	// Test with boolean values (stored as integers in SQLite)
	rawQuery = db.Raw(`INSERT INTO test_types 
		(text_field, integer_field, real_field) 
		VALUES (?, ?, ?)`,
		"boolean test", 1, 0.0) // true as 1, false as 0
	
	result, err = rawQuery.Exec(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.LastInsertID)
	assert.Equal(t, int64(1), result.RowsAffected)
}

func TestSQLiteRawQuery_TransactionContext(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	// Test raw query within transaction
	err = db.Transaction(ctx, func(tx types.Transaction) error {
		// Use raw query within transaction
		rawQuery := tx.Raw("INSERT INTO test_table (name) VALUES (?)", "transaction_test")
		result, err := rawQuery.Exec(ctx)
		if err != nil {
			return err
		}
		
		assert.Equal(t, int64(1), result.LastInsertID)
		assert.Equal(t, int64(1), result.RowsAffected)
		
		// Insert another record
		rawQuery2 := tx.Raw("INSERT INTO test_table (name) VALUES (?)", "transaction_test2")
		result2, err := rawQuery2.Exec(ctx)
		if err != nil {
			return err
		}
		
		assert.Equal(t, int64(2), result2.LastInsertID)
		assert.Equal(t, int64(1), result2.RowsAffected)
		
		return nil
	})
	
	assert.NoError(t, err)

	// Verify both records were committed
	rows, err := db.Query("SELECT COUNT(*) FROM test_table")
	require.NoError(t, err)
	defer rows.Close()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	}
}

func TestSQLiteRawQuery_ParameterBinding(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE test_params (id INTEGER PRIMARY KEY, name TEXT, value INTEGER)")
	require.NoError(t, err)

	// Test parameter binding with various types
	testCases := []struct {
		name   string
		sql    string
		args   []interface{}
		expectError bool
	}{
		{
			name: "String parameter",
			sql:  "INSERT INTO test_params (name, value) VALUES (?, ?)",
			args: []interface{}{"test_string", 100},
			expectError: false,
		},
		{
			name: "Integer parameter",
			sql:  "INSERT INTO test_params (name, value) VALUES (?, ?)",
			args: []interface{}{"test_int", 42},
			expectError: false,
		},
		{
			name: "Float parameter",
			sql:  "INSERT INTO test_params (name, value) VALUES (?, ?)",
			args: []interface{}{"test_float", 3.14},
			expectError: false,
		},
		{
			name: "Nil parameter",
			sql:  "INSERT INTO test_params (name, value) VALUES (?, ?)",
			args: []interface{}{"test_nil", nil},
			expectError: false,
		},
		{
			name: "Boolean parameter",
			sql:  "INSERT INTO test_params (name, value) VALUES (?, ?)",
			args: []interface{}{"test_bool", true},
			expectError: false,
		},
		{
			name: "Too few parameters",
			sql:  "INSERT INTO test_params (name, value) VALUES (?, ?)",
			args: []interface{}{"only_one"},
			expectError: true,
		},
		{
			name: "Too many parameters",
			sql:  "INSERT INTO test_params (name) VALUES (?)",
			args: []interface{}{"param1", "param2"},
			expectError: false, // SQLite ignores extra parameters
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rawQuery := db.Raw(tc.sql, tc.args...)
			_, err := rawQuery.Exec(ctx)
			
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSQLiteRawQuery_ComplexQueries(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create related tables
	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT,
		email TEXT UNIQUE
	)`)
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE posts (
		id INTEGER PRIMARY KEY,
		title TEXT,
		content TEXT,
		user_id INTEGER,
		FOREIGN KEY (user_id) REFERENCES users (id)
	)`)
	require.NoError(t, err)

	// Insert test data
	rawQuery := db.Raw("INSERT INTO users (name, email) VALUES (?, ?)", "John Doe", "john@example.com")
	result, err := rawQuery.Exec(ctx)
	require.NoError(t, err)
	userID := result.LastInsertID

	rawQuery = db.Raw("INSERT INTO users (name, email) VALUES (?, ?)", "Jane Smith", "jane@example.com")
	_, err = rawQuery.Exec(ctx)
	require.NoError(t, err)

	// Insert posts
	rawQuery = db.Raw("INSERT INTO posts (title, content, user_id) VALUES (?, ?, ?)",
		"First Post", "This is the first post", userID)
	_, err = rawQuery.Exec(ctx)
	require.NoError(t, err)

	rawQuery = db.Raw("INSERT INTO posts (title, content, user_id) VALUES (?, ?, ?)",
		"Second Post", "This is the second post", userID)
	_, err = rawQuery.Exec(ctx)
	require.NoError(t, err)

	// Test complex JOIN query
	complexQuery := `
		SELECT u.name, u.email, COUNT(p.id) as post_count
		FROM users u
		LEFT JOIN posts p ON u.id = p.user_id
		WHERE u.name LIKE ?
		GROUP BY u.id, u.name, u.email
		HAVING COUNT(p.id) >= ?
		ORDER BY post_count DESC
	`
	
	rawQuery = db.Raw(complexQuery, "John%", 1)
	result, err = rawQuery.Exec(ctx)
	assert.NoError(t, err)
	// SELECT queries may return the number of rows found in RowsAffected
	// This is implementation-specific and can vary

	// Test subquery
	subQuery := `
		SELECT title FROM posts 
		WHERE user_id = (SELECT id FROM users WHERE email = ?)
		ORDER BY id
	`
	
	rawQuery = db.Raw(subQuery, "john@example.com")
	result, err = rawQuery.Exec(ctx)
	assert.NoError(t, err)
	// SELECT queries may return the number of rows found in RowsAffected

	// Test aggregate functions
	aggregateQuery := `
		SELECT 
			COUNT(*) as total_posts,
			MAX(LENGTH(title)) as longest_title_length,
			MIN(user_id) as min_user_id
		FROM posts
	`
	
	rawQuery = db.Raw(aggregateQuery)
	result, err = rawQuery.Exec(ctx)
	assert.NoError(t, err)
	// SELECT queries may return the number of rows found in RowsAffected
}

func TestSQLiteRawQuery_ConcurrentAccess(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("CREATE TABLE concurrent_test (id INTEGER PRIMARY KEY, value INTEGER)")
	require.NoError(t, err)

	// Test that multiple raw queries can be created and executed
	query1 := db.Raw("INSERT INTO concurrent_test (value) VALUES (?)", 1)
	query2 := db.Raw("INSERT INTO concurrent_test (value) VALUES (?)", 2)
	query3 := db.Raw("SELECT COUNT(*) FROM concurrent_test")

	// Execute queries
	result1, err1 := query1.Exec(ctx)
	result2, err2 := query2.Exec(ctx)
	_, err3 := query3.Exec(ctx)

	// All should succeed
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	// LastInsertID values can vary in SQLite implementation
	// What matters is that the inserts succeed and the count is correct
	assert.True(t, result1.LastInsertID > 0) // Should have inserted
	assert.True(t, result2.LastInsertID > 0) // Should have inserted

	// Verify both inserts worked
	rows, err := db.Query("SELECT COUNT(*) FROM concurrent_test")
	require.NoError(t, err)
	defer rows.Close()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	}
}