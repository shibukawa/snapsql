package inspect

import (
	"fmt"
	"io"

	"github.com/shibukawa/snapsql/parser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	"github.com/shibukawa/snapsql/parser/parserstep1"
	"github.com/shibukawa/snapsql/parser/parserstep2"
	"github.com/shibukawa/snapsql/parser/parserstep3"
	"github.com/shibukawa/snapsql/parser/parserstep4"
	"github.com/shibukawa/snapsql/tokenizer"
)

// Inspect parses SQL and returns a summarized view suitable for JSON.
func Inspect(r io.Reader, opt InspectOptions) (InspectResult, error) {
	var res InspectResult

	b, err := io.ReadAll(r)
	if err != nil {
		return res, fmt.Errorf("read input: %w", err)
	}

	// First, try full parser pipeline with InspectMode (includes Step7). If it fails and not Strict,
	// fall back to lightweight pipeline (step1-4) to produce partial results.
	if parsed, ok, err := tryFullParserWithInspect(string(b), opt); ok {
		return parsed, err
	}

	// Fallback: lightweight pipeline (step1-4) with partial allowed
	tokens, err := tokenizer.Tokenize(string(b))
	if err != nil {
		return res, fmt.Errorf("tokenize: %w", err)
	}

	t1, err := parserstep1.Execute(tokens)
	if err != nil {
		return res, fmt.Errorf("parserstep1: %w", err)
	}

	stmt, err := parserstep2.Execute(t1)
	if err != nil {
		return res, fmt.Errorf("parserstep2: %w", err)
	}

	if err := parserstep3.Execute(stmt); err != nil {
		return res, fmt.Errorf("parserstep3: %w", err)
	}

	// In fallback, relax validations (InspectMode=true) so we can still extract tables
	if err := parserstep4.ExecuteWithOptions(stmt, true); err != nil {
		if opt.Strict {
			return res, fmt.Errorf("parserstep4: %w", err)
		}

		res.Notes = append(res.Notes, "partially parsed due to syntax error")
	}

	res.Statement = kindToString(stmt.Type())
	res.Tables = extractTables(stmt)

	return res, nil
}

// tryFullParserWithInspect runs parser.RawParseWithOptions under InspectMode. Returns (result, ok, err).
// ok=true indicates that the caller should return immediately with (result, err) because parsing succeeded.
// ok=false indicates fallback should be attempted; err may be nil or error to propagate based on Strict.
func tryFullParserWithInspect(sql string, opt InspectOptions) (InspectResult, bool, error) {
	var res InspectResult

	tokens, err := tokenizer.Tokenize(sql)
	if err != nil {
		if opt.Strict {
			return res, true, fmt.Errorf("tokenize: %w", err)
		}

		return res, false, nil
	}

	// Use sentinel FunctionDefinition (empty). Constants are empty.
	fd := &parser.FunctionDefinition{}

	stmt, err := parser.RawParse(tokens, fd, map[string]any{}, parser.Options{InspectMode: true})
	if err != nil {
		if opt.Strict {
			return res, true, err
		}

		return res, false, nil
	}

	res.Statement = kindToString(stmt.Type())

	// Prefer Step7 table references if available; merge join/source info from AST
	step7 := stmt.GetTableReferences()

	base := extractTables(stmt)
	if len(step7) > 0 {
		base = mergeWithStep7(base, step7, collectCTENames(stmt))
	}

	res.Tables = base

	return res, true, nil
}

// mergeWithStep7 overrides name/alias/schema with Step7 SQTableReference info while keeping
// source/joinType determined from AST. Matching is done by alias if present, otherwise by name.
func mergeWithStep7(base []TableRef, step7 map[string]*cmn.SQTableReference, cteNames map[string]struct{}) []TableRef {
	// Build lookup by alias and by real name
	byAlias := map[string]*cmn.SQTableReference{}
	byReal := map[string]*cmn.SQTableReference{}
	cteTargets := map[string]struct{}{}
	subqueryTargets := map[string]struct{}{}
	for name := range cteNames {
		cteTargets[name] = struct{}{}
	}

	for _, tr := range step7 {
		alias := ""
		if tr.RealName != "" && tr.Name != "" && tr.RealName != tr.Name {
			alias = tr.Name
		}

		if alias != "" {
			byAlias[alias] = tr
		}

		key := tr.RealName
		if key == "" {
			key = tr.Name
		}

		if key != "" {
			byReal[key] = tr
		}

		switch tr.Context {
		case cmn.SQTableContextCTE:
			if tr.QueryName != "" {
				cteTargets[tr.QueryName] = struct{}{}
			}
			if tr.RealName != "" {
				cteTargets[tr.RealName] = struct{}{}
			}
		case cmn.SQTableContextSubquery:
			if tr.QueryName != "" {
				subqueryTargets[tr.QueryName] = struct{}{}
			}
			if tr.RealName != "" {
				subqueryTargets[tr.RealName] = struct{}{}
			}
		}
	}

	out := make([]TableRef, len(base))

	for i, t := range base {
		if tr, ok := byAlias[t.Alias]; ok && t.Alias != "" {
			out[i] = overrideFromStep7(t, tr, cteTargets, subqueryTargets)
			continue
		}

		if tr, ok := byReal[t.Name]; ok {
			out[i] = overrideFromStep7(t, tr, cteTargets, subqueryTargets)
			continue
		}

		out[i] = t
	}

	return out
}

func overrideFromStep7(t TableRef, tr *cmn.SQTableReference, cteTargets, subqueryTargets map[string]struct{}) TableRef {
	// Keep base Name/Alias/Schema as parsed from AST to avoid inconsistencies.
	// Prefer Step7's classification for source/join only.
	if s := tr.Context.String(); s != "" && s != "unknown" {
		switch s {
		case "cte", "subquery":
			t.Source = s
		case "join":
			if t.Source != "main" {
				t.Source = s
			}
		default:
			t.Source = s
		}
	}

	if !(t.Source == "main" && t.JoinType == "none") {
		t.JoinType = joinToString(tr.Join)
	}

	if tr.RealName != "" {
		if _, ok := cteTargets[tr.RealName]; ok {
			t.Source = "cte"
		} else if _, ok := subqueryTargets[tr.RealName]; ok && tr.Context == cmn.SQTableContextMain {
			// Alias of derived table should remain main for readability; leave as-is.
		}
	}

	return t
}

func collectCTENames(stmt cmn.StatementNode) map[string]struct{} {
	res := map[string]struct{}{}
	if stmt == nil {
		return res
	}

	if with := stmt.CTE(); with != nil {
		for _, def := range with.CTEs {
			if def.Name != "" {
				res[def.Name] = struct{}{}
			}
		}
	}

	return res
}

func kindToString(tp cmn.NodeType) string {
	switch tp {
	case cmn.SELECT_STATEMENT:
		return "select"
	case cmn.INSERT_INTO_STATEMENT:
		return "insert"
	case cmn.UPDATE_STATEMENT:
		return "update"
	case cmn.DELETE_FROM_STATEMENT:
		return "delete"
	default:
		return "select" // safe default; upstream parser only produces 4 kinds
	}
}

func extractTables(stmt cmn.StatementNode) []TableRef {
	switch s := stmt.(type) {
	case *cmn.SelectStatement:
		return extractFromClause(s.CTE(), s.From)
	case *cmn.InsertIntoStatement:
		// Target table as main; source tables if SELECT present will be in From
		out := make([]TableRef, 0, 1)
		if s.Into != nil {
			out = append(out, TableRef{
				Name:     s.Into.Table.Name,
				Schema:   s.Into.Table.SchemaName,
				Source:   "main",
				JoinType: "none",
			})
		}
		// If INSERT ... SELECT, include source tables
		if s.From != nil {
			out = append(out, extractFromClause(s.CTE(), s.From)...)
		}

		return out
	case *cmn.UpdateStatement:
		out := make([]TableRef, 0, 1)
		if s.Update != nil {
			out = append(out, TableRef{
				Name:     s.Update.Table.Name,
				Schema:   s.Update.Table.SchemaName,
				Source:   "main",
				JoinType: "none",
			})
		}

		return out
	case *cmn.DeleteFromStatement:
		out := make([]TableRef, 0, 1)
		if s.From != nil {
			out = append(out, TableRef{
				Name:     s.From.Table.Name,
				Schema:   s.From.Table.SchemaName,
				Source:   "main",
				JoinType: "none",
			})
		}

		return out
	default:
		return nil
	}
}

func extractFromClause(with *cmn.WithClause, from *cmn.FromClause) []TableRef {
	if from == nil {
		return nil
	}
	// Build CTE name set
	cte := map[string]struct{}{}

	if with != nil {
		for _, d := range with.CTEs {
			if d.Name != "" {
				cte[d.Name] = struct{}{}
			}
		}
	}

	tmp := make([]TableRef, 0, len(from.Tables))

	for _, t := range from.Tables {
		src := "join"
		jt := joinToString(t.JoinType)

		// classify cte/subquery
		if _, ok := cte[t.Name]; ok {
			src = "cte"
		} else if isSubqueryCandidate(t) {
			// skip subquery without FROM (no real table reference)
			if !hasFromToken(t) {
				continue
			}

			src = "subquery"
		}

		// Prefer original table name when available, falling back to Name (which may be alias)
		name := t.TableName
		// For schema-qualified tables, prefer the real table name from tokens
		if t.SchemaName != "" {
			if fn := fallbackTableName(t); fn != "" {
				name = fn
			}
		}

		if name == "" || name == "." {
			name = t.Name
		}

		if (name == "" || name == ".") && t.SchemaName == "" {
			if fn := fallbackTableName(t); fn != "" {
				name = fn
			}
		}
		// For subqueries, try to set name to underlying base table in the subquery (first table after FROM)
		if src == "subquery" {
			if bn := baseTableInSubquery(t); bn != "" {
				name = bn
			}
		}

		alias := aliasIfAny(t)
		if src == "subquery" && alias == "" {
			// for subquery, the identifier in FROM ... (subquery) alias is the alias
			alias = t.Name
		}

		tmp = append(tmp, TableRef{
			Name:     name,
			Alias:    alias,
			Schema:   t.SchemaName,
			Source:   src,
			JoinType: jt,
		})
	}
	// Reclassify first remaining entry as main
	if len(tmp) > 0 {
		if tmp[0].Source == "join" { // keep cte/subquery classification
			tmp[0].Source = "main"
			tmp[0].JoinType = "none"
		}
	}

	refs := tmp

	return refs
}

func isSubqueryCandidate(t cmn.TableReferenceForFrom) bool {
	if len(t.RawTokens) > 0 {
		return true
	}

	for _, tok := range t.Expression {
		if tok.Type == tokenizer.SELECT {
			return true
		}
	}

	return false
}

func hasFromToken(t cmn.TableReferenceForFrom) bool {
	for _, tok := range t.Expression {
		if tok.Type == tokenizer.FROM {
			return true
		}
	}

	return false
}

func fallbackTableName(t cmn.TableReferenceForFrom) string {
	var last string

	for _, tok := range t.Expression {
		switch tok.Type {
		case tokenizer.IDENTIFIER, tokenizer.RESERVED_IDENTIFIER:
			last = tok.Value
		}
	}

	if last == "." {
		return ""
	}

	return last
}

// baseTableInSubquery tries to extract the first base table name after FROM inside a subquery.
func baseTableInSubquery(t cmn.TableReferenceForFrom) string {
	seenFrom := false

	for _, tok := range t.Expression {
		if tok.Type == tokenizer.FROM {
			seenFrom = true
			continue
		}

		if !seenFrom {
			continue
		}

		if tok.Type == tokenizer.IDENTIFIER || tok.Type == tokenizer.RESERVED_IDENTIFIER {
			// Return the first identifier after FROM (ignore schema for Name; schema is not tracked here)
			return tok.Value
		}
	}

	return ""
}

func aliasIfAny(t cmn.TableReferenceForFrom) string {
	// TableReference.Name holds alias if present, otherwise original name.
	// We cannot easily distinguish here without additional flags; return empty when alias == original.
	if t.TableName != "" && t.Name != "" && t.TableName != t.Name {
		return t.Name
	}

	return ""
}

func joinToString(j cmn.JoinType) string {
	switch j {
	case cmn.JoinNone:
		return "none"
	case cmn.JoinInner:
		return "inner"
	case cmn.JoinLeft:
		return "left"
	case cmn.JoinRight:
		return "right"
	case cmn.JoinFull:
		return "full"
	case cmn.JoinCross:
		return "cross"
	case cmn.JoinNatural:
		return "natural"
	case cmn.JoinNaturalLeft:
		return "natural_left"
	case cmn.JoinNaturalRight:
		return "natural_right"
	case cmn.JoinNaturalFull:
		return "natural_full"
	default:
		return "unknown"
	}
}
