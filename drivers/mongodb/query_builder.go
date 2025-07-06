package mongodb

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBQueryBuilder converts SQL-like queries to MongoDB operations
type MongoDBQueryBuilder struct {
	db *MongoDB
}

// NewMongoDBQueryBuilder creates a new query builder
func NewMongoDBQueryBuilder(db *MongoDB) *MongoDBQueryBuilder {
	return &MongoDBQueryBuilder{db: db}
}

// ConditionToFilter converts a types.Condition to MongoDB filter
func (qb *MongoDBQueryBuilder) ConditionToFilter(condition types.Condition, modelName string) (bson.M, error) {
	if condition == nil || qb == nil {
		return bson.M{}, nil
	}

	// For MongoDB conditions, get the JSON filter directly
	sqlStr, _ := condition.ToSQL(nil)

	// If the "SQL" is actually JSON (starts with {), parse it as MongoDB filter
	if strings.HasPrefix(strings.TrimSpace(sqlStr), "{") {
		var filter bson.M
		if err := json.Unmarshal([]byte(sqlStr), &filter); err != nil {
			return nil, fmt.Errorf("failed to parse MongoDB filter JSON: %w", err)
		}
		return filter, nil
	}

	// Fallback for non-MongoDB conditions
	ctx := &MongoDBConditionContext{
		ModelName:    modelName,
		QueryBuilder: qb,
	}

	return qb.conditionToFilterInternal(condition, ctx)
}

// conditionToFilterInternal recursively converts conditions
func (qb *MongoDBQueryBuilder) conditionToFilterInternal(condition types.Condition, ctx *MongoDBConditionContext) (bson.M, error) {
	if condition == nil {
		return bson.M{}, nil
	}

	// Check if it's a MongoDB-specific condition first
	switch c := condition.(type) {
	case *MongoDBCondition:
		return qb.handleMongoDBCondition(c, ctx)
	case *MongoDBAndCondition:
		return qb.handleMongoDBAndCondition(c, ctx)
	case *MongoDBOrCondition:
		return qb.handleMongoDBOrCondition(c, ctx)
	case *MongoDBNotCondition:
		return qb.handleMongoDBNotCondition(c, ctx)
	case *types.AndCondition:
		return qb.handleAndCondition(c, ctx)
	case *types.OrCondition:
		return qb.handleOrCondition(c, ctx)
	case *types.NotCondition:
		return qb.handleNotCondition(c, ctx)
	case *types.MappedFieldCondition:
		return qb.handleMappedFieldCondition(c, ctx)
	default:
		// Try to convert using the condition's ToSQL method and parse it
		if ctx == nil || ctx.ModelName == "" || qb.db == nil {
			return bson.M{"$comment": "invalid context"}, nil
		}
		
		sqlCtx := &types.ConditionContext{
			ModelName:   ctx.ModelName,
			FieldMapper: qb.db.GetFieldMapper(),
		}
		sql, _ := condition.ToSQL(sqlCtx)
		// This is a fallback - ideally we'd have MongoDB-specific condition types
		return bson.M{"$comment": sql}, nil
	}
}

// handleMongoDBCondition handles MongoDB-specific conditions
func (qb *MongoDBQueryBuilder) handleMongoDBCondition(cond *MongoDBCondition, _ *MongoDBConditionContext) (bson.M, error) {
	if cond == nil {
		return bson.M{}, nil
	}
	
	// Get the JSON from the MongoDB condition
	sqlStr, _ := cond.ToSQL(nil)
	
	// Parse the JSON as MongoDB filter
	if strings.HasPrefix(strings.TrimSpace(sqlStr), "{") {
		var filter bson.M
		if err := json.Unmarshal([]byte(sqlStr), &filter); err != nil {
			return nil, fmt.Errorf("failed to parse MongoDB condition JSON: %w", err)
		}
		return filter, nil
	}
	
	return bson.M{"$comment": sqlStr}, nil
}

// handleMongoDBAndCondition handles MongoDB AND conditions
func (qb *MongoDBQueryBuilder) handleMongoDBAndCondition(cond *MongoDBAndCondition, _ *MongoDBConditionContext) (bson.M, error) {
	if cond == nil {
		return bson.M{}, nil
	}
	
	// Get the JSON from the MongoDB AND condition
	sqlStr, _ := cond.ToSQL(nil)
	
	// Parse the JSON as MongoDB filter
	if strings.HasPrefix(strings.TrimSpace(sqlStr), "{") {
		var filter bson.M
		if err := json.Unmarshal([]byte(sqlStr), &filter); err != nil {
			return nil, fmt.Errorf("failed to parse MongoDB AND condition JSON: %w", err)
		}
		return filter, nil
	}
	
	return bson.M{"$comment": sqlStr}, nil
}

// handleMongoDBOrCondition handles MongoDB OR conditions
func (qb *MongoDBQueryBuilder) handleMongoDBOrCondition(cond *MongoDBOrCondition, _ *MongoDBConditionContext) (bson.M, error) {
	if cond == nil {
		return bson.M{}, nil
	}
	
	// Get the JSON from the MongoDB OR condition
	sqlStr, _ := cond.ToSQL(nil)
	
	// Parse the JSON as MongoDB filter
	if strings.HasPrefix(strings.TrimSpace(sqlStr), "{") {
		var filter bson.M
		if err := json.Unmarshal([]byte(sqlStr), &filter); err != nil {
			return nil, fmt.Errorf("failed to parse MongoDB OR condition JSON: %w", err)
		}
		return filter, nil
	}
	
	return bson.M{"$comment": sqlStr}, nil
}

// handleMongoDBNotCondition handles MongoDB NOT conditions
func (qb *MongoDBQueryBuilder) handleMongoDBNotCondition(cond *MongoDBNotCondition, _ *MongoDBConditionContext) (bson.M, error) {
	if cond == nil {
		return bson.M{}, nil
	}
	
	// Get the JSON from the MongoDB NOT condition
	sqlStr, _ := cond.ToSQL(nil)
	
	// Parse the JSON as MongoDB filter
	if strings.HasPrefix(strings.TrimSpace(sqlStr), "{") {
		var filter bson.M
		if err := json.Unmarshal([]byte(sqlStr), &filter); err != nil {
			return nil, fmt.Errorf("failed to parse MongoDB NOT condition JSON: %w", err)
		}
		return filter, nil
	}
	
	return bson.M{"$comment": sqlStr}, nil
}

// handleAndCondition converts AND conditions
func (qb *MongoDBQueryBuilder) handleAndCondition(cond *types.AndCondition, ctx *MongoDBConditionContext) (bson.M, error) {
	if cond == nil || ctx == nil {
		return bson.M{}, nil
	}
	
	conditions := cond.Conditions
	if len(conditions) == 0 {
		return bson.M{}, nil
	}

	andFilters := make([]bson.M, 0, len(conditions))
	for _, c := range conditions {
		if c == nil {
			continue
		}
		filter, err := qb.conditionToFilterInternal(c, ctx)
		if err != nil {
			return nil, err
		}
		if len(filter) > 0 {
			andFilters = append(andFilters, filter)
		}
	}

	if len(andFilters) == 0 {
		return bson.M{}, nil
	}
	if len(andFilters) == 1 {
		return andFilters[0], nil
	}

	return bson.M{"$and": andFilters}, nil
}

// handleOrCondition converts OR conditions
func (qb *MongoDBQueryBuilder) handleOrCondition(cond *types.OrCondition, ctx *MongoDBConditionContext) (bson.M, error) {
	if cond == nil || ctx == nil {
		return bson.M{}, nil
	}
	
	conditions := cond.Conditions
	if len(conditions) == 0 {
		return bson.M{}, nil
	}

	orFilters := make([]bson.M, 0, len(conditions))
	for _, c := range conditions {
		if c == nil {
			continue
		}
		filter, err := qb.conditionToFilterInternal(c, ctx)
		if err != nil {
			return nil, err
		}
		if len(filter) > 0 {
			orFilters = append(orFilters, filter)
		}
	}

	if len(orFilters) == 0 {
		return bson.M{}, nil
	}
	if len(orFilters) == 1 {
		return orFilters[0], nil
	}

	return bson.M{"$or": orFilters}, nil
}

// handleNotCondition converts NOT conditions
func (qb *MongoDBQueryBuilder) handleNotCondition(cond *types.NotCondition, ctx *MongoDBConditionContext) (bson.M, error) {
	if cond == nil || ctx == nil {
		return bson.M{}, nil
	}
	
	innerCond := cond.Condition
	if innerCond == nil {
		return bson.M{}, nil
	}

	filter, err := qb.conditionToFilterInternal(innerCond, ctx)
	if err != nil {
		return nil, err
	}

	// MongoDB doesn't support $not as a top-level operator
	// We need to use $nor for negating entire expressions
	result := bson.M{"$nor": []bson.M{filter}}
	return result, nil
}

// handleMappedFieldCondition converts field-specific conditions
func (qb *MongoDBQueryBuilder) handleMappedFieldCondition(cond *types.MappedFieldCondition, ctx *MongoDBConditionContext) (bson.M, error) {
	if cond == nil || ctx == nil || qb.db == nil {
		return bson.M{}, nil
	}
	
	// Parse the SQL to extract field name, operator and value
	sql := cond.GetSQL()
	args := cond.GetArgs()

	// Extract field name from the condition (more reliable than SQL parsing)
	fieldName := cond.GetFieldName()
	if fieldName == "" {
		// Fallback to parsing from SQL
		parts := strings.Fields(sql)
		if len(parts) < 2 {
			return nil, fmt.Errorf("cannot parse SQL condition: %s", sql)
		}
		fieldName = parts[0]
	}

	// Convert schema field name to column name
	columnName, err := qb.db.GetFieldMapper().SchemaToColumn(ctx.ModelName, fieldName)
	if err != nil {
		// Use field name as-is if mapping fails
		columnName = fieldName
	}

	// Analyze the SQL to determine the operation type
	sqlUpper := strings.ToUpper(sql)

	// Handle different SQL patterns using string matching (more robust than word splitting)
	// NOTE: Check NOT IN before IN to avoid false matches
	if strings.Contains(sqlUpper, "NOT IN (") {
		// Handle NOT IN operation: "name NOT IN (?,?,?)" with args [val1, val2, val3]
		fmt.Printf("[MongoDB Query] NOT IN operation: field=%s, column=%s, args=%v\n", fieldName, columnName, args)
		return bson.M{columnName: bson.M{"$nin": args}}, nil

	} else if strings.Contains(sqlUpper, " IN (") {
		// Handle IN operation: "name IN (?,?,?)" with args [val1, val2, val3]
		fmt.Printf("[MongoDB Query] IN operation: field=%s, column=%s, args=%v\n", fieldName, columnName, args)
		return bson.M{columnName: bson.M{"$in": args}}, nil

	} else if strings.Contains(sqlUpper, " LIKE ") {
		// Handle LIKE operation: "name LIKE ?" with args ["%pattern%"]
		if len(args) > 0 {
			pattern := fmt.Sprintf("%v", args[0])
			// Convert SQL LIKE pattern to regex
			pattern = strings.ReplaceAll(pattern, "%", ".*")
			pattern = strings.ReplaceAll(pattern, "_", ".")
			return bson.M{columnName: bson.M{"$regex": pattern, "$options": "i"}}, nil
		}

	} else if strings.Contains(sqlUpper, " IS NULL") {
		// Handle IS NULL: "field IS NULL"
		return bson.M{columnName: nil}, nil

	} else if strings.Contains(sqlUpper, " IS NOT NULL") {
		// Handle IS NOT NULL: "field IS NOT NULL"
		return bson.M{columnName: bson.M{"$ne": nil}}, nil

	} else if strings.Contains(sqlUpper, " = ") {
		// Handle equality: "field = ?"
		if len(args) > 0 {
			if args[0] == nil {
				return bson.M{columnName: nil}, nil
			}
			return bson.M{columnName: args[0]}, nil
		}
		return bson.M{columnName: nil}, nil

	} else if strings.Contains(sqlUpper, " != ") || strings.Contains(sqlUpper, " <> ") {
		// Handle inequality: "field != ?" or "field <> ?"
		if len(args) > 0 {
			return bson.M{columnName: bson.M{"$ne": args[0]}}, nil
		}
		return bson.M{columnName: bson.M{"$ne": nil}}, nil

	} else if strings.Contains(sqlUpper, " > ") {
		// Handle greater than: "field > ?"
		if len(args) > 0 {
			return bson.M{columnName: bson.M{"$gt": args[0]}}, nil
		}

	} else if strings.Contains(sqlUpper, " >= ") {
		// Handle greater than or equal: "field >= ?"
		if len(args) > 0 {
			return bson.M{columnName: bson.M{"$gte": args[0]}}, nil
		}

	} else if strings.Contains(sqlUpper, " < ") {
		// Handle less than: "field < ?"
		if len(args) > 0 {
			return bson.M{columnName: bson.M{"$lt": args[0]}}, nil
		}

	} else if strings.Contains(sqlUpper, " <= ") {
		// Handle less than or equal: "field <= ?"
		if len(args) > 0 {
			return bson.M{columnName: bson.M{"$lte": args[0]}}, nil
		}
	}

	// Fallback - return as comment
	return bson.M{"$comment": fmt.Sprintf("%s %v", sql, args)}, nil
}

// ConvertOrderBy converts order by fields to MongoDB sort
func (qb *MongoDBQueryBuilder) ConvertOrderBy(orderBys []types.OrderByClause, modelName string) (bson.D, error) {
	if len(orderBys) == 0 {
		return nil, nil
	}

	sort := bson.D{}
	for _, ob := range orderBys {
		columnName, err := qb.db.GetFieldMapper().SchemaToColumn(modelName, ob.Field)
		if err != nil {
			// If field mapping fails, use the field name as-is
			// This provides fallback behavior for schema lookup issues
			columnName = ob.Field
		}

		direction := 1
		if ob.Direction == types.DESC {
			direction = -1
		}

		sort = append(sort, bson.E{Key: columnName, Value: direction})
	}

	return sort, nil
}

// ConvertProjection converts selected fields to MongoDB projection
func (qb *MongoDBQueryBuilder) ConvertProjection(fields []string, modelName string) (bson.M, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	projection := bson.M{}
	for _, field := range fields {
		if field == "*" {
			// Return nil to select all fields
			return nil, nil
		}

		columnName, err := qb.db.GetFieldMapper().SchemaToColumn(modelName, field)
		if err != nil {
			// If field mapping fails, use the field as-is
			columnName = field
		}

		projection[columnName] = 1
	}

	// Always include _id unless explicitly excluded
	if _, hasID := projection["_id"]; !hasID {
		projection["_id"] = 1
	}

	return projection, nil
}

// MongoDBConditionContext provides context for condition conversion
type MongoDBConditionContext struct {
	ModelName    string
	QueryBuilder *MongoDBQueryBuilder
}
