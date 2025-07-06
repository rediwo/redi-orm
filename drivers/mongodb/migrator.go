package mongodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoDBMigrator implements DatabaseMigrator for MongoDB
type MongoDBMigrator struct {
	database *mongo.Database
	db       *MongoDB
}

// NewMongoDBMigrator creates a new MongoDB migrator
func NewMongoDBMigrator(database *mongo.Database, db *MongoDB) *MongoDBMigrator {
	return &MongoDBMigrator{
		database: database,
		db:       db,
	}
}

// GetTables returns all collections in the database
func (m *MongoDBMigrator) GetTables() ([]string, error) {
	ctx := context.Background()
	collections, err := m.database.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	// Filter out system collections
	var userCollections []string
	for _, coll := range collections {
		if !strings.HasPrefix(coll, "system.") {
			userCollections = append(userCollections, coll)
		}
	}

	return userCollections, nil
}

// GetTableInfo returns information about a collection
func (m *MongoDBMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	ctx := context.Background()
	collection := m.database.Collection(tableName)

	// Get indexes
	indexView := collection.Indexes()
	cursor, err := indexView.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer cursor.Close(ctx)

	var indexes []types.IndexInfo
	for cursor.Next(ctx) {
		var idx bson.M
		if err := cursor.Decode(&idx); err != nil {
			continue
		}

		// Parse index information
		if key, ok := idx["key"].(bson.M); ok {
			var columns []string
			for field := range key {
				columns = append(columns, field)
			}

			indexInfo := types.IndexInfo{
				Name:    idx["name"].(string),
				Columns: columns,
			}

			if unique, ok := idx["unique"].(bool); ok {
				indexInfo.Unique = unique
			}

			indexes = append(indexes, indexInfo)
		}
	}

	// MongoDB doesn't have a fixed schema, so we'll return basic info
	tableInfo := &types.TableInfo{
		Name:    tableName,
		Indexes: indexes,
		// MongoDB doesn't have columns in the traditional sense
		Columns: []types.ColumnInfo{},
	}

	return tableInfo, nil
}

// GenerateCreateTableSQL generates collection creation (not applicable for MongoDB)
func (m *MongoDBMigrator) GenerateCreateTableSQL(schema any) (string, error) {
	// MongoDB doesn't use SQL
	return "", fmt.Errorf("MongoDB doesn't use SQL for collection creation")
}

// GenerateDropTableSQL generates collection drop command
func (m *MongoDBMigrator) GenerateDropTableSQL(tableName string) string {
	// Return a pseudo-command for documentation
	return fmt.Sprintf("db.%s.drop()", tableName)
}

// GenerateAddColumnSQL is not applicable for MongoDB
func (m *MongoDBMigrator) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return "", fmt.Errorf("MongoDB doesn't have explicit column addition")
}

// GenerateModifyColumnSQL is not applicable for MongoDB
func (m *MongoDBMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	return nil, fmt.Errorf("MongoDB doesn't have explicit column modification")
}

// GenerateDropColumnSQL is not applicable for MongoDB
func (m *MongoDBMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	return nil, fmt.Errorf("MongoDB doesn't have explicit column deletion")
}

// GenerateCreateIndexSQL generates index creation command
func (m *MongoDBMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	// Generate a pseudo-command for documentation
	keys := "{"
	for i, col := range columns {
		if i > 0 {
			keys += ", "
		}
		keys += fmt.Sprintf("%s: 1", col)
	}
	keys += "}"

	options := ""
	if unique {
		options = ", {unique: true}"
	}

	return fmt.Sprintf("db.%s.createIndex(%s%s)", tableName, keys, options)
}

// GenerateDropIndexSQL generates index drop command
func (m *MongoDBMigrator) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("db.collection.dropIndex('%s')", indexName)
}

// ApplyMigration executes a migration command
func (m *MongoDBMigrator) ApplyMigration(sql string) error {
	// MongoDB doesn't use SQL migrations
	return fmt.Errorf("MongoDB doesn't support SQL migrations")
}

// GetDatabaseType returns the database type
func (m *MongoDBMigrator) GetDatabaseType() string {
	return "mongodb"
}

// CompareSchema compares existing collection with desired schema
func (m *MongoDBMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema any) (*types.MigrationPlan, error) {
	s, ok := desiredSchema.(*schema.Schema)
	if !ok {
		return nil, fmt.Errorf("expected *schema.Schema, got %T", desiredSchema)
	}

	plan := &types.MigrationPlan{
		AddIndexes:  []types.IndexChange{},
		DropIndexes: []types.IndexChange{},
	}

	// Compare indexes
	existingIndexMap := make(map[string]types.IndexInfo)
	for _, idx := range existingTable.Indexes {
		existingIndexMap[idx.Name] = idx
	}

	// Check for indexes to add
	for _, idx := range s.Indexes {
		indexName := m.getIndexName(existingTable.Name, idx.Fields)
		if _, exists := existingIndexMap[indexName]; !exists {
			plan.AddIndexes = append(plan.AddIndexes, types.IndexChange{
				TableName: existingTable.Name,
				IndexName: indexName,
				NewIndex: &types.IndexInfo{
					Name:    indexName,
					Columns: idx.Fields,
					Unique:  idx.Unique,
				},
			})
		}
	}

	return plan, nil
}

// GenerateMigrationSQL generates migration commands
func (m *MongoDBMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	var commands []string

	// Generate index creation commands
	for _, change := range plan.AddIndexes {
		if change.NewIndex != nil {
			cmd := m.GenerateCreateIndexSQL(
				change.TableName,
				change.IndexName,
				change.NewIndex.Columns,
				change.NewIndex.Unique,
			)
			commands = append(commands, cmd)
		}
	}

	// Generate index drop commands
	for _, change := range plan.DropIndexes {
		cmd := m.GenerateDropIndexSQL(change.IndexName)
		commands = append(commands, cmd)
	}

	return commands, nil
}

// getIndexName generates a consistent index name
func (m *MongoDBMigrator) getIndexName(tableName string, fields []string) string {
	return fmt.Sprintf("%s_%s_idx", tableName, strings.Join(fields, "_"))
}
