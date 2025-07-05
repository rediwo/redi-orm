package query

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/rediwo/redi-orm/schema"
)

// HierarchicalScanner handles scanning results with nested relations
type HierarchicalScanner struct {
	mainSchema       *schema.Schema
	mainAlias        string
	joinInfo         map[string]*JoinInfo // alias -> join information
	relationPaths    map[string]string    // alias -> full relation path (e.g., "posts.comments")
	includeProcessor *IncludeProcessor    // For filtering and field selection
}

// JoinInfo contains information about a joined table
type JoinInfo struct {
	Schema       *schema.Schema
	Relation     *schema.Relation
	RelationName string
	ParentAlias  string // alias of the parent table
	Path         string // full relation path
}

// RecordNode represents a record with its nested relations
type RecordNode struct {
	Data     map[string]any
	Children map[string]map[any]*RecordNode // relationName -> id -> child node
}

// NewHierarchicalScanner creates a new hierarchical scanner
func NewHierarchicalScanner(mainSchema *schema.Schema, mainAlias string) *HierarchicalScanner {
	return &HierarchicalScanner{
		mainSchema:    mainSchema,
		mainAlias:     mainAlias,
		joinInfo:      make(map[string]*JoinInfo),
		relationPaths: make(map[string]string),
	}
}

// SetIncludeProcessor sets the include processor for filtering and field selection
func (hs *HierarchicalScanner) SetIncludeProcessor(processor *IncludeProcessor) {
	hs.includeProcessor = processor
}

// AddJoinedTable adds information about a joined table with its parent
func (hs *HierarchicalScanner) AddJoinedTable(alias string, schema *schema.Schema, relation *schema.Relation, relationName string, parentAlias string, path string) {
	// fmt.Printf("[DEBUG] AddJoinedTable: alias=%s, relation=%s, path=%s, parent=%s\n", alias, relationName, path, parentAlias)
	hs.joinInfo[alias] = &JoinInfo{
		Schema:       schema,
		Relation:     relation,
		RelationName: relationName,
		ParentAlias:  parentAlias,
		Path:         path,
	}
	hs.relationPaths[alias] = path
}

// ScanRowsWithRelations scans rows with nested relations
func (hs *HierarchicalScanner) ScanRowsWithRelations(rows *sql.Rows, dest any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	sliceValue := destValue.Elem()
	elementType := sliceValue.Type().Elem()

	// Check if destination is []map[string]any
	if elementType.Kind() == reflect.Map &&
		elementType.Key().Kind() == reflect.String &&
		elementType.Elem().Kind() == reflect.Interface &&
		elementType.Elem().NumMethod() == 0 {
		return hs.scanRowsToMapsHierarchical(rows, dest)
	}

	return fmt.Errorf("struct scanning with hierarchical relations not yet implemented")
}

// scanRowsToMapsHierarchical scans rows into a hierarchical structure
func (hs *HierarchicalScanner) scanRowsToMapsHierarchical(rows *sql.Rows, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Parse column names to determine which table they belong to
	columnInfo := hs.parseColumns(columns)

	// Track records at each level
	mainRecords := make(map[any]*RecordNode)
	var recordOrder []any

	// Track all records by alias and ID for linking
	allRecords := make(map[string]map[any]*RecordNode) // alias -> id -> record
	for alias := range hs.joinInfo {
		allRecords[alias] = make(map[any]*RecordNode)
	}
	allRecords[hs.mainAlias] = mainRecords

	// Create value holders
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// First pass: scan all rows and build record nodes
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Build record maps for each table
		recordMaps := make(map[string]map[string]any)

		for i, col := range columns {
			val := values[i]
			// Handle byte arrays
			if b, ok := val.([]byte); ok {
				val = string(b)
			}

			// Determine which table this column belongs to
			tableAlias, fieldName := hs.parseColumnName(col, columnInfo)

			if _, exists := recordMaps[tableAlias]; !exists {
				recordMaps[tableAlias] = make(map[string]any)
			}

			// Map column back to schema field name
			if tableAlias == hs.mainAlias {
				if hs.mainSchema != nil {
					if mapped, err := hs.mainSchema.GetFieldNameByColumnName(fieldName); err == nil {
						fieldName = mapped
					}
				}
			} else if info, exists := hs.joinInfo[tableAlias]; exists && info.Schema != nil {
				if mapped, err := info.Schema.GetFieldNameByColumnName(fieldName); err == nil {
					fieldName = mapped
				}
			}

			recordMaps[tableAlias][fieldName] = val
		}

		// Process main record
		mainData := recordMaps[hs.mainAlias]
		if mainData == nil || isNullRecord(mainData) {
			continue
		}

		mainID := mainData["id"]
		var mainNode *RecordNode
		if existing, exists := mainRecords[mainID]; exists {
			mainNode = existing
		} else {
			mainNode = &RecordNode{
				Data:     mainData,
				Children: make(map[string]map[any]*RecordNode),
			}
			mainRecords[mainID] = mainNode
			recordOrder = append(recordOrder, mainID)
		}

		// Process joined records hierarchically
		// First, collect all aliases and sort them to ensure parent tables are processed before child tables
		aliases := make([]string, 0, len(recordMaps))
		for alias := range recordMaps {
			if alias != hs.mainAlias {
				aliases = append(aliases, alias)
			}
		}
		
		// Sort aliases so parent aliases come before child aliases
		// This is important for self-referential and nested relations
		sort.Slice(aliases, func(i, j int) bool {
			// Check if one is a parent of the other
			infoI := hs.joinInfo[aliases[i]]
			infoJ := hs.joinInfo[aliases[j]]
			
			if infoI != nil && infoJ != nil {
				// If j's parent is i, then i should come first
				if infoJ.ParentAlias == aliases[i] {
					return true
				}
				// If i's parent is j, then j should come first
				if infoI.ParentAlias == aliases[j] {
					return false
				}
				// Otherwise, sort by path length (shorter paths first)
				return len(strings.Split(infoI.Path, ".")) < len(strings.Split(infoJ.Path, "."))
			}
			return aliases[i] < aliases[j]
		})
		
		for _, alias := range aliases {
			recordData := recordMaps[alias]
			if isNullRecord(recordData) {
				continue
			}

			info, exists := hs.joinInfo[alias]
			if !exists {
				// fmt.Printf("[DEBUG] No join info for alias: %s\n", alias)
				continue
			}
			// fmt.Printf("[DEBUG] Processing joined record for alias: %s, relation: %s, path: %s, parent: %s\n", alias, info.RelationName, info.Path, info.ParentAlias)

			recordID := recordData["id"]
			if recordID == nil {
				continue
			}

			// Check if we've seen this record before
			var node *RecordNode
			if existing, exists := allRecords[alias][recordID]; exists {
				node = existing
			} else {
				node = &RecordNode{
					Data:     recordData,
					Children: make(map[string]map[any]*RecordNode),
				}
				allRecords[alias][recordID] = node
			}

			// Link to parent
			if info.ParentAlias == hs.mainAlias {
				// Direct child of main record
				if _, exists := mainNode.Children[info.RelationName]; !exists {
					mainNode.Children[info.RelationName] = make(map[any]*RecordNode)
				}
				mainNode.Children[info.RelationName][recordID] = node
			} else {
				// Nested child - find parent
				parentInfo := hs.joinInfo[info.ParentAlias]
				if parentInfo != nil {
					// For nested relations, we need to find the parent based on the relation type
					var parentID any

					// For nested relations, we always look for the parent in the current row
					// The parent record should be in the same row as this child
					parentRecordData := recordMaps[info.ParentAlias]
					if parentRecordData != nil && !isNullRecord(parentRecordData) {
						parentID = parentRecordData["id"]
					}

					// fmt.Printf("[DEBUG] Nested child: looking for parent %s with ID %v (relation type: %v)\n", info.ParentAlias, parentID, info.Relation.Type)

					if parentID != nil {
						if parentNode, exists := allRecords[info.ParentAlias][parentID]; exists {
							if _, exists := parentNode.Children[info.RelationName]; !exists {
								parentNode.Children[info.RelationName] = make(map[any]*RecordNode)
							}
							// Check if this is a single-value relation (many-to-one or one-to-one)
							if info.Relation.Type == schema.RelationManyToOne || info.Relation.Type == schema.RelationOneToOne {
								// For single-value relations, we should only have one child
								// Clear any existing entries (in case of duplicates)
								parentNode.Children[info.RelationName] = make(map[any]*RecordNode)
							}
							parentNode.Children[info.RelationName][recordID] = node
						} else {
						}
					}
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Build final result by converting RecordNodes to maps
	results := make([]map[string]any, 0, len(recordOrder))
	for _, id := range recordOrder {
		if node, exists := mainRecords[id]; exists {
			results = append(results, hs.nodeToMap(node))
		}
	}

	// Set results to destination
	destValue := reflect.ValueOf(dest).Elem()
	destValue.Set(reflect.ValueOf(results))

	return nil
}

// nodeToMap converts a RecordNode to a map with nested relations
func (hs *HierarchicalScanner) nodeToMap(node *RecordNode) map[string]any {
	result := make(map[string]any)

	// Copy data fields
	for k, v := range node.Data {
		result[k] = v
	}

	// Add nested relations
	for relationName, children := range node.Children {
		// fmt.Printf("[DEBUG] nodeToMap: processing relation %s with %d children\n", relationName, len(children))
		// Find the relation info to determine if it's one-to-many or many-to-one
		var relationType schema.RelationType
		for _, info := range hs.joinInfo {
			if info.RelationName == relationName {
				relationType = info.Relation.Type
				break
			}
		}

		switch relationType {
		case schema.RelationOneToMany:
			// Convert to array
			childArray := make([]any, 0, len(children))
			for _, childNode := range children {
				childArray = append(childArray, hs.nodeToMap(childNode))
			}

			// Apply include processor filtering if available
			if hs.includeProcessor != nil {
				// Find the relation path for this relation
				relationPath := ""
				for _, info := range hs.joinInfo {
					if info.RelationName == relationName {
						relationPath = info.Path
						break
					}
				}

				if relationPath != "" {
					// Convert to map array for processing
					mapArray := make([]map[string]any, len(childArray))
					for i, child := range childArray {
						if m, ok := child.(map[string]any); ok {
							mapArray[i] = m
						}
					}

					// Process with include options
					mapArray = hs.includeProcessor.ProcessRelationData(relationPath, mapArray)

					// Convert back to any array
					childArray = make([]any, len(mapArray))
					for i, m := range mapArray {
						childArray[i] = m
					}
				}
			}

			result[relationName] = childArray

		case schema.RelationManyToOne, schema.RelationOneToOne:
			// Single value - take the first (should only be one)
			for _, childNode := range children {
				result[relationName] = hs.nodeToMap(childNode)
				break
			}
		}
	}

	return result
}

// parseColumns analyzes column names to determine table aliases
func (hs *HierarchicalScanner) parseColumns(columns []string) map[string]string {
	info := make(map[string]string)

	for _, col := range columns {
		// Try to parse table_column format (from aliased columns)
		if parts := strings.Split(col, "_"); len(parts) >= 2 {
			// First part is table alias
			info[col] = parts[0]
		} else if parts := strings.Split(col, "."); len(parts) == 2 {
			// Try to parse table.column format
			info[col] = parts[0]
		} else {
			// Assume main table if no prefix
			info[col] = hs.mainAlias
		}
	}

	return info
}

// parseColumnName extracts table alias and field name from a column
func (hs *HierarchicalScanner) parseColumnName(column string, columnInfo map[string]string) (tableAlias, fieldName string) {
	if alias, exists := columnInfo[column]; exists {
		tableAlias = alias
	} else {
		tableAlias = hs.mainAlias
	}

	// Remove table prefix if present
	if parts := strings.Split(column, "_"); len(parts) >= 2 && parts[0] == tableAlias {
		// Handle aliased format: tableAlias_columnName
		fieldName = strings.Join(parts[1:], "_")
	} else if parts := strings.Split(column, "."); len(parts) == 2 {
		// Handle dot format: table.column
		fieldName = parts[1]
	} else {
		fieldName = column
	}

	return tableAlias, fieldName
}
