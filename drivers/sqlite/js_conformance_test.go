package sqlite

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/modules/orm"
	"github.com/rediwo/redi-orm/test"
)

func TestSQLiteJSConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping JS conformance tests in short mode")
	}

	// Get test database URI
	uri := test.GetTestDatabaseUri("sqlite")

	suite := &orm.JSConformanceTests{
		DriverName:  "SQLite",
		DatabaseURI: uri,
		SkipTests: map[string]bool{
			// SQLite doesn't support concurrent write transactions
			"TestTransactionIsolation":        true,
			"TestTransactionConcurrentAccess": true,
		},
		Characteristics: orm.JSDriverCharacteristics{
			SupportsArrayTypes:         false,
			SupportsJSONTypes:          false,
			SupportsEnumTypes:          false,
			MaxConnectionPoolSize:      1, // SQLite is single-threaded
			SupportsNestedTransactions: true,
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
			
			if sqliteDB, ok := db.(*SQLiteDB); ok {
				cleanupTables(t, sqliteDB)
			}
		},
	}

	suite.RunAll(t)
}

