package intermediate

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddSystemFieldsToUpdateTokens_WithCTE(t *testing.T) {
	sql := `WITH done_stage AS (
		SELECT stage_order AS stage_limit
		FROM lists
		WHERE board_id = 1
	), new_list AS (
		SELECT id AS new_list_id
		FROM lists
		WHERE board_id = 2
	)
	UPDATE cards
	SET list_id = 10
	WHERE list_id IN (1, 2)`

	stmt, _, err := parser.ParseSQLFile(strings.NewReader(sql), nil, "", "", parser.DefaultOptions)
	require.NoError(t, err)

	tokens := extractTokensFromStatement(stmt)

	transformer := TokenTransformer{}
	implicit := []ImplicitParameter{{Name: "updated_at", Type: "timestamp"}}

	out := transformer.addSystemFieldsToUpdateTokens(tokens, implicit)

	directiveIndex := -1

	for i, tok := range out {
		if tok.Type == tokenizer.BLOCK_COMMENT && tok.Directive != nil && tok.Directive.Type == "system_value" {
			directiveIndex = i

			assert.Equal(t, "updated_at", tok.Directive.SystemField)
			assert.Contains(t, tok.Value, "EMIT_SYSTEM_VALUE: updated_at")

			break
		}
	}

	require.NotEqual(t, -1, directiveIndex, "system_value directive not injected")

	for _, tok := range out {
		require.NotContains(t, tok.Value, "/*= updated_at */")
	}

	parenDepth := 0
	sawUpdate := false
	whereIndex := -1

findWhere:
	for i, tok := range out {
		switch tok.Type {
		case tokenizer.OPENED_PARENS:
			parenDepth++
		case tokenizer.CLOSED_PARENS:
			if parenDepth > 0 {
				parenDepth--
			}
		case tokenizer.UPDATE:
			if parenDepth == 0 {
				sawUpdate = true
			}
		case tokenizer.WHERE:
			if sawUpdate && parenDepth == 0 {
				whereIndex = i
				break findWhere
			}
		}
	}

	require.NotEqual(t, -1, whereIndex, "top-level WHERE not found")
	assert.Less(t, directiveIndex, whereIndex, "system_value directive must precede top-level WHERE")

	require.GreaterOrEqual(t, directiveIndex, 3)
	assert.Equal(t, tokenizer.COMMA, out[directiveIndex-3].Type)
	assert.Equal(t, tokenizer.IDENTIFIER, out[directiveIndex-2].Type)
	assert.Equal(t, "updated_at", out[directiveIndex-2].Value)
	assert.Equal(t, tokenizer.EQUAL, out[directiveIndex-1].Type)

	for i := whereIndex + 1; i < len(out); i++ {
		assert.NotEqual(t, tokenizer.BLOCK_COMMENT, out[i].Type)
	}
}

func TestAddSystemFieldsToUpdateTokens_ReplacesExistingAssignment(t *testing.T) {
	sql := `UPDATE cards SET updated_at = CURRENT_TIMESTAMP, list_id = 1 WHERE id = 10`

	stmt, _, err := parser.ParseSQLFile(strings.NewReader(sql), nil, "", "", parser.DefaultOptions)
	require.NoError(t, err)

	tokens := extractTokensFromStatement(stmt)

	transformer := TokenTransformer{}
	implicit := []ImplicitParameter{{Name: "updated_at", Type: "timestamp"}}

	out := transformer.addSystemFieldsToUpdateTokens(tokens, implicit)

	countUpdatedAt := 0

	for _, tok := range out {
		if tok.Type == tokenizer.IDENTIFIER || tok.Type == tokenizer.RESERVED_IDENTIFIER || tok.Type == tokenizer.CONTEXTUAL_IDENTIFIER {
			if strings.EqualFold(strings.TrimSpace(tok.Value), "updated_at") {
				countUpdatedAt++
			}
		}

		assert.NotContains(t, tok.Value, "CURRENT_TIMESTAMP")
	}

	assert.Equal(t, 1, countUpdatedAt, "updated_at should appear exactly once in SET clause")

	foundDirective := false

	for _, tok := range out {
		if tok.Type == tokenizer.BLOCK_COMMENT && tok.Directive != nil && tok.Directive.Type == "system_value" {
			foundDirective = true

			assert.Equal(t, "updated_at", tok.Directive.SystemField)

			break
		}
	}

	assert.True(t, foundDirective, "existing assignment should be replaced with system_value directive")
}
