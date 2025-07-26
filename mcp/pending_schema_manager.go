package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// PendingSchemaManager manages schemas that are waiting for their dependencies
type PendingSchemaManager struct {
	mu             sync.RWMutex
	pendingSchemas map[string]*schema.Schema // schemas waiting to be created as tables
	logger         logger.Logger
}

// NewPendingSchemaManager creates a new pending schema manager
func NewPendingSchemaManager(logger logger.Logger) *PendingSchemaManager {
	return &PendingSchemaManager{
		pendingSchemas: make(map[string]*schema.Schema),
		logger:         logger,
	}
}

// AddSchema adds a schema to the pending queue
func (p *PendingSchemaManager) AddSchema(s *schema.Schema) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.pendingSchemas[s.Name] = s
	p.logger.Debug("Added schema %s to pending queue", s.Name)
}

// GetPendingSchemas returns a copy of all pending schemas
func (p *PendingSchemaManager) GetPendingSchemas() map[string]*schema.Schema {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]*schema.Schema)
	for k, v := range p.pendingSchemas {
		result[k] = v
	}
	return result
}

// RemoveSchema removes a schema from the pending queue
func (p *PendingSchemaManager) RemoveSchema(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.pendingSchemas, name)
	p.logger.Debug("Removed schema %s from pending queue", name)
}

// TableCreationResult represents the result of attempting to create tables
type TableCreationResult struct {
	TablesCreated  []string            `json:"tables_created"`
	PendingSchemas []string            `json:"pending_schemas"`
	DependencyInfo map[string][]string `json:"dependency_info"`
	Errors         []string            `json:"errors,omitempty"`
	CircularDeps   bool                `json:"has_circular_dependencies"`
}

// ProcessPendingSchemas attempts to create tables for all pending schemas
func (p *PendingSchemaManager) ProcessPendingSchemas(ctx context.Context, db types.Database) (*TableCreationResult, error) {
	pendingSchemas := p.GetPendingSchemas()

	if len(pendingSchemas) == 0 {
		return &TableCreationResult{
			TablesCreated:  []string{},
			PendingSchemas: []string{},
			DependencyInfo: make(map[string][]string),
		}, nil
	}

	p.logger.Info("Processing %d pending schemas for table creation", len(pendingSchemas))

	// Get migrator
	migrator := db.GetMigrator()
	if migrator == nil {
		return nil, fmt.Errorf("database does not support migrations")
	}

	// Get current tables
	currentTables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing tables: %w", err)
	}

	currentTableMap := make(map[string]bool)
	for _, table := range currentTables {
		currentTableMap[table] = true
	}

	// Build dependency info for the result
	dependencyInfo := make(map[string][]string)
	for name, sch := range pendingSchemas {
		deps := p.extractDependencies(sch)
		if len(deps) > 0 {
			dependencyInfo[name] = deps
		}
	}

	// Analyze dependencies
	sortedModels, err := base.AnalyzeSchemasDependencies(pendingSchemas)
	if err != nil {
		// Circular dependency detected
		p.logger.Warn("Circular dependency detected, using deferred constraint creation")
		return p.createTablesWithDeferredConstraints(ctx, db, pendingSchemas, currentTableMap, dependencyInfo)
	}

	// Process schemas in dependency order
	result := &TableCreationResult{
		TablesCreated:  []string{},
		PendingSchemas: []string{},
		DependencyInfo: dependencyInfo,
		CircularDeps:   false,
	}

	for _, modelName := range sortedModels {
		sch, exists := pendingSchemas[modelName]
		if !exists || sch.TableName == "" {
			continue
		}

		// Check if all dependencies exist
		if !p.checkDependenciesExist(sch, currentTableMap, pendingSchemas) {
			result.PendingSchemas = append(result.PendingSchemas, modelName)
			p.logger.Debug("Schema %s still waiting for dependencies", modelName)
			continue
		}

		// Create table if it doesn't exist
		if !currentTableMap[sch.TableName] {
			if err := p.createTable(ctx, migrator, sch); err != nil {
				if result.Errors == nil {
					result.Errors = []string{}
				}
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create table %s: %v", sch.TableName, err))
				result.PendingSchemas = append(result.PendingSchemas, modelName)
				continue
			}

			result.TablesCreated = append(result.TablesCreated, sch.TableName)
			currentTableMap[sch.TableName] = true
			p.RemoveSchema(modelName)
			p.logger.Info("Successfully created table %s for schema %s", sch.TableName, modelName)
		} else {
			// Table already exists, remove from pending
			p.RemoveSchema(modelName)
			p.logger.Debug("Table %s already exists, removed schema %s from pending", sch.TableName, modelName)
		}
	}

	return result, nil
}

// extractDependencies extracts the names of models this schema depends on
func (p *PendingSchemaManager) extractDependencies(s *schema.Schema) []string {
	var deps []string
	depMap := make(map[string]bool)

	for _, relation := range s.Relations {
		// For manyToOne and oneToOne with foreign key, this schema depends on the referenced model
		if relation.Type == "manyToOne" ||
			(relation.Type == "oneToOne" && relation.ForeignKey != "") {
			if relation.Model != s.Name && !depMap[relation.Model] { // Skip self-references and duplicates
				deps = append(deps, relation.Model)
				depMap[relation.Model] = true
			}
		}
	}

	return deps
}

// checkDependenciesExist checks if all dependencies for a schema exist as tables or pending schemas
func (p *PendingSchemaManager) checkDependenciesExist(s *schema.Schema, currentTables map[string]bool, pendingSchemas map[string]*schema.Schema) bool {
	for _, relation := range s.Relations {
		if relation.Type == "manyToOne" ||
			(relation.Type == "oneToOne" && relation.ForeignKey != "") {
			if relation.Model == s.Name {
				continue // Skip self-references
			}

			// Check if the referenced model exists as a table or as a pending schema with table
			refSchema, pendingExists := pendingSchemas[relation.Model]
			if pendingExists && refSchema.TableName != "" {
				// Check if the referenced table exists
				if !currentTables[refSchema.TableName] {
					return false // Dependency table doesn't exist yet
				}
			} else {
				// Referenced model is not in pending schemas, it should exist as a table
				// We need to find the table name for this model
				// For now, assume the model name maps to a table name
				expectedTableName := schema.ModelNameToTableName(relation.Model)
				if !currentTables[expectedTableName] {
					return false // Dependency table doesn't exist
				}
			}
		}
	}
	return true
}

// createTable creates a single table with indexes
func (p *PendingSchemaManager) createTable(ctx context.Context, migrator types.DatabaseMigrator, s *schema.Schema) error {
	// Generate CREATE TABLE SQL (includes foreign key constraints)
	sql, err := migrator.GenerateCreateTableSQL(s)
	if err != nil {
		return fmt.Errorf("failed to generate CREATE TABLE SQL: %w", err)
	}

	// Apply the migration
	if err := migrator.ApplyMigration(sql); err != nil {
		return fmt.Errorf("failed to execute CREATE TABLE: %w", err)
	}

	// Create additional indexes (non-primary key, non-unique constraint indexes)
	for _, index := range s.Indexes {
		// Convert field names to column names
		columnNames := make([]string, len(index.Fields))
		for i, fieldName := range index.Fields {
			if field := s.GetFieldByName(fieldName); field != nil {
				columnNames[i] = field.GetColumnName()
			} else {
				columnNames[i] = fieldName // Fallback to field name
			}
		}

		indexSQL := migrator.GenerateCreateIndexSQL(s.TableName, index.Name, columnNames, index.Unique)
		if err := migrator.ApplyMigration(indexSQL); err != nil {
			return fmt.Errorf("failed to create index %s: %w", index.Name, err)
		}
	}

	return nil
}

// createTablesWithDeferredConstraints handles circular dependencies
func (p *PendingSchemaManager) createTablesWithDeferredConstraints(ctx context.Context, db types.Database, schemas map[string]*schema.Schema, currentTables map[string]bool, dependencyInfo map[string][]string) (*TableCreationResult, error) {
	migrator := db.GetMigrator()
	result := &TableCreationResult{
		TablesCreated:  []string{},
		PendingSchemas: []string{},
		DependencyInfo: dependencyInfo,
		CircularDeps:   true,
	}

	p.logger.Warn("Circular dependencies detected in schemas. Manual intervention may be required.")

	// For databases that don't support deferred constraints, we need to inform the user
	// that they'll need to handle this manually (e.g., by temporarily disabling foreign key checks)

	// Check if the database supports foreign keys
	dbDriver, ok := db.(interface {
		GetCapabilities() types.DriverCapabilities
	})
	if ok && dbDriver.GetCapabilities() != nil && !dbDriver.GetCapabilities().SupportsForeignKeys() {
		// Database doesn't support foreign keys, so circular deps aren't an issue
		p.logger.Info("Database doesn't support foreign key constraints - creating all tables normally")

		for _, sch := range schemas {
			if sch.TableName == "" || currentTables[sch.TableName] {
				continue
			}

			if err := p.createTable(ctx, migrator, sch); err != nil {
				if result.Errors == nil {
					result.Errors = []string{}
				}
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to create table %s: %v", sch.TableName, err))
				result.PendingSchemas = append(result.PendingSchemas, sch.Name)
			} else {
				result.TablesCreated = append(result.TablesCreated, sch.TableName)
				currentTables[sch.TableName] = true
				p.RemoveSchema(sch.Name)
			}
		}
		return result, nil
	}

	// For SQL databases with circular dependencies, we can't automatically handle this
	// because the DatabaseSpecificMigrator interface doesn't provide methods to:
	// 1. Create tables without foreign keys
	// 2. Add foreign keys separately

	if result.Errors == nil {
		result.Errors = []string{}
	}

	errorMsg := fmt.Sprintf(
		"Circular dependencies detected between schemas. The following schemas have circular dependencies: %v. "+
			"To resolve this, you need to manually: "+
			"1. Create the tables without foreign key constraints, "+
			"2. Then add the foreign key constraints after all tables are created. "+
			"This typically requires database-specific SQL commands to temporarily disable foreign key checks.",
		p.getCircularDependencyChain(schemas),
	)

	result.Errors = append(result.Errors, errorMsg)

	// Add all schemas to pending since we can't process them automatically
	for name := range schemas {
		result.PendingSchemas = append(result.PendingSchemas, name)
	}

	return result, nil
}

// getCircularDependencyChain identifies the circular dependency chain for error reporting
func (p *PendingSchemaManager) getCircularDependencyChain(schemas map[string]*schema.Schema) []string {
	var chain []string
	visited := make(map[string]bool)

	// Simple DFS to find a cycle
	var findCycle func(current string, path []string) []string
	findCycle = func(current string, path []string) []string {
		if visited[current] {
			// Found a cycle, extract the circular part
			for i, name := range path {
				if name == current {
					return path[i:]
				}
			}
			return []string{current}
		}

		visited[current] = true
		path = append(path, current)

		if sch, exists := schemas[current]; exists {
			for _, relation := range sch.Relations {
				if relation.Type == "manyToOne" ||
					(relation.Type == "oneToOne" && relation.ForeignKey != "") {
					if relation.Model != current && schemas[relation.Model] != nil {
						if cycle := findCycle(relation.Model, path); len(cycle) > 0 {
							return cycle
						}
					}
				}
			}
		}

		return nil
	}

	// Try to find a cycle starting from each schema
	for name := range schemas {
		visited = make(map[string]bool)
		if cycle := findCycle(name, nil); len(cycle) > 0 {
			chain = cycle
			break
		}
	}

	return chain
}
