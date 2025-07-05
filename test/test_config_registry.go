package test

import (
	"fmt"
	"sync"
)

// testDatabaseURIRegistry holds registered test database URIs
var (
	testDatabaseURIRegistry = make(map[string]string)
	registryMutex           sync.RWMutex
)

// RegisterTestDatabaseUri registers a test database URI for a driver
func RegisterTestDatabaseUri(driverType string, uri string) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	testDatabaseURIRegistry[driverType] = uri
}

// GetTestDatabaseUri returns test database URI for a specific driver
func GetTestDatabaseUri(driverType string) string {
	registryMutex.RLock()
	uri, exists := testDatabaseURIRegistry[driverType]
	registryMutex.RUnlock()

	if !exists {
		panic(fmt.Sprintf("no test database URI registered for driver: %s", driverType))
	}

	return uri
}
