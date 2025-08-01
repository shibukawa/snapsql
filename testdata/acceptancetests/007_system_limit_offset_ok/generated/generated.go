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
// GetUsersWithLimitOffsetResult represents the response structure for GetUsersWithLimitOffset
type GetUsersWithLimitOffsetResult struct {
	ID int `json:"id"`
	Name string `json:"name"`
	Age int `json:"age"`
}

// GetUsersWithLimitOffset specific CEL programs and mock path
var (
	getuserswithlimitoffsetPrograms []cel.Program
)

const getuserswithlimitoffsetMockPath = ""

func init() {

	// CEL environments based on intermediate format
	celEnvironments := make([]*cel.Env, 1)
	// Environment 0: Base environment
	env0, err := cel.NewEnv(
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		snapsqlgo.DecimalLibrary,
		cel.Variable("min_age", cel.IntType),
		cel.Variable("max_age", cel.IntType),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create GetUsersWithLimitOffset CEL environment 0: %v", err))
	}
	celEnvironments[0] = env0

	// Create programs for each expression using the corresponding environment
	getuserswithlimitoffsetPrograms = make([]cel.Program, 2)
	// expr_001: "min_age" using environment 0
	{
		ast, issues := celEnvironments[0].Compile("min_age")
		if issues != nil && issues.Err() != nil {
			panic(fmt.Sprintf("failed to compile CEL expression 'min_age': %v", issues.Err()))
		}
		program, err := celEnvironments[0].Program(ast)
		if err != nil {
			panic(fmt.Sprintf("failed to create CEL program for 'min_age': %v", err))
		}
		getuserswithlimitoffsetPrograms[0] = program
	}
	// expr_002: "max_age" using environment 0
	{
		ast, issues := celEnvironments[0].Compile("max_age")
		if issues != nil && issues.Err() != nil {
			panic(fmt.Sprintf("failed to compile CEL expression 'max_age': %v", issues.Err()))
		}
		program, err := celEnvironments[0].Program(ast)
		if err != nil {
			panic(fmt.Sprintf("failed to create CEL program for 'max_age': %v", err))
		}
		getuserswithlimitoffsetPrograms[1] = program
	}
}
// GetUsersWithLimitOffset - []GetUsersWithLimitOffsetResult Affinity
func GetUsersWithLimitOffset(ctx context.Context, executor snapsqlgo.DBExecutor, minAge int, maxAge int, opts ...snapsqlgo.FuncOpt) ([]GetUsersWithLimitOffsetResult, error) {
	var result []GetUsersWithLimitOffsetResult

	// Extract function configuration
	funcConfig := snapsqlgo.GetFunctionConfig(ctx, "getuserswithlimitoffset", "[]getuserswithlimitoffsetresult")

	// Check for mock mode
	if funcConfig != nil && len(funcConfig.MockDataNames) > 0 {
		mockData, err := snapsqlgo.GetMockDataFromFiles(getuserswithlimitoffsetMockPath, funcConfig.MockDataNames)
		if err != nil {
			return result, fmt.Errorf("failed to get mock data: %w", err)
		}

		result, err = snapsqlgo.MapMockDataToStruct[[]GetUsersWithLimitOffsetResult](mockData)
		if err != nil {
			return result, fmt.Errorf("failed to map mock data to []GetUsersWithLimitOffsetResult struct: %w", err)
		}

		return result, nil
	}

	// Build SQL
	query := "SELECT id, name, age FROM users WHERE age >=?AND age <=?LIMIT OFFSET "
	args := []any{
		minAge,
		maxAge,
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
	
	for rows.Next() {
	    var item GetUsersWithLimitOffsetResult
	    err := rows.Scan(
	        &item.ID,
	        &item.Name,
	        &item.Age
	    )
	    if err != nil {
	        return result, fmt.Errorf("failed to scan row: %w", err)
	    }
	    result = append(result, item)
	}
	
	if err = rows.Err(); err != nil {
	    return result, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}
