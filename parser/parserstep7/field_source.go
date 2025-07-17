package parserstep7

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Type aliases for backward compatibility
type (
	FieldSource    = cmn.SQFieldSource
	SourceType     = cmn.SQSourceType
	TableReference = cmn.SQTableReference
)

// Constants for source types
const (
	SourceTypeTable      = cmn.SQSourceTypeTable
	SourceTypeExpression = cmn.SQSourceTypeExpression
	SourceTypeSubquery   = cmn.SQSourceTypeSubquery
	SourceTypeAggregate  = cmn.SQSourceTypeAggregate
	SourceTypeLiteral    = cmn.SQSourceTypeLiteral
)

// Sentinel errors
var (
	ErrSubqueryParseError  = cmn.ErrSubqueryParseError
	ErrFieldSourceNotFound = cmn.ErrFieldSourceNotFound
	ErrTableNotFound       = cmn.ErrTableNotFound
	ErrCircularDependency  = cmn.ErrCircularDependency
	ErrScopeViolation      = cmn.ErrScopeViolation
)
