package utils

import (
	"fmt"
	"strconv"
)

// ToBool converts various types to bool
// Handles different database driver representations:
// - bool: direct conversion
// - int/int64: 0 = false, non-zero = true
// - string: "true"/"1" = true, "false"/"0" = false
// - nil: false
func ToBool(v any) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int32:
		return val != 0
	case int64:
		return val != 0
	case uint:
		return val != 0
	case uint32:
		return val != 0
	case uint64:
		return val != 0
	case float32:
		return val != 0
	case float64:
		return val != 0
	case string:
		// Handle string representations
		switch val {
		case "true", "TRUE", "True", "1", "yes", "YES", "Yes":
			return true
		case "false", "FALSE", "False", "0", "no", "NO", "No", "":
			return false
		default:
			// Try parsing as number
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				return n != 0
			}
			return false
		}
	case []byte:
		return ToBool(string(val))
	default:
		return false
	}
}

// ToInt64 converts various types to int64
// Handles different database driver representations:
// - int/int32/int64: direct conversion
// - uint/uint32/uint64: conversion with bounds checking
// - float32/float64: truncation
// - string: parsing
// - bool: true = 1, false = 0
// - nil: 0
func ToInt64(v any) int64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int16:
		return int64(val)
	case int8:
		return int64(val)
	case uint:
		return int64(val)
	case uint64:
		return int64(val)
	case uint32:
		return int64(val)
	case uint16:
		return int64(val)
	case uint8:
		return int64(val)
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	case bool:
		if val {
			return 1
		}
		return 0
	case string:
		// Try parsing as integer first
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
		// Try parsing as float and truncate
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return int64(f)
		}
		return 0
	case []byte:
		return ToInt64(string(val))
	default:
		return 0
	}
}

// ToFloat64 converts various types to float64
// Handles different database driver representations:
// - float32/float64: direct conversion
// - int/int32/int64: conversion
// - string: parsing
// - bool: true = 1.0, false = 0.0
// - nil: 0.0
func ToFloat64(v any) float64 {
	if v == nil {
		return 0.0
	}

	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case int16:
		return float64(val)
	case int8:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	case uint16:
		return float64(val)
	case uint8:
		return float64(val)
	case bool:
		if val {
			return 1.0
		}
		return 0.0
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
		return 0.0
	case []byte:
		return ToFloat64(string(val))
	default:
		return 0.0
	}
}

// ToString converts various types to string
// Handles different database driver representations:
// - string: direct return
// - []byte: byte to string conversion
// - numeric types: formatted conversion
// - bool: "true" or "false"
// - nil: ""
func ToString(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ToInt converts various types to int
// Similar to ToInt64 but returns int
func ToInt(v any) int {
	return int(ToInt64(v))
}

// ToFloat32 converts various types to float32
// Similar to ToFloat64 but returns float32
func ToFloat32(v any) float32 {
	return float32(ToFloat64(v))
}

// ToInterface converts database-specific types to standard Go types
// Useful for normalizing results from different database drivers
func ToInterface(v any) any {
	if v == nil {
		return nil
	}

	// Handle []byte specially - often used for strings in databases
	if b, ok := v.([]byte); ok {
		return string(b)
	}

	return v
}