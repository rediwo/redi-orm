package sqlite

import (
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	config := types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	}

	suite := &test.DriverConformanceTests{
		DriverName: "SQLite",
		NewDriver: func(cfg types.Config) (types.Database, error) {
			return NewSQLiteDB(cfg)
		},
		Config: config,
		SkipTests: map[string]bool{
			// SQLite in-memory specific skips
			"TestConnectWithInvalidConfig": true,  // In-memory doesn't have connection failures
			"TestCreateExistingModel":      true,  // Uses CREATE TABLE IF NOT EXISTS
			"TestTransactionIsolation":     true,  // In-memory doesn't support proper isolation
		},
	}

	suite.RunAll(t)
}