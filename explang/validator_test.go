package explang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStepsAgainstParameters_SuccessCases(t *testing.T) {
	params := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"age":  "int",
				"name": "string",
			},
		},
		"users": []any{map[string]any{
			"id":   "int",
			"name": "string",
		}},
		"ids": "int[]",
	}

	cases := []string{
		"user.profile.age",
		"users[0].name",
		"ids[0]",
	}

	for _, expr := range cases {
		steps, err := ParseSteps(expr, 1, 1)
		if !assert.NoError(t, err, expr) {
			continue
		}

		errs := ValidateStepsAgainstParameters(steps, params, nil)
		assert.Empty(t, errs, expr)
	}
}

func TestValidateStepsAgainstParameters_Errors(t *testing.T) {
	params := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"age": "int",
			},
		},
		"tags": "string[]",
	}

	tests := []struct {
		name      string
		input     string
		assertion func(t *testing.T, errs []ValidationError)
	}{
		{
			name:  "unknown root",
			input: "account.name",
			assertion: func(t *testing.T, errs []ValidationError) {
				t.Helper()

				if assert.Len(t, errs, 1) {
					assert.Contains(t, errs[0].Message, "unknown root parameter \"account\"")
					assert.Equal(t, 10, errs[0].Step.Pos.Line)
					assert.Equal(t, 5, errs[0].Step.Pos.Column)
				}
			},
		},
		{
			name:  "unknown field",
			input: "user.profile.agee",
			assertion: func(t *testing.T, errs []ValidationError) {
				t.Helper()

				if assert.Len(t, errs, 1) {
					assert.Contains(t, errs[0].Message, "unknown field \"agee\"")
					assert.Equal(t, "agee", errs[0].Step.Property)
				}
			},
		},
		{
			name:  "member on scalar",
			input: "user.profile.age.value",
			assertion: func(t *testing.T, errs []ValidationError) {
				t.Helper()

				if assert.Len(t, errs, 1) {
					assert.Contains(t, errs[0].Message, "cannot access member \"value\"")
				}
			},
		},
		{
			name:  "index on non array",
			input: "user.profile[0]",
			assertion: func(t *testing.T, errs []ValidationError) {
				t.Helper()

				if assert.Len(t, errs, 1) {
					assert.Contains(t, errs[0].Message, "is not an array")
				}
			},
		},
	}

	for _, testcase := range tests {
		steps, err := ParseSteps(testcase.input, 10, 5)
		if !assert.NoError(t, err, testcase.name) {
			continue
		}

		errs := ValidateStepsAgainstParameters(steps, params, nil)

		t.Run(testcase.name, func(t *testing.T) {
			testcase.assertion(t, errs)
		})
	}
}

func TestValidateStepsAgainstParameters_AdditionalRoots(t *testing.T) {
	steps, err := ParseSteps("system.now", 4, 2)
	if !assert.NoError(t, err) {
		return
	}

	options := &ValidatorOptions{
		AdditionalRoots: map[string]any{
			"system": map[string]any{"now": "timestamp"},
		},
	}
	errs := ValidateStepsAgainstParameters(steps, map[string]any{}, options)
	assert.Empty(t, errs)
}
