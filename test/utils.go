package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// TestDB provides utilities for testing with a real database
type TestDB struct {
	DB       types.Database
	Engine   *engine.Engine
	FilePath string
	t        *testing.T
}

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *TestDB {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	eng := engine.New(db)

	return &TestDB{
		DB:       db,
		Engine:   eng,
		FilePath: "",
		t:        t,
	}
}

// Cleanup closes the database connection
func (tdb *TestDB) Cleanup() {
	if tdb.DB != nil {
		tdb.DB.Close()
	}
}

// CreateUserSchema creates a standard user schema for testing
func (tdb *TestDB) CreateUserSchema() *schema.Schema {
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build())

	if err := tdb.Engine.RegisterSchema(userSchema); err != nil {
		tdb.t.Fatalf("Failed to register user schema: %v", err)
	}

	return userSchema
}

// CreatePostSchema creates a standard post schema for testing
func (tdb *TestDB) CreatePostSchema() *schema.Schema {
	postSchema := schema.New("Post").
		WithTableName("posts").
		AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build()).
		AddField(schema.NewField("content").String().Build()).
		AddField(schema.NewField("user_id").Int().Build()).
		AddField(schema.NewField("created_at").DateTime().Default("CURRENT_TIMESTAMP").Build())

	if err := tdb.Engine.RegisterSchema(postSchema); err != nil {
		tdb.t.Fatalf("Failed to register post schema: %v", err)
	}

	return postSchema
}

// AddSampleUsers adds sample users to the database
func (tdb *TestDB) AddSampleUsers() []int64 {
	users := []map[string]interface{}{
		{"name": "Alice", "email": "alice@example.com", "age": 25},
		{"name": "Bob", "email": "bob@example.com", "age": 30},
		{"name": "Charlie", "email": "charlie@example.com", "age": 35},
	}

	var ids []int64
	for _, user := range users {
		id, err := tdb.DB.Insert("users", user)
		if err != nil {
			tdb.t.Fatalf("Failed to insert sample user: %v", err)
		}
		ids = append(ids, id)
	}

	return ids
}

// AddSamplePosts adds sample posts to the database
func (tdb *TestDB) AddSamplePosts(userIDs []int64) []int64 {
	if len(userIDs) == 0 {
		tdb.t.Fatal("Need at least one user ID to create posts")
	}

	posts := []map[string]interface{}{
		{"title": "First Post", "content": "Hello World!", "user_id": userIDs[0]},
		{"title": "Second Post", "content": "Another post", "user_id": userIDs[1]},
		{"title": "Third Post", "content": "Yet another post", "user_id": userIDs[0]},
	}

	var ids []int64
	for _, post := range posts {
		id, err := tdb.DB.Insert("posts", post)
		if err != nil {
			tdb.t.Fatalf("Failed to insert sample post: %v", err)
		}
		ids = append(ids, id)
	}

	return ids
}

// ExecuteJS executes JavaScript code and returns the result
func (tdb *TestDB) ExecuteJS(script string) (interface{}, error) {
	return tdb.Engine.Execute(script)
}

// AssertJSResult executes JavaScript and asserts the result
func (tdb *TestDB) AssertJSResult(script string, expected interface{}) {
	result, err := tdb.ExecuteJS(script)
	if err != nil {
		tdb.t.Fatalf("Failed to execute script '%s': %v", script, err)
	}

	if result != expected {
		tdb.t.Errorf("Script '%s' returned %v, expected %v", script, result, expected)
	}
}

// AssertJSError executes JavaScript and asserts it returns an error
func (tdb *TestDB) AssertJSError(script string) {
	_, err := tdb.ExecuteJS(script)
	if err == nil {
		tdb.t.Errorf("Expected script '%s' to return an error, but it succeeded", script)
	}
}

// CountRecords counts records in a table
func (tdb *TestDB) CountRecords(tableName string) int64 {
	results, err := tdb.DB.Find(tableName, nil, 0, 0)
	if err != nil {
		tdb.t.Fatalf("Failed to count records in %s: %v", tableName, err)
	}
	return int64(len(results))
}


// AssertNoError is a helper to assert no error occurred
func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError is a helper to assert an error occurred
func AssertError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
}

// AssertEqual is a helper to assert two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual is a helper to assert two values are not equal
func AssertNotEqual(t *testing.T, expected, actual interface{}) {
	if expected == actual {
		t.Errorf("Expected values to be different, both were %v", expected)
	}
}

// AssertContains is a helper to assert a slice contains a value
func AssertContains(t *testing.T, slice []interface{}, value interface{}) {
	for _, item := range slice {
		if item == value {
			return
		}
	}
	t.Errorf("Expected slice to contain %v, but it didn't", value)
}

// AssertMapContains is a helper to assert a map contains a key-value pair
func AssertMapContains(t *testing.T, m map[string]interface{}, key string, expectedValue interface{}) {
	actualValue, exists := m[key]
	if !exists {
		t.Errorf("Expected map to contain key %s", key)
		return
	}
	if actualValue != expectedValue {
		t.Errorf("Expected map[%s] to be %v, got %v", key, expectedValue, actualValue)
	}
}
