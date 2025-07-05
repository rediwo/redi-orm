package base

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// TransactionUtils provides shared utilities for batch operations
type TransactionUtils struct {
	tx         *sql.Tx
	db         types.Database
	driverType string
}

// NewTransactionUtils creates a new TransactionUtils instance
func NewTransactionUtils(tx *sql.Tx, db types.Database, driverType string) *TransactionUtils {
	return &TransactionUtils{
		tx:         tx,
		db:         db,
		driverType: driverType,
	}
}

// CreateMany creates multiple records in a batch
func (tu *TransactionUtils) CreateMany(ctx context.Context, modelName string, data []any) (types.Result, error) {
	if len(data) == 0 {
		return types.Result{}, nil
	}

	// Get schema
	schema, err := tu.db.GetSchema(modelName)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to get schema for model %s: %w", modelName, err)
	}

	// Get table name
	tableName, err := tu.db.ResolveTableName(modelName)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to resolve table name: %w", err)
	}

	// Build bulk insert SQL based on driver type
	sql, args, err := tu.buildBulkInsertSQL(tableName, schema, data)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build bulk insert SQL: %w", err)
	}

	// Execute the query
	result, err := tu.tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute bulk insert: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return types.Result{
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
	}, nil
}

// UpdateMany updates multiple records matching a condition
func (tu *TransactionUtils) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data any) (types.Result, error) {
	// Get table name
	tableName, err := tu.db.ResolveTableName(modelName)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to resolve table name: %w", err)
	}

	// Convert data to map
	dataMap, ok := data.(map[string]any)
	if !ok {
		return types.Result{}, fmt.Errorf("update data must be a map")
	}

	// Build UPDATE SQL
	sql, args, err := tu.buildUpdateSQL(tableName, modelName, dataMap, condition)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build update SQL: %w", err)
	}

	// Execute the query
	result, err := tu.tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute update: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		RowsAffected: rowsAffected,
	}, nil
}

// DeleteMany deletes multiple records matching a condition
func (tu *TransactionUtils) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	// Get table name
	tableName, err := tu.db.ResolveTableName(modelName)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to resolve table name: %w", err)
	}

	// Build DELETE SQL
	sql, args, err := tu.buildDeleteSQL(tableName, modelName, condition)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build delete SQL: %w", err)
	}

	// Execute the query
	result, err := tu.tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute delete: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		RowsAffected: rowsAffected,
	}, nil
}

// buildBulkInsertSQL builds a bulk INSERT statement
func (tu *TransactionUtils) buildBulkInsertSQL(tableName string, schema *schema.Schema, data []any) (string, []any, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data to insert")
	}

	// Get the first record to determine columns
	firstRecord, ok := data[0].(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("data must be array of maps")
	}

	// Build column list and resolve field names to column names
	var columns []string
	var fieldOrder []string
	for fieldName := range firstRecord {
		columnName, err := tu.db.ResolveFieldName(schema.Name, fieldName)
		if err != nil {
			// Skip unknown fields
			continue
		}
		columns = append(columns, tu.quote(columnName))
		fieldOrder = append(fieldOrder, fieldName)
	}

	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no valid columns to insert")
	}

	// Build values placeholders and args
	var valueSets []string
	var args []any
	placeholderIndex := 1

	for _, record := range data {
		recordMap, ok := record.(map[string]any)
		if !ok {
			continue
		}

		var placeholders []string
		for _, fieldName := range fieldOrder {
			value := recordMap[fieldName]
			args = append(args, value)

			placeholder := tu.getPlaceholder(placeholderIndex)
			placeholders = append(placeholders, placeholder)
			placeholderIndex++
		}
		valueSets = append(valueSets, fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))
	}

	// Build the SQL
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		tu.quote(tableName),
		strings.Join(columns, ", "),
		strings.Join(valueSets, ", "))

	return sql, args, nil
}

// buildUpdateSQL builds an UPDATE statement
func (tu *TransactionUtils) buildUpdateSQL(tableName, modelName string, data map[string]any, condition types.Condition) (string, []any, error) {
	var setClauses []string
	var args []any
	placeholderIndex := 1

	// Build SET clauses
	for fieldName, value := range data {
		columnName, err := tu.db.ResolveFieldName(modelName, fieldName)
		if err != nil {
			// Skip unknown fields
			continue
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = %s",
			tu.quote(columnName),
			tu.getPlaceholder(placeholderIndex)))
		args = append(args, value)
		placeholderIndex++
	}

	if len(setClauses) == 0 {
		return "", nil, fmt.Errorf("no valid columns to update")
	}

	// Build WHERE clause
	whereSQL := ""
	if condition != nil {
		conditionContext := &types.ConditionContext{
			TableAlias: "",
			FieldMapper: &fieldMapperWrapper{
				db:        tu.db,
				modelName: modelName,
			},
			ModelName: modelName,
		}
		whereSQL, whereArgs := condition.ToSQL(conditionContext)
		if whereSQL != "" {
			whereSQL = " WHERE " + whereSQL
			args = append(args, whereArgs...)
		}
	}

	// Build the SQL
	sql := fmt.Sprintf("UPDATE %s SET %s%s",
		tu.quote(tableName),
		strings.Join(setClauses, ", "),
		whereSQL)

	return sql, args, nil
}

// buildDeleteSQL builds a DELETE statement
func (tu *TransactionUtils) buildDeleteSQL(tableName, modelName string, condition types.Condition) (string, []any, error) {
	var args []any

	// Build WHERE clause
	whereSQL := ""
	if condition != nil {
		conditionContext := &types.ConditionContext{
			TableAlias: "",
			FieldMapper: &fieldMapperWrapper{
				db:        tu.db,
				modelName: modelName,
			},
			ModelName: modelName,
		}
		whereSQL, whereArgs := condition.ToSQL(conditionContext)
		if whereSQL != "" {
			whereSQL = " WHERE " + whereSQL
			args = append(args, whereArgs...)
		}
	}

	// Build the SQL
	sql := fmt.Sprintf("DELETE FROM %s%s",
		tu.quote(tableName),
		whereSQL)

	return sql, args, nil
}

// quote returns a quoted identifier based on the driver type
func (tu *TransactionUtils) quote(name string) string {
	switch tu.driverType {
	case "mysql":
		return fmt.Sprintf("`%s`", name)
	case "postgresql":
		return fmt.Sprintf(`"%s"`, name)
	default: // sqlite
		return fmt.Sprintf("`%s`", name)
	}
}

// getPlaceholder returns a placeholder based on the driver type
func (tu *TransactionUtils) getPlaceholder(index int) string {
	switch tu.driverType {
	case "postgresql":
		return fmt.Sprintf("$%d", index)
	default: // mysql, sqlite
		return "?"
	}
}

// fieldMapperWrapper wraps database field mapping for condition context
type fieldMapperWrapper struct {
	db        types.Database
	modelName string
}

func (f *fieldMapperWrapper) SchemaToColumn(modelName, fieldName string) (string, error) {
	// Use the provided model name if the condition doesn't specify one
	if modelName == "" {
		modelName = f.modelName
	}
	return f.db.ResolveFieldName(modelName, fieldName)
}

func (f *fieldMapperWrapper) ColumnToSchema(modelName, columnName string) (string, error) {
	// This is a simplified implementation
	// In a full implementation, we'd need reverse mapping
	return columnName, nil
}

func (f *fieldMapperWrapper) SchemaFieldsToColumns(modelName string, fieldNames []string) ([]string, error) {
	if modelName == "" {
		modelName = f.modelName
	}
	return f.db.ResolveFieldNames(modelName, fieldNames)
}

func (f *fieldMapperWrapper) ColumnFieldsToSchema(modelName string, columnNames []string) ([]string, error) {
	// Simplified implementation
	return columnNames, nil
}

func (f *fieldMapperWrapper) MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for fieldName, value := range data {
		columnName, err := f.SchemaToColumn(modelName, fieldName)
		if err != nil {
			// Skip unknown fields
			continue
		}
		result[columnName] = value
	}
	return result, nil
}

func (f *fieldMapperWrapper) MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error) {
	// Simplified implementation
	return data, nil
}

func (f *fieldMapperWrapper) ModelToTable(modelName string) (string, error) {
	return utils.Pluralize(utils.ToSnakeCase(modelName)), nil
}
