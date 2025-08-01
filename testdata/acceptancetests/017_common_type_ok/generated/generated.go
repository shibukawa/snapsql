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

// GetUser specific CEL programs and mock path
var (
	getuserPrograms []cel.Program
)

const getuserMockPath = ""

func init() {

	// CEL environments based on intermediate format
	celEnvironments := make([]*cel.Env, 1)
	// Environment 0: Base environment
	env0, err := cel.NewEnv(
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		snapsqlgo.DecimalLibrary,
		cel.Variable("user_id", cel.IntType),
		cel.Variable("user", cel.types.NewObjectType("User")),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create GetUser CEL environment 0: %v", err))
	}
	celEnvironments[0] = env0

	// Create programs for each expression using the corresponding environment
	getuserPrograms = make([]cel.Program, 1)
	// expr_001: "user_id" using environment 0
	{
		ast, issues := celEnvironments[0].Compile("user_id")
		if issues != nil && issues.Err() != nil {
			panic(fmt.Sprintf("failed to compile CEL expression 'user_id': %v", issues.Err()))
		}
		program, err := celEnvironments[0].Program(ast)
		if err != nil {
			panic(fmt.Sprintf("failed to create CEL program for 'user_id': %v", err))
		}
		getuserPrograms[0] = program
	}
}
// GetUser - interface{} Affinity
func GetUser(ctx context.Context, executor snapsqlgo.DBExecutor, userID int, user User, opts ...snapsqlgo.FuncOpt) (interface{}, error) {
	var result interface{}

	// Extract function configuration
	funcConfig := snapsqlgo.GetFunctionConfig(ctx, "getuser", "interface{}")

	// Check for mock mode
	if funcConfig != nil && len(funcConfig.MockDataNames) > 0 {
		mockData, err := snapsqlgo.GetMockDataFromFiles(getuserMockPath, funcConfig.MockDataNames)
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
	query := "SELECT u.id, u.name, u.email FROM users u WHERE u.id =?"
	args := []any{
		userID,
	}

	// Execute query
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	// Execute query and scan multiple rows
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
	    return result, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()
	
	// Generic scan for interface{} result - not implemented
	// This would require runtime reflection or predefined column mapping

	return result, nil
}
