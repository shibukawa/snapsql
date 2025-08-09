package parserstep5

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/tokenizer"
)

// parseFullPipeline executes the full parsing pipeline from SQL string to parserstep3 result
func parseFullPipeline(t *testing.T, sql string) cmn.StatementNode {
	t.Helper()

	// Step 1: Tokenize
	tokens, err := tokenizer.Tokenize(sql)
	assert.NoError(t, err, "tokenization failed")

	// Step 2: Basic parsing
	step2Result, err := parserstep2.Execute(tokens)
	assert.NoError(t, err, "parserstep2 failed")

	// Step 3: Clause parsing
	err = parserstep3.Execute(step2Result)
	assert.NoError(t, err, "parserstep3 failed")

	err = parserstep4.Execute(step2Result)
	assert.NoError(t, err, "parserstep4 failed")

	return step2Result
}

// DirectiveCheck represents expected directive properties
type DirectiveCheck struct {
	index         int
	directiveType string
	nextIndex     *int // nil if not checked, otherwise expected NextIndex
}

func TestValidateAndLinkDirectives(t *testing.T) {
	testCases := []struct {
		name            string
		sql             string
		shouldSucceed   bool
		expectedError   string
		clauseType      cmn.NodeType
		expectedCount   int
		directiveChecks []DirectiveCheck
	}{
		// Success cases
		{
			name: "SimpleIfEnd",
			sql: `SELECT id, name 
				FROM users 
				WHERE active = true
					/*# if filters.department */
					AND department = /*= filters.department */'sales'
					/*# end */`,
			shouldSucceed: true,
			clauseType:    cmn.WHERE_CLAUSE,
			expectedCount: 2,
			directiveChecks: []DirectiveCheck{
				{index: 0, directiveType: "if"},
				{index: 1, directiveType: "end"},
			},
		},
		{
			name: "IfCoveringWhereClause: clause covers if is removed at parser step2",
			sql: `SELECT id, name, email FROM users /*# if filters.active */
WHERE active = /*= filters.active */true /*# end */`,
			shouldSucceed: true,
			clauseType:    cmn.FROM_CLAUSE,
			expectedCount: 0,
		},
		{
			name: "EndInWhereClause: clause covers if is removed at parser step2",
			sql: `SELECT id, name, email FROM users /*# if filters.active */
WHERE active = /*= filters.active */true /*# end */`,
			shouldSucceed: true,
			clauseType:    cmn.WHERE_CLAUSE,
			expectedCount: 0,
		},
		{
			name: "ForEnd",
			sql: `SELECT id, name,
					/*# for field : additional_fields */
					/*= field */,
					/*# end */
					created_at
				FROM users`,
			shouldSucceed: true,
			clauseType:    cmn.SELECT_CLAUSE,
			expectedCount: 2,
			directiveChecks: []DirectiveCheck{
				{index: 0, directiveType: "for"},
				{index: 1, directiveType: "end"},
			},
		},
		{
			name: "IfElseIfElseEnd",
			sql: `SELECT id, name
				FROM users
				WHERE 1=1
					/*# if filters.status == 'active' */
					AND status = 'active'
					/*# elseif filters.status == 'inactive' */
					AND status = 'inactive'
					/*# else */
					AND status IS NOT NULL
					/*# end */`,
			shouldSucceed: true,
			clauseType:    cmn.WHERE_CLAUSE,
			expectedCount: 4,
			directiveChecks: []DirectiveCheck{
				{index: 0, directiveType: "if"},
				{index: 1, directiveType: "elseif"},
				{index: 2, directiveType: "else"},
				{index: 3, directiveType: "end"},
			},
		},
		{
			name: "NestedBlocks",
			sql: `SELECT id, name
				FROM users
				WHERE 1=1
					/*# if filters.department */
					AND department = /*= filters.department */'sales'
						/*# if filters.active */
						AND active = /*= filters.active */true
						/*# end */
					/*# end */`,
			shouldSucceed: true,
			clauseType:    cmn.WHERE_CLAUSE,
			expectedCount: 4,
			directiveChecks: []DirectiveCheck{
				{index: 0, directiveType: "if"},
				{index: 1, directiveType: "if"},
				{index: 2, directiveType: "end"},
				{index: 3, directiveType: "end"},
			},
		},
		// Error cases
		{
			name: "UnmatchedIf",
			sql: `SELECT id, name
				FROM users
				WHERE active = true
					/*# if filters.department */
					AND department = /*= filters.department */'sales'`,
			shouldSucceed: false,
			expectedError: "unclosed",
		},
		{
			name: "UnmatchedFor",
			sql: `SELECT id, name,
					/*# for field : additional_fields */
					/*= field */,
				FROM users`,
			shouldSucceed: false,
			expectedError: "unclosed",
		},
		{
			name: "ExcessiveEnd",
			sql: `SELECT id, name
				FROM users
				WHERE active = true
					/*# end */`,
			shouldSucceed: false,
			expectedError: "unexpected",
		},
		{
			name: "InvalidElseifOrder",
			sql: `SELECT id, name
				FROM users
				WHERE active = true
					/*# elseif filters.department */
					AND department = /*= filters.department */'sales'
					/*# end */`,
			shouldSucceed: false,
			expectedError: "unexpected",
		},
		{
			name: "InvalidElseOrder",
			sql: `SELECT id, name
				FROM users
				WHERE active = true
					/*# else */
					AND department IS NOT NULL
					/*# end */`,
			shouldSucceed: false,
			expectedError: "unexpected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := parseFullPipeline(t, tc.sql)

			var parseErr cmn.ParseError

			// Debug for IfCoveringWhereClause test
			if tc.name == "IfCoveringWhereClause" {
				t.Log("Debugging IfCoveringWhereClause test")

				for _, clause := range stmt.Clauses() {
					t.Logf("Clause type: %s", clause.Type())
					debugTokens(t, clause.ContentTokens())
				}
			}

			validateAndLinkDirectives(stmt, &parseErr)

			if tc.shouldSucceed {
				assert.Equal(t, 0, len(parseErr.Errors), "validation should succeed for %s: %v", tc.name, parseErr.Errors)

				// Check directive structure if specified
				if tc.clauseType != 0 && tc.expectedCount > 0 {
					clause := findClauseByType(t, stmt, tc.clauseType)
					assert.True(t, clause != nil, "%s clause should exist", tc.clauseType)

					directives := extractDirectiveTokens(t, clause.ContentTokens())
					assert.Equal(t, tc.expectedCount, len(directives), "should have %d directive tokens", tc.expectedCount)

					// Check individual directives
					for _, check := range tc.directiveChecks {
						if check.index < len(directives) {
							assert.Equal(t, check.directiveType, directives[check.index].Directive.Type,
								"directive at index %d should be %s", check.index, check.directiveType)

							if check.nextIndex != nil {
								assert.Equal(t, *check.nextIndex, directives[check.index].Directive.NextIndex,
									"directive at index %d should link to index %d", check.index, *check.nextIndex)
							}
						}
					}
				}
			} else {
				assert.True(t, len(parseErr.Errors) > 0, "validation should fail for %s", tc.name)
				errorText := parseErr.Error()
				assert.Contains(t, errorText, tc.expectedError, "error should mention '%s'", tc.expectedError)
			}
		})
	}
}

// TestValidateAndLinkDirectives_MultipleErrors tests that multiple errors are collected
func TestValidateAndLinkDirectives_MultipleErrors(t *testing.T) {
	multiErrorTestCases := []struct {
		name           string
		sql            string
		expectedErrors []string
	}{
		{
			name: "MultipleUnmatchedBlocks",
			sql: `SELECT id, name
				FROM users
				WHERE 1=1
					/*# if filters.department */
					AND department = 'sales'
					/*# for field : fields */
					AND field IS NOT NULL`,
			expectedErrors: []string{"unclosed", "unclosed"},
		},
		{
			name: "MultipleUnexpectedDirectives",
			sql: `SELECT id, name
				FROM users
				WHERE 1=1
					/*# elseif filters.department */
					AND department = 'sales'
					/*# else */
					AND status IS NOT NULL
					/*# end */
					/*# end */`,
			expectedErrors: []string{"unexpected", "unexpected", "unexpected", "unexpected"},
		},
		{
			name: "MixedErrors",
			sql: `SELECT id, name
				FROM users
				WHERE 1=1
					/*# if filters.status */
					AND status = 'active'
					/*# elseif filters.department */
					AND department = 'sales'
					/*# for field : fields */
					AND field IS NOT NULL
					/*# else */
					AND 1=1`,
			expectedErrors: []string{"unexpected", "unclosed", "unclosed"},
		},
	}

	for _, tc := range multiErrorTestCases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := parseFullPipeline(t, tc.sql)

			var parseErr cmn.ParseError
			validateAndLinkDirectives(stmt, &parseErr)

			assert.Equal(t, len(tc.expectedErrors), len(parseErr.Errors),
				"should have %d errors, got %d: %v", len(tc.expectedErrors), len(parseErr.Errors), parseErr.Errors)

			errorText := parseErr.Error()
			for i, expectedError := range tc.expectedErrors {
				assert.Contains(t, errorText, expectedError,
					"error %d should mention '%s'", i, expectedError)
			}
		})
	}
}

// TestValidateAndLinkDirectives_NextIndexLinking tests NextIndex linking specifically
func TestValidateAndLinkDirectives_NextIndexLinking(t *testing.T) {
	linkingTestCases := []struct {
		name           string
		sql            string
		clauseType     cmn.NodeType
		expectedChains [][]int // indices of directive tokens that should be linked in order
	}{
		{
			name: "SimpleIfEndLinking",
			sql: `SELECT id, name 
				FROM users 
				WHERE active = true
					/*# if filters.department */
					AND department = /*= filters.department */'sales'
					/*# end */`,
			clauseType:     cmn.WHERE_CLAUSE,
			expectedChains: [][]int{{0, 1}}, // if(0) -> end(1)
		},
		{
			name: "IfElseIfElseEndLinking",
			sql: `SELECT id, name
				FROM users
				WHERE 1=1
					/*# if filters.status == 'active' */
					AND status = 'active'
					/*# elseif filters.status == 'inactive' */
					AND status = 'inactive'
					/*# else */
					AND status IS NOT NULL
					/*# end */`,
			clauseType:     cmn.WHERE_CLAUSE,
			expectedChains: [][]int{{0, 1, 2, 3}}, // if(0) -> elseif(1) -> else(2) -> end(3)
		},
		{
			name: "NestedBlocksLinking",
			sql: `SELECT id, name
				FROM users
				WHERE 1=1
					/*# if filters.department */
					AND department = /*= filters.department */'sales'
						/*# if filters.active */
						AND active = /*= filters.active */true
						/*# end */
					/*# end */`,
			clauseType: cmn.WHERE_CLAUSE,
			expectedChains: [][]int{
				{0, 3}, // outer if(0) -> outer end(3)
				{1, 2}, // inner if(1) -> inner end(2)
			},
		},
	}

	for _, tc := range linkingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := parseFullPipeline(t, tc.sql)

			var parseErr cmn.ParseError
			validateAndLinkDirectives(stmt, &parseErr)
			assert.Equal(t, 0, len(parseErr.Errors), "validation should succeed: %v", parseErr.Errors)

			clause := findClauseByType(t, stmt, tc.clauseType)
			assert.True(t, clause != nil, "clause should exist")

			directives := extractDirectiveTokens(t, clause.ContentTokens())

			// Check each expected chain
			for chainIndex, chain := range tc.expectedChains {
				for i := range len(chain) - 1 {
					fromIndex := chain[i]
					toIndex := chain[i+1]

					assert.True(t, fromIndex < len(directives) && toIndex < len(directives),
						"chain %d: indices should be valid", chainIndex)

					expectedNextIndex := directives[toIndex].Index
					actualNextIndex := directives[fromIndex].Directive.NextIndex

					assert.Equal(t, expectedNextIndex, actualNextIndex,
						"chain %d: directive at %d should link to directive at %d (Index %d)",
						chainIndex, fromIndex, toIndex, expectedNextIndex)
				}
			}
		})
	}
}

// Helper functions for testing

// findClauseByType finds a clause of the specified type in a statement
func findClauseByType(t *testing.T, stmt cmn.StatementNode, clauseType cmn.NodeType) cmn.ClauseNode {
	t.Helper()

	for _, clause := range stmt.Clauses() {
		if clause.Type() == clauseType {
			return clause
		}
	}

	return nil
}

// debugTokens prints token information for debugging
func debugTokens(t *testing.T, tokens []tokenizer.Token) {
	t.Helper()
	t.Logf("Total tokens: %d", len(tokens))

	for i, token := range tokens {
		if token.Directive != nil {
			t.Logf("Token[%d]: Type=%s, Value=%s, Directive.Type=%s",
				i, token.Type, token.Value, token.Directive.Type)
		} else {
			t.Logf("Token[%d]: Type=%s, Value=%s", i, token.Type, token.Value)
		}
	}
}

func extractDirectiveTokens(t *testing.T, tokens []tokenizer.Token) []tokenizer.Token {
	t.Helper()

	var directives []tokenizer.Token

	for _, token := range tokens {
		if token.Directive != nil && IsControlFlowDirective(token.Directive.Type) {
			directives = append(directives, token)
		}
	}

	return directives
}

// TestValidateAndLinkDirectives_ParenthesesBoundary tests parentheses boundary checking
func TestValidateAndLinkDirectives_ParenthesesBoundary(t *testing.T) {
	parenthesesTestCases := []struct {
		name          string
		sql           string
		shouldSucceed bool
		expectedError string
	}{
		{
			name:          "ValidParenthesesContainment",
			sql:           `SELECT id, name FROM users WHERE active = /*# if filters.active */ true /*# end */`,
			shouldSucceed: true,
		},
		{
			name:          "ValidNestedParentheses",
			sql:           `SELECT id FROM users WHERE status IN /*# if filters.active */ ('active', 'pending') /*# end */`,
			shouldSucceed: true,
		},
		{
			name:          "InvalidIfCrossesParentheses",
			sql:           `SELECT id FROM users WHERE (active = /*# if filters.active */ true AND status = 'active') /*# end */`,
			shouldSucceed: false,
			expectedError: "crosses parentheses boundary",
		},
		{
			name:          "InvalidForCrossesParentheses",
			sql:           `SELECT id FROM users WHERE (/*# for item in items */ item.active = true) AND status = 'active' /*# end */`,
			shouldSucceed: false,
			expectedError: "crosses parentheses boundary",
		},
		{
			name:          "InvalidEndCrossesParentheses",
			sql:           `SELECT id FROM users WHERE (active = /*# if filters.active */ true /*# end */) AND status = 'pending'`,
			shouldSucceed: true, // This should be valid as both directives are within the same parentheses level
		},
		{
			name:          "InvalidDirectiveCrossesParentheses",
			sql:           `SELECT id FROM users WHERE active /*# if filters.active */ = (true /*# end */)`,
			shouldSucceed: false,
			expectedError: "crosses parentheses boundary",
		},
	}

	for _, tc := range parenthesesTestCases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := parseFullPipeline(t, tc.sql)

			var parseErr cmn.ParseError
			validateAndLinkDirectives(stmt, &parseErr)

			t.Logf("Test case: %s, Errors count: %d", tc.name, len(parseErr.Errors))

			if len(parseErr.Errors) > 0 {
				t.Logf("Errors: %v", parseErr.Errors)
			}

			if tc.shouldSucceed {
				assert.Equal(t, 0, len(parseErr.Errors), "validation should succeed for %s: %v", tc.name, parseErr.Errors)
			} else {
				assert.True(t, len(parseErr.Errors) > 0, "validation should fail for %s", tc.name)
				errorText := parseErr.Error()
				t.Logf("Error for %s: %s", tc.name, errorText)
				assert.Contains(t, errorText, tc.expectedError, "error should mention '%s'", tc.expectedError)
			}
		})
	}
}
