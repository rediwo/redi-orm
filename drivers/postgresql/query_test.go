package postgresql

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

// getPostgreSQLTestConfig returns test database configuration
func getPostgreSQLTestConfig() types.Config {
	host := os.Getenv("POSTGRES_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	user := os.Getenv("POSTGRES_TEST_USER")
	if user == "" {
		user = "testuser"
	}

	password := os.Getenv("POSTGRES_TEST_PASSWORD")
	if password == "" {
		password = "testpass"
	}

	database := os.Getenv("POSTGRES_TEST_DATABASE")
	if database == "" {
		database = "testdb"
	}

	return types.Config{
		Type:     "postgresql",
		Host:     host,
		Port:     5432,
		User:     user,
		Password: password,
		Database: database,
		Options: map[string]string{
			"sslmode": "disable",
		},
	}
}

// cleanupTables removes all non-system tables from the database
func cleanupTables(t *testing.T, db *PostgreSQLDB) {
	ctx := context.Background()
	
	// Get all tables in public schema
	rows, err := db.DB.QueryContext(ctx, `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public'
	`)
	if err != nil {
		t.Logf("Failed to get tables: %v", err)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Logf("Failed to scan table name: %v", err)
			continue
		}
		tables = append(tables, table)
	}

	// Drop all tables with CASCADE
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE`, table))
		if err != nil {
			t.Logf("Failed to drop table %s: %v", table, err)
		}
	}
}

// setupTestDB creates a test database with schema and test data
func setupTestDB(t *testing.T) *PostgreSQLDB {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL integration test in short mode")
	}

	config := getPostgreSQLTestConfig()
	
	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Skipf("Failed to create PostgreSQL connection: %v", err)
	}

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skipf("Failed to connect to PostgreSQL: %v", err)
	}

	// Clean up existing tables
	cleanupTables(t, db)

	// Create schemas
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true}).
		AddField(schema.Field{Name: "age", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "city", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "now()"})

	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "title", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "views", Type: schema.FieldTypeInt, Default: 0}).
		AddField(schema.Field{Name: "published", Type: schema.FieldTypeBool, Default: false}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "now()"})

	commentSchema := schema.New("Comment").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "postId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "userId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime, Default: "now()"})

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

	// Add cleanup
	t.Cleanup(func() {
		cleanupTables(t, db)
		db.Close()
	})

	return db
}

func insertTestData(t *testing.T, db *PostgreSQLDB) {
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

func TestPostgreSQLDB_BasicSelect(t *testing.T) {
	db := setupTestDB(t)

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

func TestPostgreSQLDB_WhereConditions(t *testing.T) {
	db := setupTestDB(t)

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

	// Test 7: Case-sensitive comparison (PostgreSQL specific)
	users = []User{}
	err = db.Model("User").Select().WhereCondition(
		db.Model("User").Where("name").Equals("alice"), // lowercase
	).FindMany(ctx, &users)
	require.NoError(t, err)
	assert.Len(t, users, 0) // PostgreSQL is case-sensitive
}

func TestPostgreSQLDB_OrderBy(t *testing.T) {
	db := setupTestDB(t)

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

func TestPostgreSQLDB_LimitOffset(t *testing.T) {
	db := setupTestDB(t)

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

func TestPostgreSQLDB_Distinct(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// Test 1: Distinct cities
	var results []map[string]any
	err := db.Model("User").Select("city").
		Distinct().
		OrderBy("city", types.ASC).
		FindMany(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3) // Chicago, Los Angeles, New York

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

	// Test 4: DISTINCT ON (PostgreSQL specific)
	results = []map[string]any{}
	sql := `
		SELECT DISTINCT ON (age) age, name, city 
		FROM users 
		ORDER BY age, name
	`
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3) // One user per distinct age
}

func TestPostgreSQLDB_GroupBy(t *testing.T) {
	db := setupTestDB(t)

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

	// Test 5: Window functions (PostgreSQL specific)
	sql = `
		SELECT name, age, city,
			   ROW_NUMBER() OVER (PARTITION BY city ORDER BY age) as row_num,
			   COUNT(*) OVER (PARTITION BY city) as city_count
		FROM users
		ORDER BY city, age
	`
	results = []map[string]any{}
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 5)
}

func TestPostgreSQLDB_Include(t *testing.T) {
	db := setupTestDB(t)

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
			WHERE u.id = $1
			ORDER BY p.id
		`
		err := db.Raw(sql, 1).Find(ctx, &results)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Alice has 2 posts
		assert.Equal(t, "Alice", results[0]["name"])
		assert.Equal(t, "First Post", results[0]["title"])
	})

	// Test 3: Complex join with multiple relations
	t.Run("Complex join query", func(t *testing.T) {
		var results []map[string]any
		sql := `
			SELECT u.name as user_name,
			       p.title as post_title,
			       c.content as comment_content,
			       cu.name as commenter_name
			FROM posts p
			JOIN users u ON u.id = p.user_id
			LEFT JOIN comments c ON c.post_id = p.id
			LEFT JOIN users cu ON cu.id = c.user_id
			WHERE p.published = true
			ORDER BY p.id, c.id
		`
		err := db.Raw(sql).Find(ctx, &results)
		require.NoError(t, err)
		assert.Greater(t, len(results), 0)
	})
}

func TestPostgreSQLDB_ComplexQueries(t *testing.T) {
	db := setupTestDB(t)

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

	// Test 3: CTE (Common Table Expression) - PostgreSQL specific
	sql = `
		WITH user_post_counts AS (
			SELECT user_id, COUNT(*) as post_count
			FROM posts
			GROUP BY user_id
		)
		SELECT u.name, COALESCE(upc.post_count, 0) as post_count
		FROM users u
		LEFT JOIN user_post_counts upc ON upc.user_id = u.id
		ORDER BY post_count DESC, u.name
	`
	results = []map[string]any{}
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 5)
}

func TestPostgreSQLDB_Aggregations(t *testing.T) {
	db := setupTestDB(t)

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
			AVG(views)::float as avg_views,
			MIN(views) as min_views,
			MAX(views) as max_views
		FROM posts
		WHERE published = $1
	`
	err = db.Raw(sql, true).FindOne(ctx, &result)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result["count"])
	assert.Equal(t, int64(450), result["total_views"]) // 100 + 200 + 150
	assert.InDelta(t, float64(150), result["avg_views"], 0.01)
	assert.Equal(t, int64(100), result["min_views"])
	assert.Equal(t, int64(200), result["max_views"])

	// Test 4: Array aggregation (PostgreSQL specific)
	sql = `
		SELECT city, array_agg(name ORDER BY name) as names
		FROM users
		GROUP BY city
		ORDER BY city
	`
	results := []map[string]any{}
	err = db.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestPostgreSQLDB_NullValues(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// Insert a user with nullable field
	_, err := db.Exec("ALTER TABLE users ADD COLUMN nickname VARCHAR(255)")
	require.NoError(t, err)

	// Use raw SQL since nickname is not in the schema
	_, err = db.Exec(`INSERT INTO users (name, email, age, city, nickname) VALUES ($1, $2, $3, $4, $5)`,
		"Frank", "frank@example.com", 40, "Boston", nil)
	require.NoError(t, err)

	// Test querying with NULL
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE nickname IS NULL").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 6) // All users have NULL nickname

	// Test NOT NULL
	_, err = db.Exec(`INSERT INTO users (name, email, age, city, nickname) VALUES ($1, $2, $3, $4, $5)`,
		"Grace", "grace@example.com", 28, "Seattle", "Gracie")
	require.NoError(t, err)

	results = []map[string]any{}
	err = db.Raw("SELECT * FROM users WHERE nickname IS NOT NULL").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Grace", results[0]["name"])

	// Test COALESCE
	results = []map[string]any{}
	err = db.Raw("SELECT name, COALESCE(nickname, 'No nickname') as display_name FROM users ORDER BY name").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 7)
}

func TestPostgreSQLDB_FieldNameMapping(t *testing.T) {
	db := setupTestDB(t)

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

func TestPostgreSQLDB_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)

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
	assert.Contains(t, err.Error(), "duplicate key value")

	// Test 4: Foreign key constraint violation
	_, err = db.Model("Post").Insert(map[string]any{
		"title":     "Invalid Post",
		"content":   "No such user",
		"userId":    999, // Non-existent user
		"published": true,
	}).Exec(ctx)
	assert.Error(t, err)
}

func TestPostgreSQLDB_ArrayTypes(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// PostgreSQL supports array types
	_, err := db.Exec("ALTER TABLE users ADD COLUMN tags TEXT[]")
	require.NoError(t, err)

	// Insert user with array
	_, err = db.Exec("INSERT INTO users (name, email, age, city, tags) VALUES ($1, $2, $3, $4, $5)",
		"Henry", "henry@example.com", 28, "Portland", "{developer,golang,postgres}")
	require.NoError(t, err)

	// Query array contains
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE 'golang' = ANY(tags)").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Henry", results[0]["name"])
}

func TestPostgreSQLDB_JSONTypes(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// PostgreSQL supports JSON/JSONB types
	_, err := db.Exec("ALTER TABLE users ADD COLUMN metadata JSONB")
	require.NoError(t, err)

	// Insert user with JSON
	_, err = db.Exec(`INSERT INTO users (name, email, age, city, metadata) 
		VALUES ($1, $2, $3, $4, $5)`,
		"Ivy", "ivy@example.com", 32, "Austin",
		`{"role": "admin", "permissions": ["read", "write", "delete"]}`)
	require.NoError(t, err)

	// Query JSON field
	var results []map[string]any
	err = db.Raw("SELECT * FROM users WHERE metadata->>'role' = 'admin'").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Ivy", results[0]["name"])

	// Query JSON array contains - use @> operator instead of ? to avoid placeholder conflict
	err = db.Raw("SELECT * FROM users WHERE metadata->'permissions' @> '\"delete\"'::jsonb").Find(ctx, &results)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}
