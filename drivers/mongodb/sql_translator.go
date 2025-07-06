package mongodb

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/sql"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MongoDBSQLTranslator converts SQL AST to MongoDB commands
type MongoDBSQLTranslator struct {
	db   *MongoDB
	args []any // SQL parameters for substitution
}

// NewMongoDBSQLTranslator creates a new MongoDB SQL translator
func NewMongoDBSQLTranslator(db *MongoDB) *MongoDBSQLTranslator {
	return &MongoDBSQLTranslator{
		db:   db,
		args: nil,
	}
}

// SetArgs sets the SQL parameters for substitution
func (t *MongoDBSQLTranslator) SetArgs(args []any) {
	t.args = args
}

// substituteValue replaces "?" with the next parameter value
func (t *MongoDBSQLTranslator) substituteValue(value any, argIndex *int) any {
	if str, ok := value.(string); ok && str == "?" {
		if *argIndex < len(t.args) {
			result := t.args[*argIndex]
			(*argIndex)++
			return result
		}
	}
	return value
}

// TranslateToCommand translates SQL AST to MongoDB command
func (t *MongoDBSQLTranslator) TranslateToCommand(stmt sql.SQLStatement) (*MongoDBCommand, error) {
	switch s := stmt.(type) {
	case *sql.SelectStatement:
		return t.translateSelect(s)
	case *sql.InsertStatement:
		return t.translateInsert(s)
	case *sql.UpdateStatement:
		return t.translateUpdate(s)
	case *sql.DeleteStatement:
		return t.translateDelete(s)
	default:
		return nil, fmt.Errorf("unsupported SQL statement type: %T", stmt)
	}
}

// getCollectionName converts SQL table name to MongoDB collection name
func (t *MongoDBSQLTranslator) getCollectionName(tableName string) string {
	// In SQL, table names are usually already in the correct format
	// Try to find if this table name corresponds to any registered model
	for modelName := range t.db.Schemas {
		if modelTableName, err := t.db.FieldMapper.ModelToTable(modelName); err == nil && modelTableName == tableName {
			// Found matching model for this table name, use the collection name from schema
			if schema, err := t.db.GetSchema(modelName); err == nil {
				return schema.GetTableName()
			}
		}
	}
	// If no matching model found, use table name as-is
	return tableName
}

// collectionExists checks if a collection exists in the registered schemas
func (t *MongoDBSQLTranslator) collectionExists(collectionName string) bool {
	// Check if any registered schema corresponds to this collection
	for modelName := range t.db.Schemas {
		if schema, err := t.db.GetSchema(modelName); err == nil {
			if schema.GetTableName() == collectionName {
				return true
			}
		}
		// Also check if the model name matches the collection directly
		if modelTableName, err := t.db.FieldMapper.ModelToTable(modelName); err == nil && modelTableName == collectionName {
			return true
		}
	}
	return false
}

// validateParameterSubstitution checks if there are any unsubstituted parameters
func (t *MongoDBSQLTranslator) validateParameterSubstitution(pipeline []bson.M) error {
	for _, stage := range pipeline {
		if err := t.checkForUnsubstitutedParams(stage); err != nil {
			return err
		}
	}
	return nil
}

// checkForUnsubstitutedParams recursively checks for "?" values in BSON documents
func (t *MongoDBSQLTranslator) checkForUnsubstitutedParams(doc any) error {
	switch v := doc.(type) {
	case bson.M:
		for key, value := range v {
			if err := t.checkForUnsubstitutedParams(value); err != nil {
				return fmt.Errorf("unsubstituted parameter in field '%s': %w", key, err)
			}
		}
	case []any:
		for i, item := range v {
			if err := t.checkForUnsubstitutedParams(item); err != nil {
				return fmt.Errorf("unsubstituted parameter in array index %d: %w", i, err)
			}
		}
	case []primitive.M:
		for i, item := range v {
			if err := t.checkForUnsubstitutedParams(item); err != nil {
				return fmt.Errorf("unsubstituted parameter in array index %d: %w", i, err)
			}
		}
	case primitive.A:
		for i, item := range v {
			if err := t.checkForUnsubstitutedParams(item); err != nil {
				return fmt.Errorf("unsubstituted parameter in array index %d: %w", i, err)
			}
		}
	case string:
		if v == "?" {
			return fmt.Errorf("unsubstituted parameter '?' found - not enough arguments provided")
		}
	default:
		// Check if it's some other type that represents "?"
		if str := fmt.Sprintf("%v", v); str == "?" {
			return fmt.Errorf("unsubstituted parameter '?' found - not enough arguments provided")
		}
	}
	return nil
}

// translateSelect converts SELECT statement to MongoDB aggregation pipeline
func (t *MongoDBSQLTranslator) translateSelect(stmt *sql.SelectStatement) (*MongoDBCommand, error) {
	pipeline := []bson.M{}

	// Convert table name to collection name
	collection := t.getCollectionName(stmt.From.Table)

	// Validate that the collection/table exists in the schema (only if schemas are registered)
	if t.db != nil && len(t.db.Schemas) > 0 && !t.collectionExists(collection) {
		return nil, fmt.Errorf("collection '%s' does not exist", collection)
	}

	// WHERE clause → $match stage (before joins for optimization)
	argIndex := 0
	if stmt.Where != nil {
		matchStage, err := t.translateWhereToMatchWithArgs(stmt.Where, &argIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to translate WHERE clause: %w", err)
		}
		if len(matchStage) > 0 {
			pipeline = append(pipeline, bson.M{"$match": matchStage})
		}
	}

	// JOIN → $lookup stages (must be before GROUP BY)
	for _, join := range stmt.Joins {
		lookupStage, err := t.translateJoinToLookup(join)
		if err != nil {
			return nil, fmt.Errorf("failed to translate JOIN: %w", err)
		}
		pipeline = append(pipeline, lookupStage...)
	}

	// Check if this query uses both GROUP BY and aggregation functions
	hasGroupBy := len(stmt.GroupBy) > 0
	isAggregation := t.isAggregationQuery(stmt)

	// GROUP BY with aggregation → $group stage
	if hasGroupBy && isAggregation {
		groupStage, err := t.translateGroupByWithAggregation(stmt.GroupBy, stmt.Fields, stmt.From, stmt.Joins)
		if err != nil {
			return nil, fmt.Errorf("failed to translate GROUP BY with aggregation: %w", err)
		}
		pipeline = append(pipeline, bson.M{"$group": groupStage})

		// Add $project stage to restructure the result from GROUP BY
		projectStage, err := t.translateGroupByProject(stmt.GroupBy, stmt.Fields)
		if err != nil {
			return nil, fmt.Errorf("failed to create GROUP BY projection: %w", err)
		}
		if len(projectStage) > 0 {
			pipeline = append(pipeline, bson.M{"$project": projectStage})
		}
	} else if hasGroupBy {
		// GROUP BY without aggregation functions → $group stage with first values
		groupStage, err := t.translateGroupBy(stmt.GroupBy)
		if err != nil {
			return nil, fmt.Errorf("failed to translate GROUP BY: %w", err)
		}
		pipeline = append(pipeline, bson.M{"$group": groupStage})

		// Add $project stage to restructure the result from GROUP BY
		projectStage, err := t.translateGroupByProject(stmt.GroupBy, stmt.Fields)
		if err != nil {
			return nil, fmt.Errorf("failed to create GROUP BY projection: %w", err)
		}
		if len(projectStage) > 0 {
			pipeline = append(pipeline, bson.M{"$project": projectStage})
		}
	}

	// HAVING → $match stage (after group)
	if stmt.Having != nil {
		havingStage, err := t.translateHavingClause(stmt.Having, stmt.Fields, &argIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to translate HAVING clause: %w", err)
		}
		if len(havingStage) > 0 {
			pipeline = append(pipeline, bson.M{"$match": havingStage})
		}
	}

	// ORDER BY → $sort stage
	if len(stmt.OrderBy) > 0 {
		sortStage, err := t.translateOrderBy(stmt.OrderBy)
		if err != nil {
			return nil, fmt.Errorf("failed to translate ORDER BY: %w", err)
		}
		pipeline = append(pipeline, bson.M{"$sort": sortStage})
	}

	// OFFSET → $skip stage
	if stmt.Offset != nil && *stmt.Offset > 0 {
		pipeline = append(pipeline, bson.M{"$skip": *stmt.Offset})
	}

	// LIMIT → $limit stage
	if stmt.Limit != nil && *stmt.Limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": *stmt.Limit})
	}

	// Handle aggregation queries without explicit GROUP BY (e.g., SELECT COUNT(*) FROM table)
	if isAggregation && !hasGroupBy {
		// Add $group stage for aggregation without GROUP BY
		groupStage, err := t.translateAggregationFields(stmt.Fields)
		if err != nil {
			return nil, fmt.Errorf("failed to translate aggregation fields: %w", err)
		}
		pipeline = append(pipeline, bson.M{"$group": groupStage})

		// Add $project stage to rename _id to null and expose aggregation results
		projectStage := bson.M{"_id": 0}
		for _, field := range stmt.Fields {
			alias := field.Alias
			if alias == "" {
				alias = field.Expression
			}
			projectStage[alias] = "$" + alias
		}
		pipeline = append(pipeline, bson.M{"$project": projectStage})
	} else if !isAggregation {
		// SELECT fields → $project stage (add at the end if not SELECT *)
		if !sql.IsSelectAll(stmt.Fields) {
			projectStage, err := t.translateProject(stmt.Fields, stmt.From, stmt.Joins)
			if err != nil {
				return nil, fmt.Errorf("failed to translate SELECT fields: %w", err)
			}
			pipeline = append(pipeline, bson.M{"$project": projectStage})
		}
	}

	// If no pipeline stages, use simple find
	if len(pipeline) == 0 {
		// Still need to validate parameters even for simple find
		emptyPipeline := []bson.M{}
		if stmt.Where != nil {
			// Check if WHERE clause has any conditions that might contain unsubstituted parameters
			whereMatch, err := t.translateWhereToMatchWithArgs(stmt.Where, &argIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to validate WHERE clause: %w", err)
			}
			if len(whereMatch) > 0 {
				emptyPipeline = append(emptyPipeline, bson.M{"$match": whereMatch})
			}
		}
		if err := t.validateParameterSubstitution(emptyPipeline); err != nil {
			return nil, err
		}
		
		return &MongoDBCommand{
			Operation:  "find",
			Collection: collection,
		}, nil
	}

	// Validate that all parameters were substituted
	if err := t.validateParameterSubstitution(pipeline); err != nil {
		return nil, err
	}


	return &MongoDBCommand{
		Operation:  "aggregate",
		Collection: collection,
		Pipeline:   pipeline,
	}, nil
}

// translateWhereToMatch converts WHERE clause to MongoDB $match criteria
func (t *MongoDBSQLTranslator) translateWhereToMatch(where *sql.WhereClause) (bson.M, error) {
	if where == nil {
		return bson.M{}, nil
	}

	// Handle logical operators
	if where.Operator != "" {
		switch strings.ToUpper(where.Operator) {
		case "AND":
			leftMatch, err := t.translateWhereToMatch(where.Left)
			if err != nil {
				return nil, err
			}
			rightMatch, err := t.translateWhereToMatch(where.Right)
			if err != nil {
				return nil, err
			}
			return bson.M{"$and": []bson.M{leftMatch, rightMatch}}, nil

		case "OR":
			leftMatch, err := t.translateWhereToMatch(where.Left)
			if err != nil {
				return nil, err
			}
			rightMatch, err := t.translateWhereToMatch(where.Right)
			if err != nil {
				return nil, err
			}
			return bson.M{"$or": []bson.M{leftMatch, rightMatch}}, nil

		case "NOT":
			leftMatch, err := t.translateWhereToMatch(where.Left)
			if err != nil {
				return nil, err
			}
			return bson.M{"$not": leftMatch}, nil
		}
	}

	// Handle leaf condition
	if where.Condition != nil {
		return t.translateCondition(where.Condition)
	}

	return bson.M{}, nil
}

// translateWhereToMatchWithArgs converts WHERE clause to MongoDB $match criteria with parameter substitution
func (t *MongoDBSQLTranslator) translateWhereToMatchWithArgs(where *sql.WhereClause, argIndex *int) (bson.M, error) {
	if where == nil {
		return bson.M{}, nil
	}

	// Handle logical operators
	if where.Operator != "" {
		switch strings.ToUpper(where.Operator) {
		case "AND":
			leftMatch, err := t.translateWhereToMatchWithArgs(where.Left, argIndex)
			if err != nil {
				return nil, err
			}
			rightMatch, err := t.translateWhereToMatchWithArgs(where.Right, argIndex)
			if err != nil {
				return nil, err
			}
			return bson.M{"$and": []bson.M{leftMatch, rightMatch}}, nil

		case "OR":
			leftMatch, err := t.translateWhereToMatchWithArgs(where.Left, argIndex)
			if err != nil {
				return nil, err
			}
			rightMatch, err := t.translateWhereToMatchWithArgs(where.Right, argIndex)
			if err != nil {
				return nil, err
			}
			return bson.M{"$or": []bson.M{leftMatch, rightMatch}}, nil

		case "NOT":
			leftMatch, err := t.translateWhereToMatchWithArgs(where.Left, argIndex)
			if err != nil {
				return nil, err
			}
			return bson.M{"$not": leftMatch}, nil
		}
	}

	// Handle leaf condition
	if where.Condition != nil {
		return t.translateConditionWithArgs(where.Condition, argIndex)
	}

	return bson.M{}, nil
}

// translateHavingClause converts HAVING clause to MongoDB $match criteria with SELECT field context
func (t *MongoDBSQLTranslator) translateHavingClause(having *sql.WhereClause, selectFields []sql.SelectField, argIndex *int) (bson.M, error) {
	if having == nil {
		return bson.M{}, nil
	}

	// Handle logical operators
	if having.Operator != "" {
		switch strings.ToUpper(having.Operator) {
		case "AND":
			leftMatch, err := t.translateHavingClause(having.Left, selectFields, argIndex)
			if err != nil {
				return nil, err
			}
			rightMatch, err := t.translateHavingClause(having.Right, selectFields, argIndex)
			if err != nil {
				return nil, err
			}
			return bson.M{"$and": []bson.M{leftMatch, rightMatch}}, nil

		case "OR":
			leftMatch, err := t.translateHavingClause(having.Left, selectFields, argIndex)
			if err != nil {
				return nil, err
			}
			rightMatch, err := t.translateHavingClause(having.Right, selectFields, argIndex)
			if err != nil {
				return nil, err
			}
			return bson.M{"$or": []bson.M{leftMatch, rightMatch}}, nil

		case "NOT":
			leftMatch, err := t.translateHavingClause(having.Left, selectFields, argIndex)
			if err != nil {
				return nil, err
			}
			return bson.M{"$not": leftMatch}, nil
		}
	}

	// Handle leaf condition
	if having.Condition != nil {
		return t.translateConditionWithHavingContext(having.Condition, selectFields, argIndex)
	}

	return bson.M{}, nil
}

// translateConditionWithHavingContext converts a single condition to MongoDB criteria with HAVING context
func (t *MongoDBSQLTranslator) translateConditionWithHavingContext(cond *sql.Condition, selectFields []sql.SelectField, argIndex *int) (bson.M, error) {
	// Check if this is a function call (for HAVING clauses)
	if t.isFunctionCall(cond.Field) {
		return t.translateFunctionCondition(cond, selectFields, argIndex)
	}

	// For non-function conditions, use the regular logic
	return t.translateConditionWithArgs(cond, argIndex)
}

// translateCondition converts a single condition to MongoDB criteria
func (t *MongoDBSQLTranslator) translateCondition(cond *sql.Condition) (bson.M, error) {
	// Map schema field name to database column name
	fieldName, err := t.mapFieldName(cond.Field)
	if err != nil {
		fieldName = cond.Field // fallback to original name
	}

	switch strings.ToUpper(cond.Operator) {
	case "=":
		return bson.M{fieldName: cond.Value}, nil
	case "!=", "<>":
		return bson.M{fieldName: bson.M{"$ne": cond.Value}}, nil
	case ">":
		return bson.M{fieldName: bson.M{"$gt": cond.Value}}, nil
	case ">=":
		return bson.M{fieldName: bson.M{"$gte": cond.Value}}, nil
	case "<":
		return bson.M{fieldName: bson.M{"$lt": cond.Value}}, nil
	case "<=":
		return bson.M{fieldName: bson.M{"$lte": cond.Value}}, nil
	case "LIKE":
		// Convert SQL LIKE pattern to MongoDB regex
		pattern := t.convertLikeToRegex(cond.Value.(string))
		return bson.M{fieldName: bson.M{"$regex": pattern, "$options": "i"}}, nil
	case "IN":
		return bson.M{fieldName: bson.M{"$in": cond.Values}}, nil
	case "NOT IN":
		return bson.M{fieldName: bson.M{"$nin": cond.Values}}, nil
	case "IS NULL":
		return bson.M{fieldName: bson.M{"$eq": nil}}, nil
	case "IS NOT NULL":
		return bson.M{fieldName: bson.M{"$ne": nil}}, nil
	default:
		return nil, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}
}

// translateConditionWithArgs converts a single condition to MongoDB criteria with parameter substitution
func (t *MongoDBSQLTranslator) translateConditionWithArgs(cond *sql.Condition, argIndex *int) (bson.M, error) {
	// Check if this is a function call (for HAVING clauses)
	if t.isFunctionCall(cond.Field) {
		// For HAVING clauses, this should go through translateConditionWithHavingContext
		// This is a fallback for non-HAVING function calls (which shouldn't happen normally)
		return nil, fmt.Errorf("function calls should be handled through HAVING clause translation")
	}

	// Map schema field name to database column name
	fieldName, err := t.mapFieldName(cond.Field)
	if err != nil {
		fieldName = cond.Field // fallback to original name
	}

	switch strings.ToUpper(cond.Operator) {
	case "=":
		value := t.substituteValue(cond.Value, argIndex)
		return bson.M{fieldName: value}, nil
	case "!=", "<>":
		value := t.substituteValue(cond.Value, argIndex)
		return bson.M{fieldName: bson.M{"$ne": value}}, nil
	case ">":
		value := t.substituteValue(cond.Value, argIndex)
		return bson.M{fieldName: bson.M{"$gt": value}}, nil
	case ">=":
		value := t.substituteValue(cond.Value, argIndex)
		return bson.M{fieldName: bson.M{"$gte": value}}, nil
	case "<":
		value := t.substituteValue(cond.Value, argIndex)
		return bson.M{fieldName: bson.M{"$lt": value}}, nil
	case "<=":
		value := t.substituteValue(cond.Value, argIndex)
		return bson.M{fieldName: bson.M{"$lte": value}}, nil
	case "LIKE":
		// Convert SQL LIKE pattern to MongoDB regex
		value := t.substituteValue(cond.Value, argIndex)
		pattern := t.convertLikeToRegex(value.(string))
		return bson.M{fieldName: bson.M{"$regex": pattern, "$options": "i"}}, nil
	case "IN":
		if cond.Subquery != nil {
			// Handle subquery: field IN (SELECT ...)
			// We need to execute the subquery and get its results
			return t.translateSubqueryCondition(fieldName, cond.Subquery, argIndex, false)
		} else {
			// For IN clause, substitute all values in the array
			values := make([]any, len(cond.Values))
			for i, v := range cond.Values {
				values[i] = t.substituteValue(v, argIndex)
			}
			return bson.M{fieldName: bson.M{"$in": values}}, nil
		}
	case "NOT IN":
		// For NOT IN clause, substitute all values in the array
		values := make([]any, len(cond.Values))
		for i, v := range cond.Values {
			values[i] = t.substituteValue(v, argIndex)
		}
		return bson.M{fieldName: bson.M{"$nin": values}}, nil
	case "IS NULL":
		return bson.M{fieldName: bson.M{"$eq": nil}}, nil
	case "IS NOT NULL":
		return bson.M{fieldName: bson.M{"$ne": nil}}, nil
	default:
		return nil, fmt.Errorf("unsupported operator: %s", cond.Operator)
	}
}

// translateOrderBy converts ORDER BY to MongoDB $sort
func (t *MongoDBSQLTranslator) translateOrderBy(orderBy []*sql.OrderByClause) (bson.M, error) {
	sortStage := bson.M{}

	for _, clause := range orderBy {
		fieldName, err := t.mapFieldName(clause.Field)
		if err != nil {
			fieldName = clause.Field // fallback
		}

		direction := 1 // ASC
		if clause.Direction == sql.OrderDirectionDesc {
			direction = -1 // DESC
		}

		sortStage[fieldName] = direction
	}

	return sortStage, nil
}

// translateGroupBy converts GROUP BY to MongoDB $group
func (t *MongoDBSQLTranslator) translateGroupBy(groupBy []string) (bson.M, error) {
	groupStage := bson.M{
		"_id": bson.M{},
	}

	for _, field := range groupBy {
		fieldName, err := t.mapFieldName(field)
		if err != nil {
			fieldName = field // fallback
		}
		groupStage["_id"].(bson.M)[field] = "$" + fieldName
	}

	return groupStage, nil
}

// translateGroupByWithAggregation converts GROUP BY with aggregation functions to MongoDB $group
func (t *MongoDBSQLTranslator) translateGroupByWithAggregation(groupBy []string, fields []sql.SelectField, fromTable sql.TableRef, _ []*sql.JoinClause) (bson.M, error) {
	groupStage := bson.M{}

	// Set up grouping fields in _id
	if len(groupBy) == 1 {
		// Single field grouping
		field := groupBy[0]
		fieldName, err := t.mapFieldName(field)
		if err != nil {
			fieldName = field // fallback
		}
		groupStage["_id"] = "$" + fieldName
	} else {
		// Multiple field grouping
		idObj := bson.M{}
		for _, field := range groupBy {
			fieldName, err := t.mapFieldName(field)
			if err != nil {
				fieldName = field // fallback
			}

			// Use a safe key name for the _id object (replace dots with underscores)
			safeKey := strings.ReplaceAll(field, ".", "_")

			// For qualified field names in JOIN context, construct proper field reference
			if strings.Contains(field, ".") {
				parts := strings.Split(field, ".")
				if len(parts) == 2 {
					tableAlias := parts[0]
					actualFieldName := parts[1]
					if actualFieldName == "id" {
						actualFieldName = "_id"
					}

					// Check if this is referencing the main table or a joined table
					if tableAlias == fromTable.Alias || (fromTable.Alias == "" && tableAlias == fromTable.Table) {
						// This is the main table - fields are at root level after unwind
						idObj[safeKey] = "$" + actualFieldName
					} else {
						// This is a joined table - use the join alias
						idObj[safeKey] = "$" + tableAlias + "." + actualFieldName
					}
				} else {
					idObj[safeKey] = "$" + fieldName
				}
			} else {
				idObj[safeKey] = "$" + fieldName
			}
		}
		groupStage["_id"] = idObj
	}

	// Add aggregation functions
	for _, field := range fields {
		// Skip non-aggregation fields that are part of GROUP BY
		isGroupByField := false
		for _, gbField := range groupBy {
			if field.Expression == gbField {
				isGroupByField = true
				break
			}
		}
		if isGroupByField {
			continue
		}

		alias := field.Alias
		if alias == "" {
			alias = field.Expression
		}

		expr := strings.ToUpper(field.Expression)
		switch {
		case strings.HasPrefix(expr, "COUNT("):
			// COUNT(*) or COUNT(field)
			if expr == "COUNT(*)" {
				groupStage[alias] = bson.M{"$sum": 1}
			} else {
				// Extract field name from COUNT(field)
				fieldName := strings.TrimPrefix(field.Expression, "COUNT(")
				fieldName = strings.TrimSuffix(fieldName, ")")
				mappedField, err := t.mapFieldName(fieldName)
				if err != nil {
					mappedField = fieldName
				}
				groupStage[alias] = bson.M{"$sum": bson.M{"$cond": []any{
					bson.M{"$ne": []any{"$" + mappedField, nil}}, 1, 0,
				}}}
			}
		case strings.HasPrefix(expr, "SUM("):
			fieldName := strings.TrimPrefix(field.Expression, "SUM(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$sum": "$" + mappedField}
		case strings.HasPrefix(expr, "AVG("):
			fieldName := strings.TrimPrefix(field.Expression, "AVG(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$avg": "$" + mappedField}
		case strings.HasPrefix(expr, "MIN("):
			fieldName := strings.TrimPrefix(field.Expression, "MIN(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$min": "$" + mappedField}
		case strings.HasPrefix(expr, "MAX("):
			fieldName := strings.TrimPrefix(field.Expression, "MAX(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$max": "$" + mappedField}
		}
	}

	return groupStage, nil
}

// translateProject converts SELECT fields to MongoDB $project
func (t *MongoDBSQLTranslator) translateProject(fields []sql.SelectField, fromTable sql.TableRef, joins []*sql.JoinClause) (bson.M, error) {
	projectStage := bson.M{}

	for _, field := range fields {
		if field.Expression == "*" {
			// SELECT * - include all fields
			return bson.M{}, nil
		}

		// Use alias if provided, otherwise extract field name without table prefix
		outputName := field.Expression
		if field.Alias != "" {
			outputName = field.Alias
		} else if strings.Contains(field.Expression, ".") {
			// For qualified fields like "u.name", use just the field name "name"
			parts := strings.Split(field.Expression, ".")
			if len(parts) == 2 {
				outputName = parts[1] // Use the field name part
			}
		}

		// Handle qualified field names for JOIN queries
		var fieldRef string
		if strings.Contains(field.Expression, ".") {
			parts := strings.Split(field.Expression, ".")
			if len(parts) == 2 {
				tableAlias := parts[0]
				fieldName := parts[1]

				// Map field name to database column name
				if fieldName == "id" {
					fieldName = "_id"
				}

				// Check if this is referencing the main table or a joined table
				if tableAlias == fromTable.Alias || (fromTable.Alias == "" && tableAlias == fromTable.Table) {
					// This is the main table - fields are at root level
					fieldRef = "$" + fieldName
				} else {
					// This is a joined table - check if it's a valid join alias
					isValidJoinAlias := false
					for _, join := range joins {
						if join.Table.Alias == tableAlias || (join.Table.Alias == "" && join.Table.Table == tableAlias) {
							isValidJoinAlias = true
							break
						}
					}

					if isValidJoinAlias {
						// This is a joined table - use the join alias
						fieldRef = "$" + tableAlias + "." + fieldName
					} else {
						// Fallback to root level
						fieldRef = "$" + fieldName
					}
				}
			} else {
				fieldRef = "$" + field.Expression
			}
		} else {
			// Non-qualified field name
			fieldName, err := t.mapFieldName(field.Expression)
			if err != nil {
				fieldName = field.Expression // fallback
			}
			fieldRef = "$" + fieldName
		}

		projectStage[outputName] = fieldRef
	}

	return projectStage, nil
}

// isAggregationQuery checks if the query contains aggregation functions
func (t *MongoDBSQLTranslator) isAggregationQuery(stmt *sql.SelectStatement) bool {
	for _, field := range stmt.Fields {
		expr := strings.ToUpper(field.Expression)
		if strings.HasPrefix(expr, "COUNT(") || strings.HasPrefix(expr, "SUM(") ||
			strings.HasPrefix(expr, "AVG(") || strings.HasPrefix(expr, "MIN(") ||
			strings.HasPrefix(expr, "MAX(") {
			return true
		}
	}
	return false
}

// translateAggregationFields converts aggregation functions to MongoDB $group stage
func (t *MongoDBSQLTranslator) translateAggregationFields(fields []sql.SelectField) (bson.M, error) {
	groupStage := bson.M{"_id": nil} // Group all documents together

	for _, field := range fields {
		alias := field.Alias
		if alias == "" {
			alias = field.Expression
		}

		expr := strings.ToUpper(field.Expression)
		switch {
		case strings.HasPrefix(expr, "COUNT("):
			// COUNT(*) or COUNT(field)
			if expr == "COUNT(*)" {
				groupStage[alias] = bson.M{"$sum": 1}
			} else {
				// Extract field name from COUNT(field)
				fieldName := strings.TrimPrefix(field.Expression, "COUNT(")
				fieldName = strings.TrimSuffix(fieldName, ")")
				mappedField, err := t.mapFieldName(fieldName)
				if err != nil {
					mappedField = fieldName
				}
				groupStage[alias] = bson.M{"$sum": bson.M{"$cond": []any{
					bson.M{"$ne": []any{"$" + mappedField, nil}}, 1, 0,
				}}}
			}
		case strings.HasPrefix(expr, "SUM("):
			fieldName := strings.TrimPrefix(field.Expression, "SUM(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$sum": "$" + mappedField}
		case strings.HasPrefix(expr, "AVG("):
			fieldName := strings.TrimPrefix(field.Expression, "AVG(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$avg": "$" + mappedField}
		case strings.HasPrefix(expr, "MIN("):
			fieldName := strings.TrimPrefix(field.Expression, "MIN(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$min": "$" + mappedField}
		case strings.HasPrefix(expr, "MAX("):
			fieldName := strings.TrimPrefix(field.Expression, "MAX(")
			fieldName = strings.TrimSuffix(fieldName, ")")
			mappedField, err := t.mapFieldName(fieldName)
			if err != nil {
				mappedField = fieldName
			}
			groupStage[alias] = bson.M{"$max": "$" + mappedField}
		}
	}

	return groupStage, nil
}

// translateInsert converts INSERT statement to MongoDB command
func (t *MongoDBSQLTranslator) translateInsert(stmt *sql.InsertStatement) (*MongoDBCommand, error) {
	// Convert table name to collection name
	collection := t.getCollectionName(stmt.Table)

	var documents []any
	argIndex := 0

	for _, values := range stmt.Values {
		doc := bson.M{}

		if len(stmt.Fields) > 0 {
			// Use provided field names
			for i, field := range stmt.Fields {
				if i < len(values) {
					fieldName, err := t.mapFieldName(field)
					if err != nil {
						fieldName = field // fallback
					}
					// Substitute parameters in VALUES
					substitutedValue := t.substituteValue(values[i], &argIndex)
					doc[fieldName] = substitutedValue
				}
			}
		} else {
			// No field names provided - this is problematic for MongoDB
			return nil, fmt.Errorf("INSERT without field list is not supported for MongoDB")
		}

		documents = append(documents, doc)
	}

	return &MongoDBCommand{
		Operation:  "insert",
		Collection: collection,
		Documents:  documents,
	}, nil
}

// translateUpdate converts UPDATE statement to MongoDB command
func (t *MongoDBSQLTranslator) translateUpdate(stmt *sql.UpdateStatement) (*MongoDBCommand, error) {
	// Convert table name to collection name
	collection := t.getCollectionName(stmt.Table)

	// Build update document with parameter substitution
	updateDoc := bson.M{"$set": bson.M{}}
	argIndex := 0
	for field, value := range stmt.Set {
		fieldName, err := t.mapFieldName(field)
		if err != nil {
			fieldName = field // fallback
		}
		// Substitute parameters in SET clause first (as they appear first in SQL)
		substitutedValue := t.substituteValue(value, &argIndex)
		updateDoc["$set"].(bson.M)[fieldName] = substitutedValue
	}

	// Now handle WHERE clause parameters
	filter := bson.M{}
	if stmt.Where != nil {
		var err error
		filter, err = t.translateWhereToMatchWithArgs(stmt.Where, &argIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to translate WHERE clause: %w", err)
		}
	}

	return &MongoDBCommand{
		Operation:  "update",
		Collection: collection,
		Filter:     filter,
		Update:     updateDoc,
	}, nil
}

// translateDelete converts DELETE statement to MongoDB command
func (t *MongoDBSQLTranslator) translateDelete(stmt *sql.DeleteStatement) (*MongoDBCommand, error) {
	// Convert table name to collection name
	collection := t.getCollectionName(stmt.Table)

	// Build filter from WHERE clause
	filter := bson.M{}
	if stmt.Where != nil {
		argIndex := 0 // Start from first argument
		var err error
		filter, err = t.translateWhereToMatchWithArgs(stmt.Where, &argIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to translate WHERE clause: %w", err)
		}
	}

	return &MongoDBCommand{
		Operation:  "delete",
		Collection: collection,
		Filter:     filter,
	}, nil
}

// Helper functions

// mapFieldName maps schema field name to database column name
func (t *MongoDBSQLTranslator) mapFieldName(fieldName string) (string, error) {
	// Handle qualified field names (table.column)
	if strings.Contains(fieldName, ".") {
		parts := strings.Split(fieldName, ".")
		if len(parts) == 2 {
			// For qualified names, only map the field part
			if parts[1] == "id" {
				return "_id", nil
			}
			// For JOIN queries, return just the field name (without table alias)
			// The table alias will be handled in the aggregation pipeline context
			return parts[1], nil
		}
	}

	// For MongoDB, we need to handle the special _id field mapping
	if fieldName == "id" {
		return "_id", nil
	}
	// For other fields, we can use the standard field mapping if available
	// This would require access to schema information
	return fieldName, nil
}

// convertLikeToRegex converts SQL LIKE pattern to MongoDB regex
func (t *MongoDBSQLTranslator) convertLikeToRegex(pattern string) string {
	// Escape regex special characters except % and _
	escaped := strings.ReplaceAll(pattern, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, ".", "\\.")
	escaped = strings.ReplaceAll(escaped, "^", "\\^")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "*", "\\*")
	escaped = strings.ReplaceAll(escaped, "+", "\\+")
	escaped = strings.ReplaceAll(escaped, "?", "\\?")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")
	escaped = strings.ReplaceAll(escaped, "{", "\\{")
	escaped = strings.ReplaceAll(escaped, "}", "\\}")
	escaped = strings.ReplaceAll(escaped, "|", "\\|")

	// Convert SQL wildcards to regex
	escaped = strings.ReplaceAll(escaped, "%", ".*") // % = any characters
	escaped = strings.ReplaceAll(escaped, "_", ".")  // _ = any single character

	return "^" + escaped + "$"
}

// translateJoinToLookup converts SQL JOIN to MongoDB $lookup aggregation stages
func (t *MongoDBSQLTranslator) translateJoinToLookup(join *sql.JoinClause) ([]bson.M, error) {
	var stages []bson.M

	// Get the join table collection name
	joinCollection := t.getCollectionName(join.Table.Table)

	// Extract the join condition (assumes simple equality join like table1.field1 = table2.field2)
	if join.Condition == nil || join.Condition.Condition == nil {
		return nil, fmt.Errorf("JOIN condition is required")
	}

	condition := join.Condition.Condition
	if condition.Operator != "=" {
		return nil, fmt.Errorf("only equality JOINs are supported, got: %s", condition.Operator)
	}

	// Parse the field references in the condition
	localField, foreignField, err := t.parseJoinCondition(condition, join.Table.Alias)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JOIN condition: %w", err)
	}

	// Create the $lookup stage
	lookupStage := bson.M{
		"$lookup": bson.M{
			"from":         joinCollection,
			"localField":   localField,
			"foreignField": foreignField,
			"as":           join.Table.Alias, // Use table alias as the result array name
		},
	}
	stages = append(stages, lookupStage)

	// For INNER JOIN, add $match to filter out documents without matches
	if join.Type == sql.JoinTypeInner {
		matchStage := bson.M{
			"$match": bson.M{
				join.Table.Alias: bson.M{"$ne": []any{}},
			},
		}
		stages = append(stages, matchStage)
	}

	// Add $unwind stage to flatten the joined array (for single record joins)
	// Note: This assumes one-to-one or many-to-one relationships
	unwindStage := bson.M{
		"$unwind": bson.M{
			"path":                       "$" + join.Table.Alias,
			"preserveNullAndEmptyArrays": join.Type != sql.JoinTypeInner, // Keep nulls for LEFT JOIN
		},
	}
	stages = append(stages, unwindStage)

	return stages, nil
}

// parseJoinCondition parses a JOIN condition like "p.user_id = u.id" and returns local and foreign fields
func (t *MongoDBSQLTranslator) parseJoinCondition(condition *sql.Condition, joinTableAlias string) (localField, foreignField string, err error) {
	// The condition field is the left side, the value is the right side
	leftSide := condition.Field  // e.g., "p.user_id"
	rightSide := condition.Value // e.g., "u.id"

	// Make sure the right side is a field reference (string)
	rightFieldStr, ok := rightSide.(string)
	if !ok {
		return "", "", fmt.Errorf("JOIN condition right side must be a field reference, got: %T", rightSide)
	}

	// Parse qualified field names
	leftParts := strings.Split(leftSide, ".")
	rightParts := strings.Split(rightFieldStr, ".")

	if len(leftParts) != 2 || len(rightParts) != 2 {
		return "", "", fmt.Errorf("JOIN condition fields must be qualified (table.column)")
	}

	// Map field names to database column names
	leftTableAlias := leftParts[0]
	leftFieldName := leftParts[1]
	rightTableAlias := rightParts[0]
	rightFieldName := rightParts[1]

	// For MongoDB, map schema field names to database column names
	leftColumn, err := t.mapQualifiedFieldName(leftTableAlias, leftFieldName)
	if err != nil {
		leftColumn = leftFieldName // fallback
	}

	rightColumn, err := t.mapQualifiedFieldName(rightTableAlias, rightFieldName)
	if err != nil {
		rightColumn = rightFieldName // fallback
	}

	// Determine which field belongs to the main table (local) and which to the joined table (foreign)
	// The joinTableAlias tells us which table we're joining TO
	if leftTableAlias == joinTableAlias {
		// Left side is the joined table, right side is the main table
		return rightColumn, leftColumn, nil // localField=rightColumn, foreignField=leftColumn
	} else {
		// Right side is the joined table, left side is the main table
		return leftColumn, rightColumn, nil // localField=leftColumn, foreignField=rightColumn
	}
}

// mapQualifiedFieldName maps a qualified field name (table.field) to database column name
func (t *MongoDBSQLTranslator) mapQualifiedFieldName(_, fieldName string) (string, error) {
	// Handle primary key field mapping to _id
	if fieldName == "id" {
		return "_id", nil
	}
	// For other fields, return as-is (foreign keys typically don't need mapping)
	return fieldName, nil
}

// isFunctionCall checks if a field name is actually a function call
func (t *MongoDBSQLTranslator) isFunctionCall(field string) bool {
	upperField := strings.ToUpper(field)
	return strings.Contains(upperField, "(") && strings.Contains(upperField, ")")
}

// translateFunctionCondition translates function calls in HAVING conditions to MongoDB aggregation criteria
func (t *MongoDBSQLTranslator) translateFunctionCondition(cond *sql.Condition, selectFields []sql.SelectField, argIndex *int) (bson.M, error) {
	// In HAVING clause, function calls like COUNT(*) are compared to values
	// In MongoDB aggregation, this becomes a comparison on the aggregated field

	// Find the corresponding alias from the SELECT fields
	functionExpr := strings.ToUpper(cond.Field)
	var fieldAlias string

	// Look for matching function expression in SELECT fields
	for _, field := range selectFields {
		if strings.ToUpper(field.Expression) == functionExpr {
			// Found matching function - use its alias if available
			if field.Alias != "" {
				fieldAlias = field.Alias
			} else {
				fieldAlias = field.Expression
			}
			break
		}
	}

	// If not found in SELECT fields, fall back to common aliases
	if fieldAlias == "" {
		switch {
		case strings.HasPrefix(functionExpr, "COUNT("):
			fieldAlias = "count"
		case strings.HasPrefix(functionExpr, "SUM("):
			fieldAlias = "sum"
		case strings.HasPrefix(functionExpr, "AVG("):
			fieldAlias = "avg"
		case strings.HasPrefix(functionExpr, "MIN("):
			fieldAlias = "min"
		case strings.HasPrefix(functionExpr, "MAX("):
			fieldAlias = "max"
		default:
			return nil, fmt.Errorf("unsupported function in HAVING clause: %s", cond.Field)
		}
	}

	// Build the condition on the aggregated field
	value := t.substituteValue(cond.Value, argIndex)

	switch strings.ToUpper(cond.Operator) {
	case "=":
		return bson.M{fieldAlias: value}, nil
	case "!=", "<>":
		return bson.M{fieldAlias: bson.M{"$ne": value}}, nil
	case ">":
		return bson.M{fieldAlias: bson.M{"$gt": value}}, nil
	case ">=":
		return bson.M{fieldAlias: bson.M{"$gte": value}}, nil
	case "<":
		return bson.M{fieldAlias: bson.M{"$lt": value}}, nil
	case "<=":
		return bson.M{fieldAlias: bson.M{"$lte": value}}, nil
	default:
		return nil, fmt.Errorf("unsupported operator in function condition: %s", cond.Operator)
	}
}

// translateSubqueryCondition translates a subquery condition to MongoDB using two-phase execution
func (t *MongoDBSQLTranslator) translateSubqueryCondition(fieldName string, subquery *sql.SelectStatement, argIndex *int, negated bool) (bson.M, error) {
	// Note: This function now returns a special marker that indicates the need for subquery execution
	// The actual execution will be handled by the SubqueryExecutor during query execution

	// For now, we return a placeholder that will be replaced during execution
	// This maintains the existing interface while enabling subquery support

	operator := "$in"
	if negated {
		operator = "$nin"
	}

	// Return a special marker that will be processed by the query executor
	return bson.M{
		"__subquery__": bson.M{
			"field":    fieldName,
			"operator": operator,
			"subquery": subquery,
			"argIndex": *argIndex,
		},
	}, nil
}

// translateGroupByProject creates a $project stage after GROUP BY to restructure the result
func (t *MongoDBSQLTranslator) translateGroupByProject(groupBy []string, fields []sql.SelectField) (bson.M, error) {
	projectStage := bson.M{}

	// First, map all the SELECT fields to their expected names
	for _, field := range fields {
		alias := field.Alias
		if alias == "" {
			// If no alias, extract the basic field name from qualified names
			if strings.Contains(field.Expression, ".") {
				parts := strings.Split(field.Expression, ".")
				alias = parts[len(parts)-1] // Take the last part (field name)
			} else {
				alias = field.Expression
			}
		}

		// Check if this is an aggregation function
		expr := strings.ToUpper(field.Expression)
		if strings.Contains(expr, "(") && strings.Contains(expr, ")") {
			// This is an aggregation function - it's already at root level with its alias
			projectStage[alias] = "$" + alias
		} else {
			// This should be a GROUP BY field - look for it in the _id object
			found := false
			for _, gbField := range groupBy {
				if field.Expression == gbField {
					// This field is in GROUP BY, so it's in _id
					safeKey := strings.ReplaceAll(gbField, ".", "_")
					projectStage[alias] = "$_id." + safeKey
					found = true
					break
				}
			}
			if !found {
				// If not found in GROUP BY, use as-is (fallback)
				projectStage[alias] = "$" + alias
			}
		}
	}

	// Exclude the _id field unless explicitly requested
	hasExplicitId := false
	for _, field := range fields {
		if field.Expression == "_id" || field.Expression == "id" {
			hasExplicitId = true
			break
		}
	}
	if !hasExplicitId {
		projectStage["_id"] = 0
	}

	return projectStage, nil
}
