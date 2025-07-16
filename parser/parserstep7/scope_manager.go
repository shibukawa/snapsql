package parserstep7

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// Type aliases for backward compatibility
type (
	Scope        = cmn.SQScope
	ScopeManager = cmn.SQScopeManager
)

// NewScopeManager creates a new scope manager
func NewScopeManager() *ScopeManager {
	return cmn.NewSQScopeManager()
}
