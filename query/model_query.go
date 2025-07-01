package query

import (
	"context"
	"fmt"

	"github.com/rediwo/redi-orm/types"
)

// ModelQueryImpl implements the ModelQuery interface
type ModelQueryImpl struct {
	modelName   string
	database    types.Database
	fieldMapper types.FieldMapper
	conditions  []types.Condition
	includes    []string
	orderBy     []OrderClause
	groupBy     []string
	having      types.Condition
	limit       *int
	offset      *int
}

type OrderClause struct {
	FieldName string
	Direction types.Order
}

// NewModelQuery creates a new model query
func NewModelQuery(modelName string, database types.Database, fieldMapper types.FieldMapper) *ModelQueryImpl {
	return &ModelQueryImpl{
		modelName:   modelName,
		database:    database,
		fieldMapper: fieldMapper,
		conditions:  []types.Condition{},
		includes:    []string{},
		orderBy:     []OrderClause{},
		groupBy:     []string{},
	}
}

// GetModelName returns the model name
func (q *ModelQueryImpl) GetModelName() string {
	return q.modelName
}

// Select creates a new select query
func (q *ModelQueryImpl) Select(fields ...string) types.SelectQuery {
	return NewSelectQuery(q.clone(), fields)
}

// Insert creates a new insert query
func (q *ModelQueryImpl) Insert(data interface{}) types.InsertQuery {
	return NewInsertQuery(q.clone(), data)
}

// Update creates a new update query
func (q *ModelQueryImpl) Update(data interface{}) types.UpdateQuery {
	return NewUpdateQuery(q.clone(), data)
}

// Delete creates a new delete query
func (q *ModelQueryImpl) Delete() types.DeleteQuery {
	return NewDeleteQuery(q.clone())
}

// Where adds a field condition
func (q *ModelQueryImpl) Where(fieldName string) types.FieldCondition {
	return types.NewFieldCondition(q.modelName, fieldName)
}

// WhereCondition adds a condition
func (q *ModelQueryImpl) WhereCondition(condition types.Condition) types.ModelQuery {
	newQuery := q.clone()
	newQuery.conditions = append(newQuery.conditions, condition)
	return newQuery
}

// WhereRaw adds a raw SQL condition
func (q *ModelQueryImpl) WhereRaw(sql string, args ...interface{}) types.ModelQuery {
	newQuery := q.clone()
	newQuery.conditions = append(newQuery.conditions, types.NewRawCondition(sql, args...))
	return newQuery
}

// Include adds relations to include
func (q *ModelQueryImpl) Include(relations ...string) types.ModelQuery {
	newQuery := q.clone()
	newQuery.includes = append(newQuery.includes, relations...)
	return newQuery
}

// With is an alias for Include
func (q *ModelQueryImpl) With(relations ...string) types.ModelQuery {
	return q.Include(relations...)
}

// OrderBy adds ordering
func (q *ModelQueryImpl) OrderBy(fieldName string, direction types.Order) types.ModelQuery {
	newQuery := q.clone()
	newQuery.orderBy = append(newQuery.orderBy, OrderClause{
		FieldName: fieldName,
		Direction: direction,
	})
	return newQuery
}

// GroupBy adds grouping
func (q *ModelQueryImpl) GroupBy(fieldNames ...string) types.ModelQuery {
	newQuery := q.clone()
	newQuery.groupBy = append(newQuery.groupBy, fieldNames...)
	return newQuery
}

// Having adds having condition
func (q *ModelQueryImpl) Having(condition types.Condition) types.ModelQuery {
	newQuery := q.clone()
	newQuery.having = condition
	return newQuery
}

// Limit sets the limit
func (q *ModelQueryImpl) Limit(limit int) types.ModelQuery {
	newQuery := q.clone()
	newQuery.limit = &limit
	return newQuery
}

// Offset sets the offset
func (q *ModelQueryImpl) Offset(offset int) types.ModelQuery {
	newQuery := q.clone()
	newQuery.offset = &offset
	return newQuery
}

// FindMany executes the query and returns multiple results
func (q *ModelQueryImpl) FindMany(ctx context.Context, dest interface{}) error {
	selectQuery := q.Select()
	return selectQuery.FindMany(ctx, dest)
}

// FindUnique executes the query and returns a single unique result
func (q *ModelQueryImpl) FindUnique(ctx context.Context, dest interface{}) error {
	selectQuery := q.Select().Limit(1)
	return selectQuery.FindFirst(ctx, dest)
}

// FindFirst executes the query and returns the first result
func (q *ModelQueryImpl) FindFirst(ctx context.Context, dest interface{}) error {
	selectQuery := q.Select().Limit(1)
	return selectQuery.FindFirst(ctx, dest)
}

// Count returns the count of matching records
func (q *ModelQueryImpl) Count(ctx context.Context) (int64, error) {
	selectQuery := q.Select()
	return selectQuery.Count(ctx)
}

// Exists checks if any matching records exist
func (q *ModelQueryImpl) Exists(ctx context.Context) (bool, error) {
	count, err := q.Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Aggregation methods
func (q *ModelQueryImpl) Sum(ctx context.Context, fieldName string) (float64, error) {
	// Implementation will be added when we implement the SQL builder
	return 0, fmt.Errorf("sum aggregation not yet implemented")
}

func (q *ModelQueryImpl) Avg(ctx context.Context, fieldName string) (float64, error) {
	return 0, fmt.Errorf("avg aggregation not yet implemented")
}

func (q *ModelQueryImpl) Max(ctx context.Context, fieldName string) (interface{}, error) {
	return nil, fmt.Errorf("max aggregation not yet implemented")
}

func (q *ModelQueryImpl) Min(ctx context.Context, fieldName string) (interface{}, error) {
	return nil, fmt.Errorf("min aggregation not yet implemented")
}

// clone creates a copy of the query
func (q *ModelQueryImpl) clone() *ModelQueryImpl {
	newQuery := &ModelQueryImpl{
		modelName:   q.modelName,
		database:    q.database,
		fieldMapper: q.fieldMapper,
		conditions:  make([]types.Condition, len(q.conditions)),
		includes:    make([]string, len(q.includes)),
		orderBy:     make([]OrderClause, len(q.orderBy)),
		groupBy:     make([]string, len(q.groupBy)),
		having:      q.having,
	}

	copy(newQuery.conditions, q.conditions)
	copy(newQuery.includes, q.includes)
	copy(newQuery.orderBy, q.orderBy)
	copy(newQuery.groupBy, q.groupBy)

	if q.limit != nil {
		limit := *q.limit
		newQuery.limit = &limit
	}
	if q.offset != nil {
		offset := *q.offset
		newQuery.offset = &offset
	}

	return newQuery
}

// GetConditions returns all conditions (for internal use by query builders)
func (q *ModelQueryImpl) GetConditions() []types.Condition {
	return q.conditions
}

// GetIncludes returns includes (for internal use)
func (q *ModelQueryImpl) GetIncludes() []string {
	return q.includes
}

// GetOrderBy returns order by clauses (for internal use)
func (q *ModelQueryImpl) GetOrderBy() []OrderClause {
	return q.orderBy
}

// GetGroupBy returns group by fields (for internal use)
func (q *ModelQueryImpl) GetGroupBy() []string {
	return q.groupBy
}

// GetHaving returns having condition (for internal use)
func (q *ModelQueryImpl) GetHaving() types.Condition {
	return q.having
}

// GetLimit returns limit (for internal use)
func (q *ModelQueryImpl) GetLimit() *int {
	return q.limit
}

// GetOffset returns offset (for internal use)
func (q *ModelQueryImpl) GetOffset() *int {
	return q.offset
}

// GetDatabase returns the database (for internal use)
func (q *ModelQueryImpl) GetDatabase() types.Database {
	return q.database
}

// GetFieldMapper returns the field mapper (for internal use)
func (q *ModelQueryImpl) GetFieldMapper() types.FieldMapper {
	return q.fieldMapper
}
