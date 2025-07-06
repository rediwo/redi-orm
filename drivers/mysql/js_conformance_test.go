package mysql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/modules/orm"
	"github.com/rediwo/redi-orm/test"
)

func TestMySQLJSConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping JS conformance tests in short mode")
	}

	// Get test database URI
	uri := test.GetTestDatabaseUri("mysql")

	suite := &orm.JSConformanceTests{
		DriverName:  "MySQL",
		DatabaseURI: uri,
		SkipTests:   map[string]bool{
			// MySQL-specific skips if needed
		},
		Characteristics: orm.JSDriverCharacteristics{
			SupportsArrayTypes:         false,
			SupportsJSONTypes:          true, // MySQL 5.7+ supports JSON
			SupportsEnumTypes:          true,
			MaxConnectionPoolSize:      10,
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
			
			if mysqlDB, ok := db.(*MySQLDB); ok {
				cleanupTables(t, mysqlDB)
			}
		},
	}

	suite.RunAll(t)
}

