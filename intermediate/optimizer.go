package intermediate

import (
	"strings"
)

// OptimizedInstruction represents an optimized instruction
type OptimizedInstruction struct {
	Op                    string
	Value                 string
	ExprIndex             *int
	SqlFragment           string
	Variable              string // for LOOP_START
	CollectionExprIndex   *int   // for LOOP_START
	EnvIndex              *int   // for LOOP_START/LOOP_END
}

// OptimizeInstructions filters and optimizes instructions for a specific dialect
func OptimizeInstructions(instructions []Instruction, dialect string) ([]OptimizedInstruction, error) {
	var result []OptimizedInstruction
	skipUntilEnd := 0

	for i, inst := range instructions {
		// Skip instructions if we're inside a system directive block
		if skipUntilEnd > 0 {
			if inst.Op == OpEnd {
				skipUntilEnd--
			} else if inst.Op == OpIfSystemLimit || inst.Op == OpIfSystemOffset {
				skipUntilEnd++
			}
			continue
		}

		switch inst.Op {
		case OpEmitStatic:
			result = append(result, OptimizedInstruction{
				Op:    "EMIT_STATIC",
				Value: inst.Value,
			})

		case OpEmitEval:
			// Split EMIT_EVAL into placeholder and parameter handling
			result = append(result, OptimizedInstruction{
				Op:    "EMIT_STATIC",
				Value: "?",
			})
			result = append(result, OptimizedInstruction{
				Op:        "ADD_PARAM",
				ExprIndex: inst.ExprIndex,
			})

		case OpEmitIfDialect:
			// Filter by dialect - only emit if matches current dialect
			if dialect == "" || containsDialect(inst.Dialects, dialect) {
				result = append(result, OptimizedInstruction{
					Op:    "EMIT_STATIC",
					Value: inst.SqlFragment,
				})
			}
			// If dialect doesn't match, skip this instruction

		case OpEmitUnlessBoundary:
			// Check if we're in a static context (no control flow between here and next BOUNDARY)
			isStaticContext := true
			boundaryFound := false
			
			// Look ahead to find the next BOUNDARY or control flow instruction
			for j := i + 1; j < len(instructions); j++ {
				nextInst := instructions[j]
				switch nextInst.Op {
				case OpBoundary:
					boundaryFound = true
					goto checkComplete
				case OpIf, OpElseIf, OpElse, OpEnd, OpLoopStart, OpLoopEnd:
					isStaticContext = false
					goto checkComplete
				}
			}
			
			checkComplete:
			if isStaticContext && boundaryFound {
				// Skip this instruction (will be handled by BOUNDARY)
			} else if isStaticContext && !boundaryFound {
				// No BOUNDARY found, emit as static
				result = append(result, OptimizedInstruction{
					Op:    "EMIT_STATIC",
					Value: inst.Value,
				})
			} else {
				// Dynamic context - need runtime boundary handling
				result = append(result, OptimizedInstruction{
					Op:    "EMIT_UNLESS_BOUNDARY",
					Value: inst.Value,
				})
			}

		case OpBoundary:
			// In static context, BOUNDARY is just a marker
			// In dynamic context, it needs to be handled at runtime
			result = append(result, OptimizedInstruction{
				Op: "BOUNDARY",
			})

		case OpIf:
			result = append(result, OptimizedInstruction{
				Op:        "IF",
				ExprIndex: inst.ExprIndex,
			})

		case OpElseIf:
			result = append(result, OptimizedInstruction{
				Op:        "ELSEIF",
				ExprIndex: inst.ExprIndex,
			})

		case OpElse:
			result = append(result, OptimizedInstruction{
				Op: "ELSE",
			})

		case OpEnd:
			result = append(result, OptimizedInstruction{
				Op: "END",
			})

		case OpLoopStart:
			result = append(result, OptimizedInstruction{
				Op:                  "LOOP_START",
				Variable:            inst.Variable,
				CollectionExprIndex: inst.CollectionExprIndex,
				EnvIndex:            inst.EnvIndex,
			})

		case OpLoopEnd:
			result = append(result, OptimizedInstruction{
				Op:       "LOOP_END",
				EnvIndex: inst.EnvIndex,
			})

		case OpIfSystemLimit, OpIfSystemOffset:
			// Skip system directives (treat as false)
			skipUntilEnd = 1

		case OpEmitSystemLimit, OpEmitSystemOffset, OpEmitSystemValue:
			// Skip system emissions
		}
	}

	// Merge adjacent EMIT_STATIC instructions
	return MergeAdjacentStatic(result), nil
}

// containsDialect checks if the target dialect is in the list of supported dialects
func containsDialect(dialects []string, target string) bool {
	for _, dialect := range dialects {
		if dialect == target {
			return true
		}
	}
	return false
}

// MergeAdjacentStatic merges adjacent EMIT_STATIC instructions
func MergeAdjacentStatic(instructions []OptimizedInstruction) []OptimizedInstruction {
	if len(instructions) == 0 {
		return instructions
	}

	var result []OptimizedInstruction
	var currentStatic strings.Builder
	hasStatic := false

	flushStatic := func() {
		if hasStatic {
			result = append(result, OptimizedInstruction{
				Op:    "EMIT_STATIC",
				Value: currentStatic.String(),
			})
			currentStatic.Reset()
			hasStatic = false
		}
	}

	for _, inst := range instructions {
		if inst.Op == "EMIT_STATIC" {
			currentStatic.WriteString(inst.Value)
			hasStatic = true
		} else {
			flushStatic()
			result = append(result, inst)
		}
	}

	flushStatic()
	return result
}

// HasDynamicInstructions checks if instructions contain dynamic elements (IF, FOR, etc.)
func HasDynamicInstructions(instructions []OptimizedInstruction) bool {
	for _, inst := range instructions {
		switch inst.Op {
		case "IF", "ELSEIF", "ELSE", "LOOP_START", "LOOP_END":
			return true
		}
	}
	return false
}
