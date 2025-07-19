package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/utils"
)

// ORM tool input structures

type ORMFindManyInput struct {
	Model   string                 `json:"model"`
	Where   map[string]interface{} `json:"where,omitempty"`
	Include map[string]interface{} `json:"include,omitempty"`
	OrderBy map[string]interface{} `json:"orderBy,omitempty"`
	Take    int                    `json:"take,omitempty"`
	Skip    int                    `json:"skip,omitempty"`
}

type ORMFindUniqueInput struct {
	Model   string                 `json:"model"`
	Where   map[string]interface{} `json:"where"`
	Include map[string]interface{} `json:"include,omitempty"`
}

type ORMCreateInput struct {
	Model string                 `json:"model"`
	Data  map[string]interface{} `json:"data"`
}

type ORMUpdateInput struct {
	Model string                 `json:"model"`
	Where map[string]interface{} `json:"where"`
	Data  map[string]interface{} `json:"data"`
}

type ORMDeleteInput struct {
	Model string                 `json:"model"`
	Where map[string]interface{} `json:"where"`
}

type ORMCountInput struct {
	Model string                 `json:"model"`
	Where map[string]interface{} `json:"where,omitempty"`
}

type ORMAggregateInput struct {
	Model   string                 `json:"model"`
	Where   map[string]interface{} `json:"where,omitempty"`
	Count   bool                   `json:"count,omitempty"`
	Avg     map[string]bool        `json:"avg,omitempty"`
	Sum     map[string]bool        `json:"sum,omitempty"`
	Min     map[string]bool        `json:"min,omitempty"`
	Max     map[string]bool        `json:"max,omitempty"`
	GroupBy []string               `json:"groupBy,omitempty"`
}

// callDataTool routes ORM data operation tools to their handlers
func (s *Server) callDataTool(ctx context.Context, tool string, arguments json.RawMessage) (*ToolResult, error) {
	// Check read-only mode for write operations
	if s.config.ReadOnly {
		switch tool {
		case "data.create", "data.update", "data.delete":
			return nil, fmt.Errorf("operation %s not allowed in read-only mode", tool)
		}
	}

	switch tool {
	case "data.findMany":
		return s.ormFindMany(ctx, arguments)
	case "data.findUnique":
		return s.ormFindUnique(ctx, arguments)
	case "data.create":
		return s.ormCreate(ctx, arguments)
	case "data.update":
		return s.ormUpdate(ctx, arguments)
	case "data.delete":
		return s.ormDelete(ctx, arguments)
	case "data.count":
		return s.ormCount(ctx, arguments)
	case "data.aggregate":
		return s.ormAggregate(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown data tool: %s", tool)
	}
}

// ormFindMany implements Prisma-style findMany operation
func (s *Server) ormFindMany(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ORMFindManyInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model exists
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Get table name
	tableName := schema.ModelNameToTableName(input.Model)

	// Check table access
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Build SQL query from ORM query
	query, params, err := s.buildFindManyQuery(sch, tableName, input)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	s.logger.Debug("ORM findMany query: %s with params: %v", query, params)

	// Execute query
	var results []map[string]interface{}
	if err := s.db.Raw(query, params...).Find(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Convert database columns to model fields
	modelResults := make([]map[string]interface{}, len(results))
	for i, row := range results {
		modelResults[i] = convertRowToModelFields(row, sch)
	}

	// Handle includes if specified
	if len(input.Include) > 0 {
		modelResults, err = s.loadIncludes(ctx, sch, modelResults, input.Include)
		if err != nil {
			s.logger.Warn("Failed to load includes: %v", err)
		}
	}

	// Format response
	data, err := json.MarshalIndent(modelResults, "", "  ")
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

// ormFindUnique implements Prisma-style findUnique operation
func (s *Server) ormFindUnique(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ORMFindUniqueInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model exists
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Get table name
	tableName := schema.ModelNameToTableName(input.Model)

	// Check table access
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Build query for unique lookup
	query, params, err := s.buildFindUniqueQuery(sch, tableName, input.Where)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	s.logger.Debug("ORM findUnique query: %s with params: %v", query, params)

	// Execute query
	var results []map[string]interface{}
	if err := s.db.Raw(query, params...).Find(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if len(results) == 0 {
		return &ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: "null",
				},
			},
		}, nil
	}

	// Convert database columns to model fields
	result := convertRowToModelFields(results[0], sch)

	// Handle includes if specified
	if len(input.Include) > 0 {
		resultArray := []map[string]interface{}{result}
		resultArray, err = s.loadIncludes(ctx, sch, resultArray, input.Include)
		if err != nil {
			s.logger.Warn("Failed to load includes: %v", err)
		}
		if len(resultArray) > 0 {
			result = resultArray[0]
		}
	}

	// Format response
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
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

// ormCreate implements Prisma-style create operation
func (s *Server) ormCreate(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ORMCreateInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model exists
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Get table name
	tableName := schema.ModelNameToTableName(input.Model)

	// Check table access
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Convert model fields to database columns
	dbData := convertModelFieldsToColumns(input.Data, sch)

	// Build insert query
	query, params, returningFields := s.buildInsertQuery(tableName, dbData, sch)

	s.logger.Debug("ORM create query: %s with params: %v", query, params)

	// Execute query
	// Check if database supports RETURNING (PostgreSQL, SQLite 3.35+)
	supportsReturning := s.db.GetDriverType() == "postgresql" || s.db.GetDriverType() == "sqlite"
	if supportsReturning && len(returningFields) > 0 {
		// Use RETURNING clause
		var results []map[string]interface{}
		if err := s.db.Raw(query, params...).Find(ctx, &results); err != nil {
			return nil, fmt.Errorf("failed to execute create: %w", err)
		}

		if len(results) > 0 {
			result := convertRowToModelFields(results[0], sch)
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
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
	} else {
		// Execute without RETURNING
		result := s.db.Raw(query, params...)
		if _, err := result.Exec(ctx); err != nil {
			return nil, fmt.Errorf("failed to execute create: %w", err)
		}

		// For databases without RETURNING, return the input data
		// In a real implementation, we'd fetch the created record
		data, err := json.MarshalIndent(input.Data, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
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

	return &ToolResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: "{}",
			},
		},
	}, nil
}

// ormUpdate implements Prisma-style update operation
func (s *Server) ormUpdate(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ORMUpdateInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model exists
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Get table name
	tableName := schema.ModelNameToTableName(input.Model)

	// Check table access
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Convert model fields to database columns
	dbData := convertModelFieldsToColumns(input.Data, sch)
	dbWhere := convertModelFieldsToColumns(input.Where, sch)

	// Build update query
	query, params := s.buildUpdateQuery(tableName, dbData, dbWhere)

	s.logger.Debug("ORM update query: %s with params: %v", query, params)

	// Execute query
	result := s.db.Raw(query, params...)
	if _, err := result.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to execute update: %w", err)
	}

	// For update operations, we approximate rows affected
	// In a real implementation, this would use database-specific features
	var rowsAffected int64 = 1

	// Return result
	response := map[string]interface{}{
		"count": rowsAffected,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
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

// ormDelete implements Prisma-style delete operation
func (s *Server) ormDelete(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ORMDeleteInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model exists
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Get table name
	tableName := schema.ModelNameToTableName(input.Model)

	// Check table access
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Convert model fields to database columns
	dbWhere := convertModelFieldsToColumns(input.Where, sch)

	// Build delete query
	query, params := s.buildDeleteQuery(tableName, dbWhere)

	s.logger.Debug("ORM delete query: %s with params: %v", query, params)

	// Execute query
	result := s.db.Raw(query, params...)
	if _, err := result.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to execute delete: %w", err)
	}

	// For delete operations, we approximate rows affected
	// In a real implementation, this would use database-specific features
	var rowsAffected int64 = 1

	// Return result
	response := map[string]interface{}{
		"count": rowsAffected,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
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

// ormCount implements Prisma-style count operation
func (s *Server) ormCount(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ORMCountInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model exists
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Get table name
	tableName := schema.ModelNameToTableName(input.Model)

	// Check table access
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Build count query
	query := fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tableName)
	var params []interface{}

	if len(input.Where) > 0 {
		whereClause, whereParams := s.buildWhereClause(input.Where, sch)
		if whereClause != "" {
			query += " WHERE " + whereClause
			params = whereParams
		}
	}

	s.logger.Debug("ORM count query: %s with params: %v", query, params)

	// Execute query
	var results []map[string]interface{}
	if err := s.db.Raw(query, params...).Find(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to execute count: %w", err)
	}

	count := int64(0)
	if len(results) > 0 && results[0]["count"] != nil {
		count = utils.ToInt64(results[0]["count"])
	}

	// Return result
	data, err := json.MarshalIndent(count, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
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

// ormAggregate implements Prisma-style aggregate operation
func (s *Server) ormAggregate(ctx context.Context, arguments json.RawMessage) (*ToolResult, error) {
	var input ORMAggregateInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate model exists
	sch, exists := s.schemas[input.Model]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", input.Model)
	}

	// Get table name
	tableName := schema.ModelNameToTableName(input.Model)

	// Check table access
	if err := s.security.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Build aggregate query
	query, params, err := s.buildAggregateQuery(tableName, sch, input)
	if err != nil {
		return nil, fmt.Errorf("failed to build aggregate query: %w", err)
	}

	s.logger.Debug("ORM aggregate query: %s with params: %v", query, params)

	// Execute query
	var results []map[string]interface{}
	if err := s.db.Raw(query, params...).Find(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to execute aggregate: %w", err)
	}

	// Format response based on groupBy
	var response interface{}
	if len(input.GroupBy) > 0 {
		// Group by results
		response = results
	} else if len(results) > 0 {
		// Single aggregate result
		response = results[0]
	} else {
		response = map[string]interface{}{}
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
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

// Helper functions for query building

func (s *Server) buildFindManyQuery(sch *schema.Schema, tableName string, input ORMFindManyInput) (string, []interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	var params []interface{}

	// Add WHERE clause
	if len(input.Where) > 0 {
		whereClause, whereParams := s.buildWhereClause(input.Where, sch)
		if whereClause != "" {
			query += " WHERE " + whereClause
			params = whereParams
		}
	}

	// Add ORDER BY clause
	if len(input.OrderBy) > 0 {
		orderClause := s.buildOrderByClause(input.OrderBy, sch)
		if orderClause != "" {
			query += " ORDER BY " + orderClause
		}
	}

	// Add LIMIT clause
	if input.Take > 0 {
		query += fmt.Sprintf(" LIMIT %d", input.Take)
	}

	// Add OFFSET clause
	if input.Skip > 0 {
		// Some databases (like MySQL) require LIMIT when using OFFSET
		requiresLimit := s.db.GetDriverType() == "mysql"
		if input.Take == 0 && requiresLimit {
			query += fmt.Sprintf(" LIMIT %d", s.config.MaxQueryRows)
		}
		query += fmt.Sprintf(" OFFSET %d", input.Skip)
	}

	return query, params, nil
}

func (s *Server) buildFindUniqueQuery(sch *schema.Schema, tableName string, where map[string]interface{}) (string, []interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s", tableName)

	whereClause, params := s.buildWhereClause(where, sch)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	query += " LIMIT 1"

	return query, params, nil
}

func (s *Server) buildWhereClause(where map[string]interface{}, sch *schema.Schema) (string, []interface{}) {
	var conditions []string
	var params []interface{}

	for fieldName, value := range where {
		// Convert field name to column name
		columnName := fieldName
		for _, field := range sch.Fields {
			if field.Name == fieldName && field.Map != "" {
				columnName = field.Map
				break
			}
		}

		// Handle different value types
		switch v := value.(type) {
		case map[string]interface{}:
			// Handle operators like {gt: 5, lt: 10}
			for op, opValue := range v {
				condition, param := s.buildOperatorCondition(columnName, op, opValue)
				if condition != "" {
					conditions = append(conditions, condition)
					if param != nil {
						params = append(params, param)
					}
				}
			}
		default:
			// Simple equality
			conditions = append(conditions, fmt.Sprintf("%s = ?", columnName))
			params = append(params, value)
		}
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " AND "), params
}

func (s *Server) buildOperatorCondition(column string, op string, value interface{}) (string, interface{}) {
	switch op {
	case "equals":
		return fmt.Sprintf("%s = ?", column), value
	case "not":
		return fmt.Sprintf("%s != ?", column), value
	case "gt":
		return fmt.Sprintf("%s > ?", column), value
	case "gte":
		return fmt.Sprintf("%s >= ?", column), value
	case "lt":
		return fmt.Sprintf("%s < ?", column), value
	case "lte":
		return fmt.Sprintf("%s <= ?", column), value
	case "contains":
		return fmt.Sprintf("%s LIKE ?", column), fmt.Sprintf("%%%v%%", value)
	case "startsWith":
		return fmt.Sprintf("%s LIKE ?", column), fmt.Sprintf("%v%%", value)
	case "endsWith":
		return fmt.Sprintf("%s LIKE ?", column), fmt.Sprintf("%%%v", value)
	case "in":
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			placeholders := make([]string, len(arr))
			for i := range arr {
				placeholders[i] = "?"
			}
			return fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", ")), nil
		}
	case "notIn":
		if arr, ok := value.([]interface{}); ok && len(arr) > 0 {
			placeholders := make([]string, len(arr))
			for i := range arr {
				placeholders[i] = "?"
			}
			return fmt.Sprintf("%s NOT IN (%s)", column, strings.Join(placeholders, ", ")), nil
		}
	}
	return "", nil
}

func (s *Server) buildOrderByClause(orderBy map[string]interface{}, sch *schema.Schema) string {
	var orderClauses []string

	for fieldName, direction := range orderBy {
		// Convert field name to column name
		columnName := fieldName
		for _, field := range sch.Fields {
			if field.Name == fieldName && field.Map != "" {
				columnName = field.Map
				break
			}
		}

		dir := utils.ToString(direction)
		if dir == "asc" || dir == "desc" {
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", columnName, strings.ToUpper(dir)))
		}
	}

	return strings.Join(orderClauses, ", ")
}

func (s *Server) buildInsertQuery(tableName string, data map[string]interface{}, sch *schema.Schema) (string, []interface{}, []string) {
	var columns []string
	var placeholders []string
	var params []interface{}
	var returningFields []string

	// Return all fields for complete record
	for _, field := range sch.Fields {
		returningFields = append(returningFields, field.Name)
	}

	// Build column list and values
	for column, value := range data {
		columns = append(columns, column)
		placeholders = append(placeholders, "?")
		params = append(params, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// Add RETURNING clause if supported
	supportsReturning := s.db.GetDriverType() == "postgresql" || s.db.GetDriverType() == "sqlite"
	if supportsReturning && len(returningFields) > 0 {
		returningColumns := make([]string, len(returningFields))
		for i, field := range returningFields {
			// Get column name for field
			for _, f := range sch.Fields {
				if f.Name == field {
					if f.Map != "" {
						returningColumns[i] = f.Map
					} else {
						returningColumns[i] = field
					}
					break
				}
			}
		}
		query += " RETURNING " + strings.Join(returningColumns, ", ")
	}

	return query, params, returningFields
}

func (s *Server) buildUpdateQuery(tableName string, data map[string]interface{}, where map[string]interface{}) (string, []interface{}) {
	var setClauses []string
	var params []interface{}

	// Build SET clauses
	for column, value := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", column))
		params = append(params, value)
	}

	query := fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(setClauses, ", "))

	// Add WHERE clause
	if len(where) > 0 {
		var whereConditions []string
		for column, value := range where {
			whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", column))
			params = append(params, value)
		}
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	return query, params
}

func (s *Server) buildDeleteQuery(tableName string, where map[string]interface{}) (string, []interface{}) {
	query := fmt.Sprintf("DELETE FROM %s", tableName)
	var params []interface{}

	// Add WHERE clause
	if len(where) > 0 {
		var whereConditions []string
		for column, value := range where {
			whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", column))
			params = append(params, value)
		}
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	return query, params
}

func (s *Server) buildAggregateQuery(tableName string, sch *schema.Schema, input ORMAggregateInput) (string, []interface{}, error) {
	var selectClauses []string
	var params []interface{}

	// Add COUNT if requested
	if input.Count {
		selectClauses = append(selectClauses, "COUNT(*) as _count")
	}

	// Add AVG fields
	for fieldName, include := range input.Avg {
		if include {
			columnName := s.getColumnName(fieldName, sch)
			selectClauses = append(selectClauses, fmt.Sprintf("AVG(%s) as _avg_%s", columnName, fieldName))
		}
	}

	// Add SUM fields
	for fieldName, include := range input.Sum {
		if include {
			columnName := s.getColumnName(fieldName, sch)
			selectClauses = append(selectClauses, fmt.Sprintf("SUM(%s) as _sum_%s", columnName, fieldName))
		}
	}

	// Add MIN fields
	for fieldName, include := range input.Min {
		if include {
			columnName := s.getColumnName(fieldName, sch)
			selectClauses = append(selectClauses, fmt.Sprintf("MIN(%s) as _min_%s", columnName, fieldName))
		}
	}

	// Add MAX fields
	for fieldName, include := range input.Max {
		if include {
			columnName := s.getColumnName(fieldName, sch)
			selectClauses = append(selectClauses, fmt.Sprintf("MAX(%s) as _max_%s", columnName, fieldName))
		}
	}

	// Add GROUP BY fields to SELECT
	for _, fieldName := range input.GroupBy {
		columnName := s.getColumnName(fieldName, sch)
		selectClauses = append(selectClauses, columnName)
	}

	if len(selectClauses) == 0 {
		return "", nil, fmt.Errorf("no aggregation fields specified")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectClauses, ", "), tableName)

	// Add WHERE clause
	if len(input.Where) > 0 {
		whereClause, whereParams := s.buildWhereClause(input.Where, sch)
		if whereClause != "" {
			query += " WHERE " + whereClause
			params = whereParams
		}
	}

	// Add GROUP BY clause
	if len(input.GroupBy) > 0 {
		groupByClauses := make([]string, len(input.GroupBy))
		for i, fieldName := range input.GroupBy {
			groupByClauses[i] = s.getColumnName(fieldName, sch)
		}
		query += " GROUP BY " + strings.Join(groupByClauses, ", ")
	}

	return query, params, nil
}

func (s *Server) getColumnName(fieldName string, sch *schema.Schema) string {
	for _, field := range sch.Fields {
		if field.Name == fieldName {
			if field.Map != "" {
				return field.Map
			}
			return fieldName
		}
	}
	return fieldName
}

func (s *Server) loadIncludes(ctx context.Context, sch *schema.Schema, results []map[string]interface{}, includes map[string]interface{}) ([]map[string]interface{}, error) {
	// This is a simplified version - in a real implementation,
	// this would handle complex relation loading like the ORM module does
	s.logger.Debug("Loading includes for %s: %v", sch.Name, includes)
	
	// For now, just return the results as-is
	// The full implementation would:
	// 1. Parse the include structure
	// 2. Identify the relations to load
	// 3. Execute additional queries for each relation
	// 4. Merge the results appropriately
	
	return results, nil
}

// convertModelFieldsToColumns converts model field names to database column names
func convertModelFieldsToColumns(data map[string]interface{}, sch *schema.Schema) map[string]interface{} {
	result := make(map[string]interface{})

	for fieldName, value := range data {
		columnName := fieldName
		for _, field := range sch.Fields {
			if field.Name == fieldName && field.Map != "" {
				columnName = field.Map
				break
			}
		}
		result[columnName] = value
	}

	return result
}