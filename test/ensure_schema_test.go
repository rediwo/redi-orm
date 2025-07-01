package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
)

func TestEnsureSchemaWorkflow(t *testing.T) {
	// Create in-memory database
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	// Define multiple schemas
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build())

	postSchema := schema.New("Post").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build()).
		AddField(schema.NewField("content").String().Build()).
		AddField(schema.NewField("authorId").Int64().Build())

	// Register schemas (should not create tables yet)
	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register user schema: %v", err)
	}

	if err := eng.RegisterSchema(postSchema); err != nil {
		t.Fatalf("Failed to register post schema: %v", err)
	}

	// At this point, tables should not exist yet
	migrator := db.GetMigrator()
	if migrator != nil {
		existingTables, err := migrator.GetTables()
		if err != nil {
			t.Fatalf("Failed to get existing tables: %v", err)
		}

		// Should have no tables (except possibly migration table)
		for _, table := range existingTables {
			if table == "users" || table == "posts" {
				t.Errorf("Table %s should not exist before EnsureSchema", table)
			}
		}
	}

	// Now call EnsureSchema to create all tables
	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Verify tables now exist by trying to insert data
	userID, err := eng.Execute(`models.User.add({
		name: "Alice", 
		email: "alice@example.com",
		age: 30
	})`)
	if err != nil {
		t.Fatalf("Failed to create user after EnsureSchema: %v", err)
	}

	postID, err := eng.Execute(`models.Post.add({
		title: "My First Post",
		content: "Hello world!",
		authorId: ` + "1" + `
	})`)
	if err != nil {
		t.Fatalf("Failed to create post after EnsureSchema: %v", err)
	}

	// Verify data was inserted correctly
	user, err := eng.Execute(`models.User.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	userData := user.(map[string]interface{})
	if userData["name"] != "Alice" {
		t.Errorf("Expected user name 'Alice', got %v", userData["name"])
	}

	post, err := eng.Execute(`models.Post.get(1)`)
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	postData := post.(map[string]interface{})
	if postData["title"] != "My First Post" {
		t.Errorf("Expected post title 'My First Post', got %v", postData["title"])
	}

	t.Logf("✅ EnsureSchema workflow test passed")
	t.Logf("   - Created user ID: %v", userID)
	t.Logf("   - Created post ID: %v", postID)
}

func TestEnsureSchemaIdempotent(t *testing.T) {
	// Create in-memory database
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	// Register schema
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Call EnsureSchema multiple times - should be idempotent
	for i := 0; i < 3; i++ {
		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("EnsureSchema failed on iteration %d: %v", i+1, err)
		}
	}

	// Verify we can still use the table normally
	userID, err := eng.Execute(`models.User.add({name: "Test User"})`)
	if err != nil {
		t.Fatalf("Failed to create user after multiple EnsureSchema calls: %v", err)
	}

	if userID.(int64) != 1 {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	t.Log("✅ EnsureSchema idempotent test passed")
}

func TestDirectDatabaseEnsureSchema(t *testing.T) {
	// Test EnsureSchema directly on database (without engine)
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Register schemas directly with database
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := db.RegisterSchema("User", userSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	// Call EnsureSchema on database
	if err := db.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Verify table was created by attempting to insert
	_, err = db.Insert("User", map[string]interface{}{
		"name": "Direct Test User",
	})
	if err != nil {
		t.Fatalf("Failed to insert after EnsureSchema: %v", err)
	}

	t.Log("✅ Direct database EnsureSchema test passed")
}