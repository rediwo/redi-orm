package mysql

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// MySQLCapabilities implements types.DriverCapabilities for MySQL
type MySQLCapabilities struct{}

// NewMySQLCapabilities creates new MySQL capabilities
func NewMySQLCapabilities() *MySQLCapabilities {
	return &MySQLCapabilities{}
}

// SQL dialect features

func (c *MySQLCapabilities) SupportsReturning() bool {
	return false // MySQL doesn't support RETURNING clause
}

func (c *MySQLCapabilities) SupportsDefaultValues() bool {
	return false // MySQL doesn't support DEFAULT VALUES
}

func (c *MySQLCapabilities) RequiresLimitForOffset() bool {
	return true // MySQL requires LIMIT when using OFFSET
}

func (c *MySQLCapabilities) SupportsDistinctOn() bool {
	return false // MySQL doesn't support DISTINCT ON
}

// Identifier quoting

func (c *MySQLCapabilities) QuoteIdentifier(name string) string {
	return fmt.Sprintf("`%s`", name)
}

func (c *MySQLCapabilities) GetPlaceholder(index int) string {
	return "?"
}

// Type conversion

func (c *MySQLCapabilities) GetBooleanLiteral(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}

func (c *MySQLCapabilities) NeedsTypeConversion() bool {
	return true // MySQL returns numeric values as strings
}

func (c *MySQLCapabilities) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	// MySQL doesn't support NULLS FIRST/LAST syntax
	// It always puts NULLs first for ASC and last for DESC
	// This method should return only the NULLS ordering part, not the direction
	// Since MySQL doesn't support explicit NULLS ordering, return empty string
	return ""
}

// Index/Table detection

func (c *MySQLCapabilities) IsSystemIndex(indexName string) bool {
	lower := strings.ToLower(indexName)

	// MySQL system index patterns:
	// - Primary key: PRIMARY
	// - Foreign key constraints: fk_*
	// - System internal: mysql_*
	return lower == "primary" ||
		strings.HasPrefix(lower, "fk_") ||
		strings.HasPrefix(lower, "mysql_") ||
		strings.Contains(lower, "primary_key")
}

func (c *MySQLCapabilities) IsSystemTable(tableName string) bool {
	lower := strings.ToLower(tableName)

	// MySQL system table patterns:
	// - mysql.* schema tables
	// - information_schema.* tables
	// - performance_schema.* tables
	// - sys.* schema tables
	return strings.HasPrefix(lower, "mysql.") ||
		strings.HasPrefix(lower, "information_schema.") ||
		strings.HasPrefix(lower, "performance_schema.") ||
		strings.HasPrefix(lower, "sys.") ||
		lower == "mysql" ||
		lower == "information_schema" ||
		lower == "performance_schema" ||
		lower == "sys"
}

// Driver identification

func (c *MySQLCapabilities) GetDriverType() types.DriverType {
	return types.DriverMySQL
}

func (c *MySQLCapabilities) GetSupportedSchemes() []string {
	return []string{"mysql"}
}

// NoSQL features (MySQL is not a NoSQL database)

func (c *MySQLCapabilities) IsNoSQL() bool {
	return false
}

func (c *MySQLCapabilities) SupportsTransactions() bool {
	return true
}

func (c *MySQLCapabilities) SupportsNestedDocuments() bool {
	return false // MySQL has JSON but not full document support
}

func (c *MySQLCapabilities) SupportsArrayFields() bool {
	return false // MySQL has JSON arrays but not native array fields
}

func (c *MySQLCapabilities) SupportsAggregationPipeline() bool {
	return false
}
