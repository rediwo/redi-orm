package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
)

// buildCreateModelPrompt builds the create model prompt
func (s *Server) buildCreateModelPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	description := arguments["description"]
	relatedModels := arguments["related_models"]

	// Get existing schema context
	schemaContext := s.formatSchemasForPrompt(s.schemas)

	prompt := fmt.Sprintf(`Based on the following requirements, create a Prisma model definition:

Requirements: %s

Existing models in the schema:
%s

Related models to consider: %s

Please provide:
1. A complete Prisma model definition with proper field types and attributes
2. Suggested relations with existing models (if applicable)
3. Recommended indexes for common query patterns
4. Any additional fields that would be useful based on best practices
5. Example ORM queries that would work with this model
6. Potential performance considerations

Follow Prisma naming conventions and best practices.`, description, schemaContext, relatedModels)

	messages := []PromptMessage{
		{
			Role: "user",
			Content: PromptContent{
				Type: "text",
				Text: prompt,
			},
		},
	}

	return &GetPromptResult{Messages: messages}, nil
}

// buildAnalyzeRelationsPrompt builds the analyze relations prompt
func (s *Server) buildAnalyzeRelationsPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	models := arguments["models"]
	if models == "" {
		models = "all"
	}

	// Get schema context
	var relevantSchemas map[string]*schema.Schema
	if models == "all" {
		relevantSchemas = s.schemas
	} else {
		relevantSchemas = make(map[string]*schema.Schema)
		for _, modelName := range strings.Split(models, ",") {
			modelName = strings.TrimSpace(modelName)
			if sch, exists := s.schemas[modelName]; exists {
				relevantSchemas[modelName] = sch
			}
		}
	}

	schemaContext := s.formatSchemasForPrompt(relevantSchemas)

	prompt := fmt.Sprintf(`Analyze the relationships in the following models and suggest improvements:

Models to analyze:
%s

Please provide:
1. Current relationship analysis (one-to-many, many-to-many, etc.)
2. Missing relationships that should be added
3. Incorrect or problematic relationships
4. Optimization opportunities for relation queries
5. Recommendations for junction tables (for many-to-many)
6. Best practices for relation naming and structure
7. Potential circular dependency issues

Focus on database normalization, query performance, and data integrity.`, schemaContext)

	messages := []PromptMessage{
		{
			Role: "user",
			Content: PromptContent{
				Type: "text",
				Text: prompt,
			},
		},
	}

	return &GetPromptResult{Messages: messages}, nil
}

// buildGenerateAPIPrompt builds the generate API prompt
func (s *Server) buildGenerateAPIPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	modelName := arguments["model"]
	apiType := arguments["api_type"]
	if apiType == "" {
		apiType = "rest"
	}
	operations := arguments["operations"]
	if operations == "" {
		operations = "create,read,update,delete"
	}

	// Get model schema
	modelSchema, exists := s.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", modelName)
	}

	schemaStr := s.generatePrismaModel(modelName, modelSchema)

	prompt := fmt.Sprintf(`Generate %s API code for the following model:

Model:
%s

Operations to include: %s

Please provide:
1. Complete API endpoint definitions
2. Request/response type definitions
3. Validation logic
4. Error handling
5. Example requests and responses
6. Authentication/authorization considerations
7. Rate limiting recommendations
8. API documentation (OpenAPI/Swagger format if REST)

Use RediORM for database operations and follow best practices for the chosen API type.`, apiType, schemaStr, operations)

	messages := []PromptMessage{
		{
			Role: "user",
			Content: PromptContent{
				Type: "text",
				Text: prompt,
			},
		},
	}

	return &GetPromptResult{Messages: messages}, nil
}

// buildDataMigrationPrompt builds the data migration prompt
func (s *Server) buildDataMigrationPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	sourceModel := arguments["source_model"]
	targetModel := arguments["target_model"]
	transformation := arguments["transformation"]
	if transformation == "" {
		transformation = "direct mapping"
	}

	prompt := fmt.Sprintf(`Generate a data migration strategy:

Source: %s

Target: %s

Transformation requirements: %s

Please provide:
1. Step-by-step migration plan
2. Data transformation logic
3. Validation rules to ensure data integrity
4. Rollback strategy
5. Performance considerations for large datasets
6. Example migration code using RediORM
7. Testing approach
8. Potential data loss risks and mitigation

Consider batch processing, transaction management, and progress tracking.`, sourceModel, targetModel, transformation)

	messages := []PromptMessage{
		{
			Role: "user",
			Content: PromptContent{
				Type: "text",
				Text: prompt,
			},
		},
	}

	return &GetPromptResult{Messages: messages}, nil
}

// formatSchemasForPrompt formats schemas for inclusion in prompts
func (s *Server) formatSchemasForPrompt(schemas map[string]*schema.Schema) string {
	if len(schemas) == 0 {
		return "No existing models"
	}

	var sb strings.Builder
	for modelName, sch := range schemas {
		sb.WriteString(s.generatePrismaModel(modelName, sch))
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}