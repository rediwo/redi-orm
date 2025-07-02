package mysql

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
	// Check environment variables for MySQL connection
	host := os.Getenv("MYSQL_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	user := os.Getenv("MYSQL_TEST_USER")
	if user == "" {
		user = "testuser"
	}

	password := os.Getenv("MYSQL_TEST_PASSWORD")
	if password == "" {
		password = "testpass"
	}

	database := os.Getenv("MYSQL_TEST_DATABASE")
	if database == "" {
		database = "testdb"
	}

	return types.Config{
		Type:     "mysql",
		Host:     host,
		Port:     3306,
		User:     user,
		Password: password,
		Database: database,
	}
}

// Helper to check if MySQL is available
func skipIfMySQLNotAvailable(t *testing.T) {
	config := getTestConfig()
	db, err := NewMySQLDB(config)
	if err != nil {
		t.Skip("MySQL not available for testing")
	}

	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Skipf("Cannot connect to MySQL: %v", err)
	}
	db.Close()
}

func TestMySQLDB_NewMySQLDB(t *testing.T) {
	config := types.Config{
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
	}

	db, err := NewMySQLDB(config)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.Equal(t, config, db.Config)
	assert.NotNil(t, db.FieldMapper)
	assert.NotNil(t, db.Schemas)
}

func TestMySQLDB_Connect(t *testing.T) {
	skipIfMySQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewMySQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, db.DB)

	// Test ping
	err = db.Ping(ctx)
	assert.NoError(t, err)

	// Close connection
	err = db.Close()
	assert.NoError(t, err)
}

func TestMySQLDB_RegisterSchema(t *testing.T) {
	db, err := NewMySQLDB(types.Config{})
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

func TestMySQLDB_FieldMapping(t *testing.T) {
	db, err := NewMySQLDB(types.Config{})
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

func TestMySQLDB_CreateModel(t *testing.T) {
	skipIfMySQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewMySQLDB(config)
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
	err = db.QueryRow("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'test_users'").Scan(&tableName)
	assert.NoError(t, err)
	assert.Equal(t, "test_users", tableName)

	// Clean up
	err = db.DropModel(ctx, "TestUser")
	assert.NoError(t, err)
}

func TestMySQLDB_GenerateColumnSQL(t *testing.T) {
	db, err := NewMySQLDB(types.Config{})
	require.NoError(t, err)

	testCases := []struct {
		name     string
		field    schema.Field
		expected string
	}{
		{
			name:     "Simple string field",
			field:    schema.NewField("name").String().Build(),
			expected: "`name` VARCHAR(255) NOT NULL",
		},
		{
			name:     "Primary key with auto increment",
			field:    schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build(),
			expected: "`id` BIGINT NOT NULL AUTO_INCREMENT",
		},
		{
			name:     "Nullable field",
			field:    schema.NewField("description").String().Nullable().Build(),
			expected: "`description` VARCHAR(255)",
		},
		{
			name:     "Unique field",
			field:    schema.NewField("email").String().Unique().Build(),
			expected: "`email` VARCHAR(255) NOT NULL UNIQUE",
		},
		{
			name:     "Field with default",
			field:    schema.NewField("active").Bool().Default(true).Build(),
			expected: "`active` BOOLEAN NOT NULL DEFAULT TRUE",
		},
		{
			name:     "Field with column mapping",
			field:    schema.NewField("firstName").String().Map("first_name").Build(),
			expected: "`first_name` VARCHAR(255) NOT NULL",
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

func TestMySQLDB_GenerateCreateTableSQL(t *testing.T) {
	db, err := NewMySQLDB(types.Config{})
	require.NoError(t, err)

	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("createdAt").DateTime().Build())

	sql, err := db.generateCreateTableSQL(userSchema)
	require.NoError(t, err)

	// Check that SQL contains expected parts
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS `users`")
	assert.Contains(t, sql, "`id` BIGINT NOT NULL AUTO_INCREMENT")
	assert.Contains(t, sql, "`name` VARCHAR(255) NOT NULL")
	assert.Contains(t, sql, "`email` VARCHAR(255) NOT NULL UNIQUE")
	assert.Contains(t, sql, "`created_at` DATETIME NOT NULL")
	assert.Contains(t, sql, "PRIMARY KEY (`id`)")
	assert.Contains(t, sql, "ENGINE=InnoDB")
	assert.Contains(t, sql, "DEFAULT CHARSET=utf8mb4")
}

func TestMySQLDB_Transaction(t *testing.T) {
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
	defer db.Exec("DROP TABLE test_transaction")

	// Test successful transaction
	err = db.Transaction(ctx, func(tx types.Transaction) error {
		_, err := tx.Raw("INSERT INTO test_transaction (value) VALUES (?)", "test1").Exec(ctx)
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
		_, err := tx.Raw("INSERT INTO test_transaction (value) VALUES (?)", "test2").Exec(ctx)
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

func TestMySQLDB_FieldTypeMapping(t *testing.T) {
	db, err := NewMySQLDB(types.Config{})
	require.NoError(t, err)

	testCases := []struct {
		fieldType schema.FieldType
		expected  string
	}{
		{schema.FieldTypeString, "VARCHAR(255)"},
		{schema.FieldTypeInt, "INT"},
		{schema.FieldTypeInt64, "BIGINT"},
		{schema.FieldTypeFloat, "DOUBLE"},
		{schema.FieldTypeBool, "BOOLEAN"},
		{schema.FieldTypeDateTime, "DATETIME"},
		{schema.FieldTypeJSON, "JSON"},
		{schema.FieldTypeDecimal, "DECIMAL(10,2)"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := db.mapFieldTypeToSQL(tc.fieldType)
			assert.Equal(t, tc.expected, result)
		})
	}
}
