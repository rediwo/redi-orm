package drivers

import (
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func setupMySQLDB(t *testing.T) *MySQLDB {
	config := types.Config{
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	db, err := NewMySQLDB(config)
	if err != nil {
		t.Fatalf("Failed to create MySQL database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Skipf("Failed to connect to MySQL: %v (Docker might not be running)", err)
	}

	return db
}

func TestMySQLConnect(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()
}

func TestMySQLCreateModel(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Use a unique table name for this test
	schema := schema.New("create_model_test_table").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build()).
		AddField(schema.NewField("data").JSON().Nullable().Build()).
		AddField(schema.NewField("created_at").DateTime().Build())

	err := db.CreateModel(schema)
	if err != nil {
		t.Errorf("Failed to create model: %v", err)
	}

	// Clean up
	db.DropModel("create_model_test_table")
}

func TestMySQLInsert(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create a unique schema for this test
	testSchema := &schema.Schema{
		Name:      "InsertTestUser",
		TableName: "insert_test_users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString, Nullable: false},
			{Name: "email", Type: schema.FieldTypeString, Unique: true},
			{Name: "age", Type: schema.FieldTypeInt, Nullable: true},
		},
	}

	// Register schema with model name (not table name)
	err := db.RegisterSchema("InsertTestUser", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	defer db.DropModel("InsertTestUser")

	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	}

	// Use model-based Insert method
	id, err := db.Insert("InsertTestUser", data)
	if err != nil {
		t.Errorf("Failed to insert data: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}
}

func TestMySQLFindByID(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create a unique schema for this test
	testSchema := &schema.Schema{
		Name:      "FindByIDTestUser",
		TableName: "findbyid_test_users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString, Nullable: false},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "age", Type: schema.FieldTypeInt, Nullable: true},
		},
	}

	// Register schema with model name
	err := db.RegisterSchema("FindByIDTestUser", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	defer db.DropModel("FindByIDTestUser")

	data := map[string]interface{}{
		"name":  "Jane Doe",
		"email": "jane@example.com",
		"age":   25,
	}

	// Use model-based Insert method
	id, err := db.Insert("FindByIDTestUser", data)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Use model-based FindByID method
	result, err := db.FindByID("FindByIDTestUser", id)
	if err != nil {
		t.Errorf("Failed to find by ID: %v", err)
	}

	if result["name"] != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got %v", result["name"])
	}
}

func TestMySQLFind(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create a unique schema for this test
	testSchema := &schema.Schema{
		Name:      "FindTestUser",
		TableName: "find_test_users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString, Nullable: false},
			{Name: "age", Type: schema.FieldTypeInt, Nullable: false},
		},
	}

	// Register schema with model name
	err := db.RegisterSchema("FindTestUser", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	defer db.DropModel("FindTestUser")

	// Insert test data
	testData := []map[string]interface{}{
		{"name": "Alice", "age": 25},
		{"name": "Bob", "age": 30},
		{"name": "Charlie", "age": 25},
	}

	for _, data := range testData {
		_, err := db.Insert("FindTestUser", data)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Find with conditions using model-based method
	results, err := db.Find("FindTestUser", map[string]interface{}{
		"age": 25,
	}, 0, 0)

	if err != nil {
		t.Errorf("Failed to find records: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}

	// Find with limit using model-based method
	results, err = db.Find("FindTestUser", nil, 2, 0)
	if err != nil {
		t.Errorf("Failed to find with limit: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 records with limit, got %d", len(results))
	}
}

func TestMySQLUpdate(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create a unique schema for this test
	testSchema := &schema.Schema{
		Name:      "UpdateTestUser",
		TableName: "update_test_users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString, Nullable: false},
			{Name: "age", Type: schema.FieldTypeInt, Nullable: false},
		},
	}

	// Register schema with model name
	err := db.RegisterSchema("UpdateTestUser", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	defer db.DropModel("UpdateTestUser")

	data := map[string]interface{}{
		"name": "UpdateTest",
		"age":  20,
	}

	// Use model-based Insert method
	id, err := db.Insert("UpdateTestUser", data)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Update the record using model-based method
	updateData := map[string]interface{}{
		"age": 25,
	}

	err = db.Update("UpdateTestUser", id, updateData)
	if err != nil {
		t.Errorf("Failed to update data: %v", err)
	}

	// Verify update using model-based method
	result, err := db.FindByID("UpdateTestUser", id)
	if err != nil {
		t.Fatalf("Failed to find updated record: %v", err)
	}

	// Handle both int and int64 types
	var age int64
	switch v := result["age"].(type) {
	case int:
		age = int64(v)
	case int64:
		age = v
	default:
		t.Fatalf("Unexpected age type: %T", v)
	}

	if age != 25 {
		t.Errorf("Expected age 25, got %d", age)
	}
}

func TestMySQLDelete(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create a unique schema for this test
	testSchema := &schema.Schema{
		Name:      "DeleteTestUser",
		TableName: "delete_test_users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString, Nullable: false},
		},
	}

	// Register schema with model name
	err := db.RegisterSchema("DeleteTestUser", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	defer db.DropModel("DeleteTestUser")

	data := map[string]interface{}{
		"name": "DeleteTest",
	}

	// Use model-based Insert method
	id, err := db.Insert("DeleteTestUser", data)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Delete the record using model-based method
	err = db.Delete("DeleteTestUser", id)
	if err != nil {
		t.Errorf("Failed to delete data: %v", err)
	}

	// Verify deletion using model-based method
	_, err = db.FindByID("DeleteTestUser", id)
	if err == nil {
		t.Error("Expected error when finding deleted record")
	}
}

func TestMySQLTransaction(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create a unique schema for this test
	testSchema := &schema.Schema{
		Name:      "TransactionTestUser",
		TableName: "transaction_test_users",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: schema.FieldTypeString, Nullable: false},
		},
	}

	// Register schema with model name
	err := db.RegisterSchema("TransactionTestUser", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	defer db.DropModel("TransactionTestUser")

	t.Run("Successful transaction", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		data := map[string]interface{}{
			"name": "TxTest",
		}

		// Use raw Insert method since transactions work with table names
		id, err := tx.Insert("transaction_test_users", data)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert in transaction: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify data was committed using model-based method
		result, err := db.FindByID("TransactionTestUser", id)
		if err != nil {
			t.Errorf("Failed to find committed record: %v", err)
		}

		if result["name"] != "TxTest" {
			t.Errorf("Expected name 'TxTest', got %v", result["name"])
		}
	})

	t.Run("Rollback transaction", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		data := map[string]interface{}{
			"name": "RollbackTest",
		}

		// Use raw Insert method since transactions work with table names
		_, err = tx.Insert("transaction_test_users", data)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert in transaction: %v", err)
		}

		// Rollback the transaction
		err = tx.Rollback()
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		// Verify data was not committed using model-based method
		results, err := db.Find("TransactionTestUser", map[string]interface{}{
			"name": "RollbackTest",
		}, 0, 0)

		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		if len(results) > 0 {
			t.Error("Found rolled back data, expected none")
		}
	})
}

func TestMySQLDropModel(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create a unique schema for this test
	testSchema := &schema.Schema{
		Name:      "DropTestModel",
		TableName: "drop_test_table",
		Fields: []schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true},
		},
	}

	// Register schema with model name
	err := db.RegisterSchema("DropTestModel", testSchema)
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	err = db.CreateModel(testSchema)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Drop using model name
	err = db.DropModel("DropTestModel")
	if err != nil {
		t.Errorf("Failed to drop model: %v", err)
	}

	// Try to create the same model again - should succeed if dropped
	err = db.CreateModel(testSchema)
	if err != nil {
		t.Errorf("Failed to recreate model after drop: %v", err)
	}

	// Clean up
	db.DropModel("DropTestModel")
}

// Test to ensure schemas are isolated between tests
func TestMySQLSchemaIsolation(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	// Create multiple unique schemas to ensure no conflicts
	for i := 0; i < 3; i++ {
		modelName := fmt.Sprintf("IsolationTestUser%d", i)
		tableName := fmt.Sprintf("isolation_test_users_%d", i)

		testSchema := &schema.Schema{
			Name:      modelName,
			TableName: tableName,
			Fields: []schema.Field{
				{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
				{Name: "name", Type: schema.FieldTypeString, Nullable: false},
				{Name: "value", Type: schema.FieldTypeInt, Nullable: false},
			},
		}

		// Register and create each schema
		err := db.RegisterSchema(modelName, testSchema)
		if err != nil {
			t.Fatalf("Failed to register schema %s: %v", modelName, err)
		}

		err = db.CreateModel(testSchema)
		if err != nil {
			t.Fatalf("Failed to create model %s: %v", modelName, err)
		}
		defer db.DropModel(modelName)

		// Insert data specific to this schema
		data := map[string]interface{}{
			"name":  fmt.Sprintf("User%d", i),
			"value": i * 10,
		}

		id, err := db.Insert(modelName, data)
		if err != nil {
			t.Fatalf("Failed to insert data for %s: %v", modelName, err)
		}

		// Verify data was inserted correctly
		result, err := db.FindByID(modelName, id)
		if err != nil {
			t.Fatalf("Failed to find data for %s: %v", modelName, err)
		}

		expectedName := fmt.Sprintf("User%d", i)
		if result["name"] != expectedName {
			t.Errorf("Expected name '%s', got %v", expectedName, result["name"])
		}

		// Verify value (handle both int and int64)
		var value int64
		switch v := result["value"].(type) {
		case int:
			value = int64(v)
		case int64:
			value = v
		default:
			t.Fatalf("Unexpected value type: %T", v)
		}

		expectedValue := int64(i * 10)
		if value != expectedValue {
			t.Errorf("Expected value %d, got %d", expectedValue, value)
		}
	}
}