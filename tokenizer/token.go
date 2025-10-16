package tokenizer

import (
	"strconv"

	"github.com/shibukawa/snapsql"
)

// Sentinel errors
var (
	ErrUnexpectedCharacter = snapsql.ErrUnexpectedCharacter
	ErrUnterminatedString  = snapsql.ErrUnterminatedString
	ErrUnterminatedComment = snapsql.ErrUnterminatedComment
	ErrInvalidNumber       = snapsql.ErrInvalidNumber
	ErrInvalidSingleColon  = snapsql.ErrInvalidSingleColon
)

// TokenType represents the type of a token
type TokenType int

const (
	// EOF represents end of file token.
	EOF TokenType = iota + 1
	// WHITESPACE represents whitespace.
	WHITESPACE

	STRING            // string literals ('text')
	IDENTIFIER        // quoted identifiers ("col")
	NUMBER            // numeric literals
	BOOLEAN           // boolean literals (true, false)
	DUMMY_START       // marker for start of dummy literal
	DUMMY_END         // marker for end of dummy literal
	DUMMY_LITERAL     // placeholder for /*= variable */ directives
	DUMMY_PLACEHOLDER // placeholder for parsing only, will be replaced
	OPENED_PARENS     // (
	CLOSED_PARENS     // )
	COMMA             // ,
	SEMICOLON         // ;
	DOT               // .

	// LINE_COMMENT represents a line comment starting with --.
	LINE_COMMENT
	// BLOCK_COMMENT represents a block comment including SnapSQL extensions.
	BLOCK_COMMENT

	// PLUS represents '+'.
	PLUS // +
	// MINUS represents '-'.
	MINUS // -
	// MULTIPLY represents '*'.
	MULTIPLY // *
	// DIVIDE represents '/'.
	DIVIDE // /
	// MODULO represents '%'.
	MODULO // %
	// CONCAT represents '||'.
	CONCAT // ||

	// EQUAL represents '='.
	EQUAL // =
	// NOT_EQUAL represents '<>' or '!='.
	NOT_EQUAL // <>, !=
	// LESS_THAN represents '<'.
	LESS_THAN // <
	// GREATER_THAN represents '>'.
	GREATER_THAN // >
	// LESS_EQUAL represents '<='.
	LESS_EQUAL // <=
	// GREATER_EQUAL represents '>='.
	GREATER_EQUAL // >=

	// JSON_OPERATOR represents PostgreSQL JSON operators (->, ->>, #>, #>>).
	JSON_OPERATOR

	// DOUBLE_COLON represents PostgreSQL type cast '::'.
	DOUBLE_COLON

	// AND represents logical AND.
	AND // AND keyword
	// OR represents logical OR.
	OR // OR keyword
	// NOT represents logical NOT.
	NOT // NOT keyword
	// IN represents IN operator.
	IN // IN keyword
	// EXISTS represents EXISTS keyword.
	EXISTS // EXISTS keyword
	// BETWEEN represents BETWEEN operator.
	BETWEEN // BETWEEN keyword
	// LIKE represents LIKE operator.
	LIKE // LIKE keyword
	// SIMILAR represents SIMILAR keyword (PostgreSQL).
	SIMILAR // SIMILAR keyword (PostgreSQL)
	// TO represents TO keyword (PostgreSQL).
	TO // TO keyword (PostgreSQL)
	// REGEXP represents REGEXP keyword (MySQL/SQLite).
	REGEXP // REGEXP keyword (MySQL/SQLite)
	// RLIKE represents RLIKE keyword (MySQL).
	RLIKE // RLIKE keyword (MySQL)
	// ILIKE represents ILIKE keyword (PostgreSQL).
	ILIKE // ILIKE keyword (PostgreSQL)
	// IS represents IS keyword.
	IS // IS keyword
	// NULL represents NULL keyword.
	NULL // NULL keyword

	// DATE_LITERAL represents DATE '...'.
	DATE_LITERAL
	// TIMESTAMP_LITERAL represents TIMESTAMP '...'.
	TIMESTAMP_LITERAL
	// CAST represents a CAST expression token.
	CAST

	// OVER represents OVER keyword.
	OVER
	// PARTITION represents PARTITION keyword.
	PARTITION
	// ORDER represents ORDER keyword in window context.
	ORDER
	// BY represents BY keyword in window context.
	BY
	// ROWS represents ROWS keyword.
	ROWS
	// RANGE represents RANGE keyword.
	RANGE
	// UNBOUNDED represents UNBOUNDED keyword.
	UNBOUNDED
	// PRECEDING represents PRECEDING keyword.
	PRECEDING
	// FOLLOWING represents FOLLOWING keyword.
	FOLLOWING
	// CURRENT represents CURRENT keyword.
	CURRENT
	// ROW represents ROW keyword.
	ROW

	// WITH represents WITH keyword.
	WITH
	// RECURSIVE represents RECURSIVE keyword.
	RECURSIVE
	// AS represents AS keyword.
	AS

	// SELECT represents SELECT keyword.
	SELECT // SELECT keyword

	// ALL represents ALL keyword.
	ALL
	// DISTINCT represents DISTINCT keyword.
	DISTINCT

	// WHERE represents WHERE keyword.
	WHERE
	// GROUP represents GROUP keyword.
	GROUP
	// HAVING represents HAVING keyword.
	HAVING
	// LIMIT represents LIMIT keyword.
	LIMIT
	// OFFSET represents OFFSET keyword.
	OFFSET
	// RETURNING represents RETURNING keyword.
	RETURNING

	// INSERT represents INSERT keyword.
	INSERT
	// INTO represents INTO keyword.
	INTO
	// VALUES represents VALUES keyword.
	VALUES

	// UPDATE represents UPDATE keyword.
	UPDATE
	// SET represents SET keyword.
	SET
	// ON represents ON keyword.
	ON
	// DUPLICATE represents DUPLICATE keyword.
	DUPLICATE
	// KEY represents KEY keyword.
	KEY
	// CONFLICT represents CONFLICT keyword.
	CONFLICT

	// DELETE represents DELETE keyword.
	DELETE
	// FROM represents FROM keyword.
	FROM

	// FOR represents FOR row locking keyword.
	FOR
	// SHARE represents SHARE keyword.
	SHARE
	// NO represents NO keyword.
	NO
	// NOWAIT represents NOWAIT keyword.
	NOWAIT
	// SKIP represents SKIP keyword.
	SKIP
	// LOCKED represents LOCKED keyword.
	LOCKED

	// JOIN represents JOIN keyword.
	JOIN
	// INNER represents INNER keyword.
	INNER
	// OUTER represents OUTER keyword.
	OUTER
	// LEFT represents LEFT keyword.
	LEFT
	// RIGHT represents RIGHT keyword.
	RIGHT
	// FULL represents FULL keyword.
	FULL
	// CROSS represents CROSS keyword.
	CROSS
	// USING represents USING keyword for join conditions.
	USING
	// NATURAL represents NATURAL keyword for natural joins.
	NATURAL

	// ASC represents ASC keyword.
	ASC
	// DESC represents DESC keyword.
	DESC
	// COLLATE represents COLLATE keyword.
	COLLATE

	// OTHER represents other complex expressions.
	OTHER
	// UNION represents UNION keyword.
	UNION

	// CASE represents CASE expression keyword.
	CASE
	// WHEN represents WHEN keyword.
	WHEN
	// THEN represents THEN keyword.
	THEN
	// ELSE represents ELSE keyword.
	ELSE
	// END represents END keyword.
	END

	// ROLLUP represents ROLLUP keyword.
	ROLLUP
	// CUBE represents CUBE keyword.
	CUBE
	// GROUPING represents GROUPING keyword.
	GROUPING
	// SETS represents SETS keyword.
	SETS

	// CONTEXTUAL_IDENTIFIER represents a non-reserved word used as identifier.
	CONTEXTUAL_IDENTIFIER
	// RESERVED_IDENTIFIER represents a reserved word used as identifier.
	RESERVED_IDENTIFIER
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
	case MODULO:
		return "MODULO"
	case CONCAT:
		return "CONCAT"
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

// Directive represents a SnapSQL inline directive extracted from comments.
type Directive struct {
	Type        string // "if", "elseif", "else", "for", "end", "const", "variable", "system_value"
	NextIndex   int    // Index of next directive token in block chain (if->elseif->else->end, for->end)
	DummyRange  []int
	Condition   string // Condition expression for if/elseif directives
	SystemField string // System field name for "system_value" type
}
