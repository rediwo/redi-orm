package mongodb

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestMongoDBURI() string {
	// Use local MongoDB with authentication from docker-compose
	return "mongodb://testuser:testpass@localhost:27017/testdb?authSource=admin"
}

func TestMongoDB_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MongoDB test in short mode")
	}

	db, err := NewMongoDB(getTestMongoDBURI())
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer db.Close()

	// Test ping
	err = db.Ping(ctx)
	assert.NoError(t, err)
}

func TestMongoDB_SchemaOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MongoDB test in short mode")
	}

	db, err := NewMongoDB(getTestMongoDBURI())
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer db.Close()

	// Create a test schema
	userSchema := schema.New("User")
	userSchema.AddField(schema.Field{
		Name:       "id",
		Type:       schema.FieldTypeObjectId,
		PrimaryKey: true,
	})
	userSchema.AddField(schema.Field{
		Name:     "name",
		Type:     schema.FieldTypeString,
		Nullable: false,
	})
	userSchema.AddField(schema.Field{
		Name:   "email",
		Type:   schema.FieldTypeString,
		Unique: true,
	})
	userSchema.AddField(schema.Field{
		Name:     "profile",
		Type:     schema.FieldTypeDocument,
		Nullable: true,
	})

	// Register schema
	err = db.RegisterSchema("User", userSchema)
	assert.NoError(t, err)

	// Create model
	err = db.CreateModel(ctx, "User")
	assert.NoError(t, err)

	// Drop model
	err = db.DropModel(ctx, "User")
	assert.NoError(t, err)
}

func TestMongoDB_URIParser(t *testing.T) {
	parser := NewMongoDBURIParser()

	testCases := []struct {
		name        string
		uri         string
		expectError bool
	}{
		{
			name:        "Valid MongoDB URI",
			uri:         "mongodb://localhost:27017/testdb",
			expectError: false,
		},
		{
			name:        "Valid MongoDB SRV URI",
			uri:         "mongodb+srv://user:pass@cluster.mongodb.net/testdb",
			expectError: false,
		},
		{
			name:        "Invalid scheme",
			uri:         "mysql://localhost/testdb",
			expectError: true,
		},
		{
			name:        "Missing host",
			uri:         "mongodb:///testdb",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parser.ParseURI(tc.uri)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.uri, result)
			}
		})
	}
}

func TestMongoDB_Capabilities(t *testing.T) {
	caps := NewMongoDBCapabilities()

	assert.True(t, caps.IsNoSQL())
	assert.True(t, caps.SupportsTransactions())
	assert.True(t, caps.SupportsNestedDocuments())
	assert.True(t, caps.SupportsArrayFields())
	assert.True(t, caps.SupportsAggregationPipeline())
	assert.False(t, caps.SupportsDistinctOn())
	assert.Equal(t, "mongodb", string(caps.GetDriverType()))
}
