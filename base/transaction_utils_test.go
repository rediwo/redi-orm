package base

import (
	"fmt"
	"github.com/rediwo/redi-orm/types"
	"testing"
)

// mockCapabilities implements types.DriverCapabilities for testing
type mockCapabilities struct {
	driverType string
}

func (m *mockCapabilities) QuoteIdentifier(name string) string {
	if m.driverType == "postgresql" {
		return fmt.Sprintf(`"%s"`, name)
	}
	return fmt.Sprintf("`%s`", name)
}

func (m *mockCapabilities) GetPlaceholder(index int) string {
	if m.driverType == "postgresql" {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func (m *mockCapabilities) SupportsDefaultValues() bool {
	return true
}

func (m *mockCapabilities) SupportsReturning() bool {
	return m.driverType == "postgresql"
}

func (m *mockCapabilities) GetNullsOrderingSQL(direction types.Order, nullsFirst bool) string {
	if nullsFirst {
		return "NULLS FIRST"
	}
	return "NULLS LAST"
}

func (m *mockCapabilities) RequiresLimitForOffset() bool {
	return m.driverType == "mysql"
}

func (m *mockCapabilities) GetBooleanLiteral(value bool) string {
	if value {
		return "TRUE"
	}
	return "FALSE"
}

func (m *mockCapabilities) GetDriverType() types.DriverType {
	switch m.driverType {
	case "postgresql":
		return types.DriverPostgreSQL
	case "mysql":
		return types.DriverMySQL
	case "sqlite":
		return types.DriverSQLite
	default:
		return types.DriverSQLite
	}
}

func (m *mockCapabilities) GetSupportedSchemes() []string {
	switch m.driverType {
	case "postgresql":
		return []string{"postgresql", "postgres"}
	case "mysql":
		return []string{"mysql"}
	case "sqlite":
		return []string{"sqlite", "sqlite3"}
	default:
		return []string{m.driverType}
	}
}

func (m *mockCapabilities) SupportsDistinctOn() bool {
	return m.driverType == "postgresql"
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

// NoSQL features
func (m *mockCapabilities) IsNoSQL() bool {
	return false
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

func (m *mockCapabilities) SupportsForeignKeys() bool {
	return m.driverType != "sqlite"
}

func (m *mockCapabilities) SupportsAggregationPipeline() bool {
	return false
}

func TestTransactionUtils_quote(t *testing.T) {
	tests := []struct {
		name       string
		driverType string
		input      string
		want       string
	}{
		{
			name:       "mysql quote",
			driverType: "mysql",
			input:      "table_name",
			want:       "`table_name`",
		},
		{
			name:       "postgresql quote",
			driverType: "postgresql",
			input:      "table_name",
			want:       `"table_name"`,
		},
		{
			name:       "sqlite quote",
			driverType: "sqlite",
			input:      "table_name",
			want:       "`table_name`",
		},
		{
			name:       "default quote",
			driverType: "unknown",
			input:      "table_name",
			want:       "`table_name`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCaps := &mockCapabilities{driverType: tt.driverType}
			tu := &TransactionUtils{capabilities: mockCaps}
			if got := tu.quote(tt.input); got != tt.want {
				t.Errorf("quote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransactionUtils_getPlaceholder(t *testing.T) {
	tests := []struct {
		name       string
		driverType string
		index      int
		want       string
	}{
		{
			name:       "postgresql placeholder",
			driverType: "postgresql",
			index:      1,
			want:       "$1",
		},
		{
			name:       "postgresql placeholder 10",
			driverType: "postgresql",
			index:      10,
			want:       "$10",
		},
		{
			name:       "mysql placeholder",
			driverType: "mysql",
			index:      1,
			want:       "?",
		},
		{
			name:       "sqlite placeholder",
			driverType: "sqlite",
			index:      1,
			want:       "?",
		},
		{
			name:       "default placeholder",
			driverType: "unknown",
			index:      1,
			want:       "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCaps := &mockCapabilities{driverType: tt.driverType}
			tu := &TransactionUtils{capabilities: mockCaps}
			if got := tu.getPlaceholder(tt.index); got != tt.want {
				t.Errorf("getPlaceholder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldMapperWrapper_ModelToTable(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      string
	}{
		{
			name:      "simple model",
			modelName: "User",
			want:      "users",
		},
		{
			name:      "camelCase model",
			modelName: "UserProfile",
			want:      "user_profiles",
		},
		{
			name:      "already plural",
			modelName: "Settings",
			want:      "settingses", // Pluralize always adds suffix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := &fieldMapperWrapper{}
			got, err := fw.ModelToTable(tt.modelName)
			if err != nil {
				t.Errorf("ModelToTable() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ModelToTable() = %v, want %v", got, tt.want)
			}
		})
	}
}
