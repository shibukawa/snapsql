package intermediate

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for validation
var (
	ErrInvalidPositionFormat = errors.New("instruction has invalid position format")
	ErrInvalidLineNumber     = errors.New("instruction has invalid line number")
	ErrInvalidColumnNumber   = errors.New("instruction has invalid column number")
	ErrInvalidOffset         = errors.New("instruction has invalid offset")
)

// ExecutionError represents an error that occurred during instruction execution
type ExecutionError struct {
	Message     string `json:"message"`
	Instruction int    `json:"instruction_index"`
	Pos         []int  `json:"pos"` // Position [line, column, offset] from original template (required)
	SourceFile  string `json:"source_file,omitempty"`
	SourceLine  string `json:"source_line,omitempty"` // The actual line from source
}

// Error implements the error interface
func (e *ExecutionError) Error() string {
	if e.SourceFile != "" {
		return fmt.Sprintf("%s:%d:%d: %s", e.SourceFile, e.Pos[0], e.Pos[1], e.Message)
	}
	return fmt.Sprintf("line %d, column %d: %s", e.Pos[0], e.Pos[1], e.Message)
}

// DetailedError returns a detailed error message with source context
func (e *ExecutionError) DetailedError() string {
	var builder strings.Builder

	// Basic error message
	builder.WriteString(e.Error())
	builder.WriteString("\n")

	// Add source context if available
	if e.SourceLine != "" {
		builder.WriteString("\n")
		builder.WriteString(e.SourceLine)
		builder.WriteString("\n")

		// Add pointer to the error location
		if e.Pos[1] > 1 {
			builder.WriteString(strings.Repeat(" ", e.Pos[1]-1))
		}
		builder.WriteString("^")
		builder.WriteString("\n")
	}

	return builder.String()
}

// NewExecutionError creates a new execution error with position information
func NewExecutionError(message string, instructionIndex int, instruction *Instruction, sourceFile, sourceContent string) *ExecutionError {
	err := &ExecutionError{
		Message:     message,
		Instruction: instructionIndex,
		Pos:         instruction.Pos, // pos is always present
		SourceFile:  sourceFile,
	}

	// Extract the source line if we have the content
	if sourceContent != "" {
		lines := strings.Split(sourceContent, "\n")
		lineNum := instruction.Pos[0]
		if lineNum > 0 && lineNum <= len(lines) {
			err.SourceLine = lines[lineNum-1]
		}
	}

	return err
}

// ErrorReporter helps create detailed error messages for instruction execution
type ErrorReporter struct {
	SourceFile    string
	SourceContent string
	Instructions  []Instruction
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(sourceFile, sourceContent string, instructions []Instruction) *ErrorReporter {
	return &ErrorReporter{
		SourceFile:    sourceFile,
		SourceContent: sourceContent,
		Instructions:  instructions,
	}
}

// ReportError creates a detailed execution error
func (er *ErrorReporter) ReportError(message string, instructionIndex int) *ExecutionError {
	if instructionIndex < 0 || instructionIndex >= len(er.Instructions) {
		// Invalid instruction index - create error without position info
		return &ExecutionError{
			Message:     message,
			Instruction: instructionIndex,
			Pos:         []int{0, 0, 0}, // Default position
			SourceFile:  er.SourceFile,
		}
	}

	instruction := &er.Instructions[instructionIndex]
	return NewExecutionError(message, instructionIndex, instruction, er.SourceFile, er.SourceContent)
}

// ValidateInstructionPositions validates that all instructions have valid position information
func ValidateInstructionPositions(instructions []Instruction, sourceContent string) []error {
	var validationErrors []error
	lines := strings.Split(sourceContent, "\n")

	for i, inst := range instructions {
		if len(inst.Pos) < 3 {
			validationErrors = append(validationErrors, fmt.Errorf("%w: expected [line, column, offset], got %v for instruction %d (%s)", ErrInvalidPositionFormat, inst.Pos, i, inst.Op))
			continue
		}

		line, col, offset := inst.Pos[0], inst.Pos[1], inst.Pos[2]

		// Validate line number
		if line < 1 || line > len(lines) {
			validationErrors = append(validationErrors, fmt.Errorf("%w: %d for instruction %d (%s)", ErrInvalidLineNumber, line, i, inst.Op))
		}

		// Validate column number
		if line >= 1 && line <= len(lines) {
			lineContent := lines[line-1]
			if col < 1 || col > len(lineContent)+1 {
				validationErrors = append(validationErrors, fmt.Errorf("%w: %d for instruction %d (%s)", ErrInvalidColumnNumber, col, i, inst.Op))
			}
		}

		// Validate offset
		if offset < 0 || offset > len(sourceContent) {
			validationErrors = append(validationErrors, fmt.Errorf("%w: %d for instruction %d (%s)", ErrInvalidOffset, offset, i, inst.Op))
		}
	}

	return validationErrors
}
