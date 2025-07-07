package mongodb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	driverType := types.DriverMongoDB

	// Register MongoDB driver
	registry.Register(string(driverType), func(uri string) (types.Database, error) {
		return NewMongoDB(uri)
	})

	// Register MongoDB capabilities
	registry.RegisterCapabilities(driverType, NewMongoDBCapabilities())

	// Register MongoDB URI parser
	registry.RegisterURIParser(string(driverType), NewMongoDBURIParser())
}

// MongoDB implements the Database interface for MongoDB
type MongoDB struct {
	*base.Driver
	client    *mongo.Client
	nativeURI string
	dbName    string
}

// NewMongoDB creates a new MongoDB database instance
// The uri parameter should be a MongoDB connection string
func NewMongoDB(nativeURI string) (*MongoDB, error) {
	// Extract database name from URI
	dbName := extractDatabaseName(nativeURI)
	if dbName == "" {
		return nil, fmt.Errorf("database name is required in MongoDB URI")
	}

	// Create base driver
	baseDriver := base.NewDriver(nativeURI, types.DriverMongoDB)

	// Create MongoDB instance
	mongodb := &MongoDB{
		Driver:    baseDriver,
		nativeURI: nativeURI,
		dbName:    dbName,
	}

	// Replace base field mapper with MongoDB-specific field mapper
	mongoFieldMapper := NewMongoDBFieldMapper(baseDriver.FieldMapper, mongodb)
	baseDriver.FieldMapper = mongoFieldMapper

	return mongodb, nil
}

// Connect establishes connection to MongoDB
func (m *MongoDB) Connect(ctx context.Context) error {
	// Create client options
	clientOptions := options.Client().ApplyURI(m.nativeURI)

	// Create client
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client
	return nil
}

// Close closes the MongoDB connection
func (m *MongoDB) Close() error {
	if m.client != nil {
		return m.client.Disconnect(context.Background())
	}
	return nil
}

// Ping checks if the database is reachable
func (m *MongoDB) Ping(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("not connected to MongoDB")
	}
	return m.client.Ping(ctx, nil)
}

// SyncSchemas synchronizes all loaded schemas with the database
func (m *MongoDB) SyncSchemas(ctx context.Context) error {
	// In MongoDB, we don't create tables, but we can create collections and indexes
	for modelName, schema := range m.Schemas {
		if err := m.CreateModel(ctx, modelName); err != nil {
			return err
		}

		// Create indexes
		if err := m.createIndexes(ctx, modelName, schema); err != nil {
			return err
		}
	}
	return nil
}

// CreateModel creates a collection for the given model
func (m *MongoDB) CreateModel(ctx context.Context, modelName string) error {
	// For now, we just verify the schema exists
	_, err := m.GetSchema(modelName)
	if err != nil {
		return fmt.Errorf("failed to get schema for model %s: %w", modelName, err)
	}

	collectionName := m.getCollectionName(modelName)

	// Create collection (this is optional in MongoDB, collections are created on first insert)
	database := m.client.Database(m.dbName)
	err = database.CreateCollection(ctx, collectionName)
	if err != nil {
		// Ignore error if collection already exists
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// Skip validation for now - MongoDB's strict type checking conflicts with Go's type system
	// TODO: Implement more flexible validation that works with Go types
	// if err := m.createValidation(ctx, collectionName, schema); err != nil {
	// 	return err
	// }

	return nil
}

// DropModel drops the collection for the given model
func (m *MongoDB) DropModel(ctx context.Context, modelName string) error {
	collectionName := m.getCollectionName(modelName)
	collection := m.client.Database(m.dbName).Collection(collectionName)

	if err := collection.Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	return nil
}

// Model creates a new model query
func (m *MongoDB) Model(modelName string) types.ModelQuery {
	return NewMongoDBModelQuery(m, modelName)
}

// Raw creates a new raw query
func (m *MongoDB) Raw(command string, args ...any) types.RawQuery {
	return NewMongoDBRawQuery(m.client.Database(m.dbName), nil, m, command, args...)
}

// Begin starts a new transaction
func (m *MongoDB) Begin(ctx context.Context) (types.Transaction, error) {
	session, err := m.client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	if err := session.StartTransaction(); err != nil {
		session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	return NewMongoDBTransaction(session, m), nil
}

// Transaction executes a function within a transaction
func (m *MongoDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Start transaction manually for better control
	err = session.StartTransaction()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Create transaction wrapper
	tx := NewMongoDBTransaction(session, m)

	// Execute the function
	fnErr := fn(tx)

	if fnErr != nil {
		// Rollback on error
		if abortErr := session.AbortTransaction(ctx); abortErr != nil {
			return fmt.Errorf("failed to abort transaction (original error: %w): %v", fnErr, abortErr)
		}
		return fnErr
	}

	// Commit on success
	if commitErr := session.CommitTransaction(ctx); commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	return nil
}

// GetDriverType returns the database driver type
func (m *MongoDB) GetDriverType() string {
	return string(types.DriverMongoDB)
}

// GetCapabilities returns driver capabilities
func (m *MongoDB) GetCapabilities() types.DriverCapabilities {
	return NewMongoDBCapabilities()
}

// GetMigrator returns a migrator for MongoDB
func (m *MongoDB) GetMigrator() types.DatabaseMigrator {
	return NewMongoDBMigrator(m.client.Database(m.dbName), m)
}

// Exec is not directly applicable to MongoDB
func (m *MongoDB) Exec(query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("Exec is not supported for MongoDB, use Raw() instead")
}

// Query is not directly applicable to MongoDB
func (m *MongoDB) Query(query string, args ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("Query is not supported for MongoDB, use Model() or Raw() instead")
}

// QueryRow is not directly applicable to MongoDB
func (m *MongoDB) QueryRow(query string, args ...any) *sql.Row {
	// This method can't return an error, so we'll return a dummy row
	// Users should use Model() or Raw() instead
	return nil
}

// Helper methods

// getCollectionName converts model name to collection name
func (m *MongoDB) getCollectionName(modelName string) string {
	// Use pluralized snake_case for collection names
	return utils.ToSnakeCase(utils.Pluralize(modelName))
}

// extractDatabaseName extracts the database name from MongoDB URI
func extractDatabaseName(uri string) string {
	// Simple extraction - in production, use proper URI parsing
	parts := strings.Split(uri, "/")
	if len(parts) < 4 {
		return ""
	}

	dbPart := parts[3]
	// Remove query parameters
	if idx := strings.Index(dbPart, "?"); idx > 0 {
		dbPart = dbPart[:idx]
	}

	return dbPart
}

// createIndexes creates indexes for a model
func (m *MongoDB) createIndexes(ctx context.Context, modelName string, schema *schema.Schema) error {
	collection := m.client.Database(m.dbName).Collection(m.getCollectionName(modelName))

	var indexes []mongo.IndexModel

	// Create indexes for fields marked with Index: true
	for _, field := range schema.Fields {
		if field.Index && !field.PrimaryKey {
			indexes = append(indexes, mongo.IndexModel{
				Keys: bson.M{field.GetColumnName(): 1},
			})
		}

		if field.Unique && !field.PrimaryKey {
			indexes = append(indexes, mongo.IndexModel{
				Keys:    bson.M{field.GetColumnName(): 1},
				Options: options.Index().SetUnique(true),
			})
		}
	}

	// Create composite indexes if defined
	for _, index := range schema.Indexes {
		keys := bson.D{}
		for _, field := range index.Fields {
			keys = append(keys, bson.E{Key: field, Value: 1})
		}

		indexModel := mongo.IndexModel{Keys: keys}
		if index.Unique {
			indexModel.Options = options.Index().SetUnique(true)
		}
		if index.Name != "" {
			if indexModel.Options == nil {
				indexModel.Options = options.Index()
			}
			indexModel.Options.SetName(index.Name)
		}

		indexes = append(indexes, indexModel)
	}

	if len(indexes) > 0 {
		_, err := collection.Indexes().CreateMany(ctx, indexes)
		if err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	return nil
}

// createValidation creates validation rules for a collection
func (m *MongoDB) createValidation(ctx context.Context, collectionName string, schema *schema.Schema) error {
	// Build JSON Schema validation
	jsonSchema := m.buildJSONSchema(schema)

	// Apply validation rules
	database := m.client.Database(m.dbName)
	cmd := bson.D{
		{Key: "collMod", Value: collectionName},
		{Key: "validator", Value: bson.M{"$jsonSchema": jsonSchema}},
	}

	var result bson.M
	err := database.RunCommand(ctx, cmd).Decode(&result)
	if err != nil && !strings.Contains(err.Error(), "already has a validator") {
		// Ignore if validator already exists
		return fmt.Errorf("failed to create validation: %w", err)
	}

	return nil
}

// buildJSONSchema builds a JSON Schema from our schema definition
func (m *MongoDB) buildJSONSchema(s *schema.Schema) bson.M {
	properties := bson.M{}
	required := []string{}

	for _, field := range s.Fields {
		fieldSchema := m.fieldToJSONSchema(field)
		properties[field.GetColumnName()] = fieldSchema

		// Only require fields that don't have defaults and aren't auto-generated
		if !field.Nullable && !field.AutoIncrement && field.Default == nil {
			required = append(required, field.GetColumnName())
		}
	}

	return bson.M{
		"bsonType":   "object",
		"required":   required,
		"properties": properties,
	}
}

// fieldToJSONSchema converts a field to JSON Schema
func (m *MongoDB) fieldToJSONSchema(field schema.Field) bson.M {
	jsonSchema := bson.M{}

	switch field.Type {
	case schema.FieldTypeString:
		jsonSchema["bsonType"] = "string"
	case schema.FieldTypeInt, schema.FieldTypeInt64:
		// Accept both int and double for numeric fields
		jsonSchema["bsonType"] = []string{"int", "long", "double"}
	case schema.FieldTypeFloat:
		jsonSchema["bsonType"] = "double"
	case schema.FieldTypeBool:
		jsonSchema["bsonType"] = "bool"
	case schema.FieldTypeDateTime:
		jsonSchema["bsonType"] = "date"
	case schema.FieldTypeObjectId:
		jsonSchema["bsonType"] = "objectId"
	case schema.FieldTypeJSON, schema.FieldTypeDocument:
		jsonSchema["bsonType"] = "object"
	case schema.FieldTypeArray:
		jsonSchema["bsonType"] = "array"
	case schema.FieldTypeDecimal, schema.FieldTypeDecimal128:
		jsonSchema["bsonType"] = "decimal"
	default:
		// For array types
		if strings.HasSuffix(string(field.Type), "[]") {
			jsonSchema["bsonType"] = "array"
		}
	}

	return jsonSchema
}

// conditionToFilter converts a Condition to MongoDB filter
func (m *MongoDB) conditionToFilter(condition types.Condition) bson.M {
	qb := NewMongoDBQueryBuilder(m)
	filter, err := qb.ConditionToFilter(condition, "")
	if err != nil {
		// Return empty filter on error
		return bson.M{}
	}
	return filter
}
