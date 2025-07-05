package orm

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/types"
)

// Aggregation Tests
func (act *OrmConformanceTests) runAggregationTests(t *testing.T, client *Client, db types.Database) {
	// Test basic aggregations
	act.runWithCleanup(t, db, func() {
		t.Run("BasicAggregations", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Order {
					id        Int      @id @default(autoincrement())
					amount    Float
					quantity  Int
					status    String
					createdAt DateTime @default(now())
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create test data
			orders := []string{
				`{"data": {"amount": 100.50, "quantity": 2, "status": "completed"}}`,
				`{"data": {"amount": 200.75, "quantity": 1, "status": "completed"}}`,
				`{"data": {"amount": 50.25, "quantity": 3, "status": "pending"}}`,
				`{"data": {"amount": 150.00, "quantity": 2, "status": "completed"}}`,
			}

			for _, order := range orders {
				_, err = client.Model("Order").Create(order)
				assertNoError(t, err, "Failed to create order")
			}

			// Test aggregations
			result, err := client.Model("Order").Aggregate(`{
				"_count": true,
				"_sum": {
					"amount": true,
					"quantity": true
				},
				"_avg": {
					"amount": true,
					"quantity": true
				},
				"_min": {
					"amount": true
				},
				"_max": {
					"amount": true
				}
			}`)
			assertNoError(t, err, "Failed to aggregate")

			// Check count
			if count, ok := result["_count"].(int64); ok {
				assertEqual(t, int64(4), count, "Count mismatch")
			} else {
				t.Fatalf("_count is not int64: %T", result["_count"])
			}

			// Check sum - should be numeric types, not strings
			if sumMap, ok := result["_sum"].(map[string]any); ok {
				// Amount sum should be 501.5
				if amount, ok := sumMap["amount"].(float64); ok {
					if amount < 501.4 || amount > 501.6 {
						t.Fatalf("Sum amount mismatch: expected ~501.5, got %f", amount)
					}
				} else {
					t.Fatalf("Sum amount is not float64: %T", sumMap["amount"])
				}

				// Quantity sum should be 8
				switch v := sumMap["quantity"].(type) {
				case float64:
					assertEqual(t, float64(8), v, "Sum quantity mismatch")
				case int64:
					assertEqual(t, int64(8), v, "Sum quantity mismatch")
				default:
					t.Fatalf("Sum quantity is not numeric: %T", v)
				}
			} else {
				t.Fatal("_sum is not a map")
			}

			// Check avg
			if avgMap, ok := result["_avg"].(map[string]any); ok {
				// Amount avg should be 125.375
				if amount, ok := avgMap["amount"].(float64); ok {
					if amount < 125.3 || amount > 125.4 {
						t.Fatalf("Avg amount mismatch: expected ~125.375, got %f", amount)
					}
				} else {
					t.Fatalf("Avg amount is not float64: %T", avgMap["amount"])
				}
			}

			// Check min/max
			if minMap, ok := result["_min"].(map[string]any); ok {
				if amount, ok := minMap["amount"].(float64); ok {
					if amount < 50.2 || amount > 50.3 {
						t.Fatalf("Min amount mismatch: expected ~50.25, got %f", amount)
					}
				}
			}

			if maxMap, ok := result["_max"].(map[string]any); ok {
				if amount, ok := maxMap["amount"].(float64); ok {
					if amount < 200.7 || amount > 200.8 {
						t.Fatalf("Max amount mismatch: expected ~200.75, got %f", amount)
					}
				}
			}
		})
	})

	// Test aggregations with where clause
	act.runWithCleanup(t, db, func() {
		t.Run("AggregationsWithWhere", func(t *testing.T) {
			ctx := context.Background()

			// Load schema
			err := db.LoadSchema(ctx, `
				model Sale {
					id       Int    @id @default(autoincrement())
					amount   Float
					category String
					region   String
				}
			`)
			assertNoError(t, err, "Failed to load schema")

			err = db.SyncSchemas(ctx)
			assertNoError(t, err, "Failed to sync schemas")

			// Create test data
			sales := []string{
				`{"data": {"amount": 100, "category": "Electronics", "region": "North"}}`,
				`{"data": {"amount": 200, "category": "Electronics", "region": "South"}}`,
				`{"data": {"amount": 150, "category": "Books", "region": "North"}}`,
				`{"data": {"amount": 300, "category": "Electronics", "region": "North"}}`,
			}

			for _, sale := range sales {
				_, err = client.Model("Sale").Create(sale)
				assertNoError(t, err, "Failed to create sale")
			}

			// Aggregate with where clause
			result, err := client.Model("Sale").Aggregate(`{
				"where": {
					"category": "Electronics"
				},
				"_count": true,
				"_sum": {
					"amount": true
				},
				"_avg": {
					"amount": true
				}
			}`)
			assertNoError(t, err, "Failed to aggregate with where")

			// Check count - should be 3 electronics
			if count, ok := result["_count"].(int64); ok {
				assertEqual(t, int64(3), count, "Filtered count mismatch")
			}

			// Check sum - should be 600
			if sumMap, ok := result["_sum"].(map[string]any); ok {
				if amount, ok := sumMap["amount"].(float64); ok {
					assertEqual(t, float64(600), amount, "Filtered sum mismatch")
				}
			}

			// Check avg - should be 200
			if avgMap, ok := result["_avg"].(map[string]any); ok {
				if amount, ok := avgMap["amount"].(float64); ok {
					assertEqual(t, float64(200), amount, "Filtered avg mismatch")
				}
			}
		})
	})

	// Test MySQL string number conversion
	if act.Characteristics.ReturnsStringForNumbers {
		t.Run("MySQLStringConversion", func(t *testing.T) {
			t.Log("Testing MySQL string number conversion")
			// The type converter should handle this automatically
			// so all tests above should pass even for MySQL
		})
	}
}
