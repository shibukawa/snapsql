package fixtureexecutor

import (
	"errors"
	"fmt"
	"testing"
)

func TestClassifyFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want FailureKind
	}{
		{
			name: "SimpleValidation",
			err:  fmt.Errorf("simple validation failed: %w", errors.New("value mismatch")),
			want: FailureKindAssertion,
		},
		{
			name: "VerifyValidation",
			err:  fmt.Errorf("verify query validation failed: %w", errors.New("value mismatch")),
			want: FailureKindAssertion,
		},
		{
			name: "TableStateValidation",
			err:  fmt.Errorf("table state validation failed: %w", errors.New("pk mismatch")),
			want: FailureKindAssertion,
		},
		{
			name: "ExecuteFixtures",
			err:  fmt.Errorf("failed to execute fixtures: %w", errors.New("foreign key constraint failed")),
			want: FailureKindDefinition,
		},
		{
			name: "ExecuteTableFixture",
			err:  fmt.Errorf("failed to execute fixture for table users: %w", errors.New("no such table")),
			want: FailureKindDefinition,
		},
		{
			name: "ExecuteVerifyQuery",
			err:  fmt.Errorf("failed to execute verify query: %w", errors.New("no rows")),
			want: FailureKindDefinition,
		},
		{
			name: "Unknown",
			err:  errors.New("unclassified"),
			want: FailureKindUnknown,
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			got := ClassifyFailure(tc.err)
			if got != tc.want {
				t.Fatalf("ClassifyFailure() = %v, want %v", got, tc.want)
			}
		})
	}
}
