package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Prompt templates for ORM and model-focused operations
var prompts = []Prompt{
	{
		Name:        "create_model",
		Description: "Help create a new Prisma model with best practices",
		Arguments: []PromptArgument{
			{
				Name:        "description",
				Description: "Natural language description of the model",
				Required:    true,
			},
			{
				Name:        "related_models",
				Description: "Comma-separated list of related models",
				Required:    false,
			},
		},
	},
	{
		Name:        "optimize_query",
		Description: "Optimize a Prisma query for better performance",
		Arguments: []PromptArgument{
			{
				Name:        "query",
				Description: "Current Prisma query JSON or description",
				Required:    true,
			},
			{
				Name:        "performance_goal",
				Description: "Specific goal (reduce queries, minimize data transfer, etc)",
				Required:    false,
			},
		},
	},
	{
		Name:        "analyze_relations",
		Description: "Analyze and suggest improvements for model relationships",
		Arguments: []PromptArgument{
			{
				Name:        "models",
				Description: "Models to analyze (comma-separated or 'all')",
				Required:    false,
			},
		},
	},
	{
		Name:        "generate_api",
		Description: "Generate API code for a model",
		Arguments: []PromptArgument{
			{
				Name:        "model",
				Description: "Model name",
				Required:    true,
			},
			{
				Name:        "api_type",
				Description: "API type (rest, graphql, both)",
				Required:    false,
			},
			{
				Name:        "operations",
				Description: "Operations to include (create,read,update,delete)",
				Required:    false,
			},
		},
	},
	{
		Name:        "data_migration",
		Description: "Generate data migration strategy",
		Arguments: []PromptArgument{
			{
				Name:        "source_model",
				Description: "Source model or description",
				Required:    true,
			},
			{
				Name:        "target_model",
				Description: "Target model or description",
				Required:    true,
			},
			{
				Name:        "transformation",
				Description: "Data transformation requirements",
				Required:    false,
			},
		},
	},
	// Legacy prompts for backward compatibility
	{
		Name:        "analyze_schema",
		Description: "Analyze database schema and suggest improvements",
		Arguments: []PromptArgument{
			{
				Name:        "focus_area",
				Description: "Specific area to focus on (indexes, relations, performance)",
				Required:    false,
			},
		},
	},
}

// ListPrompts returns available prompts
func (s *Server) ListPrompts() []Prompt {
	return prompts
}

// GetPrompt returns a prompt with filled template
func (s *Server) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*GetPromptResult, error) {
	prompt, err := findPrompt(name)
	if err != nil {
		return nil, err
	}

	// Validate required arguments
	for _, arg := range prompt.Arguments {
		if arg.Required {
			if _, exists := arguments[arg.Name]; !exists {
				return nil, fmt.Errorf("missing required argument: %s", arg.Name)
			}
		}
	}

	// Build prompt message based on template
	switch name {
	case "create_model":
		return s.buildCreateModelPrompt(ctx, arguments)
	case "optimize_query":
		return s.buildOptimizeQueryPrompt(ctx, arguments)
	case "analyze_relations":
		return s.buildAnalyzeRelationsPrompt(ctx, arguments)
	case "generate_api":
		return s.buildGenerateAPIPrompt(ctx, arguments)
	case "data_migration":
		return s.buildDataMigrationPrompt(ctx, arguments)
	case "analyze_schema":
		return s.buildAnalyzeSchemaPrompt(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}
}

// findPrompt finds a prompt by name
func findPrompt(name string) (*Prompt, error) {
	for _, prompt := range prompts {
		if prompt.Name == name {
			return &prompt, nil
		}
	}
	return nil, fmt.Errorf("prompt not found: %s", name)
}

// buildAnalyzeSchemaPrompt builds the analyze schema prompt
func (s *Server) buildAnalyzeSchemaPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	// Get schema information
	schemaResource, err := s.readSchemaResource(ctx, "schema://database")
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	focusArea := arguments["focus_area"]
	if focusArea == "" {
		focusArea = "general"
	}

	// Build prompt
	prompt := fmt.Sprintf(`Analyze the following database schema and suggest improvements.

Focus area: %s

Database Schema:
%s

Please provide:
1. Current issues or inefficiencies in the schema
2. Specific improvement recommendations
3. Best practices that could be applied
4. Performance optimization opportunities`, focusArea, schemaResource.Text)

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

// buildGenerateQueryPrompt builds the generate query prompt
func (s *Server) buildGenerateQueryPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	description := arguments["description"]
	tables := arguments["tables"]

	// Get relevant table schemas
	var tableSchemas []string
	if tables != "" {
		for _, table := range strings.Split(tables, ",") {
			table = strings.TrimSpace(table)
			tableResource, err := s.readTableResource(ctx, fmt.Sprintf("table://%s", table))
			if err != nil {
				s.logger.Warn("Failed to get table schema for %s: %v", table, err)
				continue
			}
			tableSchemas = append(tableSchemas, fmt.Sprintf("Table %s:\n%s", table, tableResource.Text))
		}
	} else {
		// Get all table schemas
		schemaResource, err := s.readSchemaResource(ctx, "schema://database")
		if err == nil {
			tableSchemas = append(tableSchemas, schemaResource.Text)
		}
	}

	// Build prompt
	prompt := fmt.Sprintf(`Generate a SQL query based on the following description:

Description: %s

Available Schema:
%s

Please provide:
1. The SQL query that fulfills the description
2. Explanation of the query logic
3. Any assumptions made
4. Alternative approaches if applicable`, description, strings.Join(tableSchemas, "\n\n"))

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

// buildMigrateSchemaPrompt builds the migrate schema prompt
func (s *Server) buildMigrateSchemaPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	fromSchema := arguments["from_schema"]
	toSchema := arguments["to_schema"]

	// Build prompt
	prompt := fmt.Sprintf(`Generate a database migration script for the following schema changes:

Current Schema:
%s

Desired Schema:
%s

Please provide:
1. SQL migration script (CREATE, ALTER, DROP statements)
2. Rollback script to revert the changes
3. Data migration steps if needed
4. Potential risks and how to mitigate them
5. Order of operations for safe migration`, fromSchema, toSchema)

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

// buildOptimizeQueryPrompt builds the optimize query prompt
func (s *Server) buildOptimizeQueryPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	query := arguments["query"]
	performanceGoal := arguments["performance_goal"]

	// Try to get schema information for context
	schemaResource, err := s.readSchemaResource(ctx, "schema://database")
	schemaContext := ""
	if err == nil {
		schemaContext = fmt.Sprintf("\n\nDatabase Schema:\n%s", schemaResource.Text)
	}

	// Add performance goal if provided
	goalContext := ""
	if performanceGoal != "" {
		goalContext = fmt.Sprintf("\n\nSpecific Performance Goal: %s", performanceGoal)
	}

	// Build prompt
	prompt := fmt.Sprintf(`Analyze and optimize the following Prisma query:

Query:
%s
%s%s

Please provide:
1. Analysis of the current query performance characteristics
2. Optimized version of the query
3. Explanation of optimizations made
4. Index recommendations
5. Alternative query approaches
6. Estimated performance improvements
7. Specific solutions for the performance goal (if provided)`, query, schemaContext, goalContext)

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

// buildDataSummaryPrompt builds the data summary prompt
func (s *Server) buildDataSummaryPrompt(ctx context.Context, arguments map[string]string) (*GetPromptResult, error) {
	tables := arguments["tables"]
	includeSamples := arguments["include_samples"] == "true"

	tableList := strings.Split(tables, ",")
	summaries := make([]string, 0, len(tableList))

	for _, table := range tableList {
		table = strings.TrimSpace(table)
		
		// Get table schema
		tableResource, err := s.readTableResource(ctx, fmt.Sprintf("table://%s", table))
		if err != nil {
			s.logger.Warn("Failed to get table schema for %s: %v", table, err)
			continue
		}

		// Get record count
		countResult, err := s.countRecords(ctx, json.RawMessage(fmt.Sprintf(`{"table": "%s"}`, table)))
		if err != nil {
			s.logger.Warn("Failed to count records for %s: %v", table, err)
			continue
		}

		summary := fmt.Sprintf("Table: %s\nSchema:\n%s\n\nRecord Count:\n%s", 
			table, tableResource.Text, countResult.Content[0].Text)

		// Get sample data if requested
		if includeSamples && s.db != nil {
			dataResource, err := s.readDataResource(ctx, fmt.Sprintf("data://%s?limit=5", table))
			if err == nil {
				summary += fmt.Sprintf("\n\nSample Data:\n%s", dataResource.Text)
			}
		}

		summaries = append(summaries, summary)
	}

	// Build prompt
	prompt := fmt.Sprintf(`Generate a comprehensive summary of the following database tables:

%s

Please provide:
1. Overview of each table's purpose and structure
2. Key relationships between tables
3. Data quality observations
4. Usage patterns and recommendations
5. Potential improvements or concerns`, strings.Join(summaries, "\n\n---\n\n"))

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