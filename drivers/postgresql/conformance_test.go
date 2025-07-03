package postgresql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestPostgreSQLConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Skip if PostgreSQL is not available
	config := test.GetTestConfig("postgresql")
	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	if err := db.Connect(context.Background()); err != nil {
		t.Skipf("Cannot connect to PostgreSQL: %v", err)
	}
	db.Close()

	suite := &test.DriverConformanceTests{
		DriverName: "PostgreSQL",
		NewDriver: func(cfg types.Config) (types.Database, error) {
			return NewPostgreSQLDB(cfg)
		},
		Config: config,
		SkipTests: map[string]bool{
			// PostgreSQL-specific skips
			"TestCreateExistingModel": true, // Uses CREATE TABLE IF NOT EXISTS
		},
	}

	suite.RunAll(t)
}

