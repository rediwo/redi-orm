package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// SchemaPersistence manages reading and writing schema files
type SchemaPersistence struct {
	schemaPath string
	logger     logger.Logger
	generator  *SchemaGenerator

	// Track which schemas came from which files
	schemaFiles map[string]string // model name -> file path

	// Track datasource and generator per file
	fileDatasources map[string]*prisma.DatasourceStatement
	fileGenerators  map[string]*prisma.GeneratorStatement
}

// NewSchemaPersistence creates a new schema persistence manager
func NewSchemaPersistence(schemaPath string, l logger.Logger, migrator types.DatabaseMigrator) *SchemaPersistence {
	// Get the specific migrator for proper type conversion
	var specificMigrator types.DatabaseSpecificMigrator
	if wrapper, ok := migrator.(interface {
		GetSpecific() types.DatabaseSpecificMigrator
	}); ok {
		specificMigrator = wrapper.GetSpecific()
	}

	return &SchemaPersistence{
		schemaPath:      schemaPath,
		logger:          l,
		generator:       NewSchemaGenerator(specificMigrator),
		schemaFiles:     make(map[string]string),
		fileDatasources: make(map[string]*prisma.DatasourceStatement),
		fileGenerators:  make(map[string]*prisma.GeneratorStatement),
	}
}

// LoadSchemas loads all schemas and tracks their source files
func (sp *SchemaPersistence) LoadSchemas() ([]*schema.Schema, error) {
	info, err := os.Stat(sp.schemaPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If path doesn't exist, check if it looks like a directory path
			if strings.HasSuffix(sp.schemaPath, "/") || !strings.Contains(filepath.Base(sp.schemaPath), ".") {
				// Create directory if it doesn't exist
				sp.logger.Info("Creating schema directory: %s", sp.schemaPath)
				if err := os.MkdirAll(sp.schemaPath, 0755); err != nil {
					return nil, fmt.Errorf("failed to create schema directory: %w", err)
				}
				// Return empty schemas for new directory
				return []*schema.Schema{}, nil
			}
			// For file paths, return empty schemas
			return []*schema.Schema{}, nil
		}
		return nil, err
	}

	var allSchemas []*schema.Schema

	if info.IsDir() {
		// Load all .prisma files from directory
		files, err := os.ReadDir(sp.schemaPath)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".prisma" {
				continue
			}

			filePath := filepath.Join(sp.schemaPath, file.Name())
			schemas, datasource, generator, err := sp.loadSchemaFile(filePath)
			if err != nil {
				sp.logger.Warn("Failed to load schema file %s: %v", filePath, err)
				continue
			}

			// Track which models came from this file
			for _, s := range schemas {
				sp.schemaFiles[s.Name] = filePath
			}

			// Track datasource and generator for this file
			if datasource != nil {
				sp.fileDatasources[filePath] = datasource
			}
			if generator != nil {
				sp.fileGenerators[filePath] = generator
			}

			allSchemas = append(allSchemas, schemas...)
		}
	} else {
		// Load single file
		schemas, datasource, generator, err := sp.loadSchemaFile(sp.schemaPath)
		if err != nil {
			return nil, err
		}

		// Track which models came from this file
		for _, s := range schemas {
			sp.schemaFiles[s.Name] = sp.schemaPath
		}

		// Track datasource and generator
		if datasource != nil {
			sp.fileDatasources[sp.schemaPath] = datasource
		}
		if generator != nil {
			sp.fileGenerators[sp.schemaPath] = generator
		}

		allSchemas = schemas
	}

	sp.logger.Info("Loaded %d models from schema files", len(allSchemas))
	return allSchemas, nil
}

// loadSchemaFile loads schemas from a single file
func (sp *SchemaPersistence) loadSchemaFile(path string) ([]*schema.Schema, *prisma.DatasourceStatement, *prisma.GeneratorStatement, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, err
	}

	// Parse Prisma schema
	schemasMap, datasource, generator, err := parsePrismaWithMetadata(string(content))
	if err != nil {
		return nil, nil, nil, err
	}

	// Convert map to slice
	schemas := make([]*schema.Schema, 0, len(schemasMap))
	for _, s := range schemasMap {
		schemas = append(schemas, s)
	}

	return schemas, datasource, generator, nil
}

// SaveSchema saves a schema to the appropriate file
func (sp *SchemaPersistence) SaveSchema(s *schema.Schema) error {
	// Determine which file this schema should go to
	filePath := sp.getSchemaFilePath(s.Name)

	// Load all schemas from that file
	schemas, err := sp.loadSchemasFromFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Update or add the schema
	found := false
	for i, existing := range schemas {
		if existing.Name == s.Name {
			schemas[i] = s
			found = true
			break
		}
	}
	if !found {
		schemas = append(schemas, s)
	}

	// Write back to file
	return sp.writeSchemasToFile(filePath, schemas)
}

// RemoveSchema removes a schema from its file
func (sp *SchemaPersistence) RemoveSchema(modelName string) error {
	filePath := sp.getSchemaFilePath(modelName)

	// Load all schemas from that file
	schemas, err := sp.loadSchemasFromFile(filePath)
	if err != nil {
		return err
	}

	// Remove the schema
	filtered := make([]*schema.Schema, 0, len(schemas)-1)
	for _, s := range schemas {
		if s.Name != modelName {
			filtered = append(filtered, s)
		}
	}

	// Delete tracking
	delete(sp.schemaFiles, modelName)

	// If no schemas left in file, delete it
	if len(filtered) == 0 && filePath != sp.schemaPath {
		return os.Remove(filePath)
	}

	// Write back to file
	return sp.writeSchemasToFile(filePath, filtered)
}

// getSchemaFilePath determines which file a schema should be saved to
func (sp *SchemaPersistence) getSchemaFilePath(modelName string) string {
	// If we already know where this schema came from, use that
	if path, ok := sp.schemaFiles[modelName]; ok {
		return path
	}

	// Otherwise, determine based on schema path
	info, err := os.Stat(sp.schemaPath)
	if err != nil || !info.IsDir() {
		// Single file mode
		return sp.schemaPath
	}

	// Directory mode - create a new file for this model
	fileName := fmt.Sprintf("%s.prisma", strings.ToLower(modelName))
	return filepath.Join(sp.schemaPath, fileName)
}

// loadSchemasFromFile loads all schemas from a specific file
func (sp *SchemaPersistence) loadSchemasFromFile(filePath string) ([]*schema.Schema, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*schema.Schema{}, nil
	}

	schemas, _, _, err := sp.loadSchemaFile(filePath)
	return schemas, err
}

// writeSchemasToFile writes schemas to a specific file
func (sp *SchemaPersistence) writeSchemasToFile(filePath string, schemas []*schema.Schema) error {
	// Get datasource and generator for this file (if they exist from original file)
	datasource := sp.fileDatasources[filePath]
	generator := sp.fileGenerators[filePath]

	// For new auto-generated files, add a generator block to indicate it's auto-generated
	if datasource == nil && generator == nil && len(schemas) > 0 {
		// Check if this is a new file being created by auto-generation
		// (not loaded from an existing file)
		if _, exists := sp.schemaFiles[schemas[0].Name]; !exists {
			generator = &prisma.GeneratorStatement{
				Name: "client",
				Properties: []*prisma.Property{
					{
						Name:  "provider",
						Value: &prisma.StringLiteral{Value: "RediORM Auto Generator"},
					},
				},
			}
		}
	}

	// Generate Prisma content
	content, err := sp.generator.GenerateFullPrismaFile(schemas, datasource, generator)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write file
	return os.WriteFile(filePath, []byte(content), 0644)
}

// GetSchemaFiles returns the mapping of model names to file paths
func (sp *SchemaPersistence) GetSchemaFiles() map[string]string {
	return sp.schemaFiles
}

// parsePrismaWithMetadata parses Prisma schema and returns schemas plus metadata
func parsePrismaWithMetadata(content string) (map[string]*schema.Schema, *prisma.DatasourceStatement, *prisma.GeneratorStatement, error) {
	// Parse schema
	lexer := prisma.NewLexer(content)
	parser := prisma.NewParser(lexer)
	ast := parser.ParseSchema()

	if errs := parser.Errors(); len(errs) > 0 {
		return nil, nil, nil, fmt.Errorf("parser errors: %s", strings.Join(errs, "; "))
	}

	// Convert to schemas
	converter := prisma.NewConverter()
	schemas, err := converter.Convert(ast)
	if err != nil {
		return nil, nil, nil, err
	}

	return schemas, converter.GetDatasource(), converter.GetGenerator(), nil
}
