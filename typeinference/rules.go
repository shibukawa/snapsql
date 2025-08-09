package typeinference

import (
	"strings"

	"github.com/shibukawa/snapsql"
)

// TypeMappings provides database type to normalized type mappings
var TypeMappings = map[string]string{
	// String types
	"VARCHAR":    "string",
	"TEXT":       "string",
	"CHAR":       "string",
	"CHARACTER":  "string",
	"NVARCHAR":   "string",
	"NCHAR":      "string",
	"CLOB":       "string",
	"TINYTEXT":   "string",
	"MEDIUMTEXT": "string",
	"LONGTEXT":   "string",

	// Integer types
	"INTEGER":   "int",
	"INT":       "int",
	"BIGINT":    "int",
	"SMALLINT":  "int",
	"TINYINT":   "int",
	"MEDIUMINT": "int",
	"SERIAL":    "int",
	"BIGSERIAL": "int",

	// Decimal types
	"DECIMAL": "decimal",
	"NUMERIC": "decimal",
	"NUMBER":  "decimal",

	// Float types
	"FLOAT":  "float",
	"DOUBLE": "float",
	"REAL":   "float",

	// Boolean types
	"BOOLEAN": "bool",
	"BOOL":    "bool",
	"BIT":     "bool",

	// Timestamp types
	"TIMESTAMP": "timestamp",
	"DATETIME":  "timestamp",

	// Date types
	"DATE": "date",

	// Time types
	"TIME": "time",

	// JSON types
	"JSON":  "json",
	"JSONB": "json",

	// Binary types map to string for simplicity
	"BLOB":      "string",
	"BINARY":    "string",
	"VARBINARY": "string",
}

// NormalizeType converts a database type to a normalized type
func NormalizeType(dbType string) string {
	// Convert to uppercase for lookup
	upperType := strings.ToUpper(strings.TrimSpace(dbType))

	// Handle parameterized types (e.g., VARCHAR(255), DECIMAL(10,2))
	if idx := strings.Index(upperType, "("); idx != -1 {
		upperType = upperType[:idx]
	}

	if normalized, ok := TypeMappings[upperType]; ok {
		return normalized
	}

	// Default to string for unknown types
	return "string"
}

// TypeInferenceRule defines a function for inferring types from expressions
type TypeInferenceRule func(left, right *TypeInfo, operator string, dialect snapsql.Dialect) *TypeInfo

// OperatorTypeRules maps operators to their type inference rules
var OperatorTypeRules = map[string]TypeInferenceRule{
	// Arithmetic operators
	"+": ArithmeticRule,
	"-": ArithmeticRule,
	"*": ArithmeticRule,
	"/": DivisionRule,
	"%": ArithmeticRule,
	"^": ArithmeticRule,

	// Comparison operators
	"=":              ComparisonRule,
	"<>":             ComparisonRule,
	"!=":             ComparisonRule,
	"<":              ComparisonRule,
	">":              ComparisonRule,
	"<=":             ComparisonRule,
	">=":             ComparisonRule,
	"IS":             ComparisonRule,
	"IS NOT":         ComparisonRule,
	"LIKE":           ComparisonRule,
	"ILIKE":          ComparisonRule,
	"NOT LIKE":       ComparisonRule,
	"NOT ILIKE":      ComparisonRule,
	"SIMILAR TO":     ComparisonRule,
	"NOT SIMILAR TO": ComparisonRule,
	"REGEXP":         ComparisonRule,
	"NOT REGEXP":     ComparisonRule,
	"RLIKE":          ComparisonRule,
	"NOT RLIKE":      ComparisonRule,
	"IN":             ComparisonRule,
	"NOT IN":         ComparisonRule,
	"EXISTS":         ComparisonRule,
	"NOT EXISTS":     ComparisonRule,

	// Logical operators
	"AND": LogicalRule,
	"OR":  LogicalRule,
	"NOT": LogicalRule,

	// String concatenation
	"||":     StringConcatRule,
	"CONCAT": StringConcatRule,

	// JSON operators
	"->":  JSONOperatorRule,
	"->>": JSONOperatorRule,
	"#>":  JSONOperatorRule,
	"#>>": JSONOperatorRule,
	"@>":  JSONOperatorRule,
	"<@":  JSONOperatorRule,
	"?":   JSONOperatorRule,
	"?&":  JSONOperatorRule,
	"?|":  JSONOperatorRule,
	"#-":  JSONOperatorRule,
}

// ArithmeticRule handles arithmetic operations (+, -, *, %, ^)
func ArithmeticRule(left, right *TypeInfo, operator string, dialect snapsql.Dialect) *TypeInfo {
	if left == nil || right == nil {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	// Promote numeric types
	baseType := promoteNumericTypes(left.BaseType, right.BaseType)
	isNullable := left.IsNullable || right.IsNullable

	return &TypeInfo{
		BaseType:   baseType,
		IsNullable: isNullable,
	}
}

// DivisionRule handles division operations (/)
func DivisionRule(left, right *TypeInfo, operator string, dialect snapsql.Dialect) *TypeInfo {
	if left == nil || right == nil {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	// Division always returns float type to handle fractional results
	isNullable := left.IsNullable || right.IsNullable || true // Division by zero possibility

	return &TypeInfo{
		BaseType:   "float",
		IsNullable: isNullable,
	}
}

// ComparisonRule handles comparison operations
func ComparisonRule(left, right *TypeInfo, operator string, dialect snapsql.Dialect) *TypeInfo {
	if left == nil || right == nil {
		return &TypeInfo{BaseType: "bool", IsNullable: true}
	}

	// Comparison operations always return boolean
	isNullable := left.IsNullable || right.IsNullable

	return &TypeInfo{
		BaseType:   "bool",
		IsNullable: isNullable,
	}
}

// LogicalRule handles logical operations (AND, OR, NOT)
func LogicalRule(left, right *TypeInfo, operator string, dialect snapsql.Dialect) *TypeInfo {
	var isNullable bool

	if operator == "NOT" {
		// Unary NOT operation
		if left != nil {
			isNullable = left.IsNullable
		} else {
			isNullable = true
		}
	} else {
		// Binary logical operations
		if left != nil && right != nil {
			isNullable = left.IsNullable || right.IsNullable
		} else {
			isNullable = true
		}
	}

	return &TypeInfo{
		BaseType:   "bool",
		IsNullable: isNullable,
	}
}

// StringConcatRule handles string concatenation
func StringConcatRule(left, right *TypeInfo, operator string, dialect snapsql.Dialect) *TypeInfo {
	if left == nil || right == nil {
		return &TypeInfo{BaseType: "string", IsNullable: true}
	}

	// Check dialect support
	if operator == "||" && !snapsql.Capabilities[dialect][snapsql.FeatureConcatOperator] {
		// Fall back to string type but mark as nullable due to unsupported operator
		return &TypeInfo{BaseType: "string", IsNullable: true}
	}

	if operator == "CONCAT" && !snapsql.Capabilities[dialect][snapsql.FeatureConcatFunction] {
		return &TypeInfo{BaseType: "string", IsNullable: true}
	}

	isNullable := left.IsNullable || right.IsNullable

	return &TypeInfo{
		BaseType:   "string",
		IsNullable: isNullable,
	}
}

// JSONOperatorRule handles JSON operations
func JSONOperatorRule(left, right *TypeInfo, operator string, dialect snapsql.Dialect) *TypeInfo {
	switch operator {
	case "->":
		// JSON -> key: returns JSON value
		return &TypeInfo{BaseType: "json", IsNullable: true}
	case "->>":
		// JSON ->> key: returns text value
		return &TypeInfo{BaseType: "string", IsNullable: true}
	case "#>":
		// JSON #> path: returns JSON value
		return &TypeInfo{BaseType: "json", IsNullable: true}
	case "#>>":
		// JSON #>> path: returns text value
		return &TypeInfo{BaseType: "string", IsNullable: true}
	case "@>", "<@":
		// JSON containment operators: return boolean
		return &TypeInfo{BaseType: "bool", IsNullable: true}
	case "?", "?&", "?|":
		// JSON key/path existence operators: return boolean
		return &TypeInfo{BaseType: "bool", IsNullable: true}
	case "#-":
		// JSON path removal: returns JSON
		return &TypeInfo{BaseType: "json", IsNullable: true}
	default:
		// Unknown JSON operator: return any type
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}
}

// promoteNumericTypes promotes two numeric types to their common type
func promoteNumericTypes(left, right string) string {
	// Priority: string > float > decimal > int > any
	types := []string{left, right}

	for _, priority := range []string{"string", "float", "decimal", "int"} {
		for _, t := range types {
			if t == priority {
				return priority
			}
		}
	}

	return "any"
}

// FunctionInferenceRule defines a function for inferring types from function calls
type FunctionInferenceRule func(args []*TypeInfo, functionName string, castType string, dialect snapsql.Dialect) *TypeInfo

// FunctionTypeRules maps function names to their type inference rules
var FunctionTypeRules = map[string]FunctionInferenceRule{
	// Aggregate functions
	"COUNT": CountRule,
	"SUM":   AggregateRule,
	"AVG":   AverageRule,
	"MIN":   FirstArgRule,
	"MAX":   FirstArgRule,

	// String functions
	"LENGTH":      StringLengthRule,
	"CHAR_LENGTH": StringLengthRule,
	"UPPER":       StringRule,
	"LOWER":       StringRule,
	"TRIM":        StringRule,
	"LTRIM":       StringRule,
	"RTRIM":       StringRule,
	"CONCAT":      StringRule,
	"SUBSTRING":   StringRule,
	"REPLACE":     StringRule,

	// Type conversion functions
	"CAST":     CastRule,
	"COALESCE": CoalesceRule,
	"NULLIF":   FirstArgRule,

	// Window functions
	"ROW_NUMBER":  WindowIntRule,
	"RANK":        WindowIntRule,
	"DENSE_RANK":  WindowIntRule,
	"LAG":         WindowArgRule,
	"LEAD":        WindowArgRule,
	"FIRST_VALUE": WindowArgRule,
	"LAST_VALUE":  WindowArgRule,

	// Date/time functions
	"NOW":               TimestampRule,
	"CURRENT_TIMESTAMP": TimestampRule,
	"CURRENT_DATE":      DateRule,
	"CURRENT_TIME":      TimeRule,
	"EXTRACT":           ExtractRule,
	"DATE_PART":         ExtractRule,

	// Math functions
	"ABS":   FirstArgRule,
	"ROUND": FirstArgRule,
	"FLOOR": FirstArgRule,
	"CEIL":  FirstArgRule,
	"SQRT":  FloatRule,
	"EXP":   FloatRule,
	"LN":    FloatRule,
	"LOG":   FloatRule,
}

// CountRule handles COUNT function
func CountRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	return &TypeInfo{BaseType: "int", IsNullable: false}
}

// AggregateRule handles SUM and similar aggregate functions
func AggregateRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	if len(args) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	// Return the same type as the first argument, but nullable
	return &TypeInfo{
		BaseType:   args[0].BaseType,
		IsNullable: true, // Aggregates can return NULL for empty sets
	}
}

// AverageRule handles AVG function
func AverageRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	return &TypeInfo{BaseType: "float", IsNullable: true}
}

// FirstArgRule returns the type of the first argument
func FirstArgRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	if len(args) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	return args[0]
}

// StringLengthRule handles LENGTH, CHAR_LENGTH functions
func StringLengthRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	isNullable := len(args) > 0 && args[0].IsNullable
	return &TypeInfo{BaseType: "int", IsNullable: isNullable}
}

// StringRule handles string functions that return string
func StringRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	isNullable := len(args) > 0 && args[0].IsNullable
	return &TypeInfo{BaseType: "string", IsNullable: isNullable}
}

// CastRule handles CAST function
func CastRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	normalizedType := NormalizeType(castType)
	isNullable := len(args) > 0 && args[0].IsNullable

	return &TypeInfo{
		BaseType:   normalizedType,
		IsNullable: isNullable,
	}
}

// CoalesceRule handles COALESCE function
func CoalesceRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	if len(args) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	// Find the common type among all arguments
	baseType := "any"
	allNullable := true

	for _, arg := range args {
		if arg.BaseType != "any" {
			if baseType == "any" {
				baseType = arg.BaseType
			} else if baseType != arg.BaseType {
				baseType = promoteNumericTypes(baseType, arg.BaseType)
			}
		}

		if !arg.IsNullable {
			allNullable = false
		}
	}

	return &TypeInfo{
		BaseType:   baseType,
		IsNullable: allNullable,
	}
}

// WindowIntRule handles window functions that return int
func WindowIntRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	return &TypeInfo{BaseType: "int", IsNullable: false}
}

// WindowArgRule handles window functions that return the type of their argument
func WindowArgRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	if len(args) == 0 {
		return &TypeInfo{BaseType: "any", IsNullable: true}
	}

	return args[0]
}

// TimestampRule returns timestamp type
func TimestampRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	return &TypeInfo{BaseType: "timestamp", IsNullable: false}
}

// DateRule returns date type
func DateRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	return &TypeInfo{BaseType: "date", IsNullable: false}
}

// TimeRule returns time type
func TimeRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	return &TypeInfo{BaseType: "time", IsNullable: false}
}

// ExtractRule handles EXTRACT/DATE_PART functions
func ExtractRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	// EXTRACT returns numeric values
	return &TypeInfo{BaseType: "int", IsNullable: false}
}

// FloatRule returns float type
func FloatRule(args []*TypeInfo, functionName, castType string, dialect snapsql.Dialect) *TypeInfo {
	isNullable := len(args) > 0 && args[0].IsNullable
	return &TypeInfo{BaseType: "float", IsNullable: isNullable}
}
