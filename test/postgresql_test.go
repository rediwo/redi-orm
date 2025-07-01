package test

import (
	"testing"
	"time"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func setupPostgreSQLDB(t *testing.T) types.Database {
	config := types.Config{
		Type:     "postgresql",
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	db, err := database.New(config)
	if err != nil {
		t.Skipf("Failed to create PostgreSQL database: %v (Docker might not be running)", err)
	}

	if err := db.Connect(); err != nil {
		t.Skipf("Failed to connect to PostgreSQL: %v (Docker might not be running)", err)
	}

	return db
}

func TestPostgreSQLConnection(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	if err := db.Close(); err != nil {
		t.Errorf("Failed to close PostgreSQL connection: %v", err)
	}
}

func TestPostgreSQLCreateModel(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLCreateModelTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("created_at").DateTime().Build())

	err := db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}

	// Clean up
	db.DropModel(modelName)
}

func TestPostgreSQLInsert(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLInsertTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build())

	// Register schema first
	err := db.RegisterSchema(modelName, userSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}
	defer db.DropModel(modelName)

	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	// Use model-based Insert operation
	id, err := db.Insert(modelName, data)
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}
}

func TestPostgreSQLFindByID(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLFindByIDTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build())

	// Register schema first
	err := db.RegisterSchema(modelName, userSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}
	defer db.DropModel(modelName)

	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	// Use model-based Insert operation
	id, err := db.Insert(modelName, data)
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
	}

	// Use model-based FindByID operation
	result, err := db.FindByID(modelName, id)
	if err != nil {
		t.Errorf("Failed to find by ID: %v", err)
	}

	if result["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", result["name"])
	}
}

func TestPostgreSQLFind(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLFindTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Build())

	// Register schema first
	err := db.RegisterSchema(modelName, userSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}
	defer db.DropModel(modelName)

	testData := []map[string]interface{}{
		{"name": "John", "age": 25},
		{"name": "Jane", "age": 30},
		{"name": "Bob", "age": 25},
	}

	for _, data := range testData {
		_, err := db.Insert(modelName, data)
		if err != nil {
			t.Errorf("Failed to insert test data: %v", err)
		}
	}

	results, err := db.Find(modelName, map[string]interface{}{
		"age": 25,
	}, 0, 0)

	if err != nil {
		t.Errorf("Failed to find records: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}
}

func TestPostgreSQLUpdate(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLUpdateTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Build())

	// Register schema first
	err := db.RegisterSchema(modelName, userSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}
	defer db.DropModel(modelName)

	data := map[string]interface{}{
		"name": "John Doe",
		"age":  25,
	}

	id, err := db.Insert(modelName, data)
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
	}

	updateData := map[string]interface{}{
		"age": 30,
	}

	err = db.Update(modelName, id, updateData)
	if err != nil {
		t.Errorf("Failed to update data: %v", err)
	}

	result, err := db.FindByID(modelName, id)
	if err != nil {
		t.Errorf("Failed to find updated record: %v", err)
	}

	age, ok := result["age"].(int32)
	if !ok {
		age64, ok := result["age"].(int64)
		if !ok {
			t.Errorf("Age is not int32 or int64, got %T: %v", result["age"], result["age"])
		} else {
			age = int32(age64)
		}
	}

	if age != 30 {
		t.Errorf("Expected age 30, got %d", age)
	}
}

func TestPostgreSQLDelete(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLDeleteTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	// Register schema first
	err := db.RegisterSchema(modelName, userSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}
	defer db.DropModel(modelName)

	data := map[string]interface{}{
		"name": "John Doe",
	}

	id, err := db.Insert(modelName, data)
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
	}

	err = db.Delete(modelName, id)
	if err != nil {
		t.Errorf("Failed to delete data: %v", err)
	}

	_, err = db.FindByID(modelName, id)
	if err == nil {
		t.Errorf("Expected error when finding deleted record, but got none")
	}
}

func TestPostgreSQLTransaction(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLTransactionTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	// Register schema first
	err := db.RegisterSchema(modelName, userSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}
	defer db.DropModel(modelName)

	tx, err := db.Begin()
	if err != nil {
		t.Errorf("Failed to begin transaction: %v", err)
	}

	data := map[string]interface{}{
		"name": "John Doe",
	}

	id, err := tx.Insert(userSchema.TableName, data)
	if err != nil {
		tx.Rollback()
		t.Errorf("Failed to insert in transaction: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}

	result, err := db.FindByID(modelName, id)
	if err != nil {
		t.Errorf("Failed to find committed record: %v", err)
	}

	if result["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", result["name"])
	}
}

func TestPostgreSQLQueryBuilder(t *testing.T) {
	db := setupPostgreSQLDB(t)
	defer db.Close()

	// Use unique model name for this test
	modelName := "PostgreSQLQueryBuilderTest"
	userSchema := schema.New(modelName).
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Build())

	// Register schema first
	err := db.RegisterSchema(modelName, userSchema)
	if err != nil {
		t.Errorf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(userSchema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}
	defer db.DropModel(modelName)

	testData := []map[string]interface{}{
		{"name": "Alice", "age": 25},
		{"name": "Bob", "age": 30},
		{"name": "Charlie", "age": 35},
	}

	for _, data := range testData {
		_, err := db.Insert(modelName, data)
		if err != nil {
			t.Errorf("Failed to insert test data: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	qb := db.Select(modelName, []string{"name", "age"})
	results, err := qb.Where("age", ">", 25).OrderBy("age", "ASC").Execute()
	if err != nil {
		t.Errorf("Failed to execute query: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0]["name"] != "Bob" {
		t.Errorf("Expected first result name 'Bob', got %v", results[0]["name"])
	}
}
