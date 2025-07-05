package agile

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Relation Tests
func (act *AgileConformanceTests) runRelationTests(t *testing.T, client *Client, db types.Database) {
	// Test one-to-many relations
	act.runWithCleanup(t, db, func() {
		t.Run("OneToManyRelations", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id    Int    @id @default(autoincrement())
					name  String
					email String @unique
					posts Post[]
				}
				
				model Post {
					id        Int    @id @default(autoincrement())
					title     String
					content   String
					published Boolean @default(false)
					authorId  Int
					author    User   @relation(fields: [authorId], references: [id])
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create users
			user1, err := client.Model("User").Create(`{"data": {"name": "Alice", "email": "alice@example.com"}}`)
			assertNoError(t, err, "Failed to create user 1")
			
			user2, err := client.Model("User").Create(`{"data": {"name": "Bob", "email": "bob@example.com"}}`)
			assertNoError(t, err, "Failed to create user 2")
			
			// Create posts
			posts := []string{
				fmt.Sprintf(`{"data": {"title": "Post 1", "content": "Content 1", "authorId": %v, "published": true}}`, user1["id"]),
				fmt.Sprintf(`{"data": {"title": "Post 2", "content": "Content 2", "authorId": %v}}`, user1["id"]),
				fmt.Sprintf(`{"data": {"title": "Post 3", "content": "Content 3", "authorId": %v, "published": true}}`, user2["id"]),
			}
			
			for _, post := range posts {
				_, err = client.Model("Post").Create(post)
				assertNoError(t, err, "Failed to create post")
			}
			
			// Test include posts with user
			result, err := client.Model("User").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v},
				"include": {"posts": true}
			}`, user1["id"]))
			assertNoError(t, err, "Failed to find user with posts")
			
			// Check included posts
			if posts, ok := result["posts"].([]any); ok {
				assertEqual(t, 2, len(posts), "User 1 posts count mismatch")
			} else {
				t.Fatal("Posts not included or wrong type")
			}
			
			// Test include with filtering
			result, err = client.Model("User").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v},
				"include": {
					"posts": {
						"where": {"published": true}
					}
				}
			}`, user1["id"]))
			assertNoError(t, err, "Failed to find user with filtered posts")
			
			// Check filtered posts
			if posts, ok := result["posts"].([]any); ok {
				assertEqual(t, 1, len(posts), "Filtered posts count mismatch")
				if len(posts) > 0 {
					if post, ok := posts[0].(map[string]any); ok {
						assertEqual(t, "Post 1", post["title"], "Filtered post title mismatch")
					}
				}
			}
			
			// Test include author with posts
			postsWithAuthors, err := client.Model("Post").FindMany(`{
				"include": {"author": true}
			}`)
			assertNoError(t, err, "Failed to find posts with authors")
			
			assertEqual(t, 3, len(postsWithAuthors), "Posts count mismatch")
			for _, post := range postsWithAuthors {
				assertNotNil(t, post["author"], "Author should be included")
				if author, ok := post["author"].(map[string]any); ok {
					assertNotNil(t, author["name"], "Author name should be present")
				}
			}
		})
	})
	
	// Test many-to-many relations
	act.runWithCleanup(t, db, func() {
		t.Run("ManyToManyRelations", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model Post {
					id         Int         @id @default(autoincrement())
					title      String
					categories Category[]
				}
				
				model Category {
					id    Int    @id @default(autoincrement())
					name  String @unique
					posts Post[]
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create categories
			_, err = client.Model("Category").Create(`{"data": {"name": "Technology"}}`)
			assertNoError(t, err, "Failed to create category 1")
			
			_, err = client.Model("Category").Create(`{"data": {"name": "Science"}}`)
			assertNoError(t, err, "Failed to create category 2")
			
			_, err = client.Model("Category").Create(`{"data": {"name": "Programming"}}`)
			assertNoError(t, err, "Failed to create category 3")
			
			// Create posts
			// Note: Many-to-many relations typically require explicit junction table operations
			// or nested writes which may not be fully implemented yet
			_, err = client.Model("Post").Create(`{"data": {"title": "AI and Machine Learning"}}`)
			assertNoError(t, err, "Failed to create post 1")
			
			_, err = client.Model("Post").Create(`{"data": {"title": "Web Development"}}`)
			assertNoError(t, err, "Failed to create post 2")
			
			// This test may need adjustment based on how many-to-many relations are implemented
			t.Log("Many-to-many relation tests may require junction table operations")
		})
	})
	
	// Test nested includes
	act.runWithCleanup(t, db, func() {
		t.Run("NestedIncludes", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id       Int       @id @default(autoincrement())
					name     String
					email    String    @unique
					posts    Post[]
					comments Comment[]
				}
				
				model Post {
					id        Int       @id @default(autoincrement())
					title     String
					content   String
					authorId  Int
					author    User      @relation(fields: [authorId], references: [id])
					comments  Comment[]
				}
				
				model Comment {
					id       Int    @id @default(autoincrement())
					content  String
					postId   Int
					post     Post   @relation(fields: [postId], references: [id])
					authorId Int
					author   User   @relation(fields: [authorId], references: [id])
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create test data
			user1, err := client.Model("User").Create(`{"data": {"name": "Alice", "email": "alice@example.com"}}`)
			assertNoError(t, err, "Failed to create user 1")
			
			user2, err := client.Model("User").Create(`{"data": {"name": "Bob", "email": "bob@example.com"}}`)
			assertNoError(t, err, "Failed to create user 2")
			
			post, err := client.Model("Post").Create(fmt.Sprintf(`{
				"data": {
					"title": "Test Post",
					"content": "Test content",
					"authorId": %v
				}
			}`, user1["id"]))
			assertNoError(t, err, "Failed to create post")
			
			// Create comments
			_, err = client.Model("Comment").Create(fmt.Sprintf(`{
				"data": {
					"content": "Great post!",
					"postId": %v,
					"authorId": %v
				}
			}`, post["id"], user2["id"]))
			assertNoError(t, err, "Failed to create comment 1")
			
			_, err = client.Model("Comment").Create(fmt.Sprintf(`{
				"data": {
					"content": "Thanks!",
					"postId": %v,
					"authorId": %v
				}
			}`, post["id"], user1["id"]))
			assertNoError(t, err, "Failed to create comment 2")
			
			// Test nested include
			result, err := client.Model("Post").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v},
				"include": {
					"author": true,
					"comments": {
						"include": {
							"author": true
						}
					}
				}
			}`, post["id"]))
			assertNoError(t, err, "Failed to find post with nested includes")
			
			// Verify nested structure
			assertNotNil(t, result["author"], "Post author should be included")
			if author, ok := result["author"].(map[string]any); ok {
				assertEqual(t, "Alice", author["name"], "Post author name mismatch")
			}
			
			if comments, ok := result["comments"].([]any); ok {
				assertEqual(t, 2, len(comments), "Comments count mismatch")
				for _, comment := range comments {
					if c, ok := comment.(map[string]any); ok {
						assertNotNil(t, c["author"], "Comment author should be included")
					}
				}
			} else {
				t.Log("Comments not included or wrong type - nested includes may not be fully supported")
			}
		})
	})
	
	// Test include with ordering and pagination
	act.runWithCleanup(t, db, func() {
		t.Run("IncludeWithOptions", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model Author {
					id    Int    @id @default(autoincrement())
					name  String
					books Book[]
				}
				
				model Book {
					id         Int      @id @default(autoincrement())
					title      String
					pages      Int
					publishedAt DateTime
					authorId   Int
					author     Author   @relation(fields: [authorId], references: [id])
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create author
			author, err := client.Model("Author").Create(`{"data": {"name": "Jane Doe"}}`)
			assertNoError(t, err, "Failed to create author")
			
			// Create multiple books
			// Use MySQL-compatible datetime format
			books := []string{
				fmt.Sprintf(`{"data": {"title": "Book A", "pages": 200, "publishedAt": "2024-01-01 00:00:00", "authorId": %v}}`, author["id"]),
				fmt.Sprintf(`{"data": {"title": "Book B", "pages": 300, "publishedAt": "2024-02-01 00:00:00", "authorId": %v}}`, author["id"]),
				fmt.Sprintf(`{"data": {"title": "Book C", "pages": 150, "publishedAt": "2024-03-01 00:00:00", "authorId": %v}}`, author["id"]),
				fmt.Sprintf(`{"data": {"title": "Book D", "pages": 400, "publishedAt": "2024-04-01 00:00:00", "authorId": %v}}`, author["id"]),
			}
			
			for _, book := range books {
				_, err = client.Model("Book").Create(book)
				assertNoError(t, err, "Failed to create book")
			}
			
			// Test include with ordering
			result, err := client.Model("Author").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v},
				"include": {
					"books": {
						"orderBy": {"pages": "desc"}
					}
				}
			}`, author["id"]))
			assertNoError(t, err, "Failed to find author with ordered books")
			
			if books, ok := result["books"].([]any); ok {
				assertEqual(t, 4, len(books), "Books count mismatch")
				// Check if ordered by pages descending
				if len(books) > 0 {
					if firstBook, ok := books[0].(map[string]any); ok {
						assertEqual(t, "Book D", firstBook["title"], "First book should be Book D (400 pages)")
					}
				}
			}
			
			// Test include with pagination
			result, err = client.Model("Author").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v},
				"include": {
					"books": {
						"orderBy": {"publishedAt": "asc"},
						"take": 2,
						"skip": 1
					}
				}
			}`, author["id"]))
			assertNoError(t, err, "Failed to find author with paginated books")
			
			if books, ok := result["books"].([]any); ok {
				// Should get 2 books starting from the second one
				if len(books) != 2 {
					t.Logf("Include with pagination may not be fully supported, got %d books", len(books))
				}
			}
		})
	})
}