package engine

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// Mock database for testing
type mockDB struct {
	data   map[string][]map[string]interface{}
	nextID int64
	tables map[string]*schema.Schema
}

func newMockDB() *mockDB {
	return &mockDB{
		data:   make(map[string][]map[string]interface{}),
		nextID: 1,
		tables: make(map[string]*schema.Schema),
	}
}

func (m *mockDB) Connect() error { return nil }
func (m *mockDB) Close() error   { return nil }

func (m *mockDB) CreateTable(s interface{}) error {
	if schema, ok := s.(*schema.Schema); ok {
		m.tables[schema.TableName] = schema
	}
	return nil
}

func (m *mockDB) DropTable(tableName string) error {
	delete(m.data, tableName)
	delete(m.tables, tableName)
	return nil
}

func (m *mockDB) Insert(tableName string, data map[string]interface{}) (int64, error) {
	if m.data[tableName] == nil {
		m.data[tableName] = []map[string]interface{}{}
	}

	newData := make(map[string]interface{})
	for k, v := range data {
		newData[k] = v
	}
	newData["id"] = m.nextID

	m.data[tableName] = append(m.data[tableName], newData)
	currentID := m.nextID
	m.nextID++

	return currentID, nil
}

func (m *mockDB) FindByID(tableName string, id interface{}) (map[string]interface{}, error) {
	records, exists := m.data[tableName]
	if !exists {
		return nil, nil
	}

	for _, record := range records {
		if record["id"] == id {
			return record, nil
		}
	}

	return nil, nil
}

func (m *mockDB) Find(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	records, exists := m.data[tableName]
	if !exists {
		return []map[string]interface{}{}, nil
	}

	var filtered []map[string]interface{}
	for _, record := range records {
		match := true
		for key, value := range conditions {
			recordValue := record[key]

			// Handle type conversion for numbers (JavaScript numbers vs Go int/int64)
			if rv, ok := recordValue.(int); ok {
				if fv, ok := value.(float64); ok {
					if float64(rv) != fv {
						match = false
						break
					}
					continue
				}
				if iv, ok := value.(int64); ok {
					if int64(rv) != iv {
						match = false
						break
					}
					continue
				}
			}

			if record[key] != value {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, record)
		}
	}

	start := offset
	if start >= len(filtered) {
		return []map[string]interface{}{}, nil
	}

	end := len(filtered)
	if limit > 0 && start+limit < end {
		end = start + limit
	}

	return filtered[start:end], nil
}

func (m *mockDB) Update(tableName string, id interface{}, data map[string]interface{}) error {
	records, exists := m.data[tableName]
	if !exists {
		return nil
	}

	for i, record := range records {
		if record["id"] == id {
			for key, value := range data {
				m.data[tableName][i][key] = value
			}
			break
		}
	}

	return nil
}

func (m *mockDB) Delete(tableName string, id interface{}) error {
	records, exists := m.data[tableName]
	if !exists {
		return nil
	}

	for i, record := range records {
		if record["id"] == id {
			m.data[tableName] = append(records[:i], records[i+1:]...)
			break
		}
	}

	return nil
}

func (m *mockDB) Select(tableName string, columns []string) types.QueryBuilder {
	return &mockQueryBuilder{
		db:        m,
		tableName: tableName,
		columns:   columns,
	}
}

func (m *mockDB) Begin() (types.Transaction, error) {
	return nil, nil
}

func (m *mockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *mockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *mockDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

type mockQueryBuilder struct {
	db         *mockDB
	tableName  string
	columns    []string
	conditions map[string]interface{}
	limit      int
	offset     int
}

func (q *mockQueryBuilder) Where(field string, operator string, value interface{}) types.QueryBuilder {
	if q.conditions == nil {
		q.conditions = make(map[string]interface{})
	}
	// For simplicity in mock, we only store exact matches
	// In real implementation, we'd handle different operators
	if operator == "=" {
		q.conditions[field] = value
	}
	return q
}

func (q *mockQueryBuilder) WhereIn(field string, values []interface{}) types.QueryBuilder {
	if len(values) > 0 {
		q.Where(field, "=", values[0])
	}
	return q
}

func (q *mockQueryBuilder) OrderBy(field string, direction string) types.QueryBuilder {
	return q
}

func (q *mockQueryBuilder) Limit(limit int) types.QueryBuilder {
	q.limit = limit
	return q
}

func (q *mockQueryBuilder) Offset(offset int) types.QueryBuilder {
	q.offset = offset
	return q
}

func (q *mockQueryBuilder) Execute() ([]map[string]interface{}, error) {
	return q.db.Find(q.tableName, q.conditions, q.limit, q.offset)
}

func (q *mockQueryBuilder) First() (map[string]interface{}, error) {
	results, err := q.db.Find(q.tableName, q.conditions, 1, q.offset)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

func (q *mockQueryBuilder) Count() (int64, error) {
	results, err := q.db.Find(q.tableName, q.conditions, 0, 0)
	if err != nil {
		return 0, err
	}
	return int64(len(results)), nil
}

func createTestSchema() *schema.Schema {
	return schema.New("User").
		AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build())
}

func TestEngineNew(t *testing.T) {
	db := newMockDB()
	engine := New(db)

	if engine.vm == nil {
		t.Error("Expected JavaScript VM to be initialized")
	}

	if engine.db == nil {
		t.Error("Expected database to be set")
	}

	if engine.schemas == nil {
		t.Error("Expected schemas map to be initialized")
	}

	if engine.models == nil {
		t.Error("Expected models map to be initialized")
	}
}

func TestEngineRegisterSchema(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()

	err := engine.RegisterSchema(testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Check that schema was registered
	if _, exists := engine.schemas["User"]; !exists {
		t.Error("Schema was not registered")
	}

	// Check that model was created
	if _, exists := engine.models["User"]; !exists {
		t.Error("Model was not created")
	}

	// Check that model is available in JavaScript
	result, err := engine.Execute("typeof models.User")
	if err != nil {
		t.Fatalf("Failed to check JavaScript model: %v", err)
	}
	if result != "object" {
		t.Errorf("Expected models.User to be an object, got %v", result)
	}
}

func TestEngineRegisterInvalidSchema(t *testing.T) {
	db := newMockDB()
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
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

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
	userData, _ := db.FindByID("users", int64(1))
	if userData["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", userData["name"])
	}
}

func TestEngineJavaScriptModelGet(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Add test data
	testData := map[string]interface{}{
		"name":  "Jane Doe",
		"email": "jane@example.com",
		"age":   25,
	}
	id, _ := db.Insert("users", testData)

	// Test getting user via JavaScript
	script := fmt.Sprintf(`models.User.get(%d)`, id)
	result, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute get script: %v", err)
	}

	// Should return the user data
	userData, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if userData["name"] != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got %v", userData["name"])
	}
}

func TestEngineJavaScriptModelSet(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Add test data
	testData := map[string]interface{}{
		"name":  "Original Name",
		"email": "original@example.com",
		"age":   20,
	}
	id, _ := db.Insert("users", testData)

	// Test updating user via JavaScript
	script := fmt.Sprintf(`models.User.set(%d, {name: "Updated Name", age: 21})`, id)
	_, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute set script: %v", err)
	}

	// Verify update
	userData, _ := db.FindByID("users", id)
	if userData["name"] != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %v", userData["name"])
	}
	if userData["age"] != int64(21) {
		t.Errorf("Expected age 21, got %v", userData["age"])
	}
}

func TestEngineJavaScriptModelRemove(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Add test data
	testData := map[string]interface{}{
		"name":  "To Delete",
		"email": "delete@example.com",
		"age":   30,
	}
	id, _ := db.Insert("users", testData)

	// Test removing user via JavaScript
	script := fmt.Sprintf(`models.User.remove(%d)`, id)
	_, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute remove script: %v", err)
	}

	// Verify removal
	userData, _ := db.FindByID("users", id)
	if userData != nil {
		t.Error("Expected user to be deleted")
	}
}

func TestEngineJavaScriptModelSelect(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Add test data
	users := []map[string]interface{}{
		{"name": "User1", "email": "user1@example.com", "age": 20},
		{"name": "User2", "email": "user2@example.com", "age": 25},
		{"name": "User3", "email": "user3@example.com", "age": 30},
	}

	for _, user := range users {
		db.Insert("users", user)
	}

	// Test select all
	script := `models.User.select().execute()`
	result, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute select script: %v", err)
	}

	results, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected array result, got %T", result)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
}

func TestEngineJavaScriptModelSelectWithWhere(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Add test data
	users := []map[string]interface{}{
		{"name": "User1", "email": "user1@example.com", "age": 25},
		{"name": "User2", "email": "user2@example.com", "age": 25},
		{"name": "User3", "email": "user3@example.com", "age": 30},
	}

	for _, user := range users {
		db.Insert("users", user)
	}

	// Test select with where
	script := `models.User.select().where("age", "=", 25).execute()`
	result, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute select with where script: %v", err)
	}

	results, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected array result, got %T", result)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestEngineJavaScriptModelSelectFirst(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Add test data
	testData := map[string]interface{}{
		"name":  "First User",
		"email": "first@example.com",
		"age":   25,
	}
	db.Insert("users", testData)

	// Test select first
	script := `models.User.select().first()`
	result, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute select first script: %v", err)
	}

	userData, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if userData["name"] != "First User" {
		t.Errorf("Expected name 'First User', got %v", userData["name"])
	}
}

func TestEngineJavaScriptModelCount(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Add test data
	users := []map[string]interface{}{
		{"name": "User1", "email": "user1@example.com", "age": 20},
		{"name": "User2", "email": "user2@example.com", "age": 25},
	}

	for _, user := range users {
		db.Insert("users", user)
	}

	// Test count
	script := `models.User.select().count()`
	result, err := engine.Execute(script)
	if err != nil {
		t.Fatalf("Failed to execute count script: %v", err)
	}

	count, ok := result.(int64)
	if !ok {
		t.Fatalf("Expected int64 result, got %T", result)
	}

	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestEngineJavaScriptErrorHandling(t *testing.T) {
	db := newMockDB()
	engine := New(db)
	testSchema := createTestSchema()
	engine.RegisterSchema(testSchema)

	// Test error for missing arguments
	script := `models.User.get()`
	_, err := engine.Execute(script)
	if err == nil {
		t.Error("Expected error for missing arguments")
	}

	// Test error for invalid data type
	script2 := `models.User.add("not an object")`
	_, err2 := engine.Execute(script2)
	if err2 == nil {
		t.Error("Expected error for invalid data type")
	}
}

func TestEngineJavaScriptSyntaxError(t *testing.T) {
	db := newMockDB()
	engine := New(db)

	// Test syntax error
	script := `models.User.add({name: "test" // missing closing brace`
	_, err := engine.Execute(script)
	if err == nil {
		t.Error("Expected syntax error")
	}

	if !strings.Contains(err.Error(), "SyntaxError") {
		t.Errorf("Expected SyntaxError, got %v", err)
	}
}
