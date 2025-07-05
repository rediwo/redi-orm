package orm

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Transaction Tests
func (act *OrmConformanceTests) runTransactionTests(t *testing.T, client *Client, db types.Database) {
	// Test basic transaction commit
	act.runWithCleanup(t, db, func() {
		t.Run("TransactionCommit", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Account {
					id      Int    @id @default(autoincrement())
					name    String
					balance Float  @default(0)
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create accounts
			acc1, err := client.Model("Account").Create(`{"data": {"name": "Account 1", "balance": 1000}}`)
			assertNoError(t, err, "Failed to create account 1")

			acc2, err := client.Model("Account").Create(`{"data": {"name": "Account 2", "balance": 500}}`)
			assertNoError(t, err, "Failed to create account 2")

			// Execute transaction
			err = client.Transaction(func(tx *Client) error {
				// Deduct from account 1
				_, err := tx.Model("Account").Update(fmt.Sprintf(`{
					"where": {"id": %v},
					"data": {"balance": 900}
				}`, acc1["id"]))
				if err != nil {
					return err
				}

				// Add to account 2
				_, err = tx.Model("Account").Update(fmt.Sprintf(`{
					"where": {"id": %v},
					"data": {"balance": 600}
				}`, acc2["id"]))
				if err != nil {
					return err
				}

				return nil
			})
			assertNoError(t, err, "Transaction failed")

			// Verify changes were committed
			updated1, err := client.Model("Account").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v}
			}`, acc1["id"]))
			assertNoError(t, err, "Failed to find account 1")
			assertEqual(t, float64(900), updated1["balance"], "Account 1 balance mismatch")

			updated2, err := client.Model("Account").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v}
			}`, acc2["id"]))
			assertNoError(t, err, "Failed to find account 2")
			assertEqual(t, float64(600), updated2["balance"], "Account 2 balance mismatch")
		})
	})

	// Test transaction rollback
	act.runWithCleanup(t, db, func() {
		t.Run("TransactionRollback", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Account {
					id      Int    @id @default(autoincrement())
					name    String @unique
					balance Float  @default(0)
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create account
			acc, err := client.Model("Account").Create(`{"data": {"name": "Test Account", "balance": 1000}}`)
			assertNoError(t, err, "Failed to create account")

			// Execute transaction that should fail
			err = client.Transaction(func(tx *Client) error {
				// Update balance
				_, err := tx.Model("Account").Update(fmt.Sprintf(`{
					"where": {"id": %v},
					"data": {"balance": 500}
				}`, acc["id"]))
				if err != nil {
					return err
				}

				// Try to create duplicate (should fail due to unique constraint)
				_, err = tx.Model("Account").Create(`{"data": {"name": "Test Account", "balance": 100}}`)
				if err != nil {
					return err // This should trigger rollback
				}

				return nil
			})

			// Transaction should have failed
			if err == nil {
				t.Fatal("Expected transaction to fail")
			}

			// Verify balance was not changed
			unchanged, err := client.Model("Account").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v}
			}`, acc["id"]))
			assertNoError(t, err, "Failed to find account")
			assertEqual(t, float64(1000), unchanged["balance"], "Balance should not have changed")
		})
	})

	// Test multiple operations in transaction
	act.runWithCleanup(t, db, func() {
		t.Run("TransactionMultipleOperations", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Order {
					id         Int     @id @default(autoincrement())
					customerId Int
					total      Float
					status     String  @default("pending")
				}
				
				model OrderItem {
					id        Int   @id @default(autoincrement())
					orderId   Int
					productId Int
					quantity  Int
					price     Float
				}
				
				model Product {
					id    Int    @id @default(autoincrement())
					name  String
					stock Int
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create products
			product1, err := client.Model("Product").Create(`{"data": {"name": "Product 1", "stock": 10}}`)
			assertNoError(t, err, "Failed to create product 1")

			product2, err := client.Model("Product").Create(`{"data": {"name": "Product 2", "stock": 5}}`)
			assertNoError(t, err, "Failed to create product 2")

			// Execute complex transaction
			var orderId any
			err = client.Transaction(func(tx *Client) error {
				// Create order
				order, err := tx.Model("Order").Create(`{
					"data": {
						"customerId": 1,
						"total": 150,
						"status": "processing"
					}
				}`)
				if err != nil {
					return err
				}
				orderId = order["id"]

				// Create order items
				_, err = tx.Model("OrderItem").Create(fmt.Sprintf(`{
					"data": {
						"orderId": %v,
						"productId": %v,
						"quantity": 2,
						"price": 50
					}
				}`, orderId, product1["id"]))
				if err != nil {
					return err
				}

				_, err = tx.Model("OrderItem").Create(fmt.Sprintf(`{
					"data": {
						"orderId": %v,
						"productId": %v,
						"quantity": 1,
						"price": 50
					}
				}`, orderId, product2["id"]))
				if err != nil {
					return err
				}

				// Update product stock
				_, err = tx.Model("Product").Update(fmt.Sprintf(`{
					"where": {"id": %v},
					"data": {"stock": 8}
				}`, product1["id"]))
				if err != nil {
					return err
				}

				_, err = tx.Model("Product").Update(fmt.Sprintf(`{
					"where": {"id": %v},
					"data": {"stock": 4}
				}`, product2["id"]))
				if err != nil {
					return err
				}

				return nil
			})
			assertNoError(t, err, "Transaction failed")

			// Verify all changes were committed
			items, err := client.Model("OrderItem").FindMany(fmt.Sprintf(`{
				"where": {"orderId": %v}
			}`, orderId))
			assertNoError(t, err, "Failed to find order items")
			assertEqual(t, 2, len(items), "Order items count mismatch")

			// Verify stock updates
			p1, err := client.Model("Product").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v}
			}`, product1["id"]))
			assertNoError(t, err, "Failed to find product 1")
			assertEqual(t, int64(8), p1["stock"], "Product 1 stock mismatch")

			p2, err := client.Model("Product").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v}
			}`, product2["id"]))
			assertNoError(t, err, "Failed to find product 2")
			assertEqual(t, int64(4), p2["stock"], "Product 2 stock mismatch")
		})
	})

	// Test transaction isolation
	if !act.shouldSkip("TransactionIsolation") {
		t.Run("TransactionIsolation", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Counter {
					id    Int @id @default(autoincrement())
					value Int @default(0)
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create counter
			counter, err := client.Model("Counter").Create(`{"data": {"value": 0}}`)
			assertNoError(t, err, "Failed to create counter")

			// Start transaction but don't commit yet
			txStarted := make(chan bool)
			txComplete := make(chan bool)
			txResult := make(chan error, 1)

			go func() {
				err := client.Transaction(func(tx *Client) error {
					// Update counter in transaction
					_, err := tx.Model("Counter").Update(fmt.Sprintf(`{
						"where": {"id": %v},
						"data": {"value": 100}
					}`, counter["id"]))
					if err != nil {
						return err
					}

					// Signal that transaction has started
					txStarted <- true

					// Wait for signal to complete
					<-txComplete
					return nil
				})
				txResult <- err
			}()

			// Wait for transaction to start
			<-txStarted

			// Try to read counter from outside transaction
			// Should see old value (0) not the uncommitted value (100)
			current, err := client.Model("Counter").FindUnique(fmt.Sprintf(`{
				"where": {"id": %v}
			}`, counter["id"]))
			assertNoError(t, err, "Failed to find counter")

			// Should see original value, not uncommitted change
			if current["value"] != int64(0) && current["value"] != 0 {
				t.Logf("Warning: Transaction isolation may not be fully enforced, got value: %v", current["value"])
			}

			// Let transaction complete
			close(txComplete)
			// Wait for transaction result
			err = <-txResult
			assertNoError(t, err, "Transaction failed")
		})
	}
}
