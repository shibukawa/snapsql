package codegenerator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
)

// TestDebugEnvironmentPopulation checks what parameters are loaded into CEL environments
func TestDebugEnvironmentPopulation(t *testing.T) {
	sql := `/*# parameters: { u: { id: int, name: string } } */ INSERT INTO users (id, name) VALUES /*= u */( 1, 'name' )`

	reader := strings.NewReader(sql)

	stmt, typeInfoMap, funcDef, err := parser.ParseSQLFile(reader, nil, "", "", parser.Options{})
	if err != nil {
		t.Fatalf("ParseSQLFile failed: %v", err)
	}

	// Check FunctionDefinition
	fmt.Printf("\n=== FunctionDefinition ===\n")

	if funcDef != nil {
		fmt.Printf("FunctionName: %s\n", funcDef.FunctionName)
		fmt.Printf("ParameterOrder: %v\n", funcDef.ParameterOrder)
		fmt.Printf("OriginalParameters: %v\n", funcDef.OriginalParameters)

		for _, paramName := range funcDef.ParameterOrder {
			originalParamValue := funcDef.OriginalParameters[paramName]
			fmt.Printf("  Parameter '%s': %T = %v\n", paramName, originalParamValue, originalParamValue)
		}
	}

	// Create context and initialize environment
	ctx := NewGenerationContext(snapsql.DialectPostgres)
	ctx.SetTypeInfoMap(typeInfoMap)

	// Before: Check environment before adding parameters
	fmt.Printf("\n=== Before GenerateInsertInstructionsWithFunctionDef ===\n")

	for i, env := range ctx.CELEnvironments {
		fmt.Printf("Environment [%d]: Container=%s, Vars=%v\n", i, env.Container, env.AdditionalVariables)
	}

	// Call GenerateInsertInstructionsWithFunctionDef to populate environment
	_, _, _, err = GenerateInsertInstructionsWithFunctionDef(stmt, ctx, funcDef)
	if err != nil {
		t.Fatalf("GenerateInsertInstructionsWithFunctionDef failed: %v", err)
	}

	// After: Check environment after adding parameters
	fmt.Printf("\n=== After GenerateInsertInstructionsWithFunctionDef ===\n")

	for i, env := range ctx.CELEnvironments {
		fmt.Printf("Environment [%d]: Container=%s\n", i, env.Container)
		fmt.Printf("  AdditionalVariables (%d vars):\n", len(env.AdditionalVariables))

		for j, v := range env.AdditionalVariables {
			fmt.Printf("    [%d] Name=%s, Type=%s\n", j, v.Name, v.Type)
		}
	}
}
