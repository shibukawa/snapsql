package codegenerator

import (
	"strconv"
	"strings"

	"github.com/shibukawa/snapsql"
)

// OptimizedInstruction represents an optimized instruction derived from runtime generation.
type OptimizedInstruction struct {
	Op                  string
	Value               string
	ExprIndex           *int
	SqlFragment         string
	Dialects            []string
	Variable            string
	CollectionExprIndex *int
	EnvIndex            *int
	SystemField         string
	Critical            bool
	FallbackCombos      [][]RemovalLiteral
}

// OptimizeInstructions filters and optimizes instructions for a specific dialect.
func OptimizeInstructions(instructions []Instruction, dialect snapsql.Dialect) ([]OptimizedInstruction, error) {
	var result []OptimizedInstruction

	type systemClauseState struct {
		skipping bool
	}

	var systemStack []systemClauseState

	isSkippingSystem := func() bool {
		return len(systemStack) > 0 && systemStack[len(systemStack)-1].skipping
	}

	for i, inst := range instructions {
		if isSkippingSystem() {
			switch inst.Op {
			case OpIfSystemLimit, OpIfSystemOffset:
				systemStack = append(systemStack, systemClauseState{skipping: true})
				continue
			case OpElse:
				systemStack[len(systemStack)-1].skipping = false
				continue
			case OpEnd:
				systemStack = systemStack[:len(systemStack)-1]
				continue
			default:
				continue
			}
		}

		switch inst.Op {
		case OpEmitStatic:
			result = append(result, OptimizedInstruction{Op: "EMIT_STATIC", Value: inst.Value})

		case OpEmitEval:
			result = append(result, OptimizedInstruction{Op: "EMIT_STATIC", Value: "?"})
			result = append(result, OptimizedInstruction{Op: "ADD_PARAM", ExprIndex: inst.ExprIndex})

		case OpEmitUnlessBoundary:
			isStaticContext := true
			boundaryFound := false

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
				// handled by boundary
			} else if isStaticContext && !boundaryFound {
				result = append(result, OptimizedInstruction{Op: "EMIT_STATIC", Value: inst.Value})
			} else {
				result = append(result, OptimizedInstruction{Op: "EMIT_UNLESS_BOUNDARY", Value: inst.Value})
			}

		case OpBoundary:
			result = append(result, OptimizedInstruction{Op: "BOUNDARY"})

		case OpIf:
			result = append(result, OptimizedInstruction{Op: "IF", ExprIndex: inst.ExprIndex})

		case OpElseIf:
			result = append(result, OptimizedInstruction{Op: "ELSEIF", ExprIndex: inst.ExprIndex})

		case OpElse:
			if len(systemStack) > 0 {
				if systemStack[len(systemStack)-1].skipping {
					continue
				}

				continue
			}

			result = append(result, OptimizedInstruction{Op: "ELSE"})

		case OpEnd:
			if len(systemStack) > 0 {
				systemStack = systemStack[:len(systemStack)-1]
				continue
			}

			result = append(result, OptimizedInstruction{Op: "END"})

		case OpLoopStart:
			result = append(result, OptimizedInstruction{
				Op:                  "LOOP_START",
				Variable:            inst.Variable,
				CollectionExprIndex: inst.CollectionExprIndex,
				EnvIndex:            inst.EnvIndex,
			})

		case OpLoopEnd:
			result = append(result, OptimizedInstruction{Op: "LOOP_END", EnvIndex: inst.EnvIndex})

		case OpIfSystemLimit, OpIfSystemOffset:
			systemStack = append(systemStack, systemClauseState{skipping: true})
			continue

		case OpEmitSystemLimit, OpEmitSystemOffset:
			// ignored for static SQL

		case OpEmitSystemValue:
			result = append(result, OptimizedInstruction{Op: "EMIT_STATIC", Value: "?"})
			result = append(result, OptimizedInstruction{Op: "ADD_SYSTEM_PARAM", SystemField: inst.SystemField})

		case OpEmitIfDialect:
			result = append(result, OptimizedInstruction{
				Op:          "EMIT_IF_DIALECT",
				SqlFragment: inst.SqlFragment,
				Dialects:    append([]string(nil), inst.Dialects...),
			})

		case OpFallbackCondition:
			var combos [][]RemovalLiteral
			if len(inst.FallbackCombos) > 0 {
				combos = make([][]RemovalLiteral, len(inst.FallbackCombos))
				for i := range inst.FallbackCombos {
					combos[i] = append([]RemovalLiteral(nil), inst.FallbackCombos[i]...)
				}
			}

			result = append(result, OptimizedInstruction{
				Op:             OpFallbackCondition,
				Value:          inst.Value,
				Critical:       inst.Critical,
				FallbackCombos: combos,
			})
		}
	}

	merged := MergeAdjacentStatic(result)
	merged = applyPlaceholderStyle(merged, dialect)

	return merged, nil
}

// MergeAdjacentStatic merges adjacent EMIT_STATIC instructions.
func MergeAdjacentStatic(instructions []OptimizedInstruction) []OptimizedInstruction {
	if len(instructions) == 0 {
		return instructions
	}

	var (
		result        []OptimizedInstruction
		currentStatic strings.Builder
		hasStatic     bool
	)

	flushStatic := func() {
		if hasStatic {
			result = append(result, OptimizedInstruction{Op: "EMIT_STATIC", Value: currentStatic.String()})
			currentStatic.Reset()

			hasStatic = false
		}
	}

	for _, inst := range instructions {
		if inst.Op == "EMIT_STATIC" {
			currentStatic.WriteString(inst.Value)

			hasStatic = true

			continue
		}

		flushStatic()

		result = append(result, inst)
	}

	flushStatic()

	return result
}

func applyPlaceholderStyle(instructions []OptimizedInstruction, dialect snapsql.Dialect) []OptimizedInstruction {
	d := strings.ToLower(strings.TrimSpace(string(dialect)))
	if d != "postgres" && d != "postgresql" && d != "pgx" && d != "pg" {
		return instructions
	}

	nextIndex := 1

	convert := func(s string) string {
		if s == "" {
			return s
		}

		var b strings.Builder
		b.Grow(len(s) + 4)

		inSingle := false
		inDouble := false

		for idx := range len(s) {
			ch := s[idx]

			switch ch {
			case '\'':
				if !inDouble {
					inSingle = !inSingle
				}

				b.WriteByte(ch)
			case '"':
				if !inSingle {
					inDouble = !inDouble
				}

				b.WriteByte(ch)
			case '?':
				if !inSingle && !inDouble {
					b.WriteByte('$')
					b.WriteString(strconv.Itoa(nextIndex))
					nextIndex++
				} else {
					b.WriteByte(ch)
				}
			default:
				b.WriteByte(ch)
			}
		}

		return b.String()
	}

	for i := range instructions {
		switch instructions[i].Op {
		case "EMIT_STATIC", "EMIT_UNLESS_BOUNDARY":
			instructions[i].Value = convert(instructions[i].Value)
		case "EMIT_IF_DIALECT":
			instructions[i].SqlFragment = convert(instructions[i].SqlFragment)
		}
	}

	return instructions
}

// HasDynamicInstructions reports whether optimized instructions include runtime control flow.
func HasDynamicInstructions(instructions []OptimizedInstruction) bool {
	for _, inst := range instructions {
		switch inst.Op {
		case "IF", "ELSEIF", "ELSE", "LOOP_START", "LOOP_END", OpEmitSystemFor, OpFallbackCondition:
			return true
		}
	}

	return false
}

// OptimizeLoopBoundaries converts non-terminal EMIT_UNLESS_BOUNDARY instructions inside loops to EMIT_STATIC.
func OptimizeLoopBoundaries(instructions []OptimizedInstruction) []OptimizedInstruction {
	result := make([]OptimizedInstruction, 0, len(instructions))

	for i := 0; i < len(instructions); i++ {
		inst := instructions[i]

		if inst.Op == "LOOP_START" {
			result = append(result, inst)

			loopDepth := 1
			loopStart := i
			loopEnd := -1

		LoopSearch:
			for j := i + 1; j < len(instructions); j++ {
				switch instructions[j].Op {
				case "LOOP_START":
					loopDepth++
				case "LOOP_END":
					loopDepth--
					if loopDepth == 0 {
						loopEnd = j
						break LoopSearch
					}
				}
			}

			if loopEnd == -1 {
				continue
			}

			var boundaryIndices []int

			for j := loopStart + 1; j < loopEnd; j++ {
				if instructions[j].Op == "EMIT_UNLESS_BOUNDARY" {
					boundaryIndices = append(boundaryIndices, j)
				}
			}

			for j := i + 1; j < loopEnd; j++ {
				loopInst := instructions[j]
				if loopInst.Op == "EMIT_UNLESS_BOUNDARY" {
					isLast := len(boundaryIndices) > 0 && j == boundaryIndices[len(boundaryIndices)-1]
					if isLast {
						result = append(result, loopInst)
					} else {
						result = append(result, OptimizedInstruction{Op: "EMIT_STATIC", Value: loopInst.Value})
					}
				} else {
					result = append(result, loopInst)
				}
			}

			result = append(result, instructions[loopEnd])
			i = loopEnd

			continue
		}

		result = append(result, inst)
	}

	return result
}
