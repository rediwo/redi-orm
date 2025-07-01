package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func TestPostgreSQLConvertFieldNames(t *testing.T) {
	// Create test database (in-memory for testing conversion logic)
	config := types.Config{
		Type: "postgresql",
	}

	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create schema with field mappings
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name: "id",
			Type: schema.FieldTypeInt,
		}).
		AddField(schema.Field{
			Name: "fullName",
			Type: schema.FieldTypeString,
			Map:  "full_name", // Mapped field
		}).
		AddField(schema.Field{
			Name: "email",
			Type: schema.FieldTypeString,
			// No mapping - should use field name as column name
		}).
		AddField(schema.Field{
			Name: "userAge",
			Type: schema.FieldTypeInt,
			Map:  "age", // Mapped field
		})

	// Register schema
	db.RegisterSchema("User", userSchema)

	// Test field name conversion
	inputData := map[string]interface{}{
		"fullName": "John Doe",
		"email":    "john@example.com",
		"userAge":  30,
		"unknown":  "should-remain", // Unknown field should remain unchanged
	}

	convertedData, err := db.convertFieldNames("User", inputData)
	if err != nil {
		t.Fatalf("Failed to convert field names: %v", err)
	}

	// Verify conversions
	expectedData := map[string]interface{}{
		"full_name": "John Doe",         // fullName -> full_name
		"email":     "john@example.com", // no mapping, stays same
		"age":       30,                 // userAge -> age
		"unknown":   "should-remain",    // unknown field remains
	}

	if len(convertedData) != len(expectedData) {
		t.Errorf("Expected %d fields, got %d", len(expectedData), len(convertedData))
	}

	for key, expectedValue := range expectedData {
		if actualValue, exists := convertedData[key]; !exists {
			t.Errorf("Expected key '%s' not found in converted data", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key '%s': expected %v, got %v", key, expectedValue, actualValue)
		}
	}

	// Verify old field names are not present
	if _, exists := convertedData["fullName"]; exists {
		t.Error("Field name 'fullName' should be converted to 'full_name'")
	}
	if _, exists := convertedData["userAge"]; exists {
		t.Error("Field name 'userAge' should be converted to 'age'")
	}
}

func TestPostgreSQLConvertResultFieldNames(t *testing.T) {
	// Create test database
	config := types.Config{
		Type: "postgresql",
	}

	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create schema with field mappings
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name: "id",
			Type: schema.FieldTypeInt,
		}).
		AddField(schema.Field{
			Name: "fullName",
			Type: schema.FieldTypeString,
			Map:  "full_name", // Mapped field
		}).
		AddField(schema.Field{
			Name: "email",
			Type: schema.FieldTypeString,
			// No mapping
		}).
		AddField(schema.Field{
			Name: "userAge",
			Type: schema.FieldTypeInt,
			Map:  "age", // Mapped field
		})

	// Register schema
	db.RegisterSchema("User", userSchema)

	// Test result field name conversion (database columns -> field names)
	inputData := map[string]interface{}{
		"id":          1,
		"full_name":   "John Doe",
		"email":       "john@example.com",
		"age":         30,
		"unknown_col": "should-remain", // Unknown column should remain unchanged
	}

	convertedData := db.convertResultFieldNames("User", inputData)

	// Verify conversions
	expectedData := map[string]interface{}{
		"id":          1,
		"fullName":    "John Doe",         // full_name -> fullName
		"email":       "john@example.com", // no mapping, stays same
		"userAge":     30,                 // age -> userAge
		"unknown_col": "should-remain",    // unknown column remains
	}

	if len(convertedData) != len(expectedData) {
		t.Errorf("Expected %d fields, got %d", len(expectedData), len(convertedData))
	}

	for key, expectedValue := range expectedData {
		if actualValue, exists := convertedData[key]; !exists {
			t.Errorf("Expected key '%s' not found in converted data", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key '%s': expected %v, got %v", key, expectedValue, actualValue)
		}
	}

	// Verify old column names are not present
	if _, exists := convertedData["full_name"]; exists {
		t.Error("Column name 'full_name' should be converted to 'fullName'")
	}
	if _, exists := convertedData["age"]; exists {
		t.Error("Column name 'age' should be converted to 'userAge'")
	}
}

func TestPostgreSQLConvertFieldNamesWithUnregisteredSchema(t *testing.T) {
	// Create test database
	config := types.Config{
		Type: "postgresql",
	}

	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Test with unregistered schema - should return data as-is
	inputData := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
	}

	convertedData, err := db.convertFieldNames("UnregisteredModel", inputData)
	if err != nil {
		t.Fatalf("Failed to convert field names for unregistered schema: %v", err)
	}

	// Should return original data unchanged
	if len(convertedData) != len(inputData) {
		t.Errorf("Expected %d fields, got %d", len(inputData), len(convertedData))
	}

	for key, expectedValue := range inputData {
		if actualValue, exists := convertedData[key]; !exists {
			t.Errorf("Expected key '%s' not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key '%s': expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestPostgreSQLConvertResultFieldNamesWithUnregisteredSchema(t *testing.T) {
	// Create test database
	config := types.Config{
		Type: "postgresql",
	}

	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Test with unregistered schema - should return data as-is
	inputData := map[string]interface{}{
		"column1": "value1",
		"column2": "value2",
	}

	convertedData := db.convertResultFieldNames("UnregisteredModel", inputData)

	// Should return original data unchanged
	if len(convertedData) != len(inputData) {
		t.Errorf("Expected %d fields, got %d", len(inputData), len(convertedData))
	}

	for key, expectedValue := range inputData {
		if actualValue, exists := convertedData[key]; !exists {
			t.Errorf("Expected key '%s' not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("Key '%s': expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}
