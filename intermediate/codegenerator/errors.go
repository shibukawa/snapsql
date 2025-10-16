package codegenerator

import "errors"

// ErrCodeGeneration is the base error for code generation failures.
var ErrCodeGeneration = errors.New("code generation error")

// ErrClauseNil is returned when a clause is nil.
var ErrClauseNil = errors.New("clause is nil")

// ErrDirectiveMismatch is returned when directive matching fails (if/else/end).
var ErrDirectiveMismatch = errors.New("directive mismatch")

// ErrStatementTypeMismatch is returned when statement type is unexpected.
var ErrStatementTypeMismatch = errors.New("statement type mismatch")

// ErrCTENotSupported is returned when CTE is encountered in unsupported phase.
var ErrCTENotSupported = errors.New("CTE not supported in Phase 1")
