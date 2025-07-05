package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabase provides utilities for test database management
type TestDatabase struct {
	DB      types.Database
	URI     string
	Cleanup func()
	T       *testing.T
}

// NewTestDatabase creates a new test database instance
func NewTestDatabase(t *testing.T, db types.Database, uri string, cleanup func()) *TestDatabase {
	return &TestDatabase{
		DB:      db,
		URI:     uri,
		T:       t,
		Cleanup: cleanup,
	}
}

// CreateStandardSchemas creates standard test schemas
func (td *TestDatabase) CreateStandardSchemas() error {
	return td.CreateStandardSchemasWithCleanup(false)
}

// CreateStandardSchemasWithCleanup creates standard test schemas with optional data cleanup
func (td *TestDatabase) CreateStandardSchemasWithCleanup(cleanupData bool) error {
	ctx := context.Background()

	// User schema
	// Prisma equivalent:
	// model User {
	//   id        Int      @id @default(autoincrement())
	//   name      String
	//   email     String   @unique
	//   age       Int?
	//   active    Boolean  @default(true)
	//   createdAt DateTime @default(now())
	//   posts     Post[]
	// }
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true}).
		AddField(schema.Field{Name: "age", Type: schema.FieldTypeInt, Nullable: true}).
		AddField(schema.Field{Name: "active", Type: schema.FieldTypeBool, Default: true}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "now()"})

	// Post schema
	// Prisma equivalent:
	// model Post {
	//   id        Int      @id @default(autoincrement())
	//   title     String
	//   content   String?
	//   userId    Int
	//   published Boolean  @default(false)
	//   views     Int      @default(0)
	//   createdAt DateTime @default(now())
	//   user      User     @relation(fields: [userId], references: [id])
	//   comments  Comment[]
	//   tags      Tag[]
	// }
	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "title", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString, Nullable: true}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "published", Type: schema.FieldTypeBool, Default: false}).
		AddField(schema.Field{Name: "views", Type: schema.FieldTypeInt, Default: 0}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "now()"})

	// Comment schema
	// Prisma equivalent:
	// model Comment {
	//   id        Int      @id @default(autoincrement())
	//   content   String
	//   postId    Int
	//   userId    Int
	//   createdAt DateTime @default(now())
	//   post      Post     @relation(fields: [postId], references: [id])
	//   user      User     @relation(fields: [userId], references: [id])
	// }
	commentSchema := schema.New("Comment").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "postId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "now()"})

	// Tag schema (for many-to-many tests)
	// Prisma equivalent:
	// model Tag {
	//   id    Int    @id @default(autoincrement())
	//   name  String @unique
	//   posts Post[]
	// }
	tagSchema := schema.New("Tag").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString, Unique: true})

	// PostTag schema (junction table for many-to-many)
	// Prisma equivalent:
	// model PostTag {
	//   postId Int
	//   tagId  Int
	//   post   Post @relation(fields: [postId], references: [id])
	//   tag    Tag  @relation(fields: [tagId], references: [id])
	//   @@id([postId, tagId])
	// }
	postTagSchema := schema.New("PostTag").
		AddField(schema.Field{Name: "postId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "tagId", Type: schema.FieldTypeInt})
	postTagSchema.CompositeKey = []string{"postId", "tagId"}

	// Add relations
	userSchema.AddRelation("posts", schema.Relation{
		Type:       schema.RelationOneToMany,
		Model:      "Post",
		ForeignKey: "userId",
		References: "id",
	})

	postSchema.AddRelation("user", schema.Relation{
		Type:       schema.RelationManyToOne,
		Model:      "User",
		ForeignKey: "userId",
		References: "id",
	})

	postSchema.AddRelation("comments", schema.Relation{
		Type:       schema.RelationOneToMany,
		Model:      "Comment",
		ForeignKey: "postId",
		References: "id",
	})

	postSchema.AddRelation("tags", schema.Relation{
		Type:       schema.RelationManyToMany,
		Model:      "Tag",
		ForeignKey: "postId",
		References: "id",
	})

	commentSchema.AddRelation("post", schema.Relation{
		Type:       schema.RelationManyToOne,
		Model:      "Post",
		ForeignKey: "postId",
		References: "id",
	})

	commentSchema.AddRelation("user", schema.Relation{
		Type:       schema.RelationManyToOne,
		Model:      "User",
		ForeignKey: "userId",
		References: "id",
	})

	tagSchema.AddRelation("posts", schema.Relation{
		Type:       schema.RelationManyToMany,
		Model:      "Post",
		ForeignKey: "tagId",
		References: "id",
	})

	// Register schemas
	schemas := map[string]*schema.Schema{
		"User":    userSchema,
		"Post":    postSchema,
		"Comment": commentSchema,
		"Tag":     tagSchema,
		"PostTag": postTagSchema,
	}

	for name, s := range schemas {
		if err := td.DB.RegisterSchema(name, s); err != nil {
			return fmt.Errorf("failed to register schema %s: %w", name, err)
		}
	}

	// Create tables using SyncSchemas for proper dependency resolution
	if err := td.DB.SyncSchemas(ctx); err != nil {
		return fmt.Errorf("failed to sync schemas: %w", err)
	}

	// Optionally clean up any existing data
	if cleanupData {
		// Cleanup should be handled by each driver's specific cleanup function
		td.T.Log("Data cleanup should be handled by driver-specific cleanupTables function")
	}

	return nil
}

// InsertStandardTestData inserts standard test data
func (td *TestDatabase) InsertStandardTestData() error {
	ctx := context.Background()

	// Driver-specific cleanup should be done before calling this function

	// Insert users
	User := td.DB.Model("User")
	users := []map[string]any{
		{"name": "Alice", "email": "alice@example.com", "age": 25, "active": true},
		{"name": "Bob", "email": "bob@example.com", "age": 30, "active": true},
		{"name": "Charlie", "email": "charlie@example.com", "age": 35, "active": false},
		{"name": "David", "email": "david@example.com", "age": nil, "active": true},
		{"name": "Eve", "email": "eve@example.com", "age": 28, "active": true},
	}

	for _, user := range users {
		if _, err := User.Insert(user).Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
	}

	// Insert posts
	Post := td.DB.Model("Post")
	posts := []map[string]any{
		{"title": "First Post", "content": "Hello World", "userId": 1, "published": true, "views": 100},
		{"title": "Second Post", "content": "Another post", "userId": 1, "published": false, "views": 0},
		{"title": "Bob's Post", "content": "Bob's content", "userId": 2, "published": true, "views": 50},
		{"title": "Charlie's Draft", "content": nil, "userId": 3, "published": false, "views": 0},
		{"title": "Popular Post", "content": "Very popular", "userId": 2, "published": true, "views": 1000},
	}

	for _, post := range posts {
		if _, err := Post.Insert(post).Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert post: %w", err)
		}
	}

	// Insert comments
	Comment := td.DB.Model("Comment")
	comments := []map[string]any{
		{"content": "Great post!", "postId": 1, "userId": 2},
		{"content": "Thanks!", "postId": 1, "userId": 1},
		{"content": "Interesting", "postId": 3, "userId": 3},
		{"content": "Nice work", "postId": 5, "userId": 4},
	}

	for _, comment := range comments {
		if _, err := Comment.Insert(comment).Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert comment: %w", err)
		}
	}

	// Insert tags
	Tag := td.DB.Model("Tag")
	tags := []map[string]any{
		{"name": "Technology"},
		{"name": "Programming"},
		{"name": "Tutorial"},
		{"name": "News"},
	}

	for _, tag := range tags {
		if _, err := Tag.Insert(tag).Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert tag: %w", err)
		}
	}

	// Insert post-tag relationships
	PostTag := td.DB.Model("PostTag")
	postTags := []map[string]any{
		{"postId": 1, "tagId": 1},
		{"postId": 1, "tagId": 2},
		{"postId": 2, "tagId": 3},
		{"postId": 3, "tagId": 1},
		{"postId": 5, "tagId": 4},
	}

	for _, pt := range postTags {
		if _, err := PostTag.Insert(pt).Exec(ctx); err != nil {
			return fmt.Errorf("failed to insert post-tag: %w", err)
		}
	}

	return nil
}

// AssertCount asserts the count of records in a model
func (td *TestDatabase) AssertCount(modelName string, expected int64) {
	ctx := context.Background()
	model := td.DB.Model(modelName)
	count, err := model.Select().Count(ctx)
	require.NoError(td.T, err, "failed to count %s", modelName)
	assert.Equal(td.T, expected, count, "unexpected count for %s", modelName)
}

// AssertExists checks if a record exists with given conditions
func (td *TestDatabase) AssertExists(modelName string, condition types.Condition) {
	ctx := context.Background()
	model := td.DB.Model(modelName)
	exists, err := model.WhereCondition(condition).Exists(ctx)
	require.NoError(td.T, err, "failed to check existence in %s", modelName)
	assert.True(td.T, exists, "record should exist in %s", modelName)
}

// AssertNotExists checks if a record does not exist with given conditions
func (td *TestDatabase) AssertNotExists(modelName string, condition types.Condition) {
	ctx := context.Background()
	model := td.DB.Model(modelName)
	exists, err := model.WhereCondition(condition).Exists(ctx)
	require.NoError(td.T, err, "failed to check existence in %s", modelName)
	assert.False(td.T, exists, "record should not exist in %s", modelName)
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(timeout time.Duration, check func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// GetEnvOrDefault returns environment variable value or default

func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Test model structs for scanning results
type TestUser struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Age       *int      `db:"age"`
	Active    bool      `db:"active"`
	CreatedAt time.Time `db:"created_at"`
}

type TestPost struct {
	ID        int       `db:"id"`
	Title     string    `db:"title"`
	Content   *string   `db:"content"`
	UserID    int       `db:"user_id"`
	Published bool      `db:"published"`
	Views     int       `db:"views"`
	CreatedAt time.Time `db:"created_at"`
}

type TestComment struct {
	ID        int       `db:"id"`
	Content   string    `db:"content"`
	PostID    int       `db:"post_id"`
	UserID    int       `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

type TestTag struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}
