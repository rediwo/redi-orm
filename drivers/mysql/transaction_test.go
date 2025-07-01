package mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLTransaction_Basic(t *testing.T) {
	skipIfMySQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewMySQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("DROP TABLE IF EXISTS test_transaction")
	require.NoError(t, err)
	
	_, err = db.Exec(`
		CREATE TABLE test_transaction (
			id INT PRIMARY KEY AUTO_INCREMENT,
			value VARCHAR(100)
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_transaction")

	t.Run("Commit", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		require.NoError(t, err)

		// Insert data
		result, err := tx.Raw("INSERT INTO test_transaction (value) VALUES (?)", "commit_test").Exec(ctx)
		assert.NoError(t, err)
		assert.Greater(t, result.LastInsertID, int64(0))

		// Commit
		err = tx.Commit(ctx)
		assert.NoError(t, err)

		// Verify data was committed
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_transaction WHERE value = 'commit_test'").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Rollback", func(t *testing.T) {
		tx, err := db.Begin(ctx)
		require.NoError(t, err)

		// Insert data
		_, err = tx.Raw("INSERT INTO test_transaction (value) VALUES (?)", "rollback_test").Exec(ctx)
		assert.NoError(t, err)

		// Rollback
		err = tx.Rollback(ctx)
		assert.NoError(t, err)

		// Verify data was not committed
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test_transaction WHERE value = 'rollback_test'").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestMySQLTransaction_Savepoint(t *testing.T) {
	skipIfMySQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewMySQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("DROP TABLE IF EXISTS test_savepoint")
	require.NoError(t, err)
	
	_, err = db.Exec(`
		CREATE TABLE test_savepoint (
			id INT PRIMARY KEY AUTO_INCREMENT,
			value VARCHAR(100)
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_savepoint")

	tx, err := db.Begin(ctx)
	require.NoError(t, err)

	// Insert first value
	_, err = tx.Raw("INSERT INTO test_savepoint (value) VALUES (?)", "value1").Exec(ctx)
	assert.NoError(t, err)

	// Create savepoint
	err = tx.Savepoint(ctx, "sp1")
	assert.NoError(t, err)

	// Insert second value
	_, err = tx.Raw("INSERT INTO test_savepoint (value) VALUES (?)", "value2").Exec(ctx)
	assert.NoError(t, err)

	// Rollback to savepoint
	err = tx.RollbackTo(ctx, "sp1")
	assert.NoError(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	assert.NoError(t, err)

	// Verify only first value was committed
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_savepoint").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	var value string
	err = db.QueryRow("SELECT value FROM test_savepoint").Scan(&value)
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestMySQLTransaction_QueryInTransaction(t *testing.T) {
	skipIfMySQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewMySQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("DROP TABLE IF EXISTS test_query_tx")
	require.NoError(t, err)
	
	_, err = db.Exec(`
		CREATE TABLE test_query_tx (
			id INT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(100),
			value INT
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_query_tx")

	tx, err := db.Begin(ctx)
	require.NoError(t, err)

	// Insert data within transaction
	_, err = tx.Raw("INSERT INTO test_query_tx (name, value) VALUES (?, ?)", "test1", 100).Exec(ctx)
	assert.NoError(t, err)

	// Query within transaction should see the data
	var count int
	err = tx.Raw("SELECT COUNT(*) FROM test_query_tx").FindOne(ctx, &count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Query with Find
	type TestRow struct {
		ID    int
		Name  string
		Value int
	}
	var results []TestRow
	err = tx.Raw("SELECT id, name, value FROM test_query_tx").Find(ctx, &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "test1", results[0].Name)

	// Rollback
	err = tx.Rollback(ctx)
	assert.NoError(t, err)

	// Verify data was not committed
	err = db.QueryRow("SELECT COUNT(*) FROM test_query_tx").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}