package explain

import (
	"context"
	"math"
	"strings"
	"time"
)

// Analyze produces performance warnings based on the plan and table metadata.
func Analyze(ctx context.Context, doc *PlanDocument, opts AnalyzerOptions) (*PerformanceEvaluation, error) {
	if doc == nil {
		return &PerformanceEvaluation{}, nil
	}

	eval := &PerformanceEvaluation{}

	for _, root := range doc.Root {
		traverseAnalyze(root, opts, eval)
		analyzeSlowQuery(root, opts, eval)
	}

	if len(doc.Root) == 0 {
		dialect := normalizeDialect(doc.Dialect)
		if dialect == "sqlite" && strings.TrimSpace(doc.RawText) != "" {
			analyzeSQLitePlanText(doc.RawText, opts, eval)
		}
	}

	return eval, nil
}

func traverseAnalyze(node *PlanNode, opts AnalyzerOptions, eval *PerformanceEvaluation) {
	if node == nil {
		return
	}

	if tableKey := canonicalTableKey(node.Schema, node.Relation); tableKey != "" {
		if meta, ok := opts.Tables[tableKey]; ok {
			analyzeFullScan(node, tableKey, meta, eval)
		}
	}

	for _, child := range node.Children {
		traverseAnalyze(child, opts, eval)
	}
}

func analyzeFullScan(node *PlanNode, tableKey string, meta TableMetadata, eval *PerformanceEvaluation) {
	if meta.AllowFullScan {
		return
	}

	if isFullScan(node) {
		eval.Warnings = append(eval.Warnings, Warning{
			Kind:      WarningFullScan,
			QueryPath: node.QueryPath,
			Message:   "full scan detected",
			Tables:    []string{tableKey},
		})
	}
}

func isFullScan(node *PlanNode) bool {
	if node == nil {
		return false
	}

	if node.AccessType != "" {
		at := strings.ToUpper(node.AccessType)
		if at == "ALL" {
			return true
		}
	}

	nt := strings.ToLower(node.NodeType)

	return strings.Contains(nt, "seq scan") || strings.Contains(nt, "table scan")
}

func analyzeSlowQuery(node *PlanNode, opts AnalyzerOptions, eval *PerformanceEvaluation) {
	if opts.Threshold <= 0 {
		return
	}

	actual := durationFromMillis(node.ActualTotalTime)
	if actual <= 0 {
		return
	}

	estimate := estimateDuration(node, opts)
	if estimate > opts.Threshold {
		eval.Warnings = append(eval.Warnings, Warning{
			Kind:      WarningSlowQuery,
			QueryPath: node.QueryPath,
			Message:   "estimated runtime exceeds threshold",
		})
	}

	eval.Estimates = append(eval.Estimates, QueryEstimate{
		QueryPath:   node.QueryPath,
		Actual:      actual,
		Estimated:   estimate,
		Threshold:   opts.Threshold,
		ScaleFactor: computeScaleFactor(node, opts.Tables),
	})
}

func estimateDuration(node *PlanNode, opts AnalyzerOptions) time.Duration {
	scale := computeScaleFactor(node, opts.Tables)

	actual := durationFromMillis(node.ActualTotalTime)
	if scale <= 1 {
		return actual
	}

	if isFullScan(node) {
		return time.Duration(float64(actual) * scale)
	}

	// logarithmic growth for index scans or other operators
	multiplier := 1 + math.Log2(scale)
	if multiplier < 1 {
		multiplier = 1
	}

	return time.Duration(float64(actual) * multiplier)
}

func computeScaleFactor(node *PlanNode, tables map[string]TableMetadata) float64 {
	key := canonicalTableKey(node.Schema, node.Relation)
	if key == "" {
		return 1
	}

	meta, ok := tables[key]
	if !ok || meta.ExpectedRows <= 0 {
		return 1
	}

	actual := node.ActualRows
	if actual <= 0 {
		actual = node.PlanRows
	}

	if actual <= 0 {
		actual = 1
	}

	return float64(meta.ExpectedRows) / actual
}

func canonicalTableKey(schema, relation string) string {
	if relation == "" {
		return ""
	}

	if schema == "" {
		return strings.ToLower(relation)
	}

	return strings.ToLower(schema + "." + relation)
}

func durationFromMillis(ms float64) time.Duration {
	if ms <= 0 {
		return 0
	}

	return time.Duration(ms * float64(time.Millisecond))
}

func analyzeSQLitePlanText(raw string, opts AnalyzerOptions, eval *PerformanceEvaluation) {
	lines := strings.SplitSeq(raw, "\n")
	for line := range lines {
		detail := strings.TrimSpace(line)
		if detail == "" {
			continue
		}

		lower := strings.ToLower(detail)
		if !strings.HasPrefix(lower, "scan") && !strings.Contains(lower, " scan ") {
			continue
		}

		name := extractSQLiteTableName(detail)

		allowScan := false
		if meta, ok := lookupTableMeta(opts.Tables, name); ok {
			allowScan = meta.AllowFullScan
		}

		if allowScan {
			continue
		}

		tables := []string{}
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			tables = append(tables, strings.ToLower(trimmed))
		}

		eval.Warnings = append(eval.Warnings, Warning{
			Kind:    WarningFullScan,
			Message: "full scan detected",
			Tables:  tables,
		})
	}
}

func extractSQLiteTableName(detail string) string {
	fields := strings.Fields(detail)
	for i := range fields {
		if strings.EqualFold(fields[i], "table") && i+1 < len(fields) {
			return trimSQLiteIdentifier(fields[i+1])
		}
	}

	if len(fields) > 1 && strings.EqualFold(fields[0], "scan") {
		return trimSQLiteIdentifier(fields[1])
	}

	return ""
}

func trimSQLiteIdentifier(name string) string {
	name = strings.Trim(name, "`\"[]")
	return name
}

func lookupTableMeta(meta map[string]TableMetadata, name string) (TableMetadata, bool) {
	if meta == nil {
		return TableMetadata{}, false
	}

	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return TableMetadata{}, false
	}

	if value, ok := meta[key]; ok {
		return value, true
	}

	if value, ok := meta["main."+key]; ok {
		return value, true
	}

	for candidate, value := range meta {
		if strings.HasSuffix(candidate, "."+key) {
			return value, true
		}
	}

	return TableMetadata{}, false
}
