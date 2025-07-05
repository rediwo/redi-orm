package agile

import (
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Connection tests
func (act *AgileConformanceTests) runConnectionTests(t *testing.T, _ *Client, _ types.Database) {
	// TODO: Implement connection tests
	t.Skip("Connection tests not yet implemented")
}

// Schema tests
func (act *AgileConformanceTests) runSchemaTests(t *testing.T, _ *Client, _ types.Database) {
	// TODO: Implement schema tests
	t.Skip("Schema tests not yet implemented")
}

// The following tests are now implemented in their respective files:
// - Query tests: agile_conformance_tests_query.go
// - Aggregation tests: agile_conformance_tests_aggregations.go
// - Relation tests: agile_conformance_tests_relations.go
// - Transaction tests: agile_conformance_tests_transactions.go