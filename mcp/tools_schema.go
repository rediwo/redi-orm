package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
)

// Model management tool input structures

type ModelCreateInput struct {
	Name       string                 `json:"name"`
	Fields     []FieldDefinition      `json:"fields"`
	Relations  []RelationDefinition   `json:"relations,omitempty"`
	Indexes    []IndexDefinition      `json:"indexes,omitempty"`
	Attributes []string               `json:"attributes,omitempty"`
}

type ModelAddFieldInput struct {
	Model string           `json:"model"`
	Field FieldDefinition  `json:"field"`
}

type ModelRemoveFieldInput struct {
	Model     string `json:"model"`
	FieldName string `json:"field_name"`
}

type ModelAddRelationInput struct {
	Model    string             `json:"model"`
	Relation RelationDefinition `json:"relation"`
}

type ModelUpdateInput struct {
	Model   string                 `json:"model"`
	Changes map[string]interface{} `json:"changes"`
}

type ModelDeleteInput struct {
	Model string `json:"model"`
	Force bool   `json:"force,omitempty"`
}

// Schema management tool input structures

type SchemaSyncInput struct {
	Force        bool `json:"force,omitempty"`
	DryRun       bool `json:"dry_run,omitempty"`
	IncludeDrop  bool `json:"include_drop,omitempty"`
}

type SchemaDiffInput struct {
	Detailed bool `json:"detailed,omitempty"`
}

type SchemaExportInput struct {
	Format string `json:"format,omitempty"` // "prisma" or "json"
}

type SchemaImportInput struct {
	Schema string `json:"schema"`
	Format string `json:"format,omitempty"` // "prisma" or "json"
}

// createModel implements model creation
func (s *Server) createModel(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ModelCreateInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model name
	if input.Name == "" {
		return nil, fmt.Errorf("model name is required")
	}

	// Check if model already exists
	if _, exists := s.schemas[input.Name]; exists {
		return nil, fmt.Errorf("model '%s' already exists", input.Name)
	}

	// Create new schema
	newSchema := schema.New(input.Name)

	// Add fields
	for _, fieldDef := range input.Fields {
		field := schema.Field{
			Name: fieldDef.Name,
			Type: parseFieldType(fieldDef.Type),
		}

		// Parse attributes
		for _, attr := range fieldDef.Attributes {
			switch attr {
			case "@id":
				field.PrimaryKey = true
			case "@unique":
				field.Unique = true
			case "@index":
				field.Index = true
			case "@default(autoincrement())":
				field.AutoIncrement = true
			case "@default(now())", "@default(CURRENT_TIMESTAMP)":
				field.Default = "now()"
			default:
				if strings.HasPrefix(attr, "@default(") && strings.HasSuffix(attr, ")") {
					defaultVal := strings.TrimSuffix(strings.TrimPrefix(attr, "@default("), ")")
					field.Default = parseDefaultValue(defaultVal)
				} else if strings.HasPrefix(attr, "@map(") && strings.HasSuffix(attr, ")") {
					mapVal := strings.TrimSuffix(strings.TrimPrefix(attr, "@map("), ")")
					field.Map = strings.Trim(mapVal, "\"")
				}
			}
		}

		// Check for nullable (? suffix)
		if strings.HasSuffix(fieldDef.Type, "?") {
			field.Nullable = true
		}

		// Set default value if provided
		if fieldDef.Default != nil {
			field.Default = fieldDef.Default
		}

		newSchema.AddField(field)
	}

	// Add relations
	for _, relDef := range input.Relations {
		rel := schema.Relation{
			Model:      relDef.Model,
			ForeignKey: strings.Join(relDef.Fields, ","),
			References: strings.Join(relDef.References, ","),
		}

		// Determine relation type
		rel.Type = parseRelationType(relDef.Type)

		newSchema.AddRelation(relDef.Name, rel)
	}

	// Add indexes
	for _, idxDef := range input.Indexes {
		idx := schema.Index{
			Name:   idxDef.Name,
			Fields: idxDef.Fields,
			Unique: idxDef.Type == "UNIQUE",
		}
		newSchema.Indexes = append(newSchema.Indexes, idx)
	}

	// Register the new schema
	if err := s.db.RegisterSchema(input.Name, newSchema); err != nil {
		return nil, fmt.Errorf("failed to register model: %w", err)
	}

	// Add to server's schema map
	s.schemas[input.Name] = newSchema

	// Sync to database if not in read-only mode
	if !s.config.ReadOnly {
		if err := s.db.CreateModel(ctx, input.Name); err != nil {
			s.logger.Warn("Failed to create model in database: %v", err)
		}
	}

	// Generate response
	response := map[string]interface{}{
		"success": true,
		"model":   input.Name,
		"message": fmt.Sprintf("Model '%s' created successfully", input.Name),
		"schema":  s.generatePrismaModel(input.Name, newSchema),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// addFieldToModel implements field addition to existing model
func (s *Server) addFieldToModel(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ModelAddFieldInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Get existing schema
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Check if field already exists
	for _, field := range sch.Fields {
		if field.Name == input.Field.Name {
			return nil, fmt.Errorf("field '%s' already exists in model '%s'", input.Field.Name, input.Model)
		}
	}

	// Create new field
	field := schema.Field{
		Name: input.Field.Name,
		Type: parseFieldType(input.Field.Type),
	}

	// Parse attributes
	for _, attr := range input.Field.Attributes {
		switch attr {
		case "@unique":
			field.Unique = true
		case "@index":
			field.Index = true
		case "@default(now())":
			field.Default = "now()"
		default:
			if strings.HasPrefix(attr, "@default(") {
				defaultVal := strings.TrimSuffix(strings.TrimPrefix(attr, "@default("), ")")
				field.Default = parseDefaultValue(defaultVal)
			}
		}
	}

	// Check for nullable
	if strings.HasSuffix(input.Field.Type, "?") {
		field.Nullable = true
	}

	// Add field to schema
	sch.AddField(field)

	// Generate response
	response := map[string]interface{}{
		"success": true,
		"model":   input.Model,
		"field":   input.Field.Name,
		"message": fmt.Sprintf("Field '%s' added to model '%s'", input.Field.Name, input.Model),
		"schema":  s.generatePrismaModel(input.Model, sch),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// syncSchema implements schema synchronization
func (s *Server) syncSchema(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input SchemaSyncInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Check read-only mode
	if s.config.ReadOnly && !input.DryRun {
		return nil, fmt.Errorf("cannot sync schemas in read-only mode")
	}

	// Get current database state
	migrator := s.db.GetMigrator()
	tables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get current tables: %w", err)
	}

	// Prepare sync report
	changes := []map[string]interface{}{}

	// Check each schema
	for modelName := range s.schemas {
		tableName := schema.ModelNameToTableName(modelName)
		exists := false
		for _, table := range tables {
			if table == tableName {
				exists = true
				break
			}
		}

		if !exists {
			changes = append(changes, map[string]interface{}{
				"type":  "create_table",
				"model": modelName,
				"table": tableName,
			})
		} else {
			// Check for field changes
			// This would require comparing current schema with database schema
			// For now, we'll just note that the table exists
			changes = append(changes, map[string]interface{}{
				"type":  "table_exists",
				"model": modelName,
				"table": tableName,
			})
		}
	}

	// If dry run, just return the changes
	if input.DryRun {
		response := map[string]interface{}{
			"dry_run": true,
			"changes": changes,
			"message": "Dry run completed. No changes were applied.",
		}

		data, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: string(data),
				},
			},
		}, nil
	}

	// Apply changes
	if err := s.db.SyncSchemas(ctx); err != nil {
		return nil, fmt.Errorf("failed to sync schemas: %w", err)
	}

	response := map[string]interface{}{
		"success": true,
		"changes": changes,
		"message": "Schemas synchronized successfully",
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// schemaDiff shows differences between schema and database
func (s *Server) schemaDiff(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input SchemaDiffInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get current database state
	migrator := s.db.GetMigrator()
	tables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get current tables: %w", err)
	}

	// Build diff report
	diff := map[string]interface{}{
		"models_only_in_schema":   []string{},
		"tables_only_in_database": []string{},
		"synchronized":            []string{},
		"field_differences":       []map[string]interface{}{},
	}

	// Track which tables we've seen
	seenTables := make(map[string]bool)

	// Check each schema
	for modelName := range s.schemas {
		tableName := schema.ModelNameToTableName(modelName)
		seenTables[tableName] = true
		
		exists := false
		for _, table := range tables {
			if table == tableName {
				exists = true
				break
			}
		}

		if !exists {
			diff["models_only_in_schema"] = append(diff["models_only_in_schema"].([]string), modelName)
		} else {
			diff["synchronized"] = append(diff["synchronized"].([]string), modelName)
			
			// If detailed diff requested, check field differences
			if input.Detailed {
				// This would require comparing schema fields with database columns
				// For now, we'll just note it's synchronized
			}
		}
	}

	// Check for tables without schemas
	for _, table := range tables {
		if !seenTables[table] && !isSystemTable(table) {
			diff["tables_only_in_database"] = append(diff["tables_only_in_database"].([]string), table)
		}
	}

	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal diff: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// schemaExport exports the current schema
func (s *Server) schemaExport(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input SchemaExportInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Format == "" {
		input.Format = "prisma"
	}

	var output string

	switch input.Format {
	case "prisma":
		// Generate Prisma schema format
		var sb strings.Builder
		sb.WriteString("// Generated Prisma Schema\n\n")
		
		for modelName, sch := range s.schemas {
			sb.WriteString(s.generatePrismaModel(modelName, sch))
			sb.WriteString("\n\n")
		}
		
		output = sb.String()

	case "json":
		// Export as JSON
		schemaData := make(map[string]interface{})
		for modelName, sch := range s.schemas {
			schemaData[modelName] = s.schemaToJSON(sch)
		}
		
		data, err := json.MarshalIndent(schemaData, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schemas: %w", err)
		}
		output = string(data)

	default:
		return nil, fmt.Errorf("unsupported format: %s", input.Format)
	}

	return &ToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: output,
			},
		},
	}, nil
}

// Helper functions

func parseFieldType(typeStr string) schema.FieldType {
	// Remove nullable marker
	typeStr = strings.TrimSuffix(typeStr, "?")
	
	switch strings.ToLower(typeStr) {
	case "string":
		return schema.FieldTypeString
	case "int", "integer":
		return schema.FieldTypeInt
	case "bigint":
		return schema.FieldTypeInt64
	case "float", "double":
		return schema.FieldTypeFloat
	case "decimal":
		return schema.FieldTypeDecimal
	case "boolean", "bool":
		return schema.FieldTypeBool
	case "datetime":
		return schema.FieldTypeDateTime
	case "json":
		return schema.FieldTypeJSON
	case "bytes", "binary":
		return schema.FieldTypeBinary
	default:
		return schema.FieldTypeString
	}
}

func parseRelationType(typeStr string) schema.RelationType {
	switch strings.ToLower(typeStr) {
	case "one-to-one", "onetoone":
		return schema.RelationOneToOne
	case "one-to-many", "onetomany":
		return schema.RelationOneToMany
	case "many-to-one", "manytoone":
		return schema.RelationManyToOne
	case "many-to-many", "manytomany":
		return schema.RelationManyToMany
	default:
		return schema.RelationManyToOne
	}
}

func parseDefaultValue(value string) interface{} {
	// Remove quotes if present
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return strings.Trim(value, "\"")
	}
	
	// Check for boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	
	// Check for numbers
	if strings.Contains(value, ".") {
		// Try float
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err == nil {
			return f
		}
	} else {
		// Try int
		var i int64
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	
	// Return as string
	return value
}

func isSystemTable(tableName string) bool {
	// Common system table patterns
	systemPrefixes := []string{
		"sqlite_",
		"pg_",
		"mysql.",
		"information_schema.",
		"performance_schema.",
		"sys.",
		"_prisma_",
		"redi_",
	}
	
	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(strings.ToLower(tableName), prefix) {
			return true
		}
	}
	
	return false
}