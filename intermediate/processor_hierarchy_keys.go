package intermediate

import (
	"strings"

	"github.com/shibukawa/snapsql"
)

// HierarchyKeyLevelProcessor:
// 現段階では SELECT 列の構文木から直接 Response を構築していないため、
// pipeline 最終直前に name の規則だけで簡易レベル付与を行う。
// 仕様: a__b__c 形式の場合 segment 数が 3 なら最終要素 c を level=3 とみなし、
// 先頭 segment a に対応する root カラムは level=1 として扱えるよう、
// 実際には各 Response 自体（全カラム）に depth = len(segments) を暫定セットする。
// ただし root (no __) は depth=1。
// 将来: schema + PK 情報で PK でない列は 0 に再設定するフェーズを追加予定。

type HierarchyKeyLevelProcessor struct{}

func (p *HierarchyKeyLevelProcessor) Name() string { return "HierarchyKeyLevelProcessor" }

func (p *HierarchyKeyLevelProcessor) Process(ctx *ProcessingContext) error {
	// この段階では Responses がまだ ctx には無い（Execute 内で determineResponseType 使用）。
	// そこで ctx へ暫定のフィールドを保持し、Execute 側で最終 IntermediateFormat の Responses に反映する。
	// 簡易実装: Statement からは再解析せず、後工程(Execute)で再計算するヘルパーを用意。
	return nil
}

// applyHierarchyKeyLevels applies the level calculation to a slice of Response (called from Execute)
func applyHierarchyKeyLevels(responses []Response, tableInfo map[string]*snapsql.TableInfo) []Response {
	if len(responses) == 0 {
		return responses
	}

	// Build quick lookup: table -> set(primary key columns)
	// NOTE: SourceTable/SourceColumn are internal-only (json:"-") so最終出力には含めない。
	// ここでのみ利用し HierarchyKeyLevel を確定させる。
	pkMap := map[string]map[string]struct{}{}

	for _, t := range tableInfo {
		for _, c := range t.Columns {
			if c.IsPrimaryKey {
				if pkMap[t.Name] == nil {
					pkMap[t.Name] = map[string]struct{}{}
				}

				pkMap[t.Name][strings.ToLower(c.Name)] = struct{}{}
			}
		}
	}

	// Pass: assign depth & PK flags using (SourceTable, SourceColumn) exact pairing.
	for i := range responses {
		name := responses[i].Name
		if name == "" {
			continue
		}

		var depth int
		if strings.Contains(name, "__") {
			depth = len(strings.Split(name, "__"))
		} else {
			depth = 1
		}

		responses[i].HierarchyKeyLevel = 0 // default non-key

		tbl := responses[i].SourceTable

		col := strings.ToLower(responses[i].SourceColumn)
		if tbl != "" && col != "" {
			if cols, ok := pkMap[tbl]; ok {
				if _, pk := cols[col]; pk {
					responses[i].HierarchyKeyLevel = depth
				}
			}
		}
	}

	// Guarantee at least one root key (depth=1). If none, fallback: first root with SourceTable PK; else first root column.
	hasRoot := false

	for _, r := range responses {
		if r.HierarchyKeyLevel == 1 {
			hasRoot = true
			break
		}
	}

	if !hasRoot {
		idx := -1

		for i, r := range responses {
			if strings.Contains(r.Name, "__") {
				continue
			}

			if r.SourceTable != "" && r.SourceColumn != "" {
				if cols, ok := pkMap[r.SourceTable]; ok {
					if _, pk := cols[strings.ToLower(r.SourceColumn)]; pk {
						idx = i
						break
					}
				}
			}
		}

		if idx == -1 { // fallback to first root
			for i, r := range responses {
				if !strings.Contains(r.Name, "__") {
					idx = i
					break
				}
			}
		}

		if idx >= 0 {
			responses[idx].HierarchyKeyLevel = 1
		}
	}

	return responses
}
