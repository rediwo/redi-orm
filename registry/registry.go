package registry

import (
	"fmt"
	"sync"
	"github.com/rediwo/redi-orm/types"
)

// DriverFactory is a function that creates a new database instance
type DriverFactory func(config types.Config) (types.Database, error)

// driverRegistry holds all registered database drivers
var (
	drivers = make(map[types.DatabaseType]DriverFactory)
	mu      sync.RWMutex
)

// Register registers a database driver factory
func Register(dbType types.DatabaseType, factory DriverFactory) {
	mu.Lock()
	defer mu.Unlock()
	
	if _, exists := drivers[dbType]; exists {
		panic(fmt.Sprintf("driver %s already registered", dbType))
	}
	
	drivers[dbType] = factory
}

// Get retrieves a registered driver factory
func Get(dbType types.DatabaseType) (DriverFactory, error) {
	mu.RLock()
	defer mu.RUnlock()
	
	factory, exists := drivers[dbType]
	if !exists {
		return nil, fmt.Errorf("driver %s not registered", dbType)
	}
	
	return factory, nil
}