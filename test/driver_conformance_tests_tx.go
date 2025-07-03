package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
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
	for i := 0; i < 20; i++ {
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
	assert.Greater(t, result.RowsAffected, int64(0))
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
	assert.Len(t, results, 1)
}