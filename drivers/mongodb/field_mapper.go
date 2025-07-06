package mongodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBFieldMapper wraps the base field mapper to handle MongoDB-specific field mapping
type MongoDBFieldMapper struct {
	types.FieldMapper
	db *MongoDB
}

// NewMongoDBFieldMapper creates a new MongoDB field mapper
func NewMongoDBFieldMapper(baseMapper types.FieldMapper, db *MongoDB) *MongoDBFieldMapper {
	return &MongoDBFieldMapper{
		FieldMapper: baseMapper,
		db:          db,
	}
}

// RegisterSchema registers a schema with the underlying field mapper
func (m *MongoDBFieldMapper) RegisterSchema(modelName string, s *schema.Schema) {
	// Delegate to the embedded field mapper if it supports schema registration
	if mapper, ok := m.FieldMapper.(*types.DefaultFieldMapper); ok {
		mapper.RegisterSchema(modelName, s)
	}
}

// SchemaToColumn maps a schema field name to MongoDB column name
// This handles the special case of primary keys being mapped to _id
func (m *MongoDBFieldMapper) SchemaToColumn(modelName, fieldName string) (string, error) {
	// Try to get the schema to check if this is a primary key field
	schema, err := m.db.GetSchema(modelName)
	if err != nil {
		// If schema is not available, fall back to base mapper
		columnName, err := m.FieldMapper.SchemaToColumn(modelName, fieldName)
		if err != nil {
			// If base mapper also fails, just convert the field name
			return utils.ToSnakeCase(fieldName), nil
		}
		return columnName, nil
	}

	// Check if this is a primary key field
	// Only map to _id if it's a single primary key, not part of a composite key
	primaryKeyFields := m.getPrimaryKeyFields(schema)
	if len(primaryKeyFields) == 1 && primaryKeyFields[0].Name == fieldName {
		return "_id", nil
	}

	// For non-primary key fields, use the base mapper
	return m.FieldMapper.SchemaToColumn(modelName, fieldName)
}

// ColumnToSchema maps a MongoDB column name to schema field name
// This handles the special case of _id being mapped back to primary key fields
func (m *MongoDBFieldMapper) ColumnToSchema(modelName, columnName string) (string, error) {
	// Handle _id special case
	if columnName == "_id" {
		schema, err := m.db.GetSchema(modelName)
		if err != nil {
			// If schema is not available, fall back to base mapper
			return m.FieldMapper.ColumnToSchema(modelName, columnName)
		}

		// Get primary key field(s)
		primaryKeyFields := m.getPrimaryKeyFields(schema)
		if len(primaryKeyFields) == 1 {
			// Single primary key - return its name
			return primaryKeyFields[0].Name, nil
		} else if len(primaryKeyFields) > 1 {
			// Composite primary key - this is complex, for now return "id"
			// TODO: Handle composite keys properly
			return "id", nil
		}
	}

	// For non-_id fields, use the base mapper
	return m.FieldMapper.ColumnToSchema(modelName, columnName)
}

// ModelToTable converts model name to MongoDB collection name
// This handles cases where schemas haven't been registered yet
func (m *MongoDBFieldMapper) ModelToTable(modelName string) (string, error) {
	// Try to get the collection name from the schema first
	if schema, err := m.db.GetSchema(modelName); err == nil {
		return schema.GetTableName(), nil
	}

	// If schema is not available, use default naming convention
	// Convert model name to collection name (pluralized, snake_case)
	snakeCase := utils.ToSnakeCase(modelName)
	return utils.Pluralize(snakeCase), nil
}

// MapSchemaToColumnData maps a data map from schema field names to column names
// This handles MongoDB's _id field mapping and composite primary keys
func (m *MongoDBFieldMapper) MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Try to get the schema to understand primary key structure
	schema, err := m.db.GetSchema(modelName)
	if err != nil {
		// If schema is not available, fall back to base mapper
		return m.FieldMapper.MapSchemaToColumnData(modelName, data)
	}

	mapped := make(map[string]any)
	primaryKeyFields := m.getPrimaryKeyFields(schema)

	// Handle primary key mapping to _id
	if len(primaryKeyFields) == 1 {
		// Single primary key
		pkField := primaryKeyFields[0]
		if value, exists := data[pkField.Name]; exists {
			mapped["_id"] = value
		}
	} else if len(primaryKeyFields) > 1 {
		// Composite primary key - create an object for _id
		compositeKey := bson.M{}
		hasAnyKey := false
		for _, pkField := range primaryKeyFields {
			if value, exists := data[pkField.Name]; exists {
				// Map field name to column name for composite key
				columnName, err := m.FieldMapper.SchemaToColumn(modelName, pkField.Name)
				if err != nil {
					columnName = pkField.Name
				}
				compositeKey[columnName] = value
				hasAnyKey = true
			}
		}
		if hasAnyKey {
			mapped["_id"] = compositeKey
		}
	}

	// Map all other fields
	for field, value := range data {
		if !m.isPrimaryKeyField(schema, field) {
			// Use base mapper for non-primary key fields
			columnName, err := m.FieldMapper.SchemaToColumn(modelName, field)
			if err != nil {
				columnName = field
			}
			mapped[columnName] = value
		}
	}

	return mapped, nil
}

// MapColumnToSchemaData maps a data map from column names to schema field names
// This handles MongoDB's _id field mapping back to primary key fields
func (m *MongoDBFieldMapper) MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Try to get the schema to understand primary key structure
	schema, err := m.db.GetSchema(modelName)
	if err != nil {
		// If schema is not available, fall back to base mapper
		return m.FieldMapper.MapColumnToSchemaData(modelName, data)
	}

	mapped := make(map[string]any)
	primaryKeyFields := m.getPrimaryKeyFields(schema)

	// Handle _id mapping back to primary key fields
	if idValue, exists := data["_id"]; exists {
		if len(primaryKeyFields) == 1 {
			// Single primary key
			pkField := primaryKeyFields[0]
			mapped[pkField.Name] = idValue
		} else if len(primaryKeyFields) > 1 {
			// Composite primary key - extract from object
			if compositeKey, ok := idValue.(map[string]any); ok {
				for _, pkField := range primaryKeyFields {
					columnName, err := m.FieldMapper.SchemaToColumn(modelName, pkField.Name)
					if err != nil {
						columnName = pkField.Name
					}
					if value, exists := compositeKey[columnName]; exists {
						mapped[pkField.Name] = value
					}
				}
			} else if compositeKey, ok := idValue.(bson.M); ok {
				for _, pkField := range primaryKeyFields {
					columnName, err := m.FieldMapper.SchemaToColumn(modelName, pkField.Name)
					if err != nil {
						columnName = pkField.Name
					}
					if value, exists := compositeKey[columnName]; exists {
						mapped[pkField.Name] = value
					}
				}
			}
		}
	}

	// Map all other fields
	for column, value := range data {
		if column != "_id" {
			// Use base mapper for non-_id fields
			fieldName, err := m.FieldMapper.ColumnToSchema(modelName, column)
			if err != nil {
				fieldName = column
			}
			mapped[fieldName] = value
		}
	}

	return mapped, nil
}

// BuildMongoDBFilter builds a MongoDB filter from schema field names
// This properly handles primary key field mapping to _id
func (m *MongoDBFieldMapper) BuildMongoDBFilter(modelName string, conditions map[string]any) (bson.M, error) {
	if len(conditions) == 0 {
		return bson.M{}, nil
	}

	// Try to get the schema to understand primary key structure
	schema, err := m.db.GetSchema(modelName)
	if err != nil {
		// If schema is not available, use simple field mapping
		filter := bson.M{}
		for field, value := range conditions {
			// Use base mapper for field mapping
			columnName, err := m.FieldMapper.SchemaToColumn(modelName, field)
			if err != nil {
				columnName = field
			}
			filter[columnName] = value
		}
		return filter, nil
	}

	filter := bson.M{}
	primaryKeyFields := m.getPrimaryKeyFields(schema)

	// Handle primary key conditions
	pkConditions := make(map[string]any)
	for field, value := range conditions {
		if m.isPrimaryKeyField(schema, field) {
			pkConditions[field] = value
		}
	}

	if len(pkConditions) > 0 {
		if len(primaryKeyFields) == 1 {
			// Single primary key - directly map to _id
			pkField := primaryKeyFields[0]
			if value, exists := pkConditions[pkField.Name]; exists {
				filter["_id"] = value
			}
		} else if len(primaryKeyFields) > 1 {
			// Composite primary key - create nested conditions
			compositeConditions := bson.M{}
			for field, value := range pkConditions {
				columnName, err := m.FieldMapper.SchemaToColumn(modelName, field)
				if err != nil {
					columnName = field
				}
				compositeConditions[fmt.Sprintf("_id.%s", columnName)] = value
			}
			// Merge composite conditions into filter
			for k, v := range compositeConditions {
				filter[k] = v
			}
		}
	}

	// Handle non-primary key conditions
	for field, value := range conditions {
		if !m.isPrimaryKeyField(schema, field) {
			columnName, err := m.FieldMapper.SchemaToColumn(modelName, field)
			if err != nil {
				columnName = field
			}
			filter[columnName] = value
		}
	}

	return filter, nil
}

// getPrimaryKeyFields returns all primary key fields (single or composite)
func (m *MongoDBFieldMapper) getPrimaryKeyFields(s *schema.Schema) []schema.Field {
	var primaryKeyFields []schema.Field

	// Check for single primary key fields
	for _, field := range s.Fields {
		if field.PrimaryKey {
			primaryKeyFields = append(primaryKeyFields, field)
		}
	}

	// If no single primary key, check for composite key
	if len(primaryKeyFields) == 0 && len(s.CompositeKey) > 0 {
		for _, fieldName := range s.CompositeKey {
			if field, err := s.GetField(fieldName); err == nil {
				primaryKeyFields = append(primaryKeyFields, *field)
			}
		}
	}

	return primaryKeyFields
}

// isPrimaryKeyField checks if a field is part of the primary key
func (m *MongoDBFieldMapper) isPrimaryKeyField(schema *schema.Schema, fieldName string) bool {
	primaryKeyFields := m.getPrimaryKeyFields(schema)
	for _, pkField := range primaryKeyFields {
		if pkField.Name == fieldName {
			return true
		}
	}
	return false
}

// GetAutoIncrementFieldName returns the field name for auto-increment sequences
func (m *MongoDBFieldMapper) GetAutoIncrementFieldName(modelName string) string {
	return fmt.Sprintf("redi_sequence_%s", strings.ToLower(modelName))
}

// GetSequenceCollectionName returns the name of the sequence collection
func (m *MongoDBFieldMapper) GetSequenceCollectionName() string {
	return "redi_sequences"
}

// GenerateNextSequenceValue generates the next sequence value for auto-increment fields
// Uses MongoDB's findOneAndUpdate with upsert to atomically increment sequence values
func (m *MongoDBFieldMapper) GenerateNextSequenceValue(modelName string) (int64, error) {
	if m.db.client == nil {
		return 0, fmt.Errorf("MongoDB client not initialized")
	}

	// Get the collection name for the model
	collectionName, err := m.ModelToTable(modelName)
	if err != nil {
		return 0, fmt.Errorf("failed to get collection name for model %s: %w", modelName, err)
	}

	// Use the sequences collection
	sequencesCollection := m.db.client.Database(m.db.dbName).Collection(m.GetSequenceCollectionName())

	// Create filter and update documents
	filter := bson.M{"_id": collectionName}
	update := bson.M{
		"$inc": bson.M{"sequence_value": 1},
	}

	// Options for findOneAndUpdate
	opts := options.FindOneAndUpdate().
		SetUpsert(true).                 // Create if doesn't exist
		SetReturnDocument(options.After) // Return the updated document

	// Execute the atomic increment operation
	var result struct {
		ID            string `bson:"_id"`
		SequenceValue int64  `bson:"sequence_value"`
	}

	err = sequencesCollection.FindOneAndUpdate(context.Background(), filter, update, opts).Decode(&result)
	if err != nil {
		return 0, fmt.Errorf("failed to generate next sequence value for %s: %w", modelName, err)
	}

	return result.SequenceValue, nil
}
