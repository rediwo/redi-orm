package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Transaction Tests =====

func (dct *DriverConformanceTests) TestBeginCommit(t *testing.T) {
	if dct.shouldSkip("TestBeginCommit") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Begin transaction
	tx, err := td.DB.Begin(ctx)
	assert.NoError(t, err)

	// Insert data in transaction
	UserTx := tx.Model("User")
	result, err := UserTx.Insert(map[string]any{
		"name":   "TxUser",
		"email":  "tx@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)
	// PostgreSQL doesn't support LastInsertId() - it always returns 0
	if td.DB.GetDriverType() != "postgresql" {
		assert.Greater(t, result.LastInsertID, int64(0))
	}

	// Commit transaction
	err = tx.Commit(ctx)
	assert.NoError(t, err)

	// Verify data persisted
	td.AssertCount("User", 1)
	User := td.DB.Model("User")
	td.AssertExists("User", User.Where("email").Equals("tx@example.com"))
}

func (dct *DriverConformanceTests) TestBeginRollback(t *testing.T) {
	if dct.shouldSkip("TestBeginRollback") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Begin transaction
	tx, err := td.DB.Begin(ctx)
	assert.NoError(t, err)

	// Insert data in transaction
	UserTx := tx.Model("User")
	_, err = UserTx.Insert(map[string]any{
		"name":   "TxUser",
		"email":  "tx@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback(ctx)
	assert.NoError(t, err)

	// Verify data not persisted
	td.AssertCount("User", 0)
	User := td.DB.Model("User")
	td.AssertNotExists("User", User.Where("email").Equals("tx@example.com"))
}

func (dct *DriverConformanceTests) TestTransactionFunction(t *testing.T) {
	if dct.shouldSkip("TestTransactionFunction") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test successful transaction
	err = td.DB.Transaction(ctx, func(tx types.Transaction) error {
		// Insert user
		UserTx := tx.Model("User")
		result, err := UserTx.Insert(map[string]any{
			"name":   "TxFunc",
			"email":  "txfunc@example.com",
			"active": true,
		}).Exec(ctx)
		if err != nil {
			return err
		}

		// Insert post for the user
		PostTx := tx.Model("Post")
		var userId int64
		if td.DB.GetDriverType() == "postgresql" {
			// For PostgreSQL, we need to find the user ID since LastInsertId() returns 0
			// In a real application, you'd use RETURNING clause
			userId = 1 // Assume first user has ID 1 for this test
		} else {
			userId = result.LastInsertID
		}
		_, err = PostTx.Insert(map[string]any{
			"title":     "Transaction Post",
			"content":   "Created in transaction",
			"userId":    userId,
			"published": true,
		}).Exec(ctx)
		return err
	})
	assert.NoError(t, err)

	// Verify both inserts succeeded
	td.AssertCount("User", 1)
	td.AssertCount("Post", 1)

	// Test failed transaction
	err = td.DB.Transaction(ctx, func(tx types.Transaction) error {
		// Insert another user
		UserTx := tx.Model("User")
		_, err := UserTx.Insert(map[string]any{
			"name":   "FailUser",
			"email":  "fail@example.com",
			"active": true,
		}).Exec(ctx)
		if err != nil {
			return err
		}

		// Return error to rollback
		return fmt.Errorf("simulated error")
	})
	assert.Error(t, err)

	// Verify rollback occurred
	td.AssertCount("User", 1) // Only the first user should exist
}

func (dct *DriverConformanceTests) TestTransactionIsolation(t *testing.T) {
	if dct.shouldSkip("TestTransactionIsolation") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Insert initial data
	User := td.DB.Model("User")
	_, err = User.Insert(map[string]any{
		"name":   "IsolationTest",
		"email":  "isolation@example.com",
		"age":    25,
		"active": true,
	}).Exec(ctx)
	require.NoError(t, err)

	// Start two transactions
	tx1, err := td.DB.Begin(ctx)
	require.NoError(t, err)
	defer tx1.Rollback(ctx)

	tx2, err := td.DB.Begin(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	// Update in tx1
	UserTx1 := tx1.Model("User")
	_, err = UserTx1.Update(map[string]any{
		"age": 30,
	}).WhereCondition(
		UserTx1.Where("email").Equals("isolation@example.com"),
	).Exec(ctx)
	assert.NoError(t, err)

	// Read in tx2 (should see old value due to isolation)
	UserTx2 := tx2.Model("User")
	var user TestUser
	err = UserTx2.Select().
		WhereCondition(UserTx2.Where("email").Equals("isolation@example.com")).
		FindFirst(ctx, &user)
	assert.NoError(t, err)
	if assert.NotNil(t, user.Age, "User age should not be nil") {
		assert.Equal(t, 25, *user.Age) // Should still see original value
	}

	// Commit tx1
	err = tx1.Commit(ctx)
	assert.NoError(t, err)

	// Commit tx2
	err = tx2.Commit(ctx)
	assert.NoError(t, err)
}

func (dct *DriverConformanceTests) TestSavepoints(t *testing.T) {
	if dct.shouldSkip("TestSavepoints") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Begin transaction
	tx, err := td.DB.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Insert first user
	UserTx := tx.Model("User")
	_, err = UserTx.Insert(map[string]any{
		"name":   "User1",
		"email":  "user1@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Create savepoint
	err = tx.Savepoint(ctx, "sp1")
	assert.NoError(t, err)

	// Insert second user
	_, err = UserTx.Insert(map[string]any{
		"name":   "User2",
		"email":  "user2@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Rollback to savepoint
	err = tx.RollbackTo(ctx, "sp1")
	assert.NoError(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	assert.NoError(t, err)

	// Only first user should exist
	td.AssertCount("User", 1)
	User := td.DB.Model("User")
	td.AssertExists("User", User.Where("email").Equals("user1@example.com"))
	td.AssertNotExists("User", User.Where("email").Equals("user2@example.com"))
}

// ===== Field Mapping Tests =====

func (dct *DriverConformanceTests) TestFieldNameMapping(t *testing.T) {
	if dct.shouldSkip("TestFieldNameMapping") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Create schema with camelCase fields
	userSchema := schema.New("TestUser").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "firstName", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "lastName", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "emailAddress", Type: schema.FieldTypeString})

	err := td.DB.RegisterSchema("TestUser", userSchema)
	require.NoError(t, err)

	ctx := context.Background()
	err = td.DB.CreateModel(ctx, "TestUser")
	require.NoError(t, err)

	// Test field name resolution
	firstName, err := td.DB.ResolveFieldName("TestUser", "firstName")
	assert.NoError(t, err)
	assert.Equal(t, "first_name", firstName)

	// Insert using camelCase field names
	TestUser := td.DB.Model("TestUser")
	_, err = TestUser.Insert(map[string]any{
		"firstName":    "John",
		"lastName":     "Doe",
		"emailAddress": "john@example.com",
	}).Exec(ctx)
	assert.NoError(t, err)

	// Query should work with camelCase fields
	var results []map[string]any
	err = TestUser.Select("firstName", "lastName").FindMany(ctx, &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	// Results MUST have field names, not column names
	assert.Contains(t, results[0], "firstName", "Results should contain field name 'firstName'")
	assert.Contains(t, results[0], "lastName", "Results should contain field name 'lastName'")
	assert.Equal(t, "John", results[0]["firstName"])
	assert.Equal(t, "Doe", results[0]["lastName"])
	// Ensure column names are NOT returned
	assert.NotContains(t, results[0], "first_name", "Results should NOT contain column name 'first_name'")
	assert.NotContains(t, results[0], "last_name", "Results should NOT contain column name 'last_name'")
}

func (dct *DriverConformanceTests) TestTableNameMapping(t *testing.T) {
	if dct.shouldSkip("TestTableNameMapping") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Register standard schemas first
	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	// Test model to table name mapping
	tableName, err := td.DB.ResolveTableName("User")
	assert.NoError(t, err)
	assert.Equal(t, "users", tableName)

	// Test with custom table name
	postSchema := schema.New("BlogPost").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true})
	postSchema.TableName = "blog_posts"

	err = td.DB.RegisterSchema("BlogPost", postSchema)
	require.NoError(t, err)

	tableName, err = td.DB.ResolveTableName("BlogPost")
	assert.NoError(t, err)
	assert.Equal(t, "blog_posts", tableName)
}

func (dct *DriverConformanceTests) TestMapAnnotations(t *testing.T) {
	if dct.shouldSkip("TestMapAnnotations") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Create schema with @map annotations
	userSchema := schema.New("MappedUser").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "fullName", Type: schema.FieldTypeString, Map: "full_name"}).
		AddField(schema.Field{Name: "primaryEmail", Type: schema.FieldTypeString, Map: "email"})

	err := td.DB.RegisterSchema("MappedUser", userSchema)
	require.NoError(t, err)

	ctx := context.Background()
	err = td.DB.CreateModel(ctx, "MappedUser")
	require.NoError(t, err)

	// Verify mapping
	columnName, err := td.DB.ResolveFieldName("MappedUser", "primaryEmail")
	assert.NoError(t, err)
	assert.Equal(t, "email", columnName)

	// Insert using field names
	MappedUser := td.DB.Model("MappedUser")
	_, err = MappedUser.Insert(map[string]any{
		"fullName":     "Test User",
		"primaryEmail": "test@example.com",
	}).Exec(ctx)
	assert.NoError(t, err)
}

// ===== Data Type Tests =====

func (dct *DriverConformanceTests) TestIntegerTypes(t *testing.T) {
	if dct.shouldSkip("TestIntegerTypes") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Create schema with various integer types
	intSchema := schema.New("IntegerTest").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "smallInt", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "regularInt", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "bigInt", Type: schema.FieldTypeInt64})

	err := td.DB.RegisterSchema("IntegerTest", intSchema)
	require.NoError(t, err)

	ctx := context.Background()
	err = td.DB.CreateModel(ctx, "IntegerTest")
	require.NoError(t, err)

	// Test various integer values
	testCases := []struct {
		name   string
		values map[string]any
	}{
		{"zero", map[string]any{"id": 1, "smallInt": 0, "regularInt": 0, "bigInt": int64(0)}},
		{"positive", map[string]any{"id": 2, "smallInt": 32767, "regularInt": 2147483647, "bigInt": int64(9223372036854775807)}},
		{"negative", map[string]any{"id": 3, "smallInt": -32768, "regularInt": -2147483648, "bigInt": int64(-9223372036854775808)}},
	}

	IntegerTest := td.DB.Model("IntegerTest")
	for _, tc := range testCases {
		_, err := IntegerTest.Insert(tc.values).Exec(ctx)
		assert.NoError(t, err, "Failed to insert %s values", tc.name)
	}

	// Verify all inserted
	count, err := IntegerTest.Select().Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func (dct *DriverConformanceTests) TestFloatTypes(t *testing.T) {
	if dct.shouldSkip("TestFloatTypes") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Create schema with float types
	floatSchema := schema.New("FloatTest").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "floatValue", Type: schema.FieldTypeFloat}).
		AddField(schema.Field{Name: "decimalValue", Type: schema.FieldTypeDecimal})

	err := td.DB.RegisterSchema("FloatTest", floatSchema)
	require.NoError(t, err)

	ctx := context.Background()
	err = td.DB.CreateModel(ctx, "FloatTest")
	require.NoError(t, err)

	// Insert float values
	FloatTest := td.DB.Model("FloatTest")
	_, err = FloatTest.Insert(map[string]any{
		"id":           1,
		"floatValue":   3.14159,
		"decimalValue": "123.45",
	}).Exec(ctx)
	assert.NoError(t, err)
}

func (dct *DriverConformanceTests) TestStringTypes(t *testing.T) {
	if dct.shouldSkip("TestStringTypes") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Create schema with string types
	stringSchema := schema.New("StringTest").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "shortText", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "longText", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "emptyText", Type: schema.FieldTypeString})

	err := td.DB.RegisterSchema("StringTest", stringSchema)
	require.NoError(t, err)

	ctx := context.Background()
	err = td.DB.CreateModel(ctx, "StringTest")
	require.NoError(t, err)

	// Test various string values
	// Create a string that's close to but under 255 characters
	longString := ""
	for range 20 {
		longString += "Lorem ipsum "
	}
	// longString is now about 240 characters

	StringTest := td.DB.Model("StringTest")
	_, err = StringTest.Insert(map[string]any{
		"id":        1,
		"shortText": "Hello",
		"longText":  longString,
		"emptyText": "",
	}).Exec(ctx)
	assert.NoError(t, err)

	// Test special characters
	_, err = StringTest.Insert(map[string]any{
		"id":        2,
		"shortText": "Special: 'quotes' and \"double\" and \\ backslash",
		"longText":  "Unicode: 你好世界 🌍",
		"emptyText": " ",
	}).Exec(ctx)
	assert.NoError(t, err)
}

func (dct *DriverConformanceTests) TestBooleanTypes(t *testing.T) {
	if dct.shouldSkip("TestBooleanTypes") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Insert boolean values
	User := td.DB.Model("User")
	_, err = User.Insert(map[string]any{
		"name":   "BoolTest",
		"email":  "bool@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	_, err = User.Insert(map[string]any{
		"name":   "BoolTest2",
		"email":  "bool2@example.com",
		"active": false,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Query boolean values
	var activeUsers []TestUser
	err = User.Select().
		WhereCondition(User.Where("active").Equals(true)).
		FindMany(ctx, &activeUsers)
	assert.NoError(t, err)
	assert.Len(t, activeUsers, 1)

	var inactiveUsers []TestUser
	err = User.Select().
		WhereCondition(User.Where("active").Equals(false)).
		FindMany(ctx, &inactiveUsers)
	assert.NoError(t, err)
	assert.Len(t, inactiveUsers, 1)
}

func (dct *DriverConformanceTests) TestDateTimeTypes(t *testing.T) {
	if dct.shouldSkip("TestDateTimeTypes") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Insert with current time in UTC
	User := td.DB.Model("User")
	now := time.Now().UTC()
	_, err = User.Insert(map[string]any{
		"name":      "TimeTest",
		"email":     "time@example.com",
		"createdAt": now,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Query and verify time
	var user TestUser
	err = User.Select().
		WhereCondition(User.Where("email").Equals("time@example.com")).
		FindFirst(ctx, &user)
	assert.NoError(t, err)

	// Time should be close (within 1 second due to DB precision)
	// Compare in UTC to handle timezone differences
	assert.WithinDuration(t, now.UTC(), user.CreatedAt.UTC(), time.Second)

	// Test default now()
	_, err = User.Insert(map[string]any{
		"name":  "DefaultTime",
		"email": "defaulttime@example.com",
		// createdAt should default to now()
	}).Exec(ctx)
	assert.NoError(t, err)

	// Verify default was applied
	user = TestUser{}
	err = User.Select().
		WhereCondition(User.Where("email").Equals("defaulttime@example.com")).
		FindFirst(ctx, &user)
	assert.NoError(t, err)
	assert.False(t, user.CreatedAt.IsZero())
}

func (dct *DriverConformanceTests) TestNullValues(t *testing.T) {
	if dct.shouldSkip("TestNullValues") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Insert with null values
	User := td.DB.Model("User")
	_, err = User.Insert(map[string]any{
		"name":  "NullTest",
		"email": "null@example.com",
		"age":   nil, // Nullable field
	}).Exec(ctx)
	assert.NoError(t, err)

	// Query and verify null
	var user TestUser
	err = User.Select().
		WhereCondition(User.Where("email").Equals("null@example.com")).
		FindFirst(ctx, &user)
	assert.NoError(t, err)
	assert.Nil(t, user.Age)

	// Test IS NULL query
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("age").IsNull()).
		FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
}

func (dct *DriverConformanceTests) TestDefaultValues(t *testing.T) {
	if dct.shouldSkip("TestDefaultValues") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Insert a user first for foreign key
	User := td.DB.Model("User")
	result, err := User.Insert(map[string]any{
		"name":  "Test User",
		"email": "test@example.com",
	}).Exec(ctx)
	require.NoError(t, err)
	var userID int64
	if td.DB.GetDriverType() == "postgresql" {
		// For PostgreSQL, we need to find the user ID since LastInsertId() returns 0
		userID = 1 // Assume first user has ID 1 for this test
	} else {
		userID = result.LastInsertID
	}

	// Insert without providing default fields
	Post := td.DB.Model("Post")
	_, err = Post.Insert(map[string]any{
		"title":  "Default Test",
		"userId": userID,
		// published should default to false
		// views should default to 0
		// createdAt should default to now()
	}).Exec(ctx)
	assert.NoError(t, err)

	// Verify defaults were applied
	var post TestPost
	err = Post.Select().
		WhereCondition(Post.Where("title").Equals("Default Test")).
		FindFirst(ctx, &post)
	assert.NoError(t, err)
	assert.False(t, post.Published)
	assert.Equal(t, 0, post.Views)
	assert.False(t, post.CreatedAt.IsZero())
}

// ===== Error Handling Tests =====

func (dct *DriverConformanceTests) TestInvalidQuery(t *testing.T) {
	if dct.shouldSkip("TestInvalidQuery") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test invalid SQL
	var results []map[string]any
	err = td.DB.Raw("SELECT * FROM non_existent_table").Find(ctx, &results)
	assert.Error(t, err)
}

func (dct *DriverConformanceTests) TestUniqueConstraintViolation(t *testing.T) {
	if dct.shouldSkip("TestUniqueConstraintViolation") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Insert first user
	User := td.DB.Model("User")
	_, err = User.Insert(map[string]any{
		"name":  "UniqueTest",
		"email": "unique@example.com",
	}).Exec(ctx)
	assert.NoError(t, err)

	// Try to insert with same email (unique constraint)
	_, err = User.Insert(map[string]any{
		"name":  "UniqueTest2",
		"email": "unique@example.com",
	}).Exec(ctx)
	assert.Error(t, err)
}

func (dct *DriverConformanceTests) TestNotNullConstraintViolation(t *testing.T) {
	if dct.shouldSkip("TestNotNullConstraintViolation") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Try to insert without required field
	User := td.DB.Model("User")
	_, err = User.Insert(map[string]any{
		"email": "noname@example.com",
		// name is required (not null)
	}).Exec(ctx)
	assert.Error(t, err)
}

func (dct *DriverConformanceTests) TestInvalidFieldName(t *testing.T) {
	if dct.shouldSkip("TestInvalidFieldName") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test query with invalid field
	User := td.DB.Model("User")
	var users []TestUser
	err = User.Select().
		WhereCondition(User.Where("invalidField").Equals("value")).
		FindMany(ctx, &users)
	assert.Error(t, err)
}

func (dct *DriverConformanceTests) TestInvalidModelName(t *testing.T) {
	if dct.shouldSkip("TestInvalidModelName") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	ctx := context.Background()

	// Test with unregistered model
	var results []map[string]any
	err := td.DB.Model("NonExistentModel").Select().FindMany(ctx, &results)
	assert.Error(t, err)
}

// ===== Raw Query Tests =====

func (dct *DriverConformanceTests) TestRawSelect(t *testing.T) {
	if dct.shouldSkip("TestRawSelect") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test raw SELECT
	var users []map[string]any
	err = td.DB.Raw("SELECT * FROM users WHERE active = ?", true).Find(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 4)

	// Test raw SELECT ONE
	var user map[string]any
	err = td.DB.Raw("SELECT * FROM users WHERE email = ?", "alice@example.com").FindOne(ctx, &user)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", user["name"])
}

func (dct *DriverConformanceTests) TestRawInsert(t *testing.T) {
	if dct.shouldSkip("TestRawInsert") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test raw INSERT
	result, err := td.DB.Raw("INSERT INTO users (name, email, active) VALUES (?, ?, ?)",
		"RawUser", "raw@example.com", true).Exec(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result.RowsAffected)
}

func (dct *DriverConformanceTests) TestRawUpdate(t *testing.T) {
	if dct.shouldSkip("TestRawUpdate") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test raw UPDATE
	result, err := td.DB.Raw("UPDATE users SET active = ? WHERE age > ?", false, 30).Exec(ctx)
	assert.NoError(t, err)

	// Check expected rows affected based on driver characteristics
	if dct.Characteristics.ReturnsZeroRowsAffectedForUnchanged {
		// MySQL doesn't count rows where values didn't actually change
		// Charlie (age 35) is already inactive, so RowsAffected might be 0
		// But we should at least verify the query succeeded
		assert.NoError(t, err)
	} else {
		// Other databases count all matched rows
		assert.Greater(t, result.RowsAffected, int64(0))
	}
}

func (dct *DriverConformanceTests) TestRawDelete(t *testing.T) {
	if dct.shouldSkip("TestRawDelete") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test raw DELETE
	result, err := td.DB.Raw("DELETE FROM comments WHERE id > ?", 2).Exec(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result.RowsAffected)
}

func (dct *DriverConformanceTests) TestRawWithParameters(t *testing.T) {
	if dct.shouldSkip("TestRawWithParameters") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test with multiple parameters
	var results []map[string]any
	err = td.DB.Raw("SELECT * FROM posts WHERE user_id = ? AND published = ? ORDER BY id",
		1, true).Find(ctx, &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	// Test with no parameters
	err = td.DB.Raw("SELECT COUNT(*) as count FROM users").Find(ctx, &results)
	assert.NoError(t, err)
}

// Extended Raw Query Tests

func (dct *DriverConformanceTests) TestRawQueryErrorHandling(t *testing.T) {
	if dct.shouldSkip("TestRawQueryErrorHandling") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	ctx := context.Background()

	// Test invalid SQL syntax
	var results []map[string]any
	err := td.DB.Raw("INVALID SQL SYNTAX").Find(ctx, &results)
	assert.Error(t, err)

	// Test non-existent table
	err = td.DB.Raw("SELECT * FROM non_existent_table").Find(ctx, &results)
	assert.Error(t, err)

	// Test wrong number of parameters
	err = td.DB.Raw("SELECT * FROM users WHERE id = ? AND name = ?", 1).Find(ctx, &results)
	assert.Error(t, err)
}

func (dct *DriverConformanceTests) TestRawQueryWithDifferentDataTypes(t *testing.T) {
	if dct.shouldSkip("TestRawQueryWithDifferentDataTypes") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Create a table with various data types
	dataTypeSchema := schema.New("DataTypeTest").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "intVal", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "floatVal", Type: schema.FieldTypeFloat}).
		AddField(schema.Field{Name: "boolVal", Type: schema.FieldTypeBool}).
		AddField(schema.Field{Name: "stringVal", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "nullableString", Type: schema.FieldTypeString, Nullable: true}).
		AddField(schema.Field{Name: "jsonVal", Type: schema.FieldTypeJSON}).
		AddField(schema.Field{Name: "dateTimeVal", Type: schema.FieldTypeDateTime})

	err := td.DB.RegisterSchema("DataTypeTest", dataTypeSchema)
	require.NoError(t, err)

	ctx := context.Background()
	err = td.DB.CreateModel(ctx, "DataTypeTest")
	require.NoError(t, err)

	// Insert test data with raw query
	now := time.Now().UTC()
	_, err = td.DB.Raw(
		"INSERT INTO data_type_tests (id, int_val, float_val, bool_val, string_val, nullable_string, json_val, date_time_val) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		1, 42, 3.14, true, "test", nil, `{"key": "value"}`, now,
	).Exec(ctx)
	assert.NoError(t, err)

	// Query with different data types
	var result map[string]any
	err = td.DB.Raw("SELECT * FROM data_type_tests WHERE id = ?", 1).FindOne(ctx, &result)
	assert.NoError(t, err)

	// Use conversion utilities to handle different driver representations
	// Raw queries return column names as-is (snake_case)
	assert.Equal(t, int64(42), utils.ToInt64(result["int_val"]))
	assert.InDelta(t, 3.14, utils.ToFloat64(result["float_val"]), 0.001)
	assert.Equal(t, true, utils.ToBool(result["bool_val"]))
	assert.Equal(t, "test", result["string_val"])
	assert.Nil(t, result["nullable_string"])
}

func (dct *DriverConformanceTests) TestRawQueryComplexQueries(t *testing.T) {
	if dct.shouldSkip("TestRawQueryComplexQueries") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test JOIN query
	var results []map[string]any
	joinQuery := `
		SELECT u.name as user_name, p.title as post_title 
		FROM posts p 
		INNER JOIN users u ON p.user_id = u.id 
		WHERE p.published = ?
		ORDER BY p.id
	`
	err = td.DB.Raw(joinQuery, true).Find(ctx, &results)
	assert.NoError(t, err)
	assert.Greater(t, len(results), 0)
	assert.Contains(t, results[0], "user_name")
	assert.Contains(t, results[0], "post_title")

	// Test aggregate functions with GROUP BY
	aggregateQuery := `
		SELECT u.name, COUNT(p.id) as post_count 
		FROM users u 
		LEFT JOIN posts p ON u.id = p.user_id 
		GROUP BY u.id, u.name 
		HAVING COUNT(p.id) > ?
		ORDER BY u.name
	`
	err = td.DB.Raw(aggregateQuery, 0).Find(ctx, &results)
	assert.NoError(t, err)
	assert.Greater(t, len(results), 0)

	// Test subquery
	subQuery := `
		SELECT * FROM users 
		WHERE id IN (
			SELECT DISTINCT user_id FROM posts WHERE published = ?
		)
		ORDER BY id
	`
	err = td.DB.Raw(subQuery, true).Find(ctx, &results)
	assert.NoError(t, err)
	assert.Greater(t, len(results), 0)
}

func (dct *DriverConformanceTests) TestRawQueryWithFind(t *testing.T) {
	if dct.shouldSkip("TestRawQueryWithFind") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test Find with struct slice
	var users []TestUser
	err = td.DB.Raw("SELECT * FROM users WHERE active = ? ORDER BY id", true).Find(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 4)
	assert.NotEmpty(t, users[0].Name)

	// Test Find with map slice
	var mapResults []map[string]any
	// Use driver-specific LIMIT syntax
	limitQuery := "SELECT id, name FROM users ORDER BY id"
	if dct.DriverName == "SQLite" || dct.DriverName == "PostgreSQL" {
		limitQuery += " LIMIT 2"
	} else { // MySQL
		limitQuery += " LIMIT 2"
	}
	err = td.DB.Raw(limitQuery).Find(ctx, &mapResults)
	assert.NoError(t, err)
	assert.Len(t, mapResults, 2)

	// Test Find with empty slice
	var emptyResults []TestUser
	err = td.DB.Raw("SELECT * FROM users WHERE id = ?", 999).Find(ctx, &emptyResults)
	assert.NoError(t, err)
	assert.Len(t, emptyResults, 0)
}

func (dct *DriverConformanceTests) TestRawQueryWithFindOne(t *testing.T) {
	if dct.shouldSkip("TestRawQueryWithFindOne") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	err = td.InsertStandardTestData()
	require.NoError(t, err)

	ctx := context.Background()

	// Test FindOne with struct
	var user TestUser
	err = td.DB.Raw("SELECT * FROM users WHERE email = ?", "alice@example.com").FindOne(ctx, &user)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", user.Name)

	// Test FindOne with map
	var result map[string]any
	err = td.DB.Raw("SELECT COUNT(*) as count FROM users").FindOne(ctx, &result)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), result["count"])

	// Test FindOne with single value
	var count int64
	err = td.DB.Raw("SELECT COUNT(*) FROM users").FindOne(ctx, &count)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func (dct *DriverConformanceTests) TestRawQueryNoRowsFound(t *testing.T) {
	if dct.shouldSkip("TestRawQueryNoRowsFound") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test FindOne with no results
	var user TestUser
	err = td.DB.Raw("SELECT * FROM users WHERE email = ?", "nonexistent@example.com").FindOne(ctx, &user)
	assert.Error(t, err)

	// Test Find with no results (should not error)
	var users []TestUser
	err = td.DB.Raw("SELECT * FROM users WHERE email = ?", "nonexistent@example.com").Find(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 0)
}

func (dct *DriverConformanceTests) TestRawQueryParameterBinding(t *testing.T) {
	if dct.shouldSkip("TestRawQueryParameterBinding") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test various parameter types
	testCases := []struct {
		name   string
		query  string
		params []any
		verify func(t *testing.T, results []map[string]any)
	}{
		{
			name:   "String parameters",
			query:  "INSERT INTO users (name, email, active) VALUES (?, ?, ?)",
			params: []any{"Test User", "test@example.com", true},
			verify: func(t *testing.T, results []map[string]any) {
				count, err := td.DB.Model("User").Select().Count(ctx)
				assert.NoError(t, err)
				assert.Equal(t, int64(1), count)
			},
		},
		{
			name:   "Numeric parameters",
			query:  "SELECT * FROM users WHERE name = ? OR name = ?",
			params: []any{"Alice", "Bob"},
			verify: func(t *testing.T, results []map[string]any) {
				assert.Len(t, results, 2)
			},
		},
		{
			name:   "Mixed types",
			query:  "SELECT * FROM users WHERE active = ? AND id > ?",
			params: []any{true, 0},
			verify: func(t *testing.T, results []map[string]any) {
				assert.Greater(t, len(results), 0)
			},
		},
	}

	for _, tc := range testCases {
		// Clear data for each test - delete in correct order for FK constraints
		td.DB.Raw("DELETE FROM comments").Exec(ctx)
		td.DB.Raw("DELETE FROM posts").Exec(ctx)
		td.DB.Raw("DELETE FROM users").Exec(ctx)

		// Insert initial data if needed
		if tc.name != "String parameters" {
			td.InsertStandardTestData()
		}

		// Execute test
		if tc.name == "String parameters" {
			_, err := td.DB.Raw(tc.query, tc.params...).Exec(ctx)
			assert.NoError(t, err)
			tc.verify(t, nil)
		} else {
			var results []map[string]any
			err := td.DB.Raw(tc.query, tc.params...).Find(ctx, &results)
			assert.NoError(t, err)
			tc.verify(t, results)
		}
	}
}

// Extended Transaction Tests

func (dct *DriverConformanceTests) TestTransactionQueryInTransaction(t *testing.T) {
	if dct.shouldSkip("TestTransactionQueryInTransaction") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Begin transaction
	tx, err := td.DB.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Insert data in transaction
	UserTx := tx.Model("User")
	_, err = UserTx.Insert(map[string]any{
		"name":   "TxVisible",
		"email":  "txvisible@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Query should see the uncommitted data within the same transaction
	var users []TestUser
	err = UserTx.Select().FindMany(ctx, &users)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "TxVisible", users[0].Name)

	// Query from outside transaction should NOT see the data
	var outsideUsers []TestUser
	err = td.DB.Model("User").Select().FindMany(ctx, &outsideUsers)
	assert.NoError(t, err)
	assert.Len(t, outsideUsers, 0)

	// Commit transaction
	err = tx.Commit(ctx)
	assert.NoError(t, err)

	// Now query from outside should see the data
	err = td.DB.Model("User").Select().FindMany(ctx, &outsideUsers)
	assert.NoError(t, err)
	assert.Len(t, outsideUsers, 1)
}

func (dct *DriverConformanceTests) TestTransactionWithRawQueries(t *testing.T) {
	if dct.shouldSkip("TestTransactionWithRawQueries") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Begin transaction
	tx, err := td.DB.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Execute raw query in transaction
	_, err = tx.Raw("INSERT INTO users (name, email, active) VALUES (?, ?, ?)",
		"RawTxUser", "rawtx@example.com", true).Exec(ctx)
	assert.NoError(t, err)

	// Query with raw query in transaction
	var results []map[string]any
	err = tx.Raw("SELECT * FROM users WHERE email = ?", "rawtx@example.com").Find(ctx, &results)
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	// FindOne in transaction
	var user map[string]any
	err = tx.Raw("SELECT * FROM users WHERE email = ?", "rawtx@example.com").FindOne(ctx, &user)
	assert.NoError(t, err)
	assert.Equal(t, "RawTxUser", user["name"])

	// Commit
	err = tx.Commit(ctx)
	assert.NoError(t, err)
}

func (dct *DriverConformanceTests) TestTransactionErrorHandling(t *testing.T) {
	if dct.shouldSkip("TestTransactionErrorHandling") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test transaction with SQL error
	tx, err := td.DB.Begin(ctx)
	require.NoError(t, err)

	// Try to insert with invalid data
	UserTx := tx.Model("User")
	_, err = UserTx.Insert(map[string]any{
		// Missing required fields
		"email": "incomplete@example.com",
	}).Exec(ctx)
	assert.Error(t, err)

	// Transaction should still be usable
	_, err = UserTx.Insert(map[string]any{
		"name":   "ValidUser",
		"email":  "valid@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Commit should succeed
	err = tx.Commit(ctx)
	assert.NoError(t, err)

	// Verify only valid data was committed
	td.AssertCount("User", 1)
}

func (dct *DriverConformanceTests) TestTransactionConcurrentAccess(t *testing.T) {
	if dct.shouldSkip("TestTransactionConcurrentAccess") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	ctx := context.Background()

	// Test that multiple transactions can work concurrently
	tx1, err := td.DB.Begin(ctx)
	require.NoError(t, err)
	defer tx1.Rollback(ctx)

	tx2, err := td.DB.Begin(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	// Insert different data in each transaction
	_, err = tx1.Model("User").Insert(map[string]any{
		"name":   "Tx1User",
		"email":  "tx1@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	_, err = tx2.Model("User").Insert(map[string]any{
		"name":   "Tx2User",
		"email":  "tx2@example.com",
		"active": true,
	}).Exec(ctx)
	assert.NoError(t, err)

	// Each transaction should only see its own data
	var tx1Users []TestUser
	err = tx1.Model("User").Select().FindMany(ctx, &tx1Users)
	assert.NoError(t, err)
	assert.Len(t, tx1Users, 1)
	if len(tx1Users) > 0 {
		assert.Equal(t, "Tx1User", tx1Users[0].Name)
	}

	var tx2Users []TestUser
	err = tx2.Model("User").Select().FindMany(ctx, &tx2Users)
	assert.NoError(t, err)
	assert.Len(t, tx2Users, 1)
	if len(tx2Users) > 0 {
		assert.Equal(t, "Tx2User", tx2Users[0].Name)
	}

	// Commit both
	err = tx1.Commit(ctx)
	assert.NoError(t, err)

	err = tx2.Commit(ctx)
	assert.NoError(t, err)

	// Now both should be visible
	td.AssertCount("User", 2)
}
