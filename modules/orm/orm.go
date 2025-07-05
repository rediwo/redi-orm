package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	js "github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/rediwo/redi-orm/agile"
	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
	"github.com/rediwo/redi/modules"
)

// ModelsModule provides Prisma-like database operations using agile as the backend
type ModelsModule struct {
	loop    *eventloop.EventLoop
	db      types.Database
	schemas map[string]*schema.Schema
}

// Auto-register on import
func init() {
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

// createMethod creates a promise-returning method using agile
func (m *ModelsModule) createMethod(vm *js.Runtime, modelName, methodName string, db types.Database) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		// Validate arguments
		if len(call.Arguments) == 0 {
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

		// Execute async operation using agile
		m.loop.RunOnLoop(func(vm *js.Runtime) {
			// Create agile client
			client := agile.NewClient(db)

			// Build JSON query for agile
			jsonQuery, err := json.Marshal(map[string]any{
				methodName: options,
			})
			if err != nil {
				reject(vm.NewGoError(err))
				return
			}

			// Execute using agile
			result, err := client.Model(modelName).Query(string(jsonQuery))
			if err != nil {
				reject(vm.NewGoError(err))
				return
			}

			// Special handling for count method
			if methodName == "count" {
				// Ensure count returns a number
				if count, ok := result.(int64); ok {
					resolve(vm.ToValue(count))
				} else if count, ok := result.(int); ok {
					resolve(vm.ToValue(count))
				} else {
					resolve(vm.ToValue(result))
				}
			} else {
				resolve(vm.ToValue(result))
			}
		})

		return vm.ToValue(promise)
	}
}

// createFromUriFunction creates a function that returns a database instance
func (m *ModelsModule) createFromUriFunction(vm *js.Runtime) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("fromUri requires a URI string"))
		}

		uri := call.Arguments[0].String()
		dbInstance := vm.NewObject()

		// Create lazy database connection
		var db types.Database
		var connected bool

		// Parse URI to determine database type
		parts := strings.Split(uri, "://")
		if len(parts) < 2 {
			panic(vm.NewTypeError("Invalid database URI format"))
		}

		dbType := parts[0]

		// Create connection function
		dbInstance.Set("connect", func(call js.FunctionCall) js.Value {
			promise, resolve, reject := vm.NewPromise()

			m.loop.RunOnLoop(func(vm *js.Runtime) {
				if connected {
					resolve(js.Undefined())
					return
				}

				// Create database connection
				var err error
				db, err = database.NewFromURI(uri)
				if err != nil {
					reject(vm.NewGoError(err))
					return
				}

				// Connect to database
				ctx := context.Background()
				if err := db.Connect(ctx); err != nil {
					reject(vm.NewGoError(err))
					return
				}

				connected = true
				resolve(js.Undefined())
			})

			return vm.ToValue(promise)
		})

		// Create close function
		dbInstance.Set("close", func(call js.FunctionCall) js.Value {
			promise, resolve, reject := vm.NewPromise()

			m.loop.RunOnLoop(func(vm *js.Runtime) {
				if db != nil {
					if err := db.Close(); err != nil {
						reject(vm.NewGoError(err))
						return
					}
				}
				connected = false
				resolve(js.Undefined())
			})

			return vm.ToValue(promise)
		})

		// Schema management functions
		dbInstance.Set("loadSchema", m.createLoadSchemaFunction(vm, &db, &connected))
		dbInstance.Set("loadSchemaFrom", m.createLoadSchemaFromFunction(vm, &db, &connected))
		dbInstance.Set("syncSchemas", m.createSyncSchemasFunction(vm, &db, &connected))

		// Database operation methods
		dbInstance.Set("ping", m.createPingFunction(vm, &db, &connected))
		dbInstance.Set("createModel", m.createModelFunction(vm, &db, &connected))
		dbInstance.Set("dropModel", m.dropModelFunction(vm, &db, &connected))
		dbInstance.Set("getModels", func(call js.FunctionCall) js.Value {
			if !connected || db == nil {
				return vm.NewArray(0)
			}
			return vm.ToValue(db.GetModels())
		})

		// Raw query functions using agile
		dbInstance.Set("queryRaw", m.createQueryRawFunction(vm, &db, &connected))
		dbInstance.Set("executeRaw", m.createExecuteRawFunction(vm, &db, &connected))

		// Transaction function using agile
		dbInstance.Set("transaction", m.createTransactionFunction(vm, &db, &connected))

		// Models object (will be populated after syncSchemas)
		modelsObj := vm.NewObject()
		dbInstance.Set("models", modelsObj)

		// Store reference for model registration
		dbInstance.Set("__registerModels", func() {
			if db != nil {
				for _, modelName := range db.GetModels() {
					m.registerModel(vm, modelsObj, modelName, db)
				}
			}
		})

		// Additional metadata
		dbInstance.Set("driverType", dbType)

		return dbInstance
	}
}

// Transaction support using agile
func (m *ModelsModule) createTransactionFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("transaction requires a callback function"))
		}

		callback, ok := js.AssertFunction(call.Arguments[0])
		if !ok {
			panic(vm.NewTypeError("transaction requires a callback function"))
		}

		promise, resolve, reject := vm.NewPromise()

		// Execute the transaction in a goroutine
		go func() {
			// Create agile client
			client := agile.NewClient(*db)

			// Execute the transaction
			err := client.Transaction(func(tx *agile.Client) error {
				// We need to execute the callback on the event loop and wait for it
				var txErr error
				done := make(chan bool)

				m.loop.RunOnLoop(func(vm *js.Runtime) {
					// Create transaction context object
					txObj := vm.NewObject()

					// Create models for transaction
					modelsObj := vm.NewObject()
					for _, modelName := range (*db).GetModels() {
						m.registerTransactionModel(vm, modelsObj, modelName, tx)
					}
					txObj.Set("models", modelsObj)

					// Add nested transaction support
					txObj.Set("transaction", m.createNestedTransactionFunction(vm, tx))

					// Call the callback
					result, err := callback(nil, vm.ToValue(txObj))
					if err != nil {
						txErr = err
						close(done)
						return
					}

					// Check if the result is a promise
					if promiseObj := result.ToObject(vm); promiseObj != nil {
						thenMethod := promiseObj.Get("then")
						if thenFunc, ok := js.AssertFunction(thenMethod); ok && !js.IsUndefined(thenMethod) {
							// It's a promise - set up handlers
							catchMethod := promiseObj.Get("catch")
							if _, ok := js.AssertFunction(catchMethod); ok && !js.IsUndefined(catchMethod) {
								// Chain .then().catch() to handle both success and error
								thenResult, _ := thenFunc(promiseObj, vm.ToValue(func(call js.FunctionCall) js.Value {
									// Promise resolved successfully
									close(done)
									return js.Undefined()
								}))

								if thenResultObj := thenResult.ToObject(vm); thenResultObj != nil {
									if catchMethod2 := thenResultObj.Get("catch"); !js.IsUndefined(catchMethod2) {
										if catchFunc2, ok := js.AssertFunction(catchMethod2); ok {
											catchFunc2(thenResultObj, vm.ToValue(func(call js.FunctionCall) js.Value {
												// Promise rejected
												if len(call.Arguments) > 0 {
													errVal := call.Arguments[0]
													if errObj := errVal.ToObject(vm); errObj != nil {
														if msgVal := errObj.Get("message"); !js.IsUndefined(msgVal) {
															txErr = fmt.Errorf("%v", msgVal.String())
														} else {
															txErr = fmt.Errorf("%v", errVal.String())
														}
													} else {
														txErr = fmt.Errorf("%v", errVal.String())
													}
												}
												close(done)
												return js.Undefined()
											}))
										}
									}
								}
							}
							return
						}
					}
					// Not a promise, close immediately
					close(done)
				})

				// Wait for the JavaScript callback to complete
				<-done
				return txErr
			})

			// Resolve or reject the promise based on the transaction result
			m.loop.RunOnLoop(func(vm *js.Runtime) {
				if err != nil {
					reject(vm.NewGoError(err))
				} else {
					resolve(js.Undefined())
				}
			})
		}()

		return vm.ToValue(promise)
	}
}

// registerTransactionModel registers model methods for transaction context
func (m *ModelsModule) registerTransactionModel(vm *js.Runtime, modelsObj *js.Object, modelName string, tx *agile.Client) {
	modelObj := vm.NewObject()

	// Create all methods but using transaction client
	methods := []string{
		"create", "createMany", "createManyAndReturn",
		"findUnique", "findFirst", "findMany", "count", "aggregate", "groupBy",
		"update", "updateMany", "updateManyAndReturn", "upsert",
		"delete", "deleteMany",
	}

	for _, methodName := range methods {
		methodName := methodName // capture for closure
		modelObj.Set(methodName, func(call js.FunctionCall) js.Value {
			// Validate arguments
			if len(call.Arguments) == 0 {
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

			// Build JSON query
			jsonQuery, err := json.Marshal(map[string]any{
				methodName: options,
			})
			if err != nil {
				panic(vm.NewGoError(err))
			}

			// Execute using transaction client
			result, err := tx.Model(modelName).Query(string(jsonQuery))
			if err != nil {
				panic(vm.NewGoError(err))
			}

			// Special handling for count
			if methodName == "count" {
				if count, ok := result.(int64); ok {
					return vm.ToValue(count)
				} else if count, ok := result.(int); ok {
					return vm.ToValue(count)
				}
			}

			return vm.ToValue(result)
		})
	}

	modelsObj.Set(modelName, modelObj)
}

// Raw query functions using agile
func (m *ModelsModule) createQueryRawFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("queryRaw requires SQL string"))
		}

		sql := call.Arguments[0].String()

		// Extract parameters
		var args []any
		for i := 1; i < len(call.Arguments); i++ {
			args = append(args, call.Arguments[i].Export())
		}

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			// Use agile's raw query
			client := agile.NewClient(*db)
			results, err := client.Model("").Raw(sql, args...).Find()
			if err != nil {
				reject(vm.NewGoError(err))
				return
			}

			resolve(vm.ToValue(results))
		})

		return vm.ToValue(promise)
	}
}

func (m *ModelsModule) createExecuteRawFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("executeRaw requires SQL string"))
		}

		sql := call.Arguments[0].String()

		// Extract parameters
		var args []any
		for i := 1; i < len(call.Arguments); i++ {
			args = append(args, call.Arguments[i].Export())
		}

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			// Use agile's raw query
			client := agile.NewClient(*db)
			result, err := client.Model("").Raw(sql, args...).Exec()
			if err != nil {
				reject(vm.NewGoError(err))
				return
			}

			// Return result with rowsAffected
			resultObj := vm.NewObject()
			resultObj.Set("rowsAffected", result.RowsAffected)

			resolve(resultObj)
		})

		return vm.ToValue(promise)
	}
}

// Schema loading functions (keep existing implementation)
func (m *ModelsModule) createLoadSchemaFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("loadSchema requires schema string"))
		}

		schemaContent := call.Arguments[0].String()

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			ctx := context.Background()
			if err := (*db).LoadSchema(ctx, schemaContent); err != nil {
				reject(vm.NewGoError(err))
				return
			}
			resolve(js.Undefined())
		})

		return vm.ToValue(promise)
	}
}

func (m *ModelsModule) createLoadSchemaFromFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("loadSchemaFrom requires filename"))
		}

		filename := call.Arguments[0].String()

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			ctx := context.Background()
			if err := (*db).LoadSchemaFrom(ctx, filename); err != nil {
				reject(vm.NewGoError(err))
				return
			}
			resolve(js.Undefined())
		})

		return vm.ToValue(promise)
	}
}

func (m *ModelsModule) createSyncSchemasFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			ctx := context.Background()
			if err := (*db).SyncSchemas(ctx); err != nil {
				reject(vm.NewGoError(err))
				return
			}

			// Register models after sync
			if registerFn := call.This.ToObject(vm).Get("__registerModels"); registerFn != nil {
				if fn, ok := js.AssertFunction(registerFn); ok {
					fn(nil)
				}
			}

			resolve(js.Undefined())
		})

		return vm.ToValue(promise)
	}
}

// createPingFunction creates the ping method
func (m *ModelsModule) createPingFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			ctx := context.Background()
			if err := (*db).Ping(ctx); err != nil {
				reject(vm.NewGoError(err))
				return
			}
			resolve(js.Undefined())
		})

		return vm.ToValue(promise)
	}
}

// createModelFunction creates the createModel method
func (m *ModelsModule) createModelFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("createModel requires model name"))
		}

		modelName := call.Arguments[0].String()

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			ctx := context.Background()
			if err := (*db).CreateModel(ctx, modelName); err != nil {
				reject(vm.NewGoError(err))
				return
			}
			resolve(js.Undefined())
		})

		return vm.ToValue(promise)
	}
}

// dropModelFunction creates the dropModel method
func (m *ModelsModule) dropModelFunction(vm *js.Runtime, db *types.Database, connected *bool) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if !*connected || *db == nil {
			panic(vm.NewTypeError("Database not connected"))
		}

		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("dropModel requires model name"))
		}

		modelName := call.Arguments[0].String()

		promise, resolve, reject := vm.NewPromise()

		m.loop.RunOnLoop(func(vm *js.Runtime) {
			ctx := context.Background()
			if err := (*db).DropModel(ctx, modelName); err != nil {
				reject(vm.NewGoError(err))
				return
			}
			resolve(js.Undefined())
		})

		return vm.ToValue(promise)
	}
}

// createNestedTransactionFunction creates a transaction method for nested transactions
func (m *ModelsModule) createNestedTransactionFunction(vm *js.Runtime, parentTx *agile.Client) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("transaction requires a callback function"))
		}

		callback, ok := js.AssertFunction(call.Arguments[0])
		if !ok {
			panic(vm.NewTypeError("transaction requires a callback function"))
		}

		promise, resolve, reject := vm.NewPromise()

		// Execute nested transaction in a goroutine
		go func() {
			// Execute nested transaction
			err := parentTx.Transaction(func(nestedTx *agile.Client) error {
				// We need to execute the callback on the event loop and wait for it
				var txErr error
				done := make(chan bool)

				m.loop.RunOnLoop(func(vm *js.Runtime) {
					// Create nested transaction context object
					nestedTxObj := vm.NewObject()

					// Create models for nested transaction
					modelsObj := vm.NewObject()
					// Get models from the parent transaction's database
					modelNames := []string{}
					if db, ok := parentTx.GetDB().(interface{ GetModels() []string }); ok {
						modelNames = db.GetModels()
					}
					for _, modelName := range modelNames {
						m.registerTransactionModel(vm, modelsObj, modelName, nestedTx)
					}
					nestedTxObj.Set("models", modelsObj)

					// Add nested transaction support (for further nesting)
					nestedTxObj.Set("transaction", m.createNestedTransactionFunction(vm, nestedTx))

					// Call the callback with nested transaction context
					result, err := callback(nil, vm.ToValue(nestedTxObj))
					if err != nil {
						txErr = err
						close(done)
						return
					}

					// Check if the result is a promise
					if promiseObj := result.ToObject(vm); promiseObj != nil {
						thenMethod := promiseObj.Get("then")
						if thenFunc, ok := js.AssertFunction(thenMethod); ok && !js.IsUndefined(thenMethod) {
							// It's a promise - handle async
							thenResult, _ := thenFunc(promiseObj, vm.ToValue(func(call js.FunctionCall) js.Value {
								// Promise resolved successfully
								close(done)
								return js.Undefined()
							}))

							if thenResultObj := thenResult.ToObject(vm); thenResultObj != nil {
								if catchMethod := thenResultObj.Get("catch"); !js.IsUndefined(catchMethod) {
									if catchFunc, ok := js.AssertFunction(catchMethod); ok {
										catchFunc(thenResultObj, vm.ToValue(func(call js.FunctionCall) js.Value {
											// Promise rejected
											if len(call.Arguments) > 0 {
												txErr = fmt.Errorf("nested transaction error: %v", call.Arguments[0].String())
											}
											close(done)
											return js.Undefined()
										}))
									}
								}
							}
							return
						}
					}
					// Not a promise, close immediately
					close(done)
				})

				// Wait for the JavaScript callback to complete
				<-done
				return txErr
			})

			// Resolve or reject the promise based on the transaction result
			m.loop.RunOnLoop(func(vm *js.Runtime) {
				if err != nil {
					reject(vm.NewGoError(err))
				} else {
					resolve(js.Undefined())
				}
			})
		}()

		return vm.ToValue(promise)
	}
}

// Export utility functions for backward compatibility
func ConvertValue(value any) any {
	return utils.ToInterface(value)
}
