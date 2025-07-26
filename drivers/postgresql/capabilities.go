package postgresql

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// PostgreSQLCapabilities implements types.DriverCapabilities for PostgreSQL
type PostgreSQLCapabilities struct{}

// NewPostgreSQLCapabilities creates new PostgreSQL capabilities
func NewPostgreSQLCapabilities() *PostgreSQLCapabilities {
	return &PostgreSQLCapabilities{}
}

// SQL dialect features

func (c *PostgreSQLCapabilities) SupportsReturning() bool {
	return true
}

func (c *PostgreSQLCapabilities) SupportsDefaultValues() bool {
	return true
}

func (c *PostgreSQLCapabilities) RequiresLimitForOffset() bool {
	return false
}

func (c *PostgreSQLCapabilities) SupportsDistinctOn() bool {
	return true // PostgreSQL supports DISTINCT ON
}

func (c *PostgreSQLCapabilities) SupportsForeignKeys() bool {
	return true // PostgreSQL supports foreign key constraints
}

// Identifier quoting

func (c *PostgreSQLCapabilities) QuoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

func (c *PostgreSQLCapabilities) GetPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// Type conversion

func (c *PostgreSQLCapabilities) GetBooleanLiteral(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}

func (c *PostgreSQLCapabilities) NeedsTypeConversion() bool {
	return false // PostgreSQL doesn't need special type conversion
}

func (c *PostgreSQLCapabilities) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	// PostgreSQL supports NULLS FIRST/LAST
	// This method should return only the NULLS ordering part, not the direction
	nullsOrder := " NULLS LAST"
	if nullsFirst {
		nullsOrder = " NULLS FIRST"
	}
	return nullsOrder
}

// Index/Table detection

func (c *PostgreSQLCapabilities) IsSystemIndex(indexName string) bool {
	lower := strings.ToLower(indexName)

	// PostgreSQL system index patterns:
	// - Primary key: tablename_pkey
	// - Unique constraints: tablename_columnname_key
	// - Foreign key: tablename_columnname_fkey
	// - System: pg_*
	return strings.HasSuffix(lower, "_pkey") ||
		strings.HasSuffix(lower, "_key") ||
		strings.HasSuffix(lower, "_fkey") ||
		strings.HasPrefix(lower, "pg_")
}

func (c *PostgreSQLCapabilities) IsSystemTable(tableName string) bool {
	lower := strings.ToLower(tableName)

	// PostgreSQL system tables:
	// - pg_* (system catalogs)
	// - information_schema.*
	// - pg_catalog.*
	return strings.HasPrefix(lower, "pg_") ||
		strings.HasPrefix(lower, "information_schema.") ||
		strings.HasPrefix(lower, "pg_catalog.") ||
		lower == "information_schema" ||
		lower == "pg_catalog"
}

// Driver identification

func (c *PostgreSQLCapabilities) GetDriverType() types.DriverType {
	return types.DriverPostgreSQL
}

func (c *PostgreSQLCapabilities) GetSupportedSchemes() []string {
	return []string{"postgresql", "postgres"}
}

// NoSQL features (PostgreSQL is not a NoSQL database but has some features)

func (c *PostgreSQLCapabilities) IsNoSQL() bool {
	return false
}

func (c *PostgreSQLCapabilities) SupportsTransactions() bool {
	return true
}

func (c *PostgreSQLCapabilities) SupportsNestedDocuments() bool {
	return false // PostgreSQL has JSON/JSONB but not full document support
}

func (c *PostgreSQLCapabilities) SupportsArrayFields() bool {
	return true // PostgreSQL has native array support
}

func (c *PostgreSQLCapabilities) SupportsAggregationPipeline() bool {
	return false
}
