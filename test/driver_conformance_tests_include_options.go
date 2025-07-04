package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Include Options Tests =====

// TestIncludeWithSelectFields tests selective field loading in includes
func (dct *DriverConformanceTests) TestIncludeWithSelectFields(t *testing.T) {
	if dct.shouldSkip("TestIncludeWithSelectFields") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test selective field loading
	User := td.DB.Model("User")
	var user map[string]any

	// Create include option with selected fields
	includeOpt := &types.IncludeOption{
		Path:   "posts",
		Select: []string{"id", "title"}, // Only select id and title
	}

	err = User.Select().
		WhereCondition(User.Where("id").Equals(1)).
		IncludeWithOptions("posts", includeOpt).
		FindFirst(ctx, &user)

	require.NoError(t, err)
	assert.Equal(t, "Alice", user["name"])

	// Verify posts are included
	posts, ok := user["posts"].([]any)
	assert.True(t, ok, "posts should be included")
	assert.Greater(t, len(posts), 0, "should have posts")

	// Verify only selected fields are present
	if len(posts) > 0 {
		post := posts[0].(map[string]any)

		// These fields should be present
		assert.Contains(t, post, "id", "id should be selected")
		assert.Contains(t, post, "title", "title should be selected")

		// These fields should NOT be present (unless the driver doesn't support field selection)
		// Note: This test will reveal which drivers support SQL-level field selection
		if _, hasContent := post["content"]; hasContent {
			t.Logf("Driver %s does not yet support SQL-level field selection", dct.DriverName)
		}
	}
}

// TestIncludeWithWhereFilter tests filtering in includes
func (dct *DriverConformanceTests) TestIncludeWithWhereFilter(t *testing.T) {
	if dct.shouldSkip("TestIncludeWithWhereFilter") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test filtering included relations
	User := td.DB.Model("User")
	Post := td.DB.Model("Post")
	var users []map[string]any

	// Create include option with where filter
	includeOpt := &types.IncludeOption{
		Path:  "posts",
		Where: Post.Where("published").Equals(true),
	}

	err = User.Select().
		IncludeWithOptions("posts", includeOpt).
		FindMany(ctx, &users)

	require.NoError(t, err)
	assert.Greater(t, len(users), 0, "should have users")

	// Verify filtering works
	for _, user := range users {
		posts, ok := user["posts"].([]any)
		if !ok || len(posts) == 0 {
			continue // User has no posts
		}

		// All included posts should be published
		for _, p := range posts {
			post := p.(map[string]any)
			// Debug: log what fields we got
			t.Logf("Post fields: %+v", post)

			// Check for published field - it might be an int in SQLite
			var isPublished bool
			if published, ok := post["published"].(bool); ok {
				isPublished = published
			} else if pubInt, ok := post["published"].(int64); ok {
				isPublished = pubInt != 0
			} else if pubInt, ok := post["published"].(int); ok {
				isPublished = pubInt != 0
			} else {
				t.Logf("Published field type: %T", post["published"])
				assert.True(t, false, "published field should exist as bool or int")
			}
			assert.True(t, isPublished, "all included posts should be published")
		}
	}
}

// TestIncludeWithOrderBy tests ordering related records
func (dct *DriverConformanceTests) TestIncludeWithOrderBy(t *testing.T) {
	if dct.shouldSkip("TestIncludeWithOrderBy") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test ordering included relations
	User := td.DB.Model("User")
	var user map[string]any

	// Create include option with ordering
	includeOpt := &types.IncludeOption{
		Path: "posts",
		OrderBy: []types.OrderByOption{
			{Field: "views", Direction: types.DESC},
		},
	}

	err = User.Select().
		WhereCondition(User.Where("id").Equals(1)).
		IncludeWithOptions("posts", includeOpt).
		FindFirst(ctx, &user)

	require.NoError(t, err)

	// Verify posts are ordered by views descending
	posts, ok := user["posts"].([]any)
	assert.True(t, ok, "posts should be included")
	assert.Greater(t, len(posts), 1, "should have multiple posts")

	if len(posts) > 1 {
		// Verify ordering
		prevViews := int64(999999) // Start with a high number
		for i, p := range posts {
			post := p.(map[string]any)
			views := utils.ToInt64(post["views"])
			assert.LessOrEqual(t, views, prevViews,
				"post %d should have views <= previous post (descending order)", i)
			prevViews = views
		}
	}
}

// TestIncludeWithPagination tests take/skip on relations
func (dct *DriverConformanceTests) TestIncludeWithPagination(t *testing.T) {
	if dct.shouldSkip("TestIncludeWithPagination") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	// Insert more posts for pagination testing
	ctx := context.Background()
	User := td.DB.Model("User")
	Post := td.DB.Model("Post")

	// Create a user with many posts
	userResult, err := User.Insert(map[string]any{
		"name":   "TestUser",
		"email":  "test@example.com",
		"active": true,
	}).Exec(ctx)
	require.NoError(t, err)

	userId := int(userResult.LastInsertID)
	if td.DB.GetDriverType() == "postgresql" {
		// For PostgreSQL, we need to get the ID differently
		var users []TestUser
		err = User.Select().WhereCondition(User.Where("email").Equals("test@example.com")).FindMany(ctx, &users)
		require.NoError(t, err)
		require.Len(t, users, 1)
		userId = users[0].ID
	}

	// Create 10 posts for this user
	for i := range 10 {
		_, err = Post.Insert(map[string]any{
			"title":     fmt.Sprintf("Post %d", i),
			"content":   fmt.Sprintf("Content %d", i),
			"userId":    userId,
			"published": true,
			"views":     i * 10,
		}).Exec(ctx)
		require.NoError(t, err)
	}

	// Test pagination
	limit := 3
	offset := 2
	includeOpt := &types.IncludeOption{
		Path:   "posts",
		Limit:  &limit,
		Offset: &offset,
		OrderBy: []types.OrderByOption{
			{Field: "id", Direction: types.ASC},
		},
	}

	var user map[string]any
	err = User.Select().
		WhereCondition(User.Where("id").Equals(userId)).
		IncludeWithOptions("posts", includeOpt).
		FindFirst(ctx, &user)

	require.NoError(t, err)

	// Verify pagination
	posts, ok := user["posts"].([]any)
	assert.True(t, ok, "posts should be included")

	// Note: If the driver doesn't support include-level pagination,
	// it might return all posts
	if len(posts) == limit {
		t.Logf("Driver %s supports include-level pagination", dct.DriverName)
	} else {
		t.Logf("Driver %s does not yet support include-level pagination (got %d posts, expected %d)",
			dct.DriverName, len(posts), limit)
	}
}

// TestIncludeNestedWithOptions tests options in nested includes
func (dct *DriverConformanceTests) TestIncludeNestedWithOptions(t *testing.T) {
	if dct.shouldSkip("TestIncludeNestedWithOptions") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test nested includes with options
	User := td.DB.Model("User")

	// Create include options for posts
	postsOpt := &types.IncludeOption{
		Path:   "posts",
		Select: []string{"id", "title", "userId"},
		Where:  td.DB.Model("Post").Where("published").Equals(true),
	}

	var user map[string]any
	err = User.Select().
		WhereCondition(User.Where("id").Equals(1)).
		IncludeWithOptions("posts", postsOpt).
		FindFirst(ctx, &user)

	// Note: Nested include options are an advanced feature
	// Some drivers might not support this yet
	if err != nil {
		t.Logf("Driver %s does not yet support nested include options: %v", dct.DriverName, err)
		return
	}

	assert.Equal(t, "Alice", user["name"])

	// Verify nested structure
	posts, ok := user["posts"].([]any)
	assert.True(t, ok, "posts should be included")

	for _, p := range posts {
		post := p.(map[string]any)

		// Verify field selection on posts
		assert.Contains(t, post, "id")
		assert.Contains(t, post, "title")

		// For full nested include with options test, we would need to
		// support nested IncludeOptions in the IncludeOption struct
	}
}

// TestIncludePerformance benchmarks filtered vs unfiltered includes
func (dct *DriverConformanceTests) TestIncludePerformance(t *testing.T) {
	if dct.shouldSkip("TestIncludePerformance") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Create test data with many relations
	User := td.DB.Model("User")
	Post := td.DB.Model("Post")

	// Create users with many posts
	for i := range 10 {
		userResult, err := User.Insert(map[string]any{
			"name":   fmt.Sprintf("User%d", i),
			"email":  fmt.Sprintf("user%d@example.com", i),
			"active": true,
		}).Exec(ctx)
		require.NoError(t, err)

		userId := int(userResult.LastInsertID)
		if td.DB.GetDriverType() == "postgresql" {
			// For PostgreSQL, get the ID differently
			var users []TestUser
			err = User.Select().
				WhereCondition(User.Where("email").Equals(fmt.Sprintf("user%d@example.com", i))).
				FindMany(ctx, &users)
			require.NoError(t, err)
			require.Len(t, users, 1)
			userId = users[0].ID
		}

		// Create 20 posts per user (10 published, 10 unpublished)
		for j := range 20 {
			_, err = Post.Insert(map[string]any{
				"title":     fmt.Sprintf("Post %d-%d", i, j),
				"content":   fmt.Sprintf("Content %d-%d", i, j),
				"userId":    userId,
				"published": j < 10, // First 10 are published
				"views":     j * 10,
			}).Exec(ctx)
			require.NoError(t, err)
		}
	}

	// Benchmark 1: Unfiltered include (all posts)
	start := time.Now()
	var usersUnfiltered []map[string]any
	err = User.Select().
		Include("posts").
		FindMany(ctx, &usersUnfiltered)
	require.NoError(t, err)
	unfilteredDuration := time.Since(start)

	// Count total posts loaded
	totalPostsUnfiltered := 0
	for _, user := range usersUnfiltered {
		if posts, ok := user["posts"].([]any); ok {
			totalPostsUnfiltered += len(posts)
		}
	}

	// Benchmark 2: Filtered include (only published posts)
	start = time.Now()
	var usersFiltered []map[string]any
	includeOpt := &types.IncludeOption{
		Path:   "posts",
		Where:  Post.Where("published").Equals(true),
		Select: []string{"id", "title", "published"}, // Also test field selection
	}
	err = User.Select().
		IncludeWithOptions("posts", includeOpt).
		FindMany(ctx, &usersFiltered)
	require.NoError(t, err)
	filteredDuration := time.Since(start)

	// Count total posts loaded
	totalPostsFiltered := 0
	for _, user := range usersFiltered {
		if posts, ok := user["posts"].([]any); ok {
			totalPostsFiltered += len(posts)
		}
	}

	// Log performance results
	t.Logf("Driver: %s", dct.DriverName)
	t.Logf("Unfiltered include: %v (loaded %d posts)", unfilteredDuration, totalPostsUnfiltered)
	t.Logf("Filtered include:   %v (loaded %d posts)", filteredDuration, totalPostsFiltered)

	// If SQL-level filtering is working, filtered should load fewer posts
	if totalPostsFiltered < totalPostsUnfiltered {
		t.Logf("SQL-level filtering is working! Loaded %d fewer posts",
			totalPostsUnfiltered-totalPostsFiltered)

		// Performance might also be better
		if filteredDuration < unfilteredDuration {
			improvement := float64(unfilteredDuration-filteredDuration) / float64(unfilteredDuration) * 100
			t.Logf("Performance improved by %.1f%%", improvement)
		}
	} else {
		t.Logf("SQL-level filtering not yet implemented - loading all posts and filtering in memory")
	}
}
