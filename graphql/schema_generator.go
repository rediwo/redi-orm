package graphql

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// SchemaGenerator generates GraphQL schema from RediORM schemas
type SchemaGenerator struct {
	schemas       map[string]*schema.Schema
	db            types.Database
	objectTypes   map[string]*graphql.Object
	inputTypes    map[string]*graphql.InputObject
	whereInputs   map[string]*graphql.InputObject
	orderByInputs map[string]*graphql.InputObject
}

// NewSchemaGenerator creates a new schema generator
func NewSchemaGenerator(db types.Database, schemas map[string]*schema.Schema) *SchemaGenerator {
	return &SchemaGenerator{
		schemas:       schemas,
		db:            db,
		objectTypes:   make(map[string]*graphql.Object),
		inputTypes:    make(map[string]*graphql.InputObject),
		whereInputs:   make(map[string]*graphql.InputObject),
		orderByInputs: make(map[string]*graphql.InputObject),
	}
}

// GetObjectTypes returns the object types for debugging
func (g *SchemaGenerator) GetObjectTypes() map[string]*graphql.Object {
	return g.objectTypes
}

// Generate creates the complete GraphQL schema
func (g *SchemaGenerator) Generate() (*graphql.Schema, error) {
	// First pass: create basic object types (without relations)
	for modelName := range g.schemas {
		if err := g.createBasicObjectType(modelName); err != nil {
			return nil, fmt.Errorf("failed to create basic object type for %s: %w", modelName, err)
		}
	}

	// Second pass: manually add relation fields using AddFieldConfig
	for modelName := range g.schemas {
		if err := g.addRelationFields(modelName); err != nil {
			return nil, fmt.Errorf("failed to add relation fields to %s: %w", modelName, err)
		}
	}

	// Create input types
	for modelName := range g.schemas {
		if err := g.createInputTypes(modelName); err != nil {
			return nil, fmt.Errorf("failed to create input types for %s: %w", modelName, err)
		}
	}

	// Create query type
	queryType := g.createQueryType()

	// Create mutation type
	mutationType := g.createMutationType()

	// Create the schema
	schemaConfig := graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	}

	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		return nil, err
	}

	return &schema, nil
}

// createBasicObjectType creates a GraphQL object type with basic fields only
func (g *SchemaGenerator) createBasicObjectType(modelName string) error {
	modelSchema, ok := g.schemas[modelName]
	if !ok {
		return fmt.Errorf("schema not found for model %s", modelName)
	}

	// Create fields for the object type (basic fields only)
	fields := graphql.Fields{}

	// Add fields from schema
	for _, field := range modelSchema.Fields {
		fieldType := MapFieldTypeToGraphQL(field.Type)
		if !field.Nullable {
			fieldType = graphql.NewNonNull(fieldType)
		}

		fields[field.Name] = &graphql.Field{
			Type: fieldType,
			Resolve: func(p graphql.ResolveParams) (any, error) {
				// Field resolver - returns the field value from the source object
				if source, ok := p.Source.(map[string]any); ok {
					return source[p.Info.FieldName], nil
				}
				return nil, nil
			},
		}
	}

	// Create the object type
	objectType := graphql.NewObject(graphql.ObjectConfig{
		Name:        modelName,
		Description: fmt.Sprintf("%s model", modelName),
		Fields:      fields,
	})

	g.objectTypes[modelName] = objectType

	return nil
}

// addRelationFields adds relation fields to existing object types
func (g *SchemaGenerator) addRelationFields(modelName string) error {
	modelSchema, ok := g.schemas[modelName]
	if !ok {
		return fmt.Errorf("schema not found for model %s", modelName)
	}

	// Skip if no relations exist
	if len(modelSchema.Relations) == 0 {
		return nil
	}

	objectType, ok := g.objectTypes[modelName]
	if !ok {
		return fmt.Errorf("object type not found for model %s", modelName)
	}

	// Add relation fields using AddFieldConfig
	for relationName, relation := range modelSchema.Relations {
		relatedType, ok := g.objectTypes[relation.Model]
		if !ok {
			continue // Skip if related model not found
		}

		var fieldType graphql.Type
		if relation.Type == schema.RelationOneToMany || relation.Type == schema.RelationManyToMany {
			fieldType = graphql.NewList(relatedType)
		} else {
			fieldType = relatedType
		}

		// Add the field to the existing object type
		objectType.AddFieldConfig(relationName, &graphql.Field{
			Type:    fieldType,
			Resolve: createRelationResolver(g.db, modelName, relation),
		})
	}

	return nil
}

// createInputTypes creates input types for create and update operations
func (g *SchemaGenerator) createInputTypes(modelName string) error {
	modelSchema, ok := g.schemas[modelName]
	if !ok {
		return fmt.Errorf("schema not found for model %s", modelName)
	}

	// Create fields for create input
	createFields := graphql.InputObjectConfigFieldMap{}
	updateFields := graphql.InputObjectConfigFieldMap{}
	whereFields := graphql.InputObjectConfigFieldMap{}
	orderByFields := graphql.InputObjectConfigFieldMap{}

	for _, field := range modelSchema.Fields {
		fieldType := MapFieldTypeToGraphQL(field.Type)

		// Skip auto-increment fields in create input
		if !field.AutoIncrement {
			if !field.Nullable && field.Default == nil {
				// Required field for create
				createFields[field.Name] = &graphql.InputObjectFieldConfig{
					Type: graphql.NewNonNull(fieldType),
				}
			} else {
				// Optional field for create
				createFields[field.Name] = &graphql.InputObjectFieldConfig{
					Type: fieldType,
				}
			}
		}

		// All fields are optional for update
		updateFields[field.Name] = &graphql.InputObjectFieldConfig{
			Type: fieldType,
		}

		// Where input fields with operators
		whereFields[field.Name] = WhereInputField(field.Type)

		// OrderBy fields
		orderByFields[field.Name] = &graphql.InputObjectFieldConfig{
			Type: OrderByEnum,
		}
	}

	// Add AND/OR to where input
	whereInputName := fmt.Sprintf("%sWhereInput", modelName)
	whereInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   whereInputName,
		Fields: whereFields,
	})

	// Add self-referencing AND/OR after creation
	whereFields["AND"] = &graphql.InputObjectFieldConfig{
		Type: graphql.NewList(whereInput),
	}
	whereFields["OR"] = &graphql.InputObjectFieldConfig{
		Type: graphql.NewList(whereInput),
	}

	// Create input types
	g.inputTypes[modelName+"CreateInput"] = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   fmt.Sprintf("%sCreateInput", modelName),
		Fields: createFields,
	})

	g.inputTypes[modelName+"UpdateInput"] = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   fmt.Sprintf("%sUpdateInput", modelName),
		Fields: updateFields,
	})

	g.whereInputs[modelName] = whereInput

	g.orderByInputs[modelName] = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   fmt.Sprintf("%sOrderByInput", modelName),
		Fields: orderByFields,
	})

	return nil
}

// createQueryType creates the root Query type
func (g *SchemaGenerator) createQueryType() *graphql.Object {
	queryFields := graphql.Fields{}

	for modelName, objectType := range g.objectTypes {
		// findUnique query
		queryFields[fmt.Sprintf("findUnique%s", modelName)] = &graphql.Field{
			Type: objectType,
			Args: graphql.FieldConfigArgument{
				"where": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(g.whereInputs[modelName]),
				},
			},
			Resolve: createFindUniqueResolver(g.db, modelName),
		}

		// findMany query
		queryFields[fmt.Sprintf("findMany%s", modelName)] = &graphql.Field{
			Type: graphql.NewList(objectType),
			Args: graphql.FieldConfigArgument{
				"where": &graphql.ArgumentConfig{
					Type: g.whereInputs[modelName],
				},
				"orderBy": &graphql.ArgumentConfig{
					Type: g.orderByInputs[modelName],
				},
				"limit": &graphql.ArgumentConfig{
					Type: graphql.Int,
				},
				"offset": &graphql.ArgumentConfig{
					Type: graphql.Int,
				},
			},
			Resolve: createFindManyResolver(g.db, modelName),
		}

		// count query
		queryFields[fmt.Sprintf("count%s", modelName)] = &graphql.Field{
			Type: graphql.Int,
			Args: graphql.FieldConfigArgument{
				"where": &graphql.ArgumentConfig{
					Type: g.whereInputs[modelName],
				},
			},
			Resolve: createCountResolver(g.db, modelName),
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: queryFields,
	})
}

// createMutationType creates the root Mutation type
func (g *SchemaGenerator) createMutationType() *graphql.Object {
	mutationFields := graphql.Fields{}

	for modelName, objectType := range g.objectTypes {
		// create mutation
		mutationFields[fmt.Sprintf("create%s", modelName)] = &graphql.Field{
			Type: objectType,
			Args: graphql.FieldConfigArgument{
				"data": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(g.inputTypes[modelName+"CreateInput"]),
				},
			},
			Resolve: createCreateResolver(g.db, modelName),
		}

		// update mutation
		mutationFields[fmt.Sprintf("update%s", modelName)] = &graphql.Field{
			Type: objectType,
			Args: graphql.FieldConfigArgument{
				"where": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(g.whereInputs[modelName]),
				},
				"data": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(g.inputTypes[modelName+"UpdateInput"]),
				},
			},
			Resolve: createUpdateResolver(g.db, modelName),
		}

		// delete mutation
		mutationFields[fmt.Sprintf("delete%s", modelName)] = &graphql.Field{
			Type: objectType,
			Args: graphql.FieldConfigArgument{
				"where": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(g.whereInputs[modelName]),
				},
			},
			Resolve: createDeleteResolver(g.db, modelName),
		}

		// createMany mutation
		mutationFields[fmt.Sprintf("createMany%s", modelName)] = &graphql.Field{
			Type: graphql.NewObject(graphql.ObjectConfig{
				Name: fmt.Sprintf("%sCreateManyResult", modelName),
				Fields: graphql.Fields{
					"count": &graphql.Field{Type: graphql.Int},
				},
			}),
			Args: graphql.FieldConfigArgument{
				"data": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.NewList(g.inputTypes[modelName+"CreateInput"])),
				},
			},
			Resolve: createCreateManyResolver(g.db, modelName),
		}

		// updateMany mutation
		mutationFields[fmt.Sprintf("updateMany%s", modelName)] = &graphql.Field{
			Type: graphql.NewObject(graphql.ObjectConfig{
				Name: fmt.Sprintf("%sUpdateManyResult", modelName),
				Fields: graphql.Fields{
					"count": &graphql.Field{Type: graphql.Int},
				},
			}),
			Args: graphql.FieldConfigArgument{
				"where": &graphql.ArgumentConfig{
					Type: g.whereInputs[modelName],
				},
				"data": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(g.inputTypes[modelName+"UpdateInput"]),
				},
			},
			Resolve: createUpdateManyResolver(g.db, modelName),
		}

		// deleteMany mutation
		mutationFields[fmt.Sprintf("deleteMany%s", modelName)] = &graphql.Field{
			Type: graphql.NewObject(graphql.ObjectConfig{
				Name: fmt.Sprintf("%sDeleteManyResult", modelName),
				Fields: graphql.Fields{
					"count": &graphql.Field{Type: graphql.Int},
				},
			}),
			Args: graphql.FieldConfigArgument{
				"where": &graphql.ArgumentConfig{
					Type: g.whereInputs[modelName],
				},
			},
			Resolve: createDeleteManyResolver(g.db, modelName),
		}
	}

	return graphql.NewObject(graphql.ObjectConfig{
		Name:   "Mutation",
		Fields: mutationFields,
	})
}

// Helper function to convert field name to camelCase
func toCamelCase(s string) string {
	return utils.ToCamelCase(s)
}

// Helper function to convert to plural form
func toPlural(s string) string {
	return utils.Pluralize(s)
}
