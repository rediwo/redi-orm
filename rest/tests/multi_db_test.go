package tests

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rediwo/redi-orm/database"
	_ "github.com/rediwo/redi-orm/drivers/mongodb"
	_ "github.com/rediwo/redi-orm/drivers/mysql"
	_ "github.com/rediwo/redi-orm/drivers/postgresql"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
	"github.com/rediwo/redi-orm/rest"
)

// TestMultiDatabaseSupport tests REST API with different database backends
func TestMultiDatabaseSupport(t *testing.T) {
	// Define test databases
	databases := []struct {
		name string
		uri  string
		skip bool
	}{
		{
			name: "SQLite",
			uri:  "sqlite://:memory:",
			skip: false,
		},
		{
			name: "MySQL",
			uri:  os.Getenv("MYSQL_TEST_URI"),
			skip: os.Getenv("MYSQL_TEST_URI") == "",
		},
		{
			name: "PostgreSQL",
			uri:  os.Getenv("POSTGRESQL_TEST_URI"),
			skip: os.Getenv("POSTGRESQL_TEST_URI") == "",
		},
		{
			name: "MongoDB",
			uri:  os.Getenv("MONGODB_TEST_URI"),
			skip: os.Getenv("MONGODB_TEST_URI") == "",
		},
	}

	for _, dbConfig := range databases {
		t.Run(dbConfig.name, func(t *testing.T) {
			if dbConfig.skip {
				t.Skipf("Skipping %s test (set %s_TEST_URI to enable)", dbConfig.name, dbConfig.name)
			}

			// Create database
			db, err := database.NewFromURI(dbConfig.uri)
			if err != nil {
				t.Fatalf("Failed to create database: %v", err)
			}
			defer db.Close()

			// Connect to database
			ctx := context.Background()
			err = db.Connect(ctx)
			if err != nil {
				t.Fatalf("Failed to connect to database: %v", err)
			}

			// Load schema
			err = db.LoadSchema(ctx, multiDBSchema)
			if err != nil {
				t.Fatalf("Failed to load schema: %v", err)
			}

			// Sync schemas
			err = db.SyncSchemas(ctx)
			if err != nil {
				t.Fatalf("Failed to sync schemas: %v", err)
			}

			// Create REST server
			config := rest.ServerConfig{
				Database: db,
				LogLevel: "error",
			}

			server, err := rest.NewServer(config)
			if err != nil {
				t.Fatalf("Failed to create REST server: %v", err)
			}
			defer server.Stop()

			// Create test server
			ts := httptest.NewServer(server.Router)
			defer ts.Close()

			// Run basic CRUD tests
			t.Run("CRUD", func(t *testing.T) {
				testCRUDOperations(t, ts)
			})

			// Run relation tests
			t.Run("Relations", func(t *testing.T) {
				testRelationOperations(t, ts)
			})

			// Run complex query tests
			t.Run("ComplexQueries", func(t *testing.T) {
				testComplexQueryOperations(t, ts)
			})
		})
	}
}

func testCRUDOperations(t *testing.T, ts *httptest.Server) {
	// Create product
	createResp := makeRequest(t, ts, "POST", "/api/Product", map[string]any{
		"data": map[string]any{
			"name":        "Test Product",
			"description": "A test product",
			"price":       99.99,
			"stock":       100,
		},
	})

	if !createResp.Success {
		t.Fatalf("Failed to create product: %s", createResp.Error.Message)
	}

	product := createResp.Data.(map[string]any)
	productID := int(product["id"].(float64))

	// Read product
	readResp := makeRequest(t, ts, "GET", fmt.Sprintf("/api/Product/%d", productID), nil)
	if !readResp.Success {
		t.Fatalf("Failed to read product: %s", readResp.Error.Message)
	}

	// Update product
	updateResp := makeRequest(t, ts, "PUT", fmt.Sprintf("/api/Product/%d", productID), map[string]any{
		"data": map[string]any{
			"price": 89.99,
		},
	})
	if !updateResp.Success {
		t.Fatalf("Failed to update product: %s", updateResp.Error.Message)
	}

	updatedProduct := updateResp.Data.(map[string]any)
	if updatedProduct["price"].(float64) != 89.99 {
		t.Errorf("Expected price to be 89.99, got %v", updatedProduct["price"])
	}

	// Delete product
	deleteResp := makeRequest(t, ts, "DELETE", fmt.Sprintf("/api/Product/%d", productID), nil)
	if !deleteResp.Success {
		t.Fatalf("Failed to delete product: %s", deleteResp.Error.Message)
	}
}

func testRelationOperations(t *testing.T, ts *httptest.Server) {
	// Create category
	categoryResp := makeRequest(t, ts, "POST", "/api/Category", map[string]any{
		"data": map[string]any{
			"name":        "Electronics",
			"description": "Electronic products",
		},
	})
	if !categoryResp.Success {
		t.Fatalf("Failed to create category: %s", categoryResp.Error.Message)
	}

	category := categoryResp.Data.(map[string]any)
	categoryID := int(category["id"].(float64))

	// Create product with category
	productResp := makeRequest(t, ts, "POST", "/api/Product", map[string]any{
		"data": map[string]any{
			"name":        "Laptop",
			"description": "High-end laptop",
			"price":       1299.99,
			"stock":       50,
			"categoryId":  categoryID,
		},
	})
	if !productResp.Success {
		t.Fatalf("Failed to create product: %s", productResp.Error.Message)
	}

	product := productResp.Data.(map[string]any)
	productID := int(product["id"].(float64))

	// Get product with category
	includeResp := makeRequest(t, ts, "GET", fmt.Sprintf("/api/Product/%d?include=category", productID), nil)
	if !includeResp.Success {
		t.Fatalf("Failed to get product with category: %s", includeResp.Error.Message)
	}

	productWithCategory := includeResp.Data.(map[string]any)
	if productWithCategory["category"] == nil {
		t.Error("Expected category to be included")
	}

	includedCategory := productWithCategory["category"].(map[string]any)
	if includedCategory["name"] != "Electronics" {
		t.Errorf("Expected category name to be 'Electronics', got %v", includedCategory["name"])
	}

	// Get category with products
	categoryWithProductsResp := makeRequest(t, ts, "GET", fmt.Sprintf("/api/Category/%d?include=products", categoryID), nil)
	if !categoryWithProductsResp.Success {
		t.Fatalf("Failed to get category with products: %s", categoryWithProductsResp.Error.Message)
	}

	categoryWithProducts := categoryWithProductsResp.Data.(map[string]any)
	products := categoryWithProducts["products"].([]any)
	if len(products) != 1 {
		t.Errorf("Expected 1 product in category, got %d", len(products))
	}
}

func testComplexQueryOperations(t *testing.T, ts *httptest.Server) {
	// Clear existing data first
	resp := makeRequest(t, ts, "GET", "/api/Product", nil)
	if resp.Success {
		products := resp.Data.([]any)
		for _, p := range products {
			product := p.(map[string]any)
			id := int(product["id"].(float64))
			makeRequest(t, ts, "DELETE", fmt.Sprintf("/api/Product/%d", id), nil)
		}
	}

	// Create test data
	for i := 1; i <= 5; i++ {
		resp := makeRequest(t, ts, "POST", "/api/Product", map[string]any{
			"data": map[string]any{
				"name":        fmt.Sprintf("Product %d", i),
				"description": fmt.Sprintf("Description for product %d", i),
				"price":       float64(i * 10),
				"stock":       i * 5,
			},
		})
		if !resp.Success {
			t.Fatalf("Failed to create product %d", i)
		}
	}

	// Test filtering
	filterResp := makeRequest(t, ts, "GET", `/api/Product?where={"price":{"gte":30}}`, nil)
	if !filterResp.Success {
		t.Fatalf("Failed to filter products: %s", filterResp.Error.Message)
	}

	filteredProducts := filterResp.Data.([]any)
	if len(filteredProducts) != 3 {
		t.Errorf("Expected 3 products with price >= 30, got %d", len(filteredProducts))
	}

	// Test sorting and pagination
	paginationResp := makeRequest(t, ts, "GET", "/api/Product?sort=-price&page=1&limit=2", nil)
	if !paginationResp.Success {
		t.Fatalf("Failed to get paginated products: %s", paginationResp.Error.Message)
	}

	paginatedProducts := paginationResp.Data.([]any)
	if len(paginatedProducts) != 2 {
		t.Errorf("Expected 2 products per page, got %d", len(paginatedProducts))
	}

	// Check sorting (should be highest price first)
	firstProduct := paginatedProducts[0].(map[string]any)
	if firstProduct["price"].(float64) != 50 {
		t.Errorf("Expected first product price to be 50, got %v", firstProduct["price"])
	}

	// Check pagination metadata
	if paginationResp.Pagination == nil {
		t.Fatal("Expected pagination metadata")
	}

	if paginationResp.Pagination.Total < 5 {
		t.Errorf("Expected at least 5 total products, got %d", paginationResp.Pagination.Total)
	}
}

const multiDBSchema = `
model Product {
  id          Int       @id @default(autoincrement())
  name        String
  description String?
  price       Float
  stock       Int       @default(0)
  categoryId  Int?
  category    Category? @relation(fields: [categoryId], references: [id])
  orders      Order[]
  createdAt   DateTime  @default(now())
  updatedAt   DateTime  @updatedAt
}

model Category {
  id          Int       @id @default(autoincrement())
  name        String    @unique
  description String?
  products    Product[]
}

model Customer {
  id        Int      @id @default(autoincrement())
  name      String
  email     String   @unique
  phone     String?
  orders    Order[]
  createdAt DateTime @default(now())
}

model Order {
  id         Int         @id @default(autoincrement())
  customerId Int
  customer   Customer    @relation(fields: [customerId], references: [id])
  items      OrderItem[]
  total      Float
  status     String      @default("pending")
  createdAt  DateTime    @default(now())
}

model OrderItem {
  id        Int     @id @default(autoincrement())
  orderId   Int
  productId Int
  order     Order   @relation(fields: [orderId], references: [id])
  product   Product @relation(fields: [productId], references: [id])
  quantity  Int
  price     Float
}
`
