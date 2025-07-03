package test

import (
	"fmt"
	"sync"

	"github.com/rediwo/redi-orm/types"
)

// TestConfigFactory is a function that returns a test configuration for a driver
type TestConfigFactory func() types.Config

// testConfigRegistry holds registered test config factories
var (
	testConfigRegistry = make(map[string]TestConfigFactory)
	registryMutex      sync.RWMutex
)

// RegisterTestConfig registers a test configuration factory for a driver
func RegisterTestConfig(driverType string, factory TestConfigFactory) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	testConfigRegistry[driverType] = factory
}

// GetTestConfig returns test configuration for a specific driver
func GetTestConfig(driverType string) types.Config {
	registryMutex.RLock()
	factory, exists := testConfigRegistry[driverType]
	registryMutex.RUnlock()

	if !exists {
		panic(fmt.Sprintf("no test config factory registered for driver: %s", driverType))
	}

	return factory()
}