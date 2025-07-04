package query

import (
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
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
		// TODO: Implement ordering
		// This would require a custom sort function
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
