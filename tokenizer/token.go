package tokenizer

import (
	"errors"
	"strconv"
)

// Sentinel errors
var (
	ErrUnexpectedCharacter = errors.New("unexpected character")
	ErrUnterminatedString  = errors.New("unterminated string literal")
	ErrUnterminatedComment = errors.New("unterminated block comment")
	ErrInvalidNumber       = errors.New("invalid number format")
	ErrInvalidSingleColon  = errors.New("invalid single colon")
)

// TokenType represents the type of a token
type TokenType int

const (
	// --- Basic tokens ---
	EOF TokenType = iota + 1
	WHITESPACE

	STRING        // string literals ('text')
	IDENTIFIER    // quoted identifiers ("col")
	NUMBER        // numeric literals
	BOOLEAN       // boolean literals (true, false)
	DUMMY_START   // marker for start of dummy literal
	DUMMY_END     // marker for end of dummy literal
	DUMMY_LITERAL // placeholder for /*= variable */ directives
	DUMMY_PLACEHOLDER // placeholder for parsing only, will be replaced
	OPENED_PARENS // (
	CLOSED_PARENS // )
	COMMA         // ,
	SEMICOLON     // ;
	DOT           // .

	// --- Comments ---
	LINE_COMMENT  // -- line comment
	BLOCK_COMMENT // /* block comment */ (including SnapSQL extensions)

	// --- Arithmetic operators ---
	PLUS     // +
	MINUS    // -
	MULTIPLY // *
	DIVIDE   // /

	// --- Comparison operators ---
	EQUAL         // =
	NOT_EQUAL     // <>, !=
	LESS_THAN     // <
	GREATER_THAN  // >
	LESS_EQUAL    // <=
	GREATER_EQUAL // >=

	// --- Special operators ---
	JSON_OPERATOR // PostgreSQL JSON operators (->, ->>, #>, #>>)

	// --- Type cast ---
	DOUBLE_COLON // :: (PostgreSQL cast)

	// --- Logical/conditional operators ---
	AND     // AND keyword
	OR      // OR keyword
	NOT     // NOT keyword
	IN      // IN keyword
	EXISTS  // EXISTS keyword
	BETWEEN // BETWEEN keyword
	LIKE    // LIKE keyword
	SIMILAR // SIMILAR keyword (PostgreSQL)
	TO      // TO keyword (PostgreSQL)
	REGEXP  // REGEXP keyword (MySQL/SQLite)
	RLIKE   // RLIKE keyword (MySQL)
	ILIKE   // ILIKE keyword (PostgreSQL)
	IS      // IS keyword
	NULL    // NULL keyword

	// --- Typed literal tokens ---
	DATE_LITERAL      // DATE '...'
	TIMESTAMP_LITERAL // TIMESTAMP '...'
	CAST              // CAST expression (CAST(... AS type))

	// --- Window function related ---
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

	// --- CTE related ---
	WITH      // WITH keyword
	RECURSIVE // RECURSIVE keyword
	AS        // AS keyword

	// --- Select ---
	SELECT // SELECT keyword

	ALL      // ALL keyword
	DISTINCT // DISTINCT keyword

	WHERE     // WHERE keyword
	GROUP     // GROUP keyword
	HAVING    // HAVING keyword
	LIMIT     // LIMIT keyword
	OFFSET    // OFFSET keyword
	RETURNING // RETURNING keyword

	// --- Insert ---

	INSERT // INSERT keyword
	INTO   // INTO keyword
	VALUES // VALUES keyword

	// --- Update ---
	UPDATE    // UPDATE keyword
	SET       // SET keyword
	ON        // ON keyword
	DUPLICATE // DUPLICATE keyword
	KEY       // KEY keyword
	CONFLICT  // CONFLICT keyword

	// --- Delete ---
	DELETE // DELETE keyword
	FROM   // FROM keyword

	// --- Row locking and concurrency control ---
	FOR    // FOR keyword
	SHARE  // SHARE keyword
	NO     // NO keyword
	NOWAIT // NOWAIT keyword
	SKIP   // SKIP keyword
	LOCKED // LOCKED keyword

	// --- Join ---
	JOIN    // JOIN keyword
	INNER   // INNER keyword
	OUTER   // OUTER keyword
	LEFT    // LEFT keyword
	RIGHT   // RIGHT keyword
	FULL    // FULL keyword
	CROSS   // CROSS keyword
	USING   // USING keyword (for join conditions)
	NATURAL // NATURAL keyword (for natural joins)

	// --- Order By ---
	ASC     // ASC keyword
	DESC    // DESC keyword
	COLLATE // COLLATE keyword

	// --- Others ---
	OTHER // complex expressions, database-specific syntax
	UNION // UNION keyword

	// --- Expression ---
	CASE // CASE expression
	WHEN // WHEN keyword
	THEN // THEN keyword
	ELSE // ELSE keyword
	END  // END keyword

	// --- Group By ---
	ROLLUP
	CUBE
	GROUPING
	SETS

	// --- Extended token types ---
	CONTEXTUAL_IDENTIFIER // Non-reserved keyword used as identifier
	RESERVED_IDENTIFIER   // Strictly reserved keyword used as identifier (quoted)
)

// String returns the string representation of TokenType
func (t TokenType) String() string {
	switch t {
	case DOUBLE_COLON:
		return "DOUBLE_COLON"
	case EOF:
		return "EOF"
	case WHITESPACE:
		return "WHITESPACE"
	case STRING:
		return "STRING"
	case BOOLEAN:
		return "BOOLEAN"
	case DUMMY_START:
		return "DUMMY_START"
	case DUMMY_END:
		return "DUMMY_END"
	case DUMMY_LITERAL:
		return "DUMMY_LITERAL"
	case DUMMY_PLACEHOLDER:
		return "DUMMY_PLACEHOLDER"
	case IDENTIFIER:
		return "IDENTIFIER"
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
	case SIMILAR:
		return "SIMILAR"
	case TO:
		return "TO"
	case REGEXP:
		return "REGEXP"
	case RLIKE:
		return "RLIKE"
	case ILIKE:
		return "ILIKE"
	case IS:
		return "IS"
	case JSON_OPERATOR:
		return "JSON_OPERATOR"
	case NULL:
		return "NULL"
	case CAST:
		return "CAST"
	case DATE_LITERAL:
		return "DATE_LITERAL"
	case TIMESTAMP_LITERAL:
		return "TIMESTAMP_LITERAL"
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
	case LIMIT:
		return "LIMIT"
	case OFFSET:
		return "OFFSET"
	case SET:
		return "SET"
	case VALUES:
		return "VALUES"
	case RETURNING:
		return "RETURNING"
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
	case ON:
		return "ON"
	case CONFLICT:
		return "CONFLICT"
	case DUPLICATE:
		return "DUPLICATE"
	case KEY:
		return "KEY"
	case FOR:
		return "FOR"
	case SHARE:
		return "SHARE"
	case NO:
		return "NO"
	case NOWAIT:
		return "NOWAIT"
	case SKIP:
		return "SKIP"
	case LOCKED:
		return "LOCKED"
	case JOIN:
		return "JOIN"
	case INNER:
		return "INNER"
	case OUTER:
		return "OUTER"
	case LEFT:
		return "LEFT"
	case RIGHT:
		return "RIGHT"
	case FULL:
		return "FULL"
	case CROSS:
		return "CROSS"
	case USING:
		return "USING"
	case NATURAL:
		return "NATURAL"
	case ASC:
		return "ASC"
	case DESC:
		return "DESC"
	case COLLATE:
		return "COLLATE"
	// --- Expression ---
	case CASE:
		return "CASE"
	case WHEN:
		return "WHEN"
	case THEN:
		return "THEN"
	case ELSE:
		return "ELSE"
	case END:
		return "END"
	// --- Group By ---
	case ROLLUP:
		return "ROLLUP"
	case CUBE:
		return "CUBE"
	case GROUPING:
		return "GROUPING"
	case SETS:
		return "SETS"
	case CONTEXTUAL_IDENTIFIER:
		return "CONTEXTUAL_IDENTIFIER"
	case RESERVED_IDENTIFIER:
		return "RESERVED_IDENTIFIER"
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

func (p Position) String() string {
	return strconv.Itoa(p.Line) + ":" + strconv.Itoa(p.Column)
}

// Token represents a token
type Token struct {
	Index     int
	Type      TokenType
	Value     string
	Position  Position
	Directive *Directive // SnapSQL directive information. nil if not a directive
}

// String returns the string representation of Token
func (t Token) String() string {
	if t.Directive != nil {
		return t.Type.String() + "(" + t.Directive.Type + "): " + t.Value
	}
	return t.Type.String() + ": " + t.Value
}

// SnapSQL directive structure
type Directive struct {
	Type       string // "if", "elseif", "else", "for", "end", "const", "variable"
	NextIndex  int    // Index of next directive token in block chain (if->elseif->else->end, for->end)
	DummyRange []int
	Condition  string // Condition expression for if/elseif directives
}
