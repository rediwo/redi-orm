package types

import (
	"encoding/json"
	"strconv"
	"strings"
)

// QueryParams represents common query parameters for data operations
type QueryParams struct {
	// Pagination
	Page  int `json:"page"`
	Limit int `json:"limit"`

	// Filtering
	Where  map[string]any `json:"where"`
	Filter map[string]any `json:"filter"`

	// Sorting
	Sort    []string `json:"sort"`
	OrderBy []string `json:"order_by"`

	// Field selection
	Select []string `json:"select"`
	Fields []string `json:"fields"`

	// Relations
	Include any `json:"include"`

	// Search
	Search string `json:"search"`
	Q      string `json:"q"`
}

// ParseQueryParams parses query parameters from URL values
func ParseQueryParams(params map[string][]string) (*QueryParams, error) {
	qp := &QueryParams{
		Page:  1,
		Limit: 50,
	}

	// Parse pagination
	if page := params["page"]; len(page) > 0 {
		if p, err := strconv.Atoi(page[0]); err == nil {
			qp.Page = p
		}
	}

	if limit := params["limit"]; len(limit) > 0 {
		if l, err := strconv.Atoi(limit[0]); err == nil {
			qp.Limit = l
			if qp.Limit > 1000 {
				qp.Limit = 1000 // Max limit
			}
		}
	}

	// Parse where conditions
	if where := params["where"]; len(where) > 0 {
		if err := json.Unmarshal([]byte(where[0]), &qp.Where); err != nil {
			return nil, err
		}
	}

	// Parse filter conditions (alternative to where)
	qp.Filter = parseFilterParams(params)

	// Parse sorting
	if sort := params["sort"]; len(sort) > 0 {
		qp.Sort = strings.Split(sort[0], ",")
	}
	if orderBy := params["order_by"]; len(orderBy) > 0 {
		qp.OrderBy = strings.Split(orderBy[0], ",")
	}

	// Parse field selection
	if selectFields := params["select"]; len(selectFields) > 0 {
		qp.Select = strings.Split(selectFields[0], ",")
	}
	if fields := params["fields"]; len(fields) > 0 {
		qp.Fields = strings.Split(fields[0], ",")
	}

	// Parse includes
	if include := params["include"]; len(include) > 0 {
		// Try to parse as JSON first
		var includeData any
		if err := json.Unmarshal([]byte(include[0]), &includeData); err == nil {
			qp.Include = includeData
		} else {
			// Fallback to comma-separated string
			qp.Include = strings.Split(include[0], ",")
		}
	}

	// Parse search
	if search := params["search"]; len(search) > 0 {
		qp.Search = search[0]
	}
	if q := params["q"]; len(q) > 0 {
		qp.Q = q[0]
	}

	return qp, nil
}

// parseFilterParams parses filter[field]=value style parameters
func parseFilterParams(params map[string][]string) map[string]any {
	filters := make(map[string]any)

	for key, values := range params {
		if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") {
			// Extract field name from filter[field]
			fieldName := key[7 : len(key)-1]

			// Check for nested operators like filter[age][gt]
			if idx := strings.Index(fieldName, "]["); idx > 0 {
				field := fieldName[:idx]
				operator := fieldName[idx+2:]

				if filters[field] == nil {
					filters[field] = make(map[string]any)
				}

				if fieldMap, ok := filters[field].(map[string]any); ok {
					fieldMap[operator] = parseValue(values[0])
				}
			} else {
				// Simple filter[field]=value
				filters[fieldName] = parseValue(values[0])
			}
		}
	}

	return filters
}

// parseValue attempts to parse a string value to appropriate type
func parseValue(value string) any {
	// Try to parse as number
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// Try to parse as boolean
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}

	// Return as string
	return value
}

// CreateRequest represents a request to create a new record
type CreateRequest struct {
	Data any `json:"data"`
}

// BatchCreateRequest represents a request to create multiple records
type BatchCreateRequest struct {
	Data []any `json:"data"`
}

// UpdateRequest represents a request to update a record
type UpdateRequest struct {
	Data any `json:"data"`
}

// BatchUpdateRequest represents a request to update multiple records
type BatchUpdateRequest struct {
	Where map[string]any `json:"where"`
	Data  any            `json:"data"`
}

// BatchDeleteRequest represents a request to delete multiple records
type BatchDeleteRequest struct {
	Where map[string]any `json:"where"`
}

// AggregateRequest represents a request for aggregation
type AggregateRequest struct {
	GroupBy   []string          `json:"group_by"`
	Aggregate map[string]any    `json:"aggregate"`
	Where     map[string]any    `json:"where"`
	Having    map[string]any    `json:"having"`
	OrderBy   map[string]string `json:"order_by"`
}

// RawQueryRequest represents a raw SQL/MongoDB query request
type RawQueryRequest struct {
	Query      string `json:"query"`
	Parameters []any  `json:"parameters"`
}
