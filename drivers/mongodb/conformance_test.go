package mongodb

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestMongoDBConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// MongoDB test URI with authentication
	uri := "mongodb://testuser:testpass@localhost:27017/testdb?authSource=admin"

	suite := &test.DriverConformanceTests{
		DriverName: "MongoDB",
		NewDriver: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		URI: uri,
		SkipTests: map[string]bool{
			"TestDropModel": true, // MongoDB auto-creates collections on insert
			// Savepoints are not supported in MongoDB
			"TestSavepoints":                 true,
			"TestNotNullConstraintViolation": true, // MongoDB doesn't enforce NOT NULL at DB level
			"TestInvalidFieldName":           true, // MongoDB allows any field names
			"TestInvalidModelName":           true, // MongoDB doesn't validate model names
			"TestTransactionIsolation":       true, // MongoDB has different isolation semantics
			"TestTransactionErrorHandling":   true, // MongoDB allows incomplete documents
			// Migration tests are not applicable to MongoDB (document database)
			"TestGetMigrator":            true,
			"TestGetTables":              true,
			"TestGetTableInfo":           true,
			"TestGenerateCreateTableSQL": true,
			"TestGenerateDropTableSQL":   true,
			"TestGenerateAddColumnSQL":   true,
			"TestGenerateDropColumnSQL":  true,
			"TestGenerateCreateIndexSQL": true,
			"TestGenerateDropIndexSQL":   true,
			"TestApplyMigration":         true,
			"TestMigrationWorkflow":      true,
			// SQL-specific tests not applicable to MongoDB
			"TestRawQueryErrorHandling": true, // MongoDB uses JSON queries, not SQL syntax validation
			"TestGenerateColumnSQL":     true, // MongoDB doesn't use SQL column definitions
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// MongoDB-specific cleanup
			if mongoDb, ok := db.(*MongoDB); ok {
				cleanupTables(t, mongoDb)
			}
		},
		Characteristics: test.DriverCharacteristics{
			ReturnsZeroRowsAffectedForUnchanged: false,
			SupportsLastInsertID:                true,  // Now supported via sequence generation
			SupportsReturningClause:             false, // No RETURNING in MongoDB
			MigrationTableName:                  "_migrations",
			SystemIndexPatterns:                 []string{"_id_"},
			AutoIncrementIntegerType:            "int64", // Sequences return int64
		},
	}

	// Check if MongoDB is available
	db, err := suite.NewDriver(uri)
	if err != nil {
		t.Skipf("Failed to create MongoDB driver: %v", err)
		return
	}

	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Skipf("MongoDB not available: %v", err)
		return
	}
	db.Close()

	// Run the conformance tests
	suite.RunAll(t)
}
