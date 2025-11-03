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

package gogen

import (
	"strings"
	"testing"

	snapsql "github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/intermediate"
)

func TestHierarchicalGeneration(t *testing.T) {
	// Example: Generate code for a file in nested structure
	format := &intermediate.IntermediateFormat{
		FunctionName:     "get_monthly_sales",
		Description:      "Get monthly sales report from orders/reports module",
		ResponseAffinity: "many",
		Parameters: []intermediate.Parameter{
			{Name: "year", Type: "int"},
			{Name: "month", Type: "int"},
		},
		Responses: []intermediate.Response{
			{Name: "order_id", Type: "int"},
			{Name: "total", Type: "decimal"},
			{Name: "customer__name", Type: "string"},
			{Name: "customer__email", Type: "string"},
		},
		Instructions: []intermediate.Instruction{
			{Op: "EMIT_STATIC", Value: "SELECT * FROM orders WHERE year = "},
			{Op: "EMIT_EVAL", ExprIndex: &[]int{0}[0]},
		},
		CELEnvironments: []intermediate.CELEnvironment{
			{Index: 0},
		},
		CELExpressions: []intermediate.CELExpression{
			{ID: "expr_001", Expression: "year", EnvironmentIndex: 0},
		},
	}

	// Parse hierarchy for nested file
	hierarchy := ParseFileHierarchy(
		"/project/queries/orders/reports/monthly_sales.snap.sql",
		"/project/queries",
		"/project/generated",
	)

	// Create generator with hierarchy support
	var output strings.Builder

	generator := New(format,
		WithHierarchy(hierarchy),
		WithDialect(snapsql.DialectPostgres),
		WithBaseImport("github.com/example/project/generated"),
	)

	err := generator.Generate(&output)
	if err != nil {
		t.Fatalf("Failed to generate hierarchical code: %v", err)
	}

	generatedCode := output.String()

	// Verify package name is based on deepest directory
	if !strings.Contains(generatedCode, "package reports") {
		t.Errorf("Expected package name 'reports', but got: %s", generatedCode)
	}

	// Verify function is generated
	if !strings.Contains(generatedCode, "func GetMonthlySales") {
		t.Errorf("Expected function 'GetMonthlySales' to be generated")
	}

	t.Logf("Generated hierarchical code:\n%s", generatedCode)
}

// Hierarchical generation usage documentation:
//
// Input structure:
// queries/
//   ├── _common.yaml
//   ├── users/
//   │   └── find_user.snap.sql
//   └── orders/
//       ├── _common.yaml
//       ├── create_order.snap.sql
//       └── reports/
//           └── monthly_sales.snap.sql
//
// Expected output structure:
// generated/
//   ├── common_types.go (package generated)
//   ├── users/
//   │   └── find_user.go (package users)
//   └── orders/
//       ├── common_types.go (package orders)
//       ├── create_order.go (package orders)
//       └── reports/
//           └── monthly_sales.go (package reports)
