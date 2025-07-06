package mongodb

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/orm"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestMongoDBOrmConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orm conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("mongodb")

	suite := &orm.OrmConformanceTests{
		DriverName:  "MongoDB",
		DatabaseURI: uri,
		NewDatabase: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		SkipTests: map[string]bool{
			// MongoDB-specific skips
			"TestTransactionIsolation":       true, // MongoDB has different isolation semantics
			"TestTransactionErrorHandling":   true, // MongoDB allows incomplete documents
			"TestTransactionConcurrentAccess": true, // MongoDB transaction behavior differs
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// MongoDB-specific cleanup
			if mongoDb, ok := db.(*MongoDB); ok {
				cleanupTables(t, mongoDb)
			}
		},
		Characteristics: orm.OrmDriverCharacteristics{
			SupportsReturning:          false, // MongoDB doesn't support RETURNING
			MaxConnectionPoolSize:      100,   // MongoDB default pool size
			SupportsNestedTransactions: false, // MongoDB doesn't support nested transactions
			ReturnsStringForNumbers:    false, // MongoDB preserves numeric types
		},
	}

	suite.RunAll(t)
}