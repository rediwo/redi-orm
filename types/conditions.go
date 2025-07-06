package types

import (
	"fmt"
	"strings"
)

// BaseCondition implements common condition functionality
type BaseCondition struct {
	SQL  string
	Args []any
}

func (c BaseCondition) ToSQL(ctx *ConditionContext) (string, []any) {
	return c.SQL, c.Args
}

func (c BaseCondition) And(condition Condition) Condition {
	return NewAndCondition(c, condition)
}

func (c BaseCondition) Or(condition Condition) Condition {
	return NewOrCondition(c, condition)
}

func (c BaseCondition) Not() Condition {
	return NewNotCondition(c)
}

// AndCondition represents AND logic
type AndCondition struct {
	Conditions []Condition
}

func NewAndCondition(conditions ...Condition) *AndCondition {
	return &AndCondition{Conditions: conditions}
}

func (c *AndCondition) ToSQL(ctx *ConditionContext) (string, []any) {
	if len(c.Conditions) == 0 {
		return "", nil
	}

	var parts []string
	var args []any

	for _, condition := range c.Conditions {
		sql, condArgs := condition.ToSQL(ctx)
		if sql != "" {
			parts = append(parts, fmt.Sprintf("(%s)", sql))
			args = append(args, condArgs...)
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, " AND "), args
}

func (c *AndCondition) And(condition Condition) Condition {
	return NewAndCondition(append(c.Conditions, condition)...)
}

func (c *AndCondition) Or(condition Condition) Condition {
	return NewOrCondition(c, condition)
}

func (c *AndCondition) Not() Condition {
	return NewNotCondition(c)
}

// OrCondition represents OR logic
type OrCondition struct {
	Conditions []Condition
}

func NewOrCondition(conditions ...Condition) *OrCondition {
	return &OrCondition{Conditions: conditions}
}

func (c *OrCondition) ToSQL(ctx *ConditionContext) (string, []any) {
	if len(c.Conditions) == 0 {
		return "", nil
	}

	var parts []string
	var args []any

	for _, condition := range c.Conditions {
		sql, condArgs := condition.ToSQL(ctx)
		if sql != "" {
			parts = append(parts, fmt.Sprintf("(%s)", sql))
			args = append(args, condArgs...)
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, " OR "), args
}

func (c *OrCondition) And(condition Condition) Condition {
	return NewAndCondition(c, condition)
}

func (c *OrCondition) Or(condition Condition) Condition {
	return NewOrCondition(append(c.Conditions, condition)...)
}

func (c *OrCondition) Not() Condition {
	return NewNotCondition(c)
}

// NotCondition represents NOT logic
type NotCondition struct {
	Condition Condition
}

func NewNotCondition(condition Condition) *NotCondition {
	return &NotCondition{Condition: condition}
}

func (c *NotCondition) ToSQL(ctx *ConditionContext) (string, []any) {
	sql, args := c.Condition.ToSQL(ctx)
	if sql == "" {
		return "", nil
	}
	return fmt.Sprintf("NOT (%s)", sql), args
}

func (c *NotCondition) And(condition Condition) Condition {
	return NewAndCondition(c, condition)
}

func (c *NotCondition) Or(condition Condition) Condition {
	return NewOrCondition(c, condition)
}

func (c *NotCondition) Not() Condition {
	return c.Condition // Double negative
}

// RawCondition for raw SQL conditions
type RawCondition struct {
	BaseCondition
}

func NewRawCondition(sql string, args ...any) *RawCondition {
	return &RawCondition{
		BaseCondition: BaseCondition{
			SQL:  sql,
			Args: args,
		},
	}
}

// FieldConditionImpl implements FieldCondition interface
type FieldConditionImpl struct {
	ModelName string
	FieldName string
}

// MappedFieldCondition wraps a base condition with field mapping support
type MappedFieldCondition struct {
	BaseCondition
	fieldName string
	modelName string
	operator  string
}

// GetSQL returns the SQL string
func (f *MappedFieldCondition) GetSQL() string {
	return f.SQL
}

// GetArgs returns the SQL arguments
func (f *MappedFieldCondition) GetArgs() []any {
	return f.Args
}

// GetFieldName returns the field name
func (f *MappedFieldCondition) GetFieldName() string {
	return f.fieldName
}

// GetModelName returns the model name
func (f *MappedFieldCondition) GetModelName() string {
	return f.modelName
}

// ToSQL generates SQL with proper field mapping
func (f *MappedFieldCondition) ToSQL(ctx *ConditionContext) (string, []any) {
	// If no context, use the base SQL as-is
	if ctx == nil || ctx.FieldMapper == nil {
		return f.BaseCondition.ToSQL(ctx)
	}

	// Use the context with proper model name if needed
	mappingCtx := ctx
	if f.modelName != "" && ctx.ModelName != f.modelName {
		// Create a new context with the field's model name for correct mapping
		mappingCtx = &ConditionContext{
			FieldMapper:     ctx.FieldMapper,
			ModelName:       f.modelName,
			TableAlias:      ctx.TableAlias,
			JoinedTables:    ctx.JoinedTables,
			QuoteIdentifier: ctx.QuoteIdentifier,
		}
	}

	// Map field to column
	columnRef, err := mappingCtx.MapFieldToColumn(f.fieldName)
	if err != nil {
		// If mapping fails, use original field name
		columnRef = f.fieldName
	}

	// Replace field name with column reference in SQL
	sql := strings.Replace(f.SQL, f.fieldName, columnRef, 1)
	return sql, f.Args
}

// And combines this condition with another using AND logic
func (f *MappedFieldCondition) And(condition Condition) Condition {
	return NewAndCondition(f, condition)
}

// Or combines this condition with another using OR logic
func (f *MappedFieldCondition) Or(condition Condition) Condition {
	return NewOrCondition(f, condition)
}

// Not negates this condition
func (f *MappedFieldCondition) Not() Condition {
	return NewNotCondition(f)
}

// AggregationCondition represents a condition on an aggregated value
type AggregationCondition struct {
	BaseCondition
	AggregationType string // "sum", "avg", "min", "max", "count"
	FieldName       string
	Operator        string
	Value           any
}

// NewAggregationCondition creates a new aggregation condition
func NewAggregationCondition(aggType, fieldName, operator string, value any) *AggregationCondition {
	return &AggregationCondition{
		AggregationType: aggType,
		FieldName:       fieldName,
		Operator:        operator,
		Value:           value,
	}
}

// ToSQL generates the SQL for this aggregation condition
func (a *AggregationCondition) ToSQL(ctx *ConditionContext) (string, []any) {
	// Map field to column if needed
	columnName := a.FieldName
	if ctx != nil && ctx.FieldMapper != nil && ctx.ModelName != "" {
		if mapped, err := ctx.FieldMapper.SchemaToColumn(ctx.ModelName, a.FieldName); err == nil {
			columnName = mapped
		}
	}

	// Build aggregation expression
	var aggExpr string
	switch a.AggregationType {
	case "count":
		if a.FieldName == "_all" || a.FieldName == "*" {
			aggExpr = "COUNT(*)"
		} else {
			aggExpr = fmt.Sprintf("COUNT(%s)", columnName)
		}
	case "sum":
		aggExpr = fmt.Sprintf("SUM(%s)", columnName)
	case "avg":
		aggExpr = fmt.Sprintf("AVG(%s)", columnName)
	case "min":
		aggExpr = fmt.Sprintf("MIN(%s)", columnName)
	case "max":
		aggExpr = fmt.Sprintf("MAX(%s)", columnName)
	default:
		return "", nil
	}

	// Map operator
	var sqlOp string
	switch a.Operator {
	case "gt":
		sqlOp = ">"
	case "gte":
		sqlOp = ">="
	case "lt":
		sqlOp = "<"
	case "lte":
		sqlOp = "<="
	case "equals", "=":
		sqlOp = "="
	case "not", "!=":
		sqlOp = "!="
	default:
		sqlOp = "="
	}

	return fmt.Sprintf("%s %s ?", aggExpr, sqlOp), []any{a.Value}
}

func NewFieldCondition(modelName, fieldName string) *FieldConditionImpl {
	return &FieldConditionImpl{
		ModelName: modelName,
		FieldName: fieldName,
	}
}

func (f *FieldConditionImpl) GetFieldName() string {
	return f.FieldName
}

func (f *FieldConditionImpl) GetModelName() string {
	return f.ModelName
}

func (f *FieldConditionImpl) Equals(value any) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" = ?", value), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) NotEquals(value any) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" != ?", value), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) GreaterThan(value any) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" > ?", value), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) GreaterThanOrEqual(value any) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" >= ?", value), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) LessThan(value any) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" < ?", value), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) LessThanOrEqual(value any) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" <= ?", value), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) In(values ...any) Condition {
	if len(values) == 0 {
		return NewBaseCondition("1 = 0") // Always false
	}

	placeholders := strings.Repeat("?,", len(values))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	sql := fmt.Sprintf("%s IN (%s)", f.FieldName, placeholders)
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(sql, values...), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) NotIn(values ...any) Condition {
	if len(values) == 0 {
		return NewBaseCondition("1 = 1") // Always true
	}

	placeholders := strings.Repeat("?,", len(values))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	sql := fmt.Sprintf("%s NOT IN (%s)", f.FieldName, placeholders)
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(sql, values...), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) Contains(value string) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" LIKE ?", "%"+value+"%"), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) StartsWith(value string) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" LIKE ?", value+"%"), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) EndsWith(value string) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" LIKE ?", "%"+value), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) Like(pattern string) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" LIKE ?", pattern), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) IsNull() Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName + " IS NULL"), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) IsNotNull() Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName + " IS NOT NULL"), fieldName: f.FieldName, modelName: f.ModelName}
}

func (f *FieldConditionImpl) Between(min, max any) Condition {
	return &MappedFieldCondition{BaseCondition: *NewBaseCondition(f.FieldName+" BETWEEN ? AND ?", min, max), fieldName: f.FieldName, modelName: f.ModelName}
}

// Helper function to create BaseCondition
func NewBaseCondition(sql string, args ...any) *BaseCondition {
	return &BaseCondition{
		SQL:  sql,
		Args: args,
	}
}

// Utility functions for building conditions
func And(conditions ...Condition) Condition {
	return NewAndCondition(conditions...)
}

func Or(conditions ...Condition) Condition {
	return NewOrCondition(conditions...)
}

func Not(condition Condition) Condition {
	return NewNotCondition(condition)
}

func Raw(sql string, args ...any) Condition {
	return NewRawCondition(sql, args...)
}
