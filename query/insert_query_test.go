package query

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

func TestNewInsertQuery(t *testing.T) {
	mockDB := &mockDatabase{}
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"id":   "id",
				"name": "name",
			},
		},
	}
	
	baseQuery := &ModelQueryImpl{
		database:    mockDB,
		modelName:   "User",
		fieldMapper: mapper,
	}
	
	data := map[string]any{"name": "John"}
	query := NewInsertQuery(baseQuery, data)
	
	if query == nil {
		t.Fatal("NewInsertQuery returned nil")
	}
	if len(query.data) != 1 {
		t.Errorf("NewInsertQuery data length = %d, want 1", len(query.data))
	}
	if query.conflictAction != types.ConflictIgnore {
		t.Errorf("NewInsertQuery conflictAction = %v, want ConflictIgnore", query.conflictAction)
	}
}

func TestInsertQuery_Values(t *testing.T) {
	query := &InsertQueryImpl{
		ModelQueryImpl: &ModelQueryImpl{},
		data:          []any{map[string]any{"name": "John"}},
	}
	
	// Add more values
	newQuery := query.Values(
		map[string]any{"name": "Jane"},
		map[string]any{"name": "Bob"},
	)
	
	// Original query should be unchanged
	if len(query.data) != 1 {
		t.Error("Values() modified original query")
	}
	
	// New query should have all values
	insertQuery := newQuery.(*InsertQueryImpl)
	if len(insertQuery.data) != 3 {
		t.Errorf("Values() data length = %d, want 3", len(insertQuery.data))
	}
}

func TestInsertQuery_OnConflict(t *testing.T) {
	query := &InsertQueryImpl{
		ModelQueryImpl: &ModelQueryImpl{},
		conflictAction: types.ConflictIgnore,
	}
	
	tests := []struct {
		name   string
		action types.ConflictAction
	}{
		{"replace", types.ConflictReplace},
		{"update", types.ConflictUpdate},
		{"ignore", types.ConflictIgnore},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newQuery := query.OnConflict(tt.action)
			
			// Original query should be unchanged
			if query.conflictAction != types.ConflictIgnore {
				t.Error("OnConflict() modified original query")
			}
			
			// New query should have new action
			insertQuery := newQuery.(*InsertQueryImpl)
			if insertQuery.conflictAction != tt.action {
				t.Errorf("OnConflict() action = %v, want %v", insertQuery.conflictAction, tt.action)
			}
		})
	}
}

func TestInsertQuery_Returning(t *testing.T) {
	query := &InsertQueryImpl{
		ModelQueryImpl:  &ModelQueryImpl{},
		returningFields: []string{},
	}
	
	newQuery := query.Returning("id", "name", "createdAt")
	
	// Original query should be unchanged
	if len(query.returningFields) != 0 {
		t.Error("Returning() modified original query")
	}
	
	// New query should have returning fields
	insertQuery := newQuery.(*InsertQueryImpl)
	if len(insertQuery.returningFields) != 3 {
		t.Errorf("Returning() fields length = %d, want 3", len(insertQuery.returningFields))
	}
}

func TestInsertQuery_BuildSQL(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"id":        "id",
				"name":      "name",
				"email":     "email",
				"createdAt": "created_at",
			},
		},
	}
	
	tests := []struct {
		name            string
		modelName       string
		data            []any
		conflictAction  types.ConflictAction
		returningFields []string
		driverType      string
		wantSQL         string
		wantArgsCount   int
		wantErr         bool
	}{
		{
			name:      "single insert",
			modelName: "User",
			data: []any{
				map[string]any{"name": "John", "email": "john@example.com"},
			},
			driverType:    "sqlite",
			wantSQL:       "INSERT INTO users",
			wantArgsCount: 2,
		},
		{
			name:      "multiple inserts",
			modelName: "User",
			data: []any{
				map[string]any{"name": "John", "email": "john@example.com"},
				map[string]any{"name": "Jane", "email": "jane@example.com"},
			},
			driverType:    "sqlite",
			wantSQL:       "INSERT INTO users",
			wantArgsCount: 4,
		},
		{
			name:      "with returning",
			modelName: "User",
			data: []any{
				map[string]any{"name": "John"},
			},
			returningFields: []string{"id", "createdAt"},
			driverType:      "postgresql",
			wantSQL:         "INSERT INTO users (`name`) VALUES (?)",
			wantArgsCount:   1,
		},
		{
			name:           "on conflict replace",
			modelName:      "User",
			data:           []any{map[string]any{"id": 1, "name": "John"}},
			conflictAction: types.ConflictReplace,
			driverType:     "sqlite",
			wantSQL:        "INSERT INTO users OR REPLACE",
			wantArgsCount:  2,
		},
		{
			name:      "empty data array",
			modelName: "User",
			data:      []any{},
			wantErr:   true,
		},
		{
			name:      "empty map - default values",
			modelName: "User",
			data:      []any{map[string]any{}},
			driverType: "sqlite",
			wantSQL:    "INSERT INTO users DEFAULT VALUES",
			wantArgsCount: 0,
		},
		{
			name:      "empty map with returning",
			modelName: "User", 
			data:      []any{map[string]any{}},
			returningFields: []string{"id", "createdAt"},
			driverType: "postgresql",
			wantSQL:    "INSERT INTO users DEFAULT VALUES",
			wantArgsCount: 0,
		},
		{
			name:      "nil data item",
			modelName: "User",
			data:      []any{nil},
			wantErr:   true,
		},
		{
			name:      "struct data",
			modelName: "User",
			data: []any{
				struct {
					Name  string
					Email string
				}{
					Name:  "John",
					Email: "john@example.com",
				},
			},
			driverType:    "mysql",
			wantSQL:       "INSERT INTO users",
			wantArgsCount: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mockDatabase{}
			
			query := &InsertQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   tt.modelName,
					fieldMapper: mapper,
				},
				data:            tt.data,
				conflictAction:  tt.conflictAction,
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

func TestInsertQuery_Exec(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {"name": "name"},
		},
	}
	
	tests := []struct {
		name       string
		data       []any
		execResult types.Result
		execError  error
		wantErr    bool
	}{
		{
			name: "successful insert",
			data: []any{map[string]any{"name": "John"}},
			execResult: types.Result{
				RowsAffected: 1,
				LastInsertID: 123,
			},
			wantErr: false,
		},
		{
			name:      "exec error",
			data:      []any{map[string]any{"name": "John"}},
			execError: errors.New("database error"),
			wantErr:   true,
		},
		{
			name:    "build SQL error",
			data:    []any{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRaw := &insertMockRawQuery{
				execResult: tt.execResult,
				execError:  tt.execError,
			}
			
			mockDB := &insertMockDatabase{
				mockRaw: mockRaw,
			}
			
			query := &InsertQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   "User",
					fieldMapper: mapper,
				},
				data: tt.data,
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
				if result.LastInsertID != tt.execResult.LastInsertID {
					t.Errorf("Exec() LastInsertID = %d, want %d", result.LastInsertID, tt.execResult.LastInsertID)
				}
			}
		})
	}
}

func TestInsertQuery_ExecAndReturn(t *testing.T) {
	mapper := &testFieldMapper{
		mappings: map[string]map[string]string{
			"User": {
				"id":   "id",
				"name": "name",
			},
		},
	}
	
	tests := []struct {
		name            string
		data            []any
		returningFields []string
		findResult      map[string]any
		findError       error
		wantErr         bool
		errContains     string
	}{
		{
			name:            "successful exec and return",
			data:            []any{map[string]any{"name": "John"}},
			returningFields: []string{"id", "name"},
			findResult:      map[string]any{"id": 123, "name": "John"},
			wantErr:         false,
		},
		{
			name:            "no returning fields",
			data:            []any{map[string]any{"name": "John"}},
			returningFields: []string{},
			wantErr:         true,
			errContains:     "no returning fields",
		},
		{
			name:            "find error",
			data:            []any{map[string]any{"name": "John"}},
			returningFields: []string{"id"},
			findError:       errors.New("database error"),
			wantErr:         true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRaw := &insertMockRawQuery{
				findOneResult: tt.findResult,
				findOneError:  tt.findError,
			}
			
			mockDB := &insertMockDatabase{
				mockRaw: mockRaw,
			}
			
			query := &InsertQueryImpl{
				ModelQueryImpl: &ModelQueryImpl{
					database:    mockDB,
					modelName:   "User",
					fieldMapper: mapper,
				},
				data:            tt.data,
				returningFields: tt.returningFields,
			}
			
			var result map[string]any
			err := query.ExecAndReturn(context.Background(), &result)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecAndReturn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ExecAndReturn() error = %v, want to contain %v", err, tt.errContains)
				}
			}
			
			if !tt.wantErr && tt.findResult != nil {
				if fmt.Sprint(result) != fmt.Sprint(tt.findResult) {
					t.Errorf("ExecAndReturn() result = %v, want %v", result, tt.findResult)
				}
			}
		})
	}
}

// containsSQL checks if SQL contains expected pattern (case-insensitive)
func containsSQL(sql, pattern string) bool {
	return contains(strings.ToLower(sql), strings.ToLower(pattern))
}

// insertMockDatabase extends mockDatabase for insert tests
type insertMockDatabase struct {
	mockDatabase
	mockRaw *insertMockRawQuery
}

func (m *insertMockDatabase) Raw(sql string, args ...any) types.RawQuery {
	if m.mockRaw != nil {
		return m.mockRaw
	}
	return &insertMockRawQuery{}
}

// insertMockRawQuery for insert tests
type insertMockRawQuery struct {
	execResult    types.Result
	execError     error
	findOneResult map[string]any
	findOneError  error
}

func (m *insertMockRawQuery) Exec(ctx context.Context) (types.Result, error) {
	return m.execResult, m.execError
}

func (m *insertMockRawQuery) FindOne(ctx context.Context, dest any) error {
	if m.findOneError != nil {
		return m.findOneError
	}
	if m.findOneResult != nil {
		if destMap, ok := dest.(*map[string]any); ok {
			*destMap = m.findOneResult
		}
	}
	return nil
}

func (m *insertMockRawQuery) Find(ctx context.Context, dest any) error {
	return nil
}