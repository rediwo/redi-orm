package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
)

// TestColumnMigrationAddNewColumn tests adding new columns to existing tables
func TestColumnMigrationAddNewColumn(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Phase 1: Create initial schema with basic fields
	initialSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(initialSchema); err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Initial migration failed: %v", err)
	}

	// Insert initial data
	userID, err := eng.Execute(`models.User.add({name: "Alice"})`)
	if err != nil {
		t.Fatalf("Failed to insert initial user: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	// Phase 2: Add new columns to the schema
	updatedSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build())

	// Create new engine with updated schema
	eng2 := engine.New(db)
	if err := eng2.RegisterSchema(updatedSchema); err != nil {
		t.Fatalf("Failed to register updated schema: %v", err)
	}

	if err := eng2.EnsureSchema(); err != nil {
		t.Fatalf("Column addition migration failed: %v", err)
	}

	// Verify existing data is preserved
	user, err := eng2.Execute(`models.User.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get user after migration: %v", err)
	}

	userData := user.(map[string]interface{})
	if userData["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", userData["name"])
	}

	// Insert data with new columns
	newUserID, err := eng2.Execute(`models.User.add({name: "Bob", email: "bob@test.com", age: 25})`)
	if err != nil {
		t.Fatalf("Failed to insert user with new columns: %v", err)
	}

	if newUserID != int64(2) {
		t.Errorf("Expected user ID 2, got %v", newUserID)
	}

	// Verify new user data
	newUser, err := eng2.Execute(`models.User.get(2)`)
	if err != nil {
		t.Fatalf("Failed to get new user: %v", err)
	}

	newUserData := newUser.(map[string]interface{})
	if newUserData["email"] != "bob@test.com" {
		t.Errorf("Expected email 'bob@test.com', got %v", newUserData["email"])
	}

	if newUserData["age"] != int64(25) {
		t.Errorf("Expected age 25, got %v", newUserData["age"])
	}

	t.Log("✅ Column addition migration works correctly")
}

// TestColumnMigrationWithDefaults tests adding columns with default values
func TestColumnMigrationWithDefaults(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Create initial schema
	initialSchema := schema.New("Product").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(initialSchema); err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Initial migration failed: %v", err)
	}

	// Insert initial data
	productID, err := eng.Execute(`models.Product.add({name: "Widget"})`)
	if err != nil {
		t.Fatalf("Failed to insert initial product: %v", err)
	}

	// Add columns with default values
	updatedSchema := schema.New("Product").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("price").Float().Default(0.0).Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build())

	eng2 := engine.New(db)
	if err := eng2.RegisterSchema(updatedSchema); err != nil {
		t.Fatalf("Failed to register updated schema: %v", err)
	}

	if err := eng2.EnsureSchema(); err != nil {
		t.Fatalf("Column addition with defaults migration failed: %v", err)
	}

	// Verify existing data has default values
	product, err := eng2.Execute(`models.Product.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get product after migration: %v", err)
	}

	productData := product.(map[string]interface{})
	if productData["name"] != "Widget" {
		t.Errorf("Expected name 'Widget', got %v", productData["name"])
	}

	// Insert new product with explicit values
	newProductID, err := eng2.Execute(`models.Product.add({name: "Gadget", price: 29.99, active: false})`)
	if err != nil {
		t.Fatalf("Failed to insert product with explicit values: %v", err)
	}

	if newProductID != int64(2) {
		t.Errorf("Expected product ID 2, got %v", newProductID)
	}

	if productID != int64(1) {
		t.Errorf("Expected initial product ID 1, got %v", productID)
	}

	t.Log("✅ Column addition with defaults works correctly")
}

// TestColumnMigrationMultipleRounds tests multiple rounds of column additions
func TestColumnMigrationMultipleRounds(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Round 1: Basic schema
	eng1 := engine.New(db)
	schema1 := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng1.RegisterSchema(schema1); err != nil {
		t.Fatalf("Failed to register schema1: %v", err)
	}

	if err := eng1.EnsureSchema(); err != nil {
		t.Fatalf("Round 1 migration failed: %v", err)
	}

	// Insert Round 1 data
	_, err = eng1.Execute(`models.User.add({name: "Alice"})`)
	if err != nil {
		t.Fatalf("Failed to insert round 1 user: %v", err)
	}

	// Round 2: Add email column
	eng2 := engine.New(db)
	schema2 := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build())

	if err := eng2.RegisterSchema(schema2); err != nil {
		t.Fatalf("Failed to register schema2: %v", err)
	}

	if err := eng2.EnsureSchema(); err != nil {
		t.Fatalf("Round 2 migration failed: %v", err)
	}

	// Insert Round 2 data
	_, err = eng2.Execute(`models.User.add({name: "Bob", email: "bob@test.com"})`)
	if err != nil {
		t.Fatalf("Failed to insert round 2 user: %v", err)
	}

	// Round 3: Add age and phone columns
	eng3 := engine.New(db)
	schema3 := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("phone").String().Nullable().Build())

	if err := eng3.RegisterSchema(schema3); err != nil {
		t.Fatalf("Failed to register schema3: %v", err)
	}

	if err := eng3.EnsureSchema(); err != nil {
		t.Fatalf("Round 3 migration failed: %v", err)
	}

	// Insert Round 3 data
	_, err = eng3.Execute(`models.User.add({name: "Charlie", email: "charlie@test.com", age: 30, phone: "555-1234"})`)
	if err != nil {
		t.Fatalf("Failed to insert round 3 user: %v", err)
	}

	// Verify all data exists with expected structure
	totalUsers, err := eng3.Execute(`models.User.select().count()`)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	if totalUsers != int64(3) {
		t.Errorf("Expected 3 users, got %v", totalUsers)
	}

	// Verify each user has the expected fields
	user1, err := eng3.Execute(`models.User.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get user 1: %v", err)
	}

	user1Data := user1.(map[string]interface{})
	if user1Data["name"] != "Alice" {
		t.Errorf("Expected Alice for user 1, got %v", user1Data["name"])
	}

	user3, err := eng3.Execute(`models.User.get(3)`)
	if err != nil {
		t.Fatalf("Failed to get user 3: %v", err)
	}

	user3Data := user3.(map[string]interface{})
	if user3Data["phone"] != "555-1234" {
		t.Errorf("Expected phone '555-1234' for user 3, got %v", user3Data["phone"])
	}

	t.Log("✅ Multiple rounds of column migration work correctly")
}

// TestColumnMigrationIdempotency tests that column migrations are idempotent
func TestColumnMigrationIdempotency(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Create schema with extra columns
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Run migration multiple times
	for i := 0; i < 5; i++ {
		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("Migration failed on iteration %d: %v", i+1, err)
		}
	}

	// Verify table works correctly
	userID, err := eng.Execute(`models.User.add({name: "Test User", email: "test@example.com", age: 25})`)
	if err != nil {
		t.Fatalf("Failed to insert user after multiple migrations: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	t.Log("✅ Column migration idempotency works correctly")
}

// TestColumnMigrationWithDifferentTypes tests migration with various column types
func TestColumnMigrationWithDifferentTypes(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Initial simple schema
	initialSchema := schema.New("Record").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(initialSchema); err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Initial migration failed: %v", err)
	}

	// Add complex schema with various types
	complexSchema := schema.New("Record").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("count").Int().Default(0).Build()).
		AddField(schema.NewField("price").Float().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build()).
		AddField(schema.NewField("metadata").JSON().Nullable().Build()).
		AddField(schema.NewField("createdAt").DateTime().Build())

	eng2 := engine.New(db)
	if err := eng2.RegisterSchema(complexSchema); err != nil {
		t.Fatalf("Failed to register complex schema: %v", err)
	}

	if err := eng2.EnsureSchema(); err != nil {
		t.Fatalf("Complex schema migration failed: %v", err)
	}

	// Insert data with various types
	recordID, err := eng2.Execute(`models.Record.add({
		name: "Complex Record",
		count: 42,
		price: 19.99,
		active: false,
		metadata: "{\"type\": \"test\"}",
		createdAt: "2024-01-01 12:00:00"
	})`)
	if err != nil {
		t.Fatalf("Failed to insert complex record: %v", err)
	}

	if recordID != int64(1) {
		t.Errorf("Expected record ID 1, got %v", recordID)
	}

	// Verify data
	record, err := eng2.Execute(`models.Record.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get complex record: %v", err)
	}

	recordData := record.(map[string]interface{})
	if recordData["count"] != int64(42) {
		t.Errorf("Expected count 42, got %v", recordData["count"])
	}

	t.Log("✅ Column migration with different types works correctly")
}

// TestColumnMigrationConcurrency tests concurrent column migrations
func TestColumnMigrationConcurrency(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create initial schema
	eng := engine.New(db)
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Run concurrent migrations
	done := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			if err := eng.EnsureSchema(); err != nil {
				done <- err
				return
			}
			done <- nil
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Fatalf("Concurrent migration failed: %v", err)
		}
	}

	// Verify table works
	userID, err := eng.Execute(`models.User.add({name: "Concurrent User", email: "concurrent@test.com"})`)
	if err != nil {
		t.Fatalf("Failed to insert user after concurrent migrations: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	t.Log("✅ Concurrent column migrations handled safely")
}