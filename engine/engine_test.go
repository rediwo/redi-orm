package engine

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// createTestDB creates an in-memory SQLite database for testing
func createTestDB(t *testing.T) types.Database {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return db
}

// createTestSchema creates a basic User schema for testing
func createTestSchema() *schema.Schema {
	return schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build())
}

func TestEngineNew(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	if engine == nil {
		t.Error("Expected engine to be created")
	}
}

func TestEngineRegisterSchema(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	err := engine.RegisterSchema(testSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}
}

func TestEngineRegisterInvalidSchema(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)

	// Create invalid schema (no primary key)
	invalidSchema := schema.New("Invalid").
		AddField(schema.NewField("name").String().Build())

	err := engine.RegisterSchema(invalidSchema)
	if err == nil {
		t.Error("Expected error for invalid schema")
	}
}

func TestEngineJavaScriptModelAdd(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Test adding a user via JavaScript
	script := `models.User.add({name: "John Doe", email: "john@example.com", age: 30})`
	result, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute add script: %v", err)
	}

	// Should return the ID
	if result != int64(1) {
		t.Errorf("Expected ID 1, got %v", result)
	}

	// Verify data was added
	getScript := `models.User.get(1)`
	user, err := engine.Execute(getScript)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	userData := user.(map[string]interface{})
	if userData["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", userData["name"])
	}
}

func TestEngineJavaScriptModelGet(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Add test data first
	addScript := `models.User.add({name: "Alice", email: "alice@example.com", age: 25})`
	_, err := engine.Execute(addScript)
	if err != nil {
		t.Fatalf("Failed to add test user: %v", err)
	}

	// Test getting the user
	getScript := `models.User.get(1)`
	result, err := engine.Execute(getScript)
	if err != nil {
		t.Fatalf("Failed to execute get script: %v", err)
	}

	userData := result.(map[string]interface{})
	if userData["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", userData["name"])
	}
}

func TestEngineJavaScriptModelSet(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Add test data first
	addScript := `models.User.add({name: "Bob", email: "bob@example.com", age: 30})`
	_, err := engine.Execute(addScript)
	if err != nil {
		t.Fatalf("Failed to add test user: %v", err)
	}

	// Test updating the user
	setScript := `models.User.set(1, {age: 31})`
	_, err = engine.Execute(setScript)
	if err != nil {
		t.Fatalf("Failed to execute set script: %v", err)
	}

	// Verify update
	getScript := `models.User.get(1)`
	result, err := engine.Execute(getScript)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	userData := result.(map[string]interface{})
	if userData["age"] != int64(31) {
		t.Errorf("Expected age 31, got %v", userData["age"])
	}
}

func TestEngineJavaScriptModelRemove(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Add test data first
	addScript := `models.User.add({name: "Charlie", email: "charlie@example.com", age: 35})`
	_, err := engine.Execute(addScript)
	if err != nil {
		t.Fatalf("Failed to add test user: %v", err)
	}

	// Test removing the user
	removeScript := `models.User.remove(1)`
	_, err = engine.Execute(removeScript)
	if err != nil {
		t.Fatalf("Failed to execute remove script: %v", err)
	}

	// Verify removal
	getScript := `models.User.get(1)`
	_, err = engine.Execute(getScript)
	if err == nil {
		t.Error("Expected error when getting removed user")
	}
}

func TestEngineJavaScriptModelSelect(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Add test data
	testUsers := []string{
		`models.User.add({name: "Alice", email: "alice@example.com", age: 25})`,
		`models.User.add({name: "Bob", email: "bob@example.com", age: 30})`,
		`models.User.add({name: "Charlie", email: "charlie@example.com", age: 35})`,
	}

	for _, userScript := range testUsers {
		_, err := engine.Execute(userScript)
		if err != nil {
			t.Fatalf("Failed to add test user: %v", err)
		}
	}

	// Test selecting all users
	selectScript := `models.User.select().execute()`
	result, err := engine.Execute(selectScript)
	if err != nil {
		t.Fatalf("Failed to execute select script: %v", err)
	}

	users := result.([]map[string]interface{})
	if len(users) != 3 {
		t.Errorf("Expected 3 results, got %d", len(users))
	}
}

func TestEngineJavaScriptModelSelectWithWhere(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Add test data
	testUsers := []string{
		`models.User.add({name: "Alice", email: "alice@example.com", age: 25})`,
		`models.User.add({name: "Bob", email: "bob@example.com", age: 30})`,
		`models.User.add({name: "Charlie", email: "charlie@example.com", age: 25})`,
	}

	for _, userScript := range testUsers {
		_, err := engine.Execute(userScript)
		if err != nil {
			t.Fatalf("Failed to add test user: %v", err)
		}
	}

	// Test selecting users with WHERE clause
	selectScript := `models.User.select().where("age", "=", 25).execute()`
	result, err := engine.Execute(selectScript)
	if err != nil {
		t.Fatalf("Failed to execute select with where script: %v", err)
	}

	users := result.([]map[string]interface{})
	if len(users) != 2 {
		t.Errorf("Expected 2 results, got %d", len(users))
	}
}

func TestEngineJavaScriptModelSelectFirst(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Add test data
	addScript := `models.User.add({name: "Alice", email: "alice@example.com", age: 25})`
	_, err := engine.Execute(addScript)
	if err != nil {
		t.Fatalf("Failed to add test user: %v", err)
	}

	// Test selecting first user
	firstScript := `models.User.select().first()`
	result, err := engine.Execute(firstScript)
	if err != nil {
		t.Fatalf("Failed to execute first script: %v", err)
	}

	if result == nil {
		t.Error("Expected map result, got nil")
		return
	}

	userData := result.(map[string]interface{})
	if userData["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", userData["name"])
	}
}

func TestEngineJavaScriptModelCount(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)
	testSchema := createTestSchema()
	
	if err := engine.RegisterSchema(testSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	
	if err := engine.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Add test data
	testUsers := []string{
		`models.User.add({name: "Alice", email: "alice@example.com", age: 25})`,
		`models.User.add({name: "Bob", email: "bob@example.com", age: 30})`,
	}

	for _, userScript := range testUsers {
		_, err := engine.Execute(userScript)
		if err != nil {
			t.Fatalf("Failed to add test user: %v", err)
		}
	}

	// Test counting users
	countScript := `models.User.select().count()`
	result, err := engine.Execute(countScript)
	if err != nil {
		t.Fatalf("Failed to execute count script: %v", err)
	}

	count := result.(int64)
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestEngineJavaScriptErrorHandling(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)

	// Test syntax error
	_, err := engine.Execute("invalid javascript code {")
	if err == nil {
		t.Error("Expected error for invalid JavaScript")
	}
}

func TestEngineJavaScriptSyntaxError(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	engine := New(db)

	// Test execution without models
	_, err := engine.Execute("models.NonExistent.add({name: 'test'})")
	if err == nil {
		t.Error("Expected error for non-existent model")
	}
}