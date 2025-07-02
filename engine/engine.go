package engine

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"

	js "github.com/dop251/goja"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

type Engine struct {
	vm      *js.Runtime
	db      types.Database
	schemas map[string]*schema.Schema
	mu      sync.RWMutex
}

func New(db types.Database) *Engine {
	vm := js.New()
	engine := &Engine{
		vm:      vm,
		db:      db,
		schemas: make(map[string]*schema.Schema),
	}

	engine.setupGlobalObjects()
	return engine
}

func (e *Engine) RegisterSchema(s *schema.Schema) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := s.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	// Register schema with database (but don't create table yet)
	if err := e.db.RegisterSchema(s.Name, s); err != nil {
		return fmt.Errorf("failed to register schema: %w", err)
	}

	e.schemas[s.Name] = s

	// Register model in JavaScript context
	e.registerModelInJS(s.Name)

	return nil
}

// EnsureSchema performs auto-migration for all registered schemas
func (e *Engine) EnsureSchema() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// For now, just create tables for all registered schemas
	// Full migration support will be added later
	ctx := context.Background()
	for modelName := range e.schemas {
		if err := e.db.CreateModel(ctx, modelName); err != nil {
			return fmt.Errorf("failed to create model %s: %w", modelName, err)
		}
	}
	return nil
}

func (e *Engine) setupGlobalObjects() {
	// Create models object
	modelsObj := e.vm.NewObject()
	e.vm.Set("models", modelsObj)
}

func (e *Engine) registerModelInJS(name string) {
	modelObj := e.vm.NewObject()

	// Register findUnique method using new API
	modelObj.Set("findUnique", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("findUnique() requires a where argument"))
		}

		// This is a simplified implementation
		// In reality, you'd parse the where conditions properly
		return e.vm.ToValue(map[string]any{
			"error": "findUnique not yet fully implemented in JS engine",
		})
	})

	// Register findMany method using new API
	modelObj.Set("findMany", func(call js.FunctionCall) js.Value {
		// This is a simplified implementation
		return e.vm.ToValue([]any{
			map[string]any{
				"error": "findMany not yet fully implemented in JS engine",
			},
		})
	})

	// Register create method using new API
	modelObj.Set("create", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("create() requires a data argument"))
		}

		// This is a simplified implementation
		return e.vm.ToValue(map[string]any{
			"error": "create not yet fully implemented in JS engine",
		})
	})

	// Register update method using new API
	modelObj.Set("update", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("update() requires where and data arguments"))
		}

		// This is a simplified implementation
		return e.vm.ToValue(map[string]any{
			"error": "update not yet fully implemented in JS engine",
		})
	})

	// Register delete method using new API
	modelObj.Set("delete", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("delete() requires a where argument"))
		}

		// This is a simplified implementation
		return e.vm.ToValue(map[string]any{
			"error": "delete not yet fully implemented in JS engine",
		})
	})

	// Register model in models object
	modelsObj := e.vm.Get("models").(*js.Object)
	modelsObj.Set(name, modelObj)
}

// createQueryBuilderObject is removed as we now use the new API
// JavaScript integration will be implemented separately

func (e *Engine) Execute(script string) (any, error) {
	value, err := e.vm.RunString(script)
	if err != nil {
		return nil, err
	}
	return value.Export(), nil
}

func (e *Engine) GetVM() *js.Runtime {
	return e.vm
}

// LoadPrismaSchema loads and converts a Prisma schema string
func (e *Engine) LoadPrismaSchema(schemaContent string) error {
	lexer := prisma.NewLexer(schemaContent)
	parser := prisma.NewParser(lexer)

	prismaSchema := parser.ParseSchema()
	if len(parser.Errors()) > 0 {
		return fmt.Errorf("parser errors: %v", parser.Errors())
	}

	converter := prisma.NewConverter()
	schemas, err := converter.Convert(prismaSchema)
	if err != nil {
		return fmt.Errorf("failed to convert Prisma schema: %v", err)
	}

	// Register all schemas and create tables
	for _, schema := range schemas {
		if err := e.RegisterSchema(schema); err != nil {
			return fmt.Errorf("failed to register schema %s: %v", schema.Name, err)
		}

		// Create table in database
		ctx := context.Background()
		if err := e.db.CreateModel(ctx, schema.Name); err != nil {
			return fmt.Errorf("failed to create table for schema %s: %v", schema.Name, err)
		}
	}

	return nil
}

// LoadPrismaFile loads and converts a Prisma schema from a file
func (e *Engine) LoadPrismaFile(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	return e.LoadPrismaSchema(string(content))
}
