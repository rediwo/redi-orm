package generator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// MockMigrator for testing
type MockMigrator struct{}

func (m *MockMigrator) GetTables() ([]string, error) {
	return []string{"users", "posts", "system.indexes"}, nil
}

func (m *MockMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	switch tableName {
	case "users":
		return &types.TableInfo{
			Name: "users",
			Columns: []types.ColumnInfo{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
				{Name: "name", Type: "VARCHAR(255)", Nullable: false},
				{Name: "email", Type: "VARCHAR(255)", Nullable: false, Unique: true},
				{Name: "created_at", Type: "TIMESTAMP", Default: "CURRENT_TIMESTAMP"},
			},
			Indexes: []types.IndexInfo{
				{Name: "idx_users_email", Columns: []string{"email"}, Unique: true},
			},
		}, nil
	case "posts":
		return &types.TableInfo{
			Name: "posts",
			Columns: []types.ColumnInfo{
				{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
				{Name: "title", Type: "VARCHAR(255)", Nullable: false},
				{Name: "content", Type: "TEXT", Nullable: true},
				{Name: "user_id", Type: "INTEGER", Nullable: false},
				{Name: "created_at", Type: "TIMESTAMP", Default: "CURRENT_TIMESTAMP"},
			},
			ForeignKeys: []types.ForeignKeyInfo{
				{Name: "fk_posts_users", Column: "user_id", ReferencedTable: "users", ReferencedColumn: "id"},
			},
		}, nil
	default:
		return nil, nil
	}
}

func (m *MockMigrator) IsSystemTable(tableName string) bool {
	return tableName == "system.indexes"
}

func (m *MockMigrator) GenerateCreateTableSQL(schema any) (string, error) {
	return "", nil
}

func (m *MockMigrator) GenerateDropTableSQL(tableName string) string {
	return ""
}

func (m *MockMigrator) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return "", nil
}

func (m *MockMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	return nil, nil
}

func (m *MockMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	return nil, nil
}

func (m *MockMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	return ""
}

func (m *MockMigrator) GenerateDropIndexSQL(indexName string) string {
	return ""
}

func (m *MockMigrator) ApplyMigration(sql string) error {
	return nil
}

func (m *MockMigrator) GetDatabaseType() string {
	return "mock"
}

func (m *MockMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema any) (*types.MigrationPlan, error) {
	return nil, nil
}

func (m *MockMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	return nil, nil
}

// MockMigratorWrapper implements types.DatabaseMigrator by wrapping MockSpecificMigrator
type MockMigratorWrapper struct {
	*MockSpecificMigrator
}

func (w *MockMigratorWrapper) GenerateCreateTableSQL(schemaInterface any) (string, error) {
	s, ok := schemaInterface.(*schema.Schema)
	if !ok {
		return "", fmt.Errorf("expected *schema.Schema, got %T", schemaInterface)
	}
	return w.MockSpecificMigrator.GenerateCreateTableSQL(s)
}

func (w *MockMigratorWrapper) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return w.MockSpecificMigrator.GenerateAddColumnSQL(tableName, field)
}

func (w *MockMigratorWrapper) CompareSchema(existingTable *types.TableInfo, desiredSchema any) (*types.MigrationPlan, error) {
	return nil, nil
}

func (w *MockMigratorWrapper) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	return nil, nil
}

func (w *MockMigratorWrapper) GetSpecific() types.DatabaseSpecificMigrator {
	return w.MockSpecificMigrator
}

func TestGenerateSchemasFromTablesWithRelations(t *testing.T) {
	migrator := &MockMigratorWrapper{
		MockSpecificMigrator: &MockSpecificMigrator{},
	}

	schemas, err := GenerateSchemasFromTablesWithRelations(migrator)
	if err != nil {
		t.Fatalf("Failed to generate schemas: %v", err)
	}

	// Should have 2 schemas (users and posts, not system.indexes)
	if len(schemas) != 2 {
		t.Errorf("Expected 2 schemas, got %d", len(schemas))
	}

	// Find the schemas
	var userSchema, postSchema *schema.Schema
	for _, s := range schemas {
		switch s.Name {
		case "User":
			userSchema = s
		case "Post":
			postSchema = s
		}
	}

	if userSchema == nil {
		t.Fatal("User schema not found")
	}
	if postSchema == nil {
		t.Fatal("Post schema not found")
	}

	// Check User schema
	if userSchema.TableName != "users" {
		t.Errorf("Expected User table name to be 'users', got '%s'", userSchema.TableName)
	}
	if len(userSchema.Fields) != 4 {
		t.Errorf("Expected User to have 4 fields, got %d", len(userSchema.Fields))
	}

	// Check Post schema
	if postSchema.TableName != "posts" {
		t.Errorf("Expected Post table name to be 'posts', got '%s'", postSchema.TableName)
	}
	if len(postSchema.Fields) != 5 {
		t.Errorf("Expected Post to have 5 fields, got %d", len(postSchema.Fields))
	}

	// Debug: Print relations
	t.Logf("User relations: %+v", userSchema.Relations)
	t.Logf("Post relations: %+v", postSchema.Relations)

	// Check relations
	// Post should have a "user" relation (many-to-one)
	if userRelation, exists := postSchema.Relations["user"]; exists {
		if userRelation.Type != schema.RelationManyToOne {
			t.Errorf("Expected Post.user to be ManyToOne, got %s", userRelation.Type)
		}
		if userRelation.Model != "User" {
			t.Errorf("Expected Post.user to reference User, got %s", userRelation.Model)
		}
		if userRelation.ForeignKey != "userId" {
			t.Errorf("Expected Post.user foreign key to be userId, got %s", userRelation.ForeignKey)
		}
	} else {
		t.Error("Post schema missing 'user' relation")
	}

	// User should have a "posts" relation (one-to-many)
	if postsRelation, exists := userSchema.Relations["posts"]; exists {
		if postsRelation.Type != schema.RelationOneToMany {
			t.Errorf("Expected User.posts to be OneToMany, got %s", postsRelation.Type)
		}
		if postsRelation.Model != "Post" {
			t.Errorf("Expected User.posts to reference Post, got %s", postsRelation.Model)
		}
	} else {
		t.Error("User schema missing 'posts' relation")
	}
}

// MockSpecificMigrator for testing default value handling
type MockSpecificMigrator struct {
	MockMigrator
}

func (m *MockSpecificMigrator) GetTables() ([]string, error) {
	return m.MockMigrator.GetTables()
}

func (m *MockSpecificMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	return m.MockMigrator.GetTableInfo(tableName)
}

func (m *MockSpecificMigrator) GenerateCreateTableSQL(s *schema.Schema) (string, error) {
	return "", nil
}

func (m *MockSpecificMigrator) GenerateDropTableSQL(tableName string) string {
	return ""
}

func (m *MockSpecificMigrator) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return "", nil
}

func (m *MockSpecificMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	return nil, nil
}

func (m *MockSpecificMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	return nil, nil
}

func (m *MockSpecificMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	return ""
}

func (m *MockSpecificMigrator) GenerateDropIndexSQL(indexName string) string {
	return ""
}

func (m *MockSpecificMigrator) ApplyMigration(sql string) error {
	return nil
}

func (m *MockSpecificMigrator) GetDatabaseType() string {
	return "mock"
}

func (m *MockSpecificMigrator) MapFieldType(field schema.Field) string {
	return "VARCHAR"
}

func (m *MockSpecificMigrator) FormatDefaultValue(value any) string {
	return fmt.Sprintf("%v", value)
}

func (m *MockSpecificMigrator) GenerateColumnDefinitionFromColumnInfo(col types.ColumnInfo) string {
	return ""
}

func (m *MockSpecificMigrator) ConvertFieldToColumnInfo(field schema.Field) *types.ColumnInfo {
	return nil
}

func (m *MockSpecificMigrator) IsSystemTable(tableName string) bool {
	return m.MockMigrator.IsSystemTable(tableName)
}

func (m *MockSpecificMigrator) ParseDefaultValue(value any, fieldType schema.FieldType) any {
	// Simulate MySQL behavior
	if str, ok := value.(string); ok {
		upperStr := strings.ToUpper(str)
		if upperStr == "CURRENT_TIMESTAMP" || upperStr == "NOW()" {
			return "CURRENT_TIMESTAMP"
		}
	}
	return value
}

func (m *MockSpecificMigrator) NormalizeDefaultToPrismaFunction(value any, fieldType schema.FieldType) (string, bool) {
	if str, ok := value.(string); ok {
		if str == "CURRENT_TIMESTAMP" {
			return "now", true
		}
	}
	return "", false
}

func (m *MockSpecificMigrator) IsSystemIndex(indexName string) bool {
	return false
}

func (m *MockSpecificMigrator) MapDatabaseTypeToFieldType(dbType string) schema.FieldType {
	// Simple mapping for tests
	switch strings.ToLower(dbType) {
	case "varchar", "text":
		return schema.FieldTypeString
	case "integer", "int":
		return schema.FieldTypeInt
	case "timestamp", "datetime":
		return schema.FieldTypeDateTime
	default:
		return schema.FieldTypeString
	}
}

func (m *MockSpecificMigrator) IsPrimaryKeyIndex(indexName string) bool {
	return strings.Contains(strings.ToLower(indexName), "primary")
}

func TestGenerateSchemaFromTable(t *testing.T) {
	migrator := &MockSpecificMigrator{}

	tableInfo := &types.TableInfo{
		Name: "users",
		Columns: []types.ColumnInfo{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "created_at", Type: "TIMESTAMP", Default: "CURRENT_TIMESTAMP"},
		},
	}

	generatedSchema, err := GenerateSchemaFromTable(tableInfo, migrator)
	if err != nil {
		t.Fatalf("Failed to generate schema: %v", err)
	}

	// Check the created_at field has normalized default
	var createdAtField *schema.Field
	for i := range generatedSchema.Fields {
		if generatedSchema.Fields[i].Name == "createdAt" {
			createdAtField = &generatedSchema.Fields[i]
			break
		}
	}

	if createdAtField == nil {
		t.Fatal("created_at field not found")
	}

	// The default should be normalized to "CURRENT_TIMESTAMP" by the migrator
	if createdAtField.Default != "CURRENT_TIMESTAMP" {
		t.Errorf("Expected created_at default to be normalized to CURRENT_TIMESTAMP, got %v", createdAtField.Default)
	}
}

func TestGeneratePrismaSchemaWithNowDefault(t *testing.T) {
	migrator := &MockSpecificMigrator{}
	generator := NewSchemaGenerator(migrator)

	// Create schema with now() default
	testSchema := schema.New("Comment").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name: "content",
			Type: schema.FieldTypeString,
		}).
		AddField(schema.Field{
			Name:    "createdAt",
			Type:    schema.FieldTypeDateTime,
			Default: "now()",
		}).
		AddField(schema.Field{
			Name:    "updatedAt",
			Type:    schema.FieldTypeDateTime,
			Default: "now()",
		})

	// Generate Prisma schema
	prismaOutput, err := generator.GeneratePrismaSchema(testSchema)
	if err != nil {
		t.Fatalf("Failed to generate Prisma schema: %v", err)
	}

	// Check that now() is generated without quotes
	if strings.Contains(prismaOutput, `@default("now()")`) {
		t.Errorf("Generated @default(\"now()\") with quotes, expected @default(now()) without quotes")
		t.Logf("Generated schema:\n%s", prismaOutput)
	}

	if !strings.Contains(prismaOutput, `@default(now())`) {
		t.Errorf("Expected @default(now()) not found in generated schema")
		t.Logf("Generated schema:\n%s", prismaOutput)
	}
}
