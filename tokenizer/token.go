package tokenizer

import "errors"

// Sentinel errors
var (
	ErrUnexpectedCharacter = errors.New("unexpected character")
	ErrUnterminatedString  = errors.New("unterminated string literal")
	ErrUnterminatedComment = errors.New("unterminated block comment")
	ErrInvalidNumber       = errors.New("invalid number format")
)

// TokenType represents the type of a token
type TokenType int

const (
	// Basic tokens
	EOF TokenType = iota
	WHITESPACE
	WORD          // identifiers, keywords
	QUOTE         // string literals ('text', "text")
	NUMBER        // numeric literals
	OPENED_PARENS // (
	CLOSED_PARENS // )
	COMMA         // ,
	SEMICOLON     // ;
	DOT           // .

	// SQL operators
	EQUAL         // =
	NOT_EQUAL     // <>, !=
	LESS_THAN     // <
	GREATER_THAN  // >
	LESS_EQUAL    // <=
	GREATER_EQUAL // >=
	PLUS          // +
	MINUS         // -
	MULTIPLY      // *
	DIVIDE        // /

	// Window function related
	OVER      // OVER keyword
	PARTITION // PARTITION keyword
	ORDER     // ORDER keyword
	BY        // BY keyword
	ROWS      // ROWS keyword
	RANGE     // RANGE keyword
	UNBOUNDED // UNBOUNDED keyword
	PRECEDING // PRECEDING keyword
	FOLLOWING // FOLLOWING keyword
	CURRENT   // CURRENT keyword
	ROW       // ROW keyword

	// Logical operators and conditional expressions
	AND     // AND keyword
	OR      // OR keyword
	NOT     // NOT keyword
	IN      // IN keyword
	EXISTS  // EXISTS keyword
	BETWEEN // BETWEEN keyword
	LIKE    // LIKE keyword
	IS      // IS keyword
	NULL    // NULL keyword

	// Subquery and CTE related
	WITH     // WITH keyword
	AS       // AS keyword
	SELECT   // SELECT keyword
	INSERT   // INSERT keyword
	UPDATE   // UPDATE keyword
	DELETE   // DELETE keyword
	FROM     // FROM keyword
	WHERE    // WHERE keyword
	GROUP    // GROUP keyword
	HAVING   // HAVING keyword
	UNION    // UNION keyword
	ALL      // ALL keyword
	DISTINCT // DISTINCT keyword

	// Comments
	LINE_COMMENT  // -- line comment
	BLOCK_COMMENT // /* block comment */ (including SnapSQL extensions)

	// Others
	OTHER // complex expressions, database-specific syntax
)

// String returns the string representation of TokenType
func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case WHITESPACE:
		return "WHITESPACE"
	case WORD:
		return "WORD"
	case QUOTE:
		return "QUOTE"
	case NUMBER:
		return "NUMBER"
	case OPENED_PARENS:
		return "OPENED_PARENS"
	case CLOSED_PARENS:
		return "CLOSED_PARENS"
	case COMMA:
		return "COMMA"
	case SEMICOLON:
		return "SEMICOLON"
	case DOT:
		return "DOT"
	case EQUAL:
		return "EQUAL"
	case NOT_EQUAL:
		return "NOT_EQUAL"
	case LESS_THAN:
		return "LESS_THAN"
	case GREATER_THAN:
		return "GREATER_THAN"
	case LESS_EQUAL:
		return "LESS_EQUAL"
	case GREATER_EQUAL:
		return "GREATER_EQUAL"
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case MULTIPLY:
		return "MULTIPLY"
	case DIVIDE:
		return "DIVIDE"
	case OVER:
		return "OVER"
	case PARTITION:
		return "PARTITION"
	case ORDER:
		return "ORDER"
	case BY:
		return "BY"
	case ROWS:
		return "ROWS"
	case RANGE:
		return "RANGE"
	case UNBOUNDED:
		return "UNBOUNDED"
	case PRECEDING:
		return "PRECEDING"
	case FOLLOWING:
		return "FOLLOWING"
	case CURRENT:
		return "CURRENT"
	case ROW:
		return "ROW"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case NOT:
		return "NOT"
	case IN:
		return "IN"
	case EXISTS:
		return "EXISTS"
	case BETWEEN:
		return "BETWEEN"
	case LIKE:
		return "LIKE"
	case IS:
		return "IS"
	case NULL:
		return "NULL"
	case WITH:
		return "WITH"
	case AS:
		return "AS"
	case SELECT:
		return "SELECT"
	case INSERT:
		return "INSERT"
	case UPDATE:
		return "UPDATE"
	case DELETE:
		return "DELETE"
	case FROM:
		return "FROM"
	case WHERE:
		return "WHERE"
	case GROUP:
		return "GROUP"
	case HAVING:
		return "HAVING"
	case UNION:
		return "UNION"
	case ALL:
		return "ALL"
	case DISTINCT:
		return "DISTINCT"
	case LINE_COMMENT:
		return "LINE_COMMENT"
	case BLOCK_COMMENT:
		return "BLOCK_COMMENT"
	case OTHER:
		return "OTHER"
	default:
		return "UNKNOWN"
	}
}

// Position represents a position in the source code
type Position struct {
	Line   int
	Column int
	Offset int
}

// Token represents a token
type Token struct {
	Type     TokenType
	Value    string
	Position Position

	// SnapSQL extension information (Phase 1: store only, no parsing)
	IsSnapSQLDirective bool
	DirectiveType      string // "if", "for", "variable", etc.
}

// String returns the string representation of Token
func (t Token) String() string {
	if t.IsSnapSQLDirective {
		return t.Type.String() + "(" + t.DirectiveType + "): " + t.Value
	}
	return t.Type.String() + ": " + t.Value
}
