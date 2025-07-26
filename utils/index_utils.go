package utils

import (
	"fmt"
	"strings"
)

// GenerateIndexName generates a consistent index name for database indexes
// If the index already has a name, it returns that name unchanged
// Otherwise, it generates a name based on the table name and fields
func GenerateIndexName(tableName string, fields []string, unique bool, existingName string) string {
	// If an existing name is provided, use it
	if existingName != "" {
		return existingName
	}

	// Generate name based on columns
	prefix := "idx"
	if unique {
		prefix = "uniq"
	}

	columnPart := strings.Join(fields, "_")
	return fmt.Sprintf("%s_%s_%s", prefix, tableName, columnPart)
}

// GenerateFieldIndexName generates an index name for a single field index
func GenerateFieldIndexName(tableName, fieldName string) string {
	return fmt.Sprintf("idx_%s_%s", tableName, fieldName)
}

// NormalizeIndexName normalizes index name for comparison
// Removes common prefixes and suffixes to allow flexible matching
func NormalizeIndexName(name string) string {
	// Convert to lowercase for case-insensitive comparison
	normalized := strings.ToLower(name)

	// Remove common prefixes
	normalized = strings.TrimPrefix(normalized, "idx_")
	normalized = strings.TrimPrefix(normalized, "index_")
	normalized = strings.TrimPrefix(normalized, "uniq_")
	normalized = strings.TrimPrefix(normalized, "unique_")

	// Remove common suffixes
	normalized = strings.TrimSuffix(normalized, "_idx")
	normalized = strings.TrimSuffix(normalized, "_index")

	return normalized
}
