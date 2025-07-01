package test

import (
	"database/sql"
	"os"
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

// SetupTestDB creates a temporary SQLite database for testing
func SetupTestDB(t *testing.T) *TestDB {
	tempFile := "test_" + t.Name() + ".db"

	db, err := database.New(database.Config{
		Type:     "sqlite",
		FilePath: tempFile,
	})
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
		FilePath: tempFile,
		t:        t,
	}
}

// Cleanup removes the temporary database file
func (tdb *TestDB) Cleanup() {
	if tdb.DB != nil {
		tdb.DB.Close()
	}
	if tdb.FilePath != "" {
		os.Remove(tdb.FilePath)
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

// MockDatabase provides a mock database implementation for unit tests
type MockDatabase struct {
	data   map[string][]map[string]interface{}
	nextID int64
	tables map[string]*schema.Schema
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		data:   make(map[string][]map[string]interface{}),
		nextID: 1,
		tables: make(map[string]*schema.Schema),
	}
}

func (m *MockDatabase) Connect() error { return nil }
func (m *MockDatabase) Close() error   { return nil }

func (m *MockDatabase) CreateTable(schema *schema.Schema) error {
	m.tables[schema.TableName] = schema
	return nil
}

func (m *MockDatabase) DropTable(tableName string) error {
	delete(m.data, tableName)
	delete(m.tables, tableName)
	return nil
}

func (m *MockDatabase) Insert(tableName string, data map[string]interface{}) (int64, error) {
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

func (m *MockDatabase) FindByID(tableName string, id interface{}) (map[string]interface{}, error) {
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

func (m *MockDatabase) Find(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	records, exists := m.data[tableName]
	if !exists {
		return []map[string]interface{}{}, nil
	}

	var filtered []map[string]interface{}
	for _, record := range records {
		match := true
		for key, value := range conditions {
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

func (m *MockDatabase) Update(tableName string, id interface{}, data map[string]interface{}) error {
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

func (m *MockDatabase) Delete(tableName string, id interface{}) error {
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

func (m *MockDatabase) Select(tableName string, columns []string) types.QueryBuilder {
	return &MockQueryBuilder{
		db:        m,
		tableName: tableName,
		columns:   columns,
	}
}

func (m *MockDatabase) Begin() (types.Transaction, error) {
	return nil, nil
}

func (m *MockDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *MockDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *MockDatabase) GetMigrator() types.DatabaseMigrator {
	return nil
}

func (m *MockDatabase) EnsureSchema() error {
	return nil
}

// RegisterSchema registers a schema for model name resolution
func (m *MockDatabase) RegisterSchema(modelName string, schema interface{}) error {
	return nil
}

// GetRegisteredSchemas returns all registered schemas
func (m *MockDatabase) GetRegisteredSchemas() map[string]interface{} {
	return make(map[string]interface{})
}

// Raw operations (mock implementations)
func (m *MockDatabase) RawInsert(tableName string, data map[string]interface{}) (int64, error) {
	return m.Insert(tableName, data)
}

func (m *MockDatabase) RawFindByID(tableName string, id interface{}) (map[string]interface{}, error) {
	return m.FindByID(tableName, id)
}

func (m *MockDatabase) RawFind(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	return m.Find(tableName, conditions, limit, offset)
}

func (m *MockDatabase) RawUpdate(tableName string, id interface{}, data map[string]interface{}) error {
	return m.Update(tableName, id, data)
}

func (m *MockDatabase) RawDelete(tableName string, id interface{}) error {
	return m.Delete(tableName, id)
}

func (m *MockDatabase) RawSelect(tableName string, columns []string) types.QueryBuilder {
	return m.Select(tableName, columns)
}

// MockQueryBuilder provides a mock query builder for testing
type MockQueryBuilder struct {
	db         *MockDatabase
	tableName  string
	columns    []string
	conditions map[string]interface{}
	limit      int
	offset     int
}

func (q *MockQueryBuilder) Where(field string, operator string, value interface{}) types.QueryBuilder {
	if q.conditions == nil {
		q.conditions = make(map[string]interface{})
	}
	q.conditions[field] = value
	return q
}

func (q *MockQueryBuilder) WhereIn(field string, values []interface{}) types.QueryBuilder {
	if len(values) > 0 {
		q.Where(field, "=", values[0])
	}
	return q
}

func (q *MockQueryBuilder) OrderBy(field string, direction string) types.QueryBuilder {
	return q
}

func (q *MockQueryBuilder) Limit(limit int) types.QueryBuilder {
	q.limit = limit
	return q
}

func (q *MockQueryBuilder) Offset(offset int) types.QueryBuilder {
	q.offset = offset
	return q
}

func (q *MockQueryBuilder) Execute() ([]map[string]interface{}, error) {
	return q.db.Find(q.tableName, q.conditions, q.limit, q.offset)
}

func (q *MockQueryBuilder) First() (map[string]interface{}, error) {
	results, err := q.db.Find(q.tableName, q.conditions, 1, q.offset)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

func (q *MockQueryBuilder) Count() (int64, error) {
	results, err := q.db.Find(q.tableName, q.conditions, 0, 0)
	if err != nil {
		return 0, err
	}
	return int64(len(results)), nil
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
