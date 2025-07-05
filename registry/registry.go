package registry

import (
	"fmt"
	"github.com/rediwo/redi-orm/types"
	"net/url"
	"sync"
)

// DriverFactory is a function that creates a new database instance
type DriverFactory func(config types.Config) (types.Database, error)

// driverRegistry holds all registered database drivers, URI parsers, and capabilities
var (
	drivers      = make(map[string]DriverFactory)
	uriParsers   = make(map[string]types.URIParser)
	capabilities = make(map[types.DriverType]types.DriverCapabilities)
	schemes      = make(map[string]types.DriverType) // scheme -> driver type mapping
	mu           sync.RWMutex
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

	// Register schemes supported by this parser
	driverType := types.DriverType(parser.GetDriverType())
	for _, scheme := range parser.GetSupportedSchemes() {
		schemes[scheme] = driverType
	}
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

// RegisterCapabilities registers driver capabilities
func RegisterCapabilities(driverType types.DriverType, caps types.DriverCapabilities) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := capabilities[driverType]; exists {
		panic(fmt.Sprintf("capabilities for driver %s already registered", driverType))
	}

	capabilities[driverType] = caps
}

// GetCapabilities retrieves registered driver capabilities
func GetCapabilities(driverType types.DriverType) (types.DriverCapabilities, error) {
	mu.RLock()
	defer mu.RUnlock()

	caps, exists := capabilities[driverType]
	if !exists {
		return nil, fmt.Errorf("capabilities for driver %s not registered", driverType)
	}

	return caps, nil
}

// ResolveScheme resolves a URI scheme to a driver type
func ResolveScheme(scheme string) (types.DriverType, error) {
	mu.RLock()
	defer mu.RUnlock()

	driverType, exists := schemes[scheme]
	if !exists {
		return "", fmt.Errorf("unsupported scheme: %s", scheme)
	}

	return driverType, nil
}

// ParseURI attempts to parse a URI using all registered parsers
func ParseURI(uri string) (types.Config, error) {
	// First, try to determine the driver type from the scheme
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return types.Config{}, fmt.Errorf("invalid URI format: %w", err)
	}

	mu.RLock()
	defer mu.RUnlock()

	// Try to find the right parser based on scheme
	if driverType, ok := schemes[parsedURI.Scheme]; ok {
		if parser, exists := uriParsers[string(driverType)]; exists {
			return parser.ParseURI(uri)
		}
	}

	// Fallback: try all parsers (for backward compatibility)
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
