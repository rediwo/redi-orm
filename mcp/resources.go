package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/utils"
)

// ListResources returns available resources
func (s *Server) ListResources(ctx context.Context) ([]Resource, error) {
	resources := []Resource{
		{
			URI:         "schema://database",
			Name:        "Database Schema",
			Description: "Complete database schema information",
			MimeType:    "application/json",
		},
	}

	// Add schema resources if schemas are loaded
	for modelName := range s.schemas {
		resources = append(resources, Resource{
			URI:         fmt.Sprintf("model://%s", modelName),
			Name:        fmt.Sprintf("%s Model", modelName),
			Description: fmt.Sprintf("Prisma model definition for %s", modelName),
			MimeType:    "text/plain",
		})
	}

	// Add table resources if database is connected
	if s.db != nil {
		migrator := s.db.GetMigrator()
		tables, err := migrator.GetTables()
		if err != nil {
			s.logger.Error("Failed to list tables: %v", err)
			// Continue without table resources
		} else {
			for _, table := range tables {
				// Check if table is allowed
				if !s.isTableAllowed(table) {
					continue
				}

				resources = append(resources, Resource{
					URI:         fmt.Sprintf("table://%s", table),
					Name:        fmt.Sprintf("%s Table Schema", table),
					Description: fmt.Sprintf("Schema information for %s table", table),
					MimeType:    "application/json",
				})

				resources = append(resources, Resource{
					URI:         fmt.Sprintf("data://%s", table),
					Name:        fmt.Sprintf("%s Data", table),
					Description: fmt.Sprintf("Data from %s table (max %d rows)", table, s.config.MaxQueryRows),
					MimeType:    "application/json",
				})
			}
		}
	}

	return resources, nil
}

// ReadResource handles resource read requests
func (s *Server) ReadResource(ctx context.Context, uri string) (*ResourceContent, error) {
	s.logger.Debug("Reading resource: %s", uri)

	switch {
	case strings.HasPrefix(uri, ResourceSchemaPrefix):
		return s.readSchemaResource(ctx, uri)
	case strings.HasPrefix(uri, ResourceTablePrefix):
		return s.readTableResource(ctx, uri)
	case strings.HasPrefix(uri, ResourceDataPrefix):
		return s.readDataResource(ctx, uri)
	case strings.HasPrefix(uri, ResourceModelPrefix):
		return s.readModelResource(ctx, uri)
	default:
		return nil, fmt.Errorf("unknown resource URI: %s", uri)
	}
}

// readSchemaResource returns the complete database schema
func (s *Server) readSchemaResource(ctx context.Context, uri string) (*ResourceContent, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get all tables using migrator
	migrator := s.db.GetMigrator()
	tables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	// Build schema information
	schemaInfo := make(map[string]interface{})
	schemaInfo["database_type"] = s.db.GetDriverType()
	
	tableSchemas := make(map[string]interface{})
	for _, table := range tables {
		if !s.isTableAllowed(table) {
			continue
		}

		// Get table schema
		tableInfo, err := s.getTableSchema(ctx, table)
		if err != nil {
			s.logger.Warn("Failed to get schema for table %s: %v", table, err)
			continue
		}
		tableSchemas[table] = tableInfo
	}
	schemaInfo["tables"] = tableSchemas

	// Add loaded models
	if len(s.schemas) > 0 {
		models := make(map[string]interface{})
		for name, schema := range s.schemas {
			models[name] = s.schemaToJSON(schema)
		}
		schemaInfo["models"] = models
	}

	// Convert to JSON
	data, err := json.MarshalIndent(schemaInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return &ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// readTableResource returns schema information for a specific table
func (s *Server) readTableResource(ctx context.Context, uri string) (*ResourceContent, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Extract table name
	tableName := strings.TrimPrefix(uri, ResourceTablePrefix)
	if !s.isTableAllowed(tableName) {
		return nil, fmt.Errorf("table not allowed: %s", tableName)
	}

	// Get table schema
	tableInfo, err := s.getTableSchema(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table schema: %w", err)
	}

	// Convert to JSON
	data, err := json.MarshalIndent(tableInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal table schema: %w", err)
	}

	return &ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// readDataResource returns data from a specific table
func (s *Server) readDataResource(ctx context.Context, uri string) (*ResourceContent, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Parse URI to extract table name and query parameters
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	tableName := strings.TrimPrefix(parsedURI.Path, "//")
	if !s.isTableAllowed(tableName) {
		return nil, fmt.Errorf("table not allowed: %s", tableName)
	}

	// Parse query parameters
	limit := s.config.MaxQueryRows
	offset := 0
	
	if l := parsedURI.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > s.config.MaxQueryRows {
				limit = s.config.MaxQueryRows
			}
		}
	}
	
	if o := parsedURI.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	if s.db.GetDriverType() != "mongodb" {
		query += fmt.Sprintf(" LIMIT %d", limit)
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	// Execute query using Raw API
	var results []map[string]interface{}
	rawQuery := s.db.Raw(query)
	if err := rawQuery.Find(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}

	// Add metadata
	response := map[string]interface{}{
		"table":   tableName,
		"limit":   limit,
		"offset":  offset,
		"data":    results,
		"count":   len(results),
		"hasMore": len(results) == limit,
	}

	// Convert to JSON
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return &ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// readModelResource delegates to the enhanced model resource handler
func (s *Server) readModelResource(ctx context.Context, uri string) (*ResourceContent, error) {
	// Delegate to the enhanced model resource handler in resources_models.go
	return s.readModelResourceEnhanced(ctx, uri)
}

// getTableSchema retrieves schema information for a table
func (s *Server) getTableSchema(ctx context.Context, tableName string) (map[string]interface{}, error) {
	// Use DESCRIBE or equivalent for the database
	var query string
	switch s.db.GetDriverType() {
	case "mysql":
		query = fmt.Sprintf("DESCRIBE %s", tableName)
	case "postgresql":
		query = fmt.Sprintf(`
			SELECT 
				column_name,
				data_type,
				is_nullable,
				column_default,
				character_maximum_length
			FROM information_schema.columns
			WHERE table_name = '%s'
			ORDER BY ordinal_position`, tableName)
	case "sqlite":
		query = fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	case "mongodb":
		// MongoDB doesn't have fixed schemas
		return map[string]interface{}{
			"name":       tableName,
			"schemaless": true,
			"type":       "collection",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", s.db.GetDriverType())
	}

	// Execute query using Raw API
	var results []map[string]interface{}
	rawQuery := s.db.Raw(query)
	if err := rawQuery.Find(ctx, &results); err != nil {
		return nil, err
	}

	// Parse results based on database type
	columns := s.parseTableColumns(results)
	
	// Get indexes
	indexes, err := s.getTableIndexes(ctx, tableName)
	if err != nil {
		s.logger.Warn("Failed to get indexes for table %s: %v", tableName, err)
		indexes = []interface{}{}
	}

	// Build table info
	tableInfo := map[string]interface{}{
		"name":    tableName,
		"columns": columns,
		"indexes": indexes,
	}

	return tableInfo, nil
}

// parseTableColumns parses column information from query results
func (s *Server) parseTableColumns(results []map[string]interface{}) []map[string]interface{} {
	columns := make([]map[string]interface{}, 0, len(results))
	
	for _, row := range results {
		column := make(map[string]interface{})
		
		switch s.db.GetDriverType() {
		case "mysql":
			column["name"] = utils.ToString(row["Field"])
			column["type"] = utils.ToString(row["Type"])
			column["nullable"] = utils.ToString(row["Null"]) == "YES"
			column["key"] = utils.ToString(row["Key"])
			column["default"] = row["Default"]
			column["extra"] = utils.ToString(row["Extra"])
			
		case "postgresql":
			column["name"] = utils.ToString(row["column_name"])
			column["type"] = utils.ToString(row["data_type"])
			column["nullable"] = utils.ToString(row["is_nullable"]) == "YES"
			column["default"] = row["column_default"]
			if maxLen := row["character_maximum_length"]; maxLen != nil {
				column["max_length"] = utils.ToInt(maxLen)
			}
			
		case "sqlite":
			column["name"] = utils.ToString(row["name"])
			column["type"] = utils.ToString(row["type"])
			column["nullable"] = utils.ToInt(row["notnull"]) == 0
			column["default"] = row["dflt_value"]
			column["primary_key"] = utils.ToInt(row["pk"]) > 0
		}
		
		columns = append(columns, column)
	}
	
	return columns
}

// getTableIndexes retrieves index information for a table
func (s *Server) getTableIndexes(ctx context.Context, tableName string) ([]interface{}, error) {
	var query string
	switch s.db.GetDriverType() {
	case "mysql":
		query = fmt.Sprintf("SHOW INDEXES FROM %s", tableName)
	case "postgresql":
		query = fmt.Sprintf(`
			SELECT 
				indexname as index_name,
				indexdef as definition
			FROM pg_indexes
			WHERE tablename = '%s'`, tableName)
	case "sqlite":
		query = fmt.Sprintf("PRAGMA index_list(%s)", tableName)
	default:
		return []interface{}{}, nil
	}

	var results []map[string]interface{}
	rawQuery := s.db.Raw(query)
	if err := rawQuery.Find(ctx, &results); err != nil {
		return nil, err
	}

	indexes := make([]interface{}, 0, len(results))
	for _, row := range results {
		indexes = append(indexes, row)
	}
	
	return indexes, nil
}

// isTableAllowed checks if a table is allowed based on configuration
func (s *Server) isTableAllowed(tableName string) bool {
	if len(s.config.AllowedTables) == 0 {
		return true
	}
	
	for _, allowed := range s.config.AllowedTables {
		if allowed == tableName {
			return true
		}
	}
	
	return false
}

// schemaToJSON converts internal schema to JSON representation
func (s *Server) schemaToJSON(sch *schema.Schema) map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(sch.Fields))
	for _, field := range sch.Fields {
		fieldInfo := map[string]interface{}{
			"name":          field.Name,
			"type":          string(field.Type),
			"nullable":      field.Nullable,
			"primary_key":   field.PrimaryKey,
			"auto_increment": field.AutoIncrement,
			"unique":        field.Unique,
			"default":       field.Default,
			"map":           field.Map,
		}
		fields = append(fields, fieldInfo)
	}

	relations := make([]map[string]interface{}, 0, len(sch.Relations))
	for name, relation := range sch.Relations {
		relationInfo := map[string]interface{}{
			"name":       name,
			"type":       string(relation.Type),
			"model":      relation.Model,
			"foreign_key": relation.ForeignKey,
			"references": relation.References,
		}
		relations = append(relations, relationInfo)
	}

	indexes := make([]map[string]interface{}, 0, len(sch.Indexes))
	for _, index := range sch.Indexes {
		indexInfo := map[string]interface{}{
			"name":   index.Name,
			"fields": index.Fields,
			"unique": index.Unique,
		}
		indexes = append(indexes, indexInfo)
	}

	return map[string]interface{}{
		"name":          sch.Name,
		"table_name":    sch.GetTableName(),
		"fields":        fields,
		"relations":     relations,
		"indexes":       indexes,
		"composite_key": sch.CompositeKey,
	}
}

// generatePrismaModel generates Prisma-style model definition
func (s *Server) generatePrismaModel(modelName string, sch *schema.Schema) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("model %s {\n", modelName))
	
	// Add fields
	for _, field := range sch.Fields {
		sb.WriteString(fmt.Sprintf("  %s %s", field.Name, s.fieldTypeToPrisma(field.Type)))
		
		if field.Nullable {
			sb.WriteString("?")
		}
		
		// Add attributes
		var attrs []string
		if field.PrimaryKey {
			attrs = append(attrs, "@id")
		}
		if field.AutoIncrement {
			attrs = append(attrs, "@default(autoincrement())")
		}
		if field.Unique {
			attrs = append(attrs, "@unique")
		}
		if field.Default != nil {
			switch v := field.Default.(type) {
			case string:
				if v == "now()" || v == "CURRENT_TIMESTAMP" {
					attrs = append(attrs, "@default(now())")
				} else {
					attrs = append(attrs, fmt.Sprintf("@default(\"%s\")", v))
				}
			case bool:
				attrs = append(attrs, fmt.Sprintf("@default(%t)", v))
			default:
				attrs = append(attrs, fmt.Sprintf("@default(%v)", v))
			}
		}
		if field.Map != "" {
			attrs = append(attrs, fmt.Sprintf("@map(\"%s\")", field.Map))
		}
		
		if len(attrs) > 0 {
			sb.WriteString(" ")
			sb.WriteString(strings.Join(attrs, " "))
		}
		
		sb.WriteString("\n")
	}
	
	// Add relations
	for name, relation := range sch.Relations {
		switch relation.Type {
		case schema.RelationOneToMany:
			sb.WriteString(fmt.Sprintf("  %s %s[]\n", name, relation.Model))
		case schema.RelationManyToOne, schema.RelationOneToOne:
			sb.WriteString(fmt.Sprintf("  %s %s @relation(fields: [%s], references: [%s])\n",
				name, relation.Model, relation.ForeignKey, relation.References))
		}
	}
	
	// Add composite key if present
	if len(sch.CompositeKey) > 0 {
		sb.WriteString(fmt.Sprintf("  @@id([%s])\n", strings.Join(sch.CompositeKey, ", ")))
	}
	
	// Add indexes
	for _, index := range sch.Indexes {
		if index.Unique {
			sb.WriteString(fmt.Sprintf("  @@unique([%s])\n", strings.Join(index.Fields, ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("  @@index([%s])\n", strings.Join(index.Fields, ", ")))
		}
	}
	
	// Add table mapping if different from model name
	if tableName := sch.GetTableName(); tableName != utils.ToSnakeCase(utils.Pluralize(modelName)) {
		sb.WriteString(fmt.Sprintf("  @@map(\"%s\")\n", tableName))
	}
	
	sb.WriteString("}")
	
	return sb.String()
}

// fieldTypeToPrisma converts internal field type to Prisma type
func (s *Server) fieldTypeToPrisma(fieldType schema.FieldType) string {
	switch fieldType {
	case schema.FieldTypeString:
		return "String"
	case schema.FieldTypeInt:
		return "Int"
	case schema.FieldTypeInt64:
		return "BigInt"
	case schema.FieldTypeFloat:
		return "Float"
	case schema.FieldTypeDecimal:
		return "Decimal"
	case schema.FieldTypeBool:
		return "Boolean"
	case schema.FieldTypeDateTime:
		return "DateTime"
	case schema.FieldTypeJSON:
		return "Json"
	case schema.FieldTypeBinary:
		return "Bytes"
	default:
		return "String"
	}
}