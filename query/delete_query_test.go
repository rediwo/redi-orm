package query

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Mock types for delete testing
type deleteMockCondition struct {
	field string
	value any
}

func (m *deleteMockCondition) ToSQL(ctx *types.ConditionContext) (string, []any) {
	return m.field + " = ?", []any{m.value}
}

func (m *deleteMockCondition) And(other types.Condition) types.Condition { return m }
func (m *deleteMockCondition) Or(other types.Condition) types.Condition  { return m }
func (m *deleteMockCondition) Not() types.Condition                      { return m }

type deleteMockRawQuery struct {
	execResult types.Result
	execError  error
}

func (m *deleteMockRawQuery) Exec(ctx context.Context) (types.Result, error) {
	return m.execResult, m.execError
}

func (m *deleteMockRawQuery) FindOne(ctx context.Context, dest any) error {
	return nil
}

func (m *deleteMockRawQuery) Find(ctx context.Context, dest any) error {
	return nil
}

// deleteMockDatabase extends mockDatabase for delete tests
type deleteMockDatabase struct {
	mockDatabase
	mockRaw *deleteMockRawQuery
}

func (m *deleteMockDatabase) Raw(sql string, args ...any) types.RawQuery {
	if m.mockRaw != nil {
		return m.mockRaw
	}
	return &deleteMockRawQuery{}
}

func TestNewDeleteQuery(t *testing.T) {
	mockDB := &mockDatabase{}
	mapper := &testFieldMapper{}
	
	baseQuery := &ModelQueryImpl{
		database:    mockDB,
		modelName:   "User",
		fieldMapper: mapper,
	}
	
	query := NewDeleteQuery(baseQuery)
	
	if query == nil {
		t.Fatal("NewDeleteQuery returned nil")
	}
	if query.ModelQueryImpl == nil {
		t.Error("NewDeleteQuery ModelQueryImpl is nil")
	}
}

func TestDeleteQuery_Where(t *testing.T) {
	mockDB := &mockDatabase{}
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {"id": "id"},
		},
	}
	
	query := &DeleteQueryImpl{
		ModelQueryImpl: &ModelQueryImpl{
			database:    mockDB,
			modelName:   "User",
			fieldMapper: mapper,
			conditions:  []types.Condition{},
		},
	}
	
	// Test Where with field condition
	fieldCond := query.Where("id")
	
	// Test chaining with Equals
	condition := fieldCond.Equals(123)
	finalQuery := query.WhereCondition(condition)
	
	// Check that condition was added
	if deleteQuery, ok := finalQuery.(*DeleteQueryImpl); ok {
		if len(deleteQuery.whereConditions) != 1 {
			t.Error("WhereCondition() did not add condition")
		}
	}
}

func TestDeleteQuery_WhereCondition(t *testing.T) {
	query := &DeleteQueryImpl{
		ModelQueryImpl: &ModelQueryImpl{
			conditions: []types.Condition{},
		},
		whereConditions: []types.Condition{},
	}
	
	condition := &deleteMockCondition{}
	newQuery := query.WhereCondition(condition)
	
	// Original query should be unchanged
	if len(query.whereConditions) != 0 {
		t.Error("WhereCondition() modified original query")
	}
	
	// New query should have condition
	deleteQuery := newQuery.(*DeleteQueryImpl)
	if len(deleteQuery.whereConditions) != 1 {
		t.Error("WhereCondition() did not add condition")
	}
}

func TestDeleteQuery_Returning(t *testing.T) {
	query := &DeleteQueryImpl{
		ModelQueryImpl:  &ModelQueryImpl{},
		returningFields: []string{},
	}
	
	newQuery := query.Returning("id", "deletedAt")
	
	// Original query should be unchanged
	if len(query.returningFields) != 0 {
		t.Error("Returning() modified original query")
	}
	
	// New query should have returning fields
	deleteQuery := newQuery.(*DeleteQueryImpl)
	if len(deleteQuery.returningFields) != 2 {
		t.Errorf("Returning() fields length = %d, want 2", len(deleteQuery.returningFields))
	}
}

func TestDeleteQuery_BuildSQL(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"id":    "id",
				"name":  "name",
				"email": "email",
			},
		},
	}
	
	tests := []struct {
		name          string
		modelName     string
		conditions    []types.Condition
		driverType    string
		wantSQL       string
		wantArgsCount int
		wantErr       bool
	}{
		{
			name:          "simple delete",
			modelName:     "User",
			conditions:    []types.Condition{},
			driverType:    "sqlite",
			wantSQL:       "",
			wantArgsCount: 0,
			wantErr:       true, // DELETE without WHERE is not allowed
		},
		{
			name:      "delete with condition",
			modelName: "User",
			conditions: []types.Condition{
				&deleteMockCondition{field: "id", value: "123"},
			},
			driverType:    "mysql",
			wantSQL:       "DELETE FROM users WHERE",
			wantArgsCount: 1,
		},
		{
			name:      "delete with multiple conditions",
			modelName: "User",
			conditions: []types.Condition{
				&deleteMockCondition{field: "id", value: "123"},
				&deleteMockCondition{field: "name", value: "John"},
			},
			driverType:    "postgresql",
			wantSQL:       "DELETE FROM users WHERE",
			wantArgsCount: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDatabase{}
			
			query := &DeleteQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   tt.modelName,
					fieldMapper: mapper,
					conditions:  tt.conditions,
				},
			}
			
			sql, args, err := query.BuildSQL()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if !strings.Contains(sql, tt.wantSQL) {
					t.Errorf("BuildSQL() SQL = %v, want to contain %v", sql, tt.wantSQL)
				}
				if len(args) != tt.wantArgsCount {
					t.Errorf("BuildSQL() args count = %d, want %d", len(args), tt.wantArgsCount)
				}
			}
		})
	}
}

func TestDeleteQuery_Exec(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {"name": "name"},
		},
	}
	
	tests := []struct {
		name       string
		conditions []types.Condition
		execResult types.Result
		execError  error
		wantErr    bool
	}{
		{
			name: "successful delete",
			conditions: []types.Condition{
				&deleteMockCondition{field: "id", value: 1},
			},
			execResult: types.Result{
				RowsAffected: 1,
			},
			wantErr: false,
		},
		{
			name: "exec error",
			conditions: []types.Condition{
				&deleteMockCondition{field: "id", value: 1},
			},
			execError: errors.New("database error"),
			wantErr:   true,
		},
		{
			name:       "delete without condition",
			conditions: []types.Condition{},
			execResult: types.Result{
				RowsAffected: 10,
			},
			wantErr: true, // DELETE without WHERE is not allowed
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRaw := &deleteMockRawQuery{
				execResult: tt.execResult,
				execError:  tt.execError,
			}
			
			mockDB := &deleteMockDatabase{
				mockRaw: mockRaw,
			}
			
			query := &DeleteQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   "User",
					fieldMapper: mapper,
					conditions:  tt.conditions,
				},
			}
			
			result, err := query.Exec(context.Background())
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Exec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if result.RowsAffected != tt.execResult.RowsAffected {
					t.Errorf("Exec() RowsAffected = %d, want %d", result.RowsAffected, tt.execResult.RowsAffected)
				}
			}
		})
	}
}