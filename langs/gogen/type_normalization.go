package gogen

import "strings"

// normalizeTemporalAlias returns the canonical representation for temporal snap types
// so that date, time, datetime are treated as timestamp within the code generator.
func normalizeTemporalAlias(typeName string) string {
	lower := strings.ToLower(typeName)

	// Leave package-qualified types (e.g., time.Time) untouched.
	if strings.Contains(lower, ".") {
		return lower
	}

	switch lower {
	case "timestamp", "datetime", "date", "time":
		return "timestamp"
	default:
		return lower
	}
}
