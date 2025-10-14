package parsercommon

import (
	"errors"
	"fmt"
	"strings"

	snapsql "github.com/shibukawa/snapsql"
)

// SQ (SubQuery) prefixed types from parserstep7 for better type organization

// SQDependencyGraph manages dependencies between subqueries
type SQDependencyGraph struct {
	nodes        map[string]*SQDependencyNode
	edges        map[string][]string
	scopeManager *SQScopeManager
}

// NewSQDependencyGraph creates a new dependency graph
func NewSQDependencyGraph() *SQDependencyGraph {
	return &SQDependencyGraph{
		nodes:        make(map[string]*SQDependencyNode),
		edges:        make(map[string][]string),
		scopeManager: NewSQScopeManager(),
	}
}

// GetAllNodes returns all nodes in the dependency graph
func (dg *SQDependencyGraph) GetAllNodes() map[string]*SQDependencyNode {
	return dg.nodes
}

// GetAccessibleFieldsForNode returns all field sources available to a specific node
func (dg *SQDependencyGraph) GetAccessibleFieldsForNode(nodeID string) ([]*SQFieldSource, error) {
	node, exists := dg.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNodeNotFound, nodeID)
	}

	var fields []*SQFieldSource

	// Add fields from the node's scope
	if node.Scope != nil {
		fields = append(fields, node.Scope.AvailableFields...)
	}

	// Add fields from dependencies
	for _, depID := range node.Dependencies {
		if depNode, exists := dg.nodes[depID]; exists {
			fields = append(fields, depNode.FieldSources...)
		}
	}

	return fields, nil
}

// ValidateFieldAccessForNode validates if a field can be accessed from a specific node
func (dg *SQDependencyGraph) ValidateFieldAccessForNode(nodeID, fieldName string) error {
	fields, err := dg.GetAccessibleFieldsForNode(nodeID)
	if err != nil {
		return err
	}

	for _, field := range fields {
		if field.Name == fieldName || field.Alias == fieldName {
			return nil // Field is accessible
		}
	}

	return fmt.Errorf("%w: field '%s' from node '%s'", ErrFieldNotAccessible, fieldName, nodeID)
}

// ResolveFieldInNode resolves a field reference from a specific node's perspective
func (dg *SQDependencyGraph) ResolveFieldInNode(nodeID, fieldName string) ([]*SQFieldSource, error) {
	fields, err := dg.GetAccessibleFieldsForNode(nodeID)
	if err != nil {
		return nil, err
	}

	var matches []*SQFieldSource

	for _, field := range fields {
		if field.Name == fieldName || field.Alias == fieldName {
			matches = append(matches, field)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: field '%s' in node '%s'", ErrFieldSourceNotFound, fieldName, nodeID)
	}

	return matches, nil
}

// GetScopeHierarchyVisualization returns a visualization of the scope hierarchy
func (dg *SQDependencyGraph) GetScopeHierarchyVisualization() string {
	if dg.scopeManager == nil {
		return "No scope manager available"
	}

	var result strings.Builder

	result.WriteString("Scope Hierarchy:\n")

	// Start from root scope
	if dg.scopeManager.rootScope != nil {
		dg.visualizeScope(dg.scopeManager.rootScope, 0, &result)
	}

	return result.String()
}

// visualizeScope helper function for scope visualization
func (dg *SQDependencyGraph) visualizeScope(scope *SQScope, depth int, result *strings.Builder) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(result, "%s- %s (%d fields)\n", indent, scope.ID, len(scope.AvailableFields))

	for _, child := range scope.ChildScopes {
		dg.visualizeScope(child, depth+1, result)
	}
}

// AddFieldSourceToNode adds a field source to a specific node
func (dg *SQDependencyGraph) AddFieldSourceToNode(nodeID string, fieldSource *SQFieldSource) error {
	node, exists := dg.nodes[nodeID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, nodeID)
	}

	node.FieldSources = append(node.FieldSources, fieldSource)

	return nil
}

// AddNode adds a new dependency node to the graph
func (dg *SQDependencyGraph) AddNode(node *SQDependencyNode) {
	if dg.nodes == nil {
		dg.nodes = make(map[string]*SQDependencyNode)
	}

	dg.nodes[node.ID] = node
}

// SQDependencyNode represents a node in the dependency graph
type SQDependencyNode struct {
	ID           string              // Unique ID
	Statement    StatementNode       // Statement reference
	NodeType     SQDependencyType    // Node type
	Dependencies []string            // Dependent node IDs
	FieldSources []*SQFieldSource    // Field sources produced by this node
	TableRefs    []*SQTableReference // Table references used by this node
	Scope        *SQScope            // Scope information for this node
}

// SQDependencyType represents the type of dependency node
type SQDependencyType int

const (
	SQDependencyCTE            SQDependencyType = iota // CTE
	SQDependencySubquery                               // FROM/SELECT clause subquery
	SQDependencyMain                                   // Main query
	SQDependencyFromSubquery                           // FROM clause subquery
	SQDependencySelectSubquery                         // SELECT clause subquery
)

func (dt SQDependencyType) String() string {
	switch dt {
	case SQDependencyCTE:
		return "CTE"
	case SQDependencySubquery:
		return "Subquery"
	case SQDependencyMain:
		return "Main"
	case SQDependencyFromSubquery:
		return "FromSubquery"
	case SQDependencySelectSubquery:
		return "SelectSubquery"
	default:
		return "Unknown"
	}
}

// SQFieldSource represents the source information of a field
type SQFieldSource struct {
	Name        string            // Field name
	Alias       string            // Alias (if any)
	SourceType  SQSourceType      // Type of source
	TableSource *SQTableReference // Table source (for SourceType.Table)
	ExprSource  string            // Expression source (for SourceType.Expression)
	SubqueryRef string            // Subquery reference ID (for SourceType.Subquery)
	Scope       string            // Scope ID
}

// SQSourceType represents the type of field source
type SQSourceType int

const (
	SQSourceTypeTable      SQSourceType = iota // Table field
	SQSourceTypeExpression                     // Calculated expression
	SQSourceTypeSubquery                       // Subquery
	SQSourceTypeAggregate                      // Aggregate function
	SQSourceTypeLiteral                        // Literal value
)

func (st SQSourceType) String() string {
	switch st {
	case SQSourceTypeTable:
		return "Table"
	case SQSourceTypeExpression:
		return "Expression"
	case SQSourceTypeSubquery:
		return "Subquery"
	case SQSourceTypeAggregate:
		return "Aggregate"
	case SQSourceTypeLiteral:
		return "Literal"
	default:
		return "Unknown"
	}
}

// SQTableReference represents table reference information
type SQTableReference struct {
	Name       string           // Table name or alias used in the query
	RealName   string           // Actual table name (for base tables) or CTE/subquery name
	QueryName  string           // CTE or subquery alias (e.g., "sq" for "AS sq", "cte" for CTE name)
	Schema     string           // Schema name
	IsSubquery bool             // Whether this is a subquery
	SubqueryID string           // Subquery ID (if subquery)
	Fields     []*SQFieldSource // Available fields
	// Added classifications for inspect/analysis consumers
	Join    JoinType           // Join type relative to preceding table (main is JoinNone)
	Context SQTableContextKind // Context classification: main|join|cte|subquery
}

// GetField returns a field by name
func (tr *SQTableReference) GetField(fieldName string) *SQFieldSource {
	for _, field := range tr.Fields {
		if field.Name == fieldName || field.Alias == fieldName {
			return field
		}
	}

	return nil
}

// SQTableContextKind represents the context where a table reference exists
type SQTableContextKind int

const (
	SQTableContextMain SQTableContextKind = iota
	SQTableContextJoin
	SQTableContextCTE
	SQTableContextSubquery
)

func (sk SQTableContextKind) String() string {
	switch sk {
	case SQTableContextMain:
		return "main"
	case SQTableContextJoin:
		return "join"
	case SQTableContextCTE:
		return "cte"
	case SQTableContextSubquery:
		return "subquery"
	default:
		return "unknown"
	}
}

// SQScope represents a variable scope in SQL execution
type SQScope struct {
	ID              string                       // Unique scope ID
	ParentScope     *SQScope                     // Parent scope (for nested queries)
	ChildScopes     []*SQScope                   // Child scopes
	AvailableFields []*SQFieldSource             // Fields available in this scope
	TableAliases    map[string]*SQTableReference // Table aliases in this scope
	SubqueryAliases map[string]string            // Subquery aliases to their IDs
}

// SQScopeManager manages scope hierarchy and field visibility
type SQScopeManager struct {
	scopes       map[string]*SQScope // All scopes by ID
	currentScope *SQScope            // Current active scope
	rootScope    *SQScope            // Root scope
}

// NewSQScopeManager creates a new scope manager
func NewSQScopeManager() *SQScopeManager {
	rootScope := &SQScope{
		ID:              "root",
		ParentScope:     nil,
		ChildScopes:     make([]*SQScope, 0),
		AvailableFields: make([]*SQFieldSource, 0),
		TableAliases:    make(map[string]*SQTableReference),
		SubqueryAliases: make(map[string]string),
	}

	return &SQScopeManager{
		scopes:       map[string]*SQScope{"root": rootScope},
		currentScope: rootScope,
		rootScope:    rootScope,
	}
}

// CreateScope creates a new scope as a child of the current scope
func (sm *SQScopeManager) CreateScope(id string) *SQScope {
	newScope := &SQScope{
		ID:              id,
		ParentScope:     sm.currentScope,
		ChildScopes:     make([]*SQScope, 0),
		AvailableFields: make([]*SQFieldSource, 0),
		TableAliases:    make(map[string]*SQTableReference),
		SubqueryAliases: make(map[string]string),
	}

	sm.currentScope.ChildScopes = append(sm.currentScope.ChildScopes, newScope)
	sm.scopes[id] = newScope

	return newScope
}

// EnterScope enters the specified scope
func (sm *SQScopeManager) EnterScope(scopeID string) error {
	scope, exists := sm.scopes[scopeID]
	if !exists {
		return fmt.Errorf("%w: %s", ErrScopeNotFound, scopeID)
	}

	sm.currentScope = scope

	return nil
}

// GetCurrentScope returns the current active scope
func (sm *SQScopeManager) GetCurrentScope() *SQScope {
	return sm.currentScope
}

// GetScope returns a scope by ID
func (sm *SQScopeManager) GetScope(id string) (*SQScope, bool) {
	scope, exists := sm.scopes[id]
	return scope, exists
}

// SQErrorType represents the type of parsing error
type SQErrorType int

const (
	SQErrorTypeUnknown SQErrorType = iota
	SQErrorTypeCircularDependency
	SQErrorTypeUnresolvedReference
	SQErrorTypeInvalidSubquery
	SQErrorTypeScopeViolation
	SQErrorTypeTypeIncompatibility
	SQErrorTypeSyntaxError
)

// String returns the string representation of SQErrorType
func (et SQErrorType) String() string {
	switch et {
	case SQErrorTypeCircularDependency:
		return "CircularDependency"
	case SQErrorTypeUnresolvedReference:
		return "UnresolvedReference"
	case SQErrorTypeInvalidSubquery:
		return "InvalidSubquery"
	case SQErrorTypeScopeViolation:
		return "ScopeViolation"
	case SQErrorTypeTypeIncompatibility:
		return "TypeIncompatibility"
	case SQErrorTypeSyntaxError:
		return "SyntaxError"
	default:
		return "Unknown"
	}
}

// SQPosition represents a position in the source code
type SQPosition struct {
	Line   int    // Line number (1-based)
	Column int    // Column number (1-based)
	Offset int    // Byte offset (0-based)
	File   string // Source file name
}

// String returns a string representation of the position
func (p SQPosition) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}

	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// SQParseError represents a detailed parsing error
type SQParseError struct {
	Type        SQErrorType
	Message     string
	Position    SQPosition
	Context     string   // Surrounding context
	Suggestions []string // Suggested fixes
	RelatedIDs  []string // Related node/field IDs
}

// Error implements the error interface
func (pe *SQParseError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] %s", pe.Type.String(), pe.Message))

	if pe.Position.Line > 0 {
		sb.WriteString(" at " + pe.Position.String())
	}

	if pe.Context != "" {
		sb.WriteString("\nContext: " + pe.Context)
	}

	if len(pe.Suggestions) > 0 {
		sb.WriteString("\nSuggestions:")

		for _, suggestion := range pe.Suggestions {
			sb.WriteString("\n  - " + suggestion)
		}
	}

	if len(pe.RelatedIDs) > 0 {
		sb.WriteString("\nRelated: " + strings.Join(pe.RelatedIDs, ", "))
	}

	return sb.String()
}

// AddDependency adds a dependency edge from 'from' to 'to'
func (dg *SQDependencyGraph) AddDependency(from, to string) error {
	if _, exists := dg.nodes[from]; !exists {
		return fmt.Errorf("%w: source %s", snapsql.ErrNodeNotFound, from)
	}

	if _, exists := dg.nodes[to]; !exists {
		return fmt.Errorf("%w: target %s", snapsql.ErrNodeNotFound, to)
	}

	if dg.edges == nil {
		dg.edges = make(map[string][]string)
	}

	dg.edges[from] = append(dg.edges[from], to)

	return nil
}

// GetProcessingOrder returns the processing order using topological sorting
func (dg *SQDependencyGraph) GetProcessingOrder() ([]string, error) {
	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)

	// Calculate in-degrees
	for nodeID := range dg.nodes {
		inDegree[nodeID] = 0
	}

	for _, edges := range dg.edges {
		for _, to := range edges {
			inDegree[to]++
		}
	}

	// Add nodes with in-degree 0 to queue
	queue := []string{}

	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	result := []string{}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		result = append(result, current)

		// Reduce in-degree of neighboring nodes
		for _, neighbor := range dg.edges[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for circular dependencies
	if len(result) != len(dg.nodes) {
		return nil, ErrCircularDependency
	}

	return result, nil
}

// GetNode returns a node by ID
func (dg *SQDependencyGraph) GetNode(id string) *SQDependencyNode {
	return dg.nodes[id]
}

// GetScopeManager returns the scope manager
func (dg *SQDependencyGraph) GetScopeManager() *SQScopeManager {
	return dg.scopeManager
}

// GetAllEdges returns all edges in the dependency graph
func (dg *SQDependencyGraph) GetAllEdges() map[string][]string {
	return dg.edges
}

// GetEdges returns edges for a specific node
func (dg *SQDependencyGraph) GetEdges(nodeID string) []string {
	if edges, exists := dg.edges[nodeID]; exists {
		return edges
	}

	return []string{}
}

// Common errors for subquery processing
var (
	ErrSubqueryParseError  = errors.New("subquery parse error")
	ErrFieldSourceNotFound = errors.New("field source not found")
	ErrTableNotFound       = errors.New("table not found in scope")
	ErrCircularDependency  = errors.New("circular dependency in subquery")
	ErrScopeViolation      = errors.New("scope violation in field reference")
	ErrNodeNotFound        = errors.New("node not found")
	ErrFieldNotAccessible  = errors.New("field is not accessible")
	ErrScopeNotFound       = errors.New("scope not found")
	ErrNoDependencyGraph   = errors.New("no dependency graph available")
)
