package pygen

import (
	"testing"

	"github.com/shibukawa/snapsql/explang"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestPythonExpressionRenderer(t *testing.T) {
	format := &intermediate.IntermediateFormat{
		Parameters: []intermediate.Parameter{
			{Name: "user"},
			{Name: "orders"},
		},
		Expressions: []intermediate.ExplangExpression{
			{
				ID: "user_name",
				Steps: []intermediate.Expressions{
					{Kind: explang.StepIdentifier, Identifier: "user"},
					{Kind: explang.StepMember, Property: "name"},
				},
			},
			{
				ID: "safe_email",
				Steps: []intermediate.Expressions{
					{Kind: explang.StepIdentifier, Identifier: "user"},
					{Kind: explang.StepMember, Property: "profile", Safe: true},
					{Kind: explang.StepMember, Property: "email", Safe: true},
				},
			},
			{
				ID: "order_amount",
				Steps: []intermediate.Expressions{
					{Kind: explang.StepIdentifier, Identifier: "orders"},
					{Kind: explang.StepIndex, Index: 0, Safe: true},
					{Kind: explang.StepMember, Property: "amount"},
				},
			},
		},
	}

	scope := newExpressionScope(format)
	renderer := newPythonExpressionRenderer(format, scope)

	tests := []struct {
		name    string
		index   int
		want    string
		wantErr bool
	}{
		{
			name:  "simple member access",
			index: 0,
			want:  "user.name",
		},
		{
			name:  "safe nested member access",
			index: 1,
			want:  "(None if user is None else (None if user.profile is None else user.profile.email))",
		},
		{
			name:  "safe index access",
			index: 2,
			want:  "(None if (orders is None or len(orders) <= 0) else orders[0].amount)",
		},
		{
			name:    "out of range expression",
			index:   99,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.render(tt.index)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("render(%d) expected error, got nil", tt.index)
				}
				return
			}

			if err != nil {
				t.Fatalf("render(%d) error = %v", tt.index, err)
			}

			if got != tt.want {
				t.Errorf("render(%d) = %q, want %q", tt.index, got, tt.want)
			}
		})
	}
}
