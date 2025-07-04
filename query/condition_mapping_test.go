package query

import (
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// TestConditionFieldMapping tests that conditions properly map field names to column names
func TestConditionFieldMapping(t *testing.T) {
	// Create a mock field mapper
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"firstName":    "first_name",
				"lastName":     "last_name",
				"emailAddress": "email",
				"userId":       "user_id",
			},
		},
	}

	tests := []struct {
		name         string
		modelName    string
		tableAlias   string
		buildCond    func() types.Condition
		expectedSQL  string
		expectedArgs []any
	}{
		{
			name:       "simple equals with alias",
			modelName:  "User",
			tableAlias: "u",
			buildCond: func() types.Condition {
				return types.NewFieldCondition("User", "firstName").Equals("John")
			},
			expectedSQL:  "u.first_name = ?",
			expectedArgs: []any{"John"},
		},
		{
			name:       "simple equals without alias",
			modelName:  "User",
			tableAlias: "",
			buildCond: func() types.Condition {
				return types.NewFieldCondition("User", "firstName").Equals("John")
			},
			expectedSQL:  "first_name = ?",
			expectedArgs: []any{"John"},
		},
		{
			name:       "IN clause",
			modelName:  "User",
			tableAlias: "u",
			buildCond: func() types.Condition {
				return types.NewFieldCondition("User", "userId").In(1, 2, 3)
			},
			expectedSQL:  "u.user_id IN (?,?,?)",
			expectedArgs: []any{1, 2, 3},
		},
		{
			name:       "LIKE clause",
			modelName:  "User",
			tableAlias: "u",
			buildCond: func() types.Condition {
				return types.NewFieldCondition("User", "emailAddress").Like("%@example.com")
			},
			expectedSQL:  "u.email LIKE ?",
			expectedArgs: []any{"%@example.com"},
		},
		{
			name:       "IS NULL",
			modelName:  "User",
			tableAlias: "u",
			buildCond: func() types.Condition {
				return types.NewFieldCondition("User", "lastName").IsNull()
			},
			expectedSQL:  "u.last_name IS NULL",
			expectedArgs: []any{},
		},
		{
			name:       "complex AND/OR",
			modelName:  "User",
			tableAlias: "u",
			buildCond: func() types.Condition {
				cond1 := types.NewFieldCondition("User", "firstName").Equals("John")
				cond2 := types.NewFieldCondition("User", "lastName").Equals("Doe")
				cond3 := types.NewFieldCondition("User", "emailAddress").Contains("example")
				return cond1.And(cond2).Or(cond3)
			},
			expectedSQL:  "((u.first_name = ?) AND (u.last_name = ?)) OR (u.email LIKE ?)",
			expectedArgs: []any{"John", "Doe", "%example%"},
		},
		{
			name:       "BETWEEN clause",
			modelName:  "User",
			tableAlias: "",
			buildCond: func() types.Condition {
				return types.NewFieldCondition("User", "userId").Between(10, 20)
			},
			expectedSQL:  "user_id BETWEEN ? AND ?",
			expectedArgs: []any{10, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create condition context
			ctx := types.NewConditionContext(mapper, tt.modelName, tt.tableAlias)

			// Build condition
			cond := tt.buildCond()

			// Get SQL
			sql, args := cond.ToSQL(ctx)

			// Check SQL
			if sql != tt.expectedSQL {
				t.Errorf("SQL mismatch\nGot:      %s\nExpected: %s", sql, tt.expectedSQL)
			}

			// Check args
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Args length mismatch: got %d, expected %d", len(args), len(tt.expectedArgs))
			} else {
				for i, arg := range args {
					if arg != tt.expectedArgs[i] {
						t.Errorf("Arg[%d] mismatch: got %v, expected %v", i, arg, tt.expectedArgs[i])
					}
				}
			}
		})
	}
}

// testFieldMapper for testing
type testFieldMapper struct {
	mappings map[string]map[string]string
}

func (m *testFieldMapper) SchemaToColumn(modelName, fieldName string) (string, error) {
	if modelMappings, ok := m.mappings[modelName]; ok {
		if columnName, ok := modelMappings[fieldName]; ok {
			return columnName, nil
		}
	}
	return fieldName, nil
}

func (m *testFieldMapper) ColumnToSchema(modelName, columnName string) (string, error) {
	return columnName, nil
}

func (m *testFieldMapper) SchemaFieldsToColumns(modelName string, fieldNames []string) ([]string, error) {
	columns := make([]string, len(fieldNames))
	for i, field := range fieldNames {
		col, _ := m.SchemaToColumn(modelName, field)
		columns[i] = col
	}
	return columns, nil
}

func (m *testFieldMapper) ColumnFieldsToSchema(modelName string, columnNames []string) ([]string, error) {
	return columnNames, nil
}

func (m *testFieldMapper) MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range data {
		col, _ := m.SchemaToColumn(modelName, k)
		result[col] = v
	}
	return result, nil
}

func (m *testFieldMapper) MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error) {
	return data, nil
}

func (m *testFieldMapper) ModelToTable(modelName string) (string, error) {
	return strings.ToLower(modelName) + "s", nil
}
