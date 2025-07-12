package tokenizer

import (
	"fmt"
	"iter"
	"strings"
	"unicode"
)

// TokenIterator uses Go 1.24 iterator pattern
type TokenIterator iter.Seq2[Token, error]

// SqlTokenizer is a tokenizer that returns an iterator
type SqlTokenizer struct {
	input   string
	options TokenizerOptions
}

// TokenizerOptions are options for the tokenizer
type TokenizerOptions struct {
	SkipWhitespace bool
	SkipComments   bool
}

// NewSqlTokenizer creates a new SqlTokenizer
func NewSqlTokenizer(input string, options ...TokenizerOptions) *SqlTokenizer {
	opts := TokenizerOptions{
		SkipWhitespace: false,
		SkipComments:   false,
	}
	if len(options) > 0 {
		opts = options[0]
	}

	return &SqlTokenizer{
		input:   input,
		options: opts,
	}
}

// Tokens returns an iterator of tokens
func (t *SqlTokenizer) Tokens() TokenIterator {
	return func(yield func(Token, error) bool) {
		tokenizer := &tokenizer{
			input:    t.input,
			position: 0,
			line:     1,
			column:   1,
			options:  t.options,
		}

		tokenizer.readChar()

		for {
			token, err := tokenizer.nextToken()
			if err != nil {
				if !yield(Token{}, err) {
					return
				}
				continue
			}

			if token.Type == EOF {
				yield(token, nil)
				return
			}

			// Filtering based on options
			if t.options.SkipWhitespace && token.Type == WHITESPACE {
				continue
			}
			if t.options.SkipComments && (token.Type == LINE_COMMENT || token.Type == BLOCK_COMMENT) {
				continue
			}

			if !yield(token, nil) {
				return
			}
		}
	}
}

// AllTokens gets all tokens as a slice (for debugging)
func (t *SqlTokenizer) AllTokens() ([]Token, error) {
	tokens := make([]Token, 0, 64)
	var lastError error

	for token, err := range t.Tokens() {
		if err != nil {
			lastError = err
			continue
		}
		tokens = append(tokens, token)
		if token.Type == EOF {
			break
		}
	}

	return tokens, lastError
}

// Tokenize is a helper that tokenizes SQL and returns all tokens (for tests).
func Tokenize(sql string) ([]Token, error) {
	t := NewSqlTokenizer(sql)
	return t.AllTokens()
}

// Internal tokenizer implementation
type tokenizer struct {
	input    string
	position int
	line     int
	column   int
	current  rune
	options  TokenizerOptions
}

// nextToken gets the next token
func (t *tokenizer) nextToken() (Token, error) {
	for {
		switch t.current {
		case 0:
			return t.newToken(EOF, ""), nil
		case ' ', '\t', '\r', '\n':
			return t.readWhitespace(), nil
		case '(':
			token := t.newToken(OPENED_PARENS, string(t.current))
			t.readChar()
			return token, nil
		case ')':
			token := t.newToken(CLOSED_PARENS, string(t.current))
			t.readChar()
			return token, nil
		case ',':
			token := t.newToken(COMMA, string(t.current))
			t.readChar()
			return token, nil
		case ';':
			token := t.newToken(SEMICOLON, string(t.current))
			t.readChar()
			return token, nil
		case '.':
			token := t.newToken(DOT, string(t.current))
			t.readChar()
			return token, nil
		case '\'', '"', '`':
			return t.readString(t.current)
		case '-':
			// PostgreSQL JSON operators: ->, ->>
			if t.peekChar() == '>' {
				t.readChar() // consume '-'
				if t.peekChar() == '>' {
					t.readChar() // consume first '>'
					t.readChar() // consume second '>'
					return t.newToken(JSON_OPERATOR, "->>"), nil
				}
				t.readChar() // consume '>'
				return t.newToken(JSON_OPERATOR, "->"), nil
			}
			if t.peekChar() == '-' {
				return t.readLineComment()
			}
			token := t.newToken(MINUS, string(t.current))
			t.readChar()
			return token, nil
		case '#':
			// PostgreSQL JSON operators: #>, #>>
			if t.peekChar() == '>' {
				t.readChar() // consume '#'
				if t.peekChar() == '>' {
					t.readChar() // consume first '>'
					t.readChar() // consume second '>'
					return t.newToken(JSON_OPERATOR, "#>>"), nil
				}
				t.readChar() // consume '>'
				return t.newToken(JSON_OPERATOR, "#>"), nil
			}
			token := t.newToken(OTHER, string(t.current))
			t.readChar()
			return token, nil
		case ':':
			// PostgreSQL-style cast operator ::
			if t.peekChar() == ':' {
				t.readChar() // consume first
				t.readChar() // consume second
				return t.newToken(DOUBLE_COLON, "::"), nil
			}
			return Token{}, fmt.Errorf("%w: invalid ':' at line %d, column %d", ErrInvalidSingleColon, t.line, t.column-1)
		case '/':
			if t.peekChar() == '*' {
				return t.readBlockComment()
			}
			token := t.newToken(DIVIDE, string(t.current))
			t.readChar()
			return token, nil
		case '=':
			token := t.newToken(EQUAL, string(t.current))
			t.readChar()
			return token, nil
		case '<':
			if t.peekChar() == '=' {
				t.readChar()
				t.readChar()
				return t.newToken(LESS_EQUAL, "<="), nil
			} else if t.peekChar() == '>' {
				t.readChar()
				t.readChar()
				return t.newToken(NOT_EQUAL, "<>"), nil
			}
			token := t.newToken(LESS_THAN, string(t.current))
			t.readChar()
			return token, nil
		case '>':
			if t.peekChar() == '=' {
				t.readChar()
				t.readChar()
				return t.newToken(GREATER_EQUAL, ">="), nil
			}
			token := t.newToken(GREATER_THAN, string(t.current))
			t.readChar()
			return token, nil
		case '!':
			if t.peekChar() == '=' {
				t.readChar()
				t.readChar()
				return t.newToken(NOT_EQUAL, "!="), nil
			}
			// '!' alone is treated as OTHER
			return t.readOther()
		case '+':
			token := t.newToken(PLUS, string(t.current))
			t.readChar()
			return token, nil
		case '*':
			token := t.newToken(MULTIPLY, string(t.current))
			t.readChar()
			return token, nil
		default:
			if unicode.IsLetter(t.current) || t.current == '_' {
				return t.readIdentifierOrKeyword()
			} else if unicode.IsDigit(t.current) {
				return t.readNumber()
			} else {
				// Other characters are treated as OTHER
				return t.readOther()
			}
		}
	}
}

// readChar reads the next character
func (t *tokenizer) readChar() {
	if t.position >= len(t.input) {
		t.current = 0
		t.position++
		return
	}

	t.current = rune(t.input[t.position])
	t.position++

	if t.current == '\n' {
		t.line++
		t.column = 1
	} else {
		t.column++
	}
}

// peekChar looks ahead at the next character
func (t *tokenizer) peekChar() rune {
	if t.position >= len(t.input) {
		return 0
	}
	return rune(t.input[t.position])
}

// readWhitespace reads whitespace characters
func (t *tokenizer) readWhitespace() Token {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	for unicode.IsSpace(t.current) {
		builder.WriteRune(t.current)
		t.readChar()
	}

	return Token{
		Type:  WHITESPACE,
		Value: builder.String(),
		Position: Position{
			Line:   startLine,
			Column: startColumn,
			Offset: startOffset,
		},
	}
}

var keywordLikeTokenTypeMap = map[string]TokenType{
	// Row locking and concurrency control
	"AND":   AND,
	"OR":    OR,
	"NOT":   NOT,
	"NULL":  NULL,
	"TRUE":  BOOLEAN,
	"FALSE": BOOLEAN,

	"SELECT":    SELECT,
	"FROM":      FROM,
	"WHERE":     WHERE,
	"ORDER":     ORDER,
	"BY":        BY,
	"GROUP":     GROUP,
	"HAVING":    HAVING,
	"LIMIT":     LIMIT,
	"OFFSET":    OFFSET,
	"FOR":       FOR,
	"SHARE":     SHARE,
	"NO":        NO,
	"NOWAIT":    NOWAIT,
	"SKIP":      SKIP,
	"LOCKED":    LOCKED,
	"RETURNING": RETURNING,

	"INSERT": INSERT,
	"INTO":   INTO,
	"VALUES": VALUES,

	"UPDATE":   UPDATE,
	"SET":      SET,
	"ON":       ON,
	"CONFLICT": CONFLICT,

	"DELETE": DELETE,

	"WITH":      WITH,
	"RECURSIVE": RECURSIVE,
	"AS":        AS,

	"CAST":     CAST,
	"DISTINCT": DISTINCT,
	"ALL":      ALL,

	"JOIN":    JOIN,
	"INNER":   INNER,
	"OUTER":   OUTER,
	"LEFT":    LEFT,
	"RIGHT":   RIGHT,
	"FULL":    FULL,
	"USING":   USING,
	"NATURAL": NATURAL,
	"CROSS":   CROSS,

	"ASC":     ASC,
	"DESC":    DESC,
	"COLLATE": COLLATE,

	// Expression keywords
	"CASE": CASE,
	"WHEN": WHEN,
	"THEN": THEN,
	"ELSE": ELSE,
	"END":  END,

	// Group By keywords
	"ROLLUP":   ROLLUP,
	"CUBE":     CUBE,
	"GROUPING": GROUPING,
	"SETS":     SETS,
}

// readIdentifierOrKeyword reads identifiers and keywords with strict reservation checking
func (t *tokenizer) readIdentifierOrKeyword() (Token, error) {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	// Read identifier characters
	for unicode.IsLetter(t.current) || unicode.IsDigit(t.current) || t.current == '_' {
		builder.WriteRune(t.current)
		t.readChar()
	}

	originalValue := builder.String()
	upperValue := strings.ToUpper(originalValue)

	var tokenType TokenType
	if tt, ok := keywordLikeTokenTypeMap[upperValue]; ok {
		tokenType = tt
	} else if info, ok := KeywordSet[upperValue]; ok && info.Keyword {
		if info.StrictReserved {
			tokenType = RESERVED_IDENTIFIER // Strictly reserved keywords as RESERVED_IDENTIFIER
		} else {
			tokenType = IDENTIFIER
		}
	} else {
		tokenType = IDENTIFIER // Regular identifiers
	}

	return Token{
		Type:     tokenType,
		Value:    originalValue, // Preserve original case
		Position: Position{Line: startLine, Column: startColumn, Offset: startOffset},
	}, nil
}

// readString reads string literals or quoted identifiers with reserved keyword support
func (t *tokenizer) readString(delimiter rune) (Token, error) {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	builder.WriteRune(delimiter) // include opening quote
	t.readChar()

	for t.current != 0 {
		if t.current == delimiter {
			// Handle escaped quotes (e.g., '' or "" or ``)
			if t.peekChar() == delimiter {
				builder.WriteRune(delimiter)
				t.readChar()
				t.readChar()
				continue
			}
			break // closing quote
		}
		if t.current == '\\' && delimiter == '\'' {
			// Backslash escape (PostgreSQL compatible)
			builder.WriteRune(t.current)
			t.readChar()
			if t.current != 0 {
				builder.WriteRune(t.current)
				t.readChar()
			}
			continue
		}
		builder.WriteRune(t.current)
		t.readChar()
	}

	if t.current != delimiter {
		return Token{}, fmt.Errorf("%w: %c at line %d, column %d", ErrUnterminatedString, delimiter, startLine, startColumn)
	}

	builder.WriteRune(delimiter) // include closing quote
	t.readChar()

	// Determine token type based on delimiter
	var tokenType TokenType
	quotedContent := builder.String()
	// Note: We no longer need to check reserved words for quoted identifiers

	if delimiter == '"' || delimiter == '[' || delimiter == '`' {
		// All quoted identifiers are treated as regular identifiers
		// For standard SQL and SQLite, double quotes (") are used
		// For MySQL, backticks (`) are used
		// For SQL Server, square brackets ([]) are used
		tokenType = IDENTIFIER // All quoted identifiers are regular identifiers
	} else {
		tokenType = STRING // String literal
	}

	return Token{
		Type:  tokenType,
		Value: quotedContent, // Keep quotes for quoted identifiers, remove later in parser
		Position: Position{
			Line:   startLine,
			Column: startColumn,
			Offset: startOffset,
		},
	}, nil
}

// readNumber reads numeric literals
func (t *tokenizer) readNumber() (Token, error) {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	// Integer part
	for unicode.IsDigit(t.current) {
		builder.WriteRune(t.current)
		t.readChar()
	}

	// Decimal point
	if t.current == '.' && unicode.IsDigit(t.peekChar()) {
		builder.WriteRune(t.current)
		t.readChar()

		// Decimal part
		for unicode.IsDigit(t.current) {
			builder.WriteRune(t.current)
			t.readChar()
		}
	}

	// Exponential part
	if t.current == 'e' || t.current == 'E' {
		builder.WriteRune(t.current)
		t.readChar()

		if t.current == '+' || t.current == '-' {
			builder.WriteRune(t.current)
			t.readChar()
		}

		if !unicode.IsDigit(t.current) {
			return Token{}, fmt.Errorf("%w: invalid exponent at line %d, column %d", ErrInvalidNumber, startLine, startColumn)
		}

		for unicode.IsDigit(t.current) {
			builder.WriteRune(t.current)
			t.readChar()
		}
	}

	return Token{
		Type:  NUMBER,
		Value: builder.String(),
		Position: Position{
			Line:   startLine,
			Column: startColumn,
			Offset: startOffset,
		},
	}, nil
}

// readLineComment reads line comments
func (t *tokenizer) readLineComment() (Token, error) {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	// '--' を読み取る
	builder.WriteRune(t.current)
	t.readChar()
	builder.WriteRune(t.current)
	t.readChar()

	// 行末まで読み取る
	for t.current != 0 && t.current != '\n' {
		builder.WriteRune(t.current)
		t.readChar()
	}

	return Token{
		Type:  LINE_COMMENT,
		Value: builder.String(),
		Position: Position{
			Line:   startLine,
			Column: startColumn,
			Offset: startOffset,
		},
	}, nil
}

// readBlockComment reads block comments
func (t *tokenizer) readBlockComment() (Token, error) {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	// '/*' を読み取る
	builder.WriteRune(t.current)
	t.readChar()
	builder.WriteRune(t.current)
	t.readChar()

	// '*/' まで読み取る
	for t.current != 0 {
		if t.current == '*' && t.peekChar() == '/' {
			builder.WriteRune(t.current)
			t.readChar()
			builder.WriteRune(t.current)
			t.readChar()
			break
		}
		builder.WriteRune(t.current)
		t.readChar()
	}

	if !strings.HasSuffix(builder.String(), "*/") {
		return Token{}, fmt.Errorf("%w at line %d, column %d", ErrUnterminatedComment, startLine, startColumn)
	}

	comment := builder.String()
	directiveType, isDirective := t.parseSnapSQLDirective(comment)

	return Token{
		Type:               BLOCK_COMMENT,
		Value:              comment,
		Position:           Position{Line: startLine, Column: startColumn, Offset: startOffset},
		IsSnapSQLDirective: isDirective,
		DirectiveType:      directiveType,
	}, nil
}

// readOther reads other characters
func (t *tokenizer) readOther() (Token, error) {
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1
	char := t.current

	t.readChar()

	return Token{
		Type:  OTHER,
		Value: string(char),
		Position: Position{
			Line:   startLine,
			Column: startColumn,
			Offset: startOffset,
		},
	}, nil
}

// newToken creates a new token
func (t *tokenizer) newToken(tokenType TokenType, value string) Token {
	return Token{
		Type:  tokenType,
		Value: value,
		Position: Position{
			Line:   t.line,
			Column: t.column - len([]rune(value)),
			Offset: t.position - len(value),
		},
	}
}

// parseSnapSQLDirective parses SnapSQL extension directives
func (t *tokenizer) parseSnapSQLDirective(comment string) (directiveType string, isDirective bool) {
	trimmed := strings.TrimSpace(comment)

	// /*# で始まる場合
	if strings.HasPrefix(trimmed, "/*#") && strings.HasSuffix(trimmed, "*/") {
		content := strings.TrimSpace(trimmed[3 : len(trimmed)-2])

		if strings.HasPrefix(content, "if") && (len(content) == 2 || content[2] == ' ') {
			return "if", true
		} else if strings.HasPrefix(content, "elseif") && (len(content) == 6 || content[6] == ' ') {
			return "elseif", true
		} else if content == "else" {
			return "else", true
		} else if strings.HasPrefix(content, "for") && (len(content) == 3 || content[3] == ' ') {
			return "for", true
		} else if content == "end" {
			return "end", true
		}
	}

	// /*@ で始まる場合（定数ディレクティブ）
	if strings.HasPrefix(trimmed, "/*@") && strings.HasSuffix(trimmed, "*/") {
		return "const", true
	}

	// If it starts with /*=
	if strings.HasPrefix(trimmed, "/*=") && strings.HasSuffix(trimmed, "*/") {
		return "variable", true
	}

	return "", false
}
