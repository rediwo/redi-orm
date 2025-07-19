package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rediwo/redi-orm/schema"
)

// readModelResourceEnhanced handles model:// resource reads with enhanced features
func (s *Server) readModelResourceEnhanced(ctx context.Context, uri string) (*ResourceContent, error) {
	// Remove prefix first
	path := strings.TrimPrefix(uri, ResourceModelPrefix)
	
	// Separate the path from query parameters
	if queryIdx := strings.Index(path, "?"); queryIdx != -1 {
		path = path[:queryIdx]
	}
	
	// Handle different model resource paths
	switch {
	case path == "" || path == "/":
		// model:// - List all models
		return s.readAllModels(ctx)
	case strings.HasSuffix(path, "/schema"):
		// model://{name}/schema - Get Prisma schema for model
		modelName := strings.TrimSuffix(path, "/schema")
		return s.readModelSchema(ctx, modelName)
	case strings.HasSuffix(path, "/data"):
		// model://{name}/data - Get paginated data
		modelName := strings.TrimSuffix(path, "/data")
		return s.readModelData(ctx, modelName, uri)
	default:
		// model://{name} - Get model details
		return s.readModelDetails(ctx, path)
	}
}

// readAllModels returns a list of all models with metadata
func (s *Server) readAllModels(ctx context.Context) (*ResourceContent, error) {
	models := make([]map[string]interface{}, 0, len(s.schemas))
	
	for name, sch := range s.schemas {
		modelInfo := map[string]interface{}{
			"name": name,
			"fields": len(sch.Fields),
			"relations": len(sch.Relations),
			"indexes": len(sch.Indexes),
		}
		
		// Add field summary
		var primaryKey string
		requiredFields := 0
		for _, field := range sch.Fields {
			if field.PrimaryKey {
				primaryKey = field.Name
			}
			if !field.Nullable {
				requiredFields++
			}
		}
		modelInfo["primaryKey"] = primaryKey
		modelInfo["requiredFields"] = requiredFields
		
		models = append(models, modelInfo)
	}
	
	response := map[string]interface{}{
		"models": models,
		"count": len(models),
	}
	
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal models: %w", err)
	}
	
	return &ResourceContent{
		URI:      "model://",
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// readModelDetails returns detailed information about a specific model
func (s *Server) readModelDetails(ctx context.Context, modelName string) (*ResourceContent, error) {
	sch, exists := s.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", modelName)
	}
	
	// Build detailed model information
	modelInfo := map[string]interface{}{
		"name": modelName,
		"fields": convertFieldsToJSON(sch.Fields),
		"relations": convertRelationsToJSON(sch.Relations),
		"indexes": convertIndexesToJSON(sch.Indexes),
		"compositeKey": sch.CompositeKey,
	}
	
	data, err := json.MarshalIndent(modelInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal model details: %w", err)
	}
	
	return &ResourceContent{
		URI:      fmt.Sprintf("model://%s", modelName),
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// readModelSchema returns the Prisma schema definition for a model
func (s *Server) readModelSchema(ctx context.Context, modelName string) (*ResourceContent, error) {
	sch, exists := s.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", modelName)
	}
	
	// Generate Prisma schema format
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("model %s {\n", modelName))
	
	// Fields
	for _, field := range sch.Fields {
		sb.WriteString(fmt.Sprintf("  %s %s", field.Name, s.fieldTypeToPrisma(field.Type)))
		
		// Add nullable marker immediately after type
		if field.Nullable {
			sb.WriteString("?")
		}
		
		// Add attributes
		if field.PrimaryKey {
			sb.WriteString(" @id")
		}
		if field.AutoIncrement {
			sb.WriteString(" @default(autoincrement())")
		}
		if field.Unique {
			sb.WriteString(" @unique")
		}
		if field.Default != nil {
			sb.WriteString(fmt.Sprintf(" @default(%v)", field.Default))
		}
		if field.Map != "" {
			sb.WriteString(fmt.Sprintf(" @map(\"%s\")", field.Map))
		}
		sb.WriteString("\n")
	}
	
	// Relations
	for name, rel := range sch.Relations {
		sb.WriteString(fmt.Sprintf("  %s %s", name, rel.Model))
		// Check relation type to determine if it's an array
		if rel.Type == schema.RelationOneToMany {
			sb.WriteString("[]")
		}
		if rel.ForeignKey != "" {
			sb.WriteString(fmt.Sprintf(" @relation(fields: [%s], references: [%s])", 
				rel.ForeignKey, rel.References))
		}
		sb.WriteString("\n")
	}
	
	// Composite keys
	if len(sch.CompositeKey) > 0 {
		sb.WriteString(fmt.Sprintf("\n  @@id([%s])", strings.Join(sch.CompositeKey, ", ")))
	}
	
	// Indexes
	for _, idx := range sch.Indexes {
		if idx.Unique {
			sb.WriteString(fmt.Sprintf("\n  @@unique([%s])", strings.Join(idx.Fields, ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("\n  @@index([%s])", strings.Join(idx.Fields, ", ")))
		}
	}
	
	sb.WriteString("\n}")
	
	return &ResourceContent{
		URI:      fmt.Sprintf("model://%s/schema", modelName),
		MimeType: "text/plain",
		Text:     sb.String(),
	}, nil
}

// readModelData returns paginated data for a model
func (s *Server) readModelData(ctx context.Context, modelName string, uri string) (*ResourceContent, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	
	sch, exists := s.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", modelName)
	}
	
	// Parse query parameters
	params := parseResourceParams(uri)
	limit := params.limit
	if limit > s.config.MaxQueryRows {
		limit = s.config.MaxQueryRows
	}
	offset := params.offset
	
	// Convert model name to table name
	tableName := schema.ModelNameToTableName(modelName)
	
	// Check if table is allowed
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}
	
	// Build query
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, limit, offset)
	
	// Execute query
	var results []map[string]interface{}
	if err := s.db.Raw(query).Find(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	
	// Convert results from database columns to model fields
	modelResults := make([]map[string]interface{}, len(results))
	for i, row := range results {
		modelResults[i] = convertRowToModelFields(row, sch)
	}
	
	// Count total records
	var countResult []map[string]interface{}
	countQuery := fmt.Sprintf("SELECT COUNT(*) as total FROM %s", tableName)
	if err := s.db.Raw(countQuery).Find(ctx, &countResult); err == nil && len(countResult) > 0 {
		if total, ok := countResult[0]["total"]; ok {
			response := map[string]interface{}{
				"model": modelName,
				"data": modelResults,
				"pagination": map[string]interface{}{
					"limit": limit,
					"offset": offset,
					"total": total,
				},
			}
			
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
	}
	
	// Fallback without count
	response := map[string]interface{}{
		"model": modelName,
		"data": modelResults,
		"pagination": map[string]interface{}{
			"limit": limit,
			"offset": offset,
		},
	}
	
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

// Helper functions

// parseResourceParams extracts query parameters from resource URIs
func parseResourceParams(uri string) struct {
	limit  int
	offset int
} {
	params := struct {
		limit  int
		offset int
	}{
		limit:  100,
		offset: 0,
	}
	
	// Parse URI to extract query parameters
	if parsedURI, err := url.Parse(uri); err == nil {
		if l := parsedURI.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				params.limit = parsed
			}
		}
		
		if o := parsedURI.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				params.offset = parsed
			}
		}
	}
	
	return params
}

func convertFieldsToJSON(fields []schema.Field) []map[string]interface{} {
	result := make([]map[string]interface{}, len(fields))
	for i, field := range fields {
		result[i] = map[string]interface{}{
			"name":          field.Name,
			"type":          field.Type,
			"nullable":      field.Nullable,
			"primaryKey":    field.PrimaryKey,
			"autoIncrement": field.AutoIncrement,
			"unique":        field.Unique,
			"index":         field.Index,
			"default":       field.Default,
			"map":           field.Map,
		}
	}
	return result
}

func convertRelationsToJSON(relations map[string]schema.Relation) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(relations))
	for name, rel := range relations {
		result = append(result, map[string]interface{}{
			"name":       name,
			"type":       string(rel.Type),
			"model":      rel.Model,
			"foreignKey": rel.ForeignKey,
			"references": rel.References,
			"isArray":    rel.Type == schema.RelationOneToMany,
		})
	}
	return result
}

func convertIndexesToJSON(indexes []schema.Index) []map[string]interface{} {
	result := make([]map[string]interface{}, len(indexes))
	for i, idx := range indexes {
		result[i] = map[string]interface{}{
			"name":   idx.Name,
			"fields": idx.Fields,
			"unique": idx.Unique,
		}
	}
	return result
}

func convertRowToModelFields(row map[string]interface{}, sch *schema.Schema) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Convert database columns to model fields
	for _, field := range sch.Fields {
		columnName := field.Name
		if field.Map != "" {
			columnName = field.Map
		}
		
		// Try different case variations
		if value, exists := row[columnName]; exists {
			result[field.Name] = value
		} else if value, exists := row[strings.ToLower(columnName)]; exists {
			result[field.Name] = value
		} else if value, exists := row[strings.ToUpper(columnName)]; exists {
			result[field.Name] = value
		}
	}
	
	return result
}

