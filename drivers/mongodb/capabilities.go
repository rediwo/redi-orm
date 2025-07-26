package mongodb

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// MongoDBCapabilities implements DriverCapabilities for MongoDB
type MongoDBCapabilities struct{}

// NewMongoDBCapabilities creates new MongoDB capabilities
func NewMongoDBCapabilities() *MongoDBCapabilities {
	return &MongoDBCapabilities{}
}

// SQL dialect features (mostly not applicable to MongoDB)
func (c *MongoDBCapabilities) SupportsReturning() bool {
	// MongoDB doesn't support SQL-style RETURNING clause
	return false
}

func (c *MongoDBCapabilities) SupportsDefaultValues() bool {
	// MongoDB doesn't have database-level default values
	return false
}

func (c *MongoDBCapabilities) RequiresLimitForOffset() bool {
	// MongoDB supports skip without limit
	return false
}

func (c *MongoDBCapabilities) SupportsDistinctOn() bool {
	// MongoDB has distinct but works differently
	return false
}

func (c *MongoDBCapabilities) SupportsForeignKeys() bool {
	// MongoDB doesn't support foreign key constraints
	return false
}

// Identifier quoting
func (c *MongoDBCapabilities) QuoteIdentifier(name string) string {
	// MongoDB doesn't quote identifiers
	return name
}

func (c *MongoDBCapabilities) GetPlaceholder(index int) string {
	// MongoDB uses named parameters in queries, but we'll handle this differently
	return fmt.Sprintf("$%d", index)
}

// Type conversion
func (c *MongoDBCapabilities) GetBooleanLiteral(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (c *MongoDBCapabilities) NeedsTypeConversion() bool {
	// MongoDB handles types differently
	return true
}

func (c *MongoDBCapabilities) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	// MongoDB handles nulls differently in sorting
	return ""
}

// Index/Table detection
func (c *MongoDBCapabilities) IsSystemIndex(indexName string) bool {
	// MongoDB system indexes
	return indexName == "_id_" || strings.HasPrefix(indexName, "system.")
}

func (c *MongoDBCapabilities) IsSystemTable(tableName string) bool {
	// MongoDB system collections
	return strings.HasPrefix(tableName, "system.")
}

// Driver identification
func (c *MongoDBCapabilities) GetDriverType() types.DriverType {
	return types.DriverMongoDB
}

func (c *MongoDBCapabilities) GetSupportedSchemes() []string {
	return []string{"mongodb", "mongodb+srv"}
}

// NoSQL features
func (c *MongoDBCapabilities) IsNoSQL() bool {
	return true
}

func (c *MongoDBCapabilities) SupportsTransactions() bool {
	// MongoDB 4.0+ supports multi-document transactions
	return true
}

func (c *MongoDBCapabilities) SupportsNestedDocuments() bool {
	return true
}

func (c *MongoDBCapabilities) SupportsArrayFields() bool {
	return true
}

func (c *MongoDBCapabilities) SupportsAggregationPipeline() bool {
	return true
}
