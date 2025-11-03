package explain

import (
	"context"
	"database/sql"
	"time"

	"github.com/shibukawa/snapsql"
)

// Queryable represents the minimal query interface required by the collector.
type Queryable interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// CollectorOptions describes the information required to execute an EXPLAIN command.
type CollectorOptions struct {
	DB      *sql.DB
	Runner  Queryable
	Dialect snapsql.Dialect
	SQL     string
	Args    []any
	Timeout time.Duration
	Analyze bool
	Format  string // optional hint, e.g. "json"
}

// PlanDocument stores the raw plan output and its parsed representation.
type PlanDocument struct {
	Dialect  snapsql.Dialect
	RawJSON  []byte
	RawText  string
	Root     []*PlanNode
	Warnings []error
}

// PlanNode represents a single execution-plan node.
type PlanNode struct {
	ID              string
	NodeType        string
	Relation        string
	Schema          string
	Alias           string
	AccessType      string
	QueryPath       string
	ActualRows      float64
	PlanRows        float64
	ActualTotalTime float64
	EstimatedCost   float64
	Children        []*PlanNode
}

// TableMetadata describes expected runtime characteristics for a table.
type TableMetadata struct {
	ExpectedRows  int64
	AllowFullScan bool
}

// AnalyzerOptions configures the analysis step.
type AnalyzerOptions struct {
	Threshold time.Duration
	Tables    map[string]TableMetadata
}

// PerformanceEvaluation captures warnings and metrics derived from the plan.
type PerformanceEvaluation struct {
	Warnings  []Warning
	Estimates []QueryEstimate
}

// Warning conveys issues detected while analyzing the plan.
type Warning struct {
	Kind      WarningKind
	QueryPath string
	Message   string
	Tables    []string
}

// WarningKind enumerates warning categories.
type WarningKind string

const (
	WarningFullScan   WarningKind = "full_scan"
	WarningSlowQuery  WarningKind = "slow_query"
	WarningParseError WarningKind = "plan_parse_error"
)

// QueryEstimate summarises the estimated runtime characteristics of a sub-plan.
type QueryEstimate struct {
	QueryPath   string
	Actual      time.Duration
	Estimated   time.Duration
	Threshold   time.Duration
	ScaleFactor float64
}
