package prisma

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rediwo/redi-orm/schema"
)

// ParseSchema parses Prisma schema content and returns schemas map
func ParseSchema(content string) (map[string]*schema.Schema, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("schema content is empty")
	}

	// Create lexer and parser
	lexer := NewLexer(content)
	parser := NewParser(lexer)

	// Parse the schema
	prismaSchema := parser.ParseSchema()

	// Check for parsing errors
	if errors := parser.Errors(); len(errors) > 0 {
		return nil, fmt.Errorf("schema parsing errors: %v", errors)
	}

	// Convert to ReORM schemas
	converter := NewConverter()
	schemas, err := converter.Convert(prismaSchema)
	if err != nil {
		return nil, fmt.Errorf("schema conversion failed: %w", err)
	}

	if len(schemas) == 0 {
		return nil, fmt.Errorf("no models found in schema")
	}

	return schemas, nil
}

// ParseSchemaFile reads and parses Prisma schema file
func ParseSchemaFile(filename string) (map[string]*schema.Schema, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open schema file %s: %w", filename, err)
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", filename, err)
	}

	// Parse the content
	return ParseSchema(string(content))
}

// LoadSchemaFromPath loads Prisma schemas from a file or directory
// This function supports loading a single .prisma file or all .prisma files from a directory
func LoadSchemaFromPath(path string) (map[string]*schema.Schema, error) {
	// Check if path exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("schema path not found: %s", path)
	}

	allSchemas := make(map[string]*schema.Schema)

	// If it's a directory, load all .prisma files
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		var prismaFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".prisma") {
				prismaFiles = append(prismaFiles, filepath.Join(path, entry.Name()))
			}
		}

		if len(prismaFiles) == 0 {
			return nil, fmt.Errorf("no .prisma files found in directory: %s", path)
		}

		for _, file := range prismaFiles {
			schemas, err := ParseSchemaFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", filepath.Base(file), err)
			}

			// Merge schemas
			for name, schema := range schemas {
				if _, exists := allSchemas[name]; exists {
					return nil, fmt.Errorf("duplicate model '%s' found in %s", name, filepath.Base(file))
				}
				allSchemas[name] = schema
			}
		}
	} else {
		// It's a single file
		schemas, err := ParseSchemaFile(path)
		if err != nil {
			return nil, err
		}
		allSchemas = schemas
	}

	return allSchemas, nil
}
