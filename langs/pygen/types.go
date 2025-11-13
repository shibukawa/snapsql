package pygen

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
)

// ConvertToPythonType converts SnapSQL type to Python type hint string
// It handles basic types, arrays, and nullable types
func ConvertToPythonType(snapType string, nullable bool) (string, error) {
	// Handle arrays
	if before, ok := strings.CutSuffix(snapType, "[]"); ok {
		baseType := before

		pyBaseType, err := ConvertToPythonType(baseType, false)
		if err != nil {
			return "", err
		}

		result := fmt.Sprintf("List[%s]", pyBaseType)
		if nullable {
			return fmt.Sprintf("Optional[%s]", result), nil
		}

		return result, nil
	}

	// Normalize temporal aliases (date, time, datetime -> timestamp)
	normalized := normalizeTemporalAlias(strings.ToLower(snapType))

	// Handle basic types
	var pyType string

	switch normalized {
	case "int", "int32", "int64":
		pyType = "int"
	case "string":
		pyType = "str"
	case "bool":
		pyType = "bool"
	case "float", "float32", "float64", "double":
		pyType = "float"
	case "decimal":
		pyType = "Decimal"
	case "timestamp":
		pyType = "datetime"
	case "bytes":
		pyType = "bytes"
	case "any":
		pyType = "Any"
	default:
		return "", NewUnsupportedTypeError(snapType, "type conversion")
	}

	if nullable {
		return fmt.Sprintf("Optional[%s]", pyType), nil
	}

	return pyType, nil
}

// normalizeTemporalAlias returns the canonical representation for temporal snap types
// so that date, time, datetime are treated as timestamp within the code generator.
func normalizeTemporalAlias(typeName string) string {
	lower := strings.ToLower(typeName)

	switch lower {
	case "date", "time", "datetime":
		return "timestamp"
	default:
		return lower
	}
}

// GetRequiredImports returns the set of Python imports needed for the given types
func GetRequiredImports(types []string) []string {
	imports := make(map[string]bool)

	for _, t := range types {
		if strings.Contains(t, "Decimal") {
			imports["from decimal import Decimal"] = true
		}

		if strings.Contains(t, "datetime") {
			imports["from datetime import datetime"] = true
		}

		if strings.Contains(t, "Optional") || strings.Contains(t, "List") || strings.Contains(t, "Any") {
			imports["from typing import Optional, List, Any, Dict, AsyncGenerator"] = true
		}
	}

	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}

	return result
}

// GetPlaceholder returns the SQL placeholder for the given dialect and parameter index
// PostgreSQL uses $1, $2, $3, ... format
// MySQL uses %s format
// SQLite uses ? format
func GetPlaceholder(dialect snapsql.Dialect, index int) string {
	switch dialect {
	case snapsql.DialectPostgres:
		// PostgreSQL uses $1, $2, $3, ...
		return fmt.Sprintf("$%d", index)
	case snapsql.DialectMySQL:
		// MySQL uses %s for all parameters
		return "%s"
	case snapsql.DialectSQLite:
		// SQLite uses ? for all parameters
		return "?"
	default:
		panic(fmt.Sprintf("unsupported dialect for placeholder: %s", dialect))
	}
}

// GetPlaceholderList returns a list of placeholders for the given dialect and count
func GetPlaceholderList(dialect snapsql.Dialect, count int) []string {
	placeholders := make([]string, count)
	for i := range count {
		placeholders[i] = GetPlaceholder(dialect, i+1)
	}

	return placeholders
}
