package intermediate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shibukawa/snapsql/intermediate/codegenerator"
	"github.com/shibukawa/snapsql/parser"
)

func TestValidateExplangExpressions_Success(t *testing.T) {
	funcDef := &parser.FunctionDefinition{
		Parameters: map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "string",
				},
			},
		},
	}

	expr := codegenerator.CELExpression{
		ID:               "expr_001",
		Expression:       "user.profile.name",
		EnvironmentIndex: 0,
		Position:         codegenerator.Position{Line: 3, Column: 5},
	}

	envs := []codegenerator.CELEnvironment{{Index: 0}}

	steps, err := validateExplangExpressions(funcDef, []codegenerator.CELExpression{expr}, envs)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Len(t, steps[0], 3)
}

func TestValidateExplangExpressions_UnknownRoot(t *testing.T) {
	funcDef := &parser.FunctionDefinition{
		Parameters: map[string]any{
			"user": "string",
		},
	}

	expr := codegenerator.CELExpression{
		ID:               "expr_002",
		Expression:       "foo.bar",
		EnvironmentIndex: 0,
		Position:         codegenerator.Position{Line: 1, Column: 1},
	}

	envs := []codegenerator.CELEnvironment{{Index: 0}}

	_, err := validateExplangExpressions(funcDef, []codegenerator.CELExpression{expr}, envs)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExplangValidation)
	assert.Contains(t, err.Error(), `unknown root parameter "foo"`)
}

func TestValidateExplangExpressions_AdditionalRootSuccess(t *testing.T) {
	funcDef := &parser.FunctionDefinition{
		Parameters: map[string]any{
			"users": []any{map[string]any{"email": "string"}},
		},
	}

	rootEnv := codegenerator.CELEnvironment{Index: 0}
	parentIndex := 0
	loopEnv := codegenerator.CELEnvironment{
		Index:       1,
		ParentIndex: &parentIndex,
		AdditionalVariables: []codegenerator.CELVariableInfo{
			{
				Name: "item",
				Value: map[string]any{
					"email": "dummy@example.com",
				},
			},
		},
	}

	expr := codegenerator.CELExpression{
		ID:               "expr_loop",
		Expression:       "item.email",
		EnvironmentIndex: 1,
		Position:         codegenerator.Position{Line: 5, Column: 8},
	}

	steps, err := validateExplangExpressions(funcDef, []codegenerator.CELExpression{expr}, []codegenerator.CELEnvironment{rootEnv, loopEnv})
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.True(t, len(steps[0]) > 0)
}

func TestValidateExplangExpressions_AdditionalRootUnknownField(t *testing.T) {
	funcDef := &parser.FunctionDefinition{
		Parameters: map[string]any{
			"users": []any{map[string]any{"email": "string"}},
		},
	}

	parentIndex := 0
	envs := []codegenerator.CELEnvironment{
		{Index: 0},
		{
			Index:       1,
			ParentIndex: &parentIndex,
			AdditionalVariables: []codegenerator.CELVariableInfo{
				{
					Name: "item",
					Value: map[string]any{
						"email": "dummy@example.com",
					},
				},
			},
		},
	}

	expr := codegenerator.CELExpression{
		ID:               "expr_loop_bad",
		Expression:       "item.unknown",
		EnvironmentIndex: 1,
		Position:         codegenerator.Position{Line: 9, Column: 3},
	}

	_, err := validateExplangExpressions(funcDef, []codegenerator.CELExpression{expr}, envs)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrExplangValidation)
	assert.Contains(t, err.Error(), `unknown field "unknown"`)
}
