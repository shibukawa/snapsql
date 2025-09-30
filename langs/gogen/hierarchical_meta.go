package gogen

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/shibukawa/snapsql/intermediate"
)

var (
	ErrHierarchicalNoKeys = errors.New("hierarchical node has no key fields")
	ErrMultipleRootKeys   = errors.New("multiple root key fields detected; multi-root not supported")
)

// hierarchicalNodeMeta represents one hierarchical grouping unit (a path of segments like ["lists","cards"]).
// It intentionally does not contain concrete field types yet –その情報は既存 Response から直接取得できるため、
// ここではコード生成時に必要な最小限のメタだけを用意する。
type hierarchicalNodeMeta struct {
	Path       []string // e.g. ["lists","cards"]
	Depth      int      // len(Path)
	StructName string   // e.g. BoardTreeListsCards (生成名)
	ParentPath []string // nil for root level nodes (Depth==1)
	KeyFields  []string // Response 名（末尾フィールド名ではなく完全名）で PK(KeyLevel==Depth) と判定されたもの
	DataFields []string // Response 名で Key 以外 (Path prefix を共有するもの)
}

// buildHierarchicalNodeMetas analyzes responses and returns hierarchical metadata.
// 前提: Response.HierarchyKeyLevel が設定済み。
// 規則: a__b__c__col という列は Path=[a,b,c] / 最終要素 col を Data or Key 判定対象とする。
// Key 判定: Response.HierarchyKeyLevel == len(Path) かつ列名末尾が PK 列 (schema 情報に依存) の仮ルール。
// ここでは pipeline が既に level を割り当てている前提で単純に比較のみ行う。
func buildHierarchicalNodeMetas(functionName string, responses []intermediate.Response) ([]*hierarchicalNodeMeta, error) {
	// Group fields by path
	type agg struct {
		fields    []intermediate.Response
		keyFields []intermediate.Response
	}

	groups := map[string]*agg{}

	// Collect root key fields (no __) – 階層ルート識別用
	rootKeyFields := make([]intermediate.Response, 0)

	for _, r := range responses {
		if r.Name == "" {
			continue
		}

		if strings.Contains(r.Name, "__") {
			segs := strings.Split(r.Name, "__")
			if len(segs) < 2 { // safety
				continue
			}

			path := segs[:len(segs)-1]
			k := strings.Join(path, "__")

			g := groups[k]
			if g == nil {
				g = &agg{}
				groups[k] = g
			}

			g.fields = append(g.fields, r)

			depth := len(path)
			if r.HierarchyKeyLevel == depth+1 { // 子ノードのPKは path の1つ下位のレベル
				g.keyFields = append(g.keyFields, r)
			}
		} else {
			if r.HierarchyKeyLevel == 1 { // root PK 候補
				rootKeyFields = append(rootKeyFields, r)
			}
		}
	}

	// Build metas
	metas := make([]*hierarchicalNodeMeta, 0, len(groups))
	// Sort keys for deterministic output (depth asc then lexical)
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		ai := strings.Count(keys[i], "__") + 1

		aj := strings.Count(keys[j], "__") + 1
		if ai == aj {
			return keys[i] < keys[j]
		}

		return ai < aj
	})

	mainStructName := generateStructName(functionName)

	for _, k := range keys {
		segs := strings.Split(k, "__")
		g := groups[k]
		depth := len(segs)

		structSuffix := make([]string, len(segs))
		for i, s := range segs {
			structSuffix[i] = celNameToGoName(s)
		}

		structName := mainStructName + strings.Join(structSuffix, "")
		// parent path
		var parentPath []string
		if depth > 1 {
			parentPath = segs[:len(segs)-1]
		}

		meta := &hierarchicalNodeMeta{Path: segs, Depth: depth, StructName: structName, ParentPath: parentPath}
		for _, fr := range g.fields {
			// 子ノードのキーは階層の1段深い level（depth+1）に割り当てられている想定
			// 例: lists__id -> Path 深さ=1, KeyLevel=2
			if fr.HierarchyKeyLevel == depth+1 {
				meta.KeyFields = append(meta.KeyFields, fr.Name)
			} else {
				meta.DataFields = append(meta.DataFields, fr.Name)
			}
		}
		// 安全性: Key の無いノードは後段集約で重複排除できないためエラー返却（設計で skip も検討可）。
		if len(meta.KeyFields) == 0 {
			return nil, fmt.Errorf("%w (node=%s, depth=%d)", ErrHierarchicalNoKeys, k, depth)
		}

		metas = append(metas, meta)
	}

	// ルートノード複数サポートは今後検討。ここでは rootKeyFields が 0/1 以外の場合警告扱い (エラー返却)。
	if len(rootKeyFields) > 1 {
		return nil, fmt.Errorf("%w (count=%d)", ErrMultipleRootKeys, len(rootKeyFields))
	}

	return metas, nil
}
