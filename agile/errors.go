package agile

import "errors"

// Common errors
var (
	// ErrNoOperation is returned when no operation is specified in the query
	ErrNoOperation = errors.New("no operation specified in query")

	// ErrInvalidJSON is returned when the JSON query is invalid
	ErrInvalidJSON = errors.New("invalid JSON query")

	// ErrMissingWhere is returned when a where clause is required but not provided
	ErrMissingWhere = errors.New("operation requires 'where' field")

	// ErrMissingData is returned when a data field is required but not provided
	ErrMissingData = errors.New("operation requires 'data' field")

	// ErrInvalidDataType is returned when the data type is not as expected
	ErrInvalidDataType = errors.New("invalid data type")

	// ErrNotImplemented is returned for operations not yet implemented
	ErrNotImplemented = errors.New("operation not yet implemented")
)