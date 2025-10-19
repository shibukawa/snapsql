package codegenerator

// EvalResultType represents the type of value that a CEL expression evaluates to
type EvalResultType int

const (
	// EvalResultTypeUnknown indicates the type could not be determined
	EvalResultTypeUnknown EvalResultType = iota
	// EvalResultTypeScalar indicates a scalar value (string, number, boolean, etc.)
	EvalResultTypeScalar
	// EvalResultTypeArray indicates an array of scalar values
	EvalResultTypeArray
	// EvalResultTypeObject indicates a single object with fields
	EvalResultTypeObject
	// EvalResultTypeArrayOfObject indicates an array of objects
	EvalResultTypeArrayOfObject
)

// String returns a human-readable representation of the EvalResultType
func (e EvalResultType) String() string {
	switch e {
	case EvalResultTypeScalar:
		return "Scalar"
	case EvalResultTypeArray:
		return "Array"
	case EvalResultTypeObject:
		return "Object"
	case EvalResultTypeArrayOfObject:
		return "ArrayOfObject"
	default:
		return "Unknown"
	}
}

// OpEmitStatic and related constants define the instruction operation types for the intermediate query format.
const (
	// OpEmitStatic is a basic output instruction that outputs static text.
	OpEmitStatic = "EMIT_STATIC" // Output static text
	// OpEmitEval outputs an evaluated expression.
	OpEmitEval = "EMIT_EVAL" // Output evaluated expression

	// OpEmitUnlessBoundary outputs text unless followed by a boundary delimiter.
	OpEmitUnlessBoundary = "EMIT_UNLESS_BOUNDARY" // Output text unless followed by boundary
	// OpBoundary marks a boundary for delimiter removal.
	OpBoundary = "BOUNDARY" // Mark boundary for delimiter removal

	// OpIf starts an if control block.
	OpIf = "IF" // Start of if block
	// OpElseIf represents an else-if branch in control flow.
	OpElseIf = "ELSE_IF" // Else if condition
	// OpElse represents an else branch in control flow.
	OpElse = "ELSE" // Else block
	// OpEnd ends a control block (if, for, loop).
	OpEnd = "END" // End of control block (if, for)

	// OpLoopStart marks the beginning of a loop block.
	OpLoopStart = "LOOP_START" // Start of for loop block
	// OpLoopEnd marks the end of a loop block.
	OpLoopEnd = "LOOP_END" // End of for loop block

	// OpIfSystemLimit conditionally emits content based on presence of system limit.
	OpIfSystemLimit = "IF_SYSTEM_LIMIT" // Conditional based on system limit
	// OpIfSystemOffset conditionally emits content based on presence of system offset.
	OpIfSystemOffset = "IF_SYSTEM_OFFSET" // Conditional based on system offset
	// OpEmitSystemLimit outputs the system limit value.
	OpEmitSystemLimit = "EMIT_SYSTEM_LIMIT" // Output system limit value
	// OpEmitSystemOffset outputs the system offset value.
	OpEmitSystemOffset = "EMIT_SYSTEM_OFFSET" // Output system offset value
	// OpEmitSystemFor outputs the system FOR clause value (not wrapped in IF).
	OpEmitSystemFor = "EMIT_SYSTEM_FOR" // Output system FOR clause value
	// OpEmitSystemValue outputs a specific system field value.
	OpEmitSystemValue = "EMIT_SYSTEM_VALUE" // Output system value for specific field
	// OpEmitForClause outputs the generated row-lock clause string (deprecated, use OpEmitSystemFor).
	OpEmitForClause = "EMIT_FOR_CLAUSE"

	// OpEmitIfDialect outputs a SQL fragment if current dialect matches one of the given dialects.
	OpEmitIfDialect = "EMIT_IF_DIALECT" // Output SQL fragment if current dialect matches
)

// Instruction represents a single instruction in the instruction set
type Instruction struct {
	Op                  string `json:"op"`
	Pos                 string `json:"pos,omitempty"`                   // Position "line:column" from original template
	Value               string `json:"value,omitempty"`                 // For EMIT_STATIC
	Param               string `json:"param,omitempty"`                 // For EMIT_PARAM (deprecated, use ExprIndex)
	ExprIndex           *int   `json:"expr_index,omitempty"`            // Index into expressions array
	Condition           string `json:"condition,omitempty"`             // For IF, ELSE_IF (deprecated, use ExprIndex)
	Variable            string `json:"variable,omitempty"`              // For FOR
	Collection          string `json:"collection,omitempty"`            // For FOR (deprecated, use CollectionExprIndex)
	CollectionExprIndex *int   `json:"collection_expr_index,omitempty"` // Index into expressions array for collection
	EnvIndex            *int   `json:"env_index,omitempty"`             // Environment index for LOOP_START/LOOP_END
	DefaultValue        string `json:"default_value,omitempty"`         // For EMIT_SYSTEM_LIMIT, EMIT_SYSTEM_OFFSET
	SystemField         string `json:"system_field,omitempty"`          // For EMIT_SYSTEM_VALUE - system field name

	// Database dialect fields
	SqlFragment string   `json:"sql_fragment,omitempty"` // For EMIT_IF_DIALECT - SQL fragment to output
	Dialects    []string `json:"dialects,omitempty"`     // For EMIT_IF_DIALECT - target database dialects
}

// CELExpression represents a CEL expression with its metadata
type CELExpression struct {
	ID               string   `json:"id"`
	Expression       string   `json:"expression"`
	EnvironmentIndex int      `json:"environment_index"`
	Position         Position `json:"position,omitempty"`
}

// CELEnvironment represents a CEL environment with variable definitions
type CELEnvironment struct {
	Index               int               `json:"index"`
	AdditionalVariables []CELVariableInfo `json:"additional_variables"`
	Container           string            `json:"container,omitempty"`
	ParentIndex         *int              `json:"parent_index,omitempty"`
}

// CELVariableInfo represents information about a CEL variable
type CELVariableInfo struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value,omitempty"` // Dummy value for type evaluation
}

// Position represents the position of an expression in the source
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}
