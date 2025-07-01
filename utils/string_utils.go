package utils

import (
	"strings"
	"unicode"
)

// ToSnakeCase converts a camelCase string to snake_case
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s) + 5) // Pre-allocate some extra space for underscores

	for i, r := range s {
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter (except for the first character)
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToCamelCase converts a snake_case string to camelCase
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	parts := strings.Split(s, "_")
	if len(parts) <= 1 {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))

	// First part stays lowercase
	result.WriteString(parts[0])

	// Capitalize first letter of subsequent parts
	for _, part := range parts[1:] {
		if len(part) > 0 {
			result.WriteRune(unicode.ToUpper(rune(part[0])))
			if len(part) > 1 {
				result.WriteString(part[1:])
			}
		}
	}

	return result.String()
}

// ToPascalCase converts string to PascalCase
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	// If it's snake_case, convert first
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
			}
		}
		return result
	}

	// If it's already camelCase, just capitalize first letter
	if len(s) > 0 {
		return strings.ToUpper(string(s[0])) + s[1:]
	}

	return s
}

// IsSnakeCase checks if a string is in snake_case format
func IsSnakeCase(s string) bool {
	if s == "" {
		return true
	}

	// Check if contains uppercase letters (not snake_case if it does)
	for _, r := range s {
		if unicode.IsUpper(r) {
			return false
		}
	}

	// Check if it contains underscores in valid positions
	// (not at start/end, not consecutive)
	if strings.HasPrefix(s, "_") || strings.HasSuffix(s, "_") {
		return false
	}

	if strings.Contains(s, "__") {
		return false
	}

	return true
}

// IsCamelCase checks if a string is in camelCase format
func IsCamelCase(s string) bool {
	if s == "" {
		return true
	}

	// Must start with lowercase letter
	if unicode.IsUpper(rune(s[0])) {
		return false
	}

	// Must not contain underscores
	if strings.Contains(s, "_") {
		return false
	}

	// Check for valid camelCase pattern - no consecutive uppercase letters
	runes := []rune(s)
	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) && unicode.IsUpper(runes[i-1]) {
			return false // Consecutive uppercase letters not allowed in camelCase
		}
	}

	return true
}

// IsPascalCase checks if a string is in PascalCase format
func IsPascalCase(s string) bool {
	if s == "" {
		return true
	}

	// Must start with uppercase letter
	if !unicode.IsUpper(rune(s[0])) {
		return false
	}

	// Must not contain underscores
	if strings.Contains(s, "_") {
		return false
	}

	return true
}

// Pluralize adds 's' to make a word plural (simple implementation)
// For more complex pluralization, consider using a dedicated library
func Pluralize(word string) string {
	if word == "" {
		return word
	}

	word = strings.ToLower(word)

	// Simple pluralization rules
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		return word + "es"
	}

	if strings.HasSuffix(word, "y") && len(word) > 1 {
		prev := rune(word[len(word)-2])
		if !isVowel(prev) {
			return word[:len(word)-1] + "ies"
		}
	}

	if strings.HasSuffix(word, "f") {
		return word[:len(word)-1] + "ves"
	}

	if strings.HasSuffix(word, "fe") {
		return word[:len(word)-2] + "ves"
	}

	return word + "s"
}

// isVowel checks if a character is a vowel
func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}