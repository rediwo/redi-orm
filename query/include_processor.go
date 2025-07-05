package query

import (
	"sort"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// IncludeProcessor handles processing of include options for relation loading
type IncludeProcessor struct {
	database       types.Database
	fieldMapper    types.FieldMapper
	includeOptions types.IncludeOptions
}

// NewIncludeProcessor creates a new include processor
func NewIncludeProcessor(database types.Database, fieldMapper types.FieldMapper, includeOptions types.IncludeOptions) *IncludeProcessor {
	return &IncludeProcessor{
		database:       database,
		fieldMapper:    fieldMapper,
		includeOptions: includeOptions,
	}
}

// ProcessRelationData filters and processes relation data based on include options
func (ip *IncludeProcessor) ProcessRelationData(relationPath string, data []map[string]any) []map[string]any {
	opt, exists := ip.includeOptions[relationPath]
	if !exists || opt == nil {
		return data // No options, return all data
	}

	// Apply filtering (where conditions)
	if opt.Where != nil {
		filtered := make([]map[string]any, 0)
		for _, record := range data {
			if ip.matchesCondition(record, opt.Where, relationPath) {
				filtered = append(filtered, record)
			}
		}
		data = filtered
	}

	// Apply ordering
	if len(opt.OrderBy) > 0 {
		ip.sortData(data, opt.OrderBy, relationPath)
	}

	// Apply limit and offset
	if opt.Offset != nil || opt.Limit != nil {
		start := 0
		if opt.Offset != nil {
			start = *opt.Offset
		}

		end := len(data)
		if opt.Limit != nil && start+*opt.Limit < end {
			end = start + *opt.Limit
		}

		if start < len(data) {
			data = data[start:end]
		} else {
			data = []map[string]any{}
		}
	}

	// Apply field selection
	if len(opt.Select) > 0 {
		selected := make([]map[string]any, len(data))
		for i, record := range data {
			selectedRecord := make(map[string]any)
			// Always include ID for relation tracking
			if id, exists := record["id"]; exists {
				selectedRecord["id"] = id
			}
			// Include selected fields
			for _, field := range opt.Select {
				if value, exists := record[field]; exists {
					selectedRecord[field] = value
				}
			}
			selected[i] = selectedRecord
		}
		data = selected
	}

	return data
}

// matchesCondition checks if a record matches the given condition
func (ip *IncludeProcessor) matchesCondition(record map[string]any, condition types.Condition, relationPath string) bool {
	// Extract model name from relation path
	parts := strings.Split(relationPath, ".")
	modelName := ""
	if len(parts) > 0 {
		// Get the last part as the relation name
		relationName := parts[len(parts)-1]
		// Try to infer the model name from the relation name
		// This is a simplified approach - in production, we'd need better mapping
		modelName = ip.inferModelName(relationName)
	}

	// Create a simple condition context for evaluation
	_ = types.NewConditionContext(ip.fieldMapper, modelName, "")

	// For now, we'll do a simple evaluation
	// In a real implementation, we'd need to properly evaluate the condition
	// against the record data

	// This is a placeholder - actual condition evaluation would be more complex
	return true
}

// inferModelName tries to infer the model name from a relation name
func (ip *IncludeProcessor) inferModelName(relationName string) string {
	// Simple heuristic: singularize the relation name
	// In production, we'd need proper relation metadata
	singular := strings.TrimSuffix(relationName, "s")

	// Default to capitalized singular form
	if len(singular) > 0 {
		return strings.ToUpper(singular[:1]) + singular[1:]
	}

	return relationName
}

// GetSelectFields returns the fields to select for a given relation path
func (ip *IncludeProcessor) GetSelectFields(relationPath string, schema *schema.Schema) []string {
	opt, exists := ip.includeOptions[relationPath]
	if !exists || opt == nil || len(opt.Select) == 0 {
		// Return all fields
		fields := make([]string, 0, len(schema.Fields))
		for _, field := range schema.Fields {
			fields = append(fields, field.Name)
		}
		return fields
	}

	// Always include ID
	fields := []string{"id"}

	// Add selected fields
	for _, field := range opt.Select {
		// Avoid duplicating ID
		if field != "id" {
			fields = append(fields, field)
		}
	}

	return fields
}

// sortData sorts the data array based on the orderBy options
func (ip *IncludeProcessor) sortData(data []map[string]any, orderBy []types.OrderByOption, relationPath string) {
	if len(data) == 0 || len(orderBy) == 0 {
		return
	}

	sort.Slice(data, func(i, j int) bool {
		for _, order := range orderBy {
			// Get values from both records
			valI, okI := data[i][order.Field]
			valJ, okJ := data[j][order.Field]

			// Handle nil/missing values
			if !okI && !okJ {
				continue // Both nil, check next order field
			}
			if !okI {
				return order.Direction == types.DESC // nil values go to end for ASC, start for DESC
			}
			if !okJ {
				return order.Direction == types.ASC // nil values go to end for ASC, start for DESC
			}

			// Compare values based on type
			cmp := ip.compareValues(valI, valJ)
			if cmp == 0 {
				continue // Equal, check next order field
			}

			if order.Direction == types.ASC {
				return cmp < 0
			}
			return cmp > 0
		}
		return false // All fields equal
	})
}

// compareValues compares two values and returns -1, 0, or 1
func (ip *IncludeProcessor) compareValues(a, b any) int {
	// Handle same types
	switch va := a.(type) {
	case int:
		vb, ok := b.(int)
		if ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case int64:
		vb, ok := b.(int64)
		if ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case float64:
		vb, ok := b.(float64)
		if ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case string:
		vb, ok := b.(string)
		if ok {
			return strings.Compare(va, vb)
		}
	}

	// Try to convert to common types for comparison
	// Try float64 first (works for all numeric types)
	fa := utils.ToFloat64(a)
	fb := utils.ToFloat64(b)
	if fa < fb {
		return -1
	} else if fa > fb {
		return 1
	} else if fa != 0 || fb != 0 {
		// At least one value was numeric and they're equal
		return 0
	}

	// Try string comparison as fallback
	sa := utils.ToString(a)
	sb := utils.ToString(b)
	return strings.Compare(sa, sb)
}
