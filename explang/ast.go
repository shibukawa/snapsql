package explang

// Position represents the start offset of a node within the original expression.
// Offset is the rune index (0-based), Line/Column are 1-based for error reporting.
type Position struct {
	Offset int
	Line   int
	Column int
	Length int
}

// StepKind indicates what kind of explang step is described.
type StepKind int

const (
	StepIdentifier StepKind = iota
	StepMember
	StepIndex
)

// Step represents a flattened access step such as identifier, member access, or index.
type Step struct {
	Kind       StepKind
	Identifier string
	Property   string
	Index      int
	Safe       bool
	Pos        Position
}
