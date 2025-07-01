package postgresql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLRawQuery_Exec(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec("DROP TABLE IF EXISTS test_raw_query")
	require.NoError(t, err)
	
	_, err = db.Exec(`
		CREATE TABLE test_raw_query (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			value INT
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_raw_query")

	t.Run("INSERT", func(t *testing.T) {
		query := db.Raw("INSERT INTO test_raw_query (name, value) VALUES ($1, $2)", "test1", 100)
		result, err := query.Exec(ctx)
		assert.NoError(t, err)
		// PostgreSQL doesn't return LastInsertID without RETURNING clause
		assert.GreaterOrEqual(t, result.RowsAffected, int64(1))
	})

	t.Run("UPDATE", func(t *testing.T) {
		query := db.Raw("UPDATE test_raw_query SET value = $1 WHERE name = $2", 200, "test1")
		result, err := query.Exec(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.RowsAffected)
	})

	t.Run("DELETE", func(t *testing.T) {
		query := db.Raw("DELETE FROM test_raw_query WHERE name = $1", "test1")
		result, err := query.Exec(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.RowsAffected)
	})
}

func TestPostgreSQLRawQuery_Find(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table and data
	_, err = db.Exec("DROP TABLE IF EXISTS test_find")
	require.NoError(t, err)
	
	_, err = db.Exec(`
		CREATE TABLE test_find (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			value INT
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_find")

	// Insert test data
	_, err = db.Exec("INSERT INTO test_find (name, value) VALUES ($1, $2), ($3, $4), ($5, $6)", 
		"test1", 100, "test2", 200, "test3", 300)
	require.NoError(t, err)

	t.Run("Find multiple rows", func(t *testing.T) {
		type TestRow struct {
			ID    int
			Name  string
			Value int
		}

		var results []TestRow
		query := db.Raw("SELECT id, name, value FROM test_find ORDER BY id")
		err := query.Find(ctx, &results)
		assert.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, "test1", results[0].Name)
		assert.Equal(t, 100, results[0].Value)
	})

	t.Run("Find with pointers", func(t *testing.T) {
		type TestRow struct {
			ID    int
			Name  string
			Value int
		}

		var results []*TestRow
		query := db.Raw("SELECT id, name, value FROM test_find WHERE value >= $1 ORDER BY id", 200)
		err := query.Find(ctx, &results)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "test2", results[0].Name)
	})
}

func TestPostgreSQLRawQuery_FindOne(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create test table and data
	_, err = db.Exec("DROP TABLE IF EXISTS test_find_one")
	require.NoError(t, err)
	
	_, err = db.Exec(`
		CREATE TABLE test_find_one (
			id INT PRIMARY KEY,
			name VARCHAR(100)
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_find_one")

	// Insert test data
	_, err = db.Exec("INSERT INTO test_find_one (id, name) VALUES ($1, $2)", 1, "test1")
	require.NoError(t, err)

	t.Run("Find single row struct", func(t *testing.T) {
		type TestRow struct {
			ID   int
			Name string
		}

		var result TestRow
		query := db.Raw("SELECT id, name FROM test_find_one WHERE id = $1", 1)
		err := query.FindOne(ctx, &result)
		assert.NoError(t, err)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "test1", result.Name)
	})

	t.Run("Find single value", func(t *testing.T) {
		var name string
		query := db.Raw("SELECT name FROM test_find_one WHERE id = $1", 1)
		err := query.FindOne(ctx, &name)
		assert.NoError(t, err)
		assert.Equal(t, "test1", name)
	})

	t.Run("No rows found", func(t *testing.T) {
		var result string
		query := db.Raw("SELECT name FROM test_find_one WHERE id = $1", 999)
		err := query.FindOne(ctx, &result)
		assert.Error(t, err)
	})
}