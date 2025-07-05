package query

import (
	"regexp"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// ConditionEvaluator evaluates conditions against in-memory records
type ConditionEvaluator struct {
	fieldMapper types.FieldMapper
}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator(fieldMapper types.FieldMapper) *ConditionEvaluator {
	return &ConditionEvaluator{
		fieldMapper: fieldMapper,
	}
}

// Evaluate evaluates a condition against a record
func (ce *ConditionEvaluator) Evaluate(condition types.Condition, record map[string]any, modelName string) bool {
	if condition == nil {
		return true
	}

	// Handle different condition types
	switch cond := condition.(type) {
	case *types.AndCondition:
		return ce.evaluateAndCondition(cond, record, modelName)
	case *types.OrCondition:
		return ce.evaluateOrCondition(cond, record, modelName)
	case *types.NotCondition:
		return ce.evaluateNotCondition(cond, record, modelName)
	case *types.MappedFieldCondition:
		return ce.evaluateMappedFieldCondition(cond, record, modelName)
	case *types.BaseCondition:
		// For base conditions, we need to parse the SQL
		return ce.evaluateBaseCondition(cond, record, modelName)
	default:
		// Unknown condition type - be permissive
		return true
	}
}

// evaluateAndCondition evaluates an AND condition
func (ce *ConditionEvaluator) evaluateAndCondition(cond *types.AndCondition, record map[string]any, modelName string) bool {
	// All sub-conditions must be true
	for _, subCond := range cond.Conditions {
		if !ce.Evaluate(subCond, record, modelName) {
			return false
		}
	}
	return true
}

// evaluateOrCondition evaluates an OR condition
func (ce *ConditionEvaluator) evaluateOrCondition(cond *types.OrCondition, record map[string]any, modelName string) bool {
	// At least one sub-condition must be true
	for _, subCond := range cond.Conditions {
		if ce.Evaluate(subCond, record, modelName) {
			return true
		}
	}
	return false
}

// evaluateNotCondition evaluates a NOT condition
func (ce *ConditionEvaluator) evaluateNotCondition(cond *types.NotCondition, record map[string]any, modelName string) bool {
	// Negate the inner condition
	return !ce.Evaluate(cond.Condition, record, modelName)
}

// evaluateMappedFieldCondition evaluates a mapped field condition
func (ce *ConditionEvaluator) evaluateMappedFieldCondition(cond *types.MappedFieldCondition, record map[string]any, modelName string) bool {
	// Extract the field name and operator from the SQL
	// The SQL is in format "fieldName operator ?"
	sql := cond.GetSQL()
	args := cond.GetArgs()

	// Parse the SQL to extract field name and operator
	fieldName, operator, value := ce.parseFieldCondition(sql, args)
	if fieldName == "" {
		return true // Unable to parse, be permissive
	}

	// Get the field value from record
	fieldValue, exists := record[fieldName]
	if !exists {
		// Field doesn't exist - consider it as NULL
		fieldValue = nil
	}

	// Evaluate based on operator
	return ce.evaluateOperator(fieldValue, operator, value)
}

// evaluateBaseCondition evaluates a base condition
func (ce *ConditionEvaluator) evaluateBaseCondition(cond *types.BaseCondition, record map[string]any, modelName string) bool {
	// For base conditions, we need to parse the SQL
	sql := cond.SQL
	args := cond.Args

	// Parse the SQL to extract field name and operator
	fieldName, operator, value := ce.parseFieldCondition(sql, args)
	if fieldName == "" {
		return true // Unable to parse, be permissive
	}

	// Get the field value from record
	fieldValue, exists := record[fieldName]
	if !exists {
		// Field doesn't exist - consider it as NULL
		fieldValue = nil
	}

	// Evaluate based on operator
	return ce.evaluateOperator(fieldValue, operator, value)
}

// parseFieldCondition parses a simple field condition SQL
func (ce *ConditionEvaluator) parseFieldCondition(sql string, args []any) (fieldName string, operator string, value any) {
	// Remove extra spaces and convert to lowercase for comparison
	sql = strings.TrimSpace(sql)

	// Common patterns
	patterns := []struct {
		regex    string
		field    int
		op       string
		hasValue bool
	}{
		{`^(\w+)\s*=\s*\?$`, 1, "=", true},
		{`^(\w+)\s*!=\s*\?$`, 1, "!=", true},
		{`^(\w+)\s*<>\s*\?$`, 1, "!=", true},
		{`^(\w+)\s*>\s*\?$`, 1, ">", true},
		{`^(\w+)\s*>=\s*\?$`, 1, ">=", true},
		{`^(\w+)\s*<\s*\?$`, 1, "<", true},
		{`^(\w+)\s*<=\s*\?$`, 1, "<=", true},
		{`^(\w+)\s+LIKE\s+\?$`, 1, "LIKE", true},
		{`^(\w+)\s+NOT\s+LIKE\s+\?$`, 1, "NOT LIKE", true},
		{`^(\w+)\s+IS\s+NULL$`, 1, "IS NULL", false},
		{`^(\w+)\s+IS\s+NOT\s+NULL$`, 1, "IS NOT NULL", false},
		{`^(\w+)\s+IN\s+\(\?\)$`, 1, "IN", true},
		{`^(\w+)\s+NOT\s+IN\s+\(\?\)$`, 1, "NOT IN", true},
		{`^(\w+)\s+BETWEEN\s+\?\s+AND\s+\?$`, 1, "BETWEEN", true},
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile("(?i)" + pattern.regex)
		if matches := re.FindStringSubmatch(sql); matches != nil {
			fieldName = matches[pattern.field]
			operator = pattern.op
			if pattern.hasValue && len(args) > 0 {
				if operator == "BETWEEN" && len(args) >= 2 {
					value = []any{args[0], args[1]}
				} else {
					value = args[0]
				}
			}
			return
		}
	}

	return "", "", nil
}

// evaluateOperator evaluates a comparison operator
func (ce *ConditionEvaluator) evaluateOperator(fieldValue any, operator string, value any) bool {
	switch operator {
	case "=":
		return ce.compareValues(fieldValue, value) == 0
	case "!=":
		return ce.compareValues(fieldValue, value) != 0
	case ">":
		return ce.compareValues(fieldValue, value) > 0
	case ">=":
		return ce.compareValues(fieldValue, value) >= 0
	case "<":
		return ce.compareValues(fieldValue, value) < 0
	case "<=":
		return ce.compareValues(fieldValue, value) <= 0
	case "IN":
		// Check if value is in the list
		if values, ok := value.([]any); ok {
			for _, v := range values {
				if ce.compareValues(fieldValue, v) == 0 {
					return true
				}
			}
		}
		return false
	case "NOT IN":
		// Check if value is not in the list
		if values, ok := value.([]any); ok {
			for _, v := range values {
				if ce.compareValues(fieldValue, v) == 0 {
					return false
				}
			}
		}
		return true
	case "LIKE":
		// Simple pattern matching
		pattern := utils.ToString(value)
		text := utils.ToString(fieldValue)
		// Convert SQL LIKE pattern to Go pattern
		// % -> .*
		// _ -> .
		pattern = strings.ReplaceAll(pattern, "%", ".*")
		pattern = strings.ReplaceAll(pattern, "_", ".")
		matched, _ := regexp.MatchString("^"+pattern+"$", text)
		return matched
	case "NOT LIKE":
		// Inverse of LIKE
		pattern := utils.ToString(value)
		text := utils.ToString(fieldValue)
		pattern = strings.ReplaceAll(pattern, "%", ".*")
		pattern = strings.ReplaceAll(pattern, "_", ".")
		matched, _ := regexp.MatchString("^"+pattern+"$", text)
		return !matched
	case "IS NULL":
		return fieldValue == nil
	case "IS NOT NULL":
		return fieldValue != nil
	case "BETWEEN":
		// Between requires two values
		if values, ok := value.([]any); ok && len(values) == 2 {
			return ce.compareValues(fieldValue, values[0]) >= 0 &&
				ce.compareValues(fieldValue, values[1]) <= 0
		}
		return false
	default:
		// Unknown operator - be permissive
		return true
	}
}

// compareValues compares two values and returns -1, 0, or 1
func (ce *ConditionEvaluator) compareValues(a, b any) int {
	// Handle nil values
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Convert to comparable types
	switch va := a.(type) {
	case int:
		vb := utils.ToInt(b)
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case int64:
		vb := utils.ToInt64(b)
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case float32:
		vb := utils.ToFloat32(b)
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case float64:
		vb := utils.ToFloat64(b)
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case bool:
		vb := utils.ToBool(b)
		if !va && vb {
			return -1
		} else if va && !vb {
			return 1
		}
		return 0
	}

	// Fall back to string comparison
	sa := utils.ToString(a)
	sb := utils.ToString(b)
	return strings.Compare(sa, sb)
}
