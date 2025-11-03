package explain

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/shibukawa/snapsql"
)

// Collect executes an EXPLAIN statement for the given SQL.
func Collect(ctx context.Context, opts CollectorOptions) (*PlanDocument, error) {
	runner := opts.Runner
	if runner == nil && opts.DB != nil {
		runner = opts.DB
	}

	if runner == nil {
		return nil, ErrNoDatabase
	}

	dialect := normalizeDialect(opts.Dialect)

	query, analyze, err := buildExplainQuery(opts.Dialect, opts)
	if err != nil {
		return nil, err
	}

	execCtx := ctx

	cancel := func() {}
	if opts.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()

	rows, err := runner.QueryContext(execCtx, query, opts.Args...)
	if err != nil {
		return nil, fmt.Errorf("explain: query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("explain: failed to fetch columns: %w", err)
	}

	if !rows.Next() {
		return nil, ErrNoPlanRows
	}

	values := make([]any, len(cols))

	raw := make([]sql.RawBytes, len(cols))
	for i := range values {
		values[i] = &raw[i]
	}

	if err := rows.Scan(values...); err != nil {
		return nil, fmt.Errorf("explain: failed to scan row: %w", err)
	}

	var doc PlanDocument

	doc.Dialect = opts.Dialect

	switch dialect {
	case "postgres", "postgresql", "pgx":
		doc.RawJSON = extractFirstColumn(raw)
	case "mysql", "mariadb":
		doc.RawJSON = extractFirstColumn(raw)
	case "sqlite", "sqlite3":
		var builder strings.Builder
		builder.WriteString(formatSQLiteRow(cols, raw))

		for rows.Next() {
			nextRaw := make([]sql.RawBytes, len(cols))

			nextVals := make([]any, len(cols))
			for i := range nextVals {
				nextVals[i] = &nextRaw[i]
			}

			if err := rows.Scan(nextVals...); err != nil {
				return nil, fmt.Errorf("explain: failed to scan row: %w", err)
			}

			builder.WriteByte('\n')
			builder.WriteString(formatSQLiteRow(cols, nextRaw))
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("explain: failed to iterate rows: %w", err)
		}

		doc.RawText = builder.String()
	default:
		return nil, ErrUnsupportedDialect
	}

	if len(doc.RawJSON) == 0 && doc.RawText == "" {
		doc.RawText = string(raw[0])
	}

	doc.Warnings = nil

	// Parse plan JSON when available
	if len(doc.RawJSON) > 0 {
		nodes, parseErr := ParsePlanJSON(opts.Dialect, doc.RawJSON)
		if parseErr != nil {
			doc.Warnings = append(doc.Warnings, parseErr)
		} else {
			doc.Root = nodes
		}
	}

	// Record whether ANALYZE was requested
	_ = analyze

	return &doc, nil
}

func buildExplainQuery(d snapsql.Dialect, opts CollectorOptions) (string, bool, error) {
	dialect := normalizeDialect(d)

	switch dialect {
	case "postgres", "postgresql", "pgx":
		analyzeFlag := "false"
		if opts.Analyze {
			analyzeFlag = "true"
		}

		return fmt.Sprintf("EXPLAIN (ANALYZE %s, VERBOSE true, FORMAT JSON) %s", analyzeFlag, opts.SQL), opts.Analyze, nil
	case "mysql", "mariadb":
		if opts.Analyze {
			return "EXPLAIN ANALYZE FORMAT=JSON " + opts.SQL, true, nil
		}

		return "EXPLAIN FORMAT=JSON " + opts.SQL, false, nil
	case "sqlite", "sqlite3":
		return "EXPLAIN QUERY PLAN " + opts.SQL, false, nil
	default:
		return "", false, ErrUnsupportedDialect
	}
}

func formatSQLiteRow(cols []string, raw []sql.RawBytes) string {
	if len(cols) == 0 || len(raw) == 0 {
		return ""
	}

	detailIdx := -1

	for i, name := range cols {
		if strings.EqualFold(name, "detail") {
			detailIdx = i
			break
		}
	}

	if detailIdx >= 0 && detailIdx < len(raw) {
		if raw[detailIdx] == nil {
			return ""
		}

		return string(raw[detailIdx])
	}

	parts := make([]string, 0, len(cols))
	for i, name := range cols {
		value := "<nil>"
		if i < len(raw) && raw[i] != nil {
			value = string(raw[i])
		}

		parts = append(parts, fmt.Sprintf("%s=%s", name, value))
	}

	return strings.Join(parts, " | ")
}

func extractFirstColumn(raw []sql.RawBytes) []byte {
	if len(raw) == 0 {
		return nil
	}

	b := raw[0]
	if b == nil {
		return nil
	}
	// Copy to avoid retaining reference to underlying buffer
	return append([]byte(nil), []byte(b)...)
}
