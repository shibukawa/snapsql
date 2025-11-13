package intermediate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql/explang"
	"github.com/shibukawa/snapsql/intermediate/codegenerator"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/parser/parsercommon"
)

// ErrExplangValidation is returned when explang expressions fail schema validation.
var ErrExplangValidation = errors.New("explang validation failed")

func validateExplangExpressions(funcDef *parser.FunctionDefinition, expressions []codegenerator.CELExpression, envs []codegenerator.CELEnvironment) ([][]explang.Step, error) {
	if funcDef == nil || len(expressions) == 0 {
		return nil, nil
	}

	params := funcDef.Parameters
	if params == nil {
		params = map[string]any{}
	}

	stepsPerExpr := make([][]explang.Step, len(expressions))

	for idx, expr := range expressions {
		source := strings.TrimSpace(expr.Expression)
		if source == "" {
			continue
		}

		line := max(expr.Position.Line, 1)

		column := max(expr.Position.Column, 1)

		steps, err := explang.ParseSteps(source, line, column)
		if err != nil {
			return nil, fmt.Errorf("%w for %s at line %d column %d: %w", ErrExplangValidation, expr.ID, line, column, err)
		}

		var opts *explang.ValidatorOptions
		if extras := buildAdditionalRoots(funcDef, envs, expr.EnvironmentIndex); len(extras) > 0 {
			opts = &explang.ValidatorOptions{AdditionalRoots: extras}
		}

		if errs := explang.ValidateStepsAgainstParameters(steps, params, opts); len(errs) > 0 {
			ve := errs[0]
			return nil, fmt.Errorf("%w for %s at line %d column %d: %s", ErrExplangValidation, expr.ID, ve.Step.Pos.Line, ve.Step.Pos.Column, ve.Message)
		}

		stepsPerExpr[idx] = steps
	}

	return stepsPerExpr, nil
}

func buildAdditionalRoots(funcDef *parser.FunctionDefinition, envs []codegenerator.CELEnvironment, envIndex int) map[string]any {
	if len(envs) == 0 {
		return nil
	}

	if envIndex < 0 || envIndex >= len(envs) {
		envIndex = 0
	}

	params := funcDef.Parameters
	if params == nil {
		params = map[string]any{}
	}

	known := make(map[string]struct{}, len(params))
	for name := range params {
		known[name] = struct{}{}
	}

	result := make(map[string]any)
	visitedEnv := make(map[int]struct{})

	for idx := envIndex; idx >= 0 && idx < len(envs); {
		if _, ok := visitedEnv[idx]; ok {
			break
		}

		visitedEnv[idx] = struct{}{}

		env := envs[idx]
		for _, variable := range env.AdditionalVariables {
			if variable.Name == "" {
				continue
			}

			if _, skip := known[variable.Name]; skip {
				continue
			}

			if _, exists := result[variable.Name]; exists {
				continue
			}

			result[variable.Name] = definitionFromVarInfo(variable)
		}

		if env.ParentIndex == nil {
			break
		}

		parent := *env.ParentIndex
		if parent < 0 {
			break
		}

		idx = parent
	}

	if len(result) == 0 {
		result = make(map[string]any)
	}

	// Fallback: include definitions from all environments so loop variables used in collection
	// expressions (which run before loop environments are pushed) can still be validated.
	for _, env := range envs {
		for _, variable := range env.AdditionalVariables {
			if variable.Name == "" {
				continue
			}

			if _, exists := result[variable.Name]; exists {
				continue
			}

			result[variable.Name] = definitionFromVarInfo(variable)
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func definitionFromVarInfo(info codegenerator.CELVariableInfo) any {
	if info.Value != nil {
		return definitionFromValue(info.Value)
	}

	if t := strings.TrimSpace(info.Type); t != "" {
		return t
	}

	return "any"
}

func definitionFromValue(val any) any {
	switch v := val.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, child := range v {
			result[key] = definitionFromValue(child)
		}

		return result
	case []any:
		if len(v) == 0 {
			return []any{}
		}

		return []any{definitionFromValue(v[0])}
	case nil:
		return "any"
	default:
		if t := parsercommon.InferTypeStringFromDummyValue(v); t != "" {
			return t
		}

		if t := parsercommon.InferTypeStringFromActualValue(v); t != "" {
			return t
		}

		return "any"
	}
}
