package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/utils"
)

// Schema modification tool parameters
type SchemaCreateParams struct {
	Model     string                     `json:"model" jsonschema:"Model name"`
	Fields    []SchemaFieldDefinition    `json:"fields" jsonschema:"Field definitions"`
	Relations []SchemaRelationDefinition `json:"relations,omitempty" jsonschema:"Relation definitions"`
	Indexes   []SchemaIndexDefinition    `json:"indexes,omitempty" jsonschema:"Index definitions"`
	TableName string                     `json:"tableName,omitempty" jsonschema:"Custom table name"`
}

type SchemaUpdateParams struct {
	Model           string                     `json:"model" jsonschema:"Model name to update"`
	AddFields       []SchemaFieldDefinition    `json:"addFields,omitempty" jsonschema:"Fields to add"`
	RemoveFields    []string                   `json:"removeFields,omitempty" jsonschema:"Field names to remove"`
	UpdateFields    []SchemaFieldUpdate        `json:"updateFields,omitempty" jsonschema:"Fields to update"`
	AddRelations    []SchemaRelationDefinition `json:"addRelations,omitempty" jsonschema:"Relations to add"`
	RemoveRelations []string                   `json:"removeRelations,omitempty" jsonschema:"Relation names to remove"`
}

type SchemaAddFieldParams struct {
	Model string                `json:"model" jsonschema:"Model name"`
	Field SchemaFieldDefinition `json:"field" jsonschema:"Field definition"`
}

type SchemaRemoveFieldParams struct {
	Model     string `json:"model" jsonschema:"Model name"`
	FieldName string `json:"fieldName" jsonschema:"Field name to remove"`
}

type SchemaAddRelationParams struct {
	Model    string                   `json:"model" jsonschema:"Model name"`
	Relation SchemaRelationDefinition `json:"relation" jsonschema:"Relation definition"`
}

// Helper types for field and relation definitions
type SchemaFieldDefinition struct {
	Name          string `json:"name" jsonschema:"Field name"`
	Type          string `json:"type" jsonschema:"Field type (String, Int, Boolean, DateTime, etc.)"`
	Optional      bool   `json:"optional,omitempty" jsonschema:"Is field optional/nullable"`
	List          bool   `json:"list,omitempty" jsonschema:"Is field an array"`
	Unique        bool   `json:"unique,omitempty" jsonschema:"Is field unique"`
	PrimaryKey    bool   `json:"primaryKey,omitempty" jsonschema:"Is primary key"`
	AutoIncrement bool   `json:"autoIncrement,omitempty" jsonschema:"Auto-increment integer"`
	Default       any    `json:"default,omitempty" jsonschema:"Default value"`
	DbType        string `json:"dbType,omitempty" jsonschema:"Database-specific type"`
}

type SchemaFieldUpdate struct {
	Name    string         `json:"name" jsonschema:"Field name to update"`
	Changes map[string]any `json:"changes" jsonschema:"Changes to apply"`
}

type SchemaRelationDefinition struct {
	Name       string `json:"name" jsonschema:"Relation field name"`
	Type       string `json:"type" jsonschema:"Relation type (oneToOne, oneToMany, manyToOne, manyToMany)"`
	Model      string `json:"model" jsonschema:"Related model name"`
	ForeignKey string `json:"foreignKey,omitempty" jsonschema:"Foreign key field"`
	References string `json:"references,omitempty" jsonschema:"Referenced field"`
}

type SchemaIndexDefinition struct {
	Fields []string `json:"fields" jsonschema:"Fields in the index"`
	Unique bool     `json:"unique,omitempty" jsonschema:"Is unique index"`
	Name   string   `json:"name,omitempty" jsonschema:"Index name"`
}

// registerSchemaModificationTools registers all schema modification tools
func (s *SDKServer) registerSchemaModificationTools() {
	// Initialize pending schema manager if not already done
	if s.pendingSchemaManager == nil {
		s.pendingSchemaManager = NewPendingSchemaManager(s.logger)
	}
	// Schema modification operations
	schemaCreateSchema, _ := jsonschema.For[SchemaCreateParams]()
	addToolWithLogging[SchemaCreateParams, any](s, &mcp.Tool{
		Name:        "schema.create",
		Description: "Create a new model schema",
		InputSchema: schemaCreateSchema,
	}, s.handleSchemaCreate)

	schemaUpdateSchema, _ := jsonschema.For[SchemaUpdateParams]()
	addToolWithLogging[SchemaUpdateParams, any](s, &mcp.Tool{
		Name:        "schema.update",
		Description: "Update an existing model schema",
		InputSchema: schemaUpdateSchema,
	}, s.handleSchemaUpdate)

	addFieldSchema, _ := jsonschema.For[SchemaAddFieldParams]()
	addToolWithLogging[SchemaAddFieldParams, any](s, &mcp.Tool{
		Name:        "schema.addField",
		Description: "Add a field to an existing model",
		InputSchema: addFieldSchema,
	}, s.handleSchemaAddField)

	removeFieldSchema, _ := jsonschema.For[SchemaRemoveFieldParams]()
	addToolWithLogging[SchemaRemoveFieldParams, any](s, &mcp.Tool{
		Name:        "schema.removeField",
		Description: "Remove a field from a model",
		InputSchema: removeFieldSchema,
	}, s.handleSchemaRemoveField)

	addRelationSchema, _ := jsonschema.For[SchemaAddRelationParams]()
	addToolWithLogging[SchemaAddRelationParams, any](s, &mcp.Tool{
		Name:        "schema.addRelation",
		Description: "Add a relation between models",
		InputSchema: addRelationSchema,
	}, s.handleSchemaAddRelation)
}

// Schema modification handlers
func (s *SDKServer) handleSchemaCreate(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SchemaCreateParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("schema.create"); err != nil {
		return nil, err
	}

	// Check if model already exists
	for _, existing := range s.schemas {
		if existing.Name == params.Arguments.Model {
			return nil, fmt.Errorf("model %s already exists", params.Arguments.Model)
		}
	}

	// Create new schema
	newSchema := schema.New(params.Arguments.Model)

	// Set custom table name if provided
	if params.Arguments.TableName != "" {
		newSchema.WithTableName(params.Arguments.TableName)
	}

	// Add fields
	for _, fieldDef := range params.Arguments.Fields {
		field, err := s.createSchemaField(fieldDef)
		if err != nil {
			return nil, fmt.Errorf("failed to create field %s: %w", fieldDef.Name, err)
		}
		newSchema.AddField(field)
	}

	// Add relations
	for _, relDef := range params.Arguments.Relations {
		relation := s.createSchemaRelation(relDef)
		newSchema.AddRelation(relDef.Name, relation)
	}

	// Add indexes
	for _, indexDef := range params.Arguments.Indexes {
		index := schema.Index{
			Fields: indexDef.Fields,
			Unique: indexDef.Unique,
			Name:   indexDef.Name,
		}
		if index.Name == "" {
			// Use the utility function to generate index name
			index.Name = utils.GenerateIndexName(params.Arguments.Model, indexDef.Fields, index.Unique, "")
		}
		newSchema.AddIndex(index)
	}

	// Save to file
	if err := s.persistence.SaveSchema(newSchema); err != nil {
		return nil, fmt.Errorf("failed to save schema: %w", err)
	}

	// Add to in-memory schemas
	s.schemas = append(s.schemas, newSchema)

	// Register with database if connected
	if s.db != nil {
		if err := s.db.RegisterSchema(newSchema.Name, newSchema); err != nil {
			s.logger.Warn("Failed to register schema with database: %v", err)
		}
	}

	// Add to pending schema manager and attempt to create tables
	s.pendingSchemaManager.AddSchema(newSchema)

	var tableResult *TableCreationResult
	var tableErr error

	if s.db != nil {
		tableResult, tableErr = s.pendingSchemaManager.ProcessPendingSchemas(ctx, s.db)
		if tableErr != nil {
			s.logger.Error("Failed to process pending schemas: %v", tableErr)
		}
	}

	// Build enhanced result
	result := map[string]any{
		"success": true,
		"model":   newSchema.Name,
		"message": fmt.Sprintf("Created model %s with %d fields", newSchema.Name, len(newSchema.Fields)),
	}

	// Add table creation results if available
	if tableResult != nil {
		result["tables_created"] = tableResult.TablesCreated
		result["pending_schemas"] = tableResult.PendingSchemas
		result["dependency_info"] = tableResult.DependencyInfo
		result["has_circular_dependencies"] = tableResult.CircularDeps

		if len(tableResult.Errors) > 0 {
			result["table_creation_errors"] = tableResult.Errors
		}

		// Update message with table creation info
		if len(tableResult.TablesCreated) > 0 {
			result["message"] = fmt.Sprintf("Created model %s with %d fields and %d database tables",
				newSchema.Name, len(newSchema.Fields), len(tableResult.TablesCreated))
		} else if len(tableResult.PendingSchemas) > 0 {
			result["message"] = fmt.Sprintf("Created model %s with %d fields. %d schemas waiting for dependencies",
				newSchema.Name, len(newSchema.Fields), len(tableResult.PendingSchemas))
		}
	} else if tableErr != nil {
		result["table_creation_error"] = tableErr.Error()
		result["message"] = fmt.Sprintf("Created model %s with %d fields, but failed to create database tables",
			newSchema.Name, len(newSchema.Fields))
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleSchemaUpdate(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SchemaUpdateParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("schema.update"); err != nil {
		return nil, err
	}

	// Find existing schema
	var targetSchema *schema.Schema
	for _, s := range s.schemas {
		if s.Name == params.Arguments.Model {
			targetSchema = s
			break
		}
	}

	if targetSchema == nil {
		return nil, fmt.Errorf("model %s not found", params.Arguments.Model)
	}

	// Apply updates
	changes := []string{}

	// Add fields
	for _, fieldDef := range params.Arguments.AddFields {
		field, err := s.createSchemaField(fieldDef)
		if err != nil {
			return nil, fmt.Errorf("failed to create field %s: %w", fieldDef.Name, err)
		}
		targetSchema.AddField(field)
		changes = append(changes, fmt.Sprintf("added field %s", fieldDef.Name))
	}

	// Remove fields
	for _, fieldName := range params.Arguments.RemoveFields {
		newFields := []schema.Field{}
		for _, f := range targetSchema.Fields {
			if f.Name != fieldName {
				newFields = append(newFields, f)
			}
		}
		targetSchema.Fields = newFields
		changes = append(changes, fmt.Sprintf("removed field %s", fieldName))
	}

	// Update fields
	for _, fieldUpdate := range params.Arguments.UpdateFields {
		for i, f := range targetSchema.Fields {
			if f.Name == fieldUpdate.Name {
				// Apply changes
				if val, ok := fieldUpdate.Changes["type"]; ok {
					if typeStr, ok := val.(string); ok {
						fieldType, err := s.parseFieldType(typeStr)
						if err != nil {
							return nil, err
						}
						targetSchema.Fields[i].Type = fieldType
					}
				}
				if val, ok := fieldUpdate.Changes["optional"]; ok {
					if b, ok := val.(bool); ok {
						targetSchema.Fields[i].Nullable = b
					}
				}
				if val, ok := fieldUpdate.Changes["unique"]; ok {
					if b, ok := val.(bool); ok {
						targetSchema.Fields[i].Unique = b
					}
				}
				if val, ok := fieldUpdate.Changes["default"]; ok {
					targetSchema.Fields[i].Default = s.normalizeDefaultValue(val, targetSchema.Fields[i].Type)
				}
				changes = append(changes, fmt.Sprintf("updated field %s", fieldUpdate.Name))
				break
			}
		}
	}

	// Add relations
	for _, relDef := range params.Arguments.AddRelations {
		relation := s.createSchemaRelation(relDef)
		targetSchema.AddRelation(relDef.Name, relation)
		changes = append(changes, fmt.Sprintf("added relation %s", relDef.Name))
	}

	// Remove relations
	for _, relName := range params.Arguments.RemoveRelations {
		delete(targetSchema.Relations, relName)
		changes = append(changes, fmt.Sprintf("removed relation %s", relName))
	}

	// Save to file
	if err := s.persistence.SaveSchema(targetSchema); err != nil {
		return nil, fmt.Errorf("failed to save schema: %w", err)
	}

	result := map[string]any{
		"success": true,
		"model":   targetSchema.Name,
		"changes": changes,
		"message": fmt.Sprintf("Updated model %s with %d changes", targetSchema.Name, len(changes)),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleSchemaAddField(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SchemaAddFieldParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("schema.addField"); err != nil {
		return nil, err
	}

	// Find existing schema
	var targetSchema *schema.Schema
	for _, s := range s.schemas {
		if s.Name == params.Arguments.Model {
			targetSchema = s
			break
		}
	}

	if targetSchema == nil {
		return nil, fmt.Errorf("model %s not found", params.Arguments.Model)
	}

	// Create and add field
	field, err := s.createSchemaField(params.Arguments.Field)
	if err != nil {
		return nil, fmt.Errorf("failed to create field: %w", err)
	}

	targetSchema.AddField(field)

	// Save to file
	if err := s.persistence.SaveSchema(targetSchema); err != nil {
		return nil, fmt.Errorf("failed to save schema: %w", err)
	}

	result := map[string]any{
		"success": true,
		"model":   targetSchema.Name,
		"field":   field.Name,
		"message": fmt.Sprintf("Added field %s to model %s", field.Name, targetSchema.Name),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleSchemaRemoveField(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SchemaRemoveFieldParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("schema.removeField"); err != nil {
		return nil, err
	}

	// Find existing schema
	var targetSchema *schema.Schema
	for _, s := range s.schemas {
		if s.Name == params.Arguments.Model {
			targetSchema = s
			break
		}
	}

	if targetSchema == nil {
		return nil, fmt.Errorf("model %s not found", params.Arguments.Model)
	}

	// Remove field
	newFields := []schema.Field{}
	found := false
	for _, f := range targetSchema.Fields {
		if f.Name != params.Arguments.FieldName {
			newFields = append(newFields, f)
		} else {
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("field %s not found in model %s", params.Arguments.FieldName, params.Arguments.Model)
	}

	targetSchema.Fields = newFields

	// Save to file
	if err := s.persistence.SaveSchema(targetSchema); err != nil {
		return nil, fmt.Errorf("failed to save schema: %w", err)
	}

	result := map[string]any{
		"success": true,
		"model":   targetSchema.Name,
		"field":   params.Arguments.FieldName,
		"message": fmt.Sprintf("Removed field %s from model %s", params.Arguments.FieldName, targetSchema.Name),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleSchemaAddRelation(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SchemaAddRelationParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("schema.addRelation"); err != nil {
		return nil, err
	}

	// Find existing schema
	var targetSchema *schema.Schema
	for _, s := range s.schemas {
		if s.Name == params.Arguments.Model {
			targetSchema = s
			break
		}
	}

	if targetSchema == nil {
		return nil, fmt.Errorf("model %s not found", params.Arguments.Model)
	}

	// Create and add relation
	relation := s.createSchemaRelation(params.Arguments.Relation)
	targetSchema.AddRelation(params.Arguments.Relation.Name, relation)

	// Save to file
	if err := s.persistence.SaveSchema(targetSchema); err != nil {
		return nil, fmt.Errorf("failed to save schema: %w", err)
	}

	result := map[string]any{
		"success":  true,
		"model":    targetSchema.Name,
		"relation": params.Arguments.Relation.Name,
		"message":  fmt.Sprintf("Added relation %s to model %s", params.Arguments.Relation.Name, targetSchema.Name),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

// Helper methods
func (s *SDKServer) createSchemaField(def SchemaFieldDefinition) (schema.Field, error) {
	fieldType, err := s.parseFieldType(def.Type)
	if err != nil {
		return schema.Field{}, err
	}

	field := schema.Field{
		Name:          def.Name,
		Type:          fieldType,
		Nullable:      def.Optional,
		Unique:        def.Unique,
		PrimaryKey:    def.PrimaryKey,
		AutoIncrement: def.AutoIncrement,
		Default:       s.normalizeDefaultValue(def.Default, fieldType),
		DbType:        def.DbType,
	}

	return field, nil
}

func (s *SDKServer) createSchemaRelation(def SchemaRelationDefinition) schema.Relation {
	relType := schema.RelationType(def.Type)

	return schema.Relation{
		Type:       relType,
		Model:      def.Model,
		ForeignKey: def.ForeignKey,
		References: def.References,
	}
}

// normalizeDefaultValue converts string function calls back to their proper types based on field type
func (s *SDKServer) normalizeDefaultValue(value any, fieldType schema.FieldType) any {
	if value == nil {
		return nil
	}

	// If it's not a string, return as-is (already proper type)
	str, ok := value.(string)
	if !ok {
		return value
	}

	// Check for function calls first (these work for any field type)
	functionCalls := []string{
		"cuid()",
		"uuid()",
		"now()",
		"autoincrement()",
		"nanoid()",
		"dbgenerated()",
	}

	for _, fn := range functionCalls {
		if str == fn {
			return fn // Keep as function call
		}
	}

	// Check for function patterns (ends with parentheses and no spaces)
	if strings.HasSuffix(str, "()") && !strings.Contains(str, " ") {
		return str // Preserve function calls
	}

	// Check for SQL constants (work for DateTime fields mainly)
	sqlConstants := []string{
		"CURRENT_TIMESTAMP",
		"CURRENT_DATE",
		"CURRENT_TIME",
	}

	for _, constant := range sqlConstants {
		if strings.ToUpper(str) == constant {
			return str
		}
	}

	// Handle special values
	switch strings.ToLower(str) {
	case "null", "nil":
		return nil
	}

	// Now handle based on field type
	switch fieldType {
	case schema.FieldTypeBool:
		switch strings.ToLower(str) {
		case "true":
			return true
		case "false":
			return false
		default:
			return str // Let schema generator handle invalid values
		}

	case schema.FieldTypeInt, schema.FieldTypeInt64:
		// Try to parse as integer
		if isNumericString(str) {
			return str // Keep as string, let schema generator convert
		}
		return str

	case schema.FieldTypeFloat, schema.FieldTypeDecimal:
		// Try to parse as float
		if isNumericString(str) {
			return str // Keep as string, let schema generator convert
		}
		return str

	case schema.FieldTypeString:
		// For string fields, most values should remain as strings
		// except for function calls which we handled above
		return str

	case schema.FieldTypeDateTime:
		// DateTime fields can have now(), CURRENT_TIMESTAMP, or string dates
		return str

	case schema.FieldTypeJSON:
		// JSON fields can have complex default values
		return str

	default:
		// For other types, return as string
		return str
	}
}

// isNumericString checks if a string represents a valid number
func isNumericString(s string) bool {
	if s == "" {
		return false
	}

	// Simple check for numeric patterns
	validChars := "0123456789.-+"
	dotCount := 0
	signCount := 0

	for i, char := range s {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
		if char == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
		}
		if char == '-' || char == '+' {
			signCount++
			if signCount > 1 || i != 0 { // Sign must be at beginning and only one
				return false
			}
		}
	}

	return true
}

func (s *SDKServer) parseFieldType(typeStr string) (schema.FieldType, error) {
	// Handle array types
	if strings.HasSuffix(typeStr, "[]") {
		baseType := strings.TrimSuffix(typeStr, "[]")
		switch baseType {
		case "String":
			return schema.FieldTypeStringArray, nil
		case "Int":
			return schema.FieldTypeIntArray, nil
		case "BigInt":
			return schema.FieldTypeInt64Array, nil
		case "Float":
			return schema.FieldTypeFloatArray, nil
		case "Boolean":
			return schema.FieldTypeBoolArray, nil
		case "Decimal":
			return schema.FieldTypeDecimalArray, nil
		case "DateTime":
			return schema.FieldTypeDateTimeArray, nil
		default:
			return "", fmt.Errorf("unknown array type: %s", typeStr)
		}
	}

	// Handle scalar types
	switch typeStr {
	case "String":
		return schema.FieldTypeString, nil
	case "Int":
		return schema.FieldTypeInt, nil
	case "BigInt":
		return schema.FieldTypeInt64, nil
	case "Float":
		return schema.FieldTypeFloat, nil
	case "Boolean":
		return schema.FieldTypeBool, nil
	case "DateTime":
		return schema.FieldTypeDateTime, nil
	case "Json":
		return schema.FieldTypeJSON, nil
	case "Decimal":
		return schema.FieldTypeDecimal, nil
	case "Bytes":
		return schema.FieldTypeBinary, nil
	default:
		return "", fmt.Errorf("unknown field type: %s", typeStr)
	}
}
