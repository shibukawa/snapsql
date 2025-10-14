package explain

import (
	"context"
	"testing"
	"time"
)

func TestAnalyzeFullScanWarning(t *testing.T) {
	node := &PlanNode{
		NodeType:        "Seq Scan",
		Schema:          "public",
		Relation:        "users",
		ActualRows:      10,
		PlanRows:        100,
		ActualTotalTime: 2.0,
		QueryPath:       "main",
	}

	doc := &PlanDocument{Root: []*PlanNode{node}}

	eval, err := Analyze(context.Background(), doc, AnalyzerOptions{
		Threshold: 1 * time.Second,
		Tables: map[string]TableMetadata{
			"public.users": {ExpectedRows: 1000, AllowFullScan: false},
		},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if len(eval.Warnings) == 0 {
		t.Fatalf("expected warnings")
	}
}

func TestAnalyzeSlowQueryWarning(t *testing.T) {
	node := &PlanNode{
		NodeType:        "Index Scan",
		Schema:          "public",
		Relation:        "orders",
		ActualRows:      5,
		PlanRows:        5,
		ActualTotalTime: 400.0,
		QueryPath:       "main",
	}

	doc := &PlanDocument{Root: []*PlanNode{node}}

	eval, err := Analyze(context.Background(), doc, AnalyzerOptions{
		Threshold: 2 * time.Second,
		Tables: map[string]TableMetadata{
			"public.orders": {ExpectedRows: 10000, AllowFullScan: true},
		},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if len(eval.Estimates) == 0 {
		t.Fatalf("expected estimates")
	}

	hasSlow := false

	for _, w := range eval.Warnings {
		if w.Kind == WarningSlowQuery {
			hasSlow = true
		}
	}

	if !hasSlow {
		t.Fatalf("expected slow query warning")
	}
}

func TestAnalyzeSQLiteFullScanWarning(t *testing.T) {
	doc := &PlanDocument{
		Dialect: "sqlite",
		RawText: "SCAN TABLE tasks\nSEARCH TABLE users USING INDEX",
	}

	eval, err := Analyze(context.Background(), doc, AnalyzerOptions{
		Tables: map[string]TableMetadata{
			"tasks": {ExpectedRows: 1000, AllowFullScan: false},
		},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if len(eval.Warnings) == 0 {
		t.Fatalf("expected full scan warning for sqlite plan")
	}
}

func TestAnalyzeSQLiteFullScanAllowed(t *testing.T) {
	doc := &PlanDocument{
		Dialect: "sqlite",
		RawText: "SCAN TABLE tasks",
	}

	eval, err := Analyze(context.Background(), doc, AnalyzerOptions{
		Tables: map[string]TableMetadata{
			"tasks": {ExpectedRows: 1000, AllowFullScan: true},
		},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if len(eval.Warnings) != 0 {
		t.Fatalf("did not expect warning when full scan allowed")
	}
}
