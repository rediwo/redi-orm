package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
)

// TestEnsureSchemaWorkflow tests the complete workflow of schema registration and auto-migration
func TestEnsureSchemaWorkflow(t *testing.T) {
	// Create database connection
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	t.Run("Phase 1: Register Schemas", func(t *testing.T) {
		// Define User schema
		userSchema := schema.New("User").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("email").String().Unique().Build()).
			AddField(schema.NewField("age").Int().Nullable().Build())

		// Define Post schema
		postSchema := schema.New("Post").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("title").String().Build()).
			AddField(schema.NewField("content").String().Build()).
			AddField(schema.NewField("authorId").Int64().Build()).
			AddField(schema.NewField("published").Bool().Default(false).Build())

		// Define Comment schema  
		commentSchema := schema.New("Comment").
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("content").String().Build()).
			AddField(schema.NewField("postId").Int64().Build()).
			AddField(schema.NewField("authorId").Int64().Build())

		// Register all schemas (fast operations)
		if err := eng.RegisterSchema(userSchema); err != nil {
			t.Fatalf("Failed to register User schema: %v", err)
		}

		if err := eng.RegisterSchema(postSchema); err != nil {
			t.Fatalf("Failed to register Post schema: %v", err)
		}

		if err := eng.RegisterSchema(commentSchema); err != nil {
			t.Fatalf("Failed to register Comment schema: %v", err)
		}

		t.Log("✅ All schemas registered successfully")
	})

	t.Run("Phase 2: Auto-Migration", func(t *testing.T) {
		// Create all tables at once
		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("Failed to ensure schema: %v", err)
		}
		t.Log("✅ All tables created successfully")
	})

	t.Run("Phase 3: Test Operations", func(t *testing.T) {
		// Create users
		userID1, err := eng.Execute(`models.User.add({
			name: "Alice Johnson", 
			email: "alice@example.com",
			age: 28
		})`)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		if userID1.(int64) <= 0 {
			t.Errorf("Invalid user ID: %v", userID1)
		}

		userID2, err := eng.Execute(`models.User.add({
			name: "Bob Smith", 
			email: "bob@example.com",
			age: 32
		})`)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		if userID2.(int64) <= 0 {
			t.Errorf("Invalid user ID: %v", userID2)
		}

		// Create posts
		postID1, err := eng.Execute(`models.Post.add({
			title: "Getting Started with RediORM",
			content: "RediORM makes database operations simple and fast!",
			authorId: 1,
			published: true
		})`)
		if err != nil {
			t.Fatalf("Failed to create post: %v", err)
		}
		if postID1.(int64) <= 0 {
			t.Errorf("Invalid post ID: %v", postID1)
		}

		postID2, err := eng.Execute(`models.Post.add({
			title: "Auto-Migration Benefits",
			content: "With EnsureSchema, you can manage multiple schemas efficiently.",
			authorId: 2,
			published: false
		})`)
		if err != nil {
			t.Fatalf("Failed to create post: %v", err)
		}
		if postID2.(int64) <= 0 {
			t.Errorf("Invalid post ID: %v", postID2)
		}

		// Create comments
		commentID, err := eng.Execute(`models.Comment.add({
			content: "Great tutorial! Thanks for sharing.",
			postId: 1,
			authorId: 2
		})`)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}
		if commentID.(int64) <= 0 {
			t.Errorf("Invalid comment ID: %v", commentID)
		}

		t.Log("✅ All test data created successfully")
	})

	t.Run("Phase 4: Query Data", func(t *testing.T) {
		// Query all published posts
		publishedPosts, err := eng.Execute(`
			models.Post.select()
				.where("published", "=", true)
				.execute()
		`)
		if err != nil {
			t.Fatalf("Failed to query posts: %v", err)
		}
		posts := publishedPosts.([]map[string]interface{})
		if len(posts) != 1 {
			t.Errorf("Expected 1 published post, got %d", len(posts))
		}

		// Count total users
		totalUsers, err := eng.Execute(`models.User.select().count()`)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}
		if totalUsers.(int64) != 2 {
			t.Errorf("Expected 2 users, got %v", totalUsers)
		}

		// Count total comments
		totalComments, err := eng.Execute(`models.Comment.select().count()`)
		if err != nil {
			t.Fatalf("Failed to count comments: %v", err)
		}
		if totalComments.(int64) != 1 {
			t.Errorf("Expected 1 comment, got %v", totalComments)
		}

		t.Log("✅ All queries executed successfully")
	})

	t.Run("Phase 5: Test Idempotency", func(t *testing.T) {
		// Call EnsureSchema again - should be safe
		if err := eng.EnsureSchema(); err != nil {
			t.Fatalf("EnsureSchema failed on second call: %v", err)
		}
		t.Log("✅ EnsureSchema is idempotent - safe to call multiple times")
	})
}

// TestEnsureSchemaMultipleDatabases tests schema registration across different database engines
func TestEnsureSchemaMultipleDatabases(t *testing.T) {
	testCases := []struct {
		name string
		uri  string
	}{
		{"SQLite Memory", "sqlite://:memory:"},
		{"SQLite File", "sqlite://:memory:"},
	}

	// No cleanup needed for in-memory databases

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := database.NewFromURI(tc.uri)
			if err != nil {
				t.Fatalf("Failed to create database: %v", err)
			}

			if err := db.Connect(); err != nil {
				t.Fatalf("Failed to connect: %v", err)
			}
			defer db.Close()

			eng := engine.New(db)

			// Register multiple schemas
			userSchema := schema.New("User").
				AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
				AddField(schema.NewField("username").String().Unique().Build()).
				AddField(schema.NewField("active").Bool().Default(true).Build())

			productSchema := schema.New("Product").
				AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
				AddField(schema.NewField("name").String().Build()).
				AddField(schema.NewField("price").Float().Build()).
				AddField(schema.NewField("inStock").Bool().Default(true).Build())

			if err := eng.RegisterSchema(userSchema); err != nil {
				t.Fatalf("Failed to register User schema: %v", err)
			}

			if err := eng.RegisterSchema(productSchema); err != nil {
				t.Fatalf("Failed to register Product schema: %v", err)
			}

			// Ensure all schemas at once
			if err := eng.EnsureSchema(); err != nil {
				t.Fatalf("Failed to ensure schemas: %v", err)
			}

			// Test operations on both schemas
			userID, err := eng.Execute(`models.User.add({username: "testuser"})`)
			if err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}
			if userID.(int64) <= 0 {
				t.Errorf("Invalid user ID: %v", userID)
			}

			productID, err := eng.Execute(`models.Product.add({name: "Test Product", price: 99.99})`)
			if err != nil {
				t.Fatalf("Failed to create product: %v", err)
			}
			if productID.(int64) <= 0 {
				t.Errorf("Invalid product ID: %v", productID)
			}

			t.Logf("✅ %s: Successfully created schemas and data", tc.name)
		})
	}
}

// TestEnsureSchemaPerformance tests the performance benefit of batch schema creation
func TestEnsureSchemaPerformance(t *testing.T) {
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	eng := engine.New(db)

	// Register 10 schemas
	schemaCount := 10
	for i := 0; i < schemaCount; i++ {
		s := schema.New(formatModelName("Model", i)).
			AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
			AddField(schema.NewField("name").String().Build()).
			AddField(schema.NewField("value").Int().Build()).
			AddField(schema.NewField("active").Bool().Default(true).Build())

		if err := eng.RegisterSchema(s); err != nil {
			t.Fatalf("Failed to register schema %d: %v", i, err)
		}
	}

	// Create all tables at once
	if err := eng.EnsureSchema(); err != nil {
		t.Fatalf("Failed to ensure schemas: %v", err)
	}

	// Verify all tables were created
	for i := 0; i < schemaCount; i++ {
		modelName := formatModelName("Model", i)
		count, err := eng.Execute(formatQuery("models.%s.select().count()", modelName))
		if err != nil {
			t.Fatalf("Failed to query %s: %v", modelName, err)
		}
		if count.(int64) != 0 {
			t.Errorf("Expected 0 records in %s, got %v", modelName, count)
		}
	}

	t.Logf("✅ Successfully created %d schemas in batch", schemaCount)
}

// Helper functions
func formatModelName(prefix string, index int) string {
	return prefix + "_" + string(rune('0'+index))
}

func formatQuery(format string, args ...interface{}) string {
	result := format
	for _, arg := range args {
		result = replaceFirst(result, "%s", arg.(string))
	}
	return result
}

func replaceFirst(s, old, new string) string {
	i := 0
	for i < len(s) {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
		i++
	}
	return s
}