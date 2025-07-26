package query

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// mockDatabase is a shared mock implementation for testing
type mockDatabase struct {
	schemas map[string]*schema.Schema
}

func (m *mockDatabase) Connect(ctx context.Context) error { return nil }
func (m *mockDatabase) Close() error                      { return nil }
func (m *mockDatabase) Ping(ctx context.Context) error    { return nil }
func (m *mockDatabase) RegisterSchema(modelName string, s *schema.Schema) error {
	if m.schemas == nil {
		m.schemas = make(map[string]*schema.Schema)
	}
	m.schemas[modelName] = s
	return nil
}
func (m *mockDatabase) GetSchema(modelName string) (*schema.Schema, error) {
	if s, ok := m.schemas[modelName]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("schema not found")
}
func (m *mockDatabase) CreateModel(ctx context.Context, modelName string) error   { return nil }
func (m *mockDatabase) DropModel(ctx context.Context, modelName string) error     { return nil }
func (m *mockDatabase) LoadSchema(ctx context.Context, content string) error      { return nil }
func (m *mockDatabase) LoadSchemaFrom(ctx context.Context, filename string) error { return nil }
func (m *mockDatabase) SyncSchemas(ctx context.Context) error                     { return nil }
func (m *mockDatabase) Model(modelName string) types.ModelQuery                   { return nil }
func (m *mockDatabase) Raw(sql string, args ...any) types.RawQuery                { return nil }
func (m *mockDatabase) Begin(ctx context.Context) (types.Transaction, error)      { return nil, nil }
func (m *mockDatabase) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	return nil
}
func (m *mockDatabase) GetModels() []string { return nil }
func (m *mockDatabase) GetModelSchema(modelName string) (*schema.Schema, error) {
	return m.GetSchema(modelName)
}
func (m *mockDatabase) ResolveTableName(modelName string) (string, error)            { return "", nil }
func (m *mockDatabase) ResolveFieldName(modelName, fieldName string) (string, error) { return "", nil }
func (m *mockDatabase) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return nil, nil
}
func (m *mockDatabase) Exec(query string, args ...any) (sql.Result, error) { return nil, nil }
func (m *mockDatabase) Query(query string, args ...any) (*sql.Rows, error) { return nil, nil }
func (m *mockDatabase) QueryRow(query string, args ...any) *sql.Row        { return nil }
func (m *mockDatabase) GetMigrator() types.DatabaseMigrator                { return nil }
func (m *mockDatabase) GetDriverType() string                              { return "mock" }
func (m *mockDatabase) GetBooleanLiteral(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (m *mockDatabase) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

func (m *mockDatabase) SupportsDefaultValues() bool {
	return true
}

func (m *mockDatabase) SupportsReturning() bool {
	return false
}

func (m *mockDatabase) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	return "" // Mock doesn't support NULLS FIRST/LAST
}

func (m *mockDatabase) RequiresLimitForOffset() bool {
	return true // Mock requires LIMIT for OFFSET
}

func (m *mockDatabase) GetCapabilities() types.DriverCapabilities {
	return &mockCapabilities{}
}

func (m *mockDatabase) SetLogger(log logger.Logger) {
	// Mock implementation - do nothing
}

func (m *mockDatabase) GetLogger() logger.Logger {
	return nil
}

// mockCapabilities implements types.DriverCapabilities for testing
type mockCapabilities struct{}

func (m *mockCapabilities) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}

func (m *mockCapabilities) GetPlaceholder(index int) string {
	return "?"
}

func (m *mockCapabilities) GetBooleanLiteral(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (m *mockCapabilities) SupportsDefaultValues() bool {
	return true
}

func (m *mockCapabilities) SupportsReturning() bool {
	return false
}

func (m *mockCapabilities) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	return ""
}

func (m *mockCapabilities) RequiresLimitForOffset() bool {
	return true
}

func (m *mockCapabilities) GetDriverType() types.DriverType {
	return types.DriverType("mock")
}

func (m *mockCapabilities) GetSupportedSchemes() []string {
	return []string{"mock"}
}

func (m *mockCapabilities) SupportsDistinctOn() bool {
	return false
}

func (m *mockCapabilities) NeedsTypeConversion() bool {
	return false
}

func (m *mockCapabilities) IsSystemIndex(indexName string) bool {
	return false
}

func (m *mockCapabilities) IsSystemTable(tableName string) bool {
	return false
}

func (m *mockCapabilities) IsNoSQL() bool {
	return false
}

func (m *mockCapabilities) SupportsAggregation() bool {
	return true
}

func (m *mockCapabilities) SupportsTransactions() bool {
	return true
}

func (m *mockCapabilities) SupportsNestedDocuments() bool {
	return false
}

func (m *mockCapabilities) SupportsArrayFields() bool {
	return false
}

func (m *mockCapabilities) SupportsAggregationPipeline() bool {
	return false
}

func (m *mockCapabilities) SupportsForeignKeys() bool {
	return true
}
