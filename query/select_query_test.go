package query

import (
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing are in test_helpers_test.go

type mockFieldMapper struct{}

func (m *mockFieldMapper) SchemaToColumn(modelName, fieldName string) (string, error) {
	// Simple camelCase to snake_case conversion
	if fieldName == "userId" {
		return "user_id", nil
	}
	if fieldName == "createdAt" {
		return "created_at", nil
	}
	return strings.ToLower(fieldName), nil
}

func (m *mockFieldMapper) ColumnToSchema(modelName, columnName string) (string, error) {
	return "", nil
}
func (m *mockFieldMapper) SchemaFieldsToColumns(modelName string, fieldNames []string) ([]string, error) {
	return nil, nil
}
func (m *mockFieldMapper) ColumnFieldsToSchema(modelName string, columnNames []string) ([]string, error) {
	return nil, nil
}
func (m *mockFieldMapper) MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error) {
	return nil, nil
}
func (m *mockFieldMapper) MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error) {
	return nil, nil
}
func (m *mockFieldMapper) ModelToTable(modelName string) (string, error) {
	return strings.ToLower(modelName) + "s", nil
}

func TestSelectQuery_Include(t *testing.T) {
	// Setup mock database with schemas
	db := &mockDatabase{
		schemas: make(map[string]*schema.Schema),
	}

	// Create User schema
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "email", Type: schema.FieldTypeString})

	// Create Post schema
	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "title", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt})

	// Add relations
	userSchema.AddRelation("posts", schema.Relation{
		Type:       schema.RelationOneToMany,
		Model:      "Post",
		ForeignKey: "userId",
		References: "id",
	})

	postSchema.AddRelation("user", schema.Relation{
		Type:       schema.RelationManyToOne,
		Model:      "User",
		ForeignKey: "userId",
		References: "id",
	})

	db.RegisterSchema("User", userSchema)
	db.RegisterSchema("Post", postSchema)

	// Create query
	baseQuery := NewModelQuery("User", db, &mockFieldMapper{})
	selectQuery := NewSelectQuery(baseQuery, []string{})

	// Add include
	includedQuery := selectQuery.Include("posts")

	// Build SQL
	sql, args, err := includedQuery.BuildSQL()
	require.NoError(t, err)
	assert.Empty(t, args)

	// Verify SQL contains join
	assert.Contains(t, sql, "LEFT JOIN")
	assert.Contains(t, sql, "posts")

	// Verify columns are aliased to avoid ambiguity
	assert.Contains(t, sql, "u.id AS u_id")
	assert.Contains(t, sql, "u.name AS u_name")
	assert.Contains(t, sql, "u.email AS u_email")
	assert.Contains(t, sql, "p.id AS p_id")
	assert.Contains(t, sql, "p.title AS p_title")
	assert.Contains(t, sql, "p.user_id AS p_user_id")
}

func TestSelectQuery_IncludeWithWhere(t *testing.T) {
	// Setup mock database with schemas
	db := &mockDatabase{
		schemas: make(map[string]*schema.Schema),
	}

	// Create schemas
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString})

	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "title", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt})

	// Add relations
	userSchema.AddRelation("posts", schema.Relation{
		Type:       schema.RelationOneToMany,
		Model:      "Post",
		ForeignKey: "userId",
		References: "id",
	})

	db.RegisterSchema("User", userSchema)
	db.RegisterSchema("Post", postSchema)

	// Create query with where condition
	baseQuery := NewModelQuery("User", db, &mockFieldMapper{})
	selectQuery := NewSelectQuery(baseQuery, []string{})

	// Add condition and include
	condition := types.NewFieldCondition("User", "id").Equals(1)
	query := selectQuery.WhereCondition(condition).Include("posts")

	// Build SQL
	sql, args, err := query.BuildSQL()
	require.NoError(t, err)
	assert.Len(t, args, 1)
	assert.Equal(t, 1, args[0])

	// Verify WHERE clause has table alias
	assert.Contains(t, sql, "WHERE")
	assert.Contains(t, sql, "`u`.`id` = ?")

	// Verify join is present
	assert.Contains(t, sql, "LEFT JOIN")
}
