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

//  specific CEL programs and mock path
var (
	Programs []cel.Program
)

const MockPath = ""

func init() {

	// CEL environments based on intermediate format
	celEnvironments := make([]*cel.Env, 0)

	// Create programs for each expression using the corresponding environment
	Programs = make([]cel.Program, 0)
}
//  - interface{} Affinity
func (ctx context.Context, executor snapsqlgo.DBExecutor, opts ...snapsqlgo.FuncOpt) (interface{}, error) {
	var result interface{}

	// Extract function configuration
	funcConfig := snapsqlgo.GetFunctionConfig(ctx, "", "interface{}")

	// Check for mock mode
	if funcConfig != nil && len(funcConfig.MockDataNames) > 0 {
		mockData, err := snapsqlgo.GetMockDataFromFiles(MockPath, funcConfig.MockDataNames)
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
	query := ""
	args := []any{}

	// Execute query
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return result, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	// Execute query (no result expected)
	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
	    return result, fmt.Errorf("failed to execute statement: %w", err)
	}

	return result, nil
}
