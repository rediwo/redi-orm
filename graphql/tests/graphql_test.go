package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/graphql"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import SQLite driver for tests
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
)

func TestGraphQLHandler(t *testing.T) {
	// Setup test database
	db, err := database.NewFromURI("sqlite://:memory:")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Load test schema
	schemaContent := `
		model User {
			id    Int     @id @default(autoincrement())
			name  String
			email String  @unique
			posts Post[]
		}

		model Post {
			id        Int      @id @default(autoincrement())
			title     String
			content   String?
			published Boolean  @default(false)
			authorId  Int
			author    User     @relation(fields: [authorId], references: [id])
		}
	`

	schemas, err := prisma.ParseSchema(schemaContent)
	require.NoError(t, err)

	// Register schemas
	for modelName, schema := range schemas {
		err = db.RegisterSchema(modelName, schema)
		require.NoError(t, err)
	}

	// Sync schemas
	err = db.SyncSchemas(ctx)
	require.NoError(t, err)

	// Generate GraphQL schema
	generator := graphql.NewSchemaGenerator(db, schemas)
	graphqlSchema, err := generator.Generate()
	require.NoError(t, err)

	// Create handler
	handler := graphql.NewHandler(graphqlSchema)

	// Test helper function
	testGraphQL := func(query string, variables map[string]any) map[string]any {
		body := map[string]any{
			"query":     query,
			"variables": variables,
		}
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeGraphQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		return response
	}

	t.Run("CreateUser", func(t *testing.T) {
		query := `
			mutation CreateUser($data: UserCreateInput!) {
				createUser(data: $data) {
					id
					name
					email
				}
			}
		`
		variables := map[string]any{
			"data": map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		}

		response := testGraphQL(query, variables)
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]any)
		user := data["createUser"].(map[string]any)

		assert.NotNil(t, user["id"])
		assert.Equal(t, "John Doe", user["name"])
		assert.Equal(t, "john@example.com", user["email"])
	})

	t.Run("FindManyUsers", func(t *testing.T) {
		// First create another user
		createQuery := `
			mutation {
				createUser(data: {name: "Jane Smith", email: "jane@example.com"}) {
					id
				}
			}
		`
		testGraphQL(createQuery, nil)

		// Now query users
		query := `
			query {
				findManyUser(orderBy: {name: ASC}) {
					id
					name
					email
				}
			}
		`

		response := testGraphQL(query, nil)
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]any)
		users := data["findManyUser"].([]any)

		assert.Len(t, users, 2)

		user1 := users[0].(map[string]any)
		assert.Equal(t, "Jane Smith", user1["name"])

		user2 := users[1].(map[string]any)
		assert.Equal(t, "John Doe", user2["name"])
	})

	t.Run("FindUniqueUser", func(t *testing.T) {
		query := `
			query FindUser($where: UserWhereInput!) {
				findUniqueUser(where: $where) {
					id
					name
					email
				}
			}
		`
		variables := map[string]any{
			"where": map[string]any{
				"email": map[string]any{
					"equals": "john@example.com",
				},
			},
		}

		response := testGraphQL(query, variables)
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]any)
		user := data["findUniqueUser"].(map[string]any)

		assert.Equal(t, "John Doe", user["name"])
		assert.Equal(t, "john@example.com", user["email"])
	})

	t.Run("CreatePost", func(t *testing.T) {
		// First create a user for this test
		createUserQuery := `
			mutation {
				createUser(data: {name: "Post Author", email: "postauthor@example.com"}) {
					id
				}
			}
		`
		response := testGraphQL(createUserQuery, nil)
		require.NotNil(t, response["data"], "createUser query failed")
		if response["errors"] != nil {
			t.Fatalf("GraphQL errors when creating user: %v", response["errors"])
		}

		data := response["data"].(map[string]any)
		user := data["createUser"].(map[string]any)
		userID := user["id"]

		// Create post
		query := `
			mutation CreatePost($data: PostCreateInput!) {
				createPost(data: $data) {
					id
					title
				}
			}
		`
		t.Logf("Creating post with userID: %v (type: %T)", userID, userID)
		variables := map[string]any{
			"data": map[string]any{
				"title":     "My First Post",
				"content":   "Hello, World!",
				"published": true,
				"authorId":  userID,
			},
		}

		response = testGraphQL(query, variables)
		require.NotNil(t, response["data"])

		// Debug: print the response to see what's happening
		if response["errors"] != nil {
			t.Logf("GraphQL errors: %v", response["errors"])
		}

		data = response["data"].(map[string]any)
		require.NotNil(t, data["createPost"], "createPost returned nil")
		post := data["createPost"].(map[string]any)

		assert.NotNil(t, post["id"])
		assert.Equal(t, "My First Post", post["title"])
	})

	t.Run("UpdateUser", func(t *testing.T) {
		query := `
			mutation UpdateUser($where: UserWhereInput!, $data: UserUpdateInput!) {
				updateUser(where: $where, data: $data) {
					id
					name
					email
				}
			}
		`
		variables := map[string]any{
			"where": map[string]any{
				"email": map[string]any{
					"equals": "john@example.com",
				},
			},
			"data": map[string]any{
				"name": "John Updated",
			},
		}

		response := testGraphQL(query, variables)
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]any)
		user := data["updateUser"].(map[string]any)

		assert.Equal(t, "John Updated", user["name"])
		assert.Equal(t, "john@example.com", user["email"])
	})

	t.Run("DeleteUser", func(t *testing.T) {
		// Create a user to delete
		createQuery := `
			mutation {
				createUser(data: {name: "Delete Me", email: "delete@example.com"}) {
					id
				}
			}
		`
		testGraphQL(createQuery, nil)

		// Delete the user
		query := `
			mutation DeleteUser($where: UserWhereInput!) {
				deleteUser(where: $where) {
					id
					name
					email
				}
			}
		`
		variables := map[string]any{
			"where": map[string]any{
				"email": map[string]any{
					"equals": "delete@example.com",
				},
			},
		}

		response := testGraphQL(query, variables)
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]any)
		user := data["deleteUser"].(map[string]any)

		assert.Equal(t, "Delete Me", user["name"])

		// Verify user is deleted
		findQuery := `
			query {
				findUniqueUser(where: {email: {equals: "delete@example.com"}}) {
					id
				}
			}
		`
		response = testGraphQL(findQuery, nil)
		data = response["data"].(map[string]any)
		assert.Nil(t, data["findUniqueUser"])
	})

	t.Run("CountUsers", func(t *testing.T) {
		query := `
			query {
				countUser
			}
		`

		response := testGraphQL(query, nil)
		require.NotNil(t, response["data"])

		data := response["data"].(map[string]any)
		count := data["countUser"]

		// We should have at least 2 users (John and Jane)
		// GraphQL returns numbers as float64
		assert.GreaterOrEqual(t, count, float64(2))
	})

	t.Run("WhereConditions", func(t *testing.T) {
		// Test various where conditions
		query := `
			query FindUsersByName($where: UserWhereInput!) {
				findManyUser(where: $where) {
					name
					email
				}
			}
		`

		// Test contains
		variables := map[string]any{
			"where": map[string]any{
				"name": map[string]any{
					"contains": "Smith",
				},
			},
		}

		response := testGraphQL(query, variables)
		data := response["data"].(map[string]any)
		users := data["findManyUser"].([]any)
		assert.Len(t, users, 1)
		user := users[0].(map[string]any)
		assert.Equal(t, "Jane Smith", user["name"])

		// Test startsWith
		variables = map[string]any{
			"where": map[string]any{
				"email": map[string]any{
					"startsWith": "jane",
				},
			},
		}

		response = testGraphQL(query, variables)
		data = response["data"].(map[string]any)
		users = data["findManyUser"].([]any)
		assert.Len(t, users, 1)
		user = users[0].(map[string]any)
		assert.Equal(t, "jane@example.com", user["email"])
	})

	t.Run("Pagination", func(t *testing.T) {
		query := `
			query {
				findManyUser(limit: 1, offset: 1, orderBy: {id: ASC}) {
					name
				}
			}
		`

		response := testGraphQL(query, nil)
		data := response["data"].(map[string]any)
		users := data["findManyUser"].([]any)

		assert.Len(t, users, 1)
		// This should be the second user when ordered by ID
	})

	t.Run("BatchOperations", func(t *testing.T) {
		// Test createMany
		createManyQuery := `
			mutation CreateManyUsers($data: [UserCreateInput!]!) {
				createManyUser(data: $data) {
					count
				}
			}
		`
		variables := map[string]any{
			"data": []any{
				map[string]any{
					"name":  "Batch User 1",
					"email": "batch1@example.com",
				},
				map[string]any{
					"name":  "Batch User 2",
					"email": "batch2@example.com",
				},
			},
		}

		response := testGraphQL(createManyQuery, variables)
		data := response["data"].(map[string]any)
		result := data["createManyUser"].(map[string]any)
		assert.Equal(t, float64(2), result["count"])

		// Test deleteMany
		deleteManyQuery := `
			mutation {
				deleteManyUser(where: {name: {startsWith: "Batch"}}) {
					count
				}
			}
		`

		response = testGraphQL(deleteManyQuery, nil)
		data = response["data"].(map[string]any)
		result = data["deleteManyUser"].(map[string]any)
		assert.Equal(t, float64(2), result["count"])
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with invalid query
		query := `
			query {
				invalidField
			}
		`

		response := testGraphQL(query, nil)
		assert.NotNil(t, response["errors"])
		errors := response["errors"].([]any)
		assert.Greater(t, len(errors), 0)
	})

	t.Run("HTTPMethods", func(t *testing.T) {
		// Test GET request
		req := httptest.NewRequest("GET", "/graphql?query={findManyUser{id,name}}", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test invalid method
		req = httptest.NewRequest("PUT", "/graphql", nil)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("ContentTypes", func(t *testing.T) {
		// Test application/graphql content type
		query := "{ findManyUser { id name } }"
		req := httptest.NewRequest("POST", "/graphql", bytes.NewReader([]byte(query)))
		req.Header.Set("Content-Type", "application/graphql")

		w := httptest.NewRecorder()
		handler.ServeGraphQL(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	})

	t.Run("RelationFields", func(t *testing.T) {
		// First create a user for this test
		createUserQuery := `
			mutation {
				createUser(data: {name: "Relation Author", email: "relation@example.com"}) {
					id
					name
				}
			}
		`
		userResponse := testGraphQL(createUserQuery, nil)
		require.NotNil(t, userResponse["data"])

		userData := userResponse["data"].(map[string]any)
		user := userData["createUser"].(map[string]any)
		userID := user["id"]

		// Create a post for this user
		createPostQuery := `
			mutation CreatePost($data: PostCreateInput!) {
				createPost(data: $data) {
					id
					title
					authorId
				}
			}
		`
		postVariables := map[string]any{
			"data": map[string]any{
				"title":     "Relation Test Post",
				"content":   "Testing relations",
				"authorId":  int(userID.(float64)),
				"published": true,
			},
		}

		postResponse := testGraphQL(createPostQuery, postVariables)
		require.NotNil(t, postResponse["data"])

		postData := postResponse["data"].(map[string]any)
		post := postData["createPost"].(map[string]any)
		postID := post["id"]

		// Test Post -> Author relation
		postWithAuthorQuery := `
			query GetPostWithAuthor($where: PostWhereInput!) {
				findUniquePost(where: $where) {
					id
					title
					authorId
					author {
						id
						name
						email
					}
				}
			}
		`
		postQueryVars := map[string]any{
			"where": map[string]any{
				"id": map[string]any{
					"equals": postID,
				},
			},
		}

		relationResponse := testGraphQL(postWithAuthorQuery, postQueryVars)
		require.NotNil(t, relationResponse["data"])

		relationData := relationResponse["data"].(map[string]any)
		postWithAuthor := relationData["findUniquePost"].(map[string]any)

		// Verify the author relation is loaded
		assert.NotNil(t, postWithAuthor["author"])
		author := postWithAuthor["author"].(map[string]any)
		assert.Equal(t, "Relation Author", author["name"])
		assert.Equal(t, "relation@example.com", author["email"])

		// Test User -> Posts relation
		userWithPostsQuery := `
			query GetUserWithPosts($where: UserWhereInput!) {
				findUniqueUser(where: $where) {
					id
					name
					email
					posts {
						id
						title
						published
					}
				}
			}
		`
		userQueryVars := map[string]any{
			"where": map[string]any{
				"id": map[string]any{
					"equals": userID,
				},
			},
		}

		userRelationResponse := testGraphQL(userWithPostsQuery, userQueryVars)
		require.NotNil(t, userRelationResponse["data"])

		userRelationData := userRelationResponse["data"].(map[string]any)
		userWithPosts := userRelationData["findUniqueUser"].(map[string]any)

		// Verify the posts relation is loaded
		assert.NotNil(t, userWithPosts["posts"])
		posts := userWithPosts["posts"].([]any)
		assert.Len(t, posts, 1)

		relatedPost := posts[0].(map[string]any)
		assert.Equal(t, "Relation Test Post", relatedPost["title"])
		assert.Equal(t, true, relatedPost["published"])
	})
}

func TestPlayground(t *testing.T) {
	// Create a minimal schema for testing
	db, _ := database.NewFromURI("sqlite://:memory:")
	db.Connect(context.Background())
	defer db.Close()

	schemas := map[string]*schema.Schema{}
	generator := graphql.NewSchemaGenerator(db, schemas)
	graphqlSchema, _ := generator.Generate()

	handler := graphql.NewHandler(graphqlSchema).EnablePlayground()

	// Test playground HTML response
	req := httptest.NewRequest("GET", "/graphql", nil)
	req.Header.Set("Accept", "text/html")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Body.String(), "GraphQL Playground")
}
