package mongodb

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBFieldCondition represents a MongoDB-specific field condition
type MongoDBFieldCondition struct {
	fieldName string
	modelName string
	db        *MongoDB
}

// NewMongoDBFieldCondition creates a MongoDB field condition
func NewMongoDBFieldCondition(modelName, fieldName string, db *MongoDB) *MongoDBFieldCondition {
	return &MongoDBFieldCondition{
		fieldName: fieldName,
		modelName: modelName,
		db:        db,
	}
}

// Equals creates an equality condition
func (f *MongoDBFieldCondition) Equals(value any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "eq",
		value:     value,
		db:        f.db,
	}
}

// NotEquals creates a not-equals condition
func (f *MongoDBFieldCondition) NotEquals(value any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "ne",
		value:     value,
		db:        f.db,
	}
}

// GreaterThan creates a greater-than condition
func (f *MongoDBFieldCondition) GreaterThan(value any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "gt",
		value:     value,
		db:        f.db,
	}
}

// GreaterThanOrEqual creates a greater-than-or-equal condition
func (f *MongoDBFieldCondition) GreaterThanOrEqual(value any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "gte",
		value:     value,
		db:        f.db,
	}
}

// LessThan creates a less-than condition
func (f *MongoDBFieldCondition) LessThan(value any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "lt",
		value:     value,
		db:        f.db,
	}
}

// LessThanOrEqual creates a less-than-or-equal condition
func (f *MongoDBFieldCondition) LessThanOrEqual(value any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "lte",
		value:     value,
		db:        f.db,
	}
}

// In creates an IN condition
func (f *MongoDBFieldCondition) In(values ...any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "in",
		value:     values,
		db:        f.db,
	}
}

// NotIn creates a NOT IN condition
func (f *MongoDBFieldCondition) NotIn(values ...any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "nin",
		value:     values,
		db:        f.db,
	}
}

// Like creates a LIKE condition (converted to regex)
func (f *MongoDBFieldCondition) Like(pattern string) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "regex",
		value:     pattern,
		db:        f.db,
	}
}

// IsNull creates an IS NULL condition
func (f *MongoDBFieldCondition) IsNull() types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "null",
		value:     nil,
		db:        f.db,
	}
}

// IsNotNull creates an IS NOT NULL condition
func (f *MongoDBFieldCondition) IsNotNull() types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "not_null",
		value:     nil,
		db:        f.db,
	}
}

// Contains creates a contains condition (regex)
func (f *MongoDBFieldCondition) Contains(value string) types.Condition {
	// For contains, escape the value but don't add anchors
	escaped := escapeRegex(value)
	pattern := fmt.Sprintf(".*%s.*", escaped)
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "contains",
		value:     pattern,
		db:        f.db,
	}
}

// StartsWith creates a starts-with condition (regex)
func (f *MongoDBFieldCondition) StartsWith(value string) types.Condition {
	escaped := escapeRegex(value)
	pattern := fmt.Sprintf("^%s", escaped)
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "startswith",
		value:     pattern,
		db:        f.db,
	}
}

// EndsWith creates an ends-with condition (regex)
func (f *MongoDBFieldCondition) EndsWith(value string) types.Condition {
	escaped := escapeRegex(value)
	pattern := fmt.Sprintf("%s$", escaped)
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "endswith",
		value:     pattern,
		db:        f.db,
	}
}

// Between creates a between condition
func (f *MongoDBFieldCondition) Between(min, max any) types.Condition {
	return &MongoDBCondition{
		fieldName: f.fieldName,
		modelName: f.modelName,
		operator:  "between",
		value:     []any{min, max},
		db:        f.db,
	}
}

// GetFieldName returns the field name
func (f *MongoDBFieldCondition) GetFieldName() string {
	return f.fieldName
}

// GetModelName returns the model name
func (f *MongoDBFieldCondition) GetModelName() string {
	return f.modelName
}

// escapeRegex escapes special regex characters but preserves % and _ for LIKE processing
func escapeRegex(s string) string {
	// Only escape the most critical regex special characters for MongoDB
	// Don't escape % and _ as they need to be converted to .* and .
	replacer := strings.NewReplacer(
		"\\", "\\\\", // Must be first
		".", "\\.",
		"^", "\\^",
		"$", "\\$",
		"*", "\\*",
		"+", "\\+",
		"?", "\\?",
		"(", "\\(",
		")", "\\)",
		"[", "\\[",
		"]", "\\]",
		"{", "\\{",
		"}", "\\}",
		"|", "\\|",
	)
	return replacer.Replace(s)
}

// MongoDBCondition represents a MongoDB-specific condition
type MongoDBCondition struct {
	fieldName string
	modelName string
	operator  string
	value     any
	db        *MongoDB
}

// ToSQL generates MongoDB command JSON instead of SQL
func (c *MongoDBCondition) ToSQL(ctx *types.ConditionContext) (string, []any) {
	// Map field name to column name
	columnName := c.fieldName
	if c.db != nil {
		if mapped, err := c.db.GetFieldMapper().SchemaToColumn(c.modelName, c.fieldName); err == nil {
			columnName = mapped
		}
	}

	// Create MongoDB filter based on operator
	var filter bson.M
	switch c.operator {
	case "eq":
		if c.value == nil {
			filter = bson.M{columnName: nil}
		} else {
			filter = bson.M{columnName: c.value}
		}
	case "ne":
		filter = bson.M{columnName: bson.M{"$ne": c.value}}
	case "gt":
		filter = bson.M{columnName: bson.M{"$gt": c.value}}
	case "gte":
		filter = bson.M{columnName: bson.M{"$gte": c.value}}
	case "lt":
		filter = bson.M{columnName: bson.M{"$lt": c.value}}
	case "lte":
		filter = bson.M{columnName: bson.M{"$lte": c.value}}
	case "in":
		if values, ok := c.value.([]any); ok {
			filter = bson.M{columnName: bson.M{"$in": values}}
		} else {
			filter = bson.M{columnName: bson.M{"$in": []any{c.value}}}
		}
	case "nin":
		if values, ok := c.value.([]any); ok {
			filter = bson.M{columnName: bson.M{"$nin": values}}
		} else {
			filter = bson.M{columnName: bson.M{"$nin": []any{c.value}}}
		}
	case "regex":
		pattern := fmt.Sprintf("%v", c.value)
		// Convert SQL LIKE pattern to MongoDB regex
		pattern = convertLikeToRegex(pattern)
		filter = bson.M{columnName: bson.M{"$regex": pattern, "$options": "i"}}
	case "contains", "startswith", "endswith":
		// Direct regex pattern (already processed)
		pattern := fmt.Sprintf("%v", c.value)
		// Use case-sensitive matching for string operators
		fmt.Printf("[DEBUG] MongoDB filter for %s: field=%s, pattern=%s\n", c.operator, columnName, pattern)
		filter = bson.M{columnName: bson.M{"$regex": pattern}}
	case "null":
		filter = bson.M{columnName: nil}
	case "not_null":
		filter = bson.M{columnName: bson.M{"$ne": nil}}
	case "between":
		if values, ok := c.value.([]any); ok && len(values) == 2 {
			filter = bson.M{columnName: bson.M{"$gte": values[0], "$lte": values[1]}}
		} else {
			filter = bson.M{"$comment": "invalid between values"}
		}
	default:
		filter = bson.M{"$comment": fmt.Sprintf("unsupported operator: %s", c.operator)}
	}

	// Convert to JSON string (this will be our "SQL")
	jsonBytes, err := json.Marshal(filter)
	if err != nil {
		return fmt.Sprintf(`{"$comment": "failed to marshal filter: %s"}`, err.Error()), nil
	}

	return string(jsonBytes), nil
}

// convertLikeToRegex converts SQL LIKE patterns to MongoDB regex
func convertLikeToRegex(pattern string) string {
	// Escape regex special characters first, except % and _
	escaped := escapeRegex(pattern)

	// Replace SQL LIKE wildcards with regex equivalents
	escaped = strings.ReplaceAll(escaped, "%", ".*") // % matches any sequence
	escaped = strings.ReplaceAll(escaped, "_", ".")  // _ matches any single character

	// For LIKE patterns, don't anchor with ^ and $ unless the pattern explicitly requires it
	// This allows partial matching which is more typical for LIKE operations
	return escaped
}

// And combines conditions with AND logic
func (c *MongoDBCondition) And(condition types.Condition) types.Condition {
	return &MongoDBAndCondition{
		Left:  c,
		Right: condition,
		db:    c.db,
	}
}

// Or combines conditions with OR logic
func (c *MongoDBCondition) Or(condition types.Condition) types.Condition {
	return &MongoDBOrCondition{
		Left:  c,
		Right: condition,
		db:    c.db,
	}
}

// Not negates the condition
func (c *MongoDBCondition) Not() types.Condition {
	return &MongoDBNotCondition{
		Condition: c,
		db:        c.db,
	}
}

// MongoDBAndCondition represents AND logic
type MongoDBAndCondition struct {
	Left  types.Condition
	Right types.Condition
	db    *MongoDB
}

func (c *MongoDBAndCondition) ToSQL(ctx *types.ConditionContext) (string, []any) {
	leftSQL, _ := c.Left.ToSQL(ctx)
	rightSQL, _ := c.Right.ToSQL(ctx)

	// Parse left and right as JSON
	var leftFilter, rightFilter bson.M
	if err := json.Unmarshal([]byte(leftSQL), &leftFilter); err != nil {
		return `{"$comment": "failed to parse left condition"}`, nil
	}
	if err := json.Unmarshal([]byte(rightSQL), &rightFilter); err != nil {
		return `{"$comment": "failed to parse right condition"}`, nil
	}

	// Combine with $and
	combined := bson.M{"$and": []bson.M{leftFilter, rightFilter}}

	jsonBytes, err := json.Marshal(combined)
	if err != nil {
		return `{"$comment": "failed to marshal AND condition"}`, nil
	}

	return string(jsonBytes), nil
}

func (c *MongoDBAndCondition) And(condition types.Condition) types.Condition {
	return types.NewAndCondition(c, condition)
}

func (c *MongoDBAndCondition) Or(condition types.Condition) types.Condition {
	return types.NewOrCondition(c, condition)
}

func (c *MongoDBAndCondition) Not() types.Condition {
	return types.NewNotCondition(c)
}

// MongoDBOrCondition represents OR logic
type MongoDBOrCondition struct {
	Left  types.Condition
	Right types.Condition
	db    *MongoDB
}

func (c *MongoDBOrCondition) ToSQL(ctx *types.ConditionContext) (string, []any) {
	leftSQL, _ := c.Left.ToSQL(ctx)
	rightSQL, _ := c.Right.ToSQL(ctx)

	// Parse left and right as JSON
	var leftFilter, rightFilter bson.M
	if err := json.Unmarshal([]byte(leftSQL), &leftFilter); err != nil {
		return `{"$comment": "failed to parse left condition"}`, nil
	}
	if err := json.Unmarshal([]byte(rightSQL), &rightFilter); err != nil {
		return `{"$comment": "failed to parse right condition"}`, nil
	}

	// Combine with $or
	combined := bson.M{"$or": []bson.M{leftFilter, rightFilter}}

	jsonBytes, err := json.Marshal(combined)
	if err != nil {
		return `{"$comment": "failed to marshal OR condition"}`, nil
	}

	return string(jsonBytes), nil
}

func (c *MongoDBOrCondition) And(condition types.Condition) types.Condition {
	return types.NewAndCondition(c, condition)
}

func (c *MongoDBOrCondition) Or(condition types.Condition) types.Condition {
	return types.NewOrCondition(c, condition)
}

func (c *MongoDBOrCondition) Not() types.Condition {
	return types.NewNotCondition(c)
}

// MongoDBNotCondition represents NOT logic
type MongoDBNotCondition struct {
	Condition types.Condition
	db        *MongoDB
}

func (c *MongoDBNotCondition) ToSQL(ctx *types.ConditionContext) (string, []any) {
	innerSQL, _ := c.Condition.ToSQL(ctx)

	// Parse inner condition as JSON
	var innerFilter bson.M
	if err := json.Unmarshal([]byte(innerSQL), &innerFilter); err != nil {
		return `{"$comment": "failed to parse inner condition"}`, nil
	}

	// MongoDB doesn't support $not as a top-level operator
	// Use $nor for negating entire expressions
	combined := bson.M{"$nor": []bson.M{innerFilter}}

	jsonBytes, err := json.Marshal(combined)
	if err != nil {
		return `{"$comment": "failed to marshal NOT condition"}`, nil
	}

	return string(jsonBytes), nil
}

func (c *MongoDBNotCondition) And(condition types.Condition) types.Condition {
	return types.NewAndCondition(c, condition)
}

func (c *MongoDBNotCondition) Or(condition types.Condition) types.Condition {
	return types.NewOrCondition(c, condition)
}

func (c *MongoDBNotCondition) Not() types.Condition {
	return types.NewNotCondition(c)
}
