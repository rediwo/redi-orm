package drivers

import (
	"github.com/rediwo/redi-orm/schema"
)

// fieldTypeToSQL converts schema field types to SQLite column types
// SQLite type mapping:
// - All text types map to TEXT
// - All integer types map to INTEGER 
// - Float types map to REAL
// - Boolean maps to INTEGER (0/1)
// - JSON is stored as TEXT
func fieldTypeToSQL(ft schema.FieldType) string {
	switch ft {
	case schema.FieldTypeString:
		return "TEXT"
	case schema.FieldTypeInt, schema.FieldTypeInt64:
		return "INTEGER"
	case schema.FieldTypeFloat:
		return "REAL"
	case schema.FieldTypeBool:
		return "INTEGER"
	case schema.FieldTypeDateTime:
		return "DATETIME"
	case schema.FieldTypeJSON:
		return "TEXT"
	case schema.FieldTypeDecimal:
		return "DECIMAL" // SQLite stores as TEXT but preserves decimal semantics
	// Array types - SQLite stores arrays as JSON TEXT
	case schema.FieldTypeStringArray, schema.FieldTypeIntArray, schema.FieldTypeInt64Array, 
		 schema.FieldTypeFloatArray, schema.FieldTypeBoolArray, schema.FieldTypeDecimalArray, 
		 schema.FieldTypeDateTimeArray:
		return "TEXT" // Arrays stored as JSON in SQLite
	default:
		return "TEXT"
	}
}