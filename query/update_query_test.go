package query

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Mock types for testing
type updateMockCondition struct {
	field string
	value any
}

func (m *updateMockCondition) ToSQL(ctx *types.ConditionContext) (string, []any) {
	return m.field + " = ?", []any{m.value}
}

func (m *updateMockCondition) And(other types.Condition) types.Condition { return m }
func (m *updateMockCondition) Or(other types.Condition) types.Condition  { return m }
func (m *updateMockCondition) Not() types.Condition                      { return m }

type updateMockRawQuery struct {
	execResult    types.Result
	execError     error
	findResult    []map[string]any
	findError     error
	findOneResult map[string]any
	findOneError  error
}

func (m *updateMockRawQuery) Exec(ctx context.Context) (types.Result, error) {
	return m.execResult, m.execError
}

func (m *updateMockRawQuery) FindOne(ctx context.Context, dest any) error {
	if m.findError != nil {
		return m.findError
	}
	// For ExecAndReturn tests
	if m.findResult != nil && len(m.findResult) > 0 {
		if destSlice, ok := dest.(*[]map[string]any); ok {
			*destSlice = m.findResult
			return nil
		}
	}
	return m.findOneError
}

func (m *updateMockRawQuery) Find(ctx context.Context, dest any) error {
	return m.findError
}

// updateMockDatabase extends mockDatabase for update tests
type updateMockDatabase struct {
	mockDatabase
	mockRaw *updateMockRawQuery
}

func (m *updateMockDatabase) Raw(sql string, args ...any) types.RawQuery {
	if m.mockRaw != nil {
		return m.mockRaw
	}
	return &updateMockRawQuery{}
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}


func TestNewUpdateQuery(t *testing.T) {
	mockDB := &mockDatabase{}
	mapper := &testFieldMapper{}
	
	baseQuery := &ModelQueryImpl{
		database:    mockDB,
		modelName:   "User",
		fieldMapper: mapper,
	}
	
	data := map[string]any{"name": "John"}
	query := NewUpdateQuery(baseQuery, data)
	
	if query == nil {
		t.Fatal("NewUpdateQuery returned nil")
	}
	if len(query.setData) != 1 {
		t.Error("NewUpdateQuery did not initialize setData correctly")
	}
}

func TestUpdateQuery_Set(t *testing.T) {
	query := &UpdateQueryImpl{
		ModelQueryImpl: &ModelQueryImpl{},
		setData:        map[string]any{"name": "John"},
	}
	
	newData := map[string]any{"email": "john@example.com"}
	newQuery := query.Set(newData)
	
	// Original query should be unchanged
	if len(query.setData) != 1 {
		t.Error("Set() modified original query")
	}
	
	// New query should have new data
	updateQuery := newQuery.(*UpdateQueryImpl)
	if len(updateQuery.setData) != 2 {
		t.Error("Set() did not merge data correctly")
	}
}

func TestUpdateQuery_Where(t *testing.T) {
	mockDB := &mockDatabase{}
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {"id": "id"},
		},
	}
	
	query := &UpdateQueryImpl{
		ModelQueryImpl: &ModelQueryImpl{
			database:    mockDB,
			modelName:   "User",
			fieldMapper: mapper,
			conditions:  []types.Condition{},
		},
		setData: map[string]any{"name": "John"},
	}
	
	// Test Where with field condition
	fieldCond := query.Where("id")
	
	// Test chaining with Equals and WhereCondition
	condition := fieldCond.Equals(123)
	finalQuery := query.WhereCondition(condition)
	
	// Check that condition was added
	if updateQuery, ok := finalQuery.(*UpdateQueryImpl); ok {
		if len(updateQuery.whereConditions) != 1 {
			t.Error("WhereCondition() did not add condition")
		}
	}
}

func TestUpdateQuery_WhereCondition(t *testing.T) {
	query := &UpdateQueryImpl{
		ModelQueryImpl: &ModelQueryImpl{
			conditions: []types.Condition{},
		},
		setData:         map[string]any{"name": "John"},
		whereConditions: []types.Condition{},
	}
	
	condition := &updateMockCondition{}
	newQuery := query.WhereCondition(condition)
	
	// Original query should be unchanged
	if query.whereConditions != nil && len(query.whereConditions) != 0 {
		t.Error("WhereCondition() modified original query")
	}
	
	// New query should have condition
	updateQuery := newQuery.(*UpdateQueryImpl)
	if len(updateQuery.whereConditions) != 1 {
		t.Error("WhereCondition() did not add condition")
	}
}

func TestUpdateQuery_Increment_Decrement(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"loginCount": "login_count",
				"score":      "score",
			},
		},
	}
	
	tests := []struct {
		name      string
		method    string
		fieldName string
		value     int64
		wantOp    string
	}{
		{
			name:      "increment",
			method:    "increment",
			fieldName: "loginCount",
			value:     1,
			wantOp:    "+",
		},
		{
			name:      "decrement",
			method:    "decrement",
			fieldName: "score",
			value:     5,
			wantOp:    "-",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &UpdateQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					modelName:   "User",
					fieldMapper: mapper,
				},
				setData:   map[string]any{},
				atomicOps: map[string]AtomicOperation{},
			}
			
			var newQuery types.UpdateQuery
			if tt.method == "increment" {
				newQuery = query.Increment(tt.fieldName, tt.value)
			} else {
				newQuery = query.Decrement(tt.fieldName, tt.value)
			}
			
			// Check that increment/decrement was recorded
			updateQuery := newQuery.(*UpdateQueryImpl)
			op, exists := updateQuery.atomicOps[tt.fieldName]
			if !exists {
				t.Errorf("%s() operation not recorded for field %s", tt.method, tt.fieldName)
			} else {
				if op.Value != tt.value {
					t.Errorf("%s() value = %d, want %d", tt.method, op.Value, tt.value)
				}
				expectedType := tt.method
				if op.Type != expectedType {
					t.Errorf("%s() type = %s, want %s", tt.method, op.Type, expectedType)
				}
			}
		})
	}
}

func TestUpdateQuery_Returning(t *testing.T) {
	query := &UpdateQueryImpl{
		ModelQueryImpl:  &ModelQueryImpl{},
		setData:         map[string]any{"name": "John"},
		returningFields: []string{},
	}
	
	newQuery := query.Returning("id", "updatedAt")
	
	// Original query should be unchanged
	if len(query.returningFields) != 0 {
		t.Error("Returning() modified original query")
	}
	
	// New query should have returning fields
	updateQuery := newQuery.(*UpdateQueryImpl)
	if len(updateQuery.returningFields) != 2 {
		t.Errorf("Returning() fields length = %d, want 2", len(updateQuery.returningFields))
	}
}

func TestUpdateQuery_BuildSQL(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"id":         "id",
				"name":       "name",
				"email":      "email",
				"loginCount": "login_count",
				"updatedAt":  "updated_at",
			},
		},
	}
	
	tests := []struct {
		name            string
		modelName       string
		setData         map[string]any
		atomicOps       map[string]AtomicOperation
		conditions      []types.Condition
		returningFields []string
		driverType      string
		wantSQL         string
		wantArgsCount   int
		wantErr         bool
	}{
		{
			name:      "simple update",
			modelName: "User",
			setData:   map[string]any{"name": "John", "email": "john@example.com"},
			driverType: "sqlite",
			wantSQL:   "UPDATE users SET",
			wantArgsCount: 2,
		},
		{
			name:      "update with condition",
			modelName: "User",
			setData:   map[string]any{"name": "John"},
			conditions: []types.Condition{
				&updateMockCondition{field: "id", value: "123"},
			},
			driverType:    "mysql",
			wantSQL:       "UPDATE users SET `name` = ? WHERE",
			wantArgsCount: 2,
		},
		{
			name:       "update with increment",
			modelName:  "User",
			setData:    map[string]any{},
			atomicOps: map[string]AtomicOperation{
				"loginCount": {Type: "increment", Value: 1},
			},
			driverType: "postgresql",
			wantSQL:    "UPDATE users SET `login_count` = `login_count` +",
			wantArgsCount: 1,
		},
		{
			name:       "update with decrement",
			modelName:  "User",
			setData:    map[string]any{},
			atomicOps: map[string]AtomicOperation{
				"loginCount": {Type: "decrement", Value: 5},
			},
			driverType: "postgresql",
			wantSQL:    "UPDATE users SET `login_count` = `login_count` -",
			wantArgsCount: 1,
		},
		{
			name:      "update with returning",
			modelName: "User",
			setData:   map[string]any{"name": "John"},
			returningFields: []string{"id", "updatedAt"},
			driverType:      "postgresql",
			wantSQL:         "UPDATE users SET `name` = ?",
			wantArgsCount:   1,
		},
		{
			name:      "empty update",
			modelName: "User",
			setData:   map[string]any{},
			wantErr:   true,
		},
		{
			name:      "mixed set and atomic",
			modelName: "User",
			setData:   map[string]any{"name": "John"},
			atomicOps: map[string]AtomicOperation{
				"loginCount": {Type: "increment", Value: 1},
			},
			driverType:    "sqlite",
			wantSQL:       "UPDATE users SET",
			wantArgsCount: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDatabase{}
			
			query := &UpdateQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   tt.modelName,
					fieldMapper: mapper,
					conditions:  tt.conditions,
				},
				setData:         tt.setData,
				atomicOps:       tt.atomicOps,
				returningFields: tt.returningFields,
			}
			
			sql, args, err := query.BuildSQL()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if !containsSQL(sql, tt.wantSQL) {
					t.Errorf("BuildSQL() SQL = %v, want to contain %v", sql, tt.wantSQL)
				}
				if len(args) != tt.wantArgsCount {
					t.Errorf("BuildSQL() args count = %d, want %d", len(args), tt.wantArgsCount)
				}
			}
		})
	}
}

func TestUpdateQuery_Exec(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {"name": "name"},
		},
	}
	
	tests := []struct {
		name       string
		setData    map[string]any
		execResult types.Result
		execError  error
		wantErr    bool
	}{
		{
			name:    "successful update",
			setData: map[string]any{"name": "John"},
			execResult: types.Result{
				RowsAffected: 5,
			},
			wantErr: false,
		},
		{
			name:      "exec error",
			setData:   map[string]any{"name": "John"},
			execError: errors.New("database error"),
			wantErr:   true,
		},
		{
			name:    "build SQL error",
			setData: map[string]any{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRaw := &updateMockRawQuery{
				execResult: tt.execResult,
				execError:  tt.execError,
			}
			
			mockDB := &updateMockDatabase{
				mockRaw: mockRaw,
			}
			
			query := &UpdateQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   "User",
					fieldMapper: mapper,
				},
				setData: tt.setData,
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

func TestUpdateQuery_ExecAndReturn(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"id":        "id",
				"name":      "name",
				"updatedAt": "updated_at",
			},
		},
	}
	
	tests := []struct {
		name            string
		setData         map[string]any
		returningFields []string
		findResult      []map[string]any
		findError       error
		wantErr         bool
		errContains     string
	}{
		{
			name:            "successful exec and return",
			setData:         map[string]any{"name": "John"},
			returningFields: []string{"id", "updatedAt"},
			findResult: []map[string]any{
				{"id": 123, "updated_at": "2024-01-01"},
			},
			wantErr: false,
		},
		{
			name:            "no returning fields",
			setData:         map[string]any{"name": "John"},
			returningFields: []string{},
			wantErr:         true,
			errContains:     "no returning fields",
		},
		{
			name:            "find error",
			setData:         map[string]any{"name": "John"},
			returningFields: []string{"id"},
			findError:       errors.New("database error"),
			wantErr:         true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRaw := &updateMockRawQuery{
				findResult:  tt.findResult,
				findError:   tt.findError,
			}
			
			mockDB := &updateMockDatabase{
				mockRaw: mockRaw,
			}
			
			query := &UpdateQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   "User",
					fieldMapper: mapper,
				},
				setData:         tt.setData,
				returningFields: tt.returningFields,
			}
			
			var result []map[string]any
			err := query.ExecAndReturn(context.Background(), &result)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecAndReturn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ExecAndReturn() error = %v, want to contain %v", err, tt.errContains)
				}
			}
			
			if !tt.wantErr && len(result) > 0 {
				if len(result) != len(tt.findResult) {
					t.Errorf("ExecAndReturn() result length = %d, want %d", len(result), len(tt.findResult))
				}
			}
		})
	}
}