package mysql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestMySQLConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Skip if MySQL is not available
	uri := test.GetTestDatabaseUri("mysql")
	db, err := database.NewFromURI(uri)
	if err != nil {
		t.Skip("MySQL not available for testing")
	}
	if err := db.Connect(context.Background()); err != nil {
		t.Skipf("Cannot connect to MySQL: %v", err)
	}
	db.Close()

	suite := &test.DriverConformanceTests{
		DriverName: "MySQL",
		NewDriver: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		URI: uri,
		SkipTests: map[string]bool{
			// MySQL-specific skips
			"TestAggregations": true, // MySQL returns aggregation results as strings
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			mysqlDB, ok := db.(*MySQLDB)
			if ok {
				cleanupTables(t, mysqlDB)
			}
		},
		Characteristics: test.DriverCharacteristics{
			ReturnsZeroRowsAffectedForUnchanged: true,
			SupportsLastInsertID:                true,
			SupportsReturningClause:             false,
			MigrationTableName:                  "_migrations",
			SystemIndexPatterns:                 []string{"PRIMARY", "fk_*", "mysql_*"},
			AutoIncrementIntegerType:            "INT AUTO_INCREMENT",
		},
	}

	suite.RunAll(t)
}

