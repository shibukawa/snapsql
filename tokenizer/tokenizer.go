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
	dialect SqlDialect
	options TokenizerOptions
}

// TokenizerOptions are options for the tokenizer
type TokenizerOptions struct {
	SkipWhitespace bool
	SkipComments   bool
	PreserveCase   bool
}

// NewSqlTokenizer creates a new SqlTokenizer
func NewSqlTokenizer(input string, dialect SqlDialect, options ...TokenizerOptions) *SqlTokenizer {
	opts := TokenizerOptions{
		SkipWhitespace: false,
		SkipComments:   false,
		PreserveCase:   false,
	}
	if len(options) > 0 {
		opts = options[0]
	}

	return &SqlTokenizer{
		input:   input,
		dialect: dialect,
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
			dialect:  t.dialect,
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

// Internal tokenizer implementation
type tokenizer struct {
	input    string
	position int
	line     int
	column   int
	current  rune
	dialect  SqlDialect
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
			if t.peekChar() == '-' {
				return t.readLineComment()
			}
			token := t.newToken(MINUS, string(t.current))
			t.readChar()
			return token, nil
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
				return t.readWord()
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

// readWord reads words (identifiers and keywords)
func (t *tokenizer) readWord() (Token, error) {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	for unicode.IsLetter(t.current) || unicode.IsDigit(t.current) || t.current == '_' {
		builder.WriteRune(t.current)
		t.readChar()
	}

	word := builder.String()
	if !t.options.PreserveCase {
		word = strings.ToUpper(word)
	}

	// キーワード判定
	tokenType := t.getKeywordTokenType(word)

	return Token{
		Type:  tokenType,
		Value: word,
		Position: Position{
			Line:   startLine,
			Column: startColumn,
			Offset: startOffset,
		},
	}, nil
}

// readString reads string literals or quoted identifiers
func (t *tokenizer) readString(delimiter rune) (Token, error) {
	var builder strings.Builder
	startLine := t.line
	startColumn := t.column - 1
	startOffset := t.position - 1

	builder.WriteRune(delimiter) // include opening quote
	t.readChar()

	for t.current != 0 {
		if t.current == delimiter {
			// 連続するクオートはエスケープ（例: '' or "" or ``）
			if t.peekChar() == delimiter {
				builder.WriteRune(delimiter)
				t.readChar()
				t.readChar()
				continue
			}
			break // 終了クオート
		}
		if t.current == '\\' && delimiter == '\'' {
			// バックスラッシュエスケープ（PostgreSQL互換）
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

	type_ := STRING
	if delimiter == '"' || delimiter == '`' {
		type_ = IDENTIFIER
	}

	return Token{
		Type:  type_,
		Value: builder.String(),
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

// getKeywordTokenType returns the TokenType corresponding to a keyword
func (t *tokenizer) getKeywordTokenType(word string) TokenType {
	upperWord := strings.ToUpper(word)

	switch upperWord {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "FROM", "WHERE", "GROUP", "HAVING", "ORDER", "BY", "UNION", "ALL", "DISTINCT", "AS", "WITH", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "IS", "NULL":
		return KEYWORD
	// Window function keywords
	case "OVER", "PARTITION", "ROWS", "RANGE", "UNBOUNDED", "PRECEDING", "FOLLOWING", "CURRENT", "ROW":
		return KEYWORD
	// Additional SQL keywords (DDL, DML, etc.)
	case "ON", "CONFLICT", "DUPLICATE", "KEY", "CREATE", "DROP", "ALTER", "TABLE", "INDEX", "VIEW", "VALUES", "INTO", "SET", "RETURNING", "LIMIT", "OFFSET":
		return KEYWORD
	default:
		return IDENTIFIER
	}
}
