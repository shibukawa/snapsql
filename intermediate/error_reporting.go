package intermediate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Sentinel errors for validation
var (
	ErrInvalidPositionFormat = errors.New("instruction has invalid position format")
	ErrInvalidLineNumber     = errors.New("instruction has invalid line number")
	ErrInvalidColumnNumber   = errors.New("instruction has invalid column number")
)

// ExecutionError represents an error that occurred during instruction execution
type ExecutionError struct {
	Message     string `json:"message"`
	Instruction int    `json:"instruction_index"`
	Pos         string `json:"pos"` // Position "line:column" from original template (required)
	SourceFile  string `json:"source_file,omitempty"`
	SourceLine  string `json:"source_line,omitempty"` // The actual line from source
}

// Error implements the error interface
func (e *ExecutionError) Error() string {
	if e.SourceFile != "" {
		return fmt.Sprintf("%s:%s: %s", e.SourceFile, e.Pos, e.Message)
	}
	return fmt.Sprintf("position %s: %s", e.Pos, e.Message)
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
		parts := strings.Split(e.Pos, ":")
		if len(parts) == 2 {
			col, err := strconv.Atoi(parts[1])
			if err == nil && col > 1 {
				builder.WriteString(strings.Repeat(" ", col-1))
				builder.WriteString("^")
				builder.WriteString("\n")
			}
		}
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
		parts := strings.Split(instruction.Pos, ":")
		if len(parts) == 2 {
			lineNum, parseErr := strconv.Atoi(parts[0])
			if parseErr == nil && lineNum > 0 && lineNum <= len(lines) {
				err.SourceLine = lines[lineNum-1]
			}
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
			Pos:         "0:0", // Default position
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
		parts := strings.Split(inst.Pos, ":")
		if len(parts) != 2 {
			validationErrors = append(validationErrors, fmt.Errorf("%w: expected 'line:column', got %v for instruction %d (%s)", ErrInvalidPositionFormat, inst.Pos, i, inst.Op))
			continue
		}

		lineNum, err1 := strconv.Atoi(parts[0])
		colNum, err2 := strconv.Atoi(parts[1])
		
		if err1 != nil || err2 != nil {
			validationErrors = append(validationErrors, fmt.Errorf("%w: invalid format '%s' for instruction %d (%s)", ErrInvalidPositionFormat, inst.Pos, i, inst.Op))
			continue
		}

		// Validate line number
		if lineNum < 1 || lineNum > len(lines) {
			validationErrors = append(validationErrors, fmt.Errorf("%w: %d for instruction %d (%s)", ErrInvalidLineNumber, lineNum, i, inst.Op))
		}

		// Validate column number
		if lineNum >= 1 && lineNum <= len(lines) {
			lineContent := lines[lineNum-1]
			if colNum < 1 || colNum > len(lineContent)+1 {
				validationErrors = append(validationErrors, fmt.Errorf("%w: %d for instruction %d (%s)", ErrInvalidColumnNumber, colNum, i, inst.Op))
			}
		}
	}

	return validationErrors
}
