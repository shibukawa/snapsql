package parserstep7

import "errors"

// FieldSource represents the source information of a field
type FieldSource struct {
	Name        string          // Field name
	Alias       string          // Alias (if any)
	SourceType  SourceType      // Type of source
	TableSource *TableReference // Table source (for SourceType.Table)
	ExprSource  string          // Expression source (for SourceType.Expression)
	SubqueryRef string          // Subquery reference ID (for SourceType.Subquery)
	Scope       string          // Scope ID
}

// SourceType represents the type of field source
type SourceType int

const (
	SourceTypeTable      SourceType = iota // Table field
	SourceTypeExpression                   // Calculated expression
	SourceTypeSubquery                     // Subquery
	SourceTypeAggregate                    // Aggregate function
	SourceTypeLiteral                      // Literal value
)

func (st SourceType) String() string {
	switch st {
	case SourceTypeTable:
		return "Table"
	case SourceTypeExpression:
		return "Expression"
	case SourceTypeSubquery:
		return "Subquery"
	case SourceTypeAggregate:
		return "Aggregate"
	case SourceTypeLiteral:
		return "Literal"
	default:
		return "Unknown"
	}
}

// TableReference represents table reference information
type TableReference struct {
	Name       string         // Table name or alias
	RealName   string         // Actual table name
	Schema     string         // Schema name
	IsSubquery bool           // Whether this is a subquery
	SubqueryID string         // Subquery ID (if subquery)
	Fields     []*FieldSource // Available fields
}

// GetField returns a field by name
func (tr *TableReference) GetField(fieldName string) *FieldSource {
	for _, field := range tr.Fields {
		if field.Name == fieldName || field.Alias == fieldName {
			return field
		}
	}
	return nil
}

// Common errors
var (
	ErrSubqueryParseError  = errors.New("subquery parse error")
	ErrFieldSourceNotFound = errors.New("field source not found")
	ErrTableNotFound       = errors.New("table not found in scope")
	ErrCircularDependency  = errors.New("circular dependency in subquery")
	ErrScopeViolation      = errors.New("scope violation in field reference")
	ErrAmbiguousField      = errors.New("ambiguous field reference")
	ErrUnknownSourceType   = errors.New("unknown field source type")
)
