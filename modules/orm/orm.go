package orm

import (
	"context"
	"fmt"
	"strings"

	js "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
	"github.com/rediwo/redi/modules"
)

// ModelsModule provides Prisma-like database operations
type ModelsModule struct {
	loop    *eventloop.EventLoop
	db      types.Database // TODO: Currently unused, will be used for transaction support
	schemas map[string]*schema.Schema
}

// Auto-register on import
func init() {
	// Register as 'redi/orm' instead of 'models'
	modules.RegisterModule("redi/orm", initModelsModule)
}

func initModelsModule(config modules.ModuleConfig) error {
	if config.EventLoop == nil || config.VM == nil {
		return fmt.Errorf("EventLoop and VM are required for models module")
	}

	modelsModule := &ModelsModule{
		loop:    config.EventLoop,
		schemas: make(map[string]*schema.Schema),
	}

	// Register as require module with path 'redi/orm'
	config.Registry.RegisterNativeModule("redi/orm", func(vm *js.Runtime, module *js.Object) {
		exports := vm.NewObject()
		// Only export fromUri function - no global models/transaction functions without a database
		exports.Set("fromUri", modelsModule.createFromUriFunction(vm))
		module.Set("exports", exports)
	})

	return nil
}

// registerModel registers all CRUD methods for a model
func (m *ModelsModule) registerModel(vm *js.Runtime, modelsObj *js.Object, modelName string, db types.Database) {
	modelObj := vm.NewObject()

	// Create operations
	modelObj.Set("create", m.createMethod(vm, modelName, "create", db))
	modelObj.Set("createMany", m.createMethod(vm, modelName, "createMany", db))
	modelObj.Set("createManyAndReturn", m.createMethod(vm, modelName, "createManyAndReturn", db))

	// Read operations
	modelObj.Set("findUnique", m.createMethod(vm, modelName, "findUnique", db))
	modelObj.Set("findFirst", m.createMethod(vm, modelName, "findFirst", db))
	modelObj.Set("findMany", m.createMethod(vm, modelName, "findMany", db))
	modelObj.Set("count", m.createMethod(vm, modelName, "count", db))
	modelObj.Set("aggregate", m.createMethod(vm, modelName, "aggregate", db))
	modelObj.Set("groupBy", m.createMethod(vm, modelName, "groupBy", db))

	// Update operations
	modelObj.Set("update", m.createMethod(vm, modelName, "update", db))
	modelObj.Set("updateMany", m.createMethod(vm, modelName, "updateMany", db))
	modelObj.Set("updateManyAndReturn", m.createMethod(vm, modelName, "updateManyAndReturn", db))
	modelObj.Set("upsert", m.createMethod(vm, modelName, "upsert", db))

	// Delete operations
	modelObj.Set("delete", m.createMethod(vm, modelName, "delete", db))
	modelObj.Set("deleteMany", m.createMethod(vm, modelName, "deleteMany", db))

	modelsObj.Set(modelName, modelObj)
}

// createMethod creates a promise-returning method for a model operation
func (m *ModelsModule) createMethod(vm *js.Runtime, modelName, methodName string, db types.Database) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		// Validate arguments based on method
		if len(call.Arguments) == 0 {
			// Some methods don't require arguments
			if methodName != "findMany" && methodName != "count" && methodName != "deleteMany" {
				panic(vm.NewTypeError(fmt.Sprintf("%s.%s() requires options argument", modelName, methodName)))
			}
		}

		var options map[string]any
		if len(call.Arguments) > 0 && !js.IsUndefined(call.Arguments[0]) && !js.IsNull(call.Arguments[0]) {
			exported := call.Arguments[0].Export()
			if optMap, ok := exported.(map[string]any); ok {
				options = optMap
			}
		}

		promise, resolve, reject := vm.NewPromise()

		go func() {
			result, err := m.executeOperation(db, modelName, methodName, options)

			m.loop.RunOnLoop(func(vm *js.Runtime) {
				if err != nil {
					reject(m.createError(vm, err))
				} else {
					resolve(vm.ToValue(result))
				}
			})
		}()

		return vm.ToValue(promise)
	}
}

// createQueryRawMethod creates the queryRaw method for raw SQL queries on a database instance
func (m *ModelsModule) createQueryRawMethod(vm *js.Runtime, db types.Database) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("queryRaw requires SQL query"))
		}

		sql := call.Arguments[0].String()
		var args []any

		// Collect additional arguments
		for i := 1; i < len(call.Arguments); i++ {
			args = append(args, call.Arguments[i].Export())
		}

		promise, resolve, reject := vm.NewPromise()

		go func() {
			// Execute query directly to handle arbitrary result columns
			rows, err := db.Query(sql, args...)
			if err != nil {
				m.loop.RunOnLoop(func(vm *js.Runtime) {
					reject(m.createError(vm, err))
				})
				return
			}
			defer rows.Close()

			// Use utils to scan rows into maps
			results, err := utils.ScanRowsToMaps(rows)
			if err != nil {
				m.loop.RunOnLoop(func(vm *js.Runtime) {
					reject(m.createError(vm, err))
				})
				return
			}

			m.loop.RunOnLoop(func(vm *js.Runtime) {
				resolve(vm.ToValue(results))
			})
		}()

		return vm.ToValue(promise)
	}
}

// createExecuteRawMethod creates the executeRaw method for raw SQL execution on a database instance
func (m *ModelsModule) createExecuteRawMethod(vm *js.Runtime, db types.Database) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("executeRaw requires SQL query"))
		}

		sql := call.Arguments[0].String()
		var args []any

		// Collect additional arguments
		for i := 1; i < len(call.Arguments); i++ {
			args = append(args, call.Arguments[i].Export())
		}

		promise, resolve, reject := vm.NewPromise()

		go func() {
			// Execute directly
			result, err := db.Exec(sql, args...)

			m.loop.RunOnLoop(func(vm *js.Runtime) {
				if err != nil {
					reject(m.createError(vm, err))
				} else {
					rowsAffected, _ := result.RowsAffected()
					resolve(vm.ToValue(map[string]any{
						"rowsAffected": rowsAffected,
					}))
				}
			})
		}()

		return vm.ToValue(promise)
	}
}

// createError creates a JavaScript error object
func (m *ModelsModule) createError(vm *js.Runtime, err error) js.Value {
	errObj := vm.NewObject()
	errObj.Set("message", err.Error())

	// Add error code if it's a known error type
	errMsg := err.Error()
	if strings.Contains(errMsg, "UNIQUE constraint failed") ||
		strings.Contains(errMsg, "duplicate key") {
		errObj.Set("code", "P2002") // Prisma unique constraint violation code
	} else if strings.Contains(errMsg, "not found") {
		errObj.Set("code", "P2025") // Prisma record not found code
	}

	return vm.ToValue(errObj)
}

// createFromUriFunction creates the fromUri function that returns a Database object
func (m *ModelsModule) createFromUriFunction(vm *js.Runtime) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("fromUri requires a URI string"))
		}

		uri := call.Arguments[0].String()
		if uri == "" {
			panic(vm.NewTypeError("URI cannot be empty"))
		}

		// Create database instance
		db, err := database.NewFromURI(uri)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("failed to create database from URI: %w", err)))
		}

		// Create database wrapper object
		dbObj := vm.NewObject()

		// Add connect method
		dbObj.Set("connect", m.createDatabaseMethod(vm, db, "connect"))

		// Add close method
		dbObj.Set("close", m.createDatabaseMethod(vm, db, "close"))

		// Add loadSchema method
		dbObj.Set("loadSchema", m.createDatabaseMethod(vm, db, "loadSchema"))

		// Add loadSchemaFrom method
		dbObj.Set("loadSchemaFrom", m.createDatabaseMethod(vm, db, "loadSchemaFrom"))

		// Add syncSchemas method
		dbObj.Set("syncSchemas", m.createDatabaseMethod(vm, db, "syncSchemas"))

		// Add createModel method
		dbObj.Set("createModel", m.createDatabaseMethod(vm, db, "createModel"))

		// Add dropModel method
		dbObj.Set("dropModel", m.createDatabaseMethod(vm, db, "dropModel"))

		// Add getModels method (synchronous)
		dbObj.Set("getModels", func(call js.FunctionCall) js.Value {
			models := db.GetModels()
			return vm.ToValue(models)
		})

		// Add ping method
		dbObj.Set("ping", m.createDatabaseMethod(vm, db, "ping"))

		// Add raw query methods
		dbObj.Set("queryRaw", m.createQueryRawMethod(vm, db))
		dbObj.Set("executeRaw", m.createExecuteRawMethod(vm, db))

		// Add transaction method
		dbObj.Set("transaction", m.createDatabaseTransactionMethod(vm, db))

		// Create models object that will be populated after syncSchemas
		modelsObj := vm.NewObject()
		dbObj.Set("models", modelsObj)

		// Store reference to update models after sync
		dbObj.Set("_db", db)
		dbObj.Set("_vm", vm)
		dbObj.Set("_modelsObj", modelsObj)
		dbObj.Set("_module", m)

		return vm.ToValue(dbObj)
	}
}

// populateModels populates the models object on the database instance
func (m *ModelsModule) populateModels(vm *js.Runtime, db types.Database, dbObj js.Value) {
	obj := dbObj.ToObject(vm)
	modelsObj := obj.Get("_modelsObj").ToObject(vm)

	// Clear existing models
	for _, key := range modelsObj.Keys() {
		modelsObj.Delete(key)
	}

	// Register each model from database schemas
	for _, modelName := range db.GetModels() {
		m.registerModel(vm, modelsObj, modelName, db)
	}
}

// createDatabaseMethod creates a promise-returning method for database operations
func (m *ModelsModule) createDatabaseMethod(vm *js.Runtime, db types.Database, methodName string) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		promise, resolve, reject := vm.NewPromise()

		go func() {
			ctx := context.Background()
			var err error

			switch methodName {
			case "connect":
				err = db.Connect(ctx)
			case "close":
				err = db.Close()
			case "loadSchema":
				if len(call.Arguments) == 0 {
					err = fmt.Errorf("loadSchema requires schema content")
				} else {
					schemaContent := call.Arguments[0].String()
					err = db.LoadSchema(ctx, schemaContent)
				}
			case "loadSchemaFrom":
				if len(call.Arguments) == 0 {
					err = fmt.Errorf("loadSchemaFrom requires filename")
				} else {
					filename := call.Arguments[0].String()
					err = db.LoadSchemaFrom(ctx, filename)
				}
			case "syncSchemas":
				err = db.SyncSchemas(ctx)
				if err == nil {
					// After successful sync, populate the models object
					m.loop.RunOnLoop(func(vm *js.Runtime) {
						m.populateModels(vm, db, call.This)
					})
				}
			case "createModel":
				if len(call.Arguments) == 0 {
					err = fmt.Errorf("createModel requires model name")
				} else {
					modelName := call.Arguments[0].String()
					err = db.CreateModel(ctx, modelName)
				}
			case "dropModel":
				if len(call.Arguments) == 0 {
					err = fmt.Errorf("dropModel requires model name")
				} else {
					modelName := call.Arguments[0].String()
					err = db.DropModel(ctx, modelName)
				}
			case "ping":
				err = db.Ping(ctx)
			default:
				err = fmt.Errorf("unknown method: %s", methodName)
			}

			m.loop.RunOnLoop(func(vm *js.Runtime) {
				if err != nil {
					reject(m.createError(vm, err))
				} else {
					resolve(js.Undefined())
				}
			})
		}()

		return vm.ToValue(promise)
	}
}
