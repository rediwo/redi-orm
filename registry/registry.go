package registry

import (
	"fmt"
	"github.com/rediwo/redi-orm/types"
	"sync"
)

// DriverFactory is a function that creates a new database instance
type DriverFactory func(config types.Config) (types.Database, error)

// driverRegistry holds all registered database drivers and URI parsers
var (
	drivers    = make(map[string]DriverFactory)
	uriParsers = make(map[string]types.URIParser)
	mu         sync.RWMutex
)

// Register registers a database driver factory
func Register(dbType string, factory DriverFactory) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := drivers[dbType]; exists {
		panic(fmt.Sprintf("driver %s already registered", dbType))
	}

	drivers[dbType] = factory
}

// Get retrieves a registered driver factory
func Get(dbType string) (DriverFactory, error) {
	mu.RLock()
	defer mu.RUnlock()

	factory, exists := drivers[dbType]
	if !exists {
		return nil, fmt.Errorf("driver %s not registered", dbType)
	}

	return factory, nil
}

// RegisterURIParser registers a URI parser for a specific driver
func RegisterURIParser(dbType string, parser types.URIParser) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := uriParsers[dbType]; exists {
		panic(fmt.Sprintf("URI parser for driver %s already registered", dbType))
	}

	uriParsers[dbType] = parser
}

// GetURIParser retrieves a registered URI parser
func GetURIParser(dbType string) (types.URIParser, error) {
	mu.RLock()
	defer mu.RUnlock()

	parser, exists := uriParsers[dbType]
	if !exists {
		return nil, fmt.Errorf("URI parser for driver %s not registered", dbType)
	}

	return parser, nil
}

// GetAllURIParsers returns all registered URI parsers
func GetAllURIParsers() map[string]types.URIParser {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]types.URIParser)
	for dbType, parser := range uriParsers {
		result[dbType] = parser
	}
	return result
}

// ParseURI attempts to parse a URI using all registered parsers
func ParseURI(uri string) (types.Config, error) {
	mu.RLock()
	defer mu.RUnlock()

	var lastErr error
	for _, parser := range uriParsers {
		if config, err := parser.ParseURI(uri); err == nil {
			return config, nil
		} else {
			lastErr = err
		}
	}

	if lastErr != nil {
		return types.Config{}, fmt.Errorf("no driver supports URI '%s': %w", uri, lastErr)
	}
	return types.Config{}, fmt.Errorf("no URI parsers registered")
}
