package mongodb

import (
	"testing"

	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/sql"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// mockFieldMapper for testing
type mockFieldMapper struct{}

func (m *mockFieldMapper) ModelToTable(modelName string) (string, error) {
	return modelName, nil
}

func (m *mockFieldMapper) TableToModel(tableName string) (string, error) {
	return tableName, nil
}

func (m *mockFieldMapper) SchemaToColumn(modelName, fieldName string) (string, error) {
	if fieldName == "id" {
		return "_id", nil
	}
	return fieldName, nil
}

func (m *mockFieldMapper) ColumnToSchema(modelName, columnName string) (string, error) {
	if columnName == "_id" {
		return "id", nil
	}
	return columnName, nil
}

func (m *mockFieldMapper) MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range data {
		if k == "id" {
			result["_id"] = v
		} else {
			result[k] = v
		}
	}
	return result, nil
}

func (m *mockFieldMapper) MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range data {
		if k == "_id" {
			result["id"] = v
		} else {
			result[k] = v
		}
	}
	return result, nil
}

func (m *mockFieldMapper) SchemaFieldsToColumns(modelName string, fieldNames []string) ([]string, error) {
	columns := make([]string, len(fieldNames))
	for i, fieldName := range fieldNames {
		if fieldName == "id" {
			columns[i] = "_id"
		} else {
			columns[i] = fieldName
		}
	}
	return columns, nil
}

func (m *mockFieldMapper) ColumnFieldsToSchema(modelName string, columnNames []string) ([]string, error) {
	fields := make([]string, len(columnNames))
	for i, columnName := range columnNames {
		if columnName == "_id" {
			fields[i] = "id"
		} else {
			fields[i] = columnName
		}
	}
	return fields, nil
}

func TestMongoDBSQLTranslator_TranslateSelect(t *testing.T) {
	// Create a mock MongoDB instance for testing
	baseDriver := base.NewDriver("test://", types.DriverMongoDB)
	baseDriver.FieldMapper = &mockFieldMapper{}
	db := &MongoDB{
		Driver: baseDriver,
	}
	translator := NewMongoDBSQLTranslator(db)

	tests := []struct {
		name     string
		sql      string
		expected *MongoDBCommand
	}{
		{
			name: "Simple SELECT *",
			sql:  "SELECT * FROM users",
			expected: &MongoDBCommand{
				Operation:  "find",
				Collection: "users",
			},
		},
		{
			name: "SELECT with specific fields",
			sql:  "SELECT name, age FROM users",
			expected: &MongoDBCommand{
				Operation:  "aggregate",
				Collection: "users",
				Pipeline: []bson.M{
					{"$project": bson.M{
						"name": "$name",
						"age":  "$age",
					}},
				},
			},
		},
		{
			name: "SELECT with WHERE clause",
			sql:  "SELECT * FROM users WHERE age > 25",
			expected: &MongoDBCommand{
				Operation:  "aggregate",
				Collection: "users",
				Pipeline: []bson.M{
					{"$match": bson.M{
						"age": bson.M{"$gt": int64(25)},
					}},
				},
			},
		},
		{
			name: "SELECT with ORDER BY",
			sql:  "SELECT * FROM users ORDER BY name ASC",
			expected: &MongoDBCommand{
				Operation:  "aggregate",
				Collection: "users",
				Pipeline: []bson.M{
					{"$sort": bson.M{
						"name": 1,
					}},
				},
			},
		},
		{
			name: "SELECT with LIMIT",
			sql:  "SELECT * FROM users LIMIT 10",
			expected: &MongoDBCommand{
				Operation:  "aggregate",
				Collection: "users",
				Pipeline: []bson.M{
					{"$limit": 10},
				},
			},
		},
		{
			name: "Complex SELECT",
			sql:  "SELECT name, age FROM users WHERE age > 25 ORDER BY name DESC LIMIT 5",
			expected: &MongoDBCommand{
				Operation:  "aggregate",
				Collection: "users",
				Pipeline: []bson.M{
					{"$match": bson.M{
						"age": bson.M{"$gt": int64(25)},
					}},
					{"$sort": bson.M{
						"name": -1,
					}},
					{"$limit": 5},
					{"$project": bson.M{
						"name": "$name",
						"age":  "$age",
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sql.NewParser(tt.sql)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			selectStmt, ok := stmt.(*sql.SelectStatement)
			require.True(t, ok)

			result, err := translator.translateSelect(selectStmt)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Operation, result.Operation)
			assert.Equal(t, tt.expected.Collection, result.Collection)

			if tt.expected.Pipeline != nil {
				assert.Equal(t, tt.expected.Pipeline, result.Pipeline)
			}
		})
	}
}

func TestMongoDBSQLTranslator_TranslateInsert(t *testing.T) {
	baseDriver := base.NewDriver("test://", types.DriverMongoDB)
	baseDriver.FieldMapper = &mockFieldMapper{}
	db := &MongoDB{
		Driver: baseDriver,
	}
	translator := NewMongoDBSQLTranslator(db)

	tests := []struct {
		name     string
		sql      string
		expected *MongoDBCommand
	}{
		{
			name: "Simple INSERT",
			sql:  "INSERT INTO users (name, age) VALUES ('Alice', 25)",
			expected: &MongoDBCommand{
				Operation:  "insert",
				Collection: "users",
				Documents: []any{
					bson.M{
						"name": "Alice",
						"age":  int64(25),
					},
				},
			},
		},
		{
			name: "INSERT multiple rows",
			sql:  "INSERT INTO users (name, age) VALUES ('Alice', 25), ('Bob', 30)",
			expected: &MongoDBCommand{
				Operation:  "insert",
				Collection: "users",
				Documents: []any{
					bson.M{
						"name": "Alice",
						"age":  int64(25),
					},
					bson.M{
						"name": "Bob",
						"age":  int64(30),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sql.NewParser(tt.sql)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			insertStmt, ok := stmt.(*sql.InsertStatement)
			require.True(t, ok)

			result, err := translator.translateInsert(insertStmt)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Operation, result.Operation)
			assert.Equal(t, tt.expected.Collection, result.Collection)
			assert.Equal(t, tt.expected.Documents, result.Documents)
		})
	}
}

func TestMongoDBSQLTranslator_TranslateUpdate(t *testing.T) {
	baseDriver := base.NewDriver("test://", types.DriverMongoDB)
	baseDriver.FieldMapper = &mockFieldMapper{}
	db := &MongoDB{
		Driver: baseDriver,
	}
	translator := NewMongoDBSQLTranslator(db)

	tests := []struct {
		name     string
		sql      string
		expected *MongoDBCommand
	}{
		{
			name: "UPDATE without WHERE",
			sql:  "UPDATE users SET name = 'Charlie'",
			expected: &MongoDBCommand{
				Operation:  "update",
				Collection: "users",
				Filter:     bson.M{},
				Update: bson.M{
					"$set": bson.M{
						"name": "Charlie",
					},
				},
			},
		},
		{
			name: "UPDATE with WHERE",
			sql:  "UPDATE users SET name = 'Charlie', age = 35 WHERE id = 1",
			expected: &MongoDBCommand{
				Operation:  "update",
				Collection: "users",
				Filter: bson.M{
					"_id": int64(1),
				},
				Update: bson.M{
					"$set": bson.M{
						"name": "Charlie",
						"age":  int64(35),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sql.NewParser(tt.sql)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			updateStmt, ok := stmt.(*sql.UpdateStatement)
			require.True(t, ok)

			result, err := translator.translateUpdate(updateStmt)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Operation, result.Operation)
			assert.Equal(t, tt.expected.Collection, result.Collection)
			assert.Equal(t, tt.expected.Filter, result.Filter)
			assert.Equal(t, tt.expected.Update, result.Update)
		})
	}
}

func TestMongoDBSQLTranslator_TranslateDelete(t *testing.T) {
	baseDriver := base.NewDriver("test://", types.DriverMongoDB)
	baseDriver.FieldMapper = &mockFieldMapper{}
	db := &MongoDB{
		Driver: baseDriver,
	}
	translator := NewMongoDBSQLTranslator(db)

	tests := []struct {
		name     string
		sql      string
		expected *MongoDBCommand
	}{
		{
			name: "DELETE without WHERE",
			sql:  "DELETE FROM users",
			expected: &MongoDBCommand{
				Operation:  "delete",
				Collection: "users",
				Filter:     bson.M{},
			},
		},
		{
			name: "DELETE with WHERE",
			sql:  "DELETE FROM users WHERE age < 18",
			expected: &MongoDBCommand{
				Operation:  "delete",
				Collection: "users",
				Filter: bson.M{
					"age": bson.M{"$lt": int64(18)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sql.NewParser(tt.sql)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			deleteStmt, ok := stmt.(*sql.DeleteStatement)
			require.True(t, ok)

			result, err := translator.translateDelete(deleteStmt)
			require.NoError(t, err)

			assert.Equal(t, tt.expected.Operation, result.Operation)
			assert.Equal(t, tt.expected.Collection, result.Collection)
			assert.Equal(t, tt.expected.Filter, result.Filter)
		})
	}
}

func TestMongoDBSQLTranslator_TranslateWhereClause(t *testing.T) {
	baseDriver := base.NewDriver("test://", types.DriverMongoDB)
	baseDriver.FieldMapper = &mockFieldMapper{}
	db := &MongoDB{
		Driver: baseDriver,
	}
	translator := NewMongoDBSQLTranslator(db)

	tests := []struct {
		name     string
		sql      string
		expected bson.M
	}{
		{
			name: "Simple equality",
			sql:  "SELECT * FROM users WHERE name = 'Alice'",
			expected: bson.M{
				"name": "Alice",
			},
		},
		{
			name: "Greater than",
			sql:  "SELECT * FROM users WHERE age > 25",
			expected: bson.M{
				"age": bson.M{"$gt": int64(25)},
			},
		},
		{
			name: "LIKE operator",
			sql:  "SELECT * FROM users WHERE name LIKE 'A%'",
			expected: bson.M{
				"name": bson.M{
					"$regex":   "^A.*$",
					"$options": "i",
				},
			},
		},
		{
			name: "IN operator",
			sql:  "SELECT * FROM users WHERE id IN (1, 2, 3)",
			expected: bson.M{
				"_id": bson.M{
					"$in": []any{int64(1), int64(2), int64(3)},
				},
			},
		},
		{
			name: "AND condition",
			sql:  "SELECT * FROM users WHERE age > 25 AND name = 'Alice'",
			expected: bson.M{
				"$and": []bson.M{
					{"age": bson.M{"$gt": int64(25)}},
					{"name": "Alice"},
				},
			},
		},
		{
			name: "OR condition",
			sql:  "SELECT * FROM users WHERE age > 25 OR name = 'Alice'",
			expected: bson.M{
				"$or": []bson.M{
					{"age": bson.M{"$gt": int64(25)}},
					{"name": "Alice"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := sql.NewParser(tt.sql)
			stmt, err := parser.Parse()
			require.NoError(t, err)

			selectStmt, ok := stmt.(*sql.SelectStatement)
			require.True(t, ok)
			require.NotNil(t, selectStmt.Where)

			result, err := translator.translateWhereToMatch(selectStmt.Where)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertLikeToRegex(t *testing.T) {
	baseDriver := base.NewDriver("test://", types.DriverMongoDB)
	baseDriver.FieldMapper = &mockFieldMapper{}
	db := &MongoDB{
		Driver: baseDriver,
	}
	translator := NewMongoDBSQLTranslator(db)

	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "Prefix match",
			pattern:  "A%",
			expected: "^A.*$",
		},
		{
			name:     "Suffix match",
			pattern:  "%son",
			expected: "^.*son$",
		},
		{
			name:     "Contains match",
			pattern:  "%abc%",
			expected: "^.*abc.*$",
		},
		{
			name:     "Single character wildcard",
			pattern:  "A_e",
			expected: "^A.e$",
		},
		{
			name:     "Mixed wildcards",
			pattern:  "A_e%",
			expected: "^A.e.*$",
		},
		{
			name:     "Escape special characters",
			pattern:  "A.B*C?",
			expected: "^A\\.B\\*C\\?$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.convertLikeToRegex(tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}
