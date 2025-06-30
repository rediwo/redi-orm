package database

import (
	"fmt"
	"github.com/rediwo/redi-orm/types"
)

// Re-export types for backward compatibility
type DatabaseType = types.DatabaseType
type Config = types.Config
type Database = types.Database
type Transaction = types.Transaction
type QueryBuilder = types.QueryBuilder

// Re-export constants
const (
	SQLite     = types.SQLite
	MySQL      = types.MySQL
	PostgreSQL = types.PostgreSQL
)

// New creates a new database instance from a Config
func New(config Config) (Database, error) {
	factory, err := getDriver(config.Type)
	if err != nil {
		return nil, err
	}
	return factory(config)
}

// NewFromURI creates a new database instance from a URI string
// Supported formats:
// - sqlite:///path/to/database.db
// - sqlite://:memory:
// - mysql://user:pass@host:port/database
// - postgresql://user:pass@host:port/database
func NewFromURI(uri string) (Database, error) {
	config, err := ParseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	return New(config)
}
