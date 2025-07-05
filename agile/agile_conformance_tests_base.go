package agile

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// AgileDriverCharacteristics defines driver-specific behaviors for agile tests
type AgileDriverCharacteristics struct {
	// SupportsReturning indicates if the driver supports RETURNING clause
	SupportsReturning bool

	// MaxConnectionPoolSize is the maximum number of connections in the pool
	MaxConnectionPoolSize int

	// SupportsNestedTransactions indicates if the driver supports savepoints
	SupportsNestedTransactions bool

	// ReturnsStringForNumbers indicates if the driver returns strings for numeric values (MySQL)
	ReturnsStringForNumbers bool
}

// AgileConformanceTests provides a comprehensive test suite for the agile API
type AgileConformanceTests struct {
	DriverName      string
	DatabaseURI     string
	SkipTests       map[string]bool
	Characteristics AgileDriverCharacteristics
	NewDatabase     func(uri string) (types.Database, error) // Function to create database instance
	CleanupTables   func(t *testing.T, db types.Database)    // Driver-specific table cleanup
}

// RunAll runs all agile conformance tests
func (act *AgileConformanceTests) RunAll(t *testing.T) {
	// Create database instance
	db, err := act.NewDatabase(act.DatabaseURI)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create agile client
	client := NewClient(db)

	// Connection Management
	t.Run("ConnectionManagement", func(t *testing.T) {
		if act.shouldSkip("ConnectionManagement") {
			t.Skip("Test skipped by driver")
		}
		act.runConnectionTests(t, client, db)
	})

	// Schema Management
	t.Run("SchemaManagement", func(t *testing.T) {
		if act.shouldSkip("SchemaManagement") {
			t.Skip("Test skipped by driver")
		}
		act.runSchemaTests(t, client, db)
	})

	// Basic CRUD Operations
	t.Run("BasicCRUD", func(t *testing.T) {
		if act.shouldSkip("BasicCRUD") {
			t.Skip("Test skipped by driver")
		}
		act.runCRUDTests(t, client, db)
	})

	// Query Building
	t.Run("QueryBuilding", func(t *testing.T) {
		if act.shouldSkip("QueryBuilding") {
			t.Skip("Test skipped by driver")
		}
		act.runQueryTests(t, client, db)
	})

	// Aggregations
	t.Run("Aggregations", func(t *testing.T) {
		if act.shouldSkip("Aggregations") {
			t.Skip("Test skipped by driver")
		}
		act.runAggregationTests(t, client, db)
	})

	// Relations and Includes
	t.Run("Relations", func(t *testing.T) {
		if act.shouldSkip("Relations") {
			t.Skip("Test skipped by driver")
		}
		act.runRelationTests(t, client, db)
	})

	// Transactions
	t.Run("Transactions", func(t *testing.T) {
		if act.shouldSkip("Transactions") {
			t.Skip("Test skipped by driver")
		}
		act.runTransactionTests(t, client, db)
	})
}

// shouldSkip checks if a test should be skipped
func (act *AgileConformanceTests) shouldSkip(testName string) bool {
	if act.SkipTests == nil {
		return false
	}
	return act.SkipTests[testName]
}

// runWithCleanup runs a test with table cleanup before execution
func (act *AgileConformanceTests) runWithCleanup(t *testing.T, db types.Database, testFunc func()) {
	// Clean up tables before each test
	if act.CleanupTables != nil {
		act.CleanupTables(t, db)
	}

	// Run the test
	testFunc()
}

// assertNoError checks that err is nil
func assertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// assertEqual checks that two values are equal
func assertEqual(t *testing.T, expected, actual any, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// assertNotNil checks that a value is not nil
func assertNotNil(t *testing.T, value any, msg string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s: value is nil", msg)
	}
}
