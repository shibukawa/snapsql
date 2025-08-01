//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by snapsql. DO NOT EDIT.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generated

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

// InsertUserWithReturning specific CEL programs and mock path
var (
	insertuserwithreturningPrograms []cel.Program
)

const insertuserwithreturningMockPath = ""

func init() {

	// CEL environments based on intermediate format
	celEnvironments := make([]*cel.Env, 1)
	// Environment 0: Base environment
	env0, err := cel.NewEnv(
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		snapsqlgo.DecimalLibrary,
		cel.Variable("user_name", cel.StringType),
		cel.Variable("user_email", cel.StringType),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create InsertUserWithReturning CEL environment 0: %v", err))
	}
	celEnvironments[0] = env0

	// Create programs for each expression using the corresponding environment
	insertuserwithreturningPrograms = make([]cel.Program, 2)
	// expr_001: "user_name" using environment 0
	{
		ast, issues := celEnvironments[0].Compile("user_name")
		if issues != nil && issues.Err() != nil {
			panic(fmt.Sprintf("failed to compile CEL expression 'user_name': %v", issues.Err()))
		}
		program, err := celEnvironments[0].Program(ast)
		if err != nil {
			panic(fmt.Sprintf("failed to create CEL program for 'user_name': %v", err))
		}
		insertuserwithreturningPrograms[0] = program
	}
	// expr_002: "user_email" using environment 0
	{
		ast, issues := celEnvironments[0].Compile("user_email")
		if issues != nil && issues.Err() != nil {
			panic(fmt.Sprintf("failed to compile CEL expression 'user_email': %v", issues.Err()))
		}
		program, err := celEnvironments[0].Program(ast)
		if err != nil {
			panic(fmt.Sprintf("failed to create CEL program for 'user_email': %v", err))
		}
		insertuserwithreturningPrograms[1] = program
	}
}
// InsertUserWithReturning - interface{} Affinity
func InsertUserWithReturning(ctx context.Context, executor snapsqlgo.DBExecutor, userName string, userEmail string, opts ...snapsqlgo.FuncOpt) (interface{}, error) {
	var result interface{}

	// Extract function configuration
	funcConfig := snapsqlgo.GetFunctionConfig(ctx, "insertuserwithreturning", "interface{}")

	// Check for mock mode
	if funcConfig != nil && len(funcConfig.MockDataNames) > 0 {
		mockData, err := snapsqlgo.GetMockDataFromFiles(insertuserwithreturningMockPath, funcConfig.MockDataNames)
		if err != nil {
			return result, fmt.Errorf("failed to get mock data: %w", err)
		}

		result, err = snapsqlgo.MapMockDataToStruct[interface{}](mockData)
		if err != nil {
			return result, fmt.Errorf("failed to map mock data to interface{} struct: %w", err)
		}

		return result, nil
	}

	// Build SQL
	query := "INSERT INTO users (name, email, created_at) VALUES (?,?, CURRENT_TIMESTAMP) RETURNING id, name, email, created_at"
	args := []any{
		userName,
		userEmail,
	}

	// Execute query
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	// Execute query and scan single row
	row := stmt.QueryRowContext(ctx, args...)
	// Generic scan for interface{} result - not implemented
	// This would require runtime reflection or predefined column mapping

	return result, nil
}
