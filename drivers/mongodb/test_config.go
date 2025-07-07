package mongodb

import (
	"context"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	host := test.GetEnvOrDefault("MONGODB_TEST_HOST", "localhost")
	port := test.GetEnvOrDefault("MONGODB_TEST_PORT", "27017")
	user := test.GetEnvOrDefault("MONGODB_TEST_USER", "testuser")
	password := test.GetEnvOrDefault("MONGODB_TEST_PASSWORD", "testpass")
	database := test.GetEnvOrDefault("MONGODB_TEST_DATABASE", "testdb")

	uri := "mongodb://" + user + ":" + password + "@" + host + ":" + port + "/" + database + "?authSource=admin"

	test.RegisterTestDatabaseUri("mongodb", uri)
}

// cleanupTables removes all non-system collections from the database
func cleanupTables(t *testing.T, db *MongoDB) {
	ctx := context.Background()

	if db.client == nil {
		t.Logf("MongoDB client not initialized")
		return
	}

	database := db.client.Database(db.dbName)

	// List all collections in the database
	collections, err := database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		t.Logf("Failed to list collections: %v", err)
		return
	}

	// Drop all collections including redi_sequences for test isolation
	for _, collectionName := range collections {
		// Skip MongoDB internal collections (starting with system.)
		if strings.HasPrefix(collectionName, "system.") {
			continue
		}

		t.Logf("Dropping collection: %s", collectionName)
		err := database.Collection(collectionName).Drop(ctx)
		if err != nil {
			t.Logf("Failed to drop collection %s: %v", collectionName, err)
		} else {
			t.Logf("Successfully dropped collection: %s", collectionName)
		}
	}

	// Extra safety: List collections again to verify cleanup
	collectionsAfter, err := database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		t.Logf("Failed to verify cleanup: %v", err)
		return
	}

	nonSystemCollections := 0
	for _, collectionName := range collectionsAfter {
		if !strings.HasPrefix(collectionName, "system.") {
			nonSystemCollections++
			t.Logf("Warning: Collection %s still exists after cleanup", collectionName)
		}
	}

	if nonSystemCollections == 0 {
		t.Logf("Cleanup verified: all non-system collections removed")
	} else {
		t.Logf("Cleanup incomplete: %d non-system collections remain", nonSystemCollections)
	}
}
