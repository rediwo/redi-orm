//go:build integration
// +build integration

package mongodb

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoDBIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with database package
	db, err := database.NewFromURI("mongodb://localhost:27017/rediorm_integration_test")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer db.Close()

	// Create a schema
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:       "_id",
			Type:       schema.FieldTypeObjectId,
			PrimaryKey: true,
		}).
		AddField(schema.Field{
			Name: "name",
			Type: schema.FieldTypeString,
		}).
		AddField(schema.Field{
			Name:   "email",
			Type:   schema.FieldTypeString,
			Unique: true,
		}).
		AddField(schema.Field{
			Name: "tags",
			Type: schema.FieldTypeStringArray,
		})

	// Register and sync schema
	err = db.RegisterSchema("User", userSchema)
	assert.NoError(t, err)

	err = db.CreateModel(ctx, "User")
	assert.NoError(t, err)

	// Test model query
	userModel := db.Model("User")
	assert.NotNil(t, userModel)

	// Clean up
	err = db.DropModel(ctx, "User")
	assert.NoError(t, err)
}

func TestMongoDBCapabilitiesCheck(t *testing.T) {
	db, err := NewMongoDB("mongodb://localhost:27017/test")
	require.NoError(t, err)

	caps := db.GetCapabilities()
	assert.True(t, caps.IsNoSQL())
	assert.True(t, caps.SupportsTransactions())
	assert.True(t, caps.SupportsNestedDocuments())
	assert.True(t, caps.SupportsArrayFields())
	assert.True(t, caps.SupportsAggregationPipeline())
	assert.Equal(t, "mongodb", string(caps.GetDriverType()))
}
