package sqlite

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/orm"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteOrmConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orm conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("sqlite")

	suite := &orm.OrmConformanceTests{
		DriverName:  "SQLite",
		DatabaseURI: uri,
		NewDatabase: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		SkipTests: map[string]bool{
			// SQLite-specific skips
			"TestTransactionConcurrentAccess": true, // SQLite uses database-level locking
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// Use the same cleanup logic as conformance tests
			if sqliteDB, ok := db.(*SQLiteDB); ok {
				cleanupTables(t, sqliteDB)
			}
		},
		Characteristics: orm.OrmDriverCharacteristics{
			SupportsReturning:          false,
			MaxConnectionPoolSize:      1,
			SupportsNestedTransactions: false,
			ReturnsStringForNumbers:    false,
		},
	}

	suite.RunAll(t)
}
