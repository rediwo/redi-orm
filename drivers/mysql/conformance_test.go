package mysql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestMySQLConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Skip if MySQL is not available
	config := test.GetTestConfig("mysql")
	db, err := NewMySQLDB(config)
	if err != nil {
		t.Skip("MySQL not available for testing")
	}
	if err := db.Connect(context.Background()); err != nil {
		t.Skipf("Cannot connect to MySQL: %v", err)
	}
	db.Close()

	suite := &test.DriverConformanceTests{
		DriverName: "MySQL",
		NewDriver: func(cfg types.Config) (types.Database, error) {
			return NewMySQLDB(cfg)
		},
		Config: config,
		SkipTests: map[string]bool{
			// MySQL-specific skips
			"TestCreateExistingModel": true, // Uses CREATE TABLE IF NOT EXISTS
		},
	}

	suite.RunAll(t)
}

