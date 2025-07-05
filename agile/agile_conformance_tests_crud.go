package agile

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// CRUD Tests
func (act *AgileConformanceTests) runCRUDTests(t *testing.T, client *Client, db types.Database) {
	// Test create
	act.runWithCleanup(t, db, func() {
		t.Run("Create", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id    Int    @id @default(autoincrement())
					name  String
					email String @unique
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create user
			user, err := client.Model("User").Create(`{
				"data": {
					"name": "John Doe",
					"email": "john@example.com"
				}
			}`)
			assertNoError(t, err, "Failed to create user")
			
			// Check results
			assertNotNil(t, user["id"], "User ID should not be nil")
			assertEqual(t, "John Doe", user["name"], "User name mismatch")
			assertEqual(t, "john@example.com", user["email"], "User email mismatch")
		})
	})

	// Test findMany
	act.runWithCleanup(t, db, func() {
		t.Run("FindMany", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id    Int    @id @default(autoincrement())
					name  String
					email String @unique
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create test data
			_, err = client.Model("User").Create(`{"data": {"name": "User 1", "email": "user1@example.com"}}`)
			assertNoError(t, err, "Failed to create user 1")
			
			_, err = client.Model("User").Create(`{"data": {"name": "User 2", "email": "user2@example.com"}}`)
			assertNoError(t, err, "Failed to create user 2")
			
			// Find many
			users, err := client.Model("User").FindMany(`{}`)
			assertNoError(t, err, "Failed to find users")
			
			if len(users) < 2 {
				t.Fatalf("Expected at least 2 users, got %d", len(users))
			}
		})
	})

	// Test update
	act.runWithCleanup(t, db, func() {
		t.Run("Update", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id    Int    @id @default(autoincrement())
					name  String
					email String @unique
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create user
			user, err := client.Model("User").Create(`{
				"data": {"name": "Original", "email": "original@example.com"}
			}`)
			assertNoError(t, err, "Failed to create user")
			
			// Update user
			updated, err := client.Model("User").Update(`{
				"where": {"id": ` + idToString(user["id"]) + `},
				"data": {"name": "Updated"}
			}`)
			assertNoError(t, err, "Failed to update user")
			
			assertEqual(t, user["id"], updated["id"], "User ID changed")
			assertEqual(t, "Updated", updated["name"], "User name not updated")
			assertEqual(t, "original@example.com", updated["email"], "User email changed")
		})
	})

	// Test delete
	act.runWithCleanup(t, db, func() {
		t.Run("Delete", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id    Int    @id @default(autoincrement())
					name  String
					email String @unique
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create user
			user, err := client.Model("User").Create(`{
				"data": {"name": "To Delete", "email": "delete@example.com"}
			}`)
			assertNoError(t, err, "Failed to create user")
			
			// Delete user
			deleted, err := client.Model("User").Delete(`{
				"where": {"id": ` + idToString(user["id"]) + `}
			}`)
			assertNoError(t, err, "Failed to delete user")
			
			assertEqual(t, user["id"], deleted["id"], "Deleted user ID mismatch")
			
			// Verify deletion
			_, err = client.Model("User").FindUnique(`{
				"where": {"id": ` + idToString(user["id"]) + `}
			}`)
			if err == nil {
				t.Fatal("Expected error when finding deleted user")
			}
		})
	})

	// Test findUnique
	act.runWithCleanup(t, db, func() {
		t.Run("FindUnique", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model User {
					id    Int    @id @default(autoincrement())
					name  String
					email String @unique
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create user
			created, err := client.Model("User").Create(`{
				"data": {"name": "Unique User", "email": "unique@example.com"}
			}`)
			assertNoError(t, err, "Failed to create user")
			
			// Find by id
			byId, err := client.Model("User").FindUnique(`{
				"where": {"id": ` + idToString(created["id"]) + `}
			}`)
			assertNoError(t, err, "Failed to find user by id")
			assertEqual(t, "Unique User", byId["name"], "User name mismatch")
			
			// Find by unique field
			byEmail, err := client.Model("User").FindUnique(`{
				"where": {"email": "unique@example.com"}
			}`)
			assertNoError(t, err, "Failed to find user by email")
			assertEqual(t, created["id"], byEmail["id"], "User ID mismatch")
		})
	})

	// Test count
	act.runWithCleanup(t, db, func() {
		t.Run("Count", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model Product {
					id       Int     @id @default(autoincrement())
					name     String
					category String
					active   Boolean @default(true)
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create test data
			_, err = client.Model("Product").Create(`{"data": {"name": "Product 1", "category": "Electronics"}}`)
			assertNoError(t, err, "Failed to create product 1")
			
			_, err = client.Model("Product").Create(`{"data": {"name": "Product 2", "category": "Electronics"}}`)
			assertNoError(t, err, "Failed to create product 2")
			
			_, err = client.Model("Product").Create(`{"data": {"name": "Product 3", "category": "Books", "active": false}}`)
			assertNoError(t, err, "Failed to create product 3")
			
			// Count all
			total, err := client.Model("Product").Count(`{}`)
			assertNoError(t, err, "Failed to count all products")
			assertEqual(t, int64(3), total, "Total count mismatch")
			
			// Count with filter
			electronics, err := client.Model("Product").Count(`{
				"where": {"category": "Electronics"}
			}`)
			assertNoError(t, err, "Failed to count electronics")
			assertEqual(t, int64(2), electronics, "Electronics count mismatch")
			
			// Count active
			active, err := client.Model("Product").Count(`{
				"where": {"active": true}
			}`)
			assertNoError(t, err, "Failed to count active products")
			assertEqual(t, int64(2), active, "Active count mismatch")
		})
	})

	// Test createMany
	act.runWithCleanup(t, db, func() {
		t.Run("CreateMany", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model Task {
					id     Int    @id @default(autoincrement())
					title  String
					status String @default("pending")
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create multiple records
			result, err := client.Model("Task").Query(`{
				"createMany": {
					"data": [
						{"title": "Task 1"},
						{"title": "Task 2"},
						{"title": "Task 3", "status": "completed"}
					]
				}
			}`)
			assertNoError(t, err, "Failed to create many tasks")
			
			// Check result
			if resultMap, ok := result.(map[string]any); ok {
				if count, ok := resultMap["count"]; ok {
					// Handle different numeric types
					var countInt int
					switch v := count.(type) {
					case int:
						countInt = v
					case int64:
						countInt = int(v)
					case float64:
						countInt = int(v)
					}
					assertEqual(t, 3, countInt, "CreateMany count mismatch")
				}
			}
			
			// Verify records
			tasks, err := client.Model("Task").FindMany(`{"orderBy": {"title": "asc"}}`)
			assertNoError(t, err, "Failed to find tasks")
			
			assertEqual(t, 3, len(tasks), "Task count mismatch")
			assertEqual(t, "Task 1", tasks[0]["title"], "Task 1 title mismatch")
			assertEqual(t, "pending", tasks[0]["status"], "Task 1 status mismatch")
			assertEqual(t, "completed", tasks[2]["status"], "Task 3 status mismatch")
		})
	})

	// Test updateMany
	act.runWithCleanup(t, db, func() {
		t.Run("UpdateMany", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model Task {
					id     Int    @id @default(autoincrement())
					title  String
					status String @default("pending")
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create test data
			_, err = client.Model("Task").Create(`{"data": {"title": "Task 1"}}`)
			assertNoError(t, err, "Failed to create task 1")
			
			_, err = client.Model("Task").Create(`{"data": {"title": "Task 2"}}`)
			assertNoError(t, err, "Failed to create task 2")
			
			_, err = client.Model("Task").Create(`{"data": {"title": "Task 3", "status": "completed"}}`)
			assertNoError(t, err, "Failed to create task 3")
			
			// Update multiple records
			result, err := client.Model("Task").UpdateMany(`{
				"where": {"status": "pending"},
				"data": {"status": "in_progress"}
			}`)
			assertNoError(t, err, "Failed to update many tasks")
			
			// Verify count
			if count, ok := result["count"]; ok {
				assertEqual(t, int64(2), count, "UpdateMany count mismatch")
			}
			
			// Verify updates
			pending, err := client.Model("Task").Count(`{"where": {"status": "pending"}}`)
			assertNoError(t, err, "Failed to count pending tasks")
			assertEqual(t, int64(0), pending, "Pending tasks should be 0")
			
			inProgress, err := client.Model("Task").Count(`{"where": {"status": "in_progress"}}`)
			assertNoError(t, err, "Failed to count in_progress tasks")
			assertEqual(t, int64(2), inProgress, "In progress tasks should be 2")
		})
	})

	// Test deleteMany
	act.runWithCleanup(t, db, func() {
		t.Run("DeleteMany", func(t *testing.T) {
			ctx := context.Background()
			
			// Load schema
			err := db.LoadSchema(ctx, `
				model TempData {
					id        Int      @id @default(autoincrement())
					data      String
					createdAt DateTime @default(now())
					temp      Boolean  @default(true)
				}
			`)
			assertNoError(t, err, "Failed to load schema")
			
			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")
			
			// Create test data
			_, err = client.Model("TempData").Create(`{"data": {"data": "Keep 1", "temp": false}}`)
			assertNoError(t, err, "Failed to create keep 1")
			
			_, err = client.Model("TempData").Create(`{"data": {"data": "Delete 1"}}`)
			assertNoError(t, err, "Failed to create delete 1")
			
			_, err = client.Model("TempData").Create(`{"data": {"data": "Delete 2"}}`)
			assertNoError(t, err, "Failed to create delete 2")
			
			_, err = client.Model("TempData").Create(`{"data": {"data": "Keep 2", "temp": false}}`)
			assertNoError(t, err, "Failed to create keep 2")
			
			// Delete multiple records
			result, err := client.Model("TempData").DeleteMany(`{
				"where": {"temp": true}
			}`)
			assertNoError(t, err, "Failed to delete many")
			
			// Verify count
			if count, ok := result["count"]; ok {
				assertEqual(t, int64(2), count, "DeleteMany count mismatch")
			}
			
			// Verify remaining records
			remaining, err := client.Model("TempData").FindMany(`{}`)
			assertNoError(t, err, "Failed to find remaining records")
			
			assertEqual(t, 2, len(remaining), "Remaining records count mismatch")
			for _, record := range remaining {
				// SQLite returns 0/1 for booleans
				temp := record["temp"]
				if temp != false && temp != 0 && temp != int64(0) {
					t.Fatalf("Expected temp to be false, got %v (%T)", temp, temp)
				}
			}
		})
	})
}

// Helper function to convert ID to string for JSON
func idToString(id any) string {
	switch v := id.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}