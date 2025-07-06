package mongodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/sql"
)

// SubqueryExecutor handles execution of SQL subqueries in MongoDB
type SubqueryExecutor struct {
	db         *MongoDB
	translator *MongoDBSQLTranslator
}

// NewSubqueryExecutor creates a new subquery executor
func NewSubqueryExecutor(db *MongoDB, translator *MongoDBSQLTranslator) *SubqueryExecutor {
	return &SubqueryExecutor{
		db:         db,
		translator: translator,
	}
}

// ExecuteSubquery executes a subquery and returns the result values
func (e *SubqueryExecutor) ExecuteSubquery(ctx context.Context, subquery *sql.SelectStatement, args []any) ([]any, error) {
	// Validate subquery structure
	if len(subquery.Fields) != 1 {
		return nil, fmt.Errorf("subquery must select exactly one field, got %d", len(subquery.Fields))
	}

	// Set up translator with arguments
	e.translator.SetArgs(args)

	// Translate subquery to MongoDB command
	mongoCmd, err := e.translator.TranslateToCommand(subquery)
	if err != nil {
		return nil, fmt.Errorf("failed to translate subquery: %w", err)
	}

	// Execute the subquery
	collection := e.db.client.Database(e.db.dbName).Collection(mongoCmd.Collection)

	var results []map[string]any
	switch mongoCmd.Operation {
	case "find":
		rawQuery := NewMongoDBRawQuery(e.db.client.Database(e.db.dbName), nil, e.db, "", args...)
		err = rawQuery.executeFind(ctx, collection, mongoCmd, &results)
	case "aggregate":
		rawQuery := NewMongoDBRawQuery(e.db.client.Database(e.db.dbName), nil, e.db, "", args...)
		err = rawQuery.executeAggregate(ctx, collection, mongoCmd, &results)
	default:
		return nil, fmt.Errorf("unsupported subquery operation: %s", mongoCmd.Operation)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute subquery: %w", err)
	}

	// Extract values from results
	fieldName := subquery.Fields[0].Expression
	if subquery.Fields[0].Alias != "" {
		fieldName = subquery.Fields[0].Alias
	}

	// For qualified field names like "p.user_id", extract the base name "user_id"
	if len(subquery.Fields) > 0 && subquery.Fields[0].Alias == "" {
		if fieldExpr := subquery.Fields[0].Expression; fieldExpr != "" {
			// Handle qualified names like "p.user_id" -> "user_id"
			parts := strings.Split(fieldExpr, ".")
			if len(parts) > 1 {
				fieldName = parts[len(parts)-1] // Take the last part
			}
		}
	}

	values := make([]any, 0, len(results))
	seenValues := make(map[any]bool) // For DISTINCT behavior

	for _, result := range results {
		var value any
		var found bool

		// Try different field name variations
		fieldVariations := []string{
			fieldName,
			e.mapFieldName(fieldName),
		}

		for _, field := range fieldVariations {
			if v, exists := result[field]; exists {
				value = v
				found = true
				break
			}
		}

		if !found {
			// If field not found, skip this result (might be NULL)
			continue
		}

		// Implement DISTINCT behavior (SQL subqueries often use DISTINCT)
		if !seenValues[value] {
			values = append(values, value)
			seenValues[value] = true
		}
	}

	return values, nil
}

// mapFieldName is a helper that delegates to the translator
func (e *SubqueryExecutor) mapFieldName(fieldName string) string {
	mapped, err := e.translator.mapFieldName(fieldName)
	if err != nil {
		return fieldName // fallback
	}
	return mapped
}
