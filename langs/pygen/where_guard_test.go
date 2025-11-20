package pygen

import (
	"strings"
	"testing"

	"github.com/shibukawa/snapsql/intermediate"
)

func TestConvertWhereMeta(t *testing.T) {
	tests := []struct {
		name     string
		input    *intermediate.WhereClauseMeta
		expected *whereClauseMetaData
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "fullscan status",
			input: &intermediate.WhereClauseMeta{
				Status: "fullscan",
			},
			expected: &whereClauseMetaData{
				Status:         "fullscan",
				ExpressionRefs: []int{},
			},
		},
		{
			name: "exists status",
			input: &intermediate.WhereClauseMeta{
				Status: "exists",
			},
			expected: &whereClauseMetaData{
				Status:         "exists",
				ExpressionRefs: []int{},
			},
		},
		{
			name: "conditional with dynamic conditions",
			input: &intermediate.WhereClauseMeta{
				Status: "conditional",
				DynamicConditions: []intermediate.WhereDynamicCondition{
					{
						ExprIndex:        0,
						NegatedWhenEmpty: true,
						HasElse:          false,
						Description:      "user_id filter",
					},
				},
			},
			expected: &whereClauseMetaData{
				Status:         "conditional",
				ExpressionRefs: []int{},
				DynamicConditions: []whereDynamicConditionData{
					{
						ExprIndex:        0,
						NegatedWhenEmpty: true,
						HasElse:          false,
						Description:      "user_id filter",
					},
				},
			},
		},
		{
			name: "with removal combos",
			input: &intermediate.WhereClauseMeta{
				Status: "conditional",
				RemovalCombos: [][]intermediate.RemovalLiteral{
					{
						{ExprIndex: 0, When: false},
						{ExprIndex: 1, When: false},
					},
				},
			},
			expected: &whereClauseMetaData{
				Status:         "conditional",
				ExpressionRefs: []int{},
				RemovalCombos: [][]removalLiteralData{
					{
						{ExprIndex: 0, When: false},
						{ExprIndex: 1, When: false},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertWhereMeta(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}

				return
			}

			if result == nil {
				t.Fatalf("expected non-nil result")
			}

			if result.Status != tt.expected.Status {
				t.Errorf("Status: expected %q, got %q", tt.expected.Status, result.Status)
			}

			if len(result.DynamicConditions) != len(tt.expected.DynamicConditions) {
				t.Errorf("DynamicConditions length: expected %d, got %d",
					len(tt.expected.DynamicConditions), len(result.DynamicConditions))
			}

			if len(result.RemovalCombos) != len(tt.expected.RemovalCombos) {
				t.Errorf("RemovalCombos length: expected %d, got %d",
					len(tt.expected.RemovalCombos), len(result.RemovalCombos))
			}
		})
	}
}

func TestGetMutationKind(t *testing.T) {
	tests := []struct {
		statementType string
		expected      string
	}{
		{"update", "MutationUpdate"},
		{"UPDATE", "MutationUpdate"},
		{"delete", "MutationDelete"},
		{"DELETE", "MutationDelete"},
		{"select", ""},
		{"insert", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.statementType, func(t *testing.T) {
			result := getMutationKind(tt.statementType)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsMutationStatement(t *testing.T) {
	tests := []struct {
		statementType string
		expected      bool
	}{
		{"update", true},
		{"UPDATE", true},
		{"delete", true},
		{"DELETE", true},
		{"select", false},
		{"insert", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.statementType, func(t *testing.T) {
			result := isMutationStatement(tt.statementType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateWhereGuardCode(t *testing.T) {
	expressions := []intermediate.CELExpression{
		{Expression: "include_identifier_filter"},
		{Expression: "prefer_name_filter"},
	}

	tests := []struct {
		name        string
		whereMeta   *whereClauseMetaData
		expect      []string
		expectEmpty bool
	}{
		{
			name:        "nil metadata",
			whereMeta:   nil,
			expectEmpty: true,
		},
		{
			name: "fullscan status",
			whereMeta: &whereClauseMetaData{
				Status: "fullscan",
			},
			expect: []string{
				"# WHERE clause guard for delete",
				"if True:",
				"if not ctx.allow_unsafe_mutations:",
				"UnsafeQueryError",
			},
		},
		{
			name: "dynamic conditional guard",
			whereMeta: &whereClauseMetaData{
				Status: "conditional",
				DynamicConditions: []whereDynamicConditionData{
					{
						ExprIndex:        0,
						NegatedWhenEmpty: true,
						Description:      "include_identifier_filter",
					},
				},
			},
			expect: []string{
				"not (include_identifier_filter)",
				"UnsafeQueryError",
			},
		},
		{
			name: "removal combo guard",
			whereMeta: &whereClauseMetaData{
				Status: "conditional",
				RemovalCombos: [][]removalLiteralData{
					{
						{ExprIndex: 0, When: false},
						{ExprIndex: 1, When: true},
					},
				},
			},
			expect: []string{
				"if not (include_identifier_filter) and prefer_name_filter:",
				"raise UnsafeQueryError",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateWhereGuardCode("AccountDelete", "MutationDelete", tt.whereMeta, expressions)

			if tt.expectEmpty {
				if code != "" {
					t.Errorf("expected empty guard, got %q", code)
				}

				return
			}

			if code == "" {
				t.Fatalf("expected guard code, got empty string")
			}

			for _, substr := range tt.expect {
				if !strings.Contains(code, substr) {
					t.Errorf("expected guard to contain %q, got %q", substr, code)
				}
			}
		})
	}
}

func TestDescribeDynamicConditions(t *testing.T) {
	tests := []struct {
		name            string
		conditions      []whereDynamicConditionData
		filterRemovable bool
		expected        string
	}{
		{
			name:            "empty conditions",
			conditions:      []whereDynamicConditionData{},
			filterRemovable: false,
			expected:        "",
		},
		{
			name: "single condition",
			conditions: []whereDynamicConditionData{
				{
					ExprIndex:        0,
					NegatedWhenEmpty: true,
					HasElse:          false,
					Description:      "user_id filter",
				},
			},
			filterRemovable: false,
			expected:        "expr[0] user_id filter",
		},
		{
			name: "multiple conditions",
			conditions: []whereDynamicConditionData{
				{
					ExprIndex:        0,
					NegatedWhenEmpty: true,
					HasElse:          false,
					Description:      "user_id filter",
				},
				{
					ExprIndex:        1,
					NegatedWhenEmpty: true,
					HasElse:          false,
					Description:      "status filter",
				},
			},
			filterRemovable: false,
			expected:        "expr[0] user_id filter, expr[1] status filter",
		},
		{
			name: "filter removable conditions",
			conditions: []whereDynamicConditionData{
				{
					ExprIndex:        0,
					NegatedWhenEmpty: true,
					HasElse:          false,
					Description:      "user_id filter",
				},
				{
					ExprIndex:        1,
					NegatedWhenEmpty: false,
					HasElse:          false,
					Description:      "status filter",
				},
			},
			filterRemovable: true,
			expected:        "expr[0] user_id filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := describeDynamicConditions(tt.conditions, tt.filterRemovable)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
