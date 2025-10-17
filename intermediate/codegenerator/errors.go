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

// ErrCTENotSupported is returned when CTE is encountered (not yet implemented).
var ErrCTENotSupported = errors.New("CTE not supported (Phase 4 implementation in progress)")

// ErrLoopMismatch is returned when loop directive matching fails (loop/endloop).
var ErrLoopMismatch = errors.New("loop directive mismatch")

// ErrLoopNesting is returned when loop nesting exceeds maximum depth.
var ErrLoopNesting = errors.New("loop nesting too deep")
