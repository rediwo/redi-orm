package types

import "fmt"

// DriverType represents a database driver type
// It's defined as a string to allow extensibility for new database drivers
type DriverType string

// Well-known driver types (for convenience and documentation)
const (
	DriverSQLite     DriverType = "sqlite"
	DriverMySQL      DriverType = "mysql"
	DriverPostgreSQL DriverType = "postgresql"
)

// String returns the string representation of the driver type
func (d DriverType) String() string {
	return string(d)
}

// DriverCapabilities defines what a driver supports
type DriverCapabilities interface {
	// SQL dialect features
	SupportsReturning() bool
	SupportsDefaultValues() bool
	RequiresLimitForOffset() bool
	SupportsDistinctOn() bool

	// Identifier quoting
	QuoteIdentifier(name string) string
	GetPlaceholder(index int) string

	// Type conversion
	GetBooleanLiteral(value bool) string
	NeedsTypeConversion() bool
	GetNullsOrderingSQL(direction Order, nullsFirst bool) string

	// Index/Table detection
	IsSystemIndex(indexName string) bool
	IsSystemTable(tableName string) bool

	// Driver identification
	GetDriverType() DriverType
	GetSupportedSchemes() []string
}

// ParseDriverType parses a string into a DriverType
// This is primarily used for parsing configuration and maintaining backward compatibility
func ParseDriverType(s string) (DriverType, error) {
	// Allow any string as a driver type for extensibility
	if s == "" {
		return "", fmt.Errorf("driver type cannot be empty")
	}
	return DriverType(s), nil
}
