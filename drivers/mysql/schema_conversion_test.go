package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func TestMySQLSchemaRegistration(t *testing.T) {
	db := createMySQLTestDB(t)
	if db == nil {
		t.Skip("MySQL not available, skipping test")
		return
	}
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

func TestMySQLFieldNameToColumnNameConversion(t *testing.T) {
	db := createMySQLTestDB(t)
	if db == nil {
		t.Skip("MySQL not available, skipping test")
		return
	}
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

func TestMySQLFindWithFieldNameConditions(t *testing.T) {
	db := createMySQLTestDB(t)
	if db == nil {
		t.Skip("MySQL not available, skipping test")
		return
	}
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

func TestMySQLUpdateWithFieldNames(t *testing.T) {
	db := createMySQLTestDB(t)
	if db == nil {
		t.Skip("MySQL not available, skipping test")
		return
	}
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

func TestMySQLMixedFieldMapping(t *testing.T) {
	db := createMySQLTestDB(t)
	if db == nil {
		t.Skip("MySQL not available, skipping test")
		return
	}
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

// Helper function to create MySQL test database
func createMySQLTestDB(t *testing.T) *MySQLDB {
	config := types.Config{
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "password",
		Database: "test_db",
	}

	db, err := NewMySQLDB(config)
	if err != nil {
		t.Logf("Failed to create MySQL test database: %v", err)
		return nil
	}

	if err := db.Connect(); err != nil {
		t.Logf("Failed to connect to MySQL test database: %v (Docker might not be running)", err)
		return nil
	}

	return db
}
