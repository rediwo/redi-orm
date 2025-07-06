package mongodb

import (
	"context"
	"fmt"

	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBDeleteQuery implements DeleteQuery for MongoDB
type MongoDBDeleteQuery struct {
	*query.DeleteQueryImpl
	db          *MongoDB
	fieldMapper types.FieldMapper
	modelName   string
}

// NewMongoDBDeleteQuery creates a new MongoDB delete query
func NewMongoDBDeleteQuery(baseQuery *query.ModelQueryImpl, db *MongoDB, fieldMapper types.FieldMapper, modelName string) types.DeleteQuery {
	deleteQuery := query.NewDeleteQuery(baseQuery)
	return &MongoDBDeleteQuery{
		DeleteQueryImpl: deleteQuery,
		db:              db,
		fieldMapper:     fieldMapper,
		modelName:       modelName,
	}
}

// BuildSQL builds a MongoDB delete command instead of SQL
func (q *MongoDBDeleteQuery) BuildSQL() (string, []any, error) {
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
	// Delete filter: filter

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:  "delete",
		Collection: tableName,
		Filter:     filter,
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return "", nil, err
	}

	return jsonCmd, nil, nil
}

// buildFilter builds MongoDB filter from WHERE conditions
func (q *MongoDBDeleteQuery) buildFilter() (bson.M, error) {
	// Get conditions from both model query and delete query
	modelConditions := q.DeleteQueryImpl.ModelQueryImpl.GetConditions()
	deleteConditions := q.DeleteQueryImpl.GetWhereConditions()

	// Combine all conditions
	allConditions := append(modelConditions, deleteConditions...)

	if len(allConditions) == 0 {
		// MongoDB requires a filter for delete operations
		// Empty filter means delete all documents
		return bson.M{}, nil
	}

	// Use MongoDB query builder
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

// Exec executes the delete query
func (q *MongoDBDeleteQuery) Exec(ctx context.Context) (types.Result, error) {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build MongoDB command: %w", err)
	}

	rawQuery := q.db.Raw(sql, args...)
	result, err := rawQuery.Exec(ctx)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute delete: %w", err)
	}

	return result, nil
}

// ExecAndReturn is not supported for MongoDB deletes
func (q *MongoDBDeleteQuery) ExecAndReturn(ctx context.Context, dest any) error {
	// MongoDB doesn't support RETURNING clause like SQL databases
	// Deleted documents would need to be fetched before deletion
	return fmt.Errorf("ExecAndReturn is not supported for MongoDB")
}

// Override DeleteQuery methods to preserve MongoDB-specific type
func (q *MongoDBDeleteQuery) WhereCondition(condition types.Condition) types.DeleteQuery {
	newBase := q.DeleteQueryImpl.WhereCondition(condition).(*query.DeleteQueryImpl)
	return &MongoDBDeleteQuery{
		DeleteQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBDeleteQuery) Returning(fieldNames ...string) types.DeleteQuery {
	newBase := q.DeleteQueryImpl.Returning(fieldNames...).(*query.DeleteQueryImpl)
	return &MongoDBDeleteQuery{
		DeleteQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}
