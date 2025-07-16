package parserstep7

import (
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *cmn.SQDependencyGraph {
	return cmn.NewSQDependencyGraph()
}
