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
type GetUserWithJobsResultJobs struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Company string `json:"company"`
}
// GetUserWithJobsResult represents the response structure for GetUserWithJobs
type GetUserWithJobsResult struct {
	Jobs []GetUserWithJobsResultJobs `json:"jobs"`
	UID int `json:"u.id"`
	UName string `json:"u.name"`
	UEmail string `json:"u.email"`
}

// GetUserWithJobs specific CEL programs and mock path
var (
	getuserwithjobsPrograms []cel.Program
)

const getuserwithjobsMockPath = ""

func init() {
	// Static accessor functions for each type
	getuserwithjobsresultjobsCompanyAccessor := func(value interface{}) ref.Val {
		v := value.(*GetUserWithJobsResultJobs)
		return snapsqlgo.ConvertGoValueToCEL(v.Company)
	}
	getuserwithjobsresultjobsIDAccessor := func(value interface{}) ref.Val {
		v := value.(*GetUserWithJobsResultJobs)
		return snapsqlgo.ConvertGoValueToCEL(v.ID)
	}
	getuserwithjobsresultjobsTitleAccessor := func(value interface{}) ref.Val {
		v := value.(*GetUserWithJobsResultJobs)
		return snapsqlgo.ConvertGoValueToCEL(v.Title)
	}

	// Create type definitions for local registry
	typeDefinitions := map[string]map[string]snapsqlgo.FieldInfo{
		"GetUserWithJobsResultJobs": {
			"company": snapsqlgo.CreateFieldInfo(
				"company", 
				types.StringType, 
				getuserwithjobsresultjobsCompanyAccessor,
			),
			"id": snapsqlgo.CreateFieldInfo(
				"id", 
				types.IntType, 
				getuserwithjobsresultjobsIDAccessor,
			),
			"title": snapsqlgo.CreateFieldInfo(
				"title", 
				types.StringType, 
				getuserwithjobsresultjobsTitleAccessor,
			),
		},
	}

	// Create and set up local registry
	registry := snapsqlgo.NewLocalTypeRegistry()
	for typeName, fields := range typeDefinitions {
		structInfo := &snapsqlgo.StructInfo{
			Name:    typeName,
			CelType: types.NewObjectType(typeName),
			Fields:  fields,
		}
		registry.RegisterStruct(typeName, structInfo)
	}
	
	// Set global registry for nested type resolution
	snapsqlgo.SetGlobalRegistry(registry)

	// CEL environments based on intermediate format
	celEnvironments := make([]*cel.Env, 1)
	// Environment 0: Base environment
	env0, err := cel.NewEnv(
		cel.HomogeneousAggregateLiterals(),
		cel.EagerlyValidateDeclarations(true),
		snapsqlgo.DecimalLibrary,
		snapsqlgo.CreateCELOptionsWithTypes(typeDefinitions)...,
		cel.Variable("user_id", cel.IntType),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create GetUserWithJobs CEL environment 0: %v", err))
	}
	celEnvironments[0] = env0

	// Create programs for each expression using the corresponding environment
	getuserwithjobsPrograms = make([]cel.Program, 1)
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
		getuserwithjobsPrograms[0] = program
	}
}
// GetUserWithJobs - GetUserWithJobsResult Affinity
func GetUserWithJobs(ctx context.Context, executor snapsqlgo.DBExecutor, userID int, opts ...snapsqlgo.FuncOpt) (GetUserWithJobsResult, error) {
	var result GetUserWithJobsResult

	// Extract function configuration
	funcConfig := snapsqlgo.GetFunctionConfig(ctx, "getuserwithjobs", "getuserwithjobsresult")

	// Check for mock mode
	if funcConfig != nil && len(funcConfig.MockDataNames) > 0 {
		mockData, err := snapsqlgo.GetMockDataFromFiles(getuserwithjobsMockPath, funcConfig.MockDataNames)
		if err != nil {
			return result, fmt.Errorf("failed to get mock data: %w", err)
		}

		result, err = snapsqlgo.MapMockDataToStruct[GetUserWithJobsResult](mockData)
		if err != nil {
			return result, fmt.Errorf("failed to map mock data to GetUserWithJobsResult struct: %w", err)
		}

		return result, nil
	}

	// Build SQL
	query := "SELECT u.id, u.name, u.email, j.id AS jobs__id, j.title AS jobs__title, j.company AS jobs__company FROM users u LEFT JOIN jobs j ON u.id = j.user_id WHERE u.id =?"
	args := []any{
		userID,
	}

	// Execute query
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	// Execute query and scan single row
	row := stmt.QueryRowContext(ctx, args...)
	err = row.Scan(
	    &result.Jobs,
	    &result.UID,
	    &result.UName,
	    &result.UEmail
	)
	if err != nil {
	    return result, fmt.Errorf("failed to scan row: %w", err)
	}

	return result, nil
}
