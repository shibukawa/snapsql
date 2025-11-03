package explain

import (
	"strings"

	"github.com/shibukawa/snapsql"
)

// supportsJSONPlan reports whether the dialect is expected to return structured (JSON) EXPLAIN output.
func supportsJSONPlan(dialect snapsql.Dialect) bool {
	switch strings.ToLower(string(dialect)) {
	case "postgres", "postgresql", "pgx", "mysql", "mariadb":
		return true
	default:
		return false
	}
}

// SupportsStructuredPlan reports whether callers should expect structured plan data (JSON).
func SupportsStructuredPlan(dialect snapsql.Dialect) bool {
	return supportsJSONPlan(dialect)
}

// normalizeDialect lowercases the input dialect string.
func normalizeDialect(d snapsql.Dialect) string {
	return strings.ToLower(strings.TrimSpace(string(d)))
}
