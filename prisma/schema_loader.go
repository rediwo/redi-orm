package prisma

import (
	"fmt"
	"io"
	"os"
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