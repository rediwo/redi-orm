package orm

import (
	"testing"
)

// JSDriverCharacteristics defines JavaScript-specific driver behaviors
type JSDriverCharacteristics struct {
	// SupportsArrayTypes indicates if the driver supports array field types
	SupportsArrayTypes bool
	
	// SupportsJSONTypes indicates if the driver supports JSON/JSONB field types
	SupportsJSONTypes bool
	
	// SupportsEnumTypes indicates if the driver supports enum field types
	SupportsEnumTypes bool
	
	// MaxConnectionPoolSize is the maximum number of connections in the pool
	MaxConnectionPoolSize int
	
	// SupportsNestedTransactions indicates if the driver supports savepoints
	SupportsNestedTransactions bool
}

// JSConformanceTests provides a comprehensive test suite for JavaScript ORM API
type JSConformanceTests struct {
	DriverName      string
	DatabaseURI     string
	SkipTests       map[string]bool
	Characteristics JSDriverCharacteristics
	CleanupTables   func(t *testing.T, runner *JSTestRunner) // Driver-specific table cleanup
}

// RunAll runs all JavaScript conformance tests
func (jct *JSConformanceTests) RunAll(t *testing.T) {
	runner, err := NewJSTestRunner(t)
	if err != nil {
		t.Fatalf("Failed to create test runner: %v", err)
	}
	defer runner.Cleanup()

	// Set database URI for all tests
	runner.SetDatabaseURI(jct.DatabaseURI)

	// Copy test assets
	if err := runner.CopyTestAssets(); err != nil {
		t.Fatalf("Failed to copy test assets: %v", err)
	}

	// Connection Management
	t.Run("ConnectionManagement", func(t *testing.T) {
		if jct.shouldSkip("ConnectionManagement") {
			t.Skip("Test skipped by driver")
		}
		jct.runConnectionTests(t, runner)
	})

	// Schema Management
	t.Run("SchemaManagement", func(t *testing.T) {
		if jct.shouldSkip("SchemaManagement") {
			t.Skip("Test skipped by driver")
		}
		jct.runSchemaTests(t, runner)
	})

	// Basic CRUD Operations
	t.Run("BasicCRUD", func(t *testing.T) {
		if jct.shouldSkip("BasicCRUD") {
			t.Skip("Test skipped by driver")
		}
		jct.runCRUDTests(t, runner)
	})

	// Query Building
	t.Run("QueryBuilding", func(t *testing.T) {
		if jct.shouldSkip("QueryBuilding") {
			t.Skip("Test skipped by driver")
		}
		jct.runQueryTests(t, runner)
	})

	// Advanced Queries
	t.Run("AdvancedQueries", func(t *testing.T) {
		if jct.shouldSkip("AdvancedQueries") {
			t.Skip("Test skipped by driver")
		}
		jct.runAdvancedQueryTests(t, runner)
	})

	// Aggregations and Analytics
	t.Run("Aggregations", func(t *testing.T) {
		if jct.shouldSkip("Aggregations") {
			t.Skip("Test skipped by driver")
		}
		jct.runAggregationTests(t, runner)
	})

	// Relations and Includes
	t.Run("Relations", func(t *testing.T) {
		if jct.shouldSkip("Relations") {
			t.Skip("Test skipped by driver")
		}
		jct.runRelationTests(t, runner)
	})

	// Advanced Include Options
	t.Run("IncludeOptions", func(t *testing.T) {
		if jct.shouldSkip("IncludeOptions") {
			t.Skip("Test skipped by driver")
		}
		jct.runIncludeOptionsTests(t, runner)
	})

	// Transactions
	t.Run("Transactions", func(t *testing.T) {
		if jct.shouldSkip("Transactions") {
			t.Skip("Test skipped by driver")
		}
		jct.runTransactionTests(t, runner)
	})

	// Raw Queries
	t.Run("RawQueries", func(t *testing.T) {
		if jct.shouldSkip("RawQueries") {
			t.Skip("Test skipped by driver")
		}
		jct.runRawQueryTests(t, runner)
	})

	// Data Types
	t.Run("DataTypes", func(t *testing.T) {
		if jct.shouldSkip("DataTypes") {
			t.Skip("Test skipped by driver")
		}
		jct.runDataTypeTests(t, runner)
	})

	// Error Handling
	t.Run("ErrorHandling", func(t *testing.T) {
		if jct.shouldSkip("ErrorHandling") {
			t.Skip("Test skipped by driver")
		}
		jct.runErrorHandlingTests(t, runner)
	})

	// Migration Tests
	t.Run("Migrations", func(t *testing.T) {
		if jct.shouldSkip("Migrations") {
			t.Skip("Test skipped by driver")
		}
		jct.runMigrationTests(t, runner)
	})

	// Performance Tests
	t.Run("Performance", func(t *testing.T) {
		if jct.shouldSkip("Performance") {
			t.Skip("Test skipped by driver")
		}
		jct.runPerformanceTests(t, runner)
	})
}

// shouldSkip checks if a test should be skipped
func (jct *JSConformanceTests) shouldSkip(testName string) bool {
	if jct.SkipTests == nil {
		return false
	}
	return jct.SkipTests[testName]
}

// runWithCleanup runs a test with table cleanup before execution
func (jct *JSConformanceTests) runWithCleanup(t *testing.T, runner *JSTestRunner, testName string, testCode string) {
	// Clean up tables before each test
	if jct.CleanupTables != nil {
		jct.CleanupTables(t, runner)
	}
	
	// Run the test
	runner.RunInlineTest(t, testName, testCode)
}