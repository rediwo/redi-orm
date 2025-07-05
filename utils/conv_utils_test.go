package utils

import (
	"testing"
)

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		// Direct bool
		{"bool true", true, true},
		{"bool false", false, false},

		// Integer types
		{"int 1", int(1), true},
		{"int 0", int(0), false},
		{"int64 1", int64(1), true},
		{"int64 0", int64(0), false},
		{"uint 1", uint(1), true},
		{"uint 0", uint(0), false},

		// Float types
		{"float64 1.0", float64(1.0), true},
		{"float64 0.0", float64(0.0), false},
		{"float32 1.0", float32(1.0), true},
		{"float32 0.0", float32(0.0), false},

		// String types
		{"string true", "true", true},
		{"string TRUE", "TRUE", true},
		{"string 1", "1", true},
		{"string yes", "yes", true},
		{"string false", "false", false},
		{"string FALSE", "FALSE", false},
		{"string 0", "0", false},
		{"string no", "no", false},
		{"string empty", "", false},
		{"string number", "42", true},
		{"string 0.0", "0.0", false},

		// Byte array
		{"byte array true", []byte("true"), true},
		{"byte array 1", []byte("1"), true},

		// Nil and unknown
		{"nil", nil, false},
		{"unknown type", struct{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToBool(tt.input)
			if result != tt.expected {
				t.Errorf("ToBool(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int64
	}{
		// Direct int types
		{"int64", int64(42), 42},
		{"int", int(42), 42},
		{"int32", int32(42), 42},
		{"int16", int16(42), 42},
		{"int8", int8(42), 42},

		// Unsigned int types
		{"uint64", uint64(42), 42},
		{"uint", uint(42), 42},
		{"uint32", uint32(42), 42},
		{"uint16", uint16(42), 42},
		{"uint8", uint8(42), 42},

		// Float types (truncation)
		{"float64", float64(42.7), 42},
		{"float32", float32(42.7), 42},

		// Bool
		{"bool true", true, 1},
		{"bool false", false, 0},

		// String
		{"string int", "42", 42},
		{"string float", "42.7", 42},
		{"string invalid", "invalid", 0},
		{"string empty", "", 0},

		// Byte array
		{"byte array", []byte("42"), 42},

		// Nil and unknown
		{"nil", nil, 0},
		{"unknown type", struct{}{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToInt64(tt.input)
			if result != tt.expected {
				t.Errorf("ToInt64(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		// Direct float types
		{"float64", float64(42.5), 42.5},
		{"float32", float32(42.5), float64(float32(42.5))},

		// Int types
		{"int64", int64(42), 42.0},
		{"int", int(42), 42.0},
		{"uint64", uint64(42), 42.0},

		// Bool
		{"bool true", true, 1.0},
		{"bool false", false, 0.0},

		// String
		{"string float", "42.5", 42.5},
		{"string int", "42", 42.0},
		{"string invalid", "invalid", 0.0},

		// Byte array
		{"byte array", []byte("42.5"), 42.5},

		// Nil and unknown
		{"nil", nil, 0.0},
		{"unknown type", struct{}{}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToFloat64(tt.input)
			if result != tt.expected {
				t.Errorf("ToFloat64(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		// Direct string
		{"string", "hello", "hello"},

		// Byte array
		{"byte array", []byte("hello"), "hello"},

		// Numeric types
		{"int", int(42), "42"},
		{"int64", int64(42), "42"},
		{"uint64", uint64(42), "42"},
		{"float64", float64(42.5), "42.5"},
		{"float32", float32(42.5), "42.5"},

		// Bool
		{"bool true", true, "true"},
		{"bool false", false, "false"},

		// Nil
		{"nil", nil, ""},

		// Unknown type (uses fmt.Sprintf)
		{"struct", struct{ X int }{X: 42}, "{42}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			if result != tt.expected {
				t.Errorf("ToString(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	result := ToInt(int64(42))
	if result != 42 {
		t.Errorf("ToInt(42) = %v, want 42", result)
	}
}

func TestToFloat32(t *testing.T) {
	result := ToFloat32(float64(42.5))
	expected := float32(42.5)
	if result != expected {
		t.Errorf("ToFloat32(42.5) = %v, want %v", result, expected)
	}
}

func TestToInterface(t *testing.T) {
	// Test byte array conversion
	bytes := []byte("hello")
	result := ToInterface(bytes)
	if str, ok := result.(string); !ok || str != "hello" {
		t.Errorf("ToInterface([]byte) should convert to string")
	}

	// Test nil
	if ToInterface(nil) != nil {
		t.Error("ToInterface(nil) should return nil")
	}

	// Test other types (should pass through)
	num := 42
	if ToInterface(num) != num {
		t.Error("ToInterface should pass through non-byte-array types")
	}
}
