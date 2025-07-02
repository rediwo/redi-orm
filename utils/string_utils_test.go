package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"firstName", "first_name"},
		{"lastName", "last_name"},
		{"createdAt", "created_at"},
		{"updatedAt", "updated_at"},
		{"userID", "user_i_d"},
		{"XMLHttpRequest", "x_m_l_http_request"},
		{"HTTPStatusCode", "h_t_t_p_status_code"},
		{"ID", "i_d"},
		{"id", "id"},
		{"name", "name"},
		{"isActive", "is_active"},
		{"hasPermission", "has_permission"},
		{"CamelCase", "camel_case"},
		{"ALLCAPS", "a_l_l_c_a_p_s"},
		{"mixedCASE", "mixed_c_a_s_e"},
		{"a", "a"},
		{"A", "a"},
		{"aB", "a_b"},
		{"ABC", "a_b_c"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := ToSnakeCase(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"first_name", "firstName"},
		{"last_name", "lastName"},
		{"created_at", "createdAt"},
		{"updated_at", "updatedAt"},
		{"user_id", "userId"},
		{"is_active", "isActive"},
		{"has_permission", "hasPermission"},
		{"id", "id"},
		{"name", "name"},
		{"a", "a"},
		{"a_b", "aB"},
		{"a_b_c", "aBC"},
		{"user_profile_image", "userProfileImage"},
		{"api_key_secret", "apiKeySecret"},
		{"_invalid", "Invalid"},                   // Invalid snake_case, converts underscore-prefixed to PascalCase
		{"invalid_", "invalid"},                   // Invalid snake_case, removes trailing underscore
		{"already_camelCase", "alreadyCamelCase"}, // Mixed format
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := ToCamelCase(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"first_name", "FirstName"},
		{"last_name", "LastName"},
		{"created_at", "CreatedAt"},
		{"user_id", "UserId"},
		{"is_active", "IsActive"},
		{"firstName", "FirstName"},
		{"lastName", "LastName"},
		{"createdAt", "CreatedAt"},
		{"id", "Id"},
		{"name", "Name"},
		{"a", "A"},
		{"user_profile_image", "UserProfileImage"},
		{"api_key_secret", "ApiKeySecret"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := ToPascalCase(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestIsSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{"first_name", true},
		{"last_name", true},
		{"created_at", true},
		{"user_id", true},
		{"is_active", true},
		{"id", true},
		{"name", true},
		{"a", true},
		{"a_b", true},
		{"a_b_c", true},
		{"firstName", false},     // camelCase
		{"FirstName", false},     // PascalCase
		{"ALLCAPS", false},       // All uppercase
		{"_invalid", false},      // Starts with underscore
		{"invalid_", false},      // Ends with underscore
		{"invalid__name", false}, // Double underscore
		{"mixed_Case", false},    // Mixed case
		{"user_ID", false},       // Uppercase in middle
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := IsSnakeCase(test.input)
			assert.Equal(t, test.expected, result, "IsSnakeCase('%s') should return %t", test.input, test.expected)
		})
	}
}

func TestIsCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{"firstName", true},
		{"lastName", true},
		{"createdAt", true},
		{"userId", true},
		{"isActive", true},
		{"id", true},
		{"name", true},
		{"a", true},
		{"aB", true},
		{"aBC", false},        // Consecutive uppercase letters
		{"first_name", false}, // snake_case
		{"FirstName", false},  // PascalCase
		{"ALLCAPS", false},    // All uppercase
		{"mixedCASE", false},  // Mixed case with uppercase start
		{"user_ID", false},    // Contains underscore
		{"_invalid", false},   // Starts with underscore
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := IsCamelCase(test.input)
			assert.Equal(t, test.expected, result, "IsCamelCase('%s') should return %t", test.input, test.expected)
		})
	}
}

func TestIsPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{"FirstName", true},
		{"LastName", true},
		{"CreatedAt", true},
		{"UserId", true},
		{"IsActive", true},
		{"ID", true},
		{"Name", true},
		{"A", true},
		{"AB", true},
		{"ABC", true},
		{"firstName", false},  // camelCase
		{"first_name", false}, // snake_case
		{"ALLCAPS", true},     // All uppercase (technically PascalCase)
		{"mixedCASE", false},  // Mixed case
		{"User_Name", false},  // Contains underscore
		{"_Invalid", false},   // Starts with underscore
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := IsPascalCase(test.input)
			assert.Equal(t, test.expected, result, "IsPascalCase('%s') should return %t", test.input, test.expected)
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	// Test that converting camelCase -> snake_case -> camelCase preserves the original
	camelCaseInputs := []string{
		"firstName",
		"lastName",
		"createdAt",
		"updatedAt",
		"userId",
		"isActive",
		"hasPermission",
		"userProfileImage",
		"apiKeySecret",
	}

	for _, input := range camelCaseInputs {
		t.Run("camelCase_roundtrip_"+input, func(t *testing.T) {
			snakeCase := ToSnakeCase(input)
			backToCamel := ToCamelCase(snakeCase)
			assert.Equal(t, input, backToCamel, "Round trip conversion failed: %s -> %s -> %s", input, snakeCase, backToCamel)
		})
	}

	// Test that converting snake_case -> camelCase -> snake_case preserves the original
	snakeCaseInputs := []string{
		"first_name",
		"last_name",
		"created_at",
		"updated_at",
		"user_id",
		"is_active",
		"has_permission",
		"user_profile_image",
		"api_key_secret",
	}

	for _, input := range snakeCaseInputs {
		t.Run("snake_case_roundtrip_"+input, func(t *testing.T) {
			camelCase := ToCamelCase(input)
			backToSnake := ToSnakeCase(camelCase)
			assert.Equal(t, input, backToSnake, "Round trip conversion failed: %s -> %s -> %s", input, camelCase, backToSnake)
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"user", "users"},
		{"post", "posts"},
		{"category", "categories"},
		{"company", "companies"},
		{"box", "boxes"},
		{"class", "classes"},
		{"buzz", "buzzes"},
		{"church", "churches"},
		{"dish", "dishes"},
		{"leaf", "leaves"},
		{"knife", "knives"},
		{"life", "lives"},
		{"wife", "wives"},
		{"city", "cities"},
		{"party", "parties"},
		{"toy", "toys"},
		{"day", "days"},
		{"person", "persons"}, // Simple implementation, not perfect
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := Pluralize(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty_string", func(t *testing.T) {
		assert.Equal(t, "", ToSnakeCase(""))
		assert.Equal(t, "", ToCamelCase(""))
		assert.Equal(t, "", ToPascalCase(""))
		assert.Equal(t, "", Pluralize(""))
	})

	t.Run("single_character", func(t *testing.T) {
		assert.Equal(t, "a", ToSnakeCase("a"))
		assert.Equal(t, "a", ToSnakeCase("A"))
		assert.Equal(t, "a", ToCamelCase("a"))
		assert.Equal(t, "A", ToPascalCase("a"))
		assert.Equal(t, "as", Pluralize("a"))
	})

	t.Run("numbers", func(t *testing.T) {
		assert.Equal(t, "user123", ToSnakeCase("user123"))
		assert.Equal(t, "user123_name", ToSnakeCase("user123Name"))
		assert.Equal(t, "user123Name", ToCamelCase("user123_name"))
		assert.Equal(t, "User123Name", ToPascalCase("user123_name"))
	})

	t.Run("special_characters", func(t *testing.T) {
		// These are edge cases that might not be perfect but should not crash
		assert.NotPanics(t, func() {
			ToSnakeCase("user-name")
			ToCamelCase("user-name")
			ToPascalCase("user-name")
			Pluralize("user-name")
		})
	})
}
