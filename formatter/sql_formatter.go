package formatter

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLFormatter formats SnapSQL templates with go fmt style
type SQLFormatter struct {
	indentSize int
}

// NewSQLFormatter creates a new SQL formatter
func NewSQLFormatter() *SQLFormatter {
	return &SQLFormatter{
		indentSize: 4, // 4 spaces for indentation
	}
}

// Format formats a SnapSQL template
func (f *SQLFormatter) Format(sql string) (string, error) {
	// Parse the SQL into tokens while preserving SnapSQL directives
	tokens, err := f.tokenize(sql)
	if err != nil {
		return "", fmt.Errorf("failed to tokenize SQL: %w", err)
	}

	// Format the tokens
	formatted := f.formatTokens(tokens)

	return formatted, nil
}

// Token represents a SQL token
type Token struct {
	Type   TokenType
	Value  string
	IsSnap bool // true if this is a SnapSQL directive
	Indent int  // indentation level
}

type TokenType int

const (
	TokenKeyword TokenType = iota
	TokenIdentifier
	TokenOperator
	TokenLiteral
	TokenComment
	TokenSnapDirective
	TokenNewline
	TokenComma
	TokenOpenParen
	TokenCloseParen
	TokenWhitespace
)

// Tokenize breaks SQL into tokens while preserving SnapSQL directives (public for testing)
func (f *SQLFormatter) Tokenize(sql string) ([]Token, error) {
	return f.tokenize(sql)
}

// tokenize breaks SQL into tokens while preserving SnapSQL directives
func (f *SQLFormatter) tokenize(sql string) ([]Token, error) {
	var tokens []Token

	// Regular expressions for different token types
	snapDirectiveRe := regexp.MustCompile(`/\*#[^*]*\*+(?:[^/*][^*]*\*+)*/|/\*=[^*]*\*+(?:[^/*][^*]*\*+)*/`)
	commentRe := regexp.MustCompile(`--[^\n]*|/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`)
	stringLiteralRe := regexp.MustCompile(`'([^'\\]|\\.)*'|"([^"\\]|\\.)*"`)
	numberRe := regexp.MustCompile(`\d+(\.\d+)?`)
	identifierRe := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`)

	// Keywords that should be uppercase
	keywords := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "JOIN": true, "INNER": true,
		"LEFT": true, "RIGHT": true, "FULL": true, "OUTER": true, "ON": true,
		"GROUP": true, "BY": true, "HAVING": true, "ORDER": true, "LIMIT": true,
		"OFFSET": true, "INSERT": true, "INTO": true, "VALUES": true, "UPDATE": true,
		"SET": true, "DELETE": true, "CREATE": true, "TABLE": true, "ALTER": true,
		"DROP": true, "INDEX": true, "VIEW": true, "UNION": true, "ALL": true,
		"DISTINCT": true, "AS": true, "AND": true, "OR": true, "NOT": true,
		"NULL": true, "IS": true, "IN": true, "EXISTS": true, "BETWEEN": true,
		"LIKE": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
		"END": true, "IF": true, "FOR": true, "COUNT": true, "SUM": true, "AVG": true,
		"MIN": true, "MAX": true, "NOW": true,
	}

	pos := 0
	for pos < len(sql) {
		// Skip whitespace but preserve newlines
		if sql[pos] == ' ' || sql[pos] == '\t' {
			pos++
			continue
		}

		if sql[pos] == '\n' {
			tokens = append(tokens, Token{Type: TokenNewline, Value: "\n"})
			pos++

			continue
		}

		// Check for SnapSQL directives first (they can contain special characters)
		if snapMatch := snapDirectiveRe.FindStringIndex(sql[pos:]); snapMatch != nil && snapMatch[0] == 0 {
			directive := sql[pos : pos+snapMatch[1]]
			tokens = append(tokens, Token{
				Type:   TokenSnapDirective,
				Value:  directive,
				IsSnap: true,
			})
			pos += snapMatch[1]

			continue
		}

		// Check for comments (but not SnapSQL directives)
		if strings.HasPrefix(sql[pos:], "--") || (strings.HasPrefix(sql[pos:], "/*") && !strings.HasPrefix(sql[pos:], "/*#") && !strings.HasPrefix(sql[pos:], "/*=")) {
			if commentMatch := commentRe.FindStringIndex(sql[pos:]); commentMatch != nil && commentMatch[0] == 0 {
				comment := sql[pos : pos+commentMatch[1]]
				tokens = append(tokens, Token{Type: TokenComment, Value: comment})
				pos += commentMatch[1]

				continue
			}
		}

		// Check for string literals
		if sql[pos] == '\'' || sql[pos] == '"' {
			if stringMatch := stringLiteralRe.FindStringIndex(sql[pos:]); stringMatch != nil && stringMatch[0] == 0 {
				literal := sql[pos : pos+stringMatch[1]]
				tokens = append(tokens, Token{Type: TokenLiteral, Value: literal})
				pos += stringMatch[1]

				continue
			}
		}

		// Check for numbers
		if sql[pos] >= '0' && sql[pos] <= '9' {
			if numberMatch := numberRe.FindStringIndex(sql[pos:]); numberMatch != nil && numberMatch[0] == 0 {
				number := sql[pos : pos+numberMatch[1]]
				tokens = append(tokens, Token{Type: TokenLiteral, Value: number})
				pos += numberMatch[1]

				continue
			}
		}

		// Check for identifiers/keywords
		if (sql[pos] >= 'a' && sql[pos] <= 'z') || (sql[pos] >= 'A' && sql[pos] <= 'Z') || sql[pos] == '_' {
			if identMatch := identifierRe.FindStringIndex(sql[pos:]); identMatch != nil && identMatch[0] == 0 {
				ident := sql[pos : pos+identMatch[1]]
				upperIdent := strings.ToUpper(ident)

				if keywords[upperIdent] {
					tokens = append(tokens, Token{Type: TokenKeyword, Value: upperIdent})
				} else {
					tokens = append(tokens, Token{Type: TokenIdentifier, Value: ident})
				}

				pos += identMatch[1]

				continue
			}
		}

		// Check for specific characters
		char := sql[pos]
		switch char {
		case ',':
			tokens = append(tokens, Token{Type: TokenComma, Value: ","})
		case '(':
			tokens = append(tokens, Token{Type: TokenOpenParen, Value: "("})
		case ')':
			tokens = append(tokens, Token{Type: TokenCloseParen, Value: ")"})
		case '=', '<', '>', '!', '+', '-', '*', '/', '%':
			// Handle multi-character operators
			if pos+1 < len(sql) && (sql[pos+1] == '=' || (char == '<' && sql[pos+1] == '>') || (char == '!' && sql[pos+1] == '=')) {
				tokens = append(tokens, Token{Type: TokenOperator, Value: sql[pos : pos+2]})
				pos++
			} else {
				tokens = append(tokens, Token{Type: TokenOperator, Value: string(char)})
			}
		default:
			tokens = append(tokens, Token{Type: TokenOperator, Value: string(char)})
		}

		pos++
	}

	return tokens, nil
}

// formatTokens formats the tokens according to the style rules
func (f *SQLFormatter) formatTokens(tokens []Token) string {
	var (
		result       strings.Builder
		indentLevel  int
		needsNewline bool
		lastToken    *Token
		inSelectList bool
		inValuesList bool
	)

	for i, token := range tokens {
		switch token.Type {
		case TokenSnapDirective:
			if after, ok := strings.CutPrefix(token.Value, "/*#"); ok {
				// Handle if/for directives
				directive := strings.TrimSpace(strings.Trim(after, "*/"))

				if strings.HasPrefix(directive, "if ") || strings.HasPrefix(directive, "for ") {
					if needsNewline {
						result.WriteString("\n")
					}

					result.WriteString(strings.Repeat(" ", indentLevel*f.indentSize))
					result.WriteString(token.Value)
					result.WriteString("\n")

					indentLevel++
					needsNewline = false
				} else if directive == "end" {
					indentLevel--

					if needsNewline {
						result.WriteString("\n")
					}

					result.WriteString(strings.Repeat(" ", indentLevel*f.indentSize))
					result.WriteString(token.Value)
					result.WriteString("\n")

					needsNewline = false
				} else if directive == "else" {
					if needsNewline {
						result.WriteString("\n")
					}

					result.WriteString(strings.Repeat(" ", (indentLevel-1)*f.indentSize))
					result.WriteString(token.Value)
					result.WriteString("\n")

					needsNewline = false
				} else {
					// Other directives (function definition, etc.)
					if i > 0 && lastToken != nil && lastToken.Type != TokenNewline {
						result.WriteString("\n")
					}

					result.WriteString(token.Value)
					result.WriteString("\n")

					needsNewline = false
				}
			} else {
				// Inline SnapSQL expressions /*= ... */
				if lastToken != nil && lastToken.Type != TokenNewline && lastToken.Value != "(" && lastToken.Value != "," {
					result.WriteString(" ")
				}

				result.WriteString(token.Value)
			}

		case TokenKeyword:
			if f.isStatementKeyword(token.Value) {
				if needsNewline || (lastToken != nil && lastToken.Type != TokenNewline && i > 0) {
					result.WriteString("\n")
				}

				// Special indentation for ON clause
				indent := indentLevel
				if token.Value == "ON" {
					indent = indentLevel + 1
				}

				result.WriteString(strings.Repeat(" ", indent*f.indentSize))
				result.WriteString(token.Value)

				needsNewline = false

				// Track if we're in a SELECT list or VALUES list
				if token.Value == "SELECT" {
					inSelectList = true
					// Add newline and indent for first SELECT item
					result.WriteString("\n")
					result.WriteString(strings.Repeat(" ", (indentLevel+1)*f.indentSize))
					// Create a fake newline token to prevent extra spaces
					fakeNewlineToken := Token{Type: TokenNewline, Value: "\n"}
					lastToken = &fakeNewlineToken

					continue // Skip the normal lastToken update
				} else if token.Value == "VALUES" {
					inValuesList = true
				} else if token.Value == "FROM" || token.Value == "WHERE" {
					inSelectList = false
					inValuesList = false
				}
			} else if token.Value == "AND" || token.Value == "OR" {
				result.WriteString(" ")
				result.WriteString(token.Value)
			} else {
				if lastToken != nil && lastToken.Type != TokenNewline && f.needsSpaceBefore(token.Value) {
					result.WriteString(" ")
				}

				result.WriteString(token.Value)
			}

		case TokenComma:
			result.WriteString(",")

			if inSelectList || inValuesList {
				result.WriteString("\n")
				result.WriteString(strings.Repeat(" ", (indentLevel+1)*f.indentSize))
				// Create a fake newline token to prevent extra spaces
				fakeNewlineToken := Token{Type: TokenNewline, Value: "\n"}
				lastToken = &fakeNewlineToken

				continue // Skip the normal lastToken update
			}

		case TokenNewline:
			if needsNewline {
				result.WriteString("\n")
				result.WriteString(strings.Repeat(" ", indentLevel*f.indentSize))

				needsNewline = false
			}

		case TokenComment:
			if strings.HasPrefix(token.Value, "--") {
				result.WriteString(" ")
				result.WriteString(token.Value)
			} else {
				result.WriteString(token.Value)
			}

		case TokenOpenParen:
			result.WriteString("(")

			if inValuesList {
				result.WriteString("\n")
				result.WriteString(strings.Repeat(" ", (indentLevel+1)*f.indentSize))
			}

		case TokenCloseParen:
			if inValuesList {
				result.WriteString("\n")
				result.WriteString(strings.Repeat(" ", indentLevel*f.indentSize))
			}

			result.WriteString(")")

		case TokenOperator:
			if token.Value == "." {
				// Dots don't need spaces around them
				result.WriteString(token.Value)
			} else {
				if lastToken != nil && lastToken.Type != TokenNewline && lastToken.Value != "." && f.needsSpaceBefore(token.Value) {
					result.WriteString(" ")
				}

				result.WriteString(token.Value)
			}

		default:
			if lastToken != nil && lastToken.Type != TokenNewline && lastToken.Value != "." && lastToken.Type != TokenSnapDirective && f.needsSpaceBefore(token.Value) {
				result.WriteString(" ")
			}

			result.WriteString(token.Value)
		}

		lastToken = &token
	}

	// Clean up extra newlines and ensure proper formatting
	formatted := result.String()
	formatted = f.cleanupFormatting(formatted)

	return formatted
}

// isStatementKeyword checks if a keyword starts a new statement or clause
func (f *SQLFormatter) isStatementKeyword(keyword string) bool {
	statementKeywords := map[string]bool{
		"SELECT": true, "FROM": true, "WHERE": true, "GROUP": true, "HAVING": true,
		"ORDER": true, "LIMIT": true, "OFFSET": true, "INSERT": true, "UPDATE": true,
		"DELETE": true, "CREATE": true, "ALTER": true, "DROP": true,
		"JOIN": true, "INNER": true, "LEFT": true, "RIGHT": true, "FULL": true,
		"SET": true, "ON": true,
	}

	return statementKeywords[keyword]
}

// needsSpaceBefore checks if a token needs a space before it
func (f *SQLFormatter) needsSpaceBefore(value string) bool {
	return value != "(" && value != ")" && value != "," && value != ";" && value != "."
}

// cleanupFormatting cleans up the formatted SQL
func (f *SQLFormatter) cleanupFormatting(sql string) string {
	lines := strings.Split(sql, "\n")

	var cleanedLines []string

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		cleanedLines = append(cleanedLines, trimmed)
	}

	// Join lines and ensure proper spacing
	result := strings.Join(cleanedLines, "\n")

	// Fix common formatting issues
	result = regexp.MustCompile(`\s+,`).ReplaceAllString(result, ",")
	result = regexp.MustCompile(`\(\s+`).ReplaceAllString(result, "(")
	result = regexp.MustCompile(`\s+\)`).ReplaceAllString(result, ")")

	// Remove excessive blank lines
	result = regexp.MustCompile(`\n\s*\n\s*\n`).ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}
