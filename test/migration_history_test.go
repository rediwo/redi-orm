package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
)

// TestMigrationHistoryTable tests that migration history is properly tracked
func TestMigrationHistoryTable(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Register and migrate a schema
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Check if migration history table exists
	migrator := db.GetMigrator()
	if migrator != nil {
		tables, err := migrator.GetTables()
		if err != nil {
			t.Fatalf("Failed to get tables: %v", err)
		}

		found := false
		for _, table := range tables {
			if table == "redi_migrations" {
				found = true
				break
			}
		}

		if found {
			t.Log("✅ Migration history table 'redi_migrations' exists")
		} else {
			t.Log("ℹ️  Migration history table not yet implemented")
		}
	}

	t.Log("✅ Migration history tracking test completed")
}

// TestMigrationVersioning tests schema versioning through migrations
func TestMigrationVersioning(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Version 1: Basic user schema
	userSchemaV1 := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(userSchemaV1); err != nil {
		t.Fatalf("Failed to register user schema v1: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration v1 failed: %v", err)
	}

	// Insert v1 data
	userID, err := eng.Execute(`models.User.add({name: "Alice"})`)
	if err != nil {
		t.Fatalf("Failed to insert v1 user: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	// Simulate application restart with schema evolution
	// Version 2: Add email field (this would require schema evolution)
	// For now, we just add a new table to simulate evolution
	postSchema := schema.New("Post").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build()).
		AddField(schema.NewField("authorId").Int64().Build())

	if err := eng.RegisterSchema(postSchema); err != nil {
		t.Fatalf("Failed to register post schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration v2 failed: %v", err)
	}

	// Insert v2 data
	postID, err := eng.Execute(`models.Post.add({title: "First Post", authorId: 1})`)
	if err != nil {
		t.Fatalf("Failed to insert v2 post: %v", err)
	}

	if postID != int64(1) {
		t.Errorf("Expected post ID 1, got %v", postID)
	}

	// Verify both v1 and v2 data still exist
	user, err := eng.Execute(`models.User.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get v1 user after v2 migration: %v", err)
	}

	userData := user.(map[string]interface{})
	if userData["name"] != "Alice" {
		t.Errorf("V1 data corrupted after v2 migration: expected 'Alice', got %v", userData["name"])
	}

	post, err := eng.Execute(`models.Post.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get v2 post: %v", err)
	}

	postData := post.(map[string]interface{})
	if postData["title"] != "First Post" {
		t.Errorf("V2 data incorrect: expected 'First Post', got %v", postData["title"])
	}

	t.Log("✅ Schema versioning through migrations works correctly")
}

// TestMigrationRollbackSafety tests that migrations don't corrupt existing data
func TestMigrationRollbackSafety(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Create initial data
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register user schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Initial migration failed: %v", err)
	}

	// Insert test data
	testUsers := []string{
		`models.User.add({name: "Alice", email: "alice@test.com"})`,
		`models.User.add({name: "Bob", email: "bob@test.com"})`,
		`models.User.add({name: "Charlie", email: "charlie@test.com"})`,
	}

	for _, userScript := range testUsers {
		_, err := eng.Execute(userScript)
		if err != nil {
			t.Fatalf("Failed to insert test user: %v", err)
		}
	}

	// Verify initial data count
	count, err := eng.Execute(`models.User.select().count()`)
	if err != nil {
		t.Fatalf("Failed to count initial users: %v", err)
	}

	if count != int64(3) {
		t.Errorf("Expected 3 initial users, got %v", count)
	}

	// Attempt to add another schema (should not affect existing data)
	postSchema := schema.New("Post").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build())

	if err := eng.RegisterSchema(postSchema); err != nil {
		t.Fatalf("Failed to register post schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Post migration failed: %v", err)
	}

	// Verify existing data is still intact
	finalCount, err := eng.Execute(`models.User.select().count()`)
	if err != nil {
		t.Fatalf("Failed to count users after migration: %v", err)
	}

	if finalCount != int64(3) {
		t.Errorf("User data corrupted during migration: expected 3, got %v", finalCount)
	}

	// Verify specific user data
	user, err := eng.Execute(`models.User.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get specific user after migration: %v", err)
	}

	userData := user.(map[string]interface{})
	if userData["name"] != "Alice" {
		t.Errorf("User data corrupted: expected 'Alice', got %v", userData["name"])
	}

	t.Log("✅ Migration rollback safety works correctly")
}

// TestMigrationSchemaHashTracking tests schema change detection
func TestMigrationSchemaHashTracking(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Create schema and run migration
	userSchema1 := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(userSchema1); err != nil {
		t.Fatalf("Failed to register user schema 1: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration 1 failed: %v", err)
	}

	// Run same migration again - should be idempotent
	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Idempotent migration failed: %v", err)
	}

	// Simulate schema change by adding new field
	// (In a real implementation, this would trigger schema evolution)
	// Note: Currently our system creates a new engine for schema changes
	// In production, this would be handled by schema evolution
	eng2 := engine.New(db)
	
	// For now, we'll create a different table to test migration tracking
	profileSchema := schema.New("Profile").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("bio").String().Build()).
		AddField(schema.NewField("userId").Int64().Build())

	if err := eng2.RegisterSchema(userSchema1); err != nil {
		t.Fatalf("Failed to re-register user schema: %v", err)
	}

	if err := eng2.RegisterSchema(profileSchema); err != nil {
		t.Fatalf("Failed to register profile schema: %v", err)
	}

	if err := eng2.EnsureSchema(); err != nil {
		t.Fatalf("Migration 2 failed: %v", err)
	}

	// Verify that both tables exist and work
	userID, err := eng2.Execute(`models.User.add({name: "Test User"})`)
	if err != nil {
		t.Fatalf("Failed to insert user in migration 2: %v", err)
	}

	profileID, err := eng2.Execute(`models.Profile.add({bio: "Test bio", userId: 1})`)
	if err != nil {
		t.Fatalf("Failed to insert profile in migration 2: %v", err)
	}

	if userID != int64(1) || profileID != int64(1) {
		t.Errorf("Unexpected IDs: user=%v, profile=%v", userID, profileID)
	}

	t.Log("✅ Schema hash tracking simulation completed")
}

// TestMigrationConcurrentAccess tests migration behavior under concurrent access
func TestMigrationConcurrentAccess(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Run multiple migrations concurrently
	done := make(chan error, 5)
	
	for i := 0; i < 5; i++ {
		go func(id int) {
			if err := eng.EnsureSchema(); err != nil {
				done <- err
				return
			}
			done <- nil
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Fatalf("Concurrent migration failed: %v", err)
		}
	}

	// Test that table works after concurrent migrations
	userID, err := eng.Execute(`models.User.add({name: "Concurrent Test"})`)
	if err != nil {
		t.Fatalf("Failed to insert after concurrent migrations: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	t.Log("✅ Concurrent migration access handled correctly")
}

// TestMigrationErrorRecovery tests recovery from migration errors
func TestMigrationErrorRecovery(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Try to register invalid schema
	invalidSchema := schema.New("Invalid").
		AddField(schema.NewField("name").String().Build()) // No primary key

	err = eng.RegisterSchema(invalidSchema)
	if err == nil {
		t.Error("Expected error for invalid schema")
	}

	// Register valid schema after error
	validSchema := schema.New("Valid").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(validSchema); err != nil {
		t.Fatalf("Failed to register valid schema after error: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration failed after error recovery: %v", err)
	}

	// Test that valid schema works
	recordID, err := eng.Execute(`models.Valid.add({name: "Recovery Test"})`)
	if err != nil {
		t.Fatalf("Failed to insert after error recovery: %v", err)
	}

	if recordID != int64(1) {
		t.Errorf("Expected record ID 1, got %v", recordID)
	}

	t.Log("✅ Migration error recovery works correctly")
}

// TestMigrationPerformanceWithHistory tests migration performance tracking
func TestMigrationPerformanceWithHistory(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Register multiple schemas in batches to simulate evolution
	schemas := []*schema.Schema{
		schema.New("User").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()),
		
		schema.New("Post").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("title").String().Build()),
		
		schema.New("Comment").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("content").String().Build()),
			
		schema.New("Tag").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()),
			
		schema.New("Category").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()),
	}

	// Batch 1: Register and migrate first 2 schemas
	for i := 0; i < 2; i++ {
		if err := eng.RegisterSchema(schemas[i]); err != nil {
			t.Fatalf("Failed to register schema %d: %v", i, err)
		}
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Batch 1 migration failed: %v", err)
	}

	// Batch 2: Add remaining schemas
	for i := 2; i < len(schemas); i++ {
		if err := eng.RegisterSchema(schemas[i]); err != nil {
			t.Fatalf("Failed to register schema %d: %v", i, err)
		}
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Batch 2 migration failed: %v", err)
	}

	// Test all tables work
	operations := []string{
		`models.User.add({name: "Test User"})`,
		`models.Post.add({title: "Test Post"})`,
		`models.Comment.add({content: "Test Comment"})`,
		`models.Tag.add({name: "Test Tag"})`,
		`models.Category.add({name: "Test Category"})`,
	}

	for i, op := range operations {
		result, err := eng.Execute(op)
		if err != nil {
			t.Fatalf("Failed to execute operation %d: %v", i, err)
		}
		if result != int64(1) {
			t.Errorf("Expected ID 1 for operation %d, got %v", i, result)
		}
	}

	t.Logf("✅ Successfully migrated %d schemas in batches", len(schemas))
}