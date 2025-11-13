package pygen

import (
	"strings"
	"text/template"
	"unicode"
)

// getTemplateFuncs returns the function map for the Python template
func getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"snakeCase":         toSnakeCase,
		"indent":            indentString,
		"mutationKindLower": mutationKindToLower,
		"pythonBool":        toPythonBool,
	}
}

// toPythonBool converts Go boolean to Python boolean string
func toPythonBool(b bool) string {
	if b {
		return "True"
	}

	return "False"
}

// mutationKindToLower converts mutation kind to lowercase operation name
// Example: "MutationUpdate" -> "update"
func mutationKindToLower(kind string) string {
	switch kind {
	case "MutationUpdate":
		return "update"
	case "MutationDelete":
		return "delete"
	default:
		return strings.ToLower(kind)
	}
}

// toSnakeCase converts a string to snake_case
// Example: "GetUserById" -> "get_user_by_id"
func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder

	runes := []rune(s)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Add underscore before uppercase letter if:
			// 1. Not the first character
			// 2. Previous character is lowercase or next character is lowercase
			if i > 0 {
				prevIsLower := unicode.IsLower(runes[i-1])

				nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if prevIsLower || nextIsLower {
					result.WriteRune('_')
				}
			}

			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// indentString indents each line of a string by the specified number of spaces
func indentString(spaces int, s string) string {
	if s == "" {
		return ""
	}

	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")

	var result strings.Builder

	for i, line := range lines {
		if i > 0 {
			result.WriteRune('\n')
		}

		if line != "" {
			result.WriteString(indent)
			result.WriteString(line)
		}
	}

	return result.String()
}
