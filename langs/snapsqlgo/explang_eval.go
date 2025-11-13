package snapsqlgo

// ExpressionKind represents the type of explang step.
type ExpressionKind int

const (
	ExpressionIdentifier ExpressionKind = iota
	ExpressionMember
	ExpressionIndex
)

// ExpPosition stores source metadata for an expression step.
type ExpPosition struct {
	Line   int
	Column int
	Offset int
	Length int
}

// Expression describes a single access step within an explang expression.
type Expression struct {
	Kind       ExpressionKind
	Identifier string
	Property   string
	Index      int
	Safe       bool
	Position   ExpPosition
}

// ExplangExpression contains the flattened steps for an expression.
type ExplangExpression struct {
	ID          string
	Expressions []Expression
}
