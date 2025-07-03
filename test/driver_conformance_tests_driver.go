package test

import (
	"context"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Driver Initialization Tests =====

func (dct *DriverConformanceTests) TestNewDriver(t *testing.T) {
	if dct.shouldSkip("TestNewDriver") {
		t.Skip("Test skipped by driver")
	}

	db, err := dct.NewDriver(dct.Config)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	
	// Verify basic properties are initialized
	assert.Equal(t, strings.ToLower(dct.DriverName), db.GetDriverType())
}

func (dct *DriverConformanceTests) TestDriverConfig(t *testing.T) {
	if dct.shouldSkip("TestDriverConfig") {
		t.Skip("Test skipped by driver")
	}

	db, err := dct.NewDriver(dct.Config)
	require.NoError(t, err)
	
	// Driver should store config properly
	// Note: Implementation details may vary, but driver should be able to connect with stored config
	ctx := context.Background()
	err = db.Connect(ctx)
	assert.NoError(t, err)
	db.Close()
}

func (dct *DriverConformanceTests) TestFieldTypeMapping(t *testing.T) {
	if dct.shouldSkip("TestFieldTypeMapping") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Test that all field types can be mapped and used
	testSchema := schema.New("TypeTest").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "stringField", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "intField", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "int64Field", Type: schema.FieldTypeInt64}).
		AddField(schema.Field{Name: "floatField", Type: schema.FieldTypeFloat}).
		AddField(schema.Field{Name: "boolField", Type: schema.FieldTypeBool}).
		AddField(schema.Field{Name: "dateTimeField", Type: schema.FieldTypeDateTime}).
		AddField(schema.Field{Name: "jsonField", Type: schema.FieldTypeJSON}).
		AddField(schema.Field{Name: "decimalField", Type: schema.FieldTypeDecimal})

	err := td.DB.RegisterSchema("TypeTest", testSchema)
	require.NoError(t, err)

	ctx := context.Background()
	err = td.DB.CreateModel(ctx, "TypeTest")
	assert.NoError(t, err)

	// Verify table was created successfully by inserting data
	result, err := td.DB.Model("TypeTest").Insert(map[string]any{
		"id":            1,
		"stringField":   "test",
		"intField":      42,
		"int64Field":    int64(9999999999),
		"floatField":    3.14,
		"boolField":     true,
		"dateTimeField": "2024-01-01 00:00:00",
		"jsonField":     `{"key": "value"}`,
		"decimalField":  "123.45",
	}).Exec(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.RowsAffected)

	// Clean up
	err = td.DB.DropModel(ctx, "TypeTest")
	assert.NoError(t, err)
}

func (dct *DriverConformanceTests) TestGenerateColumnSQL(t *testing.T) {
	if dct.shouldSkip("TestGenerateColumnSQL") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// This test verifies that column SQL generation works correctly
	// by creating tables with various field configurations
	testCases := []struct {
		name   string
		field  schema.Field
		verify func(t *testing.T, td *TestDatabase)
	}{
		{
			name: "Simple string field",
			field: schema.Field{Name: "name", Type: schema.FieldTypeString},
			verify: func(t *testing.T, td *TestDatabase) {
				ctx := context.Background()
				schema := schema.New("Test1").
					AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
					AddField(schema.Field{Name: "name", Type: schema.FieldTypeString})
				
				err := td.DB.RegisterSchema("Test1", schema)
				require.NoError(t, err)
				err = td.DB.CreateModel(ctx, "Test1")
				assert.NoError(t, err)
				
				// Verify by inserting
				_, err = td.DB.Model("Test1").Insert(map[string]any{
					"id": 1,
					"name": "test",
				}).Exec(ctx)
				assert.NoError(t, err)
				
				td.DB.DropModel(ctx, "Test1")
			},
		},
		{
			name: "Primary key with auto increment",
			field: schema.Field{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true},
			verify: func(t *testing.T, td *TestDatabase) {
				ctx := context.Background()
				schema := schema.New("Test2").
					AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true}).
					AddField(schema.Field{Name: "value", Type: schema.FieldTypeString})
				
				err := td.DB.RegisterSchema("Test2", schema)
				require.NoError(t, err)
				err = td.DB.CreateModel(ctx, "Test2")
				assert.NoError(t, err)
				
				// Insert without ID
				result, err := td.DB.Model("Test2").Insert(map[string]any{
					"value": "test",
				}).Exec(ctx)
				assert.NoError(t, err)
				
				// Check if auto-increment worked (except PostgreSQL)
				if !dct.Characteristics.SupportsLastInsertID {
					assert.Equal(t, int64(0), result.LastInsertID) // PostgreSQL returns 0
				} else {
					assert.Greater(t, result.LastInsertID, int64(0))
				}
				
				td.DB.DropModel(ctx, "Test2")
			},
		},
		{
			name: "Nullable field",
			field: schema.Field{Name: "description", Type: schema.FieldTypeString, Nullable: true},
			verify: func(t *testing.T, td *TestDatabase) {
				ctx := context.Background()
				schema := schema.New("Test3").
					AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
					AddField(schema.Field{Name: "description", Type: schema.FieldTypeString, Nullable: true})
				
				err := td.DB.RegisterSchema("Test3", schema)
				require.NoError(t, err)
				err = td.DB.CreateModel(ctx, "Test3")
				assert.NoError(t, err)
				
				// Insert with NULL
				_, err = td.DB.Model("Test3").Insert(map[string]any{
					"id": 1,
					"description": nil,
				}).Exec(ctx)
				assert.NoError(t, err)
				
				td.DB.DropModel(ctx, "Test3")
			},
		},
		{
			name: "Unique field",
			field: schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true},
			verify: func(t *testing.T, td *TestDatabase) {
				ctx := context.Background()
				schema := schema.New("Test4").
					AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
					AddField(schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true})
				
				err := td.DB.RegisterSchema("Test4", schema)
				require.NoError(t, err)
				err = td.DB.CreateModel(ctx, "Test4")
				assert.NoError(t, err)
				
				// Insert first record
				_, err = td.DB.Model("Test4").Insert(map[string]any{
					"id": 1,
					"email": "test@example.com",
				}).Exec(ctx)
				assert.NoError(t, err)
				
				// Try to insert duplicate - should fail
				_, err = td.DB.Model("Test4").Insert(map[string]any{
					"id": 2,
					"email": "test@example.com",
				}).Exec(ctx)
				assert.Error(t, err)
				
				td.DB.DropModel(ctx, "Test4")
			},
		},
		{
			name: "Field with default",
			field: schema.Field{Name: "active", Type: schema.FieldTypeBool, Default: true},
			verify: func(t *testing.T, td *TestDatabase) {
				ctx := context.Background()
				schema := schema.New("Test5").
					AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
					AddField(schema.Field{Name: "active", Type: schema.FieldTypeBool, Default: true})
				
				err := td.DB.RegisterSchema("Test5", schema)
				require.NoError(t, err)
				err = td.DB.CreateModel(ctx, "Test5")
				assert.NoError(t, err)
				
				// Insert without specifying active
				_, err = td.DB.Model("Test5").Insert(map[string]any{
					"id": 1,
				}).Exec(ctx)
				assert.NoError(t, err)
				
				// Verify default was applied
				var result map[string]any
				err = td.DB.Model("Test5").Select().WhereCondition(
					td.DB.Model("Test5").Where("id").Equals(1),
				).FindFirst(ctx, &result)
				assert.NoError(t, err)
				// Use conversion utility to handle boolean - SQLite returns as integer
				assert.Equal(t, true, utils.ToBool(result["active"]))
				
				td.DB.DropModel(ctx, "Test5")
			},
		},
		{
			name: "Field with column mapping",
			field: schema.Field{Name: "firstName", Type: schema.FieldTypeString, Map: "first_name"},
			verify: func(t *testing.T, td *TestDatabase) {
				ctx := context.Background()
				schema := schema.New("Test6").
					AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
					AddField(schema.Field{Name: "firstName", Type: schema.FieldTypeString, Map: "first_name"})
				
				err := td.DB.RegisterSchema("Test6", schema)
				require.NoError(t, err)
				err = td.DB.CreateModel(ctx, "Test6")
				assert.NoError(t, err)
				
				// Insert using field name
				_, err = td.DB.Model("Test6").Insert(map[string]any{
					"id": 1,
					"firstName": "John",
				}).Exec(ctx)
				assert.NoError(t, err)
				
				// Query should return field name, not column name
				var result map[string]any
				err = td.DB.Model("Test6").Select().FindFirst(ctx, &result)
				assert.NoError(t, err)
				assert.Contains(t, result, "firstName")
				assert.Equal(t, "John", result["firstName"])
				
				td.DB.DropModel(ctx, "Test6")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.verify(t, td)
		})
	}
}