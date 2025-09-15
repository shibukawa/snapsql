package intermediate

import (
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// ReturningProcessor removes unsupported RETURNING clauses at token level before instruction generation.
// Policy:
//
//	INSERT: always keep
//	UPDATE: keep only for postgres, sqlite
//	DELETE: keep for postgres, sqlite, mariadb (mysql only not supported)
//
// Rationale: We currently don't mutate parser AST clauses (baseStatement fields unexported), and instruction
// generation consumes raw tokens. Therefore we filter tokens directly to avoid emitting unsupported SQL.
type ReturningProcessor struct{}

func (p *ReturningProcessor) Name() string { return "ReturningProcessor" }

func (p *ReturningProcessor) Process(ctx *ProcessingContext) error {
	if ctx == nil || len(ctx.Tokens) == 0 {
		return nil
	}

	dialect := ctx.Dialect

	// Quick dialect capability predicates
	updateSupported := (dialect == "postgres" || dialect == "sqlite")
	deleteSupported := (dialect == "postgres" || dialect == "sqlite" || dialect == "mariadb")

	// We must know the statement type to decide removal. Determine first non-whitespace keyword.
	stmtType := detectStatementType(ctx.Tokens)

	if stmtType != stmtUpdate && stmtType != stmtDelete {
		// Nothing to filter (either always supported or not relevant)
		return nil
	}

	// If current dialect supports this statement's RETURNING, skip filtering.
	if (stmtType == stmtUpdate && updateSupported) || (stmtType == stmtDelete && deleteSupported) {
		return nil
	}

	// Remove RETURNING clause tokens: pattern
	//   RETURNING <any tokens until statement terminator or end>
	// Simplicity: once we encounter RETURNING keyword at top-level (parenDepth==0), drop it and all following tokens.
	// This assumes RETURNING appears at the tail of UPDATE/DELETE in supported syntax (which is standard).
	filtered := make([]tok.Token, 0, len(ctx.Tokens))
	parenDepth := 0
	skipping := false

	for i := range len(ctx.Tokens) {
		tk := ctx.Tokens[i]
		if tk.Type == tok.OPENED_PARENS {
			parenDepth++
		} else if tk.Type == tok.CLOSED_PARENS && parenDepth > 0 {
			parenDepth--
		}

		if skipping {
			// Once skipping, continue skipping to end (RETURNING is tail)
			continue
		}

		if tk.Type == tok.RETURNING && parenDepth == 0 {
			skipping = true
			continue
		}

		filtered = append(filtered, tk)
	}

	ctx.Tokens = filtered

	return nil
}

// Internal statement type identifiers for lightweight detection.
const (
	stmtUnknown = iota
	stmtInsert
	stmtUpdate
	stmtDelete
)

// detectStatementType finds the leading DML keyword ignoring whitespace and comments.
func detectStatementType(tokens []tok.Token) int {
	for _, tk := range tokens {
		switch tk.Type {
		case tok.WHITESPACE, tok.LINE_COMMENT, tok.BLOCK_COMMENT:
			continue
		case tok.INSERT:
			return stmtInsert
		case tok.UPDATE:
			return stmtUpdate
		case tok.DELETE:
			return stmtDelete
		default:
			return stmtUnknown
		}
	}

	return stmtUnknown
}

var _ TokenProcessor = (*ReturningProcessor)(nil)
