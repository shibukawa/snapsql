package pygen

import (
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
	tests := []struct {
		name         string
		funcName     string
		mutationKind string
		whereMeta    *whereClauseMetaData
		expectCode   bool
	}{
		{
			name:         "no mutation kind",
			funcName:     "get_users",
			mutationKind: "",
			whereMeta:    nil,
			expectCode:   false,
		},
		{
			name:         "no where meta",
			funcName:     "update_user",
			mutationKind: "MutationUpdate",
			whereMeta:    nil,
			expectCode:   false,
		},
		{
			name:         "fullscan status",
			funcName:     "update_user",
			mutationKind: "MutationUpdate",
			whereMeta: &whereClauseMetaData{
				Status: "fullscan",
			},
			expectCode: true,
		},
		{
			name:         "exists status",
			funcName:     "delete_user",
			mutationKind: "MutationDelete",
			whereMeta: &whereClauseMetaData{
				Status: "exists",
			},
			expectCode: true,
		},
		{
			name:         "conditional with dynamic conditions",
			funcName:     "update_user",
			mutationKind: "MutationUpdate",
			whereMeta: &whereClauseMetaData{
				Status: "conditional",
				DynamicConditions: []whereDynamicConditionData{
					{
						ExprIndex:        0,
						NegatedWhenEmpty: true,
						HasElse:          false,
						Description:      "user_id filter",
					},
				},
			},
			expectCode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := generateWhereGuardCode(tt.funcName, tt.mutationKind, tt.whereMeta)

			if tt.expectCode {
				if code == "" {
					t.Error("expected code to be generated, got empty string")
				}

				if !contains(code, "where_meta") {
					t.Error("expected 'where_meta' in generated code")
				}

				if !contains(code, "enforce_non_empty_where_clause") {
					t.Error("expected 'enforce_non_empty_where_clause' in generated code")
				}

				if !contains(code, tt.funcName) {
					t.Errorf("expected function name %q in generated code", tt.funcName)
				}
			} else if code != "" {
				t.Errorf("expected no code, got non-empty string")
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

// Helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
