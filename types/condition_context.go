package types

// ConditionContext provides context information for SQL generation
type ConditionContext struct {
	FieldMapper     FieldMapper
	ModelName       string
	TableAlias      string
	JoinedTables    map[string]JoinInfo // For complex queries with joins
	QuoteIdentifier func(string) string // Function to quote identifiers
}

// JoinInfo contains information about a joined table
type JoinInfo struct {
	ModelName  string
	TableAlias string
}

// NewConditionContext creates a new condition context
func NewConditionContext(fieldMapper FieldMapper, modelName string, tableAlias string) *ConditionContext {
	return &ConditionContext{
		FieldMapper:  fieldMapper,
		ModelName:    modelName,
		TableAlias:   tableAlias,
		JoinedTables: make(map[string]JoinInfo),
	}
}

// MapFieldToColumn maps a field name to its column name with proper table alias
func (ctx *ConditionContext) MapFieldToColumn(fieldName string) (string, error) {
	if ctx.FieldMapper == nil {
		// No mapper, return field name as-is
		return fieldName, nil
	}

	// Map field name to column name
	columnName, err := ctx.FieldMapper.SchemaToColumn(ctx.ModelName, fieldName)
	if err != nil {
		// If mapping fails, use original field name
		columnName = fieldName
	}

	// Quote the column name if quote function is available
	if ctx.QuoteIdentifier != nil {
		columnName = ctx.QuoteIdentifier(columnName)
	}

	// Add table alias if present
	if ctx.TableAlias != "" {
		if ctx.QuoteIdentifier != nil {
			return ctx.QuoteIdentifier(ctx.TableAlias) + "." + columnName, nil
		}
		return ctx.TableAlias + "." + columnName, nil
	}

	return columnName, nil
}
