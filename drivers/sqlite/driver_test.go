package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func setupTestDB(t *testing.T) (*SQLiteDB, func()) {
	config := types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	}

	db, err := NewSQLiteDB(config)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func TestSQLiteConnect(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test that connection is established
	if db.db == nil {
		t.Error("Expected database connection to be established")
	}
}

func TestSQLiteCreateModel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test schema
	testSchema := &schema.Schema{
		Name:      "User",
		TableName: "users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString, Nullable: false},
			{Name: "email", Type: schema.FieldTypeString, Unique: true},
			{Name: "age", Type: schema.FieldTypeInt, Nullable: true},
			{Name: "active", Type: schema.FieldTypeBool, Default: true},
		},
	}

	err := db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Verify table exists
	var tableName string
	err = db.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	if err != nil {
		t.Errorf("Table 'users' was not created: %v", err)
	}
	if tableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", tableName)
	}
}

func TestSQLiteInsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create table first
	createTestTable(t, db)

	// Test raw insert (using table name)
	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}

	id, err := db.RawInsert("users", data)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	if id != 1 {
		t.Errorf("Expected first insert ID to be 1, got %d", id)
	}

	// Verify data was inserted
	var name, email string
	var age int
	err = db.db.QueryRow("SELECT name, email, age FROM users WHERE id = ?", id).Scan(&name, &email, &age)
	if err != nil {
		t.Fatalf("Failed to query inserted data: %v", err)
	}

	if name != "John Doe" || email != "john@example.com" || age != 30 {
		t.Errorf("Inserted data doesn't match: got name=%s, email=%s, age=%d", name, email, age)
	}
}

func TestSQLiteFindByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	createTestTable(t, db)

	// Insert test data
	data := map[string]interface{}{
		"name":  "Jane Doe",
		"email": "jane@example.com",
		"age":   25,
	}
	id, _ := db.RawInsert("users", data)

	// Test RawFindByID
	result, err := db.RawFindByID("users", id)
	if err != nil {
		t.Fatalf("Failed to find by ID: %v", err)
	}

	if result["name"] != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got '%v'", result["name"])
	}

	// Test non-existent ID
	_, err = db.RawFindByID("users", 999)
	if err == nil {
		t.Error("Expected error for non-existent ID")
	}
}

func TestSQLiteFind(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	createTestTable(t, db)

	// Insert multiple records
	users := []map[string]interface{}{
		{"name": "User1", "email": "user1@example.com", "age": 20},
		{"name": "User2", "email": "user2@example.com", "age": 25},
		{"name": "User3", "email": "user3@example.com", "age": 30},
	}

	for _, user := range users {
		db.RawInsert("users", user)
	}

	// Test find all
	results, err := db.RawFind("users", nil, 0, 0)
	if err != nil {
		t.Fatalf("Failed to find records: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 records, got %d", len(results))
	}

	// Test find with conditions
	conditions := map[string]interface{}{"age": 25}
	results, err = db.RawFind("users", conditions, 0, 0)
	if err != nil {
		t.Fatalf("Failed to find with conditions: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 record, got %d", len(results))
	}

	// Test with limit and offset
	results, err = db.RawFind("users", nil, 2, 1)
	if err != nil {
		t.Fatalf("Failed to find with limit/offset: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 records with limit, got %d", len(results))
	}
}

func TestSQLiteUpdate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	createTestTable(t, db)

	// Insert test data
	data := map[string]interface{}{
		"name":  "Original Name",
		"email": "original@example.com",
		"age":   20,
	}
	id, _ := db.RawInsert("users", data)

	// Update data
	updateData := map[string]interface{}{
		"name": "Updated Name",
		"age":  21,
	}
	err := db.RawUpdate("users", id, updateData)
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	// Verify update
	result, _ := db.RawFindByID("users", id)
	if result["name"] != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%v'", result["name"])
	}
	if result["email"] != "original@example.com" {
		t.Errorf("Email should not have changed, got '%v'", result["email"])
	}
}

func TestSQLiteDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	createTestTable(t, db)

	// Insert test data
	data := map[string]interface{}{
		"name":  "To Delete",
		"email": "delete@example.com",
		"age":   30,
	}
	id, _ := db.RawInsert("users", data)

	// Delete record
	err := db.RawDelete("users", id)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deletion
	_, err = db.RawFindByID("users", id)
	if err == nil {
		t.Error("Expected error when finding deleted record")
	}
}

func TestSQLiteTransaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	createTestTable(t, db)

	// Test successful transaction
	t.Run("Successful transaction", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		data := map[string]interface{}{
			"name":  "Transaction Test",
			"email": "tx@example.com",
			"age":   25,
		}

		id, err := tx.Insert("users", data)
		if err != nil {
			t.Fatalf("Failed to insert in transaction: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify data was committed
		result, err := db.RawFindByID("users", id)
		if err != nil {
			t.Error("Expected to find committed record")
		}
		if result["name"] != "Transaction Test" {
			t.Errorf("Expected name 'Transaction Test', got '%v'", result["name"])
		}
	})

	// Test rollback
	t.Run("Rollback transaction", func(t *testing.T) {
		tx, _ := db.Begin()

		data := map[string]interface{}{
			"name":  "Rollback Test",
			"email": "rollback@example.com",
			"age":   30,
		}

		tx.Insert("users", data)

		err := tx.Rollback()
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		// Verify data was not committed
		results, _ := db.RawFind("users", map[string]interface{}{"email": "rollback@example.com"}, 0, 0)
		if len(results) > 0 {
			t.Error("Expected no records after rollback")
		}
	})
}

func TestSQLiteDropModel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test schema and register it
	testSchema := &schema.Schema{
		Name:      "User",
		TableName: "users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "age", Type: schema.FieldTypeInt},
		},
	}

	// Register schema first
	err := db.RegisterSchema("User", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	if err := db.CreateModel(testSchema); err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	// Drop model
	err = db.DropModel("User")
	if err != nil {
		t.Fatalf("Failed to drop model: %v", err)
	}

	// Verify table doesn't exist
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	if err != nil || count != 0 {
		t.Error("Table should not exist after drop")
	}
}

// Helper function to create test table
func createTestTable(t *testing.T, db *SQLiteDB) {
	testSchema := &schema.Schema{
		Name:      "User",
		TableName: "users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "age", Type: schema.FieldTypeInt},
		},
	}

	if err := db.CreateModel(testSchema); err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}
}
