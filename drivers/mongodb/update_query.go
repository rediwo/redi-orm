package mongodb

import (
	"context"
	"fmt"

	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBUpdateQuery implements UpdateQuery for MongoDB
type MongoDBUpdateQuery struct {
	*query.UpdateQueryImpl
	db          *MongoDB
	fieldMapper types.FieldMapper
	modelName   string
}

// NewMongoDBUpdateQuery creates a new MongoDB update query
func NewMongoDBUpdateQuery(baseQuery *query.ModelQueryImpl, data any, db *MongoDB, fieldMapper types.FieldMapper, modelName string) types.UpdateQuery {
	updateQuery := query.NewUpdateQuery(baseQuery, data)
	return &MongoDBUpdateQuery{
		UpdateQueryImpl: updateQuery,
		db:              db,
		fieldMapper:     fieldMapper,
		modelName:       modelName,
	}
}

// BuildSQL builds a MongoDB update command instead of SQL
func (q *MongoDBUpdateQuery) BuildSQL() (string, []any, error) {
	// Get collection name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve collection name: %w", err)
	}

	// Build filter from conditions
	filter, err := q.buildFilter()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build filter: %w", err)
	}

	// Build update document
	updateDoc, err := q.buildUpdateDocument()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build update document: %w", err)
	}

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:  "update",
		Collection: tableName,
		Filter:     filter,
		Update:     updateDoc,
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return "", nil, err
	}

	return jsonCmd, nil, nil
}

// buildFilter builds MongoDB filter from WHERE conditions
func (q *MongoDBUpdateQuery) buildFilter() (bson.M, error) {
	// Get conditions from both model query and update query
	modelConditions := q.UpdateQueryImpl.ModelQueryImpl.GetConditions()
	updateConditions := q.UpdateQueryImpl.GetWhereConditions()

	// Combine all conditions
	allConditions := append(modelConditions, updateConditions...)

	if len(allConditions) == 0 {
		// MongoDB requires a filter for update operations
		// Empty filter means update all documents
		return bson.M{}, nil
	}

	// Use query builder
	qb := NewMongoDBQueryBuilder(q.db)

	// Combine all conditions with AND
	var combined types.Condition
	for i, cond := range allConditions {
		if i == 0 {
			combined = cond
		} else {
			combined = combined.And(cond)
		}
	}

	return qb.ConditionToFilter(combined, q.modelName)
}

// buildUpdateDocument builds MongoDB update document
func (q *MongoDBUpdateQuery) buildUpdateDocument() (bson.M, error) {
	setData := q.UpdateQueryImpl.GetSetData()
	atomicOps := q.UpdateQueryImpl.GetAtomicOps()

	// Map schema field names to column names for set operations
	mappedData := make(map[string]any)
	if setData != nil {
		for field, value := range setData {
			columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, field)
			if err != nil {
				columnName = field
			}
			mappedData[columnName] = value
		}
	}

	// Build update document with operators
	updateDoc := bson.M{}

	// Add $set operations
	if len(mappedData) > 0 {
		updateDoc["$set"] = mappedData
	}

	// Add atomic operations ($inc, $dec, etc.)
	if len(atomicOps) > 0 {
		incDoc := bson.M{}
		decDoc := bson.M{}

		for field, op := range atomicOps {
			// Map field name to column name
			columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, field)
			if err != nil {
				columnName = field
			}

			switch op.Type {
			case "increment":
				incDoc[columnName] = op.Value
			case "decrement":
				decDoc[columnName] = -op.Value // MongoDB uses positive values with $inc for increment, negative for decrement
			}
		}

		if len(incDoc) > 0 {
			updateDoc["$inc"] = incDoc
		}
		if len(decDoc) > 0 {
			if existing, ok := updateDoc["$inc"]; ok {
				// Merge with existing $inc operations
				for k, v := range decDoc {
					existing.(bson.M)[k] = v
				}
			} else {
				updateDoc["$inc"] = decDoc
			}
		}
	}

	return updateDoc, nil
}

// Exec executes the update query
func (q *MongoDBUpdateQuery) Exec(ctx context.Context) (types.Result, error) {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build MongoDB command: %w", err)
	}

	rawQuery := q.db.Raw(sql, args...)
	result, err := rawQuery.Exec(ctx)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute update: %w", err)
	}

	return result, nil
}

// ExecAndReturn is not supported for MongoDB updates
func (q *MongoDBUpdateQuery) ExecAndReturn(ctx context.Context, dest any) error {
	// MongoDB doesn't support RETURNING clause like SQL databases
	// Updated documents would need to be fetched separately
	return fmt.Errorf("ExecAndReturn is not supported for MongoDB")
}

// Override UpdateQuery methods to preserve MongoDB-specific type
func (q *MongoDBUpdateQuery) Set(data any) types.UpdateQuery {
	newBase := q.UpdateQueryImpl.Set(data).(*query.UpdateQueryImpl)
	return &MongoDBUpdateQuery{
		UpdateQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBUpdateQuery) WhereCondition(condition types.Condition) types.UpdateQuery {
	newBase := q.UpdateQueryImpl.WhereCondition(condition).(*query.UpdateQueryImpl)
	return &MongoDBUpdateQuery{
		UpdateQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBUpdateQuery) Returning(fieldNames ...string) types.UpdateQuery {
	newBase := q.UpdateQueryImpl.Returning(fieldNames...).(*query.UpdateQueryImpl)
	return &MongoDBUpdateQuery{
		UpdateQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBUpdateQuery) Increment(fieldName string, value int64) types.UpdateQuery {
	newBase := q.UpdateQueryImpl.Increment(fieldName, value).(*query.UpdateQueryImpl)
	return &MongoDBUpdateQuery{
		UpdateQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBUpdateQuery) Decrement(fieldName string, value int64) types.UpdateQuery {
	newBase := q.UpdateQueryImpl.Decrement(fieldName, value).(*query.UpdateQueryImpl)
	return &MongoDBUpdateQuery{
		UpdateQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}
