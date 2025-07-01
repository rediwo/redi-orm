package test

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/engine"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/types"
)

func TestPrismaASTDemo(t *testing.T) {
	// Create database
	db, err := database.New(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	// Create Prisma AST equivalent to:
	// model User {
	//   id    Int    @id @default(autoincrement())
	//   email String @unique
	//   name  String
	//   posts Post[]
	// }
	userModel := &prisma.ModelStatement{
		Name: "User",
		Fields: []*prisma.Field{
			{
				Name: "id",
				Type: &prisma.FieldType{Name: "Int"},
				Attributes: []*prisma.Attribute{
					{Name: "id"},
					{
						Name: "default",
						Args: []prisma.Expression{
							&prisma.FunctionCall{Name: "autoincrement"},
						},
					},
				},
			},
			{
				Name: "email",
				Type: &prisma.FieldType{Name: "String"},
				Attributes: []*prisma.Attribute{
					{Name: "unique"},
				},
			},
			{
				Name: "name",
				Type: &prisma.FieldType{Name: "String"},
			},
			{
				Name: "posts",
				Type: &prisma.FieldType{Name: "Post"},
				List: true,
			},
		},
	}

	// Create Post model equivalent to:
	// model Post {
	//   id       Int    @id @default(autoincrement())
	//   title    String
	//   content  String
	//   userId   Int
	//   author   User   @relation(fields: [userId], references: [id])
	// }
	postModel := &prisma.ModelStatement{
		Name: "Post",
		Fields: []*prisma.Field{
			{
				Name: "id",
				Type: &prisma.FieldType{Name: "Int"},
				Attributes: []*prisma.Attribute{
					{Name: "id"},
					{
						Name: "default",
						Args: []prisma.Expression{
							&prisma.FunctionCall{Name: "autoincrement"},
						},
					},
				},
			},
			{
				Name: "title",
				Type: &prisma.FieldType{Name: "String"},
			},
			{
				Name: "content",
				Type: &prisma.FieldType{Name: "String"},
			},
			{
				Name: "userId",
				Type: &prisma.FieldType{Name: "Int"},
			},
			{
				Name: "author",
				Type: &prisma.FieldType{Name: "User"},
				Attributes: []*prisma.Attribute{
					{
						Name: "relation",
						Args: []prisma.Expression{
							&prisma.FunctionCall{
								Name: "fields",
								Args: []prisma.Expression{
									&prisma.ArrayExpression{
										Elements: []prisma.Expression{
											&prisma.Identifier{Value: "userId"},
										},
									},
								},
							},
							&prisma.FunctionCall{
								Name: "references",
								Args: []prisma.Expression{
									&prisma.ArrayExpression{
										Elements: []prisma.Expression{
											&prisma.Identifier{Value: "id"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Prisma schema AST
	prismaSchema := &prisma.PrismaSchema{
		Statements: []prisma.Statement{userModel, postModel},
	}

	// Convert Prisma AST to ReORM schemas
	converter := prisma.NewConverter()
	schemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("Failed to convert Prisma schema: %v", err)
	}

	if len(schemas) != 2 {
		t.Errorf("Expected 2 schemas, got %d", len(schemas))
	}

	// Register schemas with database and create tables
	for name, schema := range schemas {
		// Register with database
		if err := db.RegisterSchema(name, schema); err != nil {
			t.Fatalf("Failed to register schema %s with database: %v", name, err)
		}
		
		// Create table
		if err := db.CreateModel(schema); err != nil {
			t.Fatalf("Failed to create table for schema %s: %v", name, err)
		}
	}
	
	// Register schemas with the JavaScript engine
	for name, schema := range schemas {
		if err := eng.RegisterSchema(schema); err != nil {
			t.Fatalf("Failed to register schema %s with JS engine: %v", name, err)
		}
	}

	// Test JavaScript operations
	t.Run("JavaScript operations", func(t *testing.T) {
		// Create users
		result1, err := eng.Execute(`models.User.add({name: "Alice", email: "alice@prisma.com"})`)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		if result1 != int64(1) {
			t.Errorf("Expected user ID 1, got %v", result1)
		}

		result2, err := eng.Execute(`models.User.add({name: "Bob", email: "bob@prisma.com"})`)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		if result2 != int64(2) {
			t.Errorf("Expected user ID 2, got %v", result2)
		}

		// Create posts
		_, err = eng.Execute(`models.Post.add({title: "Getting Started with Prisma", content: "This is a great tutorial", userId: 1})`)
		if err != nil {
			t.Fatalf("Failed to create post: %v", err)
		}

		// Query data
		result, err := eng.Execute(`models.User.select().count()`)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}
		if result != int64(2) {
			t.Errorf("Expected 2 users, got %v", result)
		}

		// Query with where clause
		result, err = eng.Execute(`models.Post.select().where("userId", "=", 1).count()`)
		if err != nil {
			t.Fatalf("Failed to query posts: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected 1 post for user 1, got %v", result)
		}
	})
}

func TestPrismaSchemaExample(t *testing.T) {
	// Create database
	db, err := database.New(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	// Define schema using Prisma syntax
	prismaSchema := `
// Blog application schema
enum UserRole {
  ADMIN
  AUTHOR
  READER
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  role      UserRole @default(READER)
  posts     Post[]
  comments  Comment[]
  createdAt DateTime @default(now())
  
  @@map("users")
}

model Post {
  id        Int       @id @default(autoincrement())
  title     String
  content   String
  published Boolean   @default(false)
  authorId  Int
  author    User      @relation(fields: [authorId], references: [id])
  comments  Comment[]
  tags      Tag[]
  createdAt DateTime  @default(now())
  updatedAt DateTime  @default(now())
  
  @@index([published])
  @@index([authorId])
}

model Comment {
  id       Int      @id @default(autoincrement())
  content  String
  postId   Int
  userId   Int
  post     Post     @relation(fields: [postId], references: [id])
  user     User     @relation(fields: [userId], references: [id])
  createdAt DateTime @default(now())
}

model Tag {
  id    Int    @id @default(autoincrement())
  name  String @unique
  posts Post[]
}`

	// Load Prisma schema
	if err := eng.LoadPrismaSchema(prismaSchema); err != nil {
		t.Fatalf("Failed to load Prisma schema: %v", err)
	}

	// Test JavaScript operations
	t.Run("Complex schema operations", func(t *testing.T) {
		// Create a user
		result, err := eng.Execute(`models.User.add({name: "John Doe", email: "john@example.com", role: "AUTHOR"})`)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected user ID 1, got %v", result)
		}

		// Create another user
		_, err = eng.Execute(`models.User.add({name: "Jane Smith", email: "jane@example.com", role: "READER"})`)
		if err != nil {
			t.Fatalf("Failed to create second user: %v", err)
		}

		// Create a post
		_, err = eng.Execute(`models.Post.add({title: "My First Post", content: "Hello, world!", authorId: 1, published: true})`)
		if err != nil {
			t.Fatalf("Failed to create post: %v", err)
		}

		// Create a comment
		_, err = eng.Execute(`models.Comment.add({content: "Great post!", postId: 1, userId: 2})`)
		if err != nil {
			t.Fatalf("Failed to create comment: %v", err)
		}

		// Create a tag
		_, err = eng.Execute(`models.Tag.add({name: "javascript"})`)
		if err != nil {
			t.Fatalf("Failed to create tag: %v", err)
		}

		// Count total users
		result, err = eng.Execute(`models.User.select().count()`)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}
		if result != int64(2) {
			t.Errorf("Expected 2 users, got %v", result)
		}

		// Get user by email
		result, err = eng.Execute(`models.User.select().where("email", "=", "john@example.com").first()`)
		if err != nil {
			t.Fatalf("Failed to find user by email: %v", err)
		}
		if result == nil {
			t.Error("Expected to find user, got nil")
		}

		// Get published posts
		result, err = eng.Execute(`models.Post.select().where("published", "=", true).count()`)
		if err != nil {
			t.Fatalf("Failed to count published posts: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected 1 published post, got %v", result)
		}

		// Get comments for a post
		result, err = eng.Execute(`models.Comment.select().where("postId", "=", 1).count()`)
		if err != nil {
			t.Fatalf("Failed to count comments: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected 1 comment for post 1, got %v", result)
		}

		// Update a post
		_, err = eng.Execute(`models.Post.set(1, {title: "My Updated First Post"})`)
		if err != nil {
			t.Fatalf("Failed to update post: %v", err)
		}

		// Verify update
		result, err = eng.Execute(`models.Post.get(1)`)
		if err != nil {
			t.Fatalf("Failed to get updated post: %v", err)
		}
		if post, ok := result.(map[string]interface{}); ok {
			if post["title"] != "My Updated First Post" {
				t.Errorf("Expected updated title, got %v", post["title"])
			}
		} else {
			t.Errorf("Expected map result, got %T", result)
		}
	})
}

func TestPrismaEnumMapping(t *testing.T) {
	// Create database
	db, err := database.New(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	// Define schema with enum mapping
	prismaSchema := `
enum Status {
  DRAFT     @map("draft")
  PUBLISHED @map("published")
  ARCHIVED  @map("archived")
}

model Article {
  id     Int    @id @default(autoincrement())
  title  String
  status Status @default(DRAFT)
}`

	// Load Prisma schema
	if err := eng.LoadPrismaSchema(prismaSchema); err != nil {
		t.Fatalf("Failed to load Prisma schema: %v", err)
	}

	// Test enum operations
	t.Run("Enum mapping operations", func(t *testing.T) {
		// Create article with default status
		result, err := eng.Execute(`models.Article.add({title: "Test Article"})`)
		if err != nil {
			t.Fatalf("Failed to create article: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected article ID 1, got %v", result)
		}

		// Create article with specific status
		_, err = eng.Execute(`models.Article.add({title: "Published Article", status: "PUBLISHED"})`)
		if err != nil {
			t.Fatalf("Failed to create published article: %v", err)
		}

		// Count articles
		result, err = eng.Execute(`models.Article.select().count()`)
		if err != nil {
			t.Fatalf("Failed to count articles: %v", err)
		}
		if result != int64(2) {
			t.Errorf("Expected 2 articles, got %v", result)
		}

		// Query by status
		result, err = eng.Execute(`models.Article.select().where("status", "=", "DRAFT").count()`)
		if err != nil {
			t.Fatalf("Failed to count draft articles: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected 1 draft article, got %v", result)
		}
	})
}

func TestPrismaCompositeKeys(t *testing.T) {
	// Create database
	db, err := database.New(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	// Define schema with composite primary key
	prismaSchema := `
model UserRole {
  userId   Int
  roleId   Int
  grantedAt DateTime @default(now())
  
  @@id([userId, roleId])
}

model User {
  id   Int    @id @default(autoincrement())
  name String
}

model Role {
  id   Int    @id @default(autoincrement()) 
  name String
}`

	// Load Prisma schema
	if err := eng.LoadPrismaSchema(prismaSchema); err != nil {
		t.Fatalf("Failed to load Prisma schema: %v", err)
	}

	// Test composite key operations
	t.Run("Composite key operations", func(t *testing.T) {
		// Create users and roles first
		_, err := eng.Execute(`models.User.add({name: "Alice"})`)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		_, err = eng.Execute(`models.Role.add({name: "Admin"})`)
		if err != nil {
			t.Fatalf("Failed to create role: %v", err)
		}

		// Create user role with composite key
		_, err = eng.Execute(`models.UserRole.add({userId: 1, roleId: 1})`)
		if err != nil {
			t.Fatalf("Failed to create user role: %v", err)
		}

		// Count user roles
		result, err := eng.Execute(`models.UserRole.select().count()`)
		if err != nil {
			t.Fatalf("Failed to count user roles: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected 1 user role, got %v", result)
		}

		// Query by composite key fields
		result, err = eng.Execute(`models.UserRole.select().where("userId", "=", 1).where("roleId", "=", 1).count()`)
		if err != nil {
			t.Fatalf("Failed to query by composite key: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected 1 user role for composite key, got %v", result)
		}
	})
}

func TestPrismaDecimalSupport(t *testing.T) {
	// Create database
	db, err := database.New(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create engine
	eng := engine.New(db)

	// Define schema with Decimal type
	prismaSchema := `
model Product {
  id       Int     @id @default(autoincrement())
  name     String
  price    Decimal @db.Decimal(10,2)
  discount Decimal?
}`

	// Load Prisma schema
	if err := eng.LoadPrismaSchema(prismaSchema); err != nil {
		t.Fatalf("Failed to load Prisma schema: %v", err)
	}

	// Test Decimal operations
	t.Run("Decimal type operations", func(t *testing.T) {
		// Create product with decimal price
		result, err := eng.Execute(`models.Product.add({name: "Test Product", price: 19.99})`)
		if err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected product ID 1, got %v", result)
		}

		// Get product and verify price
		result, err = eng.Execute(`models.Product.get(1)`)
		if err != nil {
			t.Fatalf("Failed to get product: %v", err)
		}

		if product, ok := result.(map[string]interface{}); ok {
			if product["name"] != "Test Product" {
				t.Errorf("Expected product name 'Test Product', got %v", product["name"])
			}
			// Price should be preserved as decimal value
			if product["price"] == nil {
				t.Error("Expected price to be set")
			}
		} else {
			t.Errorf("Expected map result, got %T", result)
		}

		// Count products
		result, err = eng.Execute(`models.Product.select().count()`)
		if err != nil {
			t.Fatalf("Failed to count products: %v", err)
		}
		if result != int64(1) {
			t.Errorf("Expected 1 product, got %v", result)
		}
	})
}
