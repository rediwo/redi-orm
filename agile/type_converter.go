package agile

import (
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// TypeConverter handles database-specific type conversions
type TypeConverter struct {
	capabilities types.DriverCapabilities
}

// NewTypeConverter creates a new type converter for the specified driver
func NewTypeConverter(capabilities types.DriverCapabilities) *TypeConverter {
	return &TypeConverter{
		capabilities: capabilities,
	}
}

// ConvertResult converts a single result map based on the driver capabilities
func (tc *TypeConverter) ConvertResult(modelName string, result map[string]any) map[string]any {
	if !tc.capabilities.NeedsTypeConversion() {
		return result // Only drivers that need conversion get special handling
	}

	// Convert MySQL string numbers to proper types
	converted := make(map[string]any, len(result))
	for key, value := range result {
		converted[key] = tc.convertValue(value)
	}

	return converted
}

// ConvertAggregateResult converts aggregation results
func (tc *TypeConverter) ConvertAggregateResult(result map[string]any) map[string]any {
	converted := make(map[string]any, len(result))

	for key, value := range result {
		switch key {
		case "_count", "_sum", "_avg", "_min", "_max":
			// These are aggregation results that might contain numeric values
			if subMap, ok := value.(map[string]any); ok {
				convertedSub := make(map[string]any, len(subMap))
				for subKey, subValue := range subMap {
					convertedSub[subKey] = tc.convertNumericValue(subValue)
				}
				converted[key] = convertedSub
			} else {
				// Direct numeric value
				converted[key] = tc.convertNumericValue(value)
			}
		default:
			// Regular field or nested structure
			converted[key] = tc.convertValue(value)
		}
	}

	return converted
}

// convertValue converts a single value based on its type
func (tc *TypeConverter) convertValue(value any) any {
	if !tc.capabilities.NeedsTypeConversion() {
		return value
	}

	// For MySQL, check if it's a string that should be a number
	if strVal, ok := value.(string); ok {
		// Try to convert to float first (handles both int and float)
		floatVal := utils.ToFloat64(strVal)
		if floatVal != 0 || strVal == "0" {
			// Check if it's actually an integer
			if floatVal == float64(int64(floatVal)) {
				return int64(floatVal)
			}
			return floatVal
		}
	}

	// Handle nested maps
	if mapVal, ok := value.(map[string]any); ok {
		return tc.ConvertResult("", mapVal)
	}

	// Handle arrays
	if arrayVal, ok := value.([]any); ok {
		converted := make([]any, len(arrayVal))
		for i, item := range arrayVal {
			converted[i] = tc.convertValue(item)
		}
		return converted
	}

	return value
}

// convertNumericValue specifically converts numeric aggregation results
func (tc *TypeConverter) convertNumericValue(value any) any {
	// Always try to convert to proper numeric type for aggregations
	switch v := value.(type) {
	case string:
		// MySQL returns strings for numeric aggregations
		floatVal := utils.ToFloat64(v)
		if floatVal != 0 || v == "0" {
			// For count operations, return int64
			if floatVal == float64(int64(floatVal)) {
				return int64(floatVal)
			}
			return floatVal
		}
		return v
	case int:
		return int64(v)
	case float32:
		return float64(v)
	default:
		return v
	}
}

// ConvertFieldValue converts a field value for database operations
func (tc *TypeConverter) ConvertFieldValue(fieldName string, value any) any {
	// This can be extended to handle driver-specific conversions for writes
	return value
}
