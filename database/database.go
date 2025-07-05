package database

import (
	"fmt"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/types"
)

// Re-export types for backward compatibility
type Database = types.Database
type Transaction = types.Transaction
type ModelQuery = types.ModelQuery
type SelectQuery = types.SelectQuery
type InsertQuery = types.InsertQuery
type UpdateQuery = types.UpdateQuery
type DeleteQuery = types.DeleteQuery
type RawQuery = types.RawQuery

// NewFromURI creates a new database instance from a URI string
// The URI is parsed by the appropriate driver's URI parser
// Supported formats depend on the registered drivers:
// - sqlite:///path/to/database.db
// - sqlite://:memory:
// - mysql://user:pass@host:port/database
// - postgresql://user:pass@host:port/database
func NewFromURI(uri string) (Database, error) {
	nativeURI, driverType, err := registry.ParseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	
	factory, err := registry.Get(string(driverType))
	if err != nil {
		return nil, err
	}
	
	return factory(nativeURI)
}

// New creates a new database instance from a URI string
// This is kept for backward compatibility and delegates to NewFromURI
func New(uri string) (Database, error) {
	return NewFromURI(uri)
}
