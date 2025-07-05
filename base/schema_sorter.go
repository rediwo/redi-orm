package base

import (
	"fmt"

	"github.com/rediwo/redi-orm/schema"
)

// SchemaNode represents a schema in the dependency graph
type SchemaNode struct {
	Name         string
	Schema       *schema.Schema
	Dependencies []string
	Visited      bool
	InStack      bool
}

// AnalyzeSchemasDependencies analyzes dependencies between schemas and returns sorted order
func AnalyzeSchemasDependencies(schemas map[string]*schema.Schema) ([]string, error) {
	// Build dependency graph
	nodes := make(map[string]*SchemaNode)

	// Initialize nodes
	for name, sch := range schemas {
		node := &SchemaNode{
			Name:         name,
			Schema:       sch,
			Dependencies: []string{},
			Visited:      false,
			InStack:      false,
		}

		// Find dependencies from relations
		for _, relation := range sch.Relations {
			if relation.Type == "manyToOne" ||
				(relation.Type == "oneToOne" && relation.ForeignKey != "") {
				// This schema depends on the referenced model
				if relation.Model != name { // Skip self-references
					node.Dependencies = append(node.Dependencies, relation.Model)
				}
			}
		}

		nodes[name] = node
	}

	// Perform topological sort using DFS
	var sorted []string
	var visitNode func(string) error

	visitNode = func(name string) error {
		node, exists := nodes[name]
		if !exists {
			// Referenced model not in current schemas, skip
			return nil
		}

		if node.InStack {
			// Circular dependency detected
			return fmt.Errorf("circular dependency detected involving table: %s", name)
		}

		if node.Visited {
			return nil
		}

		node.InStack = true

		// Visit dependencies first
		for _, dep := range node.Dependencies {
			if err := visitNode(dep); err != nil {
				return err
			}
		}

		node.InStack = false
		node.Visited = true
		sorted = append(sorted, name)

		return nil
	}

	// Visit all nodes
	for name := range nodes {
		if err := visitNode(name); err != nil {
			return nil, err
		}
	}

	return sorted, nil
}

// splitSchemasByDependency separates schemas that can be created with foreign keys
// from those that need deferred constraint creation
func splitSchemasByDependency(schemas map[string]*schema.Schema) (withFK, withoutFK map[string]*schema.Schema, err error) {
	sorted, err := AnalyzeSchemasDependencies(schemas)
	if err != nil {
		// Has circular dependencies, all schemas need deferred FK creation
		withFK = make(map[string]*schema.Schema)
		withoutFK = schemas
		return withFK, withoutFK, nil
	}

	// No circular dependencies, all can be created with FK
	withFK = schemas
	withoutFK = make(map[string]*schema.Schema)

	// Return the sorted order
	sortedSchemas := make(map[string]*schema.Schema)
	for _, name := range sorted {
		if sch, exists := schemas[name]; exists {
			sortedSchemas[name] = sch
		}
	}

	return sortedSchemas, withoutFK, nil
}
