package drivers

import (
	"testing"
)

func TestQueryBuilder(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	createTestTable(t, db)

	// Insert test data
	users := []map[string]interface{}{
		{"name": "Alice", "email": "alice@example.com", "age": 25},
		{"name": "Bob", "email": "bob@example.com", "age": 30},
		{"name": "Charlie", "email": "charlie@example.com", "age": 35},
		{"name": "David", "email": "david@example.com", "age": 25},
	}

	for _, user := range users {
		db.RawInsert("users", user)
	}

	t.Run("Simple select", func(t *testing.T) {
		qb := db.RawSelect("users", []string{"name", "email"})
		results, err := qb.Execute()
		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}
		if len(results) != 4 {
			t.Errorf("Expected 4 results, got %d", len(results))
		}
	})

	t.Run("Where clause", func(t *testing.T) {
		qb := db.RawSelect("users", nil).Where("age", "=", 25)
		results, err := qb.Execute()
		if err != nil {
			t.Fatalf("Failed to execute query with where: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results with age=25, got %d", len(results))
		}
	})

	t.Run("Multiple where clauses", func(t *testing.T) {
		qb := db.RawSelect("users", nil).
			Where("age", ">", 25).
			Where("name", "!=", "Charlie")
		results, err := qb.Execute()
		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if results[0]["name"] != "Bob" {
			t.Errorf("Expected Bob, got %v", results[0]["name"])
		}
	})

	t.Run("WhereIn", func(t *testing.T) {
		qb := db.RawSelect("users", nil).WhereIn("name", []interface{}{"Alice", "Bob"})
		results, err := qb.Execute()
		if err != nil {
			t.Fatalf("Failed to execute query with WhereIn: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	t.Run("OrderBy", func(t *testing.T) {
		qb := db.RawSelect("users", []string{"name"}).OrderBy("name", "DESC")
		results, err := qb.Execute()
		if err != nil {
			t.Fatalf("Failed to execute query with OrderBy: %v", err)
		}
		if results[0]["name"] != "David" {
			t.Errorf("Expected first result to be David, got %v", results[0]["name"])
		}
		if results[3]["name"] != "Alice" {
			t.Errorf("Expected last result to be Alice, got %v", results[3]["name"])
		}
	})

	t.Run("Limit and Offset", func(t *testing.T) {
		qb := db.RawSelect("users", nil).OrderBy("name", "ASC").Limit(2).Offset(1)
		results, err := qb.Execute()
		if err != nil {
			t.Fatalf("Failed to execute query with Limit/Offset: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results with limit, got %d", len(results))
		}
		if results[0]["name"] != "Bob" {
			t.Errorf("Expected first result to be Bob (offset 1), got %v", results[0]["name"])
		}
	})

	t.Run("First", func(t *testing.T) {
		qb := db.RawSelect("users", nil).Where("age", "=", 30)
		result, err := qb.First()
		if err != nil {
			t.Fatalf("Failed to get first result: %v", err)
		}
		if result["name"] != "Bob" {
			t.Errorf("Expected Bob, got %v", result["name"])
		}

		// Test First with no results
		qb2 := db.RawSelect("users", nil).Where("age", "=", 99)
		_, err = qb2.First()
		if err == nil {
			t.Error("Expected error for no results")
		}
	})

	t.Run("Count", func(t *testing.T) {
		qb := db.RawSelect("users", nil)
		count, err := qb.Count()
		if err != nil {
			t.Fatalf("Failed to count: %v", err)
		}
		if count != 4 {
			t.Errorf("Expected count of 4, got %d", count)
		}

		// Count with where clause
		qb2 := db.RawSelect("users", nil).Where("age", ">=", 30)
		count2, err := qb2.Count()
		if err != nil {
			t.Fatalf("Failed to count with where: %v", err)
		}
		if count2 != 2 {
			t.Errorf("Expected count of 2, got %d", count2)
		}
	})

	t.Run("Complex query", func(t *testing.T) {
		qb := db.RawSelect("users", []string{"name", "age"}).
			Where("age", ">=", 25).
			Where("age", "<=", 30).
			OrderBy("age", "ASC").
			Limit(10)

		results, err := qb.Execute()
		if err != nil {
			t.Fatalf("Failed to execute complex query: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Verify order
		ages := []int{
			int(results[0]["age"].(int64)),
			int(results[1]["age"].(int64)),
			int(results[2]["age"].(int64)),
		}
		if ages[0] > ages[1] || ages[1] > ages[2] {
			t.Error("Results not ordered by age ascending")
		}
	})
}
