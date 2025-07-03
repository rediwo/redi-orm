package orm

import (
	"context"
	"fmt"

	js "github.com/dop251/goja"
	"github.com/rediwo/redi-orm/types"
)

// createDatabaseTransactionMethod creates the db.transaction method
func (m *ModelsModule) createDatabaseTransactionMethod(vm *js.Runtime, db types.Database) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		if len(call.Arguments) == 0 {
			panic(vm.NewTypeError("transaction requires a callback function"))
		}

		// Validate that the first argument is a function
		callbackValue := call.Arguments[0]
		if js.IsUndefined(callbackValue) || js.IsNull(callbackValue) {
			panic(vm.NewTypeError("transaction requires a function as first argument"))
		}

		// Check if it's callable
		callbackObj, ok := callbackValue.(*js.Object)
		if !ok || callbackObj.Get("call") == nil {
			panic(vm.NewTypeError("transaction requires a function as first argument"))
		}

		promise, resolve, reject := vm.NewPromise()

		go func() {
			ctx := context.Background()
			err := db.Transaction(ctx, func(tx types.Transaction) error {
				// Create a channel to communicate with event loop
				resultChan := make(chan error, 1)

				// Run callback in event loop
				m.loop.RunOnLoop(func(vm *js.Runtime) {
					defer func() {
						if r := recover(); r != nil {
							resultChan <- fmt.Errorf("transaction callback error: %v", r)
						}
					}()

					// Create transaction context object
					txObj := vm.NewObject()
					txModelsObj := vm.NewObject()

					// Create a transaction-aware models module
					txModule := &TransactionModelsModule{
						ModelsModule: m,
						tx:           tx,
						vm:           vm,
						db:           db,
					}

					// Register all models for transaction
					for _, modelName := range db.GetModels() {
						txModule.registerTransactionModel(txModelsObj, modelName)
					}

					txObj.Set("models", txModelsObj)

					// Call the callback with transaction object
					// Store the callback and tx object in VM globals temporarily
					vm.Set("__txCallback", callbackObj)
					vm.Set("__txObject", txObj)

					// Execute the callback
					result, err := vm.RunString("__txCallback(__txObject)")
					if err != nil {
						resultChan <- fmt.Errorf("callback execution error: %v", err)
						return
					}

					// Clean up
					vm.Set("__txCallback", js.Undefined())
					vm.Set("__txObject", js.Undefined())

					// Handle the result
					if result != nil {
						// Check if it's a promise
						if promiseObj, ok := result.(*js.Object); ok && promiseObj.Get("then") != nil {
							// It's a promise - wait for it
							thenFunc := promiseObj.Get("then")
							catchFunc := promiseObj.Get("catch")

							if thenFunc != nil {
								var catchObj *js.Object
								if catchFunc != nil {
									catchObj = catchFunc.(*js.Object)
								}

								// Create handlers for promise resolution
								thenHandler := func(call js.FunctionCall) js.Value {
									resultChan <- nil
									return js.Undefined()
								}

								catchHandler := func(call js.FunctionCall) js.Value {
									if len(call.Arguments) > 0 {
										errMsg := call.Arguments[0].String()
										resultChan <- fmt.Errorf("%s", errMsg)
									} else {
										resultChan <- fmt.Errorf("promise rejected")
									}
									return js.Undefined()
								}

								// Attach handlers using VM execution
								vm.Set("__promise", promiseObj)
								vm.Set("__thenHandler", vm.ToValue(thenHandler))
								vm.Set("__catchHandler", vm.ToValue(catchHandler))

								vm.RunString("__promise.then(__thenHandler)")
								if catchObj != nil {
									vm.RunString("__promise.catch(__catchHandler)")
								}

								// Clean up
								vm.Set("__promise", js.Undefined())
								vm.Set("__thenHandler", js.Undefined())
								vm.Set("__catchHandler", js.Undefined())
							} else {
								// Not a proper promise
								resultChan <- nil
							}
						} else {
							// Synchronous result
							resultChan <- nil
						}
					} else {
						resultChan <- nil
					}
				})

				// Wait for callback to complete
				return <-resultChan
			})

			// Resolve or reject the main promise
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

// TransactionModelsModule wraps ModelsModule for transaction context
type TransactionModelsModule struct {
	*ModelsModule
	tx types.Transaction
	vm *js.Runtime
	db types.Database
}

// registerTransactionModel registers a model with transaction context
func (t *TransactionModelsModule) registerTransactionModel(modelsObj *js.Object, modelName string) {
	modelObj := t.vm.NewObject()

	// Create operations - all use the transaction
	modelObj.Set("create", t.createTransactionMethod(modelName, "create"))
	modelObj.Set("createMany", t.createTransactionMethod(modelName, "createMany"))
	modelObj.Set("createManyAndReturn", t.createTransactionMethod(modelName, "createManyAndReturn"))

	modelObj.Set("findUnique", t.createTransactionMethod(modelName, "findUnique"))
	modelObj.Set("findFirst", t.createTransactionMethod(modelName, "findFirst"))
	modelObj.Set("findMany", t.createTransactionMethod(modelName, "findMany"))
	modelObj.Set("count", t.createTransactionMethod(modelName, "count"))
	modelObj.Set("aggregate", t.createTransactionMethod(modelName, "aggregate"))
	modelObj.Set("groupBy", t.createTransactionMethod(modelName, "groupBy"))

	modelObj.Set("update", t.createTransactionMethod(modelName, "update"))
	modelObj.Set("updateMany", t.createTransactionMethod(modelName, "updateMany"))
	modelObj.Set("updateManyAndReturn", t.createTransactionMethod(modelName, "updateManyAndReturn"))
	modelObj.Set("upsert", t.createTransactionMethod(modelName, "upsert"))

	modelObj.Set("delete", t.createTransactionMethod(modelName, "delete"))
	modelObj.Set("deleteMany", t.createTransactionMethod(modelName, "deleteMany"))

	modelsObj.Set(modelName, modelObj)
}

// createTransactionMethod creates a method that uses the transaction
func (t *TransactionModelsModule) createTransactionMethod(modelName, methodName string) func(call js.FunctionCall) js.Value {
	return func(call js.FunctionCall) js.Value {
		// Validate arguments based on method
		if len(call.Arguments) == 0 {
			// Some methods don't require arguments
			if methodName != "findMany" && methodName != "count" && methodName != "deleteMany" {
				panic(t.vm.NewTypeError(fmt.Sprintf("%s.%s() requires options argument", modelName, methodName)))
			}
		}

		var options map[string]any
		if len(call.Arguments) > 0 && !js.IsUndefined(call.Arguments[0]) && !js.IsNull(call.Arguments[0]) {
			exported := call.Arguments[0].Export()
			if optMap, ok := exported.(map[string]any); ok {
				options = optMap
			}
		}

		promise, resolve, reject := t.vm.NewPromise()

		go func() {
			// Execute operation using transaction
			result, err := t.executeTransactionOperation(modelName, methodName, options)

			t.loop.RunOnLoop(func(vm *js.Runtime) {
				if err != nil {
					reject(t.createError(vm, err))
				} else {
					resolve(vm.ToValue(result))
				}
			})
		}()

		return t.vm.ToValue(promise)
	}
}

// executeTransactionOperation executes an operation within a transaction
func (t *TransactionModelsModule) executeTransactionOperation(modelName, methodName string, options map[string]any) (any, error) {
	ctx := context.Background()
	model := t.tx.Model(modelName)

	// Reuse the same operation logic but with transaction model
	switch methodName {
	case "create":
		return t.executeCreate(ctx, model, options)
	case "createMany":
		return t.executeCreateMany(ctx, model, modelName, options)
	case "createManyAndReturn":
		return t.executeCreateManyAndReturn(ctx, model, modelName, options)

	case "findUnique":
		return t.executeFindUnique(ctx, model, options)
	case "findFirst":
		return t.executeFindFirst(ctx, model, options)
	case "findMany":
		return t.executeFindMany(ctx, model, options)
	case "count":
		return t.executeCount(ctx, model, options)
	case "aggregate":
		return t.executeAggregate(ctx, model, options)
	case "groupBy":
		return t.executeGroupBy(ctx, model, modelName, options, t.db)

	case "update":
		return t.executeUpdate(ctx, model, options)
	case "updateMany":
		return t.executeUpdateMany(ctx, model, modelName, options)
	case "updateManyAndReturn":
		return t.executeUpdateManyAndReturn(ctx, model, modelName, options)
	case "upsert":
		return t.executeUpsert(ctx, model, options)

	case "delete":
		return t.executeDelete(ctx, model, options)
	case "deleteMany":
		return t.executeDeleteMany(ctx, model, modelName, options)

	default:
		return nil, fmt.Errorf("unknown method: %s", methodName)
	}
}
