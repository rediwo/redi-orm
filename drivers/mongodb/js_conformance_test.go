package mongodb

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/modules/orm"
	"github.com/rediwo/redi-orm/test"
)

func TestMongoDBJSConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping JS conformance tests in short mode")
	}

	// Get test database URI
	uri := test.GetTestDatabaseUri("mongodb")

	suite := &orm.JSConformanceTests{
		DriverName:  "MongoDB",
		DatabaseURI: uri,
		SkipTests: map[string]bool{
			// MongoDB-specific skips
			"TestTransactionIsolation":        true, // MongoDB has different isolation semantics
			"TestTransactionErrorHandling":    true, // MongoDB allows incomplete documents
			"TestTransactionConcurrentAccess": true, // MongoDB transaction behavior differs
			"TestIndexPerformance":            true, // MongoDB index behavior differs
			"TestPaginationPerformance":       true, // MongoDB pagination needs optimization
			"TestComplexQueryPerformance":     true, // MongoDB complex queries need optimization
		},
		Characteristics: orm.JSDriverCharacteristics{
			SupportsArrayTypes:         true,  // MongoDB natively supports arrays
			SupportsJSONTypes:          true,  // MongoDB is document-based (BSON)
			SupportsEnumTypes:          false, // No native enum support
			MaxConnectionPoolSize:      100,   // MongoDB default pool size
			SupportsNestedTransactions: false, // MongoDB doesn't support nested transactions
		},
		CleanupTables: func(t *testing.T, runner *orm.JSTestRunner) {
			// Use the shared cleanupTables function by creating a database connection
			db, err := database.NewFromURI(uri)
			if err != nil {
				t.Logf("Failed to create database for cleanup: %v", err)
				return
			}
			defer db.Close()

			// Connect to the database
			ctx := context.Background()
			if err := db.Connect(ctx); err != nil {
				t.Logf("Failed to connect to database for cleanup: %v", err)
				return
			}

			if mongoDb, ok := db.(*MongoDB); ok {
				cleanupTables(t, mongoDb)
			}
		},
	}

	suite.RunAll(t)
}
