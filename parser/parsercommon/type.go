package parsercommon

import "github.com/shibukawa/snapsql/tokenizer"

// NodeType represents the type of AST node
// This is used for type discrimination and debugging.
type NodeType int

const (
	// SQL statement structures
	UNKNOWN NodeType = iota
	SUBQUERY_STATEMENT

	// Select statement
	SELECT_STATEMENT
	SELECT_CLAUSE
	FROM_CLAUSE
	WHERE_CLAUSE
	ORDER_BY_CLAUSE
	GROUP_BY_CLAUSE
	HAVING_CLAUSE
	LIMIT_CLAUSE
	OFFSET_CLAUSE
	WITH_CLAUSE
	FOR_CLAUSE
	CTE_DEFINITION

	// Insert statement
	INSERT_INTO_STATEMENT
	INSERT_INTO_CLAUSE
	VALUES_CLAUSE
	ON_CONFLICT_CLAUSE

	// Update statement
	UPDATE_STATEMENT
	UPDATE_CLAUSE
	SET_CLAUSE

	// Delete statement
	DELETE_FROM_CLAUSE
	DELETE_FROM_STATEMENT

	// SnapSQL extensions
	TEMPLATE_IF_BLOCK
	TEMPLATE_ELSEIF_BLOCK
	TEMPLATE_ELSE_BLOCK
	TEMPLATE_FOR_BLOCK
	VARIABLE_SUBSTITUTION
	DEFERRED_VARIABLE_SUBSTITUTION
	BULK_VARIABLE_SUBSTITUTION
	ENVIRONMENT_REFERENCE
	IMPLICIT_CONDITIONAL

	// Expressions and literals
	IDENTIFIER
	LITERAL
	EXPRESSION

	// Others
	OTHER_NODE
	RETURNING_CLAUSE

	// Column reference
	COLUMN_REFERENCE
	LAST_NODE_TYPE
)

// String returns string representation of NodeType
func (n NodeType) String() string {
	switch n {
	// select
	case SELECT_STATEMENT:
		return "SELECT_STATEMENT"
	case SELECT_CLAUSE:
		return "SELECT"
	case FROM_CLAUSE:
		return "FROM"
	case WHERE_CLAUSE:
		return "WHERE"
	case ORDER_BY_CLAUSE:
		return "ORDER BY"
	case GROUP_BY_CLAUSE:
		return "GROUP BY"
	case HAVING_CLAUSE:
		return "HAVING"
	case LIMIT_CLAUSE:
		return "LIMIT"
	case OFFSET_CLAUSE:
		return "OFFSET"
	case FOR_CLAUSE:
		return "FOR"
	// insert into
	case INSERT_INTO_STATEMENT:
		return "INSERT_INTO_STATEMENT"
	case INSERT_INTO_CLAUSE:
		return "INSERT_INTO"
	case VALUES_CLAUSE:
		return "VALUES"
	case ON_CONFLICT_CLAUSE:
		return "ON_CONFLICT"
	// update
	case UPDATE_STATEMENT:
		return "UPDATE_STATEMENT"
	case UPDATE_CLAUSE:
		return "UPDATE"
	case SET_CLAUSE:
		return "SET"
	// delete
	case DELETE_FROM_STATEMENT:
		return "DELETE_FROM_STATEMENT"
	case DELETE_FROM_CLAUSE:
		return "DELETE_FROM"
	// CTE and subquery
	case WITH_CLAUSE:
		return "WITH"
	case CTE_DEFINITION:
		return "CTE_DEFINITION"
	case TEMPLATE_IF_BLOCK:
		return "TEMPLATE_IF_BLOCK"
	case TEMPLATE_ELSEIF_BLOCK:
		return "TEMPLATE_ELSEIF_BLOCK"
	case TEMPLATE_ELSE_BLOCK:
		return "TEMPLATE_ELSE_BLOCK"
	case TEMPLATE_FOR_BLOCK:
		return "TEMPLATE_FOR_BLOCK"
	case VARIABLE_SUBSTITUTION:
		return "VARIABLE_SUBSTITUTION"
	case BULK_VARIABLE_SUBSTITUTION:
		return "BULK_VARIABLE_SUBSTITUTION"
	case ENVIRONMENT_REFERENCE:
		return "ENVIRONMENT_REFERENCE"
	case IMPLICIT_CONDITIONAL:
		return "IMPLICIT_CONDITIONAL"
	case IDENTIFIER:
		return "IDENTIFIER"
	case LITERAL:
		return "LITERAL"
	case EXPRESSION:
		return "EXPRESSION"
	case OTHER_NODE:
		return "OTHER_NODE"
	case RETURNING_CLAUSE:
		return "RETURNING"
	case COLUMN_REFERENCE:
		return "COLUMN_REFERENCE"
	default:
		return "UNKNOWN"
	}
}

// AstNode represents AST (Abstract Syntax Tree) node interface
// All AST nodes must implement this interface.
type AstNode interface {
	Type() NodeType
	Position() tokenizer.Position
	String() string
	RawTokens() []tokenizer.Token // Returns the original token sequence
}
