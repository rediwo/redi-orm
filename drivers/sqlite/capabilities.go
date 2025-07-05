package sqlite

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// SQLiteCapabilities implements types.DriverCapabilities for SQLite
type SQLiteCapabilities struct{}

// NewSQLiteCapabilities creates new SQLite capabilities
func NewSQLiteCapabilities() *SQLiteCapabilities {
	return &SQLiteCapabilities{}
}

// SQL dialect features

func (c *SQLiteCapabilities) SupportsReturning() bool {
	return true
}

func (c *SQLiteCapabilities) SupportsDefaultValues() bool {
	return true
}

func (c *SQLiteCapabilities) RequiresLimitForOffset() bool {
	return true // SQLite requires LIMIT when using OFFSET
}

func (c *SQLiteCapabilities) SupportsDistinctOn() bool {
	return false // SQLite doesn't support DISTINCT ON
}

// Identifier quoting

func (c *SQLiteCapabilities) QuoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", name)
}

func (c *SQLiteCapabilities) GetPlaceholder(index int) string {
	return "?"
}

// Type conversion

func (c *SQLiteCapabilities) GetBooleanLiteral(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (c *SQLiteCapabilities) NeedsTypeConversion() bool {
	return false // SQLite doesn't need special type conversion like MySQL
}

func (c *SQLiteCapabilities) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	// SQLite supports NULLS FIRST/LAST
	// This method should return only the NULLS ordering part, not the direction
	nullsOrder := " NULLS LAST"
	if nullsFirst {
		nullsOrder = " NULLS FIRST"
	}
	return nullsOrder
}

// Index/Table detection

func (c *SQLiteCapabilities) IsSystemIndex(indexName string) bool {
	lower := strings.ToLower(indexName)

	// SQLite system index patterns:
	// - sqlite_autoindex_*
	// - sqlite_*
	// - pk_*
	return strings.HasPrefix(lower, "sqlite_autoindex_") ||
		strings.HasPrefix(lower, "sqlite_") ||
		strings.HasPrefix(lower, "pk_")
}

func (c *SQLiteCapabilities) IsSystemTable(tableName string) bool {
	lower := strings.ToLower(tableName)

	// SQLite system tables:
	// - sqlite_master
	// - sqlite_sequence
	// - sqlite_stat*
	// - sqlite_*
	return strings.HasPrefix(lower, "sqlite_")
}

// Driver identification

func (c *SQLiteCapabilities) GetDriverType() types.DriverType {
	return types.DriverSQLite
}

func (c *SQLiteCapabilities) GetSupportedSchemes() []string {
	return []string{"sqlite", "sqlite3"}
}
