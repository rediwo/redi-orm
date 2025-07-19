package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/mcp"
	"github.com/rediwo/redi-orm/schema"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
)

func TestModelResourcesList(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Test listing all models
	content, err := server.ReadResource(ctx, "model://")
	if err != nil {
		t.Fatalf("Failed to read model list: %v", err)
	}

	if content.MimeType != "application/json" {
		t.Errorf("Expected mime type application/json, got %s", content.MimeType)
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(content.Text), &response); err != nil {
		t.Fatalf("Failed to parse model list: %v", err)
	}

	models, ok := response["models"].([]interface{})
	if !ok {
		t.Fatalf("Expected models array in response")
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models (User, Post), got %d", len(models))
	}

	// Check model metadata
	foundUser := false
	foundPost := false
	for _, m := range models {
		model := m.(map[string]interface{})
		name := model["name"].(string)
		
		switch name {
		case "User":
			foundUser = true
			if fields := int(model["fields"].(float64)); fields != 5 {
				t.Errorf("Expected User to have 5 fields, got %d", fields)
			}
			if relations := int(model["relations"].(float64)); relations != 1 {
				t.Errorf("Expected User to have 1 relation, got %d", relations)
			}
		case "Post":
			foundPost = true
			if fields := int(model["fields"].(float64)); fields != 5 {
				t.Errorf("Expected Post to have 5 fields, got %d", fields)
			}
			if relations := int(model["relations"].(float64)); relations != 1 {
				t.Errorf("Expected Post to have 1 relation, got %d", relations)
			}
		}
	}

	if !foundUser || !foundPost {
		t.Errorf("Missing expected models. Found User: %v, Found Post: %v", foundUser, foundPost)
	}
}

func TestModelResourceDetails(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Test reading User model details
	content, err := server.ReadResource(ctx, "model://User")
	if err != nil {
		t.Fatalf("Failed to read User model: %v", err)
	}

	// Parse the response
	var model map[string]interface{}
	if err := json.Unmarshal([]byte(content.Text), &model); err != nil {
		t.Fatalf("Failed to parse model details: %v", err)
	}

	// Check model name
	if model["name"] != "User" {
		t.Errorf("Expected model name 'User', got %v", model["name"])
	}

	// Check fields
	fields, ok := model["fields"].([]interface{})
	if !ok {
		t.Fatalf("Expected fields array")
	}

	expectedFields := map[string]bool{
		"id":     false,
		"name":   false,
		"email":  false,
		"age":    false,
		"active": false,
	}

	for _, f := range fields {
		field := f.(map[string]interface{})
		fieldName := field["name"].(string)
		if _, exists := expectedFields[fieldName]; exists {
			expectedFields[fieldName] = true
			
			// Check specific field properties
			switch fieldName {
			case "id":
				if !field["primaryKey"].(bool) {
					t.Errorf("Expected id to be primary key")
				}
				if !field["autoIncrement"].(bool) {
					t.Errorf("Expected id to have autoIncrement")
				}
			case "email":
				if !field["unique"].(bool) {
					t.Errorf("Expected email to be unique")
				}
			case "age":
				if !field["nullable"].(bool) {
					t.Errorf("Expected age to be nullable")
				}
			case "active":
				if field["default"] != true {
					t.Errorf("Expected active to have default true")
				}
			}
		}
	}

	// Check all expected fields were found
	for field, found := range expectedFields {
		if !found {
			t.Errorf("Expected field '%s' not found", field)
		}
	}

	// Check relations
	relations, ok := model["relations"].([]interface{})
	if !ok || len(relations) != 1 {
		t.Errorf("Expected 1 relation for User model")
	}
}

func TestModelResourceSchema(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Test reading User model Prisma schema
	content, err := server.ReadResource(ctx, "model://User/schema")
	if err != nil {
		t.Fatalf("Failed to read User schema: %v", err)
	}

	if content.MimeType != "text/plain" {
		t.Errorf("Expected mime type text/plain, got %s", content.MimeType)
	}

	// Check that it's valid Prisma format
	schemaText := content.Text
	
	// Should start with model declaration
	if !strings.HasPrefix(schemaText, "model User {") {
		t.Errorf("Expected Prisma schema to start with 'model User {', got: %s", schemaText)
	}

	// Should contain field definitions
	expectedFields := []string{
		"id Int @id @default(autoincrement())",
		"name String",
		"email String @unique",
		"age Int?",
		"active Boolean @default(true)",
		"posts Post[]",
	}

	for _, expected := range expectedFields {
		if !strings.Contains(schemaText, expected) {
			t.Errorf("Expected schema to contain '%s'", expected)
		}
	}

	// Should end with closing brace
	if !strings.HasSuffix(strings.TrimSpace(schemaText), "}") {
		t.Errorf("Expected Prisma schema to end with '}'")
	}
}

func TestModelResourceData(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create some test data
	users := []map[string]interface{}{
		{"name": "Alice", "email": "alice@test.com", "age": 25},
		{"name": "Bob", "email": "bob@test.com", "age": 30},
		{"name": "Charlie", "email": "charlie@test.com", "age": 35},
	}

	for _, user := range users {
		args := map[string]interface{}{
			"model": "User",
			"data":  user,
		}
		argsJSON, _ := json.Marshal(args)
		_, err := server.CallTool(ctx, "data.create", argsJSON)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Test reading model data
	t.Run("BasicData", func(t *testing.T) {
		content, err := server.ReadResource(ctx, "model://User/data")
		if err != nil {
			t.Fatalf("Failed to read User data: %v", err)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(content.Text), &response); err != nil {
			t.Fatalf("Failed to parse data response: %v", err)
		}

		// Check response structure
		if response["model"] != "User" {
			t.Errorf("Expected model 'User', got %v", response["model"])
		}

		data, ok := response["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data array")
		}

		if len(data) != 3 {
			t.Errorf("Expected 3 users, got %d", len(data))
		}

		// Check pagination info
		pagination, ok := response["pagination"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected pagination object")
		}

		if total := int(pagination["total"].(float64)); total != 3 {
			t.Errorf("Expected total 3, got %d", total)
		}
	})

	// Test data with pagination
	t.Run("PaginatedData", func(t *testing.T) {
		content, err := server.ReadResource(ctx, "model://User/data?limit=2&offset=1")
		if err != nil {
			t.Fatalf("Failed to read paginated data: %v", err)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(content.Text), &response); err != nil {
			t.Fatalf("Failed to parse data response: %v", err)
		}

		data, _ := response["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("Expected 2 users with limit=2, got %d", len(data))
		}

		pagination, _ := response["pagination"].(map[string]interface{})
		if limit := int(pagination["limit"].(float64)); limit != 2 {
			t.Errorf("Expected limit 2, got %d", limit)
		}
		if offset := int(pagination["offset"].(float64)); offset != 1 {
			t.Errorf("Expected offset 1, got %d", offset)
		}
	})
}

func TestModelResourceErrors(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Test non-existent model
	t.Run("NonExistentModel", func(t *testing.T) {
		_, err := server.ReadResource(ctx, "model://NonExistent")
		if err == nil {
			t.Errorf("Expected error for non-existent model")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	// Test non-existent model schema
	t.Run("NonExistentModelSchema", func(t *testing.T) {
		_, err := server.ReadResource(ctx, "model://NonExistent/schema")
		if err == nil {
			t.Errorf("Expected error for non-existent model schema")
		}
	})

	// Test non-existent model data
	t.Run("NonExistentModelData", func(t *testing.T) {
		_, err := server.ReadResource(ctx, "model://NonExistent/data")
		if err == nil {
			t.Errorf("Expected error for non-existent model data")
		}
	})
}

func TestModelResourceFieldMapping(t *testing.T) {
	// Create a test server with field mapping
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create schema with field mapping
	productSchema := schema.New("Product").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name: "productName",
			Type: schema.FieldTypeString,
			Map:  "product_name", // Database column name
		}).
		AddField(schema.Field{
			Name: "unitPrice",
			Type: schema.FieldTypeFloat,
			Map:  "unit_price", // Database column name
		})

	if err := db.RegisterSchema("Product", productSchema); err != nil {
		t.Fatalf("Failed to register Product schema: %v", err)
	}

	if err := db.SyncSchemas(ctx); err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Create MCP server
	config := mcp.ServerConfig{
		DatabaseURI:  "sqlite://:memory:",
		ReadOnly:     false,
		MaxQueryRows: 1000,
	}

	server, err := mcp.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}
	
	// Set the database we created
	server.SetDatabase(db)

	server.RegisterSchema("Product", productSchema)

	// Create a product using model fields
	args := map[string]interface{}{
		"model": "Product",
		"data": map[string]interface{}{
			"productName": "Test Product",
			"unitPrice":   99.99,
		},
	}
	argsJSON, _ := json.Marshal(args)
	_, err = server.CallTool(ctx, "data.create", argsJSON)
	if err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}

	// Read product data - should return model field names
	content, err := server.ReadResource(ctx, "model://Product/data")
	if err != nil {
		t.Fatalf("Failed to read Product data: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(content.Text), &response); err != nil {
		t.Fatalf("Failed to parse data response: %v", err)
	}

	data := response["data"].([]interface{})
	if len(data) == 0 {
		t.Fatalf("Expected product data")
	}

	product := data[0].(map[string]interface{})
	
	// Should have model field names, not database column names
	if _, hasModelField := product["productName"]; !hasModelField {
		t.Errorf("Expected model field 'productName' in response")
	}
	if _, hasDBColumn := product["product_name"]; hasDBColumn {
		t.Errorf("Should not have database column 'product_name' in response")
	}

	if product["productName"] != "Test Product" {
		t.Errorf("Expected productName 'Test Product', got %v", product["productName"])
	}
	if product["unitPrice"] != 99.99 {
		t.Errorf("Expected unitPrice 99.99, got %v", product["unitPrice"])
	}

	// Check schema shows @map annotations
	schemaContent, err := server.ReadResource(ctx, "model://Product/schema")
	if err != nil {
		t.Fatalf("Failed to read Product schema: %v", err)
	}

	if !strings.Contains(schemaContent.Text, `productName String @map("product_name")`) {
		t.Errorf("Expected schema to contain field mapping for productName")
	}
	if !strings.Contains(schemaContent.Text, `unitPrice Float @map("unit_price")`) {
		t.Errorf("Expected schema to contain field mapping for unitPrice")
	}
}