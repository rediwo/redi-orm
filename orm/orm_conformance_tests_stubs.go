package orm

import (
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Connection tests
func (act *OrmConformanceTests) runConnectionTests(t *testing.T, _ *Client, _ types.Database) {
	// TODO: Implement connection tests
	t.Skip("Connection tests not yet implemented")
}

// Schema tests
func (act *OrmConformanceTests) runSchemaTests(t *testing.T, _ *Client, _ types.Database) {
	// TODO: Implement schema tests
	t.Skip("Schema tests not yet implemented")
}

// The following tests are now implemented in their respective files:
// - Query tests: orm_conformance_tests_query.go
// - Aggregation tests: orm_conformance_tests_aggregations.go
// - Relation tests: orm_conformance_tests_relations.go
// - Transaction tests: orm_conformance_tests_transactions.go
