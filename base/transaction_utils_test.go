package base

import (
	"testing"
)

func TestTransactionUtils_quote(t *testing.T) {
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

func TestTransactionUtils_getPlaceholder(t *testing.T) {
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

func TestFieldMapperWrapper_ModelToTable(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		want      string
	}{
		{
			name:      "simple model",
			modelName: "User",
			want:      "users",
		},
		{
			name:      "camelCase model",
			modelName: "UserProfile",
			want:      "user_profiles",
		},
		{
			name:      "already plural",
			modelName: "Settings",
			want:      "settingses", // Pluralize always adds suffix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := &fieldMapperWrapper{}
			got, err := fw.ModelToTable(tt.modelName)
			if err != nil {
				t.Errorf("ModelToTable() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ModelToTable() = %v, want %v", got, tt.want)
			}
		})
	}
}