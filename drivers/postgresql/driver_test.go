package postgresql

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to get test database config
func getTestConfig() types.Config {
	// Check environment variables for PostgreSQL connection
	host := os.Getenv("POSTGRES_TEST_HOST")
	if host == "" {
		host = "localhost"
	}
	
	user := os.Getenv("POSTGRES_TEST_USER")
	if user == "" {
		user = "testuser"
	}
	
	password := os.Getenv("POSTGRES_TEST_PASSWORD")
	if password == "" {
		password = "testpass"
	}
	
	database := os.Getenv("POSTGRES_TEST_DATABASE")
	if database == "" {
		database = "testdb"
	}
	
	return types.Config{
		Type:     "postgresql",
		Host:     host,
		Port:     5432,
		User:     user,
		Password: password,
		Database: database,
	}
}

// Helper to check if PostgreSQL is available
func skipIfPostgreSQLNotAvailable(t *testing.T) {
	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	
	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Skipf("Cannot connect to PostgreSQL: %v", err)
	}
	db.Close()
}

func TestPostgreSQLDB_NewPostgreSQLDB(t *testing.T) {
	config := types.Config{
		Type:     "postgresql",
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
	}

	db, err := NewPostgreSQLDB(config)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, config, db.config)
	assert.NotNil(t, db.fieldMapper)
	assert.NotNil(t, db.schemas)
}

func TestPostgreSQLDB_Connect(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, db.db)

	// Test ping
	err = db.Ping(ctx)
	assert.NoError(t, err)

	// Close connection
	err = db.Close()
	assert.NoError(t, err)
}

func TestPostgreSQLDB_RegisterSchema(t *testing.T) {
	db, err := NewPostgreSQLDB(types.Config{})
	require.NoError(t, err)

	// Create a test schema
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build())

	// Register schema
	err = db.RegisterSchema("User", userSchema)
	assert.NoError(t, err)

	// Get schema back
	retrievedSchema, err := db.GetSchema("User")
	assert.NoError(t, err)
	assert.Equal(t, userSchema, retrievedSchema)

	// Test error cases
	err = db.RegisterSchema("Invalid", nil)
	assert.Error(t, err)
}

func TestPostgreSQLDB_FieldMapping(t *testing.T) {
	db, err := NewPostgreSQLDB(types.Config{})
	require.NoError(t, err)

	// Register a schema with field mapping
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().Build()).
		AddField(schema.NewField("firstName").String().Map("first_name").Build()).
		AddField(schema.NewField("lastName").String().Map("last_name").Build())

	err = db.RegisterSchema("User", userSchema)
	require.NoError(t, err)

	// Test table name resolution
	tableName, err := db.ResolveTableName("User")
	assert.NoError(t, err)
	assert.Equal(t, "users", tableName)

	// Test field name resolution
	columnName, err := db.ResolveFieldName("User", "firstName")
	assert.NoError(t, err)
	assert.Equal(t, "first_name", columnName)

	// Test multiple field resolution
	columns, err := db.ResolveFieldNames("User", []string{"firstName", "lastName"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"first_name", "last_name"}, columns)
}

func TestPostgreSQLDB_CreateModel(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Drop table if exists
	_, _ = db.Exec("DROP TABLE IF EXISTS test_users")

	// Register schema
	userSchema := schema.New("TestUser").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build())
	userSchema.TableName = "test_users"

	err = db.RegisterSchema("TestUser", userSchema)
	require.NoError(t, err)

	// Create the model/table
	err = db.CreateModel(ctx, "TestUser")
	assert.NoError(t, err)

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'test_users'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "test_users", tableName)

	// Clean up
	err = db.DropModel(ctx, "TestUser")
	assert.NoError(t, err)
}

func TestPostgreSQLDB_GenerateColumnSQL(t *testing.T) {
	db, err := NewPostgreSQLDB(types.Config{})
	require.NoError(t, err)

	testCases := []struct {
		name     string
		field    schema.Field
		expected string
	}{
		{
			name:     "Simple string field",
			field:    schema.NewField("name").String().Build(),
			expected: `"name" VARCHAR(255) NOT NULL`,
		},
		{
			name:     "Primary key with auto increment",
			field:    schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build(),
			expected: `"id" BIGSERIAL`,
		},
		{
			name:     "Nullable field",
			field:    schema.NewField("description").String().Nullable().Build(),
			expected: `"description" VARCHAR(255)`,
		},
		{
			name:     "Unique field",
			field:    schema.NewField("email").String().Unique().Build(),
			expected: `"email" VARCHAR(255) NOT NULL UNIQUE`,
		},
		{
			name:     "Field with default",
			field:    schema.NewField("active").Bool().Default(true).Build(),
			expected: `"active" BOOLEAN NOT NULL DEFAULT TRUE`,
		},
		{
			name:     "Field with column mapping",
			field:    schema.NewField("firstName").String().Map("first_name").Build(),
			expected: `"first_name" VARCHAR(255) NOT NULL`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := db.generateColumnSQL(tc.field)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, sql)
		})
	}
}

func TestPostgreSQLDB_GenerateCreateTableSQL(t *testing.T) {
	db, err := NewPostgreSQLDB(types.Config{})
	require.NoError(t, err)

	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("createdAt").DateTime().Build())

	sql, err := db.generateCreateTableSQL(userSchema)
	require.NoError(t, err)

	// Check that SQL contains expected parts
	assert.Contains(t, sql, `CREATE TABLE IF NOT EXISTS "users"`)
	assert.Contains(t, sql, `"id" BIGSERIAL`)
	assert.Contains(t, sql, `"name" VARCHAR(255) NOT NULL`)
	assert.Contains(t, sql, `"email" VARCHAR(255) NOT NULL UNIQUE`)
	assert.Contains(t, sql, `"created_at" TIMESTAMP NOT NULL`)
	assert.Contains(t, sql, `PRIMARY KEY ("id")`)
}

func TestPostgreSQLDB_Transaction(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)
	
	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
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
			id SERIAL PRIMARY KEY,
			value VARCHAR(100)
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE test_transaction")

	// Test successful transaction
	err = db.Transaction(ctx, func(tx types.Transaction) error {
		_, err := tx.Raw("INSERT INTO test_transaction (value) VALUES ($1)", "test1").Exec(ctx)
		return err
	})
	assert.NoError(t, err)

	// Verify data was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_transaction").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Test rollback on error
	err = db.Transaction(ctx, func(tx types.Transaction) error {
		_, err := tx.Raw("INSERT INTO test_transaction (value) VALUES ($1)", "test2").Exec(ctx)
		if err != nil {
			return err
		}
		return fmt.Errorf("force rollback")
	})
	assert.Error(t, err)

	// Verify rollback worked
	err = db.QueryRow("SELECT COUNT(*) FROM test_transaction").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Should still be 1
}

func TestPostgreSQLDB_FieldTypeMapping(t *testing.T) {
	db, err := NewPostgreSQLDB(types.Config{})
	require.NoError(t, err)

	testCases := []struct {
		fieldType schema.FieldType
		expected  string
	}{
		{schema.FieldTypeString, "VARCHAR(255)"},
		{schema.FieldTypeInt, "INTEGER"},
		{schema.FieldTypeInt64, "BIGINT"},
		{schema.FieldTypeFloat, "DOUBLE PRECISION"},
		{schema.FieldTypeBool, "BOOLEAN"},
		{schema.FieldTypeDateTime, "TIMESTAMP"},
		{schema.FieldTypeJSON, "JSONB"},
		{schema.FieldTypeDecimal, "DECIMAL(10,2)"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := db.mapFieldTypeToSQL(tc.fieldType)
			assert.Equal(t, tc.expected, result)
		})
	}
}