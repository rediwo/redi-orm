package schema

import (
	"reflect"
	"testing"

	"github.com/rediwo/redi-orm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Field struct and methods
func TestField_GetColumnName(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected string
	}{
		{
			name: "field with explicit mapping",
			field: Field{
				Name: "firstName",
				Map:  "custom_first_name",
			},
			expected: "custom_first_name",
		},
		{
			name: "field without mapping - camelCase conversion",
			field: Field{
				Name: "firstName",
			},
			expected: "first_name",
		},
		{
			name: "field without mapping - simple name",
			field: Field{
				Name: "id",
			},
			expected: "id",
		},
		{
			name: "field without mapping - complex camelCase",
			field: Field{
				Name: "createdAt",
			},
			expected: "created_at",
		},
		{
			name: "field with empty mapping - uses automatic conversion",
			field: Field{
				Name: "lastName",
				Map:  "",
			},
			expected: "last_name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.field.GetColumnName()
			assert.Equal(t, test.expected, result)
		})
	}
}

// Test Schema creation and basic operations
func TestSchema_New(t *testing.T) {
	schema := New("User")
	
	assert.Equal(t, "User", schema.Name)
	assert.Equal(t, "users", schema.TableName) // Should be pluralized snake_case
	assert.Empty(t, schema.Fields)
	assert.NotNil(t, schema.Relations)
	assert.Empty(t, schema.Relations)
	assert.Empty(t, schema.Indexes)
	assert.Empty(t, schema.CompositeKey)
}

func TestSchema_WithTableName(t *testing.T) {
	schema := New("User").WithTableName("custom_users")
	
	assert.Equal(t, "User", schema.Name)
	assert.Equal(t, "custom_users", schema.TableName)
}

func TestSchema_AddField(t *testing.T) {
	schema := New("User")
	
	field1 := Field{Name: "id", Type: FieldTypeInt64, PrimaryKey: true, AutoIncrement: true}
	field2 := Field{Name: "name", Type: FieldTypeString}
	
	schema.AddField(field1).AddField(field2)
	
	assert.Len(t, schema.Fields, 2)
	assert.Equal(t, field1, schema.Fields[0])
	assert.Equal(t, field2, schema.Fields[1])
}

func TestSchema_AddRelation(t *testing.T) {
	schema := New("User")
	
	relation := Relation{
		Type:       RelationOneToMany,
		Model:      "Post",
		ForeignKey: "userId",
		References: "id",
		OnDelete:   "CASCADE",
	}
	
	schema.AddRelation("posts", relation)
	
	assert.Len(t, schema.Relations, 1)
	assert.Equal(t, relation, schema.Relations["posts"])
}

func TestSchema_AddIndex(t *testing.T) {
	schema := New("User")
	
	index := Index{
		Name:   "idx_user_email",
		Fields: []string{"email"},
		Unique: true,
	}
	
	schema.AddIndex(index)
	
	assert.Len(t, schema.Indexes, 1)
	assert.Equal(t, index, schema.Indexes[0])
}

func TestSchema_WithCompositeKey(t *testing.T) {
	schema := New("UserRole").WithCompositeKey([]string{"userId", "roleId"})
	
	assert.Equal(t, []string{"userId", "roleId"}, schema.CompositeKey)
}

// Test Schema field operations
func TestSchema_GetField(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "id", Type: FieldTypeInt64}).
		AddField(Field{Name: "name", Type: FieldTypeString})
	
	// Test existing field
	field, err := schema.GetField("name")
	require.NoError(t, err)
	assert.Equal(t, "name", field.Name)
	assert.Equal(t, FieldTypeString, field.Type)
	
	// Test non-existing field
	_, err = schema.GetField("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field nonexistent not found")
}

func TestSchema_GetPrimaryKey(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "id", Type: FieldTypeInt64, PrimaryKey: true}).
		AddField(Field{Name: "name", Type: FieldTypeString})
	
	// Test with primary key
	pk, err := schema.GetPrimaryKey()
	require.NoError(t, err)
	assert.Equal(t, "id", pk.Name)
	assert.True(t, pk.PrimaryKey)
	
	// Test without primary key
	schemaWithoutPK := New("Test")
	_, err = schemaWithoutPK.GetPrimaryKey()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no primary key found")
}

// Test Schema validation
func TestSchema_Validate(t *testing.T) {
	t.Run("valid schema with single primary key", func(t *testing.T) {
		schema := New("User").
			AddField(Field{Name: "id", Type: FieldTypeInt64, PrimaryKey: true}).
			AddField(Field{Name: "name", Type: FieldTypeString})
		
		err := schema.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("valid schema with composite primary key", func(t *testing.T) {
		schema := New("UserRole").
			AddField(Field{Name: "userId", Type: FieldTypeInt64}).
			AddField(Field{Name: "roleId", Type: FieldTypeInt64}).
			WithCompositeKey([]string{"userId", "roleId"})
		
		err := schema.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("invalid - empty name", func(t *testing.T) {
		schema := &Schema{Name: ""}
		err := schema.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema name cannot be empty")
	})
	
	t.Run("invalid - empty table name", func(t *testing.T) {
		schema := &Schema{Name: "User", TableName: ""}
		err := schema.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "table name cannot be empty")
	})
	
	t.Run("invalid - no fields", func(t *testing.T) {
		schema := New("User")
		schema.Fields = []Field{} // Empty fields
		err := schema.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema must have at least one field")
	})
	
	t.Run("invalid - multiple single primary keys", func(t *testing.T) {
		schema := New("User").
			AddField(Field{Name: "id1", Type: FieldTypeInt64, PrimaryKey: true}).
			AddField(Field{Name: "id2", Type: FieldTypeInt64, PrimaryKey: true})
		
		err := schema.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema can only have one single-field primary key")
	})
	
	t.Run("invalid - both single and composite primary keys", func(t *testing.T) {
		schema := New("User").
			AddField(Field{Name: "id", Type: FieldTypeInt64, PrimaryKey: true}).
			AddField(Field{Name: "userId", Type: FieldTypeInt64}).
			WithCompositeKey([]string{"userId"})
		
		err := schema.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema cannot have both single and composite primary keys")
	})
	
	t.Run("invalid - composite key field not found", func(t *testing.T) {
		schema := New("User").
			AddField(Field{Name: "id", Type: FieldTypeInt64}).
			WithCompositeKey([]string{"nonexistent"})
		
		err := schema.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "composite key field nonexistent not found")
	})
	
	t.Run("invalid - no primary key", func(t *testing.T) {
		schema := New("User").
			AddField(Field{Name: "name", Type: FieldTypeString})
		
		err := schema.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema must have a primary key")
	})
}

// Test field mapping operations
func TestSchema_GetFieldByColumnName(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}). // Maps to "first_name"
		AddField(Field{Name: "email", Type: FieldTypeString, Map: "user_email"}) // Custom mapping
	
	// Test automatic mapping
	field, err := schema.GetFieldByColumnName("first_name")
	require.NoError(t, err)
	assert.Equal(t, "firstName", field.Name)
	
	// Test custom mapping
	field, err = schema.GetFieldByColumnName("user_email")
	require.NoError(t, err)
	assert.Equal(t, "email", field.Name)
	
	// Test non-existing column
	_, err = schema.GetFieldByColumnName("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field with column name nonexistent not found")
}

func TestSchema_GetFieldNameByColumnName(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}). // Maps to "first_name"
		AddField(Field{Name: "email", Type: FieldTypeString, Map: "user_email"}) // Custom mapping
	
	// Test automatic mapping
	fieldName, err := schema.GetFieldNameByColumnName("first_name")
	require.NoError(t, err)
	assert.Equal(t, "firstName", fieldName)
	
	// Test custom mapping
	fieldName, err = schema.GetFieldNameByColumnName("user_email")
	require.NoError(t, err)
	assert.Equal(t, "email", fieldName)
}

func TestSchema_GetColumnNameByFieldName(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}). // Maps to "first_name"
		AddField(Field{Name: "email", Type: FieldTypeString, Map: "user_email"}) // Custom mapping
	
	// Test automatic mapping
	columnName, err := schema.GetColumnNameByFieldName("firstName")
	require.NoError(t, err)
	assert.Equal(t, "first_name", columnName)
	
	// Test custom mapping
	columnName, err = schema.GetColumnNameByFieldName("email")
	require.NoError(t, err)
	assert.Equal(t, "user_email", columnName)
}

func TestSchema_MapFieldNamesToColumns(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}). // Maps to "first_name"
		AddField(Field{Name: "lastName", Type: FieldTypeString}).  // Maps to "last_name"
		AddField(Field{Name: "email", Type: FieldTypeString, Map: "user_email"}) // Custom mapping
	
	fieldNames := []string{"firstName", "lastName", "email"}
	columnNames, err := schema.MapFieldNamesToColumns(fieldNames)
	
	require.NoError(t, err)
	expected := []string{"first_name", "last_name", "user_email"}
	assert.Equal(t, expected, columnNames)
}

func TestSchema_MapColumnNamesToFields(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}). // Maps to "first_name"
		AddField(Field{Name: "lastName", Type: FieldTypeString}).  // Maps to "last_name"
		AddField(Field{Name: "email", Type: FieldTypeString, Map: "user_email"}) // Custom mapping
	
	columnNames := []string{"first_name", "last_name", "user_email"}
	fieldNames, err := schema.MapColumnNamesToFields(columnNames)
	
	require.NoError(t, err)
	expected := []string{"firstName", "lastName", "email"}
	assert.Equal(t, expected, fieldNames)
}

func TestSchema_MapSchemaDataToColumns(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}). // Maps to "first_name"
		AddField(Field{Name: "email", Type: FieldTypeString, Map: "user_email"}) // Custom mapping
	
	schemaData := map[string]interface{}{
		"firstName": "John",
		"email":     "john@example.com",
	}
	
	columnData, err := schema.MapSchemaDataToColumns(schemaData)
	require.NoError(t, err)
	
	expected := map[string]interface{}{
		"first_name": "John",
		"user_email": "john@example.com",
	}
	assert.Equal(t, expected, columnData)
}

func TestSchema_MapColumnDataToSchema(t *testing.T) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}). // Maps to "first_name"
		AddField(Field{Name: "email", Type: FieldTypeString, Map: "user_email"}) // Custom mapping
	
	columnData := map[string]interface{}{
		"first_name": "John",
		"user_email": "john@example.com",
		"unknown_col": "unknown_value", // Should be preserved as-is
	}
	
	schemaData, err := schema.MapColumnDataToSchema(columnData)
	require.NoError(t, err)
	
	expected := map[string]interface{}{
		"firstName":   "John",
		"email":       "john@example.com",
		"unknown_col": "unknown_value", // Raw columns are preserved
	}
	assert.Equal(t, expected, schemaData)
}

// Test relation operations
func TestSchema_HasRelation(t *testing.T) {
	schema := New("User")
	
	// Initially no relations
	assert.False(t, schema.HasRelation("posts"))
	
	// Add relation
	relation := Relation{Type: RelationOneToMany, Model: "Post"}
	schema.AddRelation("posts", relation)
	
	assert.True(t, schema.HasRelation("posts"))
	assert.False(t, schema.HasRelation("nonexistent"))
}

func TestSchema_GetRelation(t *testing.T) {
	schema := New("User")
	relation := Relation{
		Type:       RelationOneToMany,
		Model:      "Post",
		ForeignKey: "userId",
		References: "id",
	}
	schema.AddRelation("posts", relation)
	
	// Test existing relation
	result, err := schema.GetRelation("posts")
	require.NoError(t, err)
	assert.Equal(t, relation, result)
	
	// Test non-existing relation
	_, err = schema.GetRelation("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "relation nonexistent not found")
}

// Test utility functions
func TestModelNameToTableName(t *testing.T) {
	tests := []struct {
		modelName string
		expected  string
	}{
		{"User", "users"},
		{"BlogPost", "blog_posts"},
		{"Category", "categories"},
		{"UserProfile", "user_profiles"},
		{"XMLDocument", "x_m_l_documents"},
		{"Company", "companies"},
	}
	
	for _, test := range tests {
		t.Run(test.modelName, func(t *testing.T) {
			result := ModelNameToTableName(test.modelName)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestFieldTypeFromGo(t *testing.T) {
	tests := []struct {
		goType   reflect.Type
		expected FieldType
	}{
		{reflect.TypeOf(""), FieldTypeString},
		{reflect.TypeOf(0), FieldTypeInt},
		{reflect.TypeOf(int32(0)), FieldTypeInt},
		{reflect.TypeOf(int64(0)), FieldTypeInt64},
		{reflect.TypeOf(float32(0)), FieldTypeFloat},
		{reflect.TypeOf(float64(0)), FieldTypeFloat},
		{reflect.TypeOf(true), FieldTypeBool},
		{reflect.TypeOf([]byte{}), FieldTypeString}, // Default case
	}
	
	for _, test := range tests {
		t.Run(test.goType.String(), func(t *testing.T) {
			result := FieldTypeFromGo(test.goType)
			assert.Equal(t, test.expected, result)
		})
	}
}

// Test integration with utils package
func TestSchemaUtilsIntegration(t *testing.T) {
	// Test that schema correctly uses utils functions
	t.Run("ToSnakeCase integration", func(t *testing.T) {
		field := Field{Name: "firstName"}
		columnName := field.GetColumnName()
		
		// Should use utils.ToSnakeCase internally
		expected := utils.ToSnakeCase("firstName")
		assert.Equal(t, expected, columnName)
		assert.Equal(t, "first_name", columnName)
	})
	
	t.Run("Pluralize integration", func(t *testing.T) {
		tableName := ModelNameToTableName("User")
		
		// Should use utils.Pluralize internally
		expected := utils.Pluralize(utils.ToSnakeCase("User"))
		assert.Equal(t, expected, tableName)
		assert.Equal(t, "users", tableName)
	})
}

// Test NewField builder functions
func TestNewField(t *testing.T) {
	// Test NewField builder pattern
	fieldBuilder := NewField("id")
	assert.NotNil(t, fieldBuilder)
	
	// Test builder methods and final Build()
	field := NewField("id").
		Int64().
		PrimaryKey().
		AutoIncrement().
		Build()
	
	assert.Equal(t, "id", field.Name)
	assert.Equal(t, FieldTypeInt64, field.Type)
	assert.True(t, field.PrimaryKey)
	assert.True(t, field.AutoIncrement)
	assert.False(t, field.Nullable) // PrimaryKey sets Nullable to false
}

func TestFieldBuilder_ChainMethods(t *testing.T) {
	// Test various field builder combinations
	t.Run("string field with map", func(t *testing.T) {
		field := NewField("firstName").
			String().
			Map("first_name").
			Build()
		
		assert.Equal(t, "firstName", field.Name)
		assert.Equal(t, FieldTypeString, field.Type)
		assert.Equal(t, "first_name", field.Map)
	})
	
	t.Run("nullable field with default", func(t *testing.T) {
		field := NewField("status").
			String().
			Nullable().
			Default("active").
			Build()
		
		assert.Equal(t, "status", field.Name)
		assert.Equal(t, FieldTypeString, field.Type)
		assert.True(t, field.Nullable)
		assert.Equal(t, "active", field.Default)
	})
	
	t.Run("unique indexed field", func(t *testing.T) {
		field := NewField("email").
			String().
			Unique().
			Index().
			Build()
		
		assert.Equal(t, "email", field.Name)
		assert.Equal(t, FieldTypeString, field.Type)
		assert.True(t, field.Unique)
		assert.True(t, field.Index)
	})
}

// Benchmark tests
func BenchmarkFieldGetColumnName(b *testing.B) {
	field := Field{Name: "firstName"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = field.GetColumnName()
	}
}

func BenchmarkModelNameToTableName(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ModelNameToTableName("UserProfile")
	}
}

func BenchmarkSchemaMapFieldNamesToColumns(b *testing.B) {
	schema := New("User").
		AddField(Field{Name: "firstName", Type: FieldTypeString}).
		AddField(Field{Name: "lastName", Type: FieldTypeString}).
		AddField(Field{Name: "email", Type: FieldTypeString}).
		AddField(Field{Name: "createdAt", Type: FieldTypeDateTime})
	
	fieldNames := []string{"firstName", "lastName", "email", "createdAt"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = schema.MapFieldNamesToColumns(fieldNames)
	}
}