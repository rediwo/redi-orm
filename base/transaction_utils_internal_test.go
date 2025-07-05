package base

import (
	"testing"
)

// These tests use internal access to test without mocking complex interfaces

func TestTransactionUtils_quote_internal(t *testing.T) {
	tests := []struct {
		name       string
		driverType string
		input      string
		want       string
	}{
		{
			name:       "mysql quote",
			driverType: "mysql",
			input:      "table_name",
			want:       "`table_name`",
		},
		{
			name:       "postgresql quote",
			driverType: "postgresql",
			input:      "table_name",
			want:       `"table_name"`,
		},
		{
			name:       "sqlite quote",
			driverType: "sqlite",
			input:      "table_name",
			want:       "`table_name`",
		},
		{
			name:       "default quote",
			driverType: "unknown",
			input:      "table_name",
			want:       "`table_name`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tu := &TransactionUtils{driverType: tt.driverType}
			if got := tu.quote(tt.input); got != tt.want {
				t.Errorf("quote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransactionUtils_getPlaceholder_internal(t *testing.T) {
	tests := []struct {
		name       string
		driverType string
		index      int
		want       string
	}{
		{
			name:       "postgresql placeholder",
			driverType: "postgresql",
			index:      1,
			want:       "$1",
		},
		{
			name:       "postgresql placeholder 10",
			driverType: "postgresql",
			index:      10,
			want:       "$10",
		},
		{
			name:       "mysql placeholder",
			driverType: "mysql",
			index:      1,
			want:       "?",
		},
		{
			name:       "sqlite placeholder",
			driverType: "sqlite",
			index:      1,
			want:       "?",
		},
		{
			name:       "default placeholder",
			driverType: "unknown",
			index:      1,
			want:       "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tu := &TransactionUtils{driverType: tt.driverType}
			if got := tu.getPlaceholder(tt.index); got != tt.want {
				t.Errorf("getPlaceholder() = %v, want %v", got, tt.want)
			}
		})
	}
}