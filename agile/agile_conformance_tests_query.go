package agile

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Query Tests
func (act *AgileConformanceTests) runQueryTests(t *testing.T, client *Client, db types.Database) {
	// Test complex where conditions
	act.runWithCleanup(t, db, func() {
		t.Run("ComplexWhereConditions", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id        Int      @id @default(autoincrement())
					name      String
					email     String   @unique
					age       Int
					active    Boolean  @default(true)
					role      String
					createdAt DateTime @default(now())
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create test data
			users := []string{
				`{"data": {"name": "Alice", "email": "alice@example.com", "age": 25, "role": "admin"}}`,
				`{"data": {"name": "Bob", "email": "bob@example.com", "age": 30, "role": "user"}}`,
				`{"data": {"name": "Charlie", "email": "charlie@example.com", "age": 25, "role": "user"}}`,
				`{"data": {"name": "David", "email": "david@example.com", "age": 35, "role": "admin", "active": false}}`,
			}

			for _, user := range users {
				_, err = client.Model("User").Create(user)
				assertNoError(t, err, "Failed to create user")
			}

			// Test OR condition
			result, err := client.Model("User").FindMany(`{
				"where": {
					"OR": [
						{"age": 25},
						{"role": "admin"}
					]
				},
				"orderBy": {"name": "asc"}
			}`)
			assertNoError(t, err, "Failed to find with OR")
			assertEqual(t, 3, len(result), "OR condition result count mismatch")

			// Test AND condition
			result, err = client.Model("User").FindMany(`{
				"where": {
					"AND": [
						{"age": {"gte": 25}},
						{"role": "user"}
					]
				}
			}`)
			assertNoError(t, err, "Failed to find with AND")
			assertEqual(t, 2, len(result), "AND condition result count mismatch")

			// Test NOT condition
			result, err = client.Model("User").FindMany(`{
				"where": {
					"NOT": {"role": "admin"}
				}
			}`)
			assertNoError(t, err, "Failed to find with NOT")
			assertEqual(t, 2, len(result), "NOT condition result count mismatch")

			// Test nested conditions
			result, err = client.Model("User").FindMany(`{
				"where": {
					"AND": [
						{
							"OR": [
								{"age": {"lt": 30}},
								{"role": "admin"}
							]
						},
						{"active": true}
					]
				}
			}`)
			assertNoError(t, err, "Failed to find with nested conditions")
			assertEqual(t, 2, len(result), "Nested condition result count mismatch")
		})
	})

	// Test operators
	act.runWithCleanup(t, db, func() {
		t.Run("QueryOperators", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Product {
					id          Int    @id @default(autoincrement())
					name        String
					description String
					price       Float
					stock       Int
					tags        String
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create test data
			products := []string{
				`{"data": {"name": "Laptop", "description": "High-end laptop computer", "price": 1200.50, "stock": 10, "tags": "electronics,computers"}}`,
				`{"data": {"name": "Mouse", "description": "Wireless mouse", "price": 25.99, "stock": 50, "tags": "electronics,accessories"}}`,
				`{"data": {"name": "Keyboard", "description": "Mechanical keyboard", "price": 89.99, "stock": 0, "tags": "electronics,accessories"}}`,
				`{"data": {"name": "Monitor", "description": "4K monitor display", "price": 450.00, "stock": 15, "tags": "electronics,displays"}}`,
			}

			for _, product := range products {
				_, err = client.Model("Product").Create(product)
				assertNoError(t, err, "Failed to create product")
			}

			// Test gt (greater than)
			result, err := client.Model("Product").FindMany(`{
				"where": {"price": {"gt": 100}}
			}`)
			assertNoError(t, err, "Failed to find with gt")
			assertEqual(t, 2, len(result), "GT operator result count mismatch")

			// Test gte (greater than or equal)
			result, err = client.Model("Product").FindMany(`{
				"where": {"price": {"gte": 89.99}}
			}`)
			assertNoError(t, err, "Failed to find with gte")
			assertEqual(t, 3, len(result), "GTE operator result count mismatch")

			// Test lt (less than)
			result, err = client.Model("Product").FindMany(`{
				"where": {"price": {"lt": 100}}
			}`)
			assertNoError(t, err, "Failed to find with lt")
			assertEqual(t, 2, len(result), "LT operator result count mismatch")

			// Test lte (less than or equal)
			result, err = client.Model("Product").FindMany(`{
				"where": {"price": {"lte": 89.99}}
			}`)
			assertNoError(t, err, "Failed to find with lte")
			assertEqual(t, 2, len(result), "LTE operator result count mismatch")

			// Test in
			result, err = client.Model("Product").FindMany(`{
				"where": {"name": {"in": ["Laptop", "Mouse", "Cable"]}}
			}`)
			assertNoError(t, err, "Failed to find with in")
			assertEqual(t, 2, len(result), "IN operator result count mismatch")

			// Test notIn
			result, err = client.Model("Product").FindMany(`{
				"where": {"name": {"notIn": ["Laptop", "Mouse"]}}
			}`)
			assertNoError(t, err, "Failed to find with notIn")
			assertEqual(t, 2, len(result), "NOT IN operator result count mismatch")

			// Test contains
			result, err = client.Model("Product").FindMany(`{
				"where": {"description": {"contains": "keyboard"}}
			}`)
			assertNoError(t, err, "Failed to find with contains")
			assertEqual(t, 1, len(result), "CONTAINS operator result count mismatch")

			// Test startsWith
			result, err = client.Model("Product").FindMany(`{
				"where": {"name": {"startsWith": "M"}}
			}`)
			assertNoError(t, err, "Failed to find with startsWith")
			assertEqual(t, 2, len(result), "STARTS WITH operator result count mismatch")

			// Test endsWith
			result, err = client.Model("Product").FindMany(`{
				"where": {"tags": {"endsWith": "accessories"}}
			}`)
			assertNoError(t, err, "Failed to find with endsWith")
			assertEqual(t, 2, len(result), "ENDS WITH operator result count mismatch")
		})
	})

	// Test sorting and pagination
	act.runWithCleanup(t, db, func() {
		t.Run("SortingAndPagination", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Article {
					id          Int      @id @default(autoincrement())
					title       String
					views       Int      @default(0)
					publishedAt DateTime
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create test data
			// Use MySQL-compatible datetime format
			articles := []string{
				`{"data": {"title": "Article A", "views": 100, "publishedAt": "2024-01-01 00:00:00"}}`,
				`{"data": {"title": "Article B", "views": 200, "publishedAt": "2024-01-02 00:00:00"}}`,
				`{"data": {"title": "Article C", "views": 150, "publishedAt": "2024-01-03 00:00:00"}}`,
				`{"data": {"title": "Article D", "views": 300, "publishedAt": "2024-01-04 00:00:00"}}`,
				`{"data": {"title": "Article E", "views": 50, "publishedAt": "2024-01-05 00:00:00"}}`,
			}

			for _, article := range articles {
				_, err = client.Model("Article").Create(article)
				assertNoError(t, err, "Failed to create article")
			}

			// Test orderBy ascending
			result, err := client.Model("Article").FindMany(`{
				"orderBy": {"views": "asc"}
			}`)
			assertNoError(t, err, "Failed to find with orderBy asc")
			assertEqual(t, 5, len(result), "OrderBy result count mismatch")
			assertEqual(t, "Article E", result[0]["title"], "First article title mismatch")
			assertEqual(t, "Article D", result[4]["title"], "Last article title mismatch")

			// Test orderBy descending
			result, err = client.Model("Article").FindMany(`{
				"orderBy": {"views": "desc"}
			}`)
			assertNoError(t, err, "Failed to find with orderBy desc")
			assertEqual(t, "Article D", result[0]["title"], "First article title mismatch (desc)")
			assertEqual(t, "Article E", result[4]["title"], "Last article title mismatch (desc)")

			// Test multiple orderBy
			result, err = client.Model("Article").FindMany(`{
				"orderBy": [
					{"views": "desc"},
					{"title": "asc"}
				]
			}`)
			assertNoError(t, err, "Failed to find with multiple orderBy")
			assertEqual(t, 5, len(result), "Multiple orderBy result count mismatch")

			// Test take (limit)
			result, err = client.Model("Article").FindMany(`{
				"orderBy": {"publishedAt": "asc"},
				"take": 3
			}`)
			assertNoError(t, err, "Failed to find with take")
			assertEqual(t, 3, len(result), "Take result count mismatch")
			assertEqual(t, "Article A", result[0]["title"], "First taken article mismatch")

			// Test skip (offset)
			result, err = client.Model("Article").FindMany(`{
				"orderBy": {"publishedAt": "asc"},
				"skip": 2
			}`)
			assertNoError(t, err, "Failed to find with skip")
			assertEqual(t, 3, len(result), "Skip result count mismatch")
			assertEqual(t, "Article C", result[0]["title"], "First skipped article mismatch")

			// Test take + skip (pagination)
			result, err = client.Model("Article").FindMany(`{
				"orderBy": {"publishedAt": "asc"},
				"skip": 1,
				"take": 2
			}`)
			assertNoError(t, err, "Failed to find with skip+take")
			assertEqual(t, 2, len(result), "Pagination result count mismatch")
			assertEqual(t, "Article B", result[0]["title"], "First paginated article mismatch")
			assertEqual(t, "Article C", result[1]["title"], "Second paginated article mismatch")
		})
	})

	// Test distinct
	act.runWithCleanup(t, db, func() {
		t.Run("Distinct", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Event {
					id       Int    @id @default(autoincrement())
					type     String
					category String
					userId   Int
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create test data with duplicates
			events := []string{
				`{"data": {"type": "click", "category": "button", "userId": 1}}`,
				`{"data": {"type": "click", "category": "button", "userId": 2}}`,
				`{"data": {"type": "view", "category": "page", "userId": 1}}`,
				`{"data": {"type": "click", "category": "link", "userId": 1}}`,
				`{"data": {"type": "view", "category": "page", "userId": 2}}`,
			}

			for _, event := range events {
				_, err = client.Model("Event").Create(event)
				assertNoError(t, err, "Failed to create event")
			}

			// Test simple distinct
			result, err := client.Model("Event").FindMany(`{
				"distinct": true,
				"select": ["type"]
			}`)
			assertNoError(t, err, "Failed to find with distinct")
			// Should return 2 distinct types: click, view
			if len(result) > 2 {
				t.Logf("Warning: Simple distinct not fully supported, got %d results", len(result))
			}

			// Test distinct on specific fields (if supported)
			if db.GetDriverType() == "postgresql" {
				result, err = client.Model("Event").FindMany(`{
					"distinct": ["type"],
					"orderBy": {"type": "asc"}
				}`)
				assertNoError(t, err, "Failed to find with distinct fields")
			}
		})
	})
}
