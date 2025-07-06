package mysql

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/orm"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestMySQLOrmConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orm conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("mysql")

	suite := &orm.OrmConformanceTests{
		DriverName:  "MySQL",
		DatabaseURI: uri,
		NewDatabase: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		SkipTests: map[string]bool{
			// MySQL-specific skips (if any)
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// MySQL-specific cleanup
			if mysqlDB, ok := db.(*MySQLDB); ok {
				cleanupTables(t, mysqlDB)
			}
		},
		Characteristics: orm.OrmDriverCharacteristics{
			SupportsReturning:          false,
			MaxConnectionPoolSize:      10,
			SupportsNestedTransactions: true,
			ReturnsStringForNumbers:    true, // MySQL returns strings for numeric aggregations
		},
	}

	suite.RunAll(t)
}

