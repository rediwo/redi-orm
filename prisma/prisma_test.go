package prisma

import (
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
)

func TestLexerBasic(t *testing.T) {
	input := `model User {
  id    Int     @id @default(autoincrement())
  name  String
  email String  @unique
}`

	lexer := NewLexer(input)

	expectedTokens := []TokenType{
		MODEL, IDENT, LBRACE,
		IDENT, INT, AT, IDENT, AT, DEFAULT, LPAREN, AUTOINCREMENT, LPAREN, RPAREN, RPAREN,
		IDENT, STRING_TYPE,
		IDENT, STRING_TYPE, AT, IDENT,
		RBRACE,
		EOF,
	}

	for i, expected := range expectedTokens {
		token := lexer.NextToken()
		if token.Type != expected {
			t.Errorf("token %d: expected %s, got %s", i, expected, token.Type)
		}
	}
}

func TestParserModel(t *testing.T) {
	input := `model User {
  id    Int     @id @default(autoincrement())
  name  String
  email String  @unique
  posts Post[]
  
  @@unique([email])
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	schema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	if len(schema.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(schema.Statements))
	}
	
	modelStmt, ok := schema.Statements[0].(*ModelStatement)
	if !ok {
		t.Fatalf("expected ModelStatement, got %T", schema.Statements[0])
	}
	
	if modelStmt.Name != "User" {
		t.Errorf("expected model name User, got %s", modelStmt.Name)
	}
	
	if len(modelStmt.Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(modelStmt.Fields))
	}
	
	// Check id field
	idField := modelStmt.Fields[0]
	if idField.Name != "id" || idField.Type.Name != "Int" {
		t.Errorf("unexpected id field: %+v", idField)
	}
	if len(idField.Attributes) != 2 {
		t.Errorf("expected 2 attributes for id field, got %d", len(idField.Attributes))
	}
	
	// Check block attributes
	if len(modelStmt.BlockAttributes) != 1 {
		t.Errorf("expected 1 block attribute, got %d", len(modelStmt.BlockAttributes))
	}
}

func TestParserEnum(t *testing.T) {
	input := `enum Role {
  ADMIN
  USER
  GUEST
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	schema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	if len(schema.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(schema.Statements))
	}
	
	enumStmt, ok := schema.Statements[0].(*EnumStatement)
	if !ok {
		t.Fatalf("expected EnumStatement, got %T", schema.Statements[0])
	}
	
	if enumStmt.Name != "Role" {
		t.Errorf("expected enum name Role, got %s", enumStmt.Name)
	}
	
	expectedValues := []string{"ADMIN", "USER", "GUEST"}
	if len(enumStmt.Values) != len(expectedValues) {
		t.Errorf("expected %d values, got %d", len(expectedValues), len(enumStmt.Values))
	}
	
	for i, expected := range expectedValues {
		if i < len(enumStmt.Values) && enumStmt.Values[i].Name != expected {
			t.Errorf("expected value %s at index %d, got %s", expected, i, enumStmt.Values[i].Name)
		}
	}
}

func TestConverterBasic(t *testing.T) {
	input := `model User {
  id    Int     @id @default(autoincrement())
  name  String
  email String  @unique
  age   Int?
  active Boolean @default(true)
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	converter := NewConverter()
	schemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	
	userSchema, exists := schemas["User"]
	if !exists {
		t.Fatalf("User schema not found")
	}
	
	if userSchema.Name != "User" {
		t.Errorf("expected schema name User, got %s", userSchema.Name)
	}
	
	if len(userSchema.Fields) != 5 {
		t.Errorf("expected 5 fields, got %d", len(userSchema.Fields))
	}
	
	// Check id field
	idField, err := userSchema.GetField("id")
	if err != nil {
		t.Fatalf("id field not found: %v", err)
	}
	if !idField.PrimaryKey || !idField.AutoIncrement {
		t.Errorf("id field should be primary key with auto increment")
	}
	if idField.Type != schema.FieldTypeInt {
		t.Errorf("expected id field type Int, got %s", idField.Type)
	}
	
	// Check email field
	emailField, err := userSchema.GetField("email")
	if err != nil {
		t.Fatalf("email field not found: %v", err)
	}
	if !emailField.Unique {
		t.Errorf("email field should be unique")
	}
	
	// Check age field
	ageField, err := userSchema.GetField("age")
	if err != nil {
		t.Fatalf("age field not found: %v", err)
	}
	if !ageField.Nullable {
		t.Errorf("age field should be nullable")
	}
	
	// Check active field
	activeField, err := userSchema.GetField("active")
	if err != nil {
		t.Fatalf("active field not found: %v", err)
	}
	if activeField.Default != true {
		t.Errorf("active field default should be true, got %v", activeField.Default)
	}
}

func TestConverterWithRelations(t *testing.T) {
	input := `model User {
  id    Int    @id @default(autoincrement())
  name  String
  posts Post[]
}

model Post {
  id      Int    @id @default(autoincrement())
  title   String
  content String
  userId  Int
  author  User   @relation(fields: [userId], references: [id])
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	converter := NewConverter()
	schemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	if len(schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(schemas))
	}
	
	userSchema, exists := schemas["User"]
	if !exists {
		t.Fatalf("User schema not found")
	}
	
	postSchema, exists := schemas["Post"]
	if !exists {
		t.Fatalf("Post schema not found")
	}
	
	// Check User relations
	if len(userSchema.Relations) != 1 {
		t.Errorf("expected 1 relation in User schema, got %d", len(userSchema.Relations))
	}
	
	postsRelation, exists := userSchema.Relations["posts"]
	if !exists {
		t.Errorf("posts relation not found in User schema")
	} else {
		if postsRelation.Type != schema.RelationOneToMany {
			t.Errorf("expected OneToMany relation, got %s", postsRelation.Type)
		}
		if postsRelation.Model != "Post" {
			t.Errorf("expected relation to Post model, got %s", postsRelation.Model)
		}
	}
	
	// Check Post relations
	if len(postSchema.Relations) != 1 {
		t.Errorf("expected 1 relation in Post schema, got %d", len(postSchema.Relations))
	}
	
	authorRelation, exists := postSchema.Relations["author"]
	if !exists {
		t.Errorf("author relation not found in Post schema")
	} else {
		if authorRelation.Type != schema.RelationManyToOne {
			t.Errorf("expected ManyToOne relation, got %s", authorRelation.Type)
		}
		if authorRelation.Model != "User" {
			t.Errorf("expected relation to User model, got %s", authorRelation.Model)
		}
		if authorRelation.ForeignKey != "userId" {
			t.Errorf("expected foreign key userId, got %s", authorRelation.ForeignKey)
		}
		if authorRelation.References != "id" {
			t.Errorf("expected references id, got %s", authorRelation.References)
		}
	}
}

func TestConverterWithEnum(t *testing.T) {
	input := `enum Role {
  ADMIN
  USER
}

model User {
  id   Int    @id @default(autoincrement())
  name String
  role Role   @default(USER)
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	converter := NewConverter()
	schemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
	
	userSchema, exists := schemas["User"]
	if !exists {
		t.Fatalf("User schema not found")
	}
	
	// Check role field
	roleField, err := userSchema.GetField("role")
	if err != nil {
		t.Fatalf("role field not found: %v", err)
	}
	
	if roleField.Type != schema.FieldTypeString {
		t.Errorf("expected role field type String (for enum), got %s", roleField.Type)
	}
	
	if roleField.Default != "USER" {
		t.Errorf("expected role field default USER, got %v", roleField.Default)
	}
}

func TestFullPrismaSchema(t *testing.T) {
	input := `// This is a comment
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-js"
}

enum UserRole {
  ADMIN
  USER
  MODERATOR
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  role      UserRole @default(USER)
  posts     Post[]
  profile   Profile?
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  @@map("users")
  @@index([email])
}

model Profile {
  id     Int    @id @default(autoincrement())
  bio    String?
  userId Int    @unique
  user   User   @relation(fields: [userId], references: [id])
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
  tags      Tag[]
  createdAt DateTime @default(now())

  @@index([published])
}

model Tag {
  id    Int    @id @default(autoincrement())
  name  String @unique
  posts Post[]
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	converter := NewConverter()
	schemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	expectedModels := []string{"User", "Profile", "Post", "Tag"}
	if len(schemas) != len(expectedModels) {
		t.Fatalf("expected %d schemas, got %d", len(expectedModels), len(schemas))
	}
	
	for _, modelName := range expectedModels {
		if _, exists := schemas[modelName]; !exists {
			t.Errorf("schema %s not found", modelName)
		}
	}
	
	// Check User schema table name mapping
	userSchema := schemas["User"]
	if userSchema.TableName != "users" {
		t.Errorf("expected User table name to be 'users', got '%s'", userSchema.TableName)
	}
	
	// Check that schemas are valid
	for name, schema := range schemas {
		if err := schema.Validate(); err != nil {
			t.Errorf("schema %s validation failed: %v", name, err)
		}
	}
}

func TestStringRepresentation(t *testing.T) {
	input := `model User {
  id   Int    @id
  name String
}`

	lexer := NewLexer(input)
	parser := NewParser(lexer)
	
	schema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	output := schema.String()
	
	// Check that the string output contains expected elements
	expectedStrings := []string{"model User", "id Int @id", "name String"}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain '%s', got:\n%s", expected, output)
		}
	}
}

func TestComplexPrismaSchema(t *testing.T) {
	// Test a complex Prisma schema with advanced features
	complexSchema := `
// Advanced Prisma schema with many features
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "prisma-client-js"
  output   = "./generated/client"
}

enum Status {
  DRAFT
  PUBLISHED
  ARCHIVED
}

enum Role {
  USER
  ADMIN
  MODERATOR
}

model User {
  id          String   @id @default(cuid())
  email       String   @unique
  username    String   @unique @db.VarChar(50)
  displayName String?
  avatar      String?
  bio         String?  @db.Text
  role        Role     @default(USER)
  isActive    Boolean  @default(true)
  lastLoginAt DateTime?
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt

  // Relations
  posts       Post[]
  comments    Comment[]
  likes       Like[]
  profile     Profile?

  @@map("users")
  @@index([email])
  @@index([username])
  @@index([role, isActive])
}

model Profile {
  id        String  @id @default(cuid())
  firstName String?
  lastName  String?
  phone     String?
  website   String?
  location  String?
  userId    String  @unique
  user      User    @relation(fields: [userId], references: [id], onDelete: Cascade)

  @@map("profiles")
}

model Post {
  id          String   @id @default(cuid())
  title       String   @db.VarChar(200)
  slug        String   @unique
  content     String   @db.Text
  excerpt     String?  @db.VarChar(500)
  status      Status   @default(DRAFT)
  publishedAt DateTime?
  viewCount   Int      @default(0)
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt

  // Relations
  authorId String
  author   User      @relation(fields: [authorId], references: [id], onDelete: Cascade)
  comments Comment[]
  likes    Like[]

  @@map("posts")
  @@index([authorId])
  @@index([status])
  @@index([publishedAt])
  @@unique([authorId, slug])
}

model Comment {
  id        String   @id @default(cuid())
  content   String   @db.Text
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  // Relations
  postId String
  post   Post   @relation(fields: [postId], references: [id], onDelete: Cascade)
  userId String
  user   User   @relation(fields: [userId], references: [id], onDelete: Cascade)

  @@map("comments")
  @@index([postId])
  @@index([userId])
}`

	lexer := NewLexer(complexSchema)
	parser := NewParser(lexer)
	
	schema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	// Count different types of statements
	var datasources, generators, enums, models int
	for _, stmt := range schema.Statements {
		switch stmt.(type) {
		case *DatasourceStatement:
			datasources++
		case *GeneratorStatement:
			generators++
		case *EnumStatement:
			enums++
		case *ModelStatement:
			models++
		}
	}
	
	// Verify counts
	if datasources != 1 {
		t.Errorf("expected 1 datasource, got %d", datasources)
	}
	if generators != 1 {
		t.Errorf("expected 1 generator, got %d", generators)
	}
	if enums != 2 {
		t.Errorf("expected 2 enums, got %d", enums)
	}
	if models != 4 {
		t.Errorf("expected 4 models, got %d", models)
	}
	
	// Test conversion
	converter := NewConverter()
	reormSchemas, err := converter.Convert(schema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	// Should have 4 model schemas (User, Profile, Post, Comment)
	if len(reormSchemas) != 4 {
		t.Errorf("expected 4 ReORM schemas, got %d", len(reormSchemas))
	}
	
	// Validate all schemas
	for name, s := range reormSchemas {
		if err := s.Validate(); err != nil {
			t.Errorf("schema %s validation failed: %v", name, err)
		}
	}
	
	// Test converter methods
	if provider := converter.GetDatabaseProvider(); provider != "postgresql" {
		t.Errorf("expected database provider 'postgresql', got '%s'", provider)
	}
	
	if url := converter.GetDatabaseURL(); url != "${DATABASE_URL}" {
		t.Errorf("expected database URL '${DATABASE_URL}', got '%s'", url)
	}
}

func TestReferentialActions(t *testing.T) {
	// Test all supported referential actions
	schema := `
model User {
  id    String @id @default(cuid())
  posts Post[]
}

model Post {
  id       String @id @default(cuid())
  authorId String
  author   User   @relation(fields: [authorId], references: [id], onDelete: Cascade, onUpdate: Cascade)
}

model Profile {
  id     String  @id @default(cuid())
  userId String? @unique
  user   User?   @relation(fields: [userId], references: [id], onDelete: SetNull, onUpdate: SetNull)
}

model Comment {
  id     String @id @default(cuid())
  postId String
  post   Post   @relation(fields: [postId], references: [id], onDelete: Restrict, onUpdate: Restrict)
}

model Like {
  id     String @id @default(cuid())
  postId String
  post   Post   @relation(fields: [postId], references: [id], onDelete: NoAction, onUpdate: NoAction)
}

model Tag {
  id     String @id @default(cuid())
  postId String
  post   Post   @relation(fields: [postId], references: [id], onDelete: SetDefault, onUpdate: SetDefault)
}`

	lexer := NewLexer(schema)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	// Test conversion
	converter := NewConverter()
	reormSchemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	// Should have 6 model schemas
	if len(reormSchemas) != 6 {
		t.Errorf("expected 6 ReORM schemas, got %d", len(reormSchemas))
	}
	
	// Check that all schemas are valid
	for name, s := range reormSchemas {
		if err := s.Validate(); err != nil {
			t.Errorf("schema %s validation failed: %v", name, err)
		}
	}
}

func TestScalarArrays(t *testing.T) {
	// Test scalar array support (PostgreSQL/CockroachDB feature)
	schema := `
model User {
  id             Int      @id @default(autoincrement())
  favoriteColors String[]
  coinflips      Boolean[]
  scores         Int[]
  weights        Float[]
  loginTimes     DateTime[]
}

enum Status {
  ACTIVE
  INACTIVE
  PENDING
}

model Post {
  id       Int      @id @default(autoincrement())
  statuses Status[]
}`

	lexer := NewLexer(schema)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	// Test conversion
	converter := NewConverter()
	reormSchemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	// Check User schema
	userSchema := reormSchemas["User"]
	if userSchema == nil {
		t.Fatal("User schema not found")
	}
	
	// Check array fields are converted correctly
	expectedArrayFields := map[string]string{
		"favoriteColors": "string[]",
		"coinflips":      "bool[]",
		"scores":         "int[]",
		"weights":        "float[]",
		"loginTimes":     "datetime[]",
	}
	
	for fieldName, expectedType := range expectedArrayFields {
		field, err := userSchema.GetField(fieldName)
		if err != nil {
			t.Errorf("field %s not found: %v", fieldName, err)
			continue
		}
		
		if string(field.Type) != expectedType {
			t.Errorf("field %s: expected type %s, got %s", fieldName, expectedType, field.Type)
		}
	}
	
	// Check Post schema with enum array
	postSchema := reormSchemas["Post"]
	if postSchema == nil {
		t.Fatal("Post schema not found")
	}
	
	statusesField, err := postSchema.GetField("statuses")
	if err != nil {
		t.Errorf("statuses field not found: %v", err)
	} else if string(statusesField.Type) != "string[]" {
		t.Errorf("statuses field: expected type string[], got %s", statusesField.Type)
	}
}

func TestCompositeKeys(t *testing.T) {
	// Test composite primary key support
	schema := `
model UserRole {
  userId Int
  roleId Int
  
  @@id([userId, roleId])
}

model PostTag {
  postId String
  tagId  String
  order  Int     @default(0)
  
  @@id([postId, tagId])
}

model RegularModel {
  id   Int    @id @default(autoincrement())
  name String
}`

	lexer := NewLexer(schema)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	// Test conversion
	converter := NewConverter()
	reormSchemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	// Check UserRole schema has composite key
	userRoleSchema := reormSchemas["UserRole"]
	if userRoleSchema == nil {
		t.Fatal("UserRole schema not found")
	}
	
	if len(userRoleSchema.CompositeKey) != 2 {
		t.Errorf("UserRole: expected 2 composite key fields, got %d", len(userRoleSchema.CompositeKey))
	}
	
	expectedCompositeKey := []string{"userId", "roleId"}
	for i, field := range expectedCompositeKey {
		if i >= len(userRoleSchema.CompositeKey) || userRoleSchema.CompositeKey[i] != field {
			t.Errorf("UserRole: expected composite key field %d to be %s, got %s", i, field, userRoleSchema.CompositeKey[i])
		}
	}
	
	// Check PostTag schema has composite key
	postTagSchema := reormSchemas["PostTag"]
	if postTagSchema == nil {
		t.Fatal("PostTag schema not found")
	}
	
	if len(postTagSchema.CompositeKey) != 2 {
		t.Errorf("PostTag: expected 2 composite key fields, got %d", len(postTagSchema.CompositeKey))
	}
	
	// Check RegularModel has single primary key (no composite key)
	regularSchema := reormSchemas["RegularModel"]
	if regularSchema == nil {
		t.Fatal("RegularModel schema not found")
	}
	
	if len(regularSchema.CompositeKey) != 0 {
		t.Errorf("RegularModel: expected no composite key, got %d fields", len(regularSchema.CompositeKey))
	}
	
	// Ensure fields marked as individual PK in UserRole don't exist
	for _, field := range userRoleSchema.Fields {
		if field.PrimaryKey {
			t.Errorf("UserRole field %s should not be marked as individual primary key when composite key exists", field.Name)
		}
	}
	
	// Validate all schemas
	for name, s := range reormSchemas {
		if err := s.Validate(); err != nil {
			t.Errorf("schema %s validation failed: %v", name, err)
		}
	}
}

func TestEnumMapping(t *testing.T) {
	// Test enum value mapping with @map
	schema := `
enum Status {
  ACTIVE   @map("active")
  INACTIVE @map("inactive")
  PENDING  @map("pending")
}`

	lexer := NewLexer(schema)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	// Check enum parsing
	if len(prismaSchema.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prismaSchema.Statements))
	}
	
	enumStmt, ok := prismaSchema.Statements[0].(*EnumStatement)
	if !ok {
		t.Fatalf("expected EnumStatement, got %T", prismaSchema.Statements[0])
	}
	
	if enumStmt.Name != "Status" {
		t.Errorf("expected enum name Status, got %s", enumStmt.Name)
	}
	
	if len(enumStmt.Values) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(enumStmt.Values))
	}
	
	// Check specific enum values and their mappings
	expectedMappings := map[string]string{
		"ACTIVE":   "active",
		"INACTIVE": "inactive", 
		"PENDING":  "pending",
	}
	
	for _, enumValue := range enumStmt.Values {
		expectedMapping, exists := expectedMappings[enumValue.Name]
		if !exists {
			t.Errorf("unexpected enum value: %s", enumValue.Name)
			continue
		}
		
		// Check that the enum value has a @map attribute
		found := false
		for _, attr := range enumValue.Attributes {
			if attr.Name == "map" && len(attr.Args) > 0 {
				if str, ok := attr.Args[0].(*StringLiteral); ok {
					if str.Value == expectedMapping {
						found = true
						break
					}
				}
			}
		}
		
		if !found {
			t.Errorf("enum value %s missing correct @map(%q) attribute", enumValue.Name, expectedMapping)
		}
	}
}

func TestDecimalSupport(t *testing.T) {
	// Test Decimal type support
	schema := `
model Product {
  id          Int     @id @default(autoincrement())
  price       Decimal
  discount    Decimal?
  priceList   Decimal[]
}`

	lexer := NewLexer(schema)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	// Debug: print statements
	t.Logf("Parsed statements: %d", len(prismaSchema.Statements))
	if len(prismaSchema.Statements) > 0 {
		if modelStmt, ok := prismaSchema.Statements[0].(*ModelStatement); ok {
			t.Logf("Model name: %s, Fields: %d", modelStmt.Name, len(modelStmt.Fields))
			for _, field := range modelStmt.Fields {
				t.Logf("  Field: %s %s", field.Name, field.Type.Name)
			}
		}
	}
	
	// Test conversion
	converter := NewConverter()
	reormSchemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	// Check Product schema
	productSchema := reormSchemas["Product"]
	if productSchema == nil {
		t.Fatal("Product schema not found")
	}
	
	// Debug: print all field names
	t.Logf("Product schema fields:")
	for _, field := range productSchema.Fields {
		t.Logf("  - %s (%s)", field.Name, field.Type)
	}
	
	// Check decimal fields
	priceField, err := productSchema.GetField("price")
	if err != nil {
		t.Errorf("price field not found: %v", err)
	} else if string(priceField.Type) != "decimal" {
		t.Errorf("price field: expected type decimal, got %s", priceField.Type)
	}
	
	discountField, err := productSchema.GetField("discount")
	if err != nil {
		t.Errorf("discount field not found: %v", err)
	} else {
		if string(discountField.Type) != "decimal" {
			t.Errorf("discount field: expected type decimal, got %s", discountField.Type)
		}
		if !discountField.Nullable {
			t.Errorf("discount field: expected nullable to be true")
		}
	}
	
	priceListField, err := productSchema.GetField("priceList")
	if err != nil {
		t.Errorf("priceList field not found: %v", err)
	} else if string(priceListField.Type) != "decimal[]" {
		t.Errorf("priceList field: expected type decimal[], got %s", priceListField.Type)
	}
	
	// Validate schema
	if err := productSchema.Validate(); err != nil {
		t.Errorf("Product schema validation failed: %v", err)
	}
}

func TestAdvancedPrismaFeatures(t *testing.T) {
	// Test comprehensive Prisma features including @db attributes and dbgenerated
	schema := `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id        String   @id @default(cuid())
  email     String   @unique @db.VarChar(255)
  salary    Decimal  @db.Money
  metadata  Json     @db.JsonB
  uuid      String   @db.Uuid
  createdAt DateTime @default(dbgenerated("NOW()")) @db.Timestamp(6)
  fullText  String   @db.Text
  bio       String?  @db.VarChar(1000)
}

model Product {
  id          Int     @id @default(autoincrement())
  name        String  @db.VarChar(200)
  price       Decimal @db.Decimal(10, 2)
  tags        String[]
  description String  @db.Text
  createdAt   DateTime @default(now())
}

enum Status {
  ACTIVE   @map("active")
  INACTIVE @map("inactive")
}`

	lexer := NewLexer(schema)
	parser := NewParser(lexer)
	
	prismaSchema := parser.ParseSchema()
	
	if len(parser.Errors()) > 0 {
		t.Fatalf("parser errors: %v", parser.Errors())
	}
	
	// Test conversion
	converter := NewConverter()
	reormSchemas, err := converter.Convert(prismaSchema)
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	
	// Check User schema with @db attributes
	userSchema := reormSchemas["User"]
	if userSchema == nil {
		t.Fatal("User schema not found")
	}
	
	// Debug: print all field names
	t.Logf("User schema fields:")
	for _, field := range userSchema.Fields {
		t.Logf("  - %s (%s) DbType: %s", field.Name, field.Type, field.DbType)
	}
	
	// Check specific @db attributes
	expectedDbTypes := map[string]string{
		"email":     "@db.VarChar",
		"metadata":  "@db.JsonB",
		"createdAt": "@db.Timestamp",
		"fullText":  "@db.Text",
		"bio":       "@db.VarChar",
	}
	
	for fieldName, expectedDbType := range expectedDbTypes {
		field, err := userSchema.GetField(fieldName)
		if err != nil {
			t.Errorf("field %s not found: %v", fieldName, err)
			continue
		}
		
		if !strings.Contains(field.DbType, expectedDbType) {
			t.Errorf("field %s: expected DbType to contain %s, got %s", fieldName, expectedDbType, field.DbType)
		}
	}
	
	// Check dbgenerated default value
	createdAtField, err := userSchema.GetField("createdAt")
	if err != nil {
		t.Errorf("createdAt field not found: %v", err)
	} else {
		if createdAtField.Default != "NOW()" {
			t.Errorf("createdAt field: expected default 'NOW()', got %v", createdAtField.Default)
		}
	}
	
	// Check Product schema with array field
	productSchema := reormSchemas["Product"]
	if productSchema == nil {
		t.Fatal("Product schema not found")
	}
	
	tagsField, err := productSchema.GetField("tags")
	if err != nil {
		t.Errorf("tags field not found: %v", err)
	} else if string(tagsField.Type) != "string[]" {
		t.Errorf("tags field: expected type string[], got %s", tagsField.Type)
	}
	
	// Validate all schemas
	for name, s := range reormSchemas {
		if err := s.Validate(); err != nil {
			t.Errorf("schema %s validation failed: %v", name, err)
		}
	}
}