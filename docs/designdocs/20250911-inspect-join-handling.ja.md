# Inspectモードにおける NATURAL/CROSS JOIN 取り扱い

## 背景
- 既存実装では SnapSQL の生成系前提で NATURAL/CROSS JOIN を原則エラーとして扱っていた。
- `inspect` サブコマンドではコード生成は行わず、解析結果（参照テーブル/結合種別など）の提示が目的。
- よって InspectMode では NATURAL/CROSS JOIN をエラーにせず情報として取得・出力したい。

関連: `docs/designdocs/20250911-inspect-command.ja.md`（InspectModeの緩和ポリシー）

## 目的
- InspectMode 時に NATURAL/CROSS JOIN を許容し、結合種別として `"natural"` / `"cross"` を返す。
- 既存の厳格モード（生成系）では従来どおりエラーのままにする（互換維持）。

## スコープ
- 解析・出力に限定（生成系の解釈・最適化は変更しない）。
- `parserstep4` の FROM 句終端処理（finalize）と JOIN 解析を中心に改修。
- `inspect` パッケージの join 文字列化ロジックに NATURAL を追加。

## 仕様
- InspectMode = false（既定）
  - これまで通り：NATURAL は `ErrInvalidForSnapSQL`、CROSS は ON/USING を伴うとエラー。
- InspectMode = true
  - NATURAL: エラーにせず `JoinNatural` として保持。条件（ON/USING）は禁止（存在したら警告扱いにするか未出力）。
  - CROSS: 既存の許容を維持（ON/USING 禁止は継続）。
  - 出力: `inspect` の `TableRef.JoinType` に `"natural"`/`"cross"` を設定。

### 出力仕様（JSON / CSV）

1) JSON（InspectResult）

- フィールド
  - `statement`: `"select" | "insert" | "update" | "delete"`
  - `tables`: `TableRef[]`
    - `name`: 文字列（テーブル名。サブクエリ時は推定基底テーブル名を優先）
    - `alias`: 文字列（省略可。存在する場合のみ出力）
    - `schema`: 文字列（省略可）
    - `source`: `"main" | "join" | "cte" | "subquery"`
    - `join_type`: `"none" | "inner" | "left" | "right" | "full" | "cross" | "natural" | "natural_left" | "natural_right" | "natural_full" | "unknown"`
- 備考
  - `notes` は現在使用しない（将来互換のため構造に残すが `omitempty`）。

例:

```json
{
  "statement": "select",
  "tables": [
    { "name": "users",  "alias": "u", "source": "main", "join_type": "none" },
    { "name": "orders", "alias": "o", "source": "join", "join_type": "natural_left" }
  ]
}
```

2) CSV（テーブル一覧）

- ヘッダ固定: `name,alias,schema,source,joinType`
- 各行: JSONの `tables[]` を1行に対応付け
- 値のマッピング
  - `joinType` は `join_type` の値をそのままスネークケース→ローワーのまま出力（例: `natural_right`）
  - 未設定は空文字

例:

```
name,alias,schema,source,joinType
users,u,,main,none
orders,o,,join,natural_left
```

## 変更方針（実装）
1) 型拡張（小）
   - `parser/parsercommon/elements.go`
     - `JoinType` に `JoinNatural` を追加。
     - `String()` に `"NATURAL JOIN"` を追加。

2) JOIN 解析の緩和
   - `parser/parserstep4/from_clause.go`
     - `finalizeFromClause` → `parseTableReference` → `parseJoin` に `inspectMode` を伝播。
     - `parseJoin` 内で NATURAL を検出した際、`inspectMode=true` の場合は `JoinNatural` を返し、エラーにしない。
     - NATURAL と ON/USING の併用は引き続き禁止（検出時の扱いは InspectMode でもエラー or notes）。今回はエラーのまま（シンプル）。

3) 文字列化の拡張
   - `inspect/inspector.go` の `joinToString` に `JoinNatural` → `"natural"` を追加。

4) 公開再エクスポートの整合
   - `parser/parse.go` の再エクスポートで `JoinNatural` を追加。

5) テスト
   - `parserstep4`：
     - 既存の NATURAL 関連異常系は厳格モードのまま維持。
     - 追加: InspectMode で `SELECT * FROM a NATURAL JOIN b` がエラーにならず、`JoinNatural` になること。
   - `inspect`：
     - NATURAL/CROSS を含むSQLで `Inspect` が `joinType: "natural"/"cross"` を出力すること。

## 非対象
- NATURAL の各バリエーション（NATURAL LEFT/RIGHT/FULL/INNER）の精緻化。現時点では `JoinNatural` の単一種別で表現する。
- ON/USING を伴う NATURAL の警告メッセージ整備は将来検討（必要になれば notes に積む）。

## 後方互換性
- 既定（InspectMode=false）では従来と同じエラー動作。生成系に影響を与えない。

## 想定影響
- `JoinType` への列挙追加に伴う `switch` 追加（影響範囲は限定的）。
