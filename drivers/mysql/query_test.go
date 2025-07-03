package mysql

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions to convert MySQL results to expected types
func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	case float64:
		return int64(val)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// Test models
type User struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Email     string    `db:"email"`
	Age       int       `db:"age"`
	City      string    `db:"city"`
	CreatedAt time.Time `db:"created_at"`
}

type Post struct {
	ID        int       `db:"id"`
	Title     string    `db:"title"`
	Content   string    `db:"content"`
	UserID    int       `db:"user_id"`
	Views     int       `db:"views"`
	Published bool      `db:"published"`
	CreatedAt time.Time `db:"created_at"`
}

type Comment struct {
	ID        int       `db:"id"`
	Content   string    `db:"content"`
	PostID    int       `db:"post_id"`
	UserID    int       `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

// getMySQLTestConfig returns test database configuration
func getMySQLTestConfig() types.Config {
	host := os.Getenv("MYSQL_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	portStr := os.Getenv("MYSQL_TEST_PORT")
	if portStr == "" {
		portStr = "3306"
	}
	port := 3306
	if p, err := strconv.Atoi(portStr); err == nil {
		port = p
	}

	user := os.Getenv("MYSQL_TEST_USER")
	if user == "" {
		user = "testuser"
	}

	password := os.Getenv("MYSQL_TEST_PASSWORD")
	if password == "" {
		password = "testpass"
	}

	database := os.Getenv("MYSQL_TEST_DATABASE")
	if database == "" {
		database = "testdb"
	}

	return types.Config{
		Type:     "mysql",
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: database,
		Options: map[string]string{
			"parseTime": "true",
		},
	}
}

// setupTestDB creates a test database with schema and test data
func setupTestDB(t *testing.T) *MySQLDB {
	if testing.Short() {
		t.Skip("Skipping MySQL integration test in short mode")
	}

	config := getMySQLTestConfig()

	// First connect without database to create it
	configWithoutDB := config
	configWithoutDB.Database = ""

	db, err := NewMySQLDB(configWithoutDB)
	if err != nil {
		t.Skipf("Failed to create MySQL connection: %v", err)
	}

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skipf("Failed to connect to MySQL: %v", err)
	}

	// Create test database
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", config.Database))
	require.NoError(t, err)

	// Close and reconnect with database
	db.Close()

	db, err = NewMySQLDB(config)
	require.NoError(t, err)

	err = db.Connect(ctx)
	require.NoError(t, err)

	// Drop existing tables
	tables := []string{"comments", "posts", "users"}
	for _, table := range tables {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	}

	// Create schemas
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true}).
		AddField(schema.Field{Name: "age", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "city", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "CURRENT_TIMESTAMP"})

	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "title", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "views", Type: schema.FieldTypeInt, Default: 0}).
		AddField(schema.Field{Name: "published", Type: schema.FieldTypeBool, Default: false}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "CURRENT_TIMESTAMP"})

	commentSchema := schema.New("Comment").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "postId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "CURRENT_TIMESTAMP"})

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

	// Register schemas
	err = db.RegisterSchema("User", userSchema)
	require.NoError(t, err)
	err = db.RegisterSchema("Post", postSchema)
	require.NoError(t, err)
	err = db.RegisterSchema("Comment", commentSchema)
	require.NoError(t, err)

	// Create tables
	err = db.CreateModel(ctx, "User")
	require.NoError(t, err)
	err = db.CreateModel(ctx, "Post")
	require.NoError(t, err)
	err = db.CreateModel(ctx, "Comment")
	require.NoError(t, err)

	// Insert test data
	insertTestData(t, db)

	return db
}

func insertTestData(t *testing.T, db *MySQLDB) {
	ctx := context.Background()

	// Insert users
	users := []map[string]any{
		{"name": "Alice", "email": "alice@example.com", "age": 25, "city": "New York"},
		{"name": "Bob", "email": "bob@example.com", "age": 30, "city": "Los Angeles"},
		{"name": "Charlie", "email": "charlie@example.com", "age": 25, "city": "New York"},
		{"name": "David", "email": "david@example.com", "age": 35, "city": "Chicago"},
		{"name": "Eve", "email": "eve@example.com", "age": 30, "city": "Los Angeles"},
	}

	for _, user := range users {
		_, err := db.Model("User").Insert(user).Exec(ctx)
		require.NoError(t, err)
	}

	// Insert posts
	posts := []map[string]any{
		{"title": "First Post", "content": "Hello World", "userId": 1, "views": 100, "published": true},
		{"title": "Second Post", "content": "Another post", "userId": 1, "views": 50, "published": false},
		{"title": "Bob's Post", "content": "Bob's content", "userId": 2, "views": 200, "published": true},
		{"title": "Charlie's Post", "content": "Charlie's content", "userId": 3, "views": 150, "published": true},
		{"title": "Draft Post", "content": "Work in progress", "userId": 2, "views": 0, "published": false},
	}

	for _, post := range posts {
		_, err := db.Model("Post").Insert(post).Exec(ctx)
		require.NoError(t, err)
	}

	// Insert comments
	comments := []map[string]any{
		{"content": "Great post!", "postId": 1, "userId": 2},
		{"content": "Thanks!", "postId": 1, "userId": 1},
		{"content": "Interesting", "postId": 3, "userId": 3},
		{"content": "Nice work", "postId": 4, "userId": 1},
	}

	for _, comment := range comments {
		_, err := db.Model("Comment").Insert(comment).Exec(ctx)
		require.NoError(t, err)
	}
}

// Basic Query Tests

func TestMySQLDB_BasicSelect(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Select all users
	var users []User
	err := db.Model("User").Select().FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 5)

	// Test 2: Select specific fields
	var names []map[string]any
	err = db.Model("User").Select("name", "email").FindMany(ctx, &names)
	require.NoError(t, err)
	assert.Len(t, names, 5)
	assert.Contains(t, names[0], "name")
	assert.Contains(t, names[0], "email")
	assert.NotContains(t, names[0], "age")

	// Test 3: FindFirst
	var user User
	err = db.Model("User").Select().FindFirst(ctx, &user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.Name)
}

func TestMySQLDB_WhereConditions(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Equals
	var users []User
	err := db.Model("User").Select().WhereCondition(
		db.Model("User").Where("name").Equals("Alice"),
	).FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].Name)

	// Test 2: Greater than
	users = []User{}
	err = db.Model("User").Select().WhereCondition(
		db.Model("User").Where("age").GreaterThan(25),
	).FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// Test 3: IN clause
	users = []User{}
	err = db.Model("User").Select().WhereCondition(
		db.Model("User").Where("city").In("New York", "Chicago"),
	).FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// Test 4: AND conditions
	users = []User{}
	err = db.Model("User").Select().WhereCondition(
		db.Model("User").Where("age").Equals(30).And(
			db.Model("User").Where("city").Equals("Los Angeles"),
		),
	).FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 2)

	// Test 5: OR conditions
	users = []User{}
	err = db.Model("User").Select().WhereCondition(
		db.Model("User").Where("name").Equals("Alice").Or(
			db.Model("User").Where("name").Equals("Bob"),
		),
	).FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 2)

	// Test 6: LIKE operations
	users = []User{}
	err = db.Model("User").Select().WhereCondition(
		db.Model("User").Where("name").Contains("li"),
	).FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 2) // Alice and Charlie
}

func TestMySQLDB_OrderBy(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Order by age ASC
	var users []User
	err := db.Model("User").Select().
		OrderBy("age", types.ASC).
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Equal(t, 25, users[0].Age)
	assert.Equal(t, 35, users[len(users)-1].Age)

	// Test 2: Order by age DESC
	users = []User{}
	err = db.Model("User").Select().
		OrderBy("age", types.DESC).
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Equal(t, 35, users[0].Age)
	assert.Equal(t, 25, users[len(users)-1].Age)

	// Test 3: Multiple order by
	users = []User{}
	err = db.Model("User").Select().
		OrderBy("age", types.ASC).
		OrderBy("name", types.ASC).
		FindMany(ctx, &users)
	require.NoError(t, err)
	// First two should be age 25, ordered by name
	assert.Equal(t, 25, users[0].Age)
	assert.Equal(t, "Alice", users[0].Name)
	assert.Equal(t, 25, users[1].Age)
	assert.Equal(t, "Charlie", users[1].Name)
}

func TestMySQLDB_LimitOffset(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Limit
	var users []User
	err := db.Model("User").Select().
		OrderBy("id", types.ASC).
		Limit(3).
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// Test 2: Offset
	users = []User{}
	err = db.Model("User").Select().
		OrderBy("id", types.ASC).
		Offset(2).
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 3)
	assert.Equal(t, "Charlie", users[0].Name)

	// Test 3: Limit + Offset (pagination)
	users = []User{}
	err = db.Model("User").Select().
		OrderBy("id", types.ASC).
		Limit(2).
		Offset(2).
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "Charlie", users[0].Name)
	assert.Equal(t, "David", users[1].Name)
}

// Advanced Query Tests

func TestMySQLDB_Distinct(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Distinct cities
	var results []map[string]any
	err := db.Model("User").Select("city").
		Distinct().
		OrderBy("city", types.ASC).
		FindMany(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3) // New York, Los Angeles, Chicago

	// Test 2: Distinct ages
	results = []map[string]any{}
	err = db.Model("User").Select("age").
		Distinct().
		OrderBy("age", types.ASC).
		FindMany(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3) // 25, 30, 35

	// Test 3: Count distinct using raw SQL
	var countResult map[string]any
	err = db.Raw("SELECT COUNT(DISTINCT city) as count FROM users").
		FindOne(ctx, &countResult)
	require.NoError(t, err)
	assert.Equal(t, int64(3), countResult["count"])
}

func TestMySQLDB_GroupBy(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Group by city with count
	var results []map[string]any
	sql := `
		SELECT city, COUNT(*) as count 
		FROM users 
		GROUP BY city 
		ORDER BY city
	`
	err := db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify counts
	cityCount := make(map[string]int64)
	for _, r := range results {
		cityCount[r["city"].(string)] = r["count"].(int64)
	}
	assert.Equal(t, int64(1), cityCount["Chicago"])
	assert.Equal(t, int64(2), cityCount["Los Angeles"])
	assert.Equal(t, int64(2), cityCount["New York"])

	// Test 2: Group by age with aggregations
	sql = `
		SELECT age, 
			   COUNT(*) as count,
			   MIN(id) as min_id,
			   MAX(id) as max_id
		FROM users 
		GROUP BY age 
		ORDER BY age
	`
	results = []map[string]any{}
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Test 3: Group by with HAVING
	sql = `
		SELECT city, COUNT(*) as count 
		FROM users 
		GROUP BY city 
		HAVING COUNT(*) > 1
		ORDER BY city
	`
	results = []map[string]any{}
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Only LA and NY have count > 1

	// Test 4: Group by on posts with aggregations
	sql = `
		SELECT user_id,
			   COUNT(*) as post_count,
			   SUM(views) as total_views,
			   AVG(views) as avg_views,
			   MAX(views) as max_views
		FROM posts
		GROUP BY user_id
		ORDER BY user_id
	`
	results = []map[string]any{}
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3) // Users 1, 2, 3 have posts
}

func TestMySQLDB_Include(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Include posts for user (one-to-many)
	// Note: Current implementation has issues with column name ambiguity
	// This test documents the expected behavior once fixed

	t.Run("Include posts for user", func(t *testing.T) {
		// Currently this will fail with "ambiguous column name: id"
		// Once fixed, it should work as follows:
		var user map[string]any
		err := db.Model("User").Select().
			WhereCondition(db.Model("User").Where("id").Equals(1)).
			Include("posts").
			FindFirst(ctx, &user)

		// Include should now work correctly
		require.NoError(t, err)
		assert.Equal(t, "Alice", user["name"])
		posts, ok := user["posts"].([]any)
		assert.True(t, ok)
		assert.Len(t, posts, 2)

		// Verify post data
		if len(posts) >= 2 {
			post1 := posts[0].(map[string]any)
			post2 := posts[1].(map[string]any)
			assert.Equal(t, "First Post", post1["title"])
			assert.Equal(t, "Second Post", post2["title"])
		}
	})

	// Test 2: Manual join query (workaround)
	t.Run("Manual join for user posts", func(t *testing.T) {
		var results []map[string]any
		sql := `
			SELECT u.id as user_id, u.name, u.email,
			       p.id as post_id, p.title, p.views
			FROM users u
			LEFT JOIN posts p ON p.user_id = u.id
			WHERE u.id = ?
			ORDER BY p.id
		`
		err := db.Raw(sql, 1).Find(ctx, &results)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Alice has 2 posts
		assert.Equal(t, "Alice", results[0]["name"])
		assert.Equal(t, "First Post", results[0]["title"])
	})
}

func TestMySQLDB_ComplexQueries(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Complex WHERE with AND/OR
	var posts []Post
	err := db.Model("Post").Select().WhereCondition(
		db.Model("Post").Where("published").Equals(true).And(
			db.Model("Post").Where("views").GreaterThan(100).Or(
				db.Model("Post").Where("userId").Equals(3),
			),
		),
	).FindMany(ctx, &posts)
	require.NoError(t, err)
	assert.Len(t, posts, 2) // Bob's post (200 views) and Charlie's post (userId=3)

	// Test 2: Subquery-like operation using raw SQL
	var results []map[string]any
	sql := `
		SELECT u.name, COUNT(p.id) as post_count
		FROM users u
		LEFT JOIN posts p ON p.user_id = u.id
		GROUP BY u.id, u.name
		HAVING COUNT(p.id) > 0
		ORDER BY post_count DESC
	`
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	// Both Alice and Bob have 2 posts, so accept either one
	firstName := results[0]["name"].(string)
	assert.True(t, firstName == "Alice" || firstName == "Bob", "Expected Alice or Bob but got %s", firstName)
	assert.Equal(t, int64(2), results[0]["post_count"].(int64))
}

func TestMySQLDB_Aggregations(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Count
	count, err := db.Model("User").Select().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Test 2: Count with condition
	count, err = db.Model("Post").Select().
		WhereCondition(db.Model("Post").Where("published").Equals(true)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Test 3: Other aggregations using raw SQL
	var result map[string]any
	sql := `
		SELECT 
			COUNT(*) as count,
			SUM(views) as total_views,
			AVG(views) as avg_views,
			MIN(views) as min_views,
			MAX(views) as max_views
		FROM posts
		WHERE published = ?
	`
	err = db.Raw(sql, true).FindOne(ctx, &result)
	require.NoError(t, err)
	
	// MySQL returns aggregation results as strings or int64 depending on the driver
	// Convert to appropriate types for comparison
	countResult := toInt64(result["count"])
	totalViews := toInt64(result["total_views"])
	avgViews := toFloat64(result["avg_views"])
	minViews := toInt64(result["min_views"])
	maxViews := toInt64(result["max_views"])
	
	assert.Equal(t, int64(3), countResult)
	assert.Equal(t, int64(450), totalViews) // 100 + 200 + 150
	assert.Equal(t, float64(150), avgViews)
	assert.Equal(t, int64(100), minViews)
	assert.Equal(t, int64(200), maxViews)
}

func TestMySQLDB_NullValues(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Insert a user with nullable field
	_, err := db.Exec("ALTER TABLE users ADD COLUMN nickname VARCHAR(255)")
	require.NoError(t, err)

	// Use raw SQL since nickname is not in the schema
	_, err = db.Exec(`INSERT INTO users (name, email, age, city, nickname) VALUES (?, ?, ?, ?, ?)`,
		"Frank", "frank@example.com", 40, "Boston", nil)
	require.NoError(t, err)

	// Test querying with NULL
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE nickname IS NULL").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 6) // All users have NULL nickname

	// Test NOT NULL
	_, err = db.Exec(`INSERT INTO users (name, email, age, city, nickname) VALUES (?, ?, ?, ?, ?)`,
		"Grace", "grace@example.com", 28, "Seattle", "Gracie")
	require.NoError(t, err)

	results = []map[string]any{}
	err = db.Raw("SELECT * FROM users WHERE nickname IS NOT NULL").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Grace", results[0]["name"])
}

func TestMySQLDB_FieldNameMapping(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test that camelCase field names are properly mapped to snake_case columns
	var posts []Post
	err := db.Model("Post").Select().
		WhereCondition(db.Model("Post").Where("userId").Equals(1)).
		FindMany(ctx, &posts)
	require.NoError(t, err)
	assert.Len(t, posts, 2)

	// Verify the actual SQL column name is user_id
	var result map[string]any
	err = db.Raw("SELECT user_id FROM posts WHERE id = 1").FindOne(ctx, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result["user_id"])
}

func TestMySQLDB_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test 1: Invalid field name
	var users []User
	err := db.Model("User").Select().
		WhereCondition(db.Model("User").Where("invalid_field").Equals("value")).
		FindMany(ctx, &users)
	assert.Error(t, err)

	// Test 2: Invalid model name
	err = db.Model("InvalidModel").Select().FindMany(ctx, &users)
	assert.Error(t, err)

	// Test 3: Unique constraint violation
	_, err = db.Model("User").Insert(map[string]any{
		"name":  "Duplicate",
		"email": "alice@example.com", // Already exists
		"age":   30,
		"city":  "Boston",
	}).Exec(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Duplicate entry")
}

func TestMySQLDB_BooleanHandling(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// MySQL stores booleans as TINYINT(1)
	// Test that boolean values work correctly
	var posts []Post
	err := db.Model("Post").Select().
		WhereCondition(db.Model("Post").Where("published").Equals(true)).
		FindMany(ctx, &posts)
	require.NoError(t, err)
	assert.Len(t, posts, 3)

	for _, post := range posts {
		assert.True(t, post.Published)
	}

	// Test false values
	posts = []Post{}
	err = db.Model("Post").Select().
		WhereCondition(db.Model("Post").Where("published").Equals(false)).
		FindMany(ctx, &posts)
	require.NoError(t, err)
	assert.Len(t, posts, 2)

	for _, post := range posts {
		assert.False(t, post.Published)
	}
}

func TestMySQLDB_DateTimeHandling(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// Test that datetime values are properly handled
	var users []User
	err := db.Model("User").Select().
		OrderBy("createdAt", types.DESC).
		Limit(1).
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 1)

	// Verify createdAt is not zero
	assert.False(t, users[0].CreatedAt.IsZero())

	// Test date comparison
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	users = []User{}
	err = db.Model("User").Select().
		WhereCondition(db.Model("User").Where("createdAt").GreaterThan(oneHourAgo)).
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 5) // All users were just created
}

func TestMySQLDB_CaseSensitivity(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		_, _ = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db.Config.Database))
		db.Close()
	}()

	ctx := context.Background()

	// MySQL is case-insensitive by default for string comparisons
	var users []User
	err := db.Model("User").Select().
		WhereCondition(db.Model("User").Where("name").Equals("alice")). // lowercase
		FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].Name) // Actual name is capitalized
}
