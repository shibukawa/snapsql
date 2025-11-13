package explang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSteps(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Step
		wantErr bool
	}{
		{
			name:  "single identifier",
			input: "user",
			want: []Step{
				{Kind: StepIdentifier, Identifier: "user", Pos: defaultPos(0, 4)},
			},
		},
		{
			name:  "member and index",
			input: "users[0].profile",
			want: []Step{
				{Kind: StepIdentifier, Identifier: "users", Pos: defaultPos(0, 5)},
				{Kind: StepIndex, Index: 0, Pos: defaultPos(5, 3)},
				{Kind: StepMember, Property: "profile", Pos: defaultPos(8, 8)},
			},
		},
		{
			name:  "safe member and index",
			input: "order?.items?[10]",
			want: []Step{
				{Kind: StepIdentifier, Identifier: "order", Pos: defaultPos(0, 5)},
				{Kind: StepMember, Property: "items", Safe: true, Pos: defaultPos(5, 7)},
				{Kind: StepIndex, Index: 10, Safe: true, Pos: defaultPos(12, 5)},
			},
		},
		{
			name:    "invalid start",
			input:   "1foo",
			wantErr: true,
		},
		{
			name:    "missing close bracket",
			input:   "users[0",
			wantErr: true,
		},
		{
			name:    "dangling safe",
			input:   "user?",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSteps(tt.input, 1, 1)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSteps_WithBaseLineColumn(t *testing.T) {
	steps, err := ParseSteps("a.b", 3, 5)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []Step{
		{Kind: StepIdentifier, Identifier: "a", Pos: Position{Offset: 0, Line: 3, Column: 5, Length: 1}},
		{Kind: StepMember, Property: "b", Pos: Position{Offset: 1, Line: 3, Column: 6, Length: 2}},
	}, steps)
}

func defaultPos(offset, length int) Position {
	return Position{Offset: offset, Line: 1, Column: offset + 1, Length: length}
}
