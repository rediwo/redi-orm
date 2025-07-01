package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func TestSchemaRegistration(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create test schema
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "name",
			Type:     schema.FieldTypeString,
			Nullable: false,
		})

	// Test schema registration
	err := db.RegisterSchema("User", userSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Verify schema is registered
	schemas := db.GetRegisteredSchemas()
	if len(schemas) != 1 {
		t.Errorf("Expected 1 registered schema, got %d", len(schemas))
	}

	if _, exists := schemas["User"]; !exists {
		t.Error("User schema not found in registered schemas")
	}
}

func TestModelNameToTableNameConversion(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create and register schema
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "name",
			Type:     schema.FieldTypeString,
			Nullable: false,
		})

	// Create table first
	db.CreateTable(userSchema)
	db.RegisterSchema("User", userSchema)

	// Test Insert with model name
	id, err := db.Insert("User", map[string]interface{}{
		"name": "John Doe",
	})

	if err != nil {
		t.Fatalf("Failed to insert using model name: %v", err)
	}

	// Test FindByID with model name
	user, err := db.FindByID("User", id)
	if err != nil {
		t.Fatalf("Failed to find by ID using model name: %v", err)
	}

	if user["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", user["name"])
	}

	// Test that raw operations work with actual table name
	rawUser, err := db.RawFindByID("users", id)
	if err != nil {
		t.Fatalf("Failed to raw find by ID: %v", err)
	}

	if rawUser["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe' in raw result, got %v", rawUser["name"])
	}
}

func TestFieldNameToColumnNameConversion(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create schema with field mappings
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "fullName",
			Type:     schema.FieldTypeString,
			Nullable: false,
			Map:      "full_name", // Map field to column
		}).
		AddField(schema.Field{
			Name:     "emailAddress",
			Type:     schema.FieldTypeString,
			Unique:   true,
			Nullable: false,
			Map:      "email", // Map field to column
		}).
		AddField(schema.Field{
			Name:    "userAge",
			Type:    schema.FieldTypeInt,
			Default: 0,
			Map:     "age", // Map field to column
		})

	// Create table and register schema
	db.CreateTable(userSchema)
	db.RegisterSchema("User", userSchema)

	// Test Insert with field names
	id, err := db.Insert("User", map[string]interface{}{
		"fullName":     "John Doe",
		"emailAddress": "john@example.com",
		"userAge":      30,
	})

	if err != nil {
		t.Fatalf("Failed to insert with field names: %v", err)
	}

	// Test FindByID returns field names
	user, err := db.FindByID("User", id)
	if err != nil {
		t.Fatalf("Failed to find by ID: %v", err)
	}

	// Verify field names are returned, not column names
	if _, hasFullName := user["fullName"]; !hasFullName {
		t.Error("Expected 'fullName' field in result")
	}
	if _, hasEmail := user["emailAddress"]; !hasEmail {
		t.Error("Expected 'emailAddress' field in result")
	}
	if _, hasAge := user["userAge"]; !hasAge {
		t.Error("Expected 'userAge' field in result")
	}

	// Verify values
	if user["fullName"] != "John Doe" {
		t.Errorf("Expected fullName 'John Doe', got %v", user["fullName"])
	}
	if user["emailAddress"] != "john@example.com" {
		t.Errorf("Expected emailAddress 'john@example.com', got %v", user["emailAddress"])
	}
	if user["userAge"] != int64(30) {
		t.Errorf("Expected userAge 30, got %v", user["userAge"])
	}

	// Test raw operations use column names
	rawUser, err := db.RawFindByID("users", id)
	if err != nil {
		t.Fatalf("Failed to raw find by ID: %v", err)
	}

	// Verify column names are returned in raw operations
	if _, hasFullName := rawUser["full_name"]; !hasFullName {
		t.Error("Expected 'full_name' column in raw result")
	}
	if _, hasEmail := rawUser["email"]; !hasEmail {
		t.Error("Expected 'email' column in raw result")
	}
	if _, hasAge := rawUser["age"]; !hasAge {
		t.Error("Expected 'age' column in raw result")
	}
}

func TestFindWithFieldNameConditions(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create schema with field mappings
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "fullName",
			Type:     schema.FieldTypeString,
			Nullable: false,
			Map:      "full_name",
		}).
		AddField(schema.Field{
			Name:    "userAge",
			Type:    schema.FieldTypeInt,
			Default: 0,
			Map:     "age",
		})

	db.CreateTable(userSchema)
	db.RegisterSchema("User", userSchema)

	// Insert test data
	testUsers := []map[string]interface{}{
		{"fullName": "John Doe", "userAge": 30},
		{"fullName": "Jane Doe", "userAge": 25},
		{"fullName": "Bob Smith", "userAge": 30},
	}

	for _, userData := range testUsers {
		_, err := db.Insert("User", userData)
		if err != nil {
			t.Fatalf("Failed to insert test user: %v", err)
		}
	}

	// Test Find with field name conditions
	users, err := db.Find("User", map[string]interface{}{
		"userAge": 30, // Using field name, not column name
	}, 10, 0)

	if err != nil {
		t.Fatalf("Failed to find users: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users with age 30, got %d", len(users))
	}

	// Verify results have field names
	for _, user := range users {
		if _, hasUserAge := user["userAge"]; !hasUserAge {
			t.Error("Expected 'userAge' field in result")
		}
		if user["userAge"] != int64(30) {
			t.Errorf("Expected userAge 30, got %v", user["userAge"])
		}
	}
}

func TestUpdateWithFieldNames(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create schema with field mappings
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "fullName",
			Type:     schema.FieldTypeString,
			Nullable: false,
			Map:      "full_name",
		}).
		AddField(schema.Field{
			Name:    "userAge",
			Type:    schema.FieldTypeInt,
			Default: 0,
			Map:     "age",
		})

	db.CreateTable(userSchema)
	db.RegisterSchema("User", userSchema)

	// Insert test data
	id, err := db.Insert("User", map[string]interface{}{
		"fullName": "John Doe",
		"userAge":  30,
	})

	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	// Update using field names
	err = db.Update("User", id, map[string]interface{}{
		"fullName": "John Updated",
		"userAge":  31,
	})

	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
	user, err := db.FindByID("User", id)
	if err != nil {
		t.Fatalf("Failed to find updated user: %v", err)
	}

	if user["fullName"] != "John Updated" {
		t.Errorf("Expected fullName 'John Updated', got %v", user["fullName"])
	}
	if user["userAge"] != int64(31) {
		t.Errorf("Expected userAge 31, got %v", user["userAge"])
	}
}

func TestQueryBuilderWithModelName(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create schema
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "name",
			Type:     schema.FieldTypeString,
			Nullable: false,
		}).
		AddField(schema.Field{
			Name: "age",
			Type: schema.FieldTypeInt,
		})

	db.CreateTable(userSchema)
	db.RegisterSchema("User", userSchema)

	// Insert test data
	for i := 1; i <= 5; i++ {
		_, err := db.Insert("User", map[string]interface{}{
			"name": "User" + string(rune('0'+i)),
			"age":  20 + i,
		})
		if err != nil {
			t.Fatalf("Failed to insert test user: %v", err)
		}
	}

	// Test QueryBuilder with model name
	qb := db.Select("User", []string{"name", "age"})
	results, err := qb.Where("age", ">=", 23).Execute()

	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestSchemaNotRegisteredError(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Try to use model name without registering schema
	_, err := db.Insert("UnregisteredModel", map[string]interface{}{
		"field": "value",
	})

	if err == nil {
		t.Error("Expected error for unregistered schema")
	}

	expectedError := "schema for model 'UnregisteredModel' not registered"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMixedFieldMapping(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create schema with mixed field mappings (some mapped, some not)
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "firstName",
			Type:     schema.FieldTypeString,
			Nullable: false,
			Map:      "first_name", // Mapped
		}).
		AddField(schema.Field{
			Name:     "lastName",
			Type:     schema.FieldTypeString,
			Nullable: false,
			// No mapping - uses field name as column name
		}).
		AddField(schema.Field{
			Name:    "userAge",
			Type:    schema.FieldTypeInt,
			Default: 0,
			Map:     "age", // Mapped
		})

	db.CreateTable(userSchema)
	db.RegisterSchema("User", userSchema)

	// Insert with mixed field names
	id, err := db.Insert("User", map[string]interface{}{
		"firstName": "John",
		"lastName":  "Doe",
		"userAge":   30,
	})

	if err != nil {
		t.Fatalf("Failed to insert with mixed field mapping: %v", err)
	}

	// Verify field names are returned correctly
	user, err := db.FindByID("User", id)
	if err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}

	// Check all fields are present with correct names
	expectedFields := map[string]interface{}{
		"id":        id,
		"firstName": "John",
		"lastName":  "Doe",
		"userAge":   int64(30),
	}

	for field, expectedValue := range expectedFields {
		if actualValue, exists := user[field]; !exists {
			t.Errorf("Expected field '%s' not found in result", field)
		} else if actualValue != expectedValue {
			t.Errorf("Field '%s': expected %v, got %v", field, expectedValue, actualValue)
		}
	}
}

func TestRawOperationsWithColumnNames(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create schema with field mappings
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "fullName",
			Type:     schema.FieldTypeString,
			Nullable: false,
			Map:      "full_name",
		})

	db.CreateTable(userSchema)
	db.RegisterSchema("User", userSchema)

	// Test RawInsert with column names
	id, err := db.RawInsert("users", map[string]interface{}{
		"full_name": "Raw User", // Using actual column name
	})

	if err != nil {
		t.Fatalf("Failed to raw insert: %v", err)
	}

	// Test RawFindByID returns column names
	rawUser, err := db.RawFindByID("users", id)
	if err != nil {
		t.Fatalf("Failed to raw find by ID: %v", err)
	}

	// Verify column names are used
	if fullName, exists := rawUser["full_name"]; !exists {
		t.Error("Expected 'full_name' column in raw result")
	} else if fullName != "Raw User" {
		t.Errorf("Expected full_name 'Raw User', got %v", fullName)
	}

	// Verify field name is NOT present
	if _, exists := rawUser["fullName"]; exists {
		t.Error("Field name 'fullName' should not be present in raw result")
	}
}

func TestComplexFieldConversion(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Create schema with various field types and mappings
	postSchema := schema.New("Post").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:     "postTitle",
			Type:     schema.FieldTypeString,
			Nullable: false,
			Map:      "title",
		}).
		AddField(schema.Field{
			Name:     "postContent",
			Type:     schema.FieldTypeString,
			Nullable: true,
			Map:      "content",
		}).
		AddField(schema.Field{
			Name:     "authorId",
			Type:     schema.FieldTypeInt,
			Nullable: false,
			Map:      "author_id",
		}).
		AddField(schema.Field{
			Name:    "viewCount",
			Type:    schema.FieldTypeInt,
			Default: 0,
			Map:     "views",
		}).
		AddField(schema.Field{
			Name:     "publishedAt",
			Type:     schema.FieldTypeDateTime,
			Nullable: true,
			Map:      "published_at",
		})

	db.CreateTable(postSchema)
	db.RegisterSchema("Post", postSchema)

	// Insert complex data
	postData := map[string]interface{}{
		"postTitle":   "Test Post",
		"postContent": "This is a test post",
		"authorId":    123,
		"viewCount":   0,
		"publishedAt": "2024-01-01 00:00:00",
	}

	id, err := db.Insert("Post", postData)
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
	}

	// Retrieve and verify all fields are converted correctly
	post, err := db.FindByID("Post", id)
	if err != nil {
		t.Fatalf("Failed to find post: %v", err)
	}

	// Verify all field names are present (not column names)
	expectedFields := []string{"id", "postTitle", "postContent", "authorId", "viewCount", "publishedAt"}
	for _, field := range expectedFields {
		if _, exists := post[field]; !exists {
			t.Errorf("Expected field '%s' not found in result", field)
		}
	}

	// Verify column names are NOT present
	columnNames := []string{"title", "content", "author_id", "views", "published_at"}
	for _, col := range columnNames {
		if _, exists := post[col]; exists {
			t.Errorf("Column name '%s' should not be present in result", col)
		}
	}
}

// Helper function to create in-memory test database
func createTestDB(t *testing.T) *SQLiteDB {
	config := types.Config{
		Type:     "sqlite",
		FilePath: ":memory:", // In-memory database
	}

	db, err := NewSQLiteDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return db
}
