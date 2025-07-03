package types

import (
	"testing"
)

// Mock FieldMapper for testing
type mockFieldMapper struct {
	mappings map[string]map[string]string // modelName -> fieldName -> columnName
}

func (m *mockFieldMapper) SchemaToColumn(modelName, fieldName string) (string, error) {
	if modelMappings, ok := m.mappings[modelName]; ok {
		if columnName, ok := modelMappings[fieldName]; ok {
			return columnName, nil
		}
	}
	return fieldName, nil
}

func (m *mockFieldMapper) ColumnToSchema(modelName, columnName string) (string, error) {
	return columnName, nil
}

func (m *mockFieldMapper) SchemaFieldsToColumns(modelName string, fieldNames []string) ([]string, error) {
	columns := make([]string, len(fieldNames))
	for i, field := range fieldNames {
		col, _ := m.SchemaToColumn(modelName, field)
		columns[i] = col
	}
	return columns, nil
}

func (m *mockFieldMapper) ColumnFieldsToSchema(modelName string, columnNames []string) ([]string, error) {
	return columnNames, nil
}

func (m *mockFieldMapper) MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error) {
	return data, nil
}

func (m *mockFieldMapper) MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error) {
	return data, nil
}

func (m *mockFieldMapper) ModelToTable(modelName string) (string, error) {
	return modelName + "s", nil
}

func TestConditionContext_MapFieldToColumn(t *testing.T) {
	mapper := &mockFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"firstName": "first_name",
				"userId":    "user_id",
			},
		},
	}

	tests := []struct {
		name       string
		ctx        *ConditionContext
		fieldName  string
		want       string
	}{
		{
			name: "with table alias",
			ctx: NewConditionContext(mapper, "User", "u"),
			fieldName: "firstName",
			want: "u.first_name",
		},
		{
			name: "without table alias",
			ctx: NewConditionContext(mapper, "User", ""),
			fieldName: "firstName",
			want: "first_name",
		},
		{
			name: "unmapped field",
			ctx: NewConditionContext(mapper, "User", "u"),
			fieldName: "email",
			want: "u.email",
		},
		{
			name: "nil mapper",
			ctx: NewConditionContext(nil, "User", "u"),
			fieldName: "firstName",
			want: "firstName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.ctx.MapFieldToColumn(tt.fieldName)
			if got != tt.want {
				t.Errorf("MapFieldToColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMappedFieldCondition_ToSQL(t *testing.T) {
	mapper := &mockFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"firstName": "first_name",
				"userId":    "user_id",
			},
		},
	}

	tests := []struct {
		name     string
		cond     *MappedFieldCondition
		ctx      *ConditionContext
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "equals with mapping",
			cond: &MappedFieldCondition{
				BaseCondition: BaseCondition{
					SQL:  "firstName = ?",
					Args: []any{"John"},
				},
				fieldName: "firstName",
				modelName: "User",
			},
			ctx:      NewConditionContext(mapper, "User", "u"),
			wantSQL:  "u.first_name = ?",
			wantArgs: []any{"John"},
		},
		{
			name: "in clause with mapping",
			cond: &MappedFieldCondition{
				BaseCondition: BaseCondition{
					SQL:  "userId IN (?,?,?)",
					Args: []any{1, 2, 3},
				},
				fieldName: "userId",
				modelName: "User",
			},
			ctx:      NewConditionContext(mapper, "User", ""),
			wantSQL:  "user_id IN (?,?,?)",
			wantArgs: []any{1, 2, 3},
		},
		{
			name: "nil context",
			cond: &MappedFieldCondition{
				BaseCondition: BaseCondition{
					SQL:  "firstName = ?",
					Args: []any{"John"},
				},
				fieldName: "firstName",
				modelName: "User",
			},
			ctx:      nil,
			wantSQL:  "firstName = ?",
			wantArgs: []any{"John"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.cond.ToSQL(tt.ctx)
			if gotSQL != tt.wantSQL {
				t.Errorf("ToSQL() SQL = %v, want %v", gotSQL, tt.wantSQL)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("ToSQL() Args length = %v, want %v", len(gotArgs), len(tt.wantArgs))
			}
			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("ToSQL() Args[%d] = %v, want %v", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestFieldConditionImpl_Methods(t *testing.T) {
	mapper := &mockFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"firstName": "first_name",
				"userId":    "user_id",
			},
		},
	}
	ctx := NewConditionContext(mapper, "User", "u")

	field := NewFieldCondition("User", "firstName")

	// Test Equals
	cond := field.Equals("John")
	sql, args := cond.ToSQL(ctx)
	if sql != "u.first_name = ?" || len(args) != 1 || args[0] != "John" {
		t.Errorf("Equals() failed: sql=%v, args=%v", sql, args)
	}

	// Test In
	cond = field.In("A", "B", "C")
	sql, args = cond.ToSQL(ctx)
	if sql != "u.first_name IN (?,?,?)" || len(args) != 3 {
		t.Errorf("In() failed: sql=%v, args=%v", sql, args)
	}

	// Test IsNull
	cond = field.IsNull()
	sql, args = cond.ToSQL(ctx)
	if sql != "u.first_name IS NULL" || len(args) != 0 {
		t.Errorf("IsNull() failed: sql=%v, args=%v", sql, args)
	}

	// Test Between
	cond = field.Between(10, 20)
	sql, args = cond.ToSQL(ctx)
	if sql != "u.first_name BETWEEN ? AND ?" || len(args) != 2 {
		t.Errorf("Between() failed: sql=%v, args=%v", sql, args)
	}
}

func TestComplexConditions_ToSQL(t *testing.T) {
	mapper := &mockFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"firstName": "first_name",
				"lastName":  "last_name",
				"age":       "age",
			},
		},
	}
	ctx := NewConditionContext(mapper, "User", "u")

	// Create complex condition: (firstName = 'John' AND lastName = 'Doe') OR age > 30
	firstNameCond := NewFieldCondition("User", "firstName").Equals("John")
	lastNameCond := NewFieldCondition("User", "lastName").Equals("Doe")
	ageCond := NewFieldCondition("User", "age").GreaterThan(30)

	complexCond := And(firstNameCond, lastNameCond).Or(ageCond)

	sql, args := complexCond.ToSQL(ctx)
	expectedSQL := "((u.first_name = ?) AND (u.last_name = ?)) OR (u.age > ?)"
	if sql != expectedSQL {
		t.Errorf("Complex condition SQL = %v, want %v", sql, expectedSQL)
	}
	if len(args) != 3 || args[0] != "John" || args[1] != "Doe" || args[2] != 30 {
		t.Errorf("Complex condition Args = %v", args)
	}
}