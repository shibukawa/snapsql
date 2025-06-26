package intermediate

import (
	"slices"
	"strings"

	"github.com/google/cel-go/cel"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// CELVariableExtractor extracts variables from CEL expressions using proper parsing
type CELVariableExtractor struct {
	env *cel.Env
}

// NewCELVariableExtractor creates a new CEL-based variable extractor
func NewCELVariableExtractor() (*CELVariableExtractor, error) {
	// Create a permissive CEL environment for parsing
	env, err := cel.NewEnv(
		cel.Variable("_", cel.AnyType), // Wildcard variable for parsing
	)
	if err != nil {
		return nil, err
	}

	return &CELVariableExtractor{env: env}, nil
}

// ExtractVariables extracts all variable references from a CEL expression
func (cve *CELVariableExtractor) ExtractVariables(expression string) ([]string, error) {
	if expression == "" {
		return nil, nil
	}

	// Try to parse the expression
	parsed, issues := cve.env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		// If parsing fails, fall back to simple extraction
		return cve.extractVariablesSimple(expression), issues.Err()
	}

	// Extract variables from the AST
	variables := make(map[string]bool)
	parsedExpr, _ := cel.AstToParsedExpr(parsed)
	if parsedExpr != nil && parsedExpr.GetExpr() != nil {
		cve.extractFromExpr(parsedExpr.GetExpr(), variables)
	}

	// Convert to sorted slice
	result := make([]string, 0, len(variables))
	for variable := range variables {
		result = append(result, variable)
	}

	// Sort for consistent output
	slices.Sort(result)

	return result, nil
}

// extractFromExpr recursively extracts variables from CEL expression protobuf
func (cve *CELVariableExtractor) extractFromExpr(expr *exprpb.Expr, variables map[string]bool) {
	if expr == nil {
		return
	}

	switch expr.GetExprKind().(type) {
	case *exprpb.Expr_IdentExpr:
		// Direct variable reference
		ident := expr.GetIdentExpr()
		if ident.GetName() != "_" { // Skip wildcard
			variables[ident.GetName()] = true
		}

	case *exprpb.Expr_SelectExpr:
		// Member access (e.g., user.name)
		sel := expr.GetSelectExpr()
		// Extract from the operand (the object being accessed)
		cve.extractFromExpr(sel.GetOperand(), variables)

	case *exprpb.Expr_CallExpr:
		// Function call - extract variables from arguments
		call := expr.GetCallExpr()
		for _, arg := range call.GetArgs() {
			cve.extractFromExpr(arg, variables)
		}
		// Also check the target if it's a method call
		if call.GetTarget() != nil {
			cve.extractFromExpr(call.GetTarget(), variables)
		}

	case *exprpb.Expr_ListExpr:
		// List literal - extract from elements
		list := expr.GetListExpr()
		for _, elem := range list.GetElements() {
			cve.extractFromExpr(elem, variables)
		}

	case *exprpb.Expr_StructExpr:
		// Struct literal - extract from field values
		structExpr := expr.GetStructExpr()
		for _, entry := range structExpr.GetEntries() {
			if entry.GetKeyKind() != nil {
				switch entry.GetKeyKind().(type) {
				case *exprpb.Expr_CreateStruct_Entry_FieldKey:
					// Field key is a string, no variables
				case *exprpb.Expr_CreateStruct_Entry_MapKey:
					// Map key expression
					cve.extractFromExpr(entry.GetMapKey(), variables)
				}
			}
			cve.extractFromExpr(entry.GetValue(), variables)
		}

	case *exprpb.Expr_ComprehensionExpr:
		// Comprehension - extract from iter range and result
		comp := expr.GetComprehensionExpr()
		cve.extractFromExpr(comp.GetIterRange(), variables)
		cve.extractFromExpr(comp.GetResult(), variables)
		if comp.GetLoopCondition() != nil {
			cve.extractFromExpr(comp.GetLoopCondition(), variables)
		}

	case *exprpb.Expr_ConstExpr:
		// Literal values - no variables to extract
		return

	default:
		// For any other expression types, we don't extract variables
		return
	}
}

// extractVariablesSimple provides fallback simple variable extraction
func (cve *CELVariableExtractor) extractVariablesSimple(expression string) []string {
	variables := make(map[string]bool)

	// Handle negation
	expr := strings.TrimSpace(expression)
	if strings.HasPrefix(expr, "!") {
		expr = strings.TrimPrefix(expr, "!")
		if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
			expr = strings.TrimPrefix(strings.TrimSuffix(expr, ")"), "(")
		}
	}

	// Handle logical operators
	for _, op := range []string{" || ", " && "} {
		if strings.Contains(expr, op) {
			parts := strings.Split(expr, op)
			for _, part := range parts {
				subVars := cve.extractVariablesSimple(strings.TrimSpace(part))
				for _, v := range subVars {
					variables[v] = true
				}
			}
			return cve.mapKeysToSlice(variables)
		}
	}

	// Extract root variable from dot notation
	if strings.Contains(expr, ".") {
		parts := strings.Split(expr, ".")
		if len(parts) > 0 {
			rootVar := strings.TrimSpace(parts[0])
			if isValidVariableName(rootVar) {
				variables[rootVar] = true
			}
		}
	} else {
		// Simple variable name
		if isValidVariableName(expr) {
			variables[expr] = true
		}
	}

	return cve.mapKeysToSlice(variables)
}

// mapKeysToSlice converts map keys to sorted slice
func (cve *CELVariableExtractor) mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	slices.Sort(keys)
	return keys
}

// EnhancedVariableExtractor uses CEL parsing for more accurate variable extraction
type EnhancedVariableExtractor struct {
	celExtractor   *CELVariableExtractor
	allVars        map[string]bool
	structuralVars map[string]bool
	parameterVars  map[string]bool
}

// NewEnhancedVariableExtractor creates a new enhanced variable extractor
func NewEnhancedVariableExtractor() (*EnhancedVariableExtractor, error) {
	celExtractor, err := NewCELVariableExtractor()
	if err != nil {
		return nil, err
	}

	return &EnhancedVariableExtractor{
		celExtractor:   celExtractor,
		allVars:        make(map[string]bool),
		structuralVars: make(map[string]bool),
		parameterVars:  make(map[string]bool),
	}, nil
}

// ExtractFromInstructions extracts variable dependencies using CEL parsing
func (eve *EnhancedVariableExtractor) ExtractFromInstructions(instructions []Instruction) (VariableDependencies, error) {
	// Reset state
	eve.allVars = make(map[string]bool)
	eve.structuralVars = make(map[string]bool)
	eve.parameterVars = make(map[string]bool)

	for _, inst := range instructions {
		switch inst.Op {
		case "JUMP_IF_EXP":
			// Structural variables - affect SQL structure
			vars, err := eve.celExtractor.ExtractVariables(inst.Exp)
			if err != nil {
				continue // Skip on error, fallback handled in CEL extractor
			}
			for _, v := range vars {
				eve.allVars[v] = true
				eve.structuralVars[v] = true
			}

		case "LOOP_START":
			// Collection variables - affect SQL structure
			vars, err := eve.celExtractor.ExtractVariables(inst.Collection)
			if err != nil {
				continue
			}
			for _, v := range vars {
				eve.allVars[v] = true
				eve.structuralVars[v] = true
			}

		case "EMIT_PARAM":
			// Parameter variables - only affect values
			if inst.Param != "" {
				vars := eve.extractVariablesFromPath(inst.Param)
				for _, v := range vars {
					eve.allVars[v] = true
					eve.parameterVars[v] = true
				}
			}

		case "EMIT_EVAL":
			// CEL expression variables - only affect values
			vars, err := eve.celExtractor.ExtractVariables(inst.Exp)
			if err != nil {
				continue
			}
			for _, v := range vars {
				eve.allVars[v] = true
				eve.parameterVars[v] = true
			}
		}
	}

	return VariableDependencies{
		AllVariables:        eve.mapKeysToSlice(eve.allVars),
		StructuralVariables: eve.mapKeysToSlice(eve.structuralVars),
		ParameterVariables:  eve.mapKeysToSlice(eve.parameterVars),
		DependencyGraph:     eve.buildDependencyGraph(),
		CacheKeyTemplate:    eve.generateCacheKeyTemplate(),
	}, nil
}

// extractVariablesFromPath extracts variables from dot notation paths
func (eve *EnhancedVariableExtractor) extractVariablesFromPath(path string) []string {
	if path == "" {
		return nil
	}

	parts := strings.Split(path, ".")
	if len(parts) > 0 {
		rootVar := strings.TrimSpace(parts[0])
		if isValidVariableName(rootVar) {
			return []string{rootVar}
		}
	}

	return nil
}

// mapKeysToSlice converts map keys to sorted slice
func (eve *EnhancedVariableExtractor) mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	slices.Sort(keys)
	return keys
}

// buildDependencyGraph builds a dependency graph between variables
func (eve *EnhancedVariableExtractor) buildDependencyGraph() map[string][]string {
	graph := make(map[string][]string)

	// Create a simple graph where structural variables depend on themselves
	for structVar := range eve.structuralVars {
		graph[structVar] = []string{structVar}
	}

	return graph
}

// generateCacheKeyTemplate generates a template for cache key generation
func (eve *EnhancedVariableExtractor) generateCacheKeyTemplate() string {
	if len(eve.structuralVars) == 0 {
		return "static" // No structural variables, SQL is static
	}

	// Create cache key template based on structural variables
	vars := eve.mapKeysToSlice(eve.structuralVars)
	return strings.Join(vars, ",")
}
