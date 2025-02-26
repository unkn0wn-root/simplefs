package utils

import (
	"strings"
	"unicode"
)

// SanitizeFilename removes invalid characters from a filename
func SanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	invalid := []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}

	for _, char := range invalid {
		name = strings.ReplaceAll(name, string(char), "_")
	}

	// Trim spaces
	name = strings.TrimSpace(name)

	// Replace multiple spaces with a single space
	for strings.Contains(name, "  ") {
		name = strings.ReplaceAll(name, "  ", " ")
	}

	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")

	return name
}

// IsValidFilename checks if a filename is valid
func IsValidFilename(name string) bool {
	// Check for empty name
	if name == "" {
		return false
	}

	// Check for invalid characters
	invalid := []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}
	for _, char := range invalid {
		if strings.ContainsRune(name, char) {
			return false
		}
	}

	return true
}

// IsASCII checks if a string contains only ASCII characters
func IsASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
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

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	// Split by underscores or spaces
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == ' '
	})

	// Join parts with proper casing
	var result strings.Builder
	for i, part := range parts {
		if i == 0 {
			// First part is lowercase
			result.WriteString(strings.ToLower(part))
		} else {
			// Other parts start with uppercase
			if len(part) > 0 {
				result.WriteString(strings.ToUpper(part[:1]))
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}

	return result.String()
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	// Split by underscores or spaces
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == ' '
	})

	// Join parts with proper casing
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			result.WriteString(strings.ToLower(part[1:]))
		}
	}

	return result.String()
}

// Truncate truncates a string to the given length, adding ellipsis if needed
func Truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}

	// Truncate to length - 3 to make room for ellipsis
	return s[:length-3] + "..."
}
