package parser

import (
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/tokenizer"
)

// SqlClause represents the type of SQL clause
type SqlClause int

const (
	UNKNOWN_CLAUSE SqlClause = iota
	WITH_CLAUSE_SECTION
	SELECT_CLAUSE_SECTION
	FROM_CLAUSE_SECTION
	WHERE_CLAUSE_SECTION
	GROUP_BY_CLAUSE_SECTION
	HAVING_CLAUSE_SECTION
	ORDER_BY_CLAUSE_SECTION
	LIMIT_CLAUSE_SECTION
	OFFSET_CLAUSE_SECTION
)

// String returns the string representation of SqlClause
func (c SqlClause) String() string {
	switch c {
	case WITH_CLAUSE_SECTION:
		return "WITH"
	case SELECT_CLAUSE_SECTION:
		return "SELECT"
	case FROM_CLAUSE_SECTION:
		return "FROM"
	case WHERE_CLAUSE_SECTION:
		return "WHERE"
	case GROUP_BY_CLAUSE_SECTION:
		return "GROUP BY"
	case HAVING_CLAUSE_SECTION:
		return "HAVING"
	case ORDER_BY_CLAUSE_SECTION:
		return "ORDER BY"
	case LIMIT_CLAUSE_SECTION:
		return "LIMIT"
	case OFFSET_CLAUSE_SECTION:
		return "OFFSET"
	default:
		return "UNKNOWN"
	}
}

// ValidateDirectiveClauseConstraints validates clause constraints for directives
func ValidateDirectiveClauseConstraints(tokens []tokenizer.Token) []ParseError {
	var errors []ParseError

	// Identify clause position for each token
	clauseMap := buildClauseMap(tokens)

	// Check clause constraints for if/elseif/else/end
	errors = append(errors, validateIfBlockConstraints(tokens, clauseMap)...)

	// Check clause constraints for for/end
	errors = append(errors, validateForBlockConstraints(tokens, clauseMap)...)

	return errors
}

// buildClauseMap builds a map of which clause each token belongs to
func buildClauseMap(tokens []tokenizer.Token) map[int]SqlClause {
	clauseMap := make(map[int]SqlClause)
	currentClause := UNKNOWN_CLAUSE

	for i, token := range tokens {
		// Skip whitespace tokens and detect clause boundaries
		if token.Type == tokenizer.WHITESPACE {
			clauseMap[i] = currentClause
			continue
		}

		// Detect clause boundaries
		switch token.Type {
		case tokenizer.WITH:
			currentClause = WITH_CLAUSE_SECTION
		case tokenizer.SELECT:
			currentClause = SELECT_CLAUSE_SECTION
		case tokenizer.FROM:
			currentClause = FROM_CLAUSE_SECTION
		case tokenizer.WHERE:
			currentClause = WHERE_CLAUSE_SECTION
		case tokenizer.GROUP:
			// Detect GROUP BY (check if next token is BY)
			if i+1 < len(tokens) {
				nextNonWhitespace := findNextNonWhitespaceToken(tokens, i+1)
				if nextNonWhitespace != -1 && tokens[nextNonWhitespace].Type == tokenizer.BY {
					currentClause = GROUP_BY_CLAUSE_SECTION
				}
			}
		case tokenizer.HAVING:
			currentClause = HAVING_CLAUSE_SECTION
		case tokenizer.ORDER:
			// Detect ORDER BY (check if next token is BY)
			if i+1 < len(tokens) {
				nextNonWhitespace := findNextNonWhitespaceToken(tokens, i+1)
				if nextNonWhitespace != -1 && tokens[nextNonWhitespace].Type == tokenizer.BY {
					currentClause = ORDER_BY_CLAUSE_SECTION
				}
			}
		case tokenizer.WORD:
			// LIMIT, OFFSET の検出
			upperValue := strings.ToUpper(token.Value)
			switch upperValue {
			case "LIMIT":
				currentClause = LIMIT_CLAUSE_SECTION
			case "OFFSET":
				currentClause = OFFSET_CLAUSE_SECTION
			}
		}

		clauseMap[i] = currentClause
	}

	return clauseMap
}

// findNextNonWhitespaceToken は指定されたインデックス以降の最初の非空白トークンのインデックスを返す
func findNextNonWhitespaceToken(tokens []tokenizer.Token, startIndex int) int {
	for i := startIndex; i < len(tokens); i++ {
		if tokens[i].Type != tokenizer.WHITESPACE {
			return i
		}
	}
	return -1
}

// isWithinWithClause determines whether to relax clause constraints within WITH clause
func isWithinWithClause(startClause, _ SqlClause) bool {
	// Allow if starting from WITH clause and ending in other clauses
	// Or allow if starting from UNKNOWN clause and ending in other clauses within WITH clause
	if startClause == WITH_CLAUSE_SECTION || startClause == UNKNOWN_CLAUSE {
		return true
	}
	return false
}

// validateIfBlockConstraints validates clause constraints for if statements
func validateIfBlockConstraints(tokens []tokenizer.Token, clauseMap map[int]SqlClause) []ParseError {
	errors := make([]ParseError, 0, len(tokens))
	ifStack := make([]IfBlockInfo, 0, len(tokens))

	for i, token := range tokens {
		if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
			clause := clauseMap[i]

			switch token.DirectiveType {
			case "if":
				// if文の開始
				ifStack = append(ifStack, IfBlockInfo{
					StartIndex:  i,
					StartClause: clause,
					Position:    token.Position,
				})

			case "elseif", "else":
				// elseif/else文
				if len(ifStack) == 0 {
					errors = append(errors, ParseError{
						Message:  fmt.Sprintf("%s without matching if", token.DirectiveType),
						Position: token.Position,
						Token:    token,
						Severity: ERROR,
					})
					continue
				}

				currentIf := &ifStack[len(ifStack)-1]
				// Relax clause constraints within WITH clause
				if clause != currentIf.StartClause && !isWithinWithClause(currentIf.StartClause, clause) {
					errors = append(errors, ParseError{
						Message: fmt.Sprintf("SnapSQL directive spans multiple SQL clauses: 'if' starts in %s clause at line %d, '%s' found in %s clause at line %d",
							currentIf.StartClause, currentIf.Position.Line, token.DirectiveType, clause, token.Position.Line),
						Position: token.Position,
						Token:    token,
						Severity: ERROR,
					})
				}

			case "end":
				// end文
				if len(ifStack) == 0 {
					// Might be end of for statement, so don't error here
					continue
				}

				currentIf := ifStack[len(ifStack)-1]
				// Relax clause constraints within WITH clause
				if clause != currentIf.StartClause && !isWithinWithClause(currentIf.StartClause, clause) {
					errors = append(errors, ParseError{
						Message: fmt.Sprintf("SnapSQL directive spans multiple SQL clauses: 'if' starts in %s clause at line %d, 'end' found in %s clause at line %d",
							currentIf.StartClause, currentIf.Position.Line, clause, token.Position.Line),
						Position: token.Position,
						Token:    token,
						Severity: ERROR,
					})
				}

				// Remove from if stack
				ifStack = ifStack[:len(ifStack)-1]
			}
		}
	}

	// 未閉じのif文をチェック
	for _, ifInfo := range ifStack {
		errors = append(errors, ParseError{
			Message:  "unclosed if directive",
			Position: ifInfo.Position,
			Token:    tokens[ifInfo.StartIndex],
			Severity: ERROR,
		})
	}

	return errors
}

// validateForBlockConstraints validates clause constraints for loop statements
func validateForBlockConstraints(tokens []tokenizer.Token, clauseMap map[int]SqlClause) []ParseError {
	errors := make([]ParseError, 0, len(tokens))
	forStack := make([]ForBlockInfo, 0, len(tokens))

	for i, token := range tokens {
		if token.Type == tokenizer.BLOCK_COMMENT && token.IsSnapSQLDirective {
			clause := clauseMap[i]

			switch token.DirectiveType {
			case "for":
				// for文の開始
				forStack = append(forStack, ForBlockInfo{
					StartIndex:  i,
					StartClause: clause,
					Position:    token.Position,
				})

			case "end":
				// end文（for文の終了の可能性）
				if len(forStack) > 0 {
					currentFor := forStack[len(forStack)-1]
					// Relax clause constraints within WITH clause
					if clause != currentFor.StartClause && !isWithinWithClause(currentFor.StartClause, clause) {
						errors = append(errors, ParseError{
							Message: fmt.Sprintf("SnapSQL directive spans multiple SQL clauses: 'for' starts in %s clause at line %d, 'end' found in %s clause at line %d",
								currentFor.StartClause, currentFor.Position.Line, clause, token.Position.Line),
							Position: token.Position,
							Token:    token,
							Severity: ERROR,
						})
					}

					// Remove from for stack
					forStack = forStack[:len(forStack)-1]
				}
			}
		}
	}

	// 未閉じのfor文をチェック
	for _, forInfo := range forStack {
		errors = append(errors, ParseError{
			Message:  "unclosed for directive",
			Position: forInfo.Position,
			Token:    tokens[forInfo.StartIndex],
			Severity: ERROR,
		})
	}

	return errors
}

// IfBlockInfo はif文ブロックの情報を保持する
type IfBlockInfo struct {
	StartIndex  int
	StartClause SqlClause
	Position    tokenizer.Position
}

// ForBlockInfo はfor文ブロックの情報を保持する
type ForBlockInfo struct {
	StartIndex  int
	StartClause SqlClause
	Position    tokenizer.Position
}
