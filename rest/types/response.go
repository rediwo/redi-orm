package types

import "time"

// Response represents the standard API response format
type Response struct {
	Success    bool         `json:"success"`
	Data       any          `json:"data"`
	Pagination *Pagination  `json:"pagination,omitempty"`
	Meta       *Meta        `json:"meta,omitempty"`
	Error      *ErrorDetail `json:"error,omitempty"`
}

// Pagination contains pagination information
type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
	Pages int `json:"pages"`
}

// Meta contains metadata about the request
type Meta struct {
	ExecutionTime string `json:"execution_time"`
	QueryCount    int    `json:"query_count"`
	Timestamp     string `json:"timestamp"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewSuccessResponse creates a successful response
func NewSuccessResponse(data any) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse(data any, page, limit, total int) *Response {
	pages := (total + limit - 1) / limit
	return &Response{
		Success: true,
		Data:    data,
		Pagination: &Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
			Pages: pages,
		},
		Meta: &Meta{
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(code, message string, details ...string) *Response {
	errorDetail := &ErrorDetail{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		errorDetail.Details = details[0]
	}

	return &Response{
		Success: false,
		Error:   errorDetail,
		Meta: &Meta{
			Timestamp: time.Now().Format(time.RFC3339),
		},
	}
}

// WithExecutionTime adds execution time to the response
func (r *Response) WithExecutionTime(duration time.Duration) *Response {
	if r.Meta == nil {
		r.Meta = &Meta{}
	}
	r.Meta.ExecutionTime = duration.String()
	return r
}

// WithQueryCount adds query count to the response
func (r *Response) WithQueryCount(count int) *Response {
	if r.Meta == nil {
		r.Meta = &Meta{}
	}
	r.Meta.QueryCount = count
	return r
}
