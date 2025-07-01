package models

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Create SQLite in-memory database for testing
func newTestDB() types.Database {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		panic(err)
	}
	if err := db.Connect(); err != nil {
		panic(err)
	}
	return db
}

func createTestSchema() *schema.Schema {
	return schema.New("User").
		AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build())
}

func TestModelGet(t *testing.T) {
	db := newTestDB()
	defer db.Close()
	
	testSchema := createTestSchema()
	
	// Register schema and create table
	err := db.RegisterSchema("User", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropModel("User")
	
	model := New(testSchema, db)

	// Insert test data
	testData := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}
	id, err := db.Insert("User", testData)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Test Get
	result, err := model.Get(id)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}

	if result["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%v'", result["name"])
	}
}

func TestModelAdd(t *testing.T) {
	db := newTestDB()
	defer db.Close()
	
	testSchema := createTestSchema()
	
	// Register schema and create table
	err := db.RegisterSchema("User", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropModel("User")
	
	model := New(testSchema, db)

	// Test Add
	data := map[string]interface{}{
		"name":  "Jane Doe",
		"email": "jane@example.com",
		"age":   25,
	}

	id, err := model.Add(data)
	if err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %v", id)
	}

	// Verify the record was added
	result, err := model.Get(id)
	if err != nil {
		t.Fatalf("Failed to get added record: %v", err)
	}

	if result["name"] != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got '%v'", result["name"])
	}
}

func TestModelAddValidation(t *testing.T) {
	db := newTestDB()
	defer db.Close()
	
	testSchema := createTestSchema()
	
	// Register schema and create table
	err := db.RegisterSchema("User", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropModel("User")
	
	model := New(testSchema, db)

	// Test validation - missing required field
	data := map[string]interface{}{
		"email": "test@example.com",
		// Missing required "name" field
	}

	_, err = model.Add(data)
	if err == nil {
		t.Error("Expected validation error for missing required field")
	}

	// Test validation - invalid field type
	data = map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
		"age":   "invalid_age", // Should be int
	}

	_, err = model.Add(data)
	if err == nil {
		t.Error("Expected validation error for invalid field type")
	}
}

func TestModelSet(t *testing.T) {
	db := newTestDB()
	defer db.Close()
	
	testSchema := createTestSchema()
	
	// Register schema and create table
	err := db.RegisterSchema("User", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropModel("User")
	
	model := New(testSchema, db)

	// Insert initial data
	data := map[string]interface{}{
		"name":  "Original Name",
		"email": "original@example.com",
		"age":   30,
	}
	id, _ := model.Add(data)

	// Test Set
	updateData := map[string]interface{}{
		"name": "Updated Name",
		"age":  35,
	}

	err = model.Set(id, updateData)
	if err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	// Verify the update
	result, err := model.Get(id)
	if err != nil {
		t.Fatalf("Failed to get updated record: %v", err)
	}

	if result["name"] != "Updated Name" {
		t.Errorf("Expected updated name 'Updated Name', got '%v'", result["name"])
	}

	if result["age"] != int64(35) {
		t.Errorf("Expected updated age 35, got '%v'", result["age"])
	}
}

func TestModelRemove(t *testing.T) {
	db := newTestDB()
	defer db.Close()
	
	testSchema := createTestSchema()
	
	// Register schema and create table
	err := db.RegisterSchema("User", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropModel("User")
	
	model := New(testSchema, db)

	// Insert test data
	data := map[string]interface{}{
		"name":  "To Be Deleted",
		"email": "delete@example.com",
		"age":   25,
	}
	id, _ := model.Add(data)

	// Test Remove
	err = model.Remove(id)
	if err != nil {
		t.Fatalf("Failed to remove record: %v", err)
	}

	// Verify the record was removed
	_, err = model.Get(id)
	if err == nil {
		t.Error("Expected error when getting removed record")
	}
}

func TestModelSelect(t *testing.T) {
	db := newTestDB()
	defer db.Close()
	
	testSchema := createTestSchema()
	
	// Register schema and create table
	err := db.RegisterSchema("User", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropModel("User")
	
	model := New(testSchema, db)

	// Insert test data
	users := []map[string]interface{}{
		{"name": "User1", "email": "user1@example.com", "age": 25},
		{"name": "User2", "email": "user2@example.com", "age": 30},
		{"name": "User3", "email": "user3@example.com", "age": 35},
	}

	for _, user := range users {
		_, err := model.Add(user)
		if err != nil {
			t.Fatalf("Failed to add test user: %v", err)
		}
	}

	// Test Select
	qb := model.Select("name", "age")
	results, err := qb.Where("age", ">", 25).OrderBy("age", "ASC").Execute()
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0]["name"] != "User2" {
		t.Errorf("Expected first result name 'User2', got '%v'", results[0]["name"])
	}
}

func TestValidateFieldType(t *testing.T) {
	testCases := []struct {
		name        string
		field       *schema.Field
		value       interface{}
		expectError bool
	}{
		{
			name:        "Valid string",
			field:       &schema.Field{Name: "name", Type: schema.FieldTypeString},
			value:       "test",
			expectError: false,
		},
		{
			name:        "Invalid string type",
			field:       &schema.Field{Name: "name", Type: schema.FieldTypeString},
			value:       123,
			expectError: true,
		},
		{
			name:        "Null value for nullable field",
			field:       &schema.Field{Name: "name", Type: schema.FieldTypeString, Nullable: true},
			value:       nil,
			expectError: false,
		},
		{
			name:        "Null value for non-nullable field",
			field:       &schema.Field{Name: "name", Type: schema.FieldTypeString},
			value:       nil,
			expectError: true,
		},
		{
			name:        "Valid int",
			field:       &schema.Field{Name: "age", Type: schema.FieldTypeInt},
			value:       25,
			expectError: false,
		},
		{
			name:        "Valid int64",
			field:       &schema.Field{Name: "age", Type: schema.FieldTypeInt64},
			value:       int64(25),
			expectError: false,
		},
		{
			name:        "Valid float as int",
			field:       &schema.Field{Name: "age", Type: schema.FieldTypeInt},
			value:       25.0,
			expectError: false,
		},
		{
			name:        "Valid bool",
			field:       &schema.Field{Name: "active", Type: schema.FieldTypeBool},
			value:       true,
			expectError: false,
		},
		{
			name:        "Invalid bool type",
			field:       &schema.Field{Name: "active", Type: schema.FieldTypeBool},
			value:       "true",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFieldType(tc.field, tc.value)
			if tc.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}