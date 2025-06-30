package database

import (
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/types"
)

// getDriver retrieves a registered driver factory
func getDriver(dbType types.DatabaseType) (registry.DriverFactory, error) {
	return registry.Get(dbType)
}