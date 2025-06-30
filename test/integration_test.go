package test

import (
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func TestIntegrationMultiDatabase(t *testing.T) {
	databases := []struct {
		name   string
		config types.Config
	}{
		{
			name: "SQLite",
			config: types.Config{
				Type:     types.SQLite,
				FilePath: ":memory:",
			},
		},
		{
			name: "MySQL",
			config: types.Config{
				Type:     types.MySQL,
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
		},
		{
			name: "PostgreSQL",
			config: types.Config{
				Type:     types.PostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
		},
	}

	for _, dbConfig := range databases {
		t.Run(dbConfig.name, func(t *testing.T) {
			testDatabaseWithJavaScript(t, dbConfig.config)
		})
	}
}

func testDatabaseWithJavaScript(t *testing.T, config types.Config) {
	db, err := database.New(config)
	if err != nil {
		if config.Type != types.SQLite {
			t.Skipf("Failed to create %s database: %v (Docker might not be running)", config.Type, err)
		} else {
			t.Fatalf("Failed to create SQLite database: %v", err)
		}
	}

	if err := db.Connect(); err != nil {
		if config.Type != types.SQLite {
			t.Skipf("Failed to connect to %s: %v (Docker might not be running)", config.Type, err)
		} else {
			t.Fatalf("Failed to connect to SQLite: %v", err)
		}
	}
	defer db.Close()

	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build())

	postSchema := schema.New("Post").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build()).
		AddField(schema.NewField("content").String().Build()).
		AddField(schema.NewField("user_id").Int64().Build())

	jsEngine := engine.New(db)
	
	err = jsEngine.RegisterSchema(userSchema)
	if err != nil {
		t.Fatalf("Failed to register User schema: %v", err)
	}
	defer db.DropTable(userSchema.TableName)
	
	err = jsEngine.RegisterSchema(postSchema)
	if err != nil {
		t.Fatalf("Failed to register Post schema: %v", err)
	}
	defer db.DropTable(postSchema.TableName)

	testScript := `
		// Test User operations
		var user1Id = models.User.add({
			name: "John Doe",
			email: "john@example.com",
			age: 30
		});
		
		if (!user1Id || user1Id <= 0) {
			throw new Error("Failed to create user - returned: " + user1Id);
		}
		
		var foundUser = models.User.get(user1Id);
		if (!foundUser || foundUser.name !== "John Doe") {
			throw new Error("Failed to find user by ID");
		}
		
		// Test User update
		models.User.set(user1Id, { age: 31 });
		var updatedUser = models.User.get(user1Id);
		if (updatedUser.age !== 31) {
			throw new Error("Failed to update user");
		}
		
		// Test User select
		var user2Id = models.User.add({
			name: "Jane Smith",
			email: "jane@example.com",
			age: 25
		});
		
		var users = models.User.select().execute();
		if (users.length !== 2) {
			throw new Error("Expected 2 users, got " + users.length);
		}
		
		// Test Post operations
		var post1Id = models.Post.add({
			title: "First Post",
			content: "This is the first post",
			user_id: user1Id
		});
		
		if (!post1Id || post1Id <= 0) {
			throw new Error("Failed to create post - returned: " + post1Id);
		}
		
		var foundPost = models.Post.get(post1Id);
		if (!foundPost || foundPost.title !== "First Post") {
			throw new Error("Failed to find post by ID");
		}
		
		// Test Post select with conditions
		var posts = models.Post.select().where("user_id", "=", user1Id).execute();
		if (posts.length !== 1) {
			throw new Error("Expected 1 post for user, got " + posts.length);
		}
		
		// Test remove operations
		models.Post.remove(post1Id);
		models.User.remove(user2Id);
		models.User.remove(user1Id);
		
		var remainingUsers = models.User.select().execute();
		if (remainingUsers.length !== 0) {
			throw new Error("Expected 0 users after cleanup, got " + remainingUsers.length);
		}
		
		"Integration test passed for " + dbType;
	`

	// Set dbType variable in JavaScript context
	jsEngine.GetVM().Set("dbType", string(config.Type))
	
	result, err := jsEngine.Execute(testScript)
	if err != nil {
		t.Fatalf("JavaScript execution failed: %v", err)
	}

	expectedMessage := fmt.Sprintf("Integration test passed for %s", config.Type)
	if result != expectedMessage {
		t.Errorf("Expected result '%s', got '%v'", expectedMessage, result)
	}
}

func TestIntegrationTransactions(t *testing.T) {
	databases := []struct {
		name   string
		config types.Config
	}{
		{
			name: "SQLite",
			config: types.Config{
				Type:     types.SQLite,
				FilePath: ":memory:",
			},
		},
		{
			name: "MySQL",
			config: types.Config{
				Type:     types.MySQL,
				Host:     "localhost",
				Port:     3306,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
		},
		{
			name: "PostgreSQL",
			config: types.Config{
				Type:     types.PostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "testdb",
				User:     "testuser",
				Password: "testpass",
			},
		},
	}

	for _, dbConfig := range databases {
		t.Run(dbConfig.name, func(t *testing.T) {
			testTransactions(t, dbConfig.config)
		})
	}
}

func testTransactions(t *testing.T, config types.Config) {
	db, err := database.New(config)
	if err != nil {
		if config.Type != types.SQLite {
			t.Skipf("Failed to create %s database: %v (Docker might not be running)", config.Type, err)
		} else {
			t.Fatalf("Failed to create SQLite database: %v", err)
		}
	}

	if err := db.Connect(); err != nil {
		if config.Type != types.SQLite {
			t.Skipf("Failed to connect to %s: %v (Docker might not be running)", config.Type, err)
		} else {
			t.Fatalf("Failed to connect to SQLite: %v", err)
		}
	}
	defer db.Close()

	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build())

	err = db.CreateTable(userSchema)
	if err != nil {
		t.Fatalf("Failed to create User table: %v", err)
	}
	defer db.DropTable(userSchema.TableName)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	user1ID, err := tx.Insert(userSchema.TableName, map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert user in transaction: %v", err)
	}

	user2ID, err := tx.Insert(userSchema.TableName, map[string]interface{}{
		"name":  "Jane Smith",
		"email": "jane@example.com",
	})
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to insert second user in transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	user1, err := db.FindByID(userSchema.TableName, user1ID)
	if err != nil {
		t.Fatalf("Failed to find committed user1: %v", err)
	}
	if user1["name"] != "John Doe" {
		t.Errorf("Expected user1 name 'John Doe', got %v", user1["name"])
	}

	user2, err := db.FindByID(userSchema.TableName, user2ID)
	if err != nil {
		t.Fatalf("Failed to find committed user2: %v", err)
	}
	if user2["name"] != "Jane Smith" {
		t.Errorf("Expected user2 name 'Jane Smith', got %v", user2["name"])
	}

	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin rollback transaction: %v", err)
	}

	_, err = tx2.Insert(userSchema.TableName, map[string]interface{}{
		"name":  "Bob Wilson",
		"email": "bob@example.com",
	})
	if err != nil {
		tx2.Rollback()
		t.Fatalf("Failed to insert user in rollback transaction: %v", err)
	}

	if err := tx2.Rollback(); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	users, err := db.Find(userSchema.TableName, map[string]interface{}{}, 0, 0)
	if err != nil {
		t.Fatalf("Failed to find all users: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users after rollback, got %d", len(users))
	}
}