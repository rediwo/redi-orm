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

func (c BaseCondition) ToSQL() (string, []any) {
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

func (c *AndCondition) ToSQL() (string, []any) {
	if len(c.Conditions) == 0 {
		return "", nil
	}

	var parts []string
	var args []any

	for _, condition := range c.Conditions {
		sql, condArgs := condition.ToSQL()
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

func (c *OrCondition) ToSQL() (string, []any) {
	if len(c.Conditions) == 0 {
		return "", nil
	}

	var parts []string
	var args []any

	for _, condition := range c.Conditions {
		sql, condArgs := condition.ToSQL()
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

func (c *NotCondition) ToSQL() (string, []any) {
	sql, args := c.Condition.ToSQL()
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
	return NewBaseCondition(f.FieldName+" = ?", value)
}

func (f *FieldConditionImpl) NotEquals(value any) Condition {
	return NewBaseCondition(f.FieldName+" != ?", value)
}

func (f *FieldConditionImpl) GreaterThan(value any) Condition {
	return NewBaseCondition(f.FieldName+" > ?", value)
}

func (f *FieldConditionImpl) GreaterThanOrEqual(value any) Condition {
	return NewBaseCondition(f.FieldName+" >= ?", value)
}

func (f *FieldConditionImpl) LessThan(value any) Condition {
	return NewBaseCondition(f.FieldName+" < ?", value)
}

func (f *FieldConditionImpl) LessThanOrEqual(value any) Condition {
	return NewBaseCondition(f.FieldName+" <= ?", value)
}

func (f *FieldConditionImpl) In(values ...any) Condition {
	if len(values) == 0 {
		return NewBaseCondition("1 = 0") // Always false
	}

	placeholders := strings.Repeat("?,", len(values))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	sql := fmt.Sprintf("%s IN (%s)", f.FieldName, placeholders)
	return NewBaseCondition(sql, values...)
}

func (f *FieldConditionImpl) NotIn(values ...any) Condition {
	if len(values) == 0 {
		return NewBaseCondition("1 = 1") // Always true
	}

	placeholders := strings.Repeat("?,", len(values))
	placeholders = placeholders[:len(placeholders)-1] // Remove trailing comma

	sql := fmt.Sprintf("%s NOT IN (%s)", f.FieldName, placeholders)
	return NewBaseCondition(sql, values...)
}

func (f *FieldConditionImpl) Contains(value string) Condition {
	return NewBaseCondition(f.FieldName+" LIKE ?", "%"+value+"%")
}

func (f *FieldConditionImpl) StartsWith(value string) Condition {
	return NewBaseCondition(f.FieldName+" LIKE ?", value+"%")
}

func (f *FieldConditionImpl) EndsWith(value string) Condition {
	return NewBaseCondition(f.FieldName+" LIKE ?", "%"+value)
}

func (f *FieldConditionImpl) Like(pattern string) Condition {
	return NewBaseCondition(f.FieldName+" LIKE ?", pattern)
}

func (f *FieldConditionImpl) IsNull() Condition {
	return NewBaseCondition(f.FieldName + " IS NULL")
}

func (f *FieldConditionImpl) IsNotNull() Condition {
	return NewBaseCondition(f.FieldName + " IS NOT NULL")
}

func (f *FieldConditionImpl) Between(min, max any) Condition {
	return NewBaseCondition(f.FieldName+" BETWEEN ? AND ?", min, max)
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
