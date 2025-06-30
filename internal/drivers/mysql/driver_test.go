package drivers

import (
	"testing"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func setupMySQLDB(t *testing.T) *MySQLDB {
	config := types.Config{
		Type:     types.MySQL,
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

func TestMySQLCreateTable(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	schema := schema.New("test_table").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build()).
		AddField(schema.NewField("data").JSON().Nullable().Build()).
		AddField(schema.NewField("created_at").DateTime().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Errorf("Failed to create table: %v", err)
	}

	// Clean up
	db.DropTable(schema.TableName)
}

func TestMySQLInsert(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	schema := schema.New("test_users").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropTable(schema.TableName)

	data := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	id, err := db.Insert(schema.TableName, data)
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

	schema := schema.New("test_users").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropTable(schema.TableName)

	data := map[string]interface{}{
		"name":  "Jane Doe",
		"email": "jane@example.com",
	}

	id, err := db.Insert(schema.TableName, data)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	result, err := db.FindByID(schema.TableName, id)
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

	schema := schema.New("test_users").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropTable(schema.TableName)

	// Insert test data
	testData := []map[string]interface{}{
		{"name": "Alice", "age": 25},
		{"name": "Bob", "age": 30},
		{"name": "Charlie", "age": 25},
	}

	for _, data := range testData {
		_, err := db.Insert(schema.TableName, data)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Find with conditions
	results, err := db.Find(schema.TableName, map[string]interface{}{
		"age": 25,
	}, 0, 0)

	if err != nil {
		t.Errorf("Failed to find records: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 records, got %d", len(results))
	}

	// Find with limit
	results, err = db.Find(schema.TableName, nil, 2, 0)
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

	schema := schema.New("test_users").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropTable(schema.TableName)

	data := map[string]interface{}{
		"name": "UpdateTest",
		"age":  20,
	}

	id, err := db.Insert(schema.TableName, data)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Update the record
	updateData := map[string]interface{}{
		"age": 25,
	}

	err = db.Update(schema.TableName, id, updateData)
	if err != nil {
		t.Errorf("Failed to update data: %v", err)
	}

	// Verify update
	result, err := db.FindByID(schema.TableName, id)
	if err != nil {
		t.Fatalf("Failed to find updated record: %v", err)
	}

	age := result["age"].(int64)
	if age != 25 {
		t.Errorf("Expected age 25, got %d", age)
	}
}

func TestMySQLDelete(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	schema := schema.New("test_users").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropTable(schema.TableName)

	data := map[string]interface{}{
		"name": "DeleteTest",
	}

	id, err := db.Insert(schema.TableName, data)
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Delete the record
	err = db.Delete(schema.TableName, id)
	if err != nil {
		t.Errorf("Failed to delete data: %v", err)
	}

	// Verify deletion
	_, err = db.FindByID(schema.TableName, id)
	if err == nil {
		t.Error("Expected error when finding deleted record")
	}
}

func TestMySQLTransaction(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	schema := schema.New("test_users").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	defer db.DropTable(schema.TableName)

	t.Run("Successful transaction", func(t *testing.T) {
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}

		data := map[string]interface{}{
			"name": "TxTest",
		}

		id, err := tx.Insert(schema.TableName, data)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert in transaction: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify data was committed
		result, err := db.FindByID(schema.TableName, id)
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

		_, err = tx.Insert(schema.TableName, data)
		if err != nil {
			tx.Rollback()
			t.Fatalf("Failed to insert in transaction: %v", err)
		}

		// Rollback the transaction
		err = tx.Rollback()
		if err != nil {
			t.Fatalf("Failed to rollback transaction: %v", err)
		}

		// Verify data was not committed
		results, err := db.Find(schema.TableName, map[string]interface{}{
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

func TestMySQLDropTable(t *testing.T) {
	db := setupMySQLDB(t)
	defer db.Close()

	schema := schema.New("test_drop").
		AddField(schema.NewField("id").Int().PrimaryKey().Build())

	err := db.CreateTable(schema)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	err = db.DropTable(schema.TableName)
	if err != nil {
		t.Errorf("Failed to drop table: %v", err)
	}

	// Try to create the same table again - should succeed if dropped
	err = db.CreateTable(schema)
	if err != nil {
		t.Errorf("Failed to recreate table after drop: %v", err)
	}

	// Clean up
	db.DropTable(schema.TableName)
}