package agile

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/rediwo/redi-orm/types"
)

// Model represents a database model with agile query capabilities
type Model struct {
	client    *Client
	modelName string
	db        types.Database
}

// Query executes a query with the given JSON string
func (m *Model) Query(jsonQuery string) (any, error) {
	options, err := parseJSON(jsonQuery)
	if err != nil {
		return nil, err
	}

	// Determine the operation from the JSON
	for operation, params := range options {
		paramsMap, ok := params.(map[string]any)
		if !ok {
			// Some operations might not have parameters
			paramsMap = make(map[string]any)
		}

		return executeOperation(m.db, m.modelName, operation, paramsMap, m.client.typeConverter)
	}

	return nil, fmt.Errorf("no operation specified in query")
}

// QueryTyped executes a query and scans the result into the provided destination
func (m *Model) QueryTyped(jsonQuery string, dest any) error {
	result, err := m.Query(jsonQuery)
	if err != nil {
		return err
	}

	// Use JSON marshaling/unmarshaling for type conversion
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, dest); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

// Convenience methods for common operations

// Create creates a new record
func (m *Model) Create(jsonData string) (map[string]any, error) {
	query := fmt.Sprintf(`{"create": %s}`, jsonData)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		return m.client.typeConverter.ConvertResult(m.modelName, resultMap), nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// CreateTyped creates a new record and scans it into dest
func (m *Model) CreateTyped(jsonData string, dest any) error {
	result, err := m.Create(jsonData)
	if err != nil {
		return err
	}

	// Use JSON marshaling for type conversion
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return json.Unmarshal(jsonBytes, dest)
}

// FindMany finds multiple records
func (m *Model) FindMany(jsonQuery string) ([]map[string]any, error) {
	query := fmt.Sprintf(`{"findMany": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if results, ok := result.([]map[string]any); ok {
		// Convert each result
		converted := make([]map[string]any, len(results))
		for i, r := range results {
			converted[i] = m.client.typeConverter.ConvertResult(m.modelName, r)
		}
		return converted, nil
	}

	// Handle case where result is []any
	if results, ok := result.([]any); ok {
		converted := make([]map[string]any, 0, len(results))
		for _, r := range results {
			if resultMap, ok := r.(map[string]any); ok {
				converted = append(converted, m.client.typeConverter.ConvertResult(m.modelName, resultMap))
			}
		}
		return converted, nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// FindManyTyped finds multiple records and scans them into dest
func (m *Model) FindManyTyped(jsonQuery string, dest any) error {
	// Ensure dest is a pointer to a slice
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	results, err := m.FindMany(jsonQuery)
	if err != nil {
		return err
	}

	// Use JSON marshaling for type conversion
	jsonBytes, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	return json.Unmarshal(jsonBytes, dest)
}

// FindUnique finds a single unique record
func (m *Model) FindUnique(jsonQuery string) (map[string]any, error) {
	query := fmt.Sprintf(`{"findUnique": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		return m.client.typeConverter.ConvertResult(m.modelName, resultMap), nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// FindUniqueTyped finds a single unique record and scans it into dest
func (m *Model) FindUniqueTyped(jsonQuery string, dest any) error {
	result, err := m.FindUnique(jsonQuery)
	if err != nil {
		return err
	}

	// Use JSON marshaling for type conversion
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return json.Unmarshal(jsonBytes, dest)
}

// FindFirst finds the first matching record
func (m *Model) FindFirst(jsonQuery string) (map[string]any, error) {
	query := fmt.Sprintf(`{"findFirst": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		return m.client.typeConverter.ConvertResult(m.modelName, resultMap), nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// FindFirstTyped finds the first matching record and scans it into dest
func (m *Model) FindFirstTyped(jsonQuery string, dest any) error {
	result, err := m.FindFirst(jsonQuery)
	if err != nil {
		return err
	}

	// Use JSON marshaling for type conversion
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return json.Unmarshal(jsonBytes, dest)
}

// Update updates records
func (m *Model) Update(jsonQuery string) (map[string]any, error) {
	query := fmt.Sprintf(`{"update": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		return m.client.typeConverter.ConvertResult(m.modelName, resultMap), nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// UpdateMany updates multiple records
func (m *Model) UpdateMany(jsonQuery string) (map[string]any, error) {
	query := fmt.Sprintf(`{"updateMany": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		return resultMap, nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// Delete deletes a record
func (m *Model) Delete(jsonQuery string) (map[string]any, error) {
	query := fmt.Sprintf(`{"delete": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		return m.client.typeConverter.ConvertResult(m.modelName, resultMap), nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// DeleteMany deletes multiple records
func (m *Model) DeleteMany(jsonQuery string) (map[string]any, error) {
	query := fmt.Sprintf(`{"deleteMany": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		return resultMap, nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// Count counts records
func (m *Model) Count(jsonQuery string) (int64, error) {
	query := fmt.Sprintf(`{"count": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return 0, err
	}

	// Handle different return types
	switch v := result.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unexpected count result type: %T", result)
	}
}

// Aggregate performs aggregation queries
func (m *Model) Aggregate(jsonQuery string) (map[string]any, error) {
	query := fmt.Sprintf(`{"aggregate": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	if resultMap, ok := result.(map[string]any); ok {
		// Convert aggregation results
		return m.client.typeConverter.ConvertAggregateResult(resultMap), nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// GroupBy performs group by queries
func (m *Model) GroupBy(jsonQuery string) ([]map[string]any, error) {
	query := fmt.Sprintf(`{"groupBy": %s}`, jsonQuery)
	result, err := m.Query(query)
	if err != nil {
		return nil, err
	}

	// Handle the result as array of maps
	if results, ok := result.([]map[string]any); ok {
		// Convert each result
		converted := make([]map[string]any, len(results))
		for i, r := range results {
			converted[i] = m.client.typeConverter.ConvertAggregateResult(r)
		}
		return converted, nil
	}

	// Handle case where result is []any
	if results, ok := result.([]any); ok {
		converted := make([]map[string]any, 0, len(results))
		for _, r := range results {
			if resultMap, ok := r.(map[string]any); ok {
				converted = append(converted, m.client.typeConverter.ConvertAggregateResult(resultMap))
			}
		}
		return converted, nil
	}

	return nil, fmt.Errorf("unexpected result type: %T", result)
}

// Raw executes a raw SQL query
func (m *Model) Raw(sql string, args ...any) *RawQuery {
	return &RawQuery{
		db:            m.db,
		sql:           sql,
		args:          args,
		typeConverter: m.client.typeConverter,
	}
}

// RawQuery represents a raw SQL query
type RawQuery struct {
	db            types.Database
	sql           string
	args          []any
	typeConverter *TypeConverter
}

// Exec executes the raw query
func (r *RawQuery) Exec() (types.Result, error) {
	return r.db.Raw(r.sql, r.args...).Exec(context.Background())
}

// Find executes the query and returns multiple results
func (r *RawQuery) Find() ([]map[string]any, error) {
	var results []map[string]any
	err := r.db.Raw(r.sql, r.args...).Find(context.Background(), &results)
	if err != nil {
		return nil, err
	}

	// Convert results
	for i, result := range results {
		results[i] = r.typeConverter.ConvertResult("", result)
	}

	return results, nil
}

// FindOne executes the query and returns a single result
func (r *RawQuery) FindOne() (map[string]any, error) {
	var result map[string]any
	err := r.db.Raw(r.sql, r.args...).FindOne(context.Background(), &result)
	if err != nil {
		return nil, err
	}

	return r.typeConverter.ConvertResult("", result), nil
}
