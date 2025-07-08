package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/rediwo/redi-orm/drivers/mysql"      // Import MySQL driver for tests
	_ "github.com/rediwo/redi-orm/drivers/postgresql" // Import PostgreSQL driver for tests
	_ "github.com/rediwo/redi-orm/drivers/sqlite"     // Import SQLite driver for tests
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock database implementation for testing
type mockDatabase struct {
	connected bool
	uri       string
}

type mockCapabilities struct{}

func (c *mockCapabilities) SupportsReturning() bool            { return false }
func (c *mockCapabilities) SupportsDefaultValues() bool        { return true }
func (c *mockCapabilities) RequiresLimitForOffset() bool       { return true }
func (c *mockCapabilities) SupportsDistinctOn() bool           { return false }
func (c *mockCapabilities) QuoteIdentifier(name string) string { return "`" + name + "`" }
func (c *mockCapabilities) GetPlaceholder(index int) string    { return "?" }
func (c *mockCapabilities) GetBooleanLiteral(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
func (c *mockCapabilities) NeedsTypeConversion() bool { return false }
func (c *mockCapabilities) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	return ""
}
func (c *mockCapabilities) IsSystemIndex(indexName string) bool { return false }
func (c *mockCapabilities) IsSystemTable(tableName string) bool { return false }
func (c *mockCapabilities) GetDriverType() types.DriverType     { return "mock" }
func (c *mockCapabilities) GetSupportedSchemes() []string       { return []string{"mock"} }

// NoSQL features
func (c *mockCapabilities) IsNoSQL() bool                     { return false }
func (c *mockCapabilities) SupportsTransactions() bool        { return true }
func (c *mockCapabilities) SupportsNestedDocuments() bool     { return false }
func (c *mockCapabilities) SupportsArrayFields() bool         { return false }
func (c *mockCapabilities) SupportsAggregationPipeline() bool { return false }

func (m *mockDatabase) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockDatabase) Close() error {
	m.connected = false
	return nil
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	if !m.connected {
		return fmt.Errorf("database not connected")
	}
	return nil
}

func (m *mockDatabase) RegisterSchema(modelName string, s *schema.Schema) error {
	return nil
}

func (m *mockDatabase) GetSchema(modelName string) (*schema.Schema, error) {
	return nil, fmt.Errorf("schema not found")
}

func (m *mockDatabase) CreateModel(ctx context.Context, modelName string) error {
	return nil
}

func (m *mockDatabase) DropModel(ctx context.Context, modelName string) error {
	return nil
}

func (m *mockDatabase) Model(modelName string) types.ModelQuery {
	return nil
}

func (m *mockDatabase) Raw(sql string, args ...any) types.RawQuery {
	return nil
}

func (m *mockDatabase) Begin(ctx context.Context) (types.Transaction, error) {
	return nil, nil
}

func (m *mockDatabase) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	return nil
}

func (m *mockDatabase) GetModels() []string {
	return []string{}
}

func (m *mockDatabase) GetModelSchema(modelName string) (*schema.Schema, error) {
	return nil, fmt.Errorf("schema not found")
}

func (m *mockDatabase) ResolveTableName(modelName string) (string, error) {
	return "", nil
}

func (m *mockDatabase) ResolveFieldName(modelName, fieldName string) (string, error) {
	return "", nil
}

func (m *mockDatabase) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return fieldNames, nil
}

func (m *mockDatabase) GetMigrator() types.DatabaseMigrator {
	return nil
}

func (m *mockDatabase) GetFieldMapper() types.FieldMapper {
	return nil
}

func (m *mockDatabase) GetDriverType() string {
	return "mock"
}

func (m *mockDatabase) GetCapabilities() types.DriverCapabilities {
	return &mockCapabilities{}
}

func (m *mockDatabase) GetBooleanLiteral(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (m *mockDatabase) Exec(query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabase) Query(query string, args ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabase) QueryRow(query string, args ...any) *sql.Row {
	return nil
}

func (m *mockDatabase) LoadSchema(ctx context.Context, schemaContent string) error {
	return nil
}

func (m *mockDatabase) LoadSchemaFrom(ctx context.Context, filename string) error {
	return nil
}

func (m *mockDatabase) SyncSchemas(ctx context.Context) error {
	return nil
}

func (m *mockDatabase) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

func (m *mockDatabase) SupportsDefaultValues() bool {
	return true
}

func (m *mockDatabase) SupportsReturning() bool {
	return false
}

func (m *mockDatabase) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	return ""
}

func (m *mockDatabase) RequiresLimitForOffset() bool {
	return true
}

func (m *mockDatabase) SetLogger(logger any) {
	// Mock implementation - do nothing
}

func (m *mockDatabase) GetLogger() any {
	return nil
}

// Mock factory function
func mockFactory(uri string) (types.Database, error) {
	if uri != "mock://test" {
		return nil, fmt.Errorf("invalid URI for mock database")
	}
	return &mockDatabase{uri: uri}, nil
}

// Mock URI parser implementation
type mockURIParserImpl struct{}

func (p *mockURIParserImpl) ParseURI(uri string) (string, error) {
	if uri == "mock://test" {
		return "mock://test", nil
	}
	return "", fmt.Errorf("invalid mock URI")
}

func (p *mockURIParserImpl) GetSupportedSchemes() []string {
	return []string{"mock"}
}

func (p *mockURIParserImpl) GetDriverType() string {
	return "mock"
}

func TestNew(t *testing.T) {
	// Try to register mock driver, ignore if already registered
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Ignore panic if driver already registered
			}
		}()
		registry.Register("mock", mockFactory)
		registry.RegisterURIParser("mock", &mockURIParserImpl{})
	}()

	t.Run("Create with valid config", func(t *testing.T) {
		uri := "mock://test"
		db, err := New(uri)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Verify it's our mock database
		mockDB, ok := db.(*mockDatabase)
		assert.True(t, ok)
		assert.Equal(t, "mock://test", mockDB.uri)
	})

	t.Run("Create with unregistered driver", func(t *testing.T) {
		uri := "nonexistent://invalid"
		db, err := New(uri)
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to parse URI")
	})

	t.Run("Create with SQLite", func(t *testing.T) {
		uri := "sqlite://:memory:"
		db, err := New(uri)
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Test that we can connect
		ctx := context.Background()
		err = db.Connect(ctx)
		assert.NoError(t, err)
		defer db.Close()
	})

	t.Run("Create with MySQL", func(t *testing.T) {
		uri := "mysql://testuser:testpass@localhost:3306/testdb"
		db, err := New(uri)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		// Don't try to connect as MySQL may not be available
	})

	t.Run("Create with PostgreSQL", func(t *testing.T) {
		uri := "postgresql://testuser:testpass@localhost:5432/testdb"
		db, err := New(uri)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		// Don't try to connect as PostgreSQL may not be available
	})
}

func TestNewFromURI(t *testing.T) {
	// Try to register mock driver, ignore if already registered
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Ignore panic if driver already registered
			}
		}()
		registry.Register("mock", mockFactory)
		registry.RegisterURIParser("mock", &mockURIParserImpl{})
	}()

	t.Run("Create from valid URI", func(t *testing.T) {
		db, err := NewFromURI("mock://test")
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Verify it's our mock database
		mockDB, ok := db.(*mockDatabase)
		assert.True(t, ok)
		assert.Equal(t, "mock://test", mockDB.uri)
	})

	t.Run("Create from invalid URI", func(t *testing.T) {
		db, err := NewFromURI("invalid://uri")
		assert.Error(t, err)
		assert.Nil(t, db)
		assert.Contains(t, err.Error(), "failed to parse URI")
	})

	t.Run("Create SQLite from URI", func(t *testing.T) {
		db, err := NewFromURI("sqlite://:memory:")
		assert.NoError(t, err)
		assert.NotNil(t, db)

		// Test that we can connect
		ctx := context.Background()
		err = db.Connect(ctx)
		assert.NoError(t, err)
		defer db.Close()
	})

	t.Run("Create SQLite with file path", func(t *testing.T) {
		db, err := NewFromURI("sqlite:///tmp/test.db")
		assert.NoError(t, err)
		assert.NotNil(t, db)
		// Don't connect to avoid creating file
	})

	t.Run("Create MySQL from URI", func(t *testing.T) {
		db, err := NewFromURI("mysql://testuser:testpass@localhost:3306/testdb")
		assert.NoError(t, err)
		assert.NotNil(t, db)
		// Don't try to connect as MySQL may not be available
	})

	t.Run("Create PostgreSQL from URI", func(t *testing.T) {
		db, err := NewFromURI("postgresql://testuser:testpass@localhost:5432/testdb")
		assert.NoError(t, err)
		assert.NotNil(t, db)
		// Don't try to connect as PostgreSQL may not be available
	})

	t.Run("Create PostgreSQL with postgres:// scheme", func(t *testing.T) {
		db, err := NewFromURI("postgres://testuser:testpass@localhost:5432/testdb")
		assert.NoError(t, err)
		assert.NotNil(t, db)
		// Don't try to connect as PostgreSQL may not be available
	})

	t.Run("URI with query parameters", func(t *testing.T) {
		db, err := NewFromURI("postgresql://testuser:testpass@localhost:5432/testdb?sslmode=disable")
		assert.NoError(t, err)
		assert.NotNil(t, db)
	})
}

func TestDefaultDriversLoaded(t *testing.T) {
	// This test verifies that default drivers are automatically loaded
	// We'll try to create instances to verify they're registered

	t.Run("SQLite driver is registered", func(t *testing.T) {
		factory, err := registry.Get("sqlite")
		assert.NoError(t, err)
		assert.NotNil(t, factory, "SQLite driver should be registered by default")
	})

	t.Run("MySQL driver is registered", func(t *testing.T) {
		factory, err := registry.Get("mysql")
		assert.NoError(t, err)
		assert.NotNil(t, factory, "MySQL driver should be registered by default")
	})

	t.Run("PostgreSQL driver is registered", func(t *testing.T) {
		factory, err := registry.Get("postgresql")
		assert.NoError(t, err)
		assert.NotNil(t, factory, "PostgreSQL driver should be registered by default")
	})
}

func TestURIParsingExamples(t *testing.T) {
	testCases := []struct {
		name               string
		uri                string
		shouldError        bool
		expectedDriverType string
	}{
		{
			name:               "SQLite memory",
			uri:                "sqlite://:memory:",
			shouldError:        false,
			expectedDriverType: "sqlite",
		},
		{
			name:               "SQLite file",
			uri:                "sqlite:///path/to/db.sqlite",
			shouldError:        false,
			expectedDriverType: "sqlite",
		},
		{
			name:               "MySQL full URI",
			uri:                "mysql://user:pass@localhost:3306/database",
			shouldError:        false,
			expectedDriverType: "mysql",
		},
		{
			name:               "PostgreSQL with SSL",
			uri:                "postgresql://user:pass@host:5432/db?sslmode=require",
			shouldError:        false,
			expectedDriverType: "postgresql",
		},
		{
			name:        "Invalid scheme",
			uri:         "invalid://something",
			shouldError: true,
		},
		{
			name:        "Empty URI",
			uri:         "",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nativeURI, driverType, err := registry.ParseURI(tc.uri)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, nativeURI)
				assert.Equal(t, tc.expectedDriverType, string(driverType))
			}
		})
	}
}

func TestIntegrationWithRealDatabase(t *testing.T) {
	t.Run("SQLite integration", func(t *testing.T) {
		// This test actually creates a database connection
		db, err := NewFromURI("sqlite://:memory:")
		require.NoError(t, err)
		require.NotNil(t, db)

		ctx := context.Background()

		// Connect
		err = db.Connect(ctx)
		require.NoError(t, err)
		defer db.Close()

		// Ping
		err = db.Ping(ctx)
		assert.NoError(t, err)

		// Execute raw query
		raw := db.Raw("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
		result, err := raw.Exec(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Insert data
		raw = db.Raw("INSERT INTO test (name) VALUES (?)", "test")
		result, err = raw.Exec(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result.LastInsertID)
		assert.Equal(t, int64(1), result.RowsAffected)

		// Query data
		type TestRow struct {
			ID   int
			Name string
		}
		var rows []TestRow
		raw = db.Raw("SELECT id, name FROM test")
		err = raw.Find(ctx, &rows)
		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, 1, rows[0].ID)
		assert.Equal(t, "test", rows[0].Name)
	})
}
