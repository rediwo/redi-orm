package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/utils"
)

// Tool definitions
var tools = []Tool{
	// ORM Data Operation Tools
	{
		Name:        "data.findMany",
		Description: "Query multiple records with Prisma-style filters",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "description": "Model name"},
				"where": {"type": "object", "description": "Filter conditions"},
				"include": {"type": "object", "description": "Relations to include"},
				"orderBy": {"type": "object", "description": "Sort order"},
				"take": {"type": "integer", "description": "Limit results"},
				"skip": {"type": "integer", "description": "Skip results"}
			},
			"required": ["model"]
		}`),
	},
	{
		Name:        "data.findUnique",
		Description: "Find a single record by unique field",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "description": "Model name"},
				"where": {"type": "object", "description": "Unique identifier"},
				"include": {"type": "object", "description": "Relations to include"}
			},
			"required": ["model", "where"]
		}`),
	},
	{
		Name:        "data.create",
		Description: "Create a new record using ORM",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "description": "Model name"},
				"data": {"type": "object", "description": "Data to create"}
			},
			"required": ["model", "data"]
		}`),
	},
	{
		Name:        "data.update",
		Description: "Update records using ORM",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "description": "Model name"},
				"where": {"type": "object", "description": "Filter to find records"},
				"data": {"type": "object", "description": "Data to update"}
			},
			"required": ["model", "where", "data"]
		}`),
	},
	{
		Name:        "data.delete",
		Description: "Delete records using ORM",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "description": "Model name"},
				"where": {"type": "object", "description": "Filter to find records"}
			},
			"required": ["model", "where"]
		}`),
	},
	{
		Name:        "data.count",
		Description: "Count records with optional filters",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "description": "Model name"},
				"where": {"type": "object", "description": "Filter conditions"}
			},
			"required": ["model"]
		}`),
	},
	{
		Name:        "data.aggregate",
		Description: "Perform aggregation queries",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"model": {"type": "string", "description": "Model name"},
				"where": {"type": "object", "description": "Filter conditions"},
				"count": {"type": "boolean"},
				"avg": {"type": "object"},
				"sum": {"type": "object"},
				"min": {"type": "object"},
				"max": {"type": "object"},
				"groupBy": {"type": "array", "items": {"type": "string"}}
			},
			"required": ["model"]
		}`),
	},
	// Legacy SQL Tools
	{
		Name:        "query",
		Description: "Execute a read-only SQL query",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"sql": {
					"type": "string",
					"description": "SQL query to execute (SELECT only)"
				},
				"parameters": {
					"type": "array",
					"description": "Query parameters",
					"items": {}
				}
			},
			"required": ["sql"]
		}`),
	},
	{
		Name:        "inspect_schema",
		Description: "Get detailed schema information for a table",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"table": {
					"type": "string",
					"description": "Table name to inspect"
				}
			},
			"required": ["table"]
		}`),
	},
	{
		Name:        "list_tables",
		Description: "List all available tables in the database",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	},
	{
		Name:        "count_records",
		Description: "Count records in a table with optional filters",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"table": {
					"type": "string",
					"description": "Table name"
				},
				"where": {
					"type": "object",
					"description": "Filter conditions"
				}
			},
			"required": ["table"]
		}`),
	},
	{
		Name:        "batch_query",
		Description: "Execute multiple read-only SQL queries in a batch",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"queries": {
					"type": "array",
					"description": "Array of SQL queries to execute",
					"items": {
						"type": "object",
						"properties": {
							"sql": {
								"type": "string",
								"description": "SQL query to execute"
							},
							"parameters": {
								"type": "array",
								"description": "Query parameters",
								"items": {}
							},
							"label": {
								"type": "string",
								"description": "Optional label for this query"
							}
						},
						"required": ["sql"]
					}
				},
				"fail_fast": {
					"type": "boolean",
					"description": "Stop execution on first error (default: false)",
					"default": false
				}
			},
			"required": ["queries"]
		}`),
	},
	{
		Name:        "stream_query",
		Description: "Execute a query with streaming results for large datasets",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"sql": {
					"type": "string",
					"description": "SQL query to execute (SELECT only)"
				},
				"parameters": {
					"type": "array",
					"description": "Query parameters",
					"items": {}
				},
				"batch_size": {
					"type": "integer",
					"description": "Number of rows per batch (default: 100)",
					"default": 100,
					"minimum": 1,
					"maximum": 1000
				}
			},
			"required": ["sql"]
		}`),
	},
	{
		Name:        "analyze_table",
		Description: "Perform statistical analysis on a table",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"table": {
					"type": "string",
					"description": "Table name to analyze"
				},
				"columns": {
					"type": "array",
					"description": "Specific columns to analyze (default: all)",
					"items": {
						"type": "string"
					}
				},
				"sample_size": {
					"type": "integer",
					"description": "Number of sample rows to analyze (default: 1000)",
					"default": 1000,
					"minimum": 100,
					"maximum": 10000
				}
			},
			"required": ["table"]
		}`),
	},
	{
		Name:        "generate_sample",
		Description: "Generate sample data from a table",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"table": {
					"type": "string",
					"description": "Table name to sample from"
				},
				"count": {
					"type": "integer",
					"description": "Number of sample rows (default: 10)",
					"default": 10,
					"minimum": 1,
					"maximum": 100
				},
				"random": {
					"type": "boolean",
					"description": "Use random sampling (default: false)",
					"default": false
				},
				"where": {
					"type": "object",
					"description": "Filter conditions for sampling"
				}
			},
			"required": ["table"]
		}`),
	},
}

// ListTools returns available tools
func (s *Server) ListTools() []Tool {
	result := make([]Tool, len(tools))
	copy(result, tools)
	
	// Add model and schema management tools if schemas are loaded
	if len(s.schemas) > 0 {
		// These tools are only available when schemas are loaded
		result = append(result, Tool{
			Name:        "model.create",
			Description: "Create a new Prisma model definition",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "Model name"},
					"fields": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"type": {"type": "string"},
								"attributes": {"type": "array", "items": {"type": "string"}},
								"default": {},
								"relation": {"type": "object"}
							},
							"required": ["name", "type"]
						}
					}
				},
				"required": ["name", "fields"]
			}`),
		})
		
		result = append(result, Tool{
			Name:        "schema.sync",
			Description: "Synchronize schema changes to database",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"force": {"type": "boolean", "description": "Force destructive changes"}
				}
			}`),
		})
	}
	
	return result
}

// CallTool executes a tool
func (s *Server) CallTool(ctx context.Context, name string, arguments json.RawMessage) (*ToolResult, error) {
	s.logger.Debug("Calling tool: %s", name)

	// Handle ORM data tools
	if strings.HasPrefix(name, "data.") {
		return s.callDataTool(ctx, name, arguments)
	}
	
	// Handle model management tools
	if strings.HasPrefix(name, "model.") {
		return s.callModelTool(ctx, name, arguments)
	}
	
	// Handle schema management tools
	if strings.HasPrefix(name, "schema.") {
		return s.callSchemaTool(ctx, name, arguments)
	}

	// Handle legacy SQL tools
	switch name {
	case "query":
		return s.executeQuery(ctx, arguments)
	case "inspect_schema":
		return s.inspectSchema(ctx, arguments)
	case "list_tables":
		return s.listTables(ctx, arguments)
	case "count_records":
		return s.countRecords(ctx, arguments)
	case "batch_query":
		return s.executeBatchQuery(ctx, arguments)
	case "stream_query":
		return s.executeStreamQuery(ctx, arguments)
	case "analyze_table":
		return s.analyzeTable(ctx, arguments)
	case "generate_sample":
		return s.generateSample(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// executeQuery handles SQL query execution
func (s *Server) executeQuery(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var args struct {
		SQL        string        `json:"sql"`
		Parameters []interface{} `json:"parameters"`
	}

	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate query through security manager
	if err := s.security.ValidateQuery(args.SQL); err != nil {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Security error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Execute query using Raw API
	var results []map[string]interface{}
	rawQuery := s.db.Raw(args.SQL, args.Parameters...)
	if err := rawQuery.Find(ctx, &results); err != nil {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Query error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Check row limit
	if len(results) > s.config.MaxQueryRows {
		results = results[:s.config.MaxQueryRows]
		s.logger.Warn("Query results truncated to limit: %d", s.config.MaxQueryRows)
	}

	// Format results
	response := map[string]interface{}{
		"query":   args.SQL,
		"results": results,
		"count":   len(results),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
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

// inspectSchema handles table schema inspection
func (s *Server) inspectSchema(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var args struct {
		Table string `json:"table"`
	}

	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate table access through security manager
	if err := s.security.ValidateTableAccess(args.Table); err != nil {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Security error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	
	// Legacy check for backward compatibility
	if !s.isTableAllowed(args.Table) {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Error: Table '%s' is not allowed", args.Table),
				},
			},
			IsError: true,
		}, nil
	}

	// Get table schema
	tableInfo, err := s.getTableSchema(ctx, args.Table)
	if err != nil {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Error inspecting table: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Format results
	data, err := json.MarshalIndent(tableInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
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

// listTables handles table listing
func (s *Server) listTables(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get all tables using migrator
	migrator := s.db.GetMigrator()
	tables, err := migrator.GetTables()
	if err != nil {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Error listing tables: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Filter allowed tables
	allowedTables := make([]string, 0, len(tables))
	for _, table := range tables {
		if s.isTableAllowed(table) {
			allowedTables = append(allowedTables, table)
		}
	}

	// Format results
	response := map[string]interface{}{
		"database_type": s.db.GetDriverType(),
		"tables":        allowedTables,
		"count":         len(allowedTables),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tables: %w", err)
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

// countRecords handles record counting
func (s *Server) countRecords(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var args struct {
		Table string                 `json:"table"`
		Where map[string]interface{} `json:"where"`
	}

	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate table access through security manager
	if err := s.security.ValidateTableAccess(args.Table); err != nil {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Security error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}
	
	// Legacy check for backward compatibility
	if !s.isTableAllowed(args.Table) {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Error: Table '%s' is not allowed", args.Table),
				},
			},
			IsError: true,
		}, nil
	}

	// Build count query
	query := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", args.Table)
	var parameters []interface{}

	// Add WHERE conditions if provided
	if len(args.Where) > 0 {
		conditions := make([]string, 0, len(args.Where))
		for field, value := range args.Where {
			conditions = append(conditions, fmt.Sprintf("%s = ?", field))
			parameters = append(parameters, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Execute query using Raw API
	var results []map[string]interface{}
	rawQuery := s.db.Raw(query, parameters...)
	if err := rawQuery.Find(ctx, &results); err != nil {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Error counting records: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Extract count
	count := int64(0)
	if len(results) > 0 {
		if c, exists := results[0]["count"]; exists {
			count = utils.ToInt64(c)
		}
	}

	// Format results
	response := map[string]interface{}{
		"table":      args.Table,
		"count":      count,
		"conditions": args.Where,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal count: %w", err)
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

// isReadOnlyQuery checks if a SQL query is read-only
func isReadOnlyQuery(sql string) bool {
	// Trim and convert to uppercase for checking
	trimmed := strings.TrimSpace(strings.ToUpper(sql))
	
	// Check if it starts with SELECT
	if !strings.HasPrefix(trimmed, "SELECT") {
		return false
	}
	
	// Check for dangerous keywords
	dangerousKeywords := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER",
		"TRUNCATE", "GRANT", "REVOKE", "EXEC", "EXECUTE",
		"INTO OUTFILE", "INTO DUMPFILE",
	}
	
	for _, keyword := range dangerousKeywords {
		if strings.Contains(trimmed, keyword) {
			return false
		}
	}
	
	return true
}

// executeBatchQuery handles batch SQL query execution
func (s *Server) executeBatchQuery(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var args struct {
		Queries  []BatchQuery `json:"queries"`
		FailFast bool         `json:"fail_fast"`
	}

	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if len(args.Queries) == 0 {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: "Error: No queries provided",
			}},
			IsError: true,
		}, nil
	}

	results := make([]BatchQueryResult, 0, len(args.Queries))

	for i, query := range args.Queries {
		result := BatchQueryResult{
			Index: i,
			Label: query.Label,
			SQL:   query.SQL,
		}

		// Validate query through security manager
		if err := s.security.ValidateQuery(query.SQL); err != nil {
			result.Error = fmt.Sprintf("Security error: %v", err)
			results = append(results, result)
			if args.FailFast {
				break
			}
			continue
		}

		// Execute query
		var queryResults []map[string]interface{}
		rawQuery := s.db.Raw(query.SQL, query.Parameters...)
		if err := rawQuery.Find(ctx, &queryResults); err != nil {
			result.Error = fmt.Sprintf("Query error: %v", err)
			results = append(results, result)
			if args.FailFast {
				break
			}
			continue
		}

		// Check row limit
		if len(queryResults) > s.config.MaxQueryRows {
			queryResults = queryResults[:s.config.MaxQueryRows]
			result.Truncated = true
		}

		result.Results = queryResults
		result.Count = len(queryResults)
		result.Success = true
		results = append(results, result)
	}

	// Format response
	response := map[string]interface{}{
		"batch_size":     len(args.Queries),
		"executed":       len(results),
		"results":        results,
		"fail_fast_mode": args.FailFast,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

// executeStreamQuery handles streaming query execution (simulated with batches)
func (s *Server) executeStreamQuery(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var args struct {
		SQL        string        `json:"sql"`
		Parameters []interface{} `json:"parameters"`
		BatchSize  int           `json:"batch_size"`
	}

	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.BatchSize <= 0 {
		args.BatchSize = 100
	}

	// Validate query through security manager
	if err := s.security.ValidateQuery(args.SQL); err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Security error: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// For streaming simulation, we'll execute the full query and divide into batches
	var allResults []map[string]interface{}
	rawQuery := s.db.Raw(args.SQL, args.Parameters...)
	if err := rawQuery.Find(ctx, &allResults); err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Query error: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Divide results into batches
	var batches [][]map[string]interface{}
	for i := 0; i < len(allResults); i += args.BatchSize {
		end := i + args.BatchSize
		if end > len(allResults) {
			end = len(allResults)
		}
		batches = append(batches, allResults[i:end])
	}

	// Format streaming response
	response := map[string]interface{}{
		"query":       args.SQL,
		"total_rows":  len(allResults),
		"batch_size":  args.BatchSize,
		"batch_count": len(batches),
		"batches":     batches,
		"streaming":   true,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

// analyzeTable performs statistical analysis on a table
func (s *Server) analyzeTable(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var args struct {
		Table      string   `json:"table"`
		Columns    []string `json:"columns"`
		SampleSize int      `json:"sample_size"`
	}

	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.SampleSize <= 0 {
		args.SampleSize = 1000
	}

	// Validate table access through security manager
	if err := s.security.ValidateTableAccess(args.Table); err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Security error: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Get table schema first
	tableInfo, err := s.getTableSchema(ctx, args.Table)
	if err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Error getting table schema: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Get total record count
	countQuery := fmt.Sprintf("SELECT COUNT(*) as total FROM %s", args.Table)
	var countResults []map[string]interface{}
	if err := s.db.Raw(countQuery).Find(ctx, &countResults); err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Error counting records: %v", err),
			}},
			IsError: true,
		}, nil
	}

	totalRows := int64(0)
	if len(countResults) > 0 {
		if count, exists := countResults[0]["total"]; exists {
			totalRows = utils.ToInt64(count)
		}
	}

	// Get sample data for analysis
	sampleQuery := fmt.Sprintf("SELECT * FROM %s LIMIT %d", args.Table, args.SampleSize)
	var sampleResults []map[string]interface{}
	if err := s.db.Raw(sampleQuery).Find(ctx, &sampleResults); err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Error sampling data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Perform basic statistical analysis
	analysis := TableAnalysis{
		Table:       args.Table,
		TotalRows:   totalRows,
		SampleSize:  len(sampleResults),
		Schema:      tableInfo,
		Statistics:  make(map[string]ColumnStats),
	}

	// Analyze each column
	if len(sampleResults) > 0 {
		for column := range sampleResults[0] {
			// Skip if specific columns requested and this isn't one
			if len(args.Columns) > 0 {
				found := false
				for _, col := range args.Columns {
					if col == column {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			stats := s.analyzeColumn(column, sampleResults)
			analysis.Statistics[column] = stats
		}
	}

	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal analysis: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

// generateSample generates sample data from a table
func (s *Server) generateSample(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var args struct {
		Table  string                 `json:"table"`
		Count  int                    `json:"count"`
		Random bool                   `json:"random"`
		Where  map[string]interface{} `json:"where"`
	}

	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Count <= 0 {
		args.Count = 10
	}

	// Validate table access through security manager
	if err := s.security.ValidateTableAccess(args.Table); err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Security error: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Build sample query
	query := fmt.Sprintf("SELECT * FROM %s", args.Table)
	var parameters []interface{}

	// Add WHERE conditions if provided
	if len(args.Where) > 0 {
		conditions := make([]string, 0, len(args.Where))
		for field, value := range args.Where {
			conditions = append(conditions, fmt.Sprintf("%s = ?", field))
			parameters = append(parameters, value)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add random sampling or limit
	if args.Random {
		// Note: Random sampling varies by database
		switch s.db.GetDriverType() {
		case "mysql":
			query += " ORDER BY RAND()"
		case "postgresql":
			query += " ORDER BY RANDOM()"
		case "sqlite":
			query += " ORDER BY RANDOM()"
		case "mongodb":
			// For MongoDB, we'd use aggregation pipeline with $sample
			// but we'll fall back to limit for now
		}
	}

	query += fmt.Sprintf(" LIMIT %d", args.Count)

	// Execute sample query
	var results []map[string]interface{}
	rawQuery := s.db.Raw(query, parameters...)
	if err := rawQuery.Find(ctx, &results); err != nil {
		return &ToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("Error sampling data: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format results
	response := map[string]interface{}{
		"table":       args.Table,
		"sample_size": len(results),
		"requested":   args.Count,
		"random":      args.Random,
		"conditions":  args.Where,
		"data":        results,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sample: %w", err)
	}

	return &ToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

// analyzeColumn performs statistical analysis on a column
func (s *Server) analyzeColumn(columnName string, data []map[string]interface{}) ColumnStats {
	stats := ColumnStats{
		DataType:     "unknown",
		SampleValues: make([]interface{}, 0, 10),
	}

	if len(data) == 0 {
		return stats
	}

	var values []interface{}
	uniqueValues := make(map[interface{}]bool)
	nullCount := 0

	// Collect values and count nulls/uniques
	for _, row := range data {
		if value, exists := row[columnName]; exists {
			if value == nil {
				nullCount++
			} else {
				values = append(values, value)
				uniqueValues[value] = true
				
				// Collect sample values (first 10 unique)
				if len(stats.SampleValues) < 10 {
					found := false
					for _, sample := range stats.SampleValues {
						if sample == value {
							found = true
							break
						}
					}
					if !found {
						stats.SampleValues = append(stats.SampleValues, value)
					}
				}
			}
		} else {
			nullCount++
		}
	}

	stats.NullCount = nullCount
	stats.UniqueCount = len(uniqueValues)

	// Determine data type and min/max from first non-null value
	if len(values) > 0 {
		firstValue := values[0]
		switch firstValue.(type) {
		case int, int32, int64:
			stats.DataType = "integer"
			min, max := s.findNumericMinMax(values, true)
			stats.MinValue = min
			stats.MaxValue = max
		case float32, float64:
			stats.DataType = "float"
			min, max := s.findNumericMinMax(values, false)
			stats.MinValue = min
			stats.MaxValue = max
		case string:
			stats.DataType = "string"
			min, max := s.findStringMinMax(values)
			stats.MinValue = min
			stats.MaxValue = max
		case bool:
			stats.DataType = "boolean"
		default:
			stats.DataType = "mixed"
		}
	}

	return stats
}

// findNumericMinMax finds min and max values for numeric columns
func (s *Server) findNumericMinMax(values []interface{}, isInteger bool) (interface{}, interface{}) {
	if len(values) == 0 {
		return nil, nil
	}

	var min, max float64
	first := true

	for _, value := range values {
		var numValue float64
		if isInteger {
			numValue = float64(utils.ToInt64(value))
		} else {
			numValue = utils.ToFloat64(value)
		}

		if first {
			min = numValue
			max = numValue
			first = false
		} else {
			if numValue < min {
				min = numValue
			}
			if numValue > max {
				max = numValue
			}
		}
	}

	if isInteger {
		return int64(min), int64(max)
	}
	return min, max
}

// findStringMinMax finds min and max values for string columns (by length and lexicographic order)
func (s *Server) findStringMinMax(values []interface{}) (interface{}, interface{}) {
	if len(values) == 0 {
		return nil, nil
	}

	minStr := utils.ToString(values[0])
	maxStr := minStr

	for _, value := range values[1:] {
		str := utils.ToString(value)
		if str < minStr {
			minStr = str
		}
		if str > maxStr {
			maxStr = str
		}
	}

	return minStr, maxStr
}

// callModelTool handles model management tools
func (s *Server) callModelTool(ctx context.Context, tool string, arguments json.RawMessage) (*ToolResult, error) {
	switch tool {
	case "model.create":
		return s.createModel(ctx, arguments)
	case "model.addField":
		return s.addFieldToModel(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown model tool: %s", tool)
	}
}

// callSchemaTool handles schema management tools
func (s *Server) callSchemaTool(ctx context.Context, tool string, arguments json.RawMessage) (*ToolResult, error) {
	switch tool {
	case "schema.sync":
		return s.syncSchema(ctx, arguments)
	case "schema.diff":
		return s.schemaDiff(ctx, arguments)
	case "schema.export":
		return s.schemaExport(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown schema tool: %s", tool)
	}
}

// Note: createModel is now implemented in tools_schema.go
// Note: syncSchema is now implemented in tools_schema.go

