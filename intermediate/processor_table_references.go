package intermediate

import (
	"sort"
	"strings"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// TableReferencesProcessor extracts table reference information from the statement
type TableReferencesProcessor struct{}

func (p *TableReferencesProcessor) Name() string {
	return "TableReferencesProcessor"
}

func (p *TableReferencesProcessor) Process(ctx *ProcessingContext) error {
	ctx.TableReferences = convertTableReferences(ctx.Statement, ctx.TableInfo)
	return nil
}

func convertTableReferences(stmt parser.StatementNode, tableInfo map[string]*snapsql.TableInfo) []TableReferenceInfo {
	if stmt == nil {
		return nil
	}

	tableMap := stmt.GetTableReferences()
	if len(tableMap) == 0 {
		return nil
	}

	keys := make([]string, 0, len(tableMap))
	for key := range tableMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	refs := make([]TableReferenceInfo, 0, len(keys))
	for _, key := range keys {
		ref := tableMap[key]
		if ref == nil {
			continue
		}

		refs = append(refs, convertSQTableReference(ref, tableInfo))
	}

	return refs
}

func convertSQTableReference(ref *cmn.SQTableReference, tableInfo map[string]*snapsql.TableInfo) TableReferenceInfo {
	alias := strings.TrimSpace(ref.Name)
	realName := strings.TrimSpace(ref.RealName)

	info := TableReferenceInfo{
		QueryName: strings.TrimSpace(ref.QueryName),
		Context:   mapTableContext(ref.Context),
	}

	physicalName := resolvePhysicalTableName(ref, tableInfo)
	if physicalName != "" {
		info.TableName = physicalName
	}

	if realName != "" {
		info.Name = realName
	} else if physicalName != "" {
		info.Name = physicalName
	} else if alias != "" {
		info.Name = alias
	}

	if alias != "" && alias != info.Name {
		info.Alias = alias
	}

	if info.Name == "" {
		info.Name = alias
	}

	return info
}

func resolvePhysicalTableName(ref *cmn.SQTableReference, tableInfo map[string]*snapsql.TableInfo) string {
	if len(tableInfo) == 0 {
		return ""
	}

	realName := strings.TrimSpace(ref.RealName)
	if realName == "" {
		return ""
	}

	candidates := []string{}
	if ref.Schema != "" {
		candidates = append(candidates, ref.Schema+"."+realName)
	}

	candidates = append(candidates, realName)

	for _, candidate := range candidates {
		if tbl := lookupTableInfo(tableInfo, candidate); tbl != nil {
			if tbl.Schema != "" && !strings.Contains(candidate, ".") {
				return tbl.Schema + "." + tbl.Name
			}

			return tbl.Name
		}
	}

	return ""
}

func mapTableContext(context cmn.SQTableContextKind) string {
	switch context {
	case cmn.SQTableContextMain:
		return "main"
	case cmn.SQTableContextJoin:
		return "join"
	case cmn.SQTableContextCTE:
		return "cte"
	case cmn.SQTableContextSubquery:
		return "subquery"
	default:
		return ""
	}
}
