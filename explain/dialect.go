package explain

import "strings"

// supportsJSONPlan reports whether the dialect is expected to return structured (JSON) EXPLAIN output.
func supportsJSONPlan(dialect string) bool {
	switch strings.ToLower(dialect) {
	case "postgres", "postgresql", "pgx", "mysql", "mariadb":
		return true
	default:
		return false
	}
}

// SupportsStructuredPlan reports whether callers should expect structured plan data (JSON).
func SupportsStructuredPlan(dialect string) bool {
	return supportsJSONPlan(dialect)
}

// normalizeDialect lowercases the input dialect string.
func normalizeDialect(d string) string {
	return strings.ToLower(strings.TrimSpace(d))
}
