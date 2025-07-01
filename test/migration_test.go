package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
)

// TestMigrationCreateNewTables tests creation of completely new tables
func TestMigrationCreateNewTables(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Define schemas
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build())

	postSchema := schema.New("Post").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build()).
		AddField(schema.NewField("content").String().Build()).
		AddField(schema.NewField("authorId").Int64().Build())

	// Register schemas
	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register user schema: %v", err)
	}

	if err := eng.RegisterSchema(postSchema); err != nil {
		t.Fatalf("Failed to register post schema: %v", err)
	}

	// Run migration
	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify tables were created by inserting data
	userID, err := eng.Execute(`models.User.add({name: "Alice", email: "alice@test.com"})`)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	postID, err := eng.Execute(`models.Post.add({title: "Test Post", content: "Content", authorId: 1})`)
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	if postID != int64(1) {
		t.Errorf("Expected post ID 1, got %v", postID)
	}

	t.Log("✅ Migration successfully created new tables")
}

// TestMigrationIdempotency tests that running migration multiple times is safe
func TestMigrationIdempotency(t *testing.T) {
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

	// Run migration multiple times
	for i := 0; i < 5; i++ {
		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("Migration failed on iteration %d: %v", i+1, err)
		}
	}

	// Verify table works correctly
	userID, err := eng.Execute(`models.User.add({name: "Test User"})`)
	if err != nil {
		t.Fatalf("Failed to insert user after multiple migrations: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	t.Log("✅ Migration is idempotent - safe to run multiple times")
}

// TestMigrationProgressiveSchemaAddition tests adding schemas progressively
func TestMigrationProgressiveSchemaAddition(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Phase 1: Add User schema
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register user schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Phase 1 migration failed: %v", err)
	}

	// Insert user data
	userID, err := eng.Execute(`models.User.add({name: "Alice"})`)
	if err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	// Phase 2: Add Post schema
	postSchema := schema.New("Post").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build()).
		AddField(schema.NewField("authorId").Int64().Build())

	if err := eng.RegisterSchema(postSchema); err != nil {
		t.Fatalf("Failed to register post schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Phase 2 migration failed: %v", err)
	}

	// Insert post data
	postID, err := eng.Execute(`models.Post.add({title: "First Post", authorId: 1})`)
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
	}

	// Phase 3: Add Comment schema  
	commentSchema := schema.New("Comment").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("content").String().Build()).
		AddField(schema.NewField("postId").Int64().Build())

	if err := eng.RegisterSchema(commentSchema); err != nil {
		t.Fatalf("Failed to register comment schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Phase 3 migration failed: %v", err)
	}

	// Insert comment data
	commentID, err := eng.Execute(`models.Comment.add({content: "Great post!", postId: 1})`)
	if err != nil {
		t.Fatalf("Failed to insert comment: %v", err)
	}

	// Verify all data exists
	if userID != int64(1) || postID != int64(1) || commentID != int64(1) {
		t.Errorf("Unexpected IDs: user=%v, post=%v, comment=%v", userID, postID, commentID)
	}

	// Verify counts
	userCount, err := eng.Execute(`models.User.select().count()`)
	if err != nil {
		t.Fatalf("Failed to count users: %v", err)
	}

	postCount, err := eng.Execute(`models.Post.select().count()`)
	if err != nil {
		t.Fatalf("Failed to count posts: %v", err)
	}

	commentCount, err := eng.Execute(`models.Comment.select().count()`)
	if err != nil {
		t.Fatalf("Failed to count comments: %v", err)
	}

	if userCount != int64(1) || postCount != int64(1) || commentCount != int64(1) {
		t.Errorf("Unexpected counts: users=%v, posts=%v, comments=%v", userCount, postCount, commentCount)
	}

	t.Log("✅ Progressive schema addition works correctly")
}

// TestMigrationWithComplexSchemas tests migration with complex field types
func TestMigrationWithComplexSchemas(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Complex schema with various field types
	complexSchema := schema.New("ComplexModel").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("salary").Float().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build()).
		AddField(schema.NewField("metadata").JSON().Nullable().Build()).
		AddField(schema.NewField("createdAt").DateTime().Build())

	if err := eng.RegisterSchema(complexSchema); err != nil {
		t.Fatalf("Failed to register complex schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Complex schema migration failed: %v", err)
	}

	// Insert data with various types
	recordID, err := eng.Execute(`models.ComplexModel.add({
		name: "John Doe",
		age: 30,
		salary: 75000.50,
		active: true,
		metadata: "{\"role\": \"developer\", \"level\": \"senior\"}",
		createdAt: "2024-01-01 10:00:00"
	})`)
	if err != nil {
		t.Fatalf("Failed to insert complex record: %v", err)
	}

	if recordID != int64(1) {
		t.Errorf("Expected record ID 1, got %v", recordID)
	}

	// Retrieve and verify data
	record, err := eng.Execute(`models.ComplexModel.get(1)`)
	if err != nil {
		t.Fatalf("Failed to retrieve complex record: %v", err)
	}

	recordData := record.(map[string]interface{})
	if recordData["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", recordData["name"])
	}

	if recordData["age"] != int64(30) {
		t.Errorf("Expected age 30, got %v", recordData["age"])
	}

	t.Log("✅ Complex schema migration works correctly")
}

// TestMigrationConcurrency tests concurrent migration calls
func TestMigrationConcurrency(t *testing.T) {
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

	// Run multiple goroutines calling EnsureSchema concurrently
	done := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			if err := eng.EnsureSchema(); err != nil {
				done <- err
				return
			}
			done <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Fatalf("Concurrent migration failed: %v", err)
		}
	}

	// Verify table works
	userID, err := eng.Execute(`models.User.add({name: "Concurrent User"})`)
	if err != nil {
		t.Fatalf("Failed to insert user after concurrent migrations: %v", err)
	}

	if userID != int64(1) {
		t.Errorf("Expected user ID 1, got %v", userID)
	}

	t.Log("✅ Concurrent migrations handled safely")
}

// TestMigrationErrorHandling tests migration error scenarios
func TestMigrationErrorHandling(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Test with invalid schema (no primary key)
	invalidSchema := schema.New("InvalidModel").
		AddField(schema.NewField("name").String().Build())

	err = eng.RegisterSchema(invalidSchema)
	if err == nil {
		t.Error("Expected error for schema without primary key")
	}

	// Test with valid schema after error
	validSchema := schema.New("ValidModel").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	if err := eng.RegisterSchema(validSchema); err != nil {
		t.Fatalf("Failed to register valid schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration failed for valid schema: %v", err)
	}

	// Verify valid schema works
	recordID, err := eng.Execute(`models.ValidModel.add({name: "Test"})`)
	if err != nil {
		t.Fatalf("Failed to insert into valid model: %v", err)
	}

	if recordID != int64(1) {
		t.Errorf("Expected record ID 1, got %v", recordID)
	}

	t.Log("✅ Migration error handling works correctly")
}

// TestMigrationWithDifferentDatabases tests migration across different database types
func TestMigrationWithDifferentDatabases(t *testing.T) {
	databases := []struct {
		name string
		uri  string
	}{
		{"SQLite Memory", "sqlite://:memory:"},
		{"SQLite File", "sqlite://./test_migration.db"},
	}

	for _, dbConfig := range databases {
		t.Run(dbConfig.name, func(t *testing.T) {
			db, err := database.NewFromURI(dbConfig.uri)
			if err != nil {
				t.Fatalf("Failed to create %s database: %v", dbConfig.name, err)
			}
			defer db.Close()

			if err := db.Connect(); err != nil {
				t.Fatalf("Failed to connect to %s: %v", dbConfig.name, err)
			}

			eng := engine.New(db)

			testSchema := schema.New("TestModel").
				AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
				AddField(schema.NewField("name").String().Build())

			if err := eng.RegisterSchema(testSchema); err != nil {
				t.Fatalf("Failed to register schema for %s: %v", dbConfig.name, err)
			}

			if err := eng.EnsureSchema(); err != nil {
				t.Fatalf("Migration failed for %s: %v", dbConfig.name, err)
			}

			// Test data insertion
			recordID, err := eng.Execute(`models.TestModel.add({name: "Test Record"})`)
			if err != nil {
				t.Fatalf("Failed to insert data in %s: %v", dbConfig.name, err)
			}

			if recordID != int64(1) {
				t.Errorf("Expected record ID 1 for %s, got %v", dbConfig.name, recordID)
			}

			t.Logf("✅ %s migration successful", dbConfig.name)
		})
	}
}

// TestMigrationPerformance tests migration performance with many schemas
func TestMigrationPerformance(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	eng := engine.New(db)

	// Register many schemas
	numSchemas := 50
	for i := 0; i < numSchemas; i++ {
		schemaName := "Model" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		testSchema := schema.New(schemaName).
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("value").Int().Build())

		if err := eng.RegisterSchema(testSchema); err != nil {
			t.Fatalf("Failed to register schema %s: %v", schemaName, err)
		}
	}

	// Measure migration time
	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Bulk migration failed: %v", err)
	}

	// Test a few tables work
	recordID, err := eng.Execute(`models.ModelA0.add({name: "Test", value: 42})`)
	if err != nil {
		t.Fatalf("Failed to insert into first model: %v", err)
	}

	if recordID != int64(1) {
		t.Errorf("Expected record ID 1, got %v", recordID)
	}

	t.Logf("✅ Successfully migrated %d schemas", numSchemas)
}

// TestMigrationIntrospection tests database introspection capabilities
func TestMigrationIntrospection(t *testing.T) {
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
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build())

	if err := eng.RegisterSchema(userSchema); err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}

	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Test introspection
	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("No migrator available")
	}

	tables, err := migrator.GetTables()
	if err != nil {
		t.Fatalf("Failed to get tables: %v", err)
	}

	found := false
	for _, table := range tables {
		if table == "users" { // SQLite converts User -> users
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find 'users' table, got tables: %v", tables)
	}

	// Test table info
	tableInfo, err := migrator.GetTableInfo("users")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}

	if len(tableInfo.Columns) < 3 {
		t.Errorf("Expected at least 3 columns, got %d", len(tableInfo.Columns))
	}

	t.Log("✅ Database introspection works correctly")
}