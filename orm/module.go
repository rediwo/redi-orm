package orm

import (
	"strconv"

	js "github.com/dop251/goja"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi/modules"
)

// ORMModuleInitializer initializes the ORM module for JavaScript access
func ORMModuleInitializer(config modules.ModuleConfig) error {
	// Register the redi/orm module
	config.Registry.RegisterNativeModule("redi/orm", func(vm *js.Runtime, module *js.Object) {
		// Create models object
		models := vm.NewObject()

		// Get all schemas and create model objects
		schemas := GetSchemas()
		if schemas != nil {
			for name, schema := range schemas {
				modelObj := createModelObject(vm, name, schema)
				models.Set(name, modelObj)
			}
		}

		// Export models
		exports := module.Get("exports").(*js.Object)
		exports.Set("models", models)
	})

	return nil
}

// createModelObject creates a JavaScript object representing a database model
func createModelObject(vm *js.Runtime, name string, schema *schema.Schema) *js.Object {
	model := vm.NewObject()

	// Add schema information
	model.Set("name", name)
	model.Set("tableName", schema.TableName)

	// Add CRUD operations
	model.Set("get", func(call js.FunctionCall) js.Value {
		// Get a single record by ID or conditions
		if len(call.Arguments) == 0 {
			// Return all records
			return vm.ToValue(map[string]interface{}{
				"success": false,
				"error":   "get() requires an ID or condition object",
			})
		}

		arg := call.Arguments[0]
		if arg.ExportType().String() == "number" {
			// Get by ID
			id := arg.ToInteger()
			return vm.ToValue(map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"id":     id,
					"model":  name,
					"method": "get_by_id",
				},
			})
		}

		// Get by conditions
		return vm.ToValue(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"conditions": arg.Export(),
				"model":      name,
				"method":     "get_by_conditions",
			},
		})
	})

	model.Set("select", func(call js.FunctionCall) js.Value {
		// Select multiple records with optional conditions
		conditions := make(map[string]interface{})
		if len(call.Arguments) > 0 {
			conditions = call.Arguments[0].Export().(map[string]interface{})
		}

		return vm.ToValue(map[string]interface{}{
			"success": true,
			"data": []map[string]interface{}{
				{
					"conditions": conditions,
					"model":      name,
					"method":     "select",
				},
			},
		})
	})

	model.Set("add", func(call js.FunctionCall) js.Value {
		// Add a new record
		if len(call.Arguments) == 0 {
			return vm.ToValue(map[string]interface{}{
				"success": false,
				"error":   "add() requires data object",
			})
		}

		data := call.Arguments[0].Export()
		return vm.ToValue(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"created": data,
				"model":   name,
				"method":  "add",
				"id":      1, // Mock ID
			},
		})
	})

	model.Set("set", func(call js.FunctionCall) js.Value {
		// Update existing record
		if len(call.Arguments) < 2 {
			return vm.ToValue(map[string]interface{}{
				"success": false,
				"error":   "set() requires ID and data object",
			})
		}

		id := call.Arguments[0].Export()
		data := call.Arguments[1].Export()
		return vm.ToValue(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":      id,
				"updated": data,
				"model":   name,
				"method":  "set",
			},
		})
	})

	model.Set("remove", func(call js.FunctionCall) js.Value {
		// Remove record
		if len(call.Arguments) == 0 {
			return vm.ToValue(map[string]interface{}{
				"success": false,
				"error":   "remove() requires ID or condition object",
			})
		}

		condition := call.Arguments[0].Export()
		return vm.ToValue(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"removed":  condition,
				"model":    name,
				"method":   "remove",
				"affected": 1,
			},
		})
	})

	// Add query builder methods
	model.Set("where", func(call js.FunctionCall) js.Value {
		if len(call.Arguments) == 0 {
			return vm.ToValue(map[string]interface{}{
				"success": false,
				"error":   "where() requires condition object",
			})
		}

		conditions := call.Arguments[0].Export()

		// Return a query builder object
		queryBuilder := vm.NewObject()
		queryBuilder.Set("conditions", conditions)
		queryBuilder.Set("model", name)

		// Add chainable methods
		queryBuilder.Set("orderBy", func(call js.FunctionCall) js.Value {
			if len(call.Arguments) > 0 {
				queryBuilder.Set("orderBy", call.Arguments[0].Export())
			}
			return queryBuilder
		})

		queryBuilder.Set("limit", func(call js.FunctionCall) js.Value {
			if len(call.Arguments) > 0 {
				queryBuilder.Set("limit", call.Arguments[0].Export())
			}
			return queryBuilder
		})

		queryBuilder.Set("offset", func(call js.FunctionCall) js.Value {
			if len(call.Arguments) > 0 {
				queryBuilder.Set("offset", call.Arguments[0].Export())
			}
			return queryBuilder
		})

		// Execute method
		queryBuilder.Set("execute", func(call js.FunctionCall) js.Value {
			return vm.ToValue(map[string]interface{}{
				"success": true,
				"data": []map[string]interface{}{
					{
						"query":  queryBuilder.Export(),
						"model":  name,
						"method": "query_builder",
					},
				},
			})
		})

		return queryBuilder
	})

	// Add field information
	fields := vm.NewArray()
	for i, field := range schema.Fields {
		fieldObj := vm.NewObject()
		fieldObj.Set("name", field.Name)
		fieldObj.Set("type", string(field.Type))
		fieldObj.Set("required", !field.Nullable)
		fieldObj.Set("unique", field.Unique)
		fieldObj.Set("defaultValue", field.Default)
		fields.Set(strconv.Itoa(i), fieldObj)
	}
	model.Set("fields", fields)

	return model
}

// init registers the ORM module
func init() {
	modules.RegisterModule("redi/orm", ORMModuleInitializer)
}
