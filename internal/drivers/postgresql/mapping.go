package drivers

import (
	"github.com/rediwo/redi-orm/schema"
)

// fieldTypeToSQL converts schema field types to PostgreSQL column types
// PostgreSQL type mapping:
// - String maps to VARCHAR(255) (can be customized)
// - Int maps to INTEGER, Int64 to BIGINT
// - Float maps to DOUBLE PRECISION
// - Boolean maps to BOOLEAN
// - DateTime maps to TIMESTAMP
// - JSON maps to JSONB for better performance
func fieldTypeToSQL(ft schema.FieldType) string {
	switch ft {
	case schema.FieldTypeString:
		return "VARCHAR(255)"
	case schema.FieldTypeInt:
		return "INTEGER"
	case schema.FieldTypeInt64:
		return "BIGINT"
	case schema.FieldTypeFloat:
		return "DOUBLE PRECISION"
	case schema.FieldTypeBool:
		return "BOOLEAN"
	case schema.FieldTypeDateTime:
		return "TIMESTAMP"
	case schema.FieldTypeJSON:
		return "JSONB"
	default:
		return "VARCHAR(255)"
	}
}