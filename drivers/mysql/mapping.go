package drivers

import (
	"github.com/rediwo/redi-orm/schema"
)

// fieldTypeToSQL converts schema field types to MySQL column types
// MySQL type mapping:
// - String maps to VARCHAR(255) (can be customized)
// - Int maps to INT, Int64 to BIGINT
// - Float maps to DOUBLE
// - Boolean maps to BOOLEAN
// - DateTime maps to DATETIME
// - JSON maps to native JSON type (MySQL 5.7+)
func fieldTypeToSQL(ft schema.FieldType) string {
	switch ft {
	case schema.FieldTypeString:
		return "VARCHAR(255)"
	case schema.FieldTypeInt:
		return "INT"
	case schema.FieldTypeInt64:
		return "BIGINT"
	case schema.FieldTypeFloat:
		return "DOUBLE"
	case schema.FieldTypeBool:
		return "BOOLEAN"
	case schema.FieldTypeDateTime:
		return "DATETIME"
	case schema.FieldTypeJSON:
		return "JSON"
	default:
		return "VARCHAR(255)"
	}
}
