package intermediate

import (
	"fmt"
	"strings"
)

// GenerateError collects multiple generation errors
type GenerateError struct {
	Errors []string
}

// Add adds an error to the collection
func (ge *GenerateError) Add(err error) {
	if err != nil {
		ge.Errors = append(ge.Errors, err.Error())
	}
}

// AddError adds an error to the collection (alias for Add for backward compatibility)
func (ge *GenerateError) AddError(err error) {
	ge.Add(err)
}

// HasErrors returns true if there are any errors
func (ge *GenerateError) HasErrors() bool {
	return len(ge.Errors) > 0
}

// Error implements the error interface
func (ge *GenerateError) Error() string {
	if len(ge.Errors) == 0 {
		return ""
	}
	if len(ge.Errors) == 1 {
		return ge.Errors[0]
	}
	return fmt.Sprintf("Multiple generation errors:\n- %s", strings.Join(ge.Errors, "\n- "))
}

// Clear clears all errors
func (ge *GenerateError) Clear() {
	ge.Errors = ge.Errors[:0]
}
