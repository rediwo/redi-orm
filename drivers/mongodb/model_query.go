package mongodb

import (
	"context"
	"fmt"

	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBModelQuery extends the base ModelQuery with MongoDB-specific implementations
type MongoDBModelQuery struct {
	*query.ModelQueryImpl
	db          *MongoDB
	fieldMapper types.FieldMapper
	modelName   string
}

// NewMongoDBModelQuery creates a new MongoDB model query
func NewMongoDBModelQuery(db *MongoDB, modelName string) types.ModelQuery {
	fieldMapper := db.GetFieldMapper()
	baseQuery := query.NewModelQuery(modelName, db, fieldMapper)
	return &MongoDBModelQuery{
		ModelQueryImpl: baseQuery,
		db:             db,
		fieldMapper:    fieldMapper,
		modelName:      modelName,
	}
}

// Insert creates a MongoDB-specific insert query
func (q *MongoDBModelQuery) Insert(data any) types.InsertQuery {
	return NewMongoDBInsertQuery(q.ModelQueryImpl, data, q.db, q.fieldMapper, q.modelName)
}

// Select creates a MongoDB-specific select query
func (q *MongoDBModelQuery) Select(fieldNames ...string) types.SelectQuery {
	return NewMongoDBSelectQuery(q.ModelQueryImpl, fieldNames, q.db, q.fieldMapper, q.modelName)
}

// Update creates a MongoDB-specific update query
func (q *MongoDBModelQuery) Update(data any) types.UpdateQuery {
	return NewMongoDBUpdateQuery(q.ModelQueryImpl, data, q.db, q.fieldMapper, q.modelName)
}

// Delete creates a MongoDB-specific delete query
func (q *MongoDBModelQuery) Delete() types.DeleteQuery {
	return NewMongoDBDeleteQuery(q.ModelQueryImpl, q.db, q.fieldMapper, q.modelName)
}

// Where creates a MongoDB-specific field condition
func (q *MongoDBModelQuery) Where(fieldName string) types.FieldCondition {
	return NewMongoDBFieldCondition(q.modelName, fieldName, q.db)
}

// Override condition methods to preserve MongoDB-specific type
func (q *MongoDBModelQuery) WhereCondition(condition types.Condition) types.ModelQuery {
	// Call the base method to get a new ModelQueryImpl
	newBase := q.ModelQueryImpl.WhereCondition(condition).(*query.ModelQueryImpl)
	// Return a new MongoDB model query with the updated base
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

func (q *MongoDBModelQuery) WhereRaw(sql string, args ...any) types.ModelQuery {
	newBase := q.ModelQueryImpl.WhereRaw(sql, args...).(*query.ModelQueryImpl)
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

func (q *MongoDBModelQuery) Include(relations ...string) types.ModelQuery {
	newBase := q.ModelQueryImpl.Include(relations...).(*query.ModelQueryImpl)
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

func (q *MongoDBModelQuery) With(relations ...string) types.ModelQuery {
	return q.Include(relations...)
}

func (q *MongoDBModelQuery) OrderBy(fieldName string, direction types.Order) types.ModelQuery {
	newBase := q.ModelQueryImpl.OrderBy(fieldName, direction).(*query.ModelQueryImpl)
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

func (q *MongoDBModelQuery) GroupBy(fieldNames ...string) types.ModelQuery {
	newBase := q.ModelQueryImpl.GroupBy(fieldNames...).(*query.ModelQueryImpl)
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

func (q *MongoDBModelQuery) Having(condition types.Condition) types.ModelQuery {
	newBase := q.ModelQueryImpl.Having(condition).(*query.ModelQueryImpl)
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

func (q *MongoDBModelQuery) Limit(limit int) types.ModelQuery {
	newBase := q.ModelQueryImpl.Limit(limit).(*query.ModelQueryImpl)
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

func (q *MongoDBModelQuery) Offset(offset int) types.ModelQuery {
	newBase := q.ModelQueryImpl.Offset(offset).(*query.ModelQueryImpl)
	return &MongoDBModelQuery{
		ModelQueryImpl: newBase,
		db:             q.db,
		fieldMapper:    q.fieldMapper,
		modelName:      q.modelName,
	}
}

// Override execution methods to ensure MongoDB-specific queries are used
func (q *MongoDBModelQuery) Count(ctx context.Context) (int64, error) {
	// Create a select query that preserves all conditions from the model query
	selectQuery := q.Select()
	
	// The base ModelQueryImpl already has the conditions, and they should be passed
	// to the SelectQuery through the base query mechanism
	return selectQuery.Count(ctx)
}

func (q *MongoDBModelQuery) Exists(ctx context.Context) (bool, error) {
	count, err := q.Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Avg calculates the average value of a numeric field using MongoDB aggregation
func (q *MongoDBModelQuery) Avg(ctx context.Context, fieldName string) (float64, error) {
	result, err := q.aggregateField(ctx, fieldName, "$avg")
	if err != nil {
		return 0, err
	}
	
	if result == nil {
		return 0, nil
	}
	
	// Convert result to float64
	switch v := result.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unexpected result type for avg: %T", result)
	}
}

// Sum calculates the sum of a numeric field using MongoDB aggregation
func (q *MongoDBModelQuery) Sum(ctx context.Context, fieldName string) (float64, error) {
	result, err := q.aggregateField(ctx, fieldName, "$sum")
	if err != nil {
		return 0, err
	}
	
	if result == nil {
		return 0, nil
	}
	
	// Convert result to float64
	switch v := result.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unexpected result type for sum: %T", result)
	}
}

// Min finds the minimum value of a field using MongoDB aggregation
func (q *MongoDBModelQuery) Min(ctx context.Context, fieldName string) (any, error) {
	return q.aggregateField(ctx, fieldName, "$min")
}

// Max finds the maximum value of a field using MongoDB aggregation
func (q *MongoDBModelQuery) Max(ctx context.Context, fieldName string) (any, error) {
	return q.aggregateField(ctx, fieldName, "$max")
}

// aggregateField performs aggregation on a field using MongoDB aggregation pipeline
func (q *MongoDBModelQuery) aggregateField(ctx context.Context, fieldName, operation string) (any, error) {
	// Get collection name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve collection name: %w", err)
	}

	// Map field name to column name
	columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, fieldName)
	if err != nil {
		return nil, fmt.Errorf("failed to map field name %s: %w", fieldName, err)
	}

	// Build aggregation pipeline
	pipeline := []bson.M{}

	// Add $match stage for WHERE conditions if any
	conditions := q.GetConditions()
	if len(conditions) > 0 {
		qb := NewMongoDBQueryBuilder(q.db)
		
		// Combine all conditions with AND
		var combined types.Condition
		for i, cond := range conditions {
			if i == 0 {
				combined = cond
			} else {
				combined = combined.And(cond)
			}
		}
		
		filter, err := qb.ConditionToFilter(combined, q.modelName)
		if err != nil {
			return nil, fmt.Errorf("failed to build filter: %w", err)
		}
		
		if len(filter) > 0 {
			pipeline = append(pipeline, bson.M{"$match": filter})
		}
	}

	// Add aggregation stage
	groupStage := bson.M{
		"$group": bson.M{
			"_id":    nil, // Group all documents
			"result": bson.M{operation: "$" + columnName},
		},
	}
	pipeline = append(pipeline, groupStage)

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:  "aggregate",
		Collection: tableName,
		Pipeline:   pipeline,
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return nil, err
	}

	// Execute aggregation
	rawQuery := q.db.Raw(jsonCmd)
	
	var result []map[string]any
	err = rawQuery.Find(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation: %w", err)
	}

	// Extract result
	if len(result) == 0 {
		return nil, nil
	}
	
	return result[0]["result"], nil
}
