package parserstep4

// FixedFunctionReturnTypes is a map of well-known SQL functions to their fixed return types.
// The return type is not dependent on arguments. This is used for type inference in SELECT clause analysis.
// Note: All keys are lower-case. Types are SQL standard or common DBMS types.
var fixedFunctionReturnTypes = map[string]string{
	// Aggregates
	"count":     "integer",
	"count_big": "bigint",
	"sum":       "numeric", // depends on arg, but numeric is safe default
	"avg":       "numeric",
	//"min":       "same_as_arg",
	//"max":       "same_as_arg",
	//"array_agg": "array",

	// String
	"length":       "integer",
	"char_length":  "integer",
	"octet_length": "integer",
	"lower":        "text",
	"upper":        "text",
	"trim":         "text",
	"concat":       "text",

	// Date/time
	"date_part":         "numeric", // PostgreSQL: always numeric, regardless of unit
	"extract":           "numeric", // synonym for date_part
	"now":               "timestamp",
	"current_date":      "date",
	"current_time":      "time",
	"current_timestamp": "timestamp",

	// Type conversion
	"cast": "depends_on_type",
	// PostgreSQL specific
	"to_char":      "text",
	"to_date":      "date",
	"to_timestamp": "timestamp",
	"to_number":    "numeric",

	// Boolean
	"exists":   "boolean",
	"isnull":   "boolean",
	"notnull":  "boolean",
	"isfinite": "boolean",

	// Type check
	"pg_typeof": "text",

	// JSON
	"jsonb_array_length": "integer",
	"jsonb_typeof":       "text",

	// Math
	//"abs":     "same_as_arg",
	"ceil":    "numeric",
	"ceiling": "numeric",
	"floor":   "numeric",

	// Misc
	"row_number": "integer",
	"rank":       "integer",

	"dense_rank": "integer",
}
