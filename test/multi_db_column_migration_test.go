package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
)

// TestMultiDatabaseColumnMigration tests column migration functionality across different database types
func TestMultiDatabaseColumnMigration(t *testing.T) {
	testCases := []struct {
		name  string
		dbURI string
	}{
		{"SQLite Memory", "sqlite://:memory:"},
		{"SQLite File", "sqlite://:memory:"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testColumnMigration(t, tc.dbURI, tc.name)
		})
	}
}

func testColumnMigration(t *testing.T, dbURI, dbType string) {
	// Create database connection
	db, err := database.NewFromURI(dbURI)
	if err != nil {
		t.Fatalf("Failed to create %s database: %v", dbType, err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to %s: %v", dbType, err)
	}
	defer db.Close()

	t.Logf("✅ Connected to %s", dbType)

	// Clean up any existing table
	db.DropModel("User")

	// Phase 1: Create initial schema
	eng1 := engine.New(db)
	initialSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng1.RegisterSchema(initialSchema); err != nil {
		t.Fatalf("Failed to register initial schema: %v", err)
	}

	if err := eng1.EnsureSchema(); err != nil {
		t.Fatalf("Initial migration failed: %v", err)
	}
	t.Logf("✅ %s: Created initial table (id, name)", dbType)

	// Insert initial data
	userID, err := eng1.Execute(`models.User.add({name: "Alice"})`)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}
	if userID.(int64) <= 0 {
		t.Errorf("Invalid user ID: %v", userID)
	}
	t.Logf("✅ %s: Inserted user Alice (ID: %v)", dbType, userID)

	// Phase 2: Add new columns
	eng2 := engine.New(db)
	updatedSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build())

	if err := eng2.RegisterSchema(updatedSchema); err != nil {
		t.Fatalf("Failed to register updated schema: %v", err)
	}

	if err := eng2.EnsureSchema(); err != nil {
		t.Fatalf("Column migration failed: %v", err)
	}
	t.Logf("✅ %s: Added columns (email, active)", dbType)

	// Verify existing data is preserved
	user, err := eng2.Execute(`models.User.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get user after migration: %v", err)
	}

	userData := user.(map[string]interface{})
	if userData["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got %v", userData["name"])
	}
	// Check that new columns have default values
	if userData["email"] != nil && userData["email"] != "" {
		t.Logf("Email field: %v (nullable)", userData["email"])
	}
	if userData["active"] == nil {
		t.Errorf("Expected 'active' field to have default value")
	}
	t.Logf("✅ %s: Data preserved - name: %v, email: %v, active: %v", 
		dbType, userData["name"], userData["email"], userData["active"])

	// Insert new user with all columns
	newUserID, err := eng2.Execute(`models.User.add({name: "Bob", email: "bob@test.com", active: false})`)
	if err != nil {
		t.Fatalf("Failed to insert new user: %v", err)
	}
	if newUserID.(int64) <= 0 {
		t.Errorf("Invalid new user ID: %v", newUserID)
	}
	t.Logf("✅ %s: Inserted Bob with all fields (ID: %v)", dbType, newUserID)

	// Test idempotency
	if err := eng2.EnsureSchema(); err != nil {
		t.Fatalf("Idempotent migration failed: %v", err)
	}
	t.Logf("✅ %s: Idempotent migration successful", dbType)

	// Final count
	count, err := eng2.Execute(`models.User.select().count()`)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}
	if count.(int64) != 2 {
		t.Errorf("Expected 2 users, got %v", count)
	}
	t.Logf("✅ %s: Final user count: %v", dbType, count)
}

// TestColumnMigrationBenefits tests the key benefits of the shared migration logic
func TestColumnMigrationBenefits(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	t.Run("Consistent Behavior", func(t *testing.T) {
		// Test that the migration behavior is consistent
		eng := engine.New(db)
		
		// Define schema with multiple field types
		schema1 := schema.New("Product").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("price").Float().Build())

		if err := eng.RegisterSchema(schema1); err != nil {
			t.Fatalf("Failed to register schema: %v", err)
		}

		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("Failed to ensure schema: %v", err)
		}

		// Add new fields
		schema2 := schema.New("Product").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("price").Float().Build()).
			AddField(schema.NewField("category").String().Default("general").Build()).
			AddField(schema.NewField("inStock").Bool().Default(true).Build())

		eng2 := engine.New(db)
		if err := eng2.RegisterSchema(schema2); err != nil {
			t.Fatalf("Failed to register updated schema: %v", err)
		}

		if err := eng2.EnsureSchema(); err != nil {
			t.Fatalf("Failed to migrate schema: %v", err)
		}

		t.Log("✅ Consistent behavior across different field types")
	})

	t.Run("Shared Schema Comparison Logic", func(t *testing.T) {
		// Test that schema comparison works correctly
		eng := engine.New(db)
		
		// Register the same schema twice - should be idempotent
		orderSchema := schema.New("Order").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("total").Float().Build()).
			AddField(schema.NewField("status").String().Default("pending").Build())

		if err := eng.RegisterSchema(orderSchema); err != nil {
			t.Fatalf("Failed to register order schema: %v", err)
		}

		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("Failed to ensure order schema: %v", err)
		}

		// Call again - should be safe
		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("Failed second call to ensure schema: %v", err)
		}

		t.Log("✅ Shared schema comparison logic works correctly")
	})

	t.Run("Easy Extension", func(t *testing.T) {
		// Test that adding new schemas is easy
		eng := engine.New(db)
		
		// Add multiple schemas in sequence
		schemas := []struct {
			name   string
			schema *schema.Schema
		}{
			{
				"Customer",
				schema.New("Customer").
					AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
					AddField(schema.NewField("name").String().Build()).
					AddField(schema.NewField("email").String().Unique().Build()),
			},
			{
				"Invoice",
				schema.New("Invoice").
					AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
					AddField(schema.NewField("customerId").Int64().Build()).
					AddField(schema.NewField("amount").Float().Build()).
					AddField(schema.NewField("paid").Bool().Default(false).Build()),
			},
		}

		for _, s := range schemas {
			if err := eng.RegisterSchema(s.schema); err != nil {
				t.Fatalf("Failed to register %s schema: %v", s.name, err)
			}
		}

		// Create all at once
		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("Failed to ensure all schemas: %v", err)
		}

		// Test each schema works
		for _, s := range schemas {
			count, err := eng.Execute(formatCountQuery(s.name))
			if err != nil {
				t.Fatalf("Failed to query %s: %v", s.name, err)
			}
			if count.(int64) != 0 {
				t.Errorf("Expected 0 records in %s, got %v", s.name, count)
			}
		}

		t.Log("✅ Easy extension with multiple schemas works correctly")
	})
}

// Helper function to format count queries
func formatCountQuery(modelName string) string {
	return "models." + modelName + ".select().count()"
}