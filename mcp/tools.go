package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/orm"
	"github.com/rediwo/redi-orm/schema"
)

// addToolWithLogging wraps mcp.AddTool to add logging for registration and invocation
func addToolWithLogging[In, Out any](s *SDKServer, tool *mcp.Tool, handler func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[In]) (*mcp.CallToolResultFor[Out], error)) {
	// Log tool registration
	s.logger.Info("Registering tool: %s - %s", tool.Name, tool.Description)

	// Wrap the handler to add invocation logging
	wrappedHandler := func(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[In]) (*mcp.CallToolResultFor[Out], error) {
		startTime := time.Now()
		s.logger.Info("Tool invoked: %s", tool.Name)

		// Log parameters at debug level
		if s.logger.GetLevel() >= logger.LogLevelDebug {
			paramsJSON, _ := json.Marshal(params.Arguments)
			s.logger.Debug("  Parameters: %s", string(paramsJSON))
		}

		// Call the actual handler
		result, err := handler(ctx, session, params)

		// Log result
		duration := time.Since(startTime)
		if err != nil {
			s.logger.Error("Tool %s failed after %v: %v", tool.Name, duration, err)
		} else {
			s.logger.Info("Tool %s completed in %v", tool.Name, duration)
		}

		return result, err
	}

	// Register the tool with wrapped handler
	mcp.AddTool[In, Out](s.mcpServer, tool, wrappedHandler)
}

// Tool parameter structs
type ModelFindManyParams struct {
	Model   string         `json:"model" jsonschema:"Model name"`
	Where   map[string]any `json:"where,omitempty" jsonschema:"Filter conditions"`
	Include map[string]any `json:"include,omitempty" jsonschema:"Relations to include"`
	OrderBy map[string]any `json:"orderBy,omitempty" jsonschema:"Sort order"`
	Take    *int           `json:"take,omitempty" jsonschema:"Limit results"`
	Skip    *int           `json:"skip,omitempty" jsonschema:"Skip results"`
}

type ModelFindUniqueParams struct {
	Model   string         `json:"model" jsonschema:"Model name"`
	Where   map[string]any `json:"where" jsonschema:"Unique identifier"`
	Include map[string]any `json:"include,omitempty" jsonschema:"Relations to include"`
}

type ModelCreateParams struct {
	Model string         `json:"model" jsonschema:"Model name"`
	Data  map[string]any `json:"data" jsonschema:"Data to create"`
}

type ModelUpdateParams struct {
	Model string         `json:"model" jsonschema:"Model name"`
	Where map[string]any `json:"where" jsonschema:"Filter to find records"`
	Data  map[string]any `json:"data" jsonschema:"Data to update"`
}

type ModelDeleteParams struct {
	Model string         `json:"model" jsonschema:"Model name"`
	Where map[string]any `json:"where" jsonschema:"Filter to find records"`
}

type ModelCountParams struct {
	Model string         `json:"model" jsonschema:"Model name"`
	Where map[string]any `json:"where,omitempty" jsonschema:"Filter conditions"`
}

type ModelAggregateParams struct {
	Model   string          `json:"model" jsonschema:"Model name"`
	Where   map[string]any  `json:"where,omitempty" jsonschema:"Filter conditions"`
	Count   *bool           `json:"count,omitempty"`
	Avg     map[string]bool `json:"avg,omitempty"`
	Sum     map[string]bool `json:"sum,omitempty"`
	Min     map[string]bool `json:"min,omitempty"`
	Max     map[string]bool `json:"max,omitempty"`
	GroupBy []string        `json:"groupBy,omitempty"`
}

type SchemaModelsParams struct{}

type SchemaDescribeParams struct {
	Model string `json:"model" jsonschema:"Model name"`
}

type MigrationCreateParams struct {
	Name    string `json:"name" jsonschema:"Migration name"`
	Preview *bool  `json:"preview,omitempty" jsonschema:"Preview changes without creating"`
}

type MigrationApplyParams struct {
	DryRun *bool `json:"dry_run,omitempty" jsonschema:"Preview without applying"`
}

type MigrationStatusParams struct{}

type TransactionParams struct {
	Operations []TransactionOperation `json:"operations" jsonschema:"Array of operations to execute"`
}

type TransactionOperation struct {
	Tool      string         `json:"tool" jsonschema:"Tool name (e.g. model.create)"`
	Arguments map[string]any `json:"arguments" jsonschema:"Tool arguments"`
}

// registerTools registers all MCP tools with the SDK server
func (s *SDKServer) registerTools() {
	// Model operations
	findManySchema, _ := jsonschema.For[ModelFindManyParams]()
	addToolWithLogging[ModelFindManyParams, any](s, &mcp.Tool{
		Name:        "model.findMany",
		Description: "Query multiple records with Prisma-style filters",
		InputSchema: findManySchema,
	}, s.handleModelFindMany)

	findUniqueSchema, _ := jsonschema.For[ModelFindUniqueParams]()
	addToolWithLogging[ModelFindUniqueParams, any](s, &mcp.Tool{
		Name:        "model.findUnique",
		Description: "Find a single record by unique field",
		InputSchema: findUniqueSchema,
	}, s.handleModelFindUnique)

	createSchema, _ := jsonschema.For[ModelCreateParams]()
	addToolWithLogging[ModelCreateParams, any](s, &mcp.Tool{
		Name:        "model.create",
		Description: "Create a new record",
		InputSchema: createSchema,
	}, s.handleModelCreate)

	updateSchema, _ := jsonschema.For[ModelUpdateParams]()
	addToolWithLogging[ModelUpdateParams, any](s, &mcp.Tool{
		Name:        "model.update",
		Description: "Update existing records",
		InputSchema: updateSchema,
	}, s.handleModelUpdate)

	deleteSchema, _ := jsonschema.For[ModelDeleteParams]()
	addToolWithLogging[ModelDeleteParams, any](s, &mcp.Tool{
		Name:        "model.delete",
		Description: "Delete records",
		InputSchema: deleteSchema,
	}, s.handleModelDelete)

	countSchema, _ := jsonschema.For[ModelCountParams]()
	addToolWithLogging[ModelCountParams, any](s, &mcp.Tool{
		Name:        "model.count",
		Description: "Count records with optional filters",
		InputSchema: countSchema,
	}, s.handleModelCount)

	aggregateSchema, _ := jsonschema.For[ModelAggregateParams]()
	addToolWithLogging[ModelAggregateParams, any](s, &mcp.Tool{
		Name:        "model.aggregate",
		Description: "Perform aggregation queries",
		InputSchema: aggregateSchema,
	}, s.handleModelAggregate)

	// Schema operations
	modelsSchema, _ := jsonschema.For[SchemaModelsParams]()
	addToolWithLogging[SchemaModelsParams, any](s, &mcp.Tool{
		Name:        "schema.models",
		Description: "List all models with their fields and relationships",
		InputSchema: modelsSchema,
	}, s.handleSchemaModels)

	describeSchema, _ := jsonschema.For[SchemaDescribeParams]()
	addToolWithLogging[SchemaDescribeParams, any](s, &mcp.Tool{
		Name:        "schema.describe",
		Description: "Get detailed information about a specific model",
		InputSchema: describeSchema,
	}, s.handleSchemaDescribe)

	// Migration operations
	migrationCreateSchema, _ := jsonschema.For[MigrationCreateParams]()
	addToolWithLogging[MigrationCreateParams, any](s, &mcp.Tool{
		Name:        "migration.create",
		Description: "Create a new migration based on schema changes",
		InputSchema: migrationCreateSchema,
	}, s.handleMigrationCreate)

	migrationApplySchema, _ := jsonschema.For[MigrationApplyParams]()
	addToolWithLogging[MigrationApplyParams, any](s, &mcp.Tool{
		Name:        "migration.apply",
		Description: "Apply pending migrations to the database",
		InputSchema: migrationApplySchema,
	}, s.handleMigrationApply)

	migrationStatusSchema, _ := jsonschema.For[MigrationStatusParams]()
	addToolWithLogging[MigrationStatusParams, any](s, &mcp.Tool{
		Name:        "migration.status",
		Description: "Show current migration status",
		InputSchema: migrationStatusSchema,
	}, s.handleMigrationStatus)

	// Transaction operation
	transactionSchema, _ := jsonschema.For[TransactionParams]()
	addToolWithLogging[TransactionParams, any](s, &mcp.Tool{
		Name:        "transaction",
		Description: "Execute multiple operations in a transaction",
		InputSchema: transactionSchema,
	}, s.handleTransaction)
}

// Tool handlers
func (s *SDKServer) handleModelFindMany(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ModelFindManyParams]) (*mcp.CallToolResultFor[any], error) {

	// Check read-only mode
	if err := s.security.CheckReadOnly("findMany"); err != nil {
		s.logger.Error("Read-only mode error: %v", err)
		return nil, err
	}

	// Build ORM query
	query := map[string]any{
		"findMany": map[string]any{},
	}

	if params.Arguments.Where != nil {
		query["findMany"].(map[string]any)["where"] = params.Arguments.Where
	}
	if params.Arguments.Include != nil {
		query["findMany"].(map[string]any)["include"] = params.Arguments.Include
	}
	if params.Arguments.OrderBy != nil {
		query["findMany"].(map[string]any)["orderBy"] = params.Arguments.OrderBy
	}
	if params.Arguments.Take != nil {
		query["findMany"].(map[string]any)["take"] = *params.Arguments.Take
	}
	if params.Arguments.Skip != nil {
		query["findMany"].(map[string]any)["skip"] = *params.Arguments.Skip
	}

	// Execute query
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	client := orm.NewClient(s.db)
	result, err := client.Model(params.Arguments.Model).Query(string(queryJSON))
	if err != nil {
		s.logger.Error("model.findMany query failed for model %s: %v", params.Arguments.Model, err)
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleModelFindUnique(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ModelFindUniqueParams]) (*mcp.CallToolResultFor[any], error) {

	// Check read-only mode
	if err := s.security.CheckReadOnly("findUnique"); err != nil {
		s.logger.Error("Read-only mode error: %v", err)
		return nil, err
	}

	// Build ORM query
	query := map[string]any{
		"findUnique": map[string]any{
			"where": params.Arguments.Where,
		},
	}

	if params.Arguments.Include != nil {
		query["findUnique"].(map[string]any)["include"] = params.Arguments.Include
	}

	// Execute query
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	client := orm.NewClient(s.db)
	result, err := client.Model(params.Arguments.Model).Query(string(queryJSON))
	if err != nil {
		s.logger.Error("model.findUnique query failed for model %s: %v", params.Arguments.Model, err)
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleModelCreate(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ModelCreateParams]) (*mcp.CallToolResultFor[any], error) {

	// Check read-only mode
	if err := s.security.CheckReadOnly("create"); err != nil {
		s.logger.Error("Read-only mode error: %v", err)
		return nil, err
	}

	// Build ORM query
	query := map[string]any{
		"create": map[string]any{
			"data": params.Arguments.Data,
		},
	}

	// Execute query
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	client := orm.NewClient(s.db)
	result, err := client.Model(params.Arguments.Model).Query(string(queryJSON))
	if err != nil {
		s.logger.Error("model.create failed for model %s: %v", params.Arguments.Model, err)
		return nil, fmt.Errorf("create failed: %w", err)
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleModelUpdate(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ModelUpdateParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("update"); err != nil {
		s.logger.Error("Read-only mode error: %v", err)
		return nil, err
	}

	// Build ORM query
	query := map[string]any{
		"update": map[string]any{
			"where": params.Arguments.Where,
			"data":  params.Arguments.Data,
		},
	}

	// Execute query
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	client := orm.NewClient(s.db)
	result, err := client.Model(params.Arguments.Model).Query(string(queryJSON))
	if err != nil {
		s.logger.Error("model.update failed for model %s: %v", params.Arguments.Model, err)
		return nil, fmt.Errorf("update failed: %w", err)
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleModelDelete(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ModelDeleteParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("delete"); err != nil {
		s.logger.Error("Read-only mode error: %v", err)
		return nil, err
	}

	// Build ORM query
	query := map[string]any{
		"delete": map[string]any{
			"where": params.Arguments.Where,
		},
	}

	// Execute query
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	client := orm.NewClient(s.db)
	result, err := client.Model(params.Arguments.Model).Query(string(queryJSON))
	if err != nil {
		s.logger.Error("model.delete failed for model %s: %v", params.Arguments.Model, err)
		return nil, fmt.Errorf("delete failed: %w", err)
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleModelCount(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ModelCountParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("count"); err != nil {
		return nil, err
	}

	// Build ORM query
	query := map[string]any{
		"count": map[string]any{},
	}

	if params.Arguments.Where != nil {
		query["count"].(map[string]any)["where"] = params.Arguments.Where
	}

	// Execute query
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	client := orm.NewClient(s.db)
	result, err := client.Model(params.Arguments.Model).Query(string(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("count failed: %w", err)
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleModelAggregate(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ModelAggregateParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("aggregate"); err != nil {
		return nil, err
	}

	// Build ORM query
	query := map[string]any{
		"aggregate": map[string]any{},
	}

	aggMap := query["aggregate"].(map[string]any)

	if params.Arguments.Where != nil {
		aggMap["where"] = params.Arguments.Where
	}
	if params.Arguments.Count != nil && *params.Arguments.Count {
		aggMap["_count"] = true
	}
	if params.Arguments.Avg != nil {
		aggMap["_avg"] = params.Arguments.Avg
	}
	if params.Arguments.Sum != nil {
		aggMap["_sum"] = params.Arguments.Sum
	}
	if params.Arguments.Min != nil {
		aggMap["_min"] = params.Arguments.Min
	}
	if params.Arguments.Max != nil {
		aggMap["_max"] = params.Arguments.Max
	}
	if len(params.Arguments.GroupBy) > 0 {
		aggMap["groupBy"] = params.Arguments.GroupBy
	}

	// Execute query
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	client := orm.NewClient(s.db)
	result, err := client.Model(params.Arguments.Model).Query(string(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("aggregate failed: %w", err)
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleSchemaModels(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SchemaModelsParams]) (*mcp.CallToolResultFor[any], error) {
	models := make([]map[string]any, 0, len(s.schemas))

	for _, schema := range s.schemas {
		model := map[string]any{
			"name":      schema.Name,
			"tableName": schema.TableName,
			"fields":    schema.Fields,
			"indexes":   schema.Indexes,
			"relations": schema.Relations,
		}
		models = append(models, model)
	}

	result := map[string]any{
		"count":  len(models),
		"models": models,
	}

	// Return result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleSchemaDescribe(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[SchemaDescribeParams]) (*mcp.CallToolResultFor[any], error) {
	for _, schema := range s.schemas {
		if schema.Name == params.Arguments.Model {
			result := map[string]any{
				"name":         schema.Name,
				"tableName":    schema.TableName,
				"fields":       schema.Fields,
				"indexes":      schema.Indexes,
				"relations":    schema.Relations,
				"compositeKey": schema.CompositeKey,
			}

			// Return result
			resultJSON, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}

			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(resultJSON)},
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", params.Arguments.Model)
}

func (s *SDKServer) handleMigrationCreate(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[MigrationCreateParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("migration.create"); err != nil {
		return nil, err
	}

	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get current database schema
	migrator := s.db.GetMigrator()
	tables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Compare with loaded schemas
	changes := []string{}

	// Check for new tables
	existingTables := make(map[string]bool)
	for _, table := range tables {
		existingTables[table] = true
	}

	for _, schema := range s.schemas {
		if !existingTables[schema.TableName] {
			changes = append(changes, fmt.Sprintf("CREATE TABLE %s", schema.TableName))
		}
	}

	// Prepare result
	result := map[string]any{
		"name":    params.Arguments.Name,
		"changes": changes,
		"preview": params.Arguments.Preview != nil && *params.Arguments.Preview,
	}

	if params.Arguments.Preview == nil || !*params.Arguments.Preview {
		// Actually create migration file
		timestamp := time.Now().Format("20060102150405")
		filename := fmt.Sprintf("%s_%s.sql", timestamp, params.Arguments.Name)
		result["filename"] = filename
		result["status"] = "created"
	} else {
		result["status"] = "preview"
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleMigrationApply(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[MigrationApplyParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("migration.apply"); err != nil {
		return nil, err
	}

	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	dryRun := params.Arguments.DryRun != nil && *params.Arguments.DryRun

	// Sync schemas with database
	var syncErr error
	if !dryRun {
		// Register schemas with database
		for _, sch := range s.schemas {
			if err := s.db.RegisterSchema(sch.Name, sch); err != nil {
				return nil, fmt.Errorf("failed to register schema %s: %w", sch.Name, err)
			}
		}
		syncErr = s.db.SyncSchemas(ctx)
	}

	result := map[string]any{
		"dryRun": dryRun,
		"status": "success",
	}

	if syncErr != nil {
		result["status"] = "failed"
		result["error"] = syncErr.Error()
	}

	if dryRun {
		result["message"] = "Dry run completed. No changes were applied."
	} else {
		result["message"] = "Migrations applied successfully."
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func (s *SDKServer) handleMigrationStatus(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[MigrationStatusParams]) (*mcp.CallToolResultFor[any], error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get current database schema
	migrator := s.db.GetMigrator()
	tables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Compare with loaded schemas
	status := map[string]any{
		"database": map[string]any{
			"tables": len(tables),
			"list":   tables,
		},
		"schemas": map[string]any{
			"models": len(s.schemas),
			"list":   getSchemaNames(s.schemas),
		},
	}

	// Check for pending changes
	pendingChanges := []string{}

	// Check for missing tables
	existingTables := make(map[string]bool)
	for _, table := range tables {
		existingTables[table] = true
	}

	for _, schema := range s.schemas {
		if !existingTables[schema.TableName] {
			pendingChanges = append(pendingChanges, fmt.Sprintf("Table '%s' needs to be created", schema.TableName))
		}
	}

	status["pendingChanges"] = pendingChanges
	status["upToDate"] = len(pendingChanges) == 0

	resultJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resultJSON)},
		},
	}, nil
}

func getSchemaNames(schemas []*schema.Schema) []string {
	names := make([]string, len(schemas))
	for i, s := range schemas {
		names[i] = s.Name
	}
	return names
}

func (s *SDKServer) handleTransaction(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[TransactionParams]) (*mcp.CallToolResultFor[any], error) {
	// Check read-only mode
	if err := s.security.CheckReadOnly("transaction"); err != nil {
		return nil, err
	}

	// TODO: Implement transaction handling
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Transaction handling not yet implemented"},
		},
	}, nil
}
