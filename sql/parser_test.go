package sql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "SELECT statement",
			input:    "SELECT * FROM users",
			expected: true,
		},
		{
			name:     "INSERT statement",
			input:    "INSERT INTO users (name) VALUES ('Alice')",
			expected: true,
		},
		{
			name:     "UPDATE statement",
			input:    "UPDATE users SET name = 'Bob'",
			expected: true,
		},
		{
			name:     "DELETE statement",
			input:    "DELETE FROM users WHERE id = 1",
			expected: true,
		},
		{
			name:     "JSON command",
			input:    `{"operation": "find", "collection": "users"}`,
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Random text",
			input:    "hello world",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectSQL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSelectStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *SelectStatement
	}{
		{
			name:  "Simple SELECT *",
			input: "SELECT * FROM users",
			expected: &SelectStatement{
				Fields: []SelectField{{Expression: "*"}},
				From:   TableRef{Table: "users"},
			},
		},
		{
			name:  "SELECT with fields",
			input: "SELECT name, age FROM users",
			expected: &SelectStatement{
				Fields: []SelectField{
					{Expression: "name"},
					{Expression: "age"},
				},
				From: TableRef{Table: "users"},
			},
		},
		{
			name:  "SELECT with WHERE",
			input: "SELECT * FROM users WHERE age > 25",
			expected: &SelectStatement{
				Fields: []SelectField{{Expression: "*"}},
				From:   TableRef{Table: "users"},
				Where: &WhereClause{
					Condition: &Condition{
						Field:    "age",
						Operator: ">",
						Value:    int64(25),
					},
				},
			},
		},
		{
			name:  "SELECT with ORDER BY",
			input: "SELECT * FROM users ORDER BY name ASC",
			expected: &SelectStatement{
				Fields: []SelectField{{Expression: "*"}},
				From:   TableRef{Table: "users"},
				OrderBy: []*OrderByClause{
					{Field: "name", Direction: OrderDirectionAsc},
				},
			},
		},
		{
			name:  "SELECT with LIMIT",
			input: "SELECT * FROM users LIMIT 10",
			expected: &SelectStatement{
				Fields: []SelectField{{Expression: "*"}},
				From:   TableRef{Table: "users"},
				Limit:  intPtr(10),
			},
		},
		{
			name:  "Complex SELECT",
			input: "SELECT name, age FROM users WHERE age > 25 AND name LIKE 'A%' ORDER BY name DESC LIMIT 5 OFFSET 10",
			expected: &SelectStatement{
				Fields: []SelectField{
					{Expression: "name"},
					{Expression: "age"},
				},
				From: TableRef{Table: "users"},
				Where: &WhereClause{
					Operator: "AND",
					Left: &WhereClause{
						Condition: &Condition{
							Field:    "age",
							Operator: ">",
							Value:    int64(25),
						},
					},
					Right: &WhereClause{
						Condition: &Condition{
							Field:    "name",
							Operator: "LIKE",
							Value:    "A%",
						},
					},
				},
				OrderBy: []*OrderByClause{
					{Field: "name", Direction: OrderDirectionDesc},
				},
				Limit:  intPtr(5),
				Offset: intPtr(10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			selectStmt, ok := stmt.(*SelectStatement)
			require.True(t, ok)

			assert.Equal(t, tt.expected.Fields, selectStmt.Fields)
			assert.Equal(t, tt.expected.From, selectStmt.From)

			if tt.expected.Where != nil {
				require.NotNil(t, selectStmt.Where)
				assertWhereClauseEqual(t, tt.expected.Where, selectStmt.Where)
			} else {
				assert.Nil(t, selectStmt.Where)
			}

			assert.Equal(t, tt.expected.OrderBy, selectStmt.OrderBy)
			assert.Equal(t, tt.expected.Limit, selectStmt.Limit)
			assert.Equal(t, tt.expected.Offset, selectStmt.Offset)
		})
	}
}

func TestParseInsertStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *InsertStatement
	}{
		{
			name:  "INSERT with field list",
			input: "INSERT INTO users (name, age) VALUES ('Alice', 25)",
			expected: &InsertStatement{
				Table:  "users",
				Fields: []string{"name", "age"},
				Values: [][]any{{"Alice", int64(25)}},
			},
		},
		{
			name:  "INSERT without field list",
			input: "INSERT INTO users VALUES ('Bob', 30)",
			expected: &InsertStatement{
				Table:  "users",
				Fields: nil,
				Values: [][]any{{"Bob", int64(30)}},
			},
		},
		{
			name:  "INSERT multiple rows",
			input: "INSERT INTO users (name, age) VALUES ('Alice', 25), ('Bob', 30)",
			expected: &InsertStatement{
				Table:  "users",
				Fields: []string{"name", "age"},
				Values: [][]any{
					{"Alice", int64(25)},
					{"Bob", int64(30)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			insertStmt, ok := stmt.(*InsertStatement)
			require.True(t, ok)

			assert.Equal(t, tt.expected, insertStmt)
		})
	}
}

func TestParseUpdateStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *UpdateStatement
	}{
		{
			name:  "UPDATE without WHERE",
			input: "UPDATE users SET name = 'Charlie'",
			expected: &UpdateStatement{
				Table: "users",
				Set:   map[string]any{"name": "Charlie"},
			},
		},
		{
			name:  "UPDATE with WHERE",
			input: "UPDATE users SET name = 'Charlie', age = 35 WHERE id = 1",
			expected: &UpdateStatement{
				Table: "users",
				Set:   map[string]any{"name": "Charlie", "age": int64(35)},
				Where: &WhereClause{
					Condition: &Condition{
						Field:    "id",
						Operator: "=",
						Value:    int64(1),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			updateStmt, ok := stmt.(*UpdateStatement)
			require.True(t, ok)

			assert.Equal(t, tt.expected.Table, updateStmt.Table)
			assert.Equal(t, tt.expected.Set, updateStmt.Set)

			if tt.expected.Where != nil {
				require.NotNil(t, updateStmt.Where)
				assertWhereClauseEqual(t, tt.expected.Where, updateStmt.Where)
			} else {
				assert.Nil(t, updateStmt.Where)
			}
		})
	}
}

func TestParseDeleteStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *DeleteStatement
	}{
		{
			name:  "DELETE without WHERE",
			input: "DELETE FROM users",
			expected: &DeleteStatement{
				Table: "users",
			},
		},
		{
			name:  "DELETE with WHERE",
			input: "DELETE FROM users WHERE age < 18",
			expected: &DeleteStatement{
				Table: "users",
				Where: &WhereClause{
					Condition: &Condition{
						Field:    "age",
						Operator: "<",
						Value:    int64(18),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			deleteStmt, ok := stmt.(*DeleteStatement)
			require.True(t, ok)

			assert.Equal(t, tt.expected.Table, deleteStmt.Table)

			if tt.expected.Where != nil {
				require.NotNil(t, deleteStmt.Where)
				assertWhereClauseEqual(t, tt.expected.Where, deleteStmt.Where)
			} else {
				assert.Nil(t, deleteStmt.Where)
			}
		})
	}
}

func TestParseComplexWhereClause(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "OR condition",
			input: "SELECT * FROM users WHERE age > 25 OR name = 'Alice'",
		},
		{
			name:  "AND condition",
			input: "SELECT * FROM users WHERE age > 25 AND active = true",
		},
		{
			name:  "NOT condition",
			input: "SELECT * FROM users WHERE NOT age < 18",
		},
		{
			name:  "Parentheses",
			input: "SELECT * FROM users WHERE (age > 25 OR age < 18) AND active = true",
		},
		{
			name:  "IN condition",
			input: "SELECT * FROM users WHERE id IN (1, 2, 3)",
		},
		{
			name:  "IS NULL condition",
			input: "SELECT * FROM users WHERE email IS NULL",
		},
		{
			name:  "IS NOT NULL condition",
			input: "SELECT * FROM users WHERE email IS NOT NULL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			selectStmt, ok := stmt.(*SelectStatement)
			require.True(t, ok)
			require.NotNil(t, selectStmt.Where)
		})
	}
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func assertWhereClauseEqual(t *testing.T, expected, actual *WhereClause) {
	if expected.Condition != nil {
		require.NotNil(t, actual.Condition)
		assert.Equal(t, expected.Condition, actual.Condition)
	} else {
		assert.Nil(t, actual.Condition)
	}

	assert.Equal(t, expected.Operator, actual.Operator)

	if expected.Left != nil {
		require.NotNil(t, actual.Left)
		assertWhereClauseEqual(t, expected.Left, actual.Left)
	} else {
		assert.Nil(t, actual.Left)
	}

	if expected.Right != nil {
		require.NotNil(t, actual.Right)
		assertWhereClauseEqual(t, expected.Right, actual.Right)
	} else {
		assert.Nil(t, actual.Right)
	}
}
