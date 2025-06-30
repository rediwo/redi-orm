package engine

import (
	"fmt"
	"io/ioutil"
	"sync"

	js "github.com/dop251/goja"
	"github.com/rediwo/redi-orm/models"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

type Engine struct {
	vm      *js.Runtime
	db      types.Database
	schemas map[string]*schema.Schema
	models  map[string]*models.Model
	mu      sync.RWMutex
}

func New(db types.Database) *Engine {
	vm := js.New()
	engine := &Engine{
		vm:      vm,
		db:      db,
		schemas: make(map[string]*schema.Schema),
		models:  make(map[string]*models.Model),
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

	// Create table
	if err := e.db.CreateTable(s); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create model
	model := models.New(s, e.db)

	e.schemas[s.Name] = s
	e.models[s.Name] = model

	// Register model in JavaScript context
	e.registerModelInJS(s.Name, model)

	return nil
}

func (e *Engine) setupGlobalObjects() {
	// Create models object
	modelsObj := e.vm.NewObject()
	e.vm.Set("models", modelsObj)
}

func (e *Engine) registerModelInJS(name string, model *models.Model) {
	modelObj := e.vm.NewObject()

	// Register get method
	modelObj.Set("get", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("get() requires an ID argument"))
		}

		id := call.Arguments[0].Export()
		result, err := model.Get(id)
		if err != nil {
			panic(e.vm.NewGoError(err))
		}

		return e.vm.ToValue(result)
	})

	// Register select method
	modelObj.Set("select", func(call js.FunctionCall) js.Value {
		columns := []string{}
		if len(call.Arguments) > 0 {
			if arr, ok := call.Arguments[0].Export().([]interface{}); ok {
				for _, col := range arr {
					if str, ok := col.(string); ok {
						columns = append(columns, str)
					}
				}
			}
		}

		queryBuilder := model.Select(columns...)
		return e.createQueryBuilderObject(queryBuilder)
	})

	// Register add method
	modelObj.Set("add", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("add() requires a data argument"))
		}

		data, ok := call.Arguments[0].Export().(map[string]interface{})
		if !ok {
			panic(e.vm.NewTypeError("add() argument must be an object"))
		}

		id, err := model.Add(data)
		if err != nil {
			panic(e.vm.NewGoError(err))
		}

		return e.vm.ToValue(id)
	})

	// Register set method (update)
	modelObj.Set("set", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 2 {
			panic(e.vm.NewTypeError("set() requires ID and data arguments"))
		}

		id := call.Arguments[0].Export()
		data, ok := call.Arguments[1].Export().(map[string]interface{})
		if !ok {
			panic(e.vm.NewTypeError("set() second argument must be an object"))
		}

		err := model.Set(id, data)
		if err != nil {
			panic(e.vm.NewGoError(err))
		}

		return js.Undefined()
	})

	// Register remove method
	modelObj.Set("remove", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("remove() requires an ID argument"))
		}

		id := call.Arguments[0].Export()
		err := model.Remove(id)
		if err != nil {
			panic(e.vm.NewGoError(err))
		}

		return js.Undefined()
	})

	// Register model in models object
	modelsObj := e.vm.Get("models").(*js.Object)
	modelsObj.Set(name, modelObj)
}

func (e *Engine) createQueryBuilderObject(qb *models.QueryBuilder) js.Value {
	obj := e.vm.NewObject()

	obj.Set("where", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 3 {
			panic(e.vm.NewTypeError("where() requires field, operator, and value arguments"))
		}

		field := call.Arguments[0].String()
		operator := call.Arguments[1].String()
		value := call.Arguments[2].Export()

		qb.Where(field, operator, value)
		return obj
	})

	obj.Set("orderBy", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("orderBy() requires at least field argument"))
		}

		field := call.Arguments[0].String()
		direction := "ASC"
		if len(call.Arguments) > 1 {
			direction = call.Arguments[1].String()
		}

		qb.OrderBy(field, direction)
		return obj
	})

	obj.Set("limit", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("limit() requires a number argument"))
		}

		limit := call.Arguments[0].ToInteger()
		qb.Limit(int(limit))
		return obj
	})

	obj.Set("offset", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) < 1 {
			panic(e.vm.NewTypeError("offset() requires a number argument"))
		}

		offset := call.Arguments[0].ToInteger()
		qb.Offset(int(offset))
		return obj
	})

	obj.Set("execute", func(call js.FunctionCall) js.Value {
		results, err := qb.Execute()
		if err != nil {
			panic(e.vm.NewGoError(err))
		}
		return e.vm.ToValue(results)
	})

	obj.Set("first", func(call js.FunctionCall) js.Value {
		result, err := qb.First()
		if err != nil {
			panic(e.vm.NewGoError(err))
		}
		return e.vm.ToValue(result)
	})

	obj.Set("count", func(call js.FunctionCall) js.Value {
		count, err := qb.Count()
		if err != nil {
			panic(e.vm.NewGoError(err))
		}
		return e.vm.ToValue(count)
	})

	return obj
}

func (e *Engine) Execute(script string) (interface{}, error) {
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

	// Register all schemas
	for _, schema := range schemas {
		if err := e.RegisterSchema(schema); err != nil {
			return fmt.Errorf("failed to register schema %s: %v", schema.Name, err)
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
