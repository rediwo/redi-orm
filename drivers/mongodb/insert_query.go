package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBInsertQuery implements InsertQuery for MongoDB
type MongoDBInsertQuery struct {
	*query.InsertQueryImpl
	data         []any // Keep our own copy of data
	db           *MongoDB
	fieldMapper  types.FieldMapper
	modelName    string
	lastInsertID int64 // Track generated auto-increment ID
}

// NewMongoDBInsertQuery creates a new MongoDB insert query
func NewMongoDBInsertQuery(baseQuery *query.ModelQueryImpl, data any, db *MongoDB, fieldMapper types.FieldMapper, modelName string) types.InsertQuery {
	insertQuery := query.NewInsertQuery(baseQuery, data)
	return &MongoDBInsertQuery{
		InsertQueryImpl: insertQuery,
		data:            []any{data},
		db:              db,
		fieldMapper:     fieldMapper,
		modelName:       modelName,
	}
}

// Values adds more data to insert
func (q *MongoDBInsertQuery) Values(data ...any) types.InsertQuery {
	// Call parent method
	newBase := q.InsertQueryImpl.Values(data...)
	// Create new MongoDB query with updated data
	return &MongoDBInsertQuery{
		InsertQueryImpl: newBase.(*query.InsertQueryImpl),
		data:            append(q.data, data...),
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
		lastInsertID:    q.lastInsertID, // Preserve lastInsertID
	}
}

// GetModelName returns the model name from the base query
func (q *MongoDBInsertQuery) GetModelName() string {
	if q.InsertQueryImpl == nil {
		return ""
	}
	return q.InsertQueryImpl.GetModelName()
}

// BuildSQL builds a MongoDB insert command instead of SQL
func (q *MongoDBInsertQuery) BuildSQL() (string, []any, error) {
	if len(q.data) == 0 {
		return "", nil, fmt.Errorf("no data to insert")
	}

	// Get collection name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve collection name: %w", err)
	}

	// Convert data to documents
	documents := make([]any, 0, len(q.data))
	for _, item := range q.data {
		doc, err := q.convertToDocument(item)
		if err != nil {
			return "", nil, fmt.Errorf("failed to convert data to document: %w", err)
		}
		documents = append(documents, doc)
	}

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:    "insert",
		Collection:   tableName,
		Documents:    documents,
		LastInsertID: q.lastInsertID,
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return "", nil, err
	}

	return jsonCmd, nil, nil
}

// convertToDocument converts input data to a MongoDB document
func (q *MongoDBInsertQuery) convertToDocument(data any) (bson.M, error) {
	// Convert data to map
	var dataMap map[string]any
	switch v := data.(type) {
	case map[string]any:
		dataMap = v
	case bson.M:
		dataMap = v
	default:
		// Use reflection to convert struct to map
		dataMap = make(map[string]any)
		val := reflect.ValueOf(data)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			return nil, fmt.Errorf("unsupported data type: %T", data)
		}

		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			fieldName := field.Name

			// Check for db tag
			if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
				fieldName = tag
			} else if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				fieldName = tag
			}

			fieldValue := val.Field(i).Interface()
			if !val.Field(i).IsZero() {
				dataMap[fieldName] = fieldValue
			}
		}
	}

	// Use MongoDB field mapper for proper _id handling
	mongoMapper, ok := q.fieldMapper.(*MongoDBFieldMapper)
	if !ok {
		return nil, fmt.Errorf("expected MongoDB field mapper, got %T", q.fieldMapper)
	}

	// Check for auto-increment primary key fields and apply default values
	if schema, err := q.db.GetSchema(q.modelName); err == nil {
		// First, apply default values for fields not provided
		for _, field := range schema.Fields {
			if _, exists := dataMap[field.Name]; !exists && field.Default != nil {
				// Apply default value
				switch v := field.Default.(type) {
				case string:
					if v == "now()" {
						dataMap[field.Name] = time.Now()
					} else {
						dataMap[field.Name] = v
					}
				default:
					dataMap[field.Name] = v
				}
			}
		}

		// Then handle auto-increment fields
		primaryKeyFields := mongoMapper.getPrimaryKeyFields(schema)
		for _, pkField := range primaryKeyFields {
			// If this is an auto-increment field and no value was provided
			if pkField.AutoIncrement && dataMap[pkField.Name] == nil {
				// Generate next sequence value
				nextID, err := mongoMapper.GenerateNextSequenceValue(q.modelName)
				if err != nil {
					return nil, fmt.Errorf("failed to generate auto-increment ID: %w", err)
				}
				dataMap[pkField.Name] = nextID
				// Store the generated ID for LastInsertID
				q.lastInsertID = nextID
				// Generated ID for sequence: nextID
			}
		}
	}

	// Map schema field names to MongoDB column names (handles _id mapping)
	docMap, err := mongoMapper.MapSchemaToColumnData(q.modelName, dataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to map field names: %w", err)
	}


	return docMap, nil
}

// Exec executes the insert query
func (q *MongoDBInsertQuery) Exec(ctx context.Context) (types.Result, error) {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build MongoDB command: %w", err)
	}

	rawQuery := q.db.Raw(sql, args...)
	result, err := rawQuery.Exec(ctx)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute insert: %w", err)
	}

	// Override LastInsertID if we generated a sequence value
	if q.lastInsertID > 0 {
		result.LastInsertID = q.lastInsertID
	}

	return result, nil
}

// ExecAndReturn is not supported for MongoDB inserts
func (q *MongoDBInsertQuery) ExecAndReturn(ctx context.Context, dest any) error {
	// MongoDB doesn't support RETURNING clause like SQL databases
	// The inserted documents with their generated IDs would need to be fetched separately
	return fmt.Errorf("ExecAndReturn is not supported for MongoDB")
}

// GetFieldMapper returns the field mapper
func (q *MongoDBInsertQuery) GetFieldMapper() types.FieldMapper {
	if q.InsertQueryImpl == nil {
		return nil
	}
	v := reflect.ValueOf(q.InsertQueryImpl).Elem()
	mqField := v.FieldByName("ModelQueryImpl")
	if !mqField.IsValid() || mqField.IsNil() {
		return nil
	}
	mqValue := mqField.Elem()
	fmField := mqValue.FieldByName("fieldMapper")
	if fmField.IsValid() && fmField.CanInterface() {
		return fmField.Interface().(types.FieldMapper)
	}
	return nil
}

// GetDatabase returns the database
func (q *MongoDBInsertQuery) GetDatabase() types.Database {
	if q.InsertQueryImpl == nil {
		return nil
	}
	v := reflect.ValueOf(q.InsertQueryImpl).Elem()
	mqField := v.FieldByName("ModelQueryImpl")
	if !mqField.IsValid() || mqField.IsNil() {
		return nil
	}
	mqValue := mqField.Elem()
	dbField := mqValue.FieldByName("database")
	if dbField.IsValid() && dbField.CanInterface() {
		return dbField.Interface().(types.Database)
	}
	return nil
}
