package test

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Query Building Tests =====

func (dct *DriverConformanceTests) TestWhereEquals(t *testing.T) {
	if dct.shouldSkip("TestWhereEquals") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test equals condition
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("name").Equals("Alice")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].Name)

	// Test equals with different types
	Post := td.DB.Model("Post")
	var posts []TestPost
	err = Post.Select().
		WhereCondition(Post.Where("userId").Equals(1)).
		FindMany(ctx, &posts)
	assert.NoError(t, err)
	assert.Len(t, posts, 2)
}

func (dct *DriverConformanceTests) TestWhereNotEquals(t *testing.T) {
	if dct.shouldSkip("TestWhereNotEquals") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test not equals condition
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("name").NotEquals("Alice")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 4)
	for _, u := range users {
		assert.NotEqual(t, "Alice", u.Name)
	}
}

func (dct *DriverConformanceTests) TestWhereComparisons(t *testing.T) {
	if dct.shouldSkip("TestWhereComparisons") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test GreaterThan
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("age").GreaterThan(28)).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 2) // Bob (30) and Charlie (35)

	// Test GreaterThanOrEqual
	users = []TestUser{}
	err = User.Select().
		WhereCondition(User.Where("age").GreaterThanOrEqual(30)).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// Test LessThan
	users = []TestUser{}
	err = User.Select().
		WhereCondition(User.Where("age").LessThan(30)).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 2) // Alice (25) and Eve (28)

	// Test LessThanOrEqual
	users = []TestUser{}
	err = User.Select().
		WhereCondition(User.Where("age").LessThanOrEqual(25)).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
}

func (dct *DriverConformanceTests) TestWhereIn(t *testing.T) {
	if dct.shouldSkip("TestWhereIn") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test IN condition
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("name").In("Alice", "Bob", "Charlie")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 3)

	// Test IN with integers
	Post := td.DB.Model("Post")
	var posts []TestPost
	err = Post.Select().
		WhereCondition(Post.Where("id").In(1, 3, 5)).
		FindMany(ctx, &posts)
	assert.NoError(t, err)
	assert.Len(t, posts, 3)
}

func (dct *DriverConformanceTests) TestWhereNotIn(t *testing.T) {
	if dct.shouldSkip("TestWhereNotIn") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test NOT IN condition
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("name").NotIn("Alice", "Bob")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 3)
}

func (dct *DriverConformanceTests) TestWhereLike(t *testing.T) {
	if dct.shouldSkip("TestWhereLike") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test Contains
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("email").Contains("example.com")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 5)

	// Test StartsWith
	users = []TestUser{}
	err = User.Select().
		WhereCondition(User.Where("name").StartsWith("A")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1)

	// Test EndsWith
	users = []TestUser{}
	err = User.Select().
		WhereCondition(User.Where("email").EndsWith("@example.com")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 5)

	// Test Like pattern
	users = []TestUser{}
	err = User.Select().
		WhereCondition(User.Where("name").Like("%e%")).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Greater(t, len(users), 0)
}

func (dct *DriverConformanceTests) TestWhereNull(t *testing.T) {
	if dct.shouldSkip("TestWhereNull") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test IsNull
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("age").IsNull()).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1) // David has null age

	// Test IsNotNull
	users = []TestUser{}
	err = User.Select().
		WhereCondition(User.Where("age").IsNotNull()).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 4)

	// Test null content in posts
	Post := td.DB.Model("Post")
	var posts []TestPost
	err = Post.Select().
		WhereCondition(Post.Where("content").IsNull()).
		FindMany(ctx, &posts)
	assert.NoError(t, err)
	assert.Len(t, posts, 1) // Charlie's Draft has null content
}

func (dct *DriverConformanceTests) TestWhereBetween(t *testing.T) {
	if dct.shouldSkip("TestWhereBetween") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test Between
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("age").Between(25, 30)).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 3) // Alice (25), Bob (30), Eve (28)

	// Test Between with views
	Post := td.DB.Model("Post")
	var posts []TestPost
	err = Post.Select().
		WhereCondition(Post.Where("views").Between(50, 200)).
		FindMany(ctx, &posts)
	assert.NoError(t, err)
	assert.Len(t, posts, 2) // 50 and 100 are between 50 and 200
}

func (dct *DriverConformanceTests) TestComplexWhereConditions(t *testing.T) {
	if dct.shouldSkip("TestComplexWhereConditions") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test AND conditions
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(
			User.Where("age").GreaterThanOrEqual(25).And(
				User.Where("active").Equals(true),
			),
		).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 3) // Alice (25), Bob (30), Eve (28) - all active with age >= 25

	// Test OR conditions
	users = []TestUser{}
	err = User.Select().
		WhereCondition(
			User.Where("name").Equals("Alice").Or(
				User.Where("name").Equals("Bob"),
			),
		).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// Test NOT condition
	users = []TestUser{}
	err = User.Select().
		WhereCondition(
			User.Where("active").Equals(true).Not(),
		).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1) // Only Charlie

	// Test complex nested conditions
	Post := td.DB.Model("Post")
	var posts []TestPost
	err = Post.Select().
		WhereCondition(
			Post.Where("published").Equals(true).And(
				Post.Where("views").GreaterThan(50).Or(
					Post.Where("userId").Equals(1),
				),
			),
		).
		FindMany(ctx, &posts)
	assert.NoError(t, err)
	assert.Greater(t, len(posts), 0)
}

// ===== Advanced Query Tests =====

func (dct *DriverConformanceTests) TestOrderBy(t *testing.T) {
	if dct.shouldSkip("TestOrderBy") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test ORDER BY ASC
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		OrderBy("name", types.ASC).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", users[0].Name)
	assert.Equal(t, "Eve", users[len(users)-1].Name)

	// Test ORDER BY DESC
	users = []TestUser{}
	err = User.Select().
		OrderBy("age", types.DESC).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	// Charlie has highest age (35)
	assert.Equal(t, "Charlie", users[0].Name)
}

func (dct *DriverConformanceTests) TestOrderByMultiple(t *testing.T) {
	if dct.shouldSkip("TestOrderByMultiple") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Add some users with same age for testing
	User := td.DB.Model("User")
	_, err = User.Insert(map[string]any{
		"name":   "Frank",
		"email":  "frank@example.com",
		"age":    25,
		"active": true,
	}).Exec(ctx)
	require.NoError(t, err)

	// Test multiple ORDER BY
	var users []TestUser
	err = User.Select().
		OrderBy("age", types.ASC).
		OrderBy("name", types.ASC).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	
	// Users with age 25 should be ordered by name
	var age25Users []string
	for _, u := range users {
		if u.Age != nil && *u.Age == 25 {
			age25Users = append(age25Users, u.Name)
		}
	}
	assert.Equal(t, []string{"Alice", "Frank"}, age25Users)
}

func (dct *DriverConformanceTests) TestGroupBy(t *testing.T) {
	if dct.shouldSkip("TestGroupBy") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test GROUP BY with raw query (as query builder doesn't expose aggregates yet)
	var results []map[string]any
	err = td.DB.Raw("SELECT user_id, COUNT(*) as post_count FROM posts GROUP BY user_id ORDER BY user_id").
		Find(ctx, &results)
	assert.NoError(t, err)
	assert.Greater(t, len(results), 0)
}

func (dct *DriverConformanceTests) TestHaving(t *testing.T) {
	if dct.shouldSkip("TestHaving") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test HAVING with raw query
	var results []map[string]any
	err = td.DB.Raw("SELECT user_id, COUNT(*) as post_count FROM posts GROUP BY user_id HAVING COUNT(*) > 1").
		Find(ctx, &results)
	assert.NoError(t, err)
	assert.Greater(t, len(results), 0)
}

func (dct *DriverConformanceTests) TestLimit(t *testing.T) {
	if dct.shouldSkip("TestLimit") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test LIMIT
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		OrderBy("id", types.ASC).
		Limit(3).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 3)
}

func (dct *DriverConformanceTests) TestOffset(t *testing.T) {
	if dct.shouldSkip("TestOffset") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test OFFSET
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		OrderBy("id", types.ASC).
		Offset(2).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 3) // 5 total - 2 offset = 3
}

func (dct *DriverConformanceTests) TestLimitWithOffset(t *testing.T) {
	if dct.shouldSkip("TestLimitWithOffset") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test pagination with LIMIT and OFFSET
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		OrderBy("id", types.ASC).
		Limit(2).
		Offset(1).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func (dct *DriverConformanceTests) TestDistinct(t *testing.T) {
	if dct.shouldSkip("TestDistinct") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Add duplicate data
	User := td.DB.Model("User")
	_, err = User.Insert(map[string]any{
		"name":   "Alice",
		"email":  "alice2@example.com",
		"age":    25,
		"active": true,
	}).Exec(ctx)
	require.NoError(t, err)

	// Test DISTINCT
	var results []map[string]any
	err = User.Select("name").
		Distinct().
		OrderBy("name", types.ASC).
		FindMany(ctx, &results)
	assert.NoError(t, err)
	assert.Len(t, results, 5) // Should not include duplicate Alice
}

func (dct *DriverConformanceTests) TestCount(t *testing.T) {
	if dct.shouldSkip("TestCount") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test COUNT all
	User := td.DB.Model("User")
	count, err := User.Select().Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Test COUNT with condition
	count, err = User.Select().
		WhereCondition(User.Where("active").Equals(true)).
		Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(4), count)
}

func (dct *DriverConformanceTests) TestAggregations(t *testing.T) {
	if dct.shouldSkip("TestAggregations") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test aggregations using raw SQL (as query builder might not expose all)
	var result map[string]any
	err = td.DB.Raw(`
		SELECT 
			COUNT(*) as count,
			SUM(views) as total_views,
			AVG(views) as avg_views,
			MIN(views) as min_views,
			MAX(views) as max_views
		FROM posts
		WHERE published = ?
	`, true).FindOne(ctx, &result)
	assert.NoError(t, err)
	
	// MySQL returns aggregation results as strings, so use conversion utilities
	countResult := utils.ToInt64(result["count"])
	totalViews := utils.ToInt64(result["total_views"])
	avgViews := utils.ToFloat64(result["avg_views"])
	minViews := utils.ToInt64(result["min_views"])
	maxViews := utils.ToInt64(result["max_views"])
	
	assert.Equal(t, int64(3), countResult)
	assert.Equal(t, int64(1150), totalViews) // 100 + 50 + 1000
	assert.InDelta(t, float64(383.33), avgViews, 0.5) // Average of 100, 50, 1000
	assert.Equal(t, int64(50), minViews)
	assert.Equal(t, int64(1000), maxViews)
}

// ===== Include/Join Tests =====

func (dct *DriverConformanceTests) TestInclude(t *testing.T) {
	if dct.shouldSkip("TestInclude") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test 1: Include posts for user (one-to-many)
	t.Run("Include posts for user", func(t *testing.T) {
		User := td.DB.Model("User")
		var user map[string]any
		err := User.Select().
			WhereCondition(User.Where("id").Equals(1)).
			Include("posts").
			FindFirst(ctx, &user)

		// Include should work correctly
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

	// Test 2: Manual join query (workaround for drivers that don't support Include)
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
		err := td.DB.Raw(sql, 1).Find(ctx, &results)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Alice has 2 posts
		assert.Equal(t, "Alice", results[0]["name"])
		assert.Equal(t, "First Post", results[0]["title"])
	})
}

// ===== Complex Query Tests =====

func (dct *DriverConformanceTests) TestComplexQueries(t *testing.T) {
	if dct.shouldSkip("TestComplexQueries") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test 1: Complex WHERE with AND/OR
	Post := td.DB.Model("Post")
	var posts []TestPost
	err = Post.Select().WhereCondition(
		Post.Where("published").Equals(true).And(
			Post.Where("views").GreaterThan(100).Or(
				Post.Where("userId").Equals(3),
			),
		),
	).FindMany(ctx, &posts)
	require.NoError(t, err)
	assert.Len(t, posts, 1) // Only "Popular Post" matches (views=1000 > 100)

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
	err = td.DB.Raw(sql).Find(ctx, &results)
	require.NoError(t, err)
	// Both Alice and Bob have 2 posts, so accept either one
	firstName := results[0]["name"].(string)
	assert.True(t, firstName == "Alice" || firstName == "Bob", "Expected Alice or Bob but got %s", firstName)
	assert.Equal(t, int64(2), utils.ToInt64(results[0]["post_count"]))
}

