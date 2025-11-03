# gogen / query 統合コード生成設計 (IR=intermediate 活用)

日付: 2025-09-13
作者: AI支援 (要レビュー)
ステータス: Draft
対象バージョン: 未リリース (後方互換性制約なし)

## 1. 背景
`langs/gogen` は静的な関数IF/実装生成を行い、`query` サブコマンドは `.snap.sql` から実行用ダイナミックコード/SQL整形・検証を行う。両者で以下の重複/ばらつきがある:
- プレースホルダ・方言変換ロジック
- パラメータ型/戻り列型の対応 (CEL 型・DB 型・Go 型)
- エラー分類/フェーズ区分
- 生成テンプレート (スキャン, Exec, 型補助)

既存 `intermediate` パッケージは事実上の IR (中間表現) に近く、Instruction / Processor / Optimizer 等のパイプラインを持つ。これを“唯一の IR” と定義し、gogen/query の両方は `intermediate` から得られる統一アダプタ層を介してコード生成 or 実行処理を行うように再編する。

## 2. 目的
1. intermediate を正式に IR と位置づけ、外部公開は避けつつ安定 API を内部合意。
2. gogen と query が共有する **Adapter 層 (Mapper)** を導入し、IR→(コード生成用構造 / 実行用構造) へ変換。
3. 方言 (Dialect) 差異吸収を一元化: プレースホルダ, LIMIT/OFFSET, RETURNING, JSON/UUID/Decimal 型, boolean 変換。
4. テスト容易化: IR から生成物のスナップショット / 差分テスト。gogen/query 両面で同じ Golden を再利用。
5. 将来 TypeScript / OpenAPI / GraphQL Schema など追加出力先へ拡張可能な形にする。

## 3. 非目的
- ランタイム接続管理やトランザクション抽象の再設計。
- 完全な SQL パーサ再実装 (現状 parser + intermediate 利用)。
- ORM 化。

## 4. 要求とスコープ
| 項目 | 要件 |
|------|------|
| IR 入力 | `.snap.sql`, markdown フォーマット, 単一SQL, 将来複数ステートメント(検討) |
| IR 出力 | Function (パラメータ, 戻り列, placeholder 配置, 方言特徴) |
| 方言統合 | Postgres / MySQL / SQLite (初期) + 拡張容易な Interface |
| 型変換 | LogicalType -> { GoType, ScanType, DBNativeType, CELType } |
| エラー | Parse / Analyze / Dialect / Generate の段階分類 |
| 生成対象 | (a) 静的: gogen, (b) 実行: query runtime helpers |

## 5. 既存資産再利用
- `intermediate/intermediate_format.go` 等: Node/Instruction 構造を **IR 起点** とする。
- `intermediate/optimizer.go`: 定数畳み込み/不要節省略は IR 前処理。
- `langs/gogen` のテンプレ群: 分離し `internal/templates/go` へ移設予定 (差分最小)。

## 6. 新規コンポーネント
### 6.1 Adapter 層
```
internal/codegen/adapter/
  ir_adapter.go   // intermediate の構造体を CodegenFunction へ map
  types.go        // CodegenFunction, CodegenParam, CodegenResult など
```
`CodegenFunction` はテンプレに渡す最小限構造:
```go
type CodegenFunction struct {
  Name       string
  Exported   bool
  Dialect    string
  Params     []CodegenParam
  Results    []CodegenResult
  SQL        SQLRenderMeta
  Features   map[string]bool
}
```

### 6.2 Dialect 統合 (更新方針)
Dialect 吸収は **intermediate パイプライン内の専用 Processor ステップ** として実行し、IR 生成完了直後に:
1. 方言条件付きトークン/ノードを解決し静的命令列へ畳み込み
2. RETURNING / LIMIT / OFFSET / placeholder を **トークン/ノードレベル** で安全に書き換え
3. 方言非対応機能は命令削除ではなく Feature フラグ + 警告 (将来) による顕在化を検討

これにより文字列後処理 (脆弱なヒューリスティック) を排除し SQL 構造情報を保持した段階で調整可能。

最終的に Adapter が受け取る `IntermediateFormat` には方言切替用の命令は存在せず、純粋な静的/ループ/評価命令のみとなる。

簡易インタフェース (Executor / Generator 共有) は以下の通り:
```go
type Dialect interface {
  Name() string
  PlaceholderStyle() PlaceholderStyle // ? / $n / :vN
  SupportsReturningUpdate() bool
  SupportsReturningDelete() bool
}
```
`MapLogicalType` のような型変換は Dialect ではなく types 層で一元化 (Dialect 依存が必要な特殊型のみ追加検討)。

### 6.3 型マッピング
`typeinference` / `langs/gogen` に散在するロジックを `internal/codegen/types` へ集約。
```go
type GoTypeInfo struct {
  GoExpr      string // "int", "string", "sql.NullInt64", カスタム型など
  ScanTarget  string // 直接スキャン用変数型
  Nullable    bool
  Imports     []string
}
```

### 6.4 テンプレエンジン
- `template.FuncMap` に: `camel`, `snake`, `placeholders`, `joinImports`。
- gogen/query 共通: 実行関数 / スキャン関数 / エラーヘルパ。

## 7. フロー統合
```
Parse (.snap.sql) → intermediate pipeline → IR (intermediate) → Adapter(Map) → CodegenFunction(s)
   → (A) gogen: 静的ソース出力
   → (B) query: 実行時 DSL / Prepared 構築
```

## 8. 方言吸収詳細 (更新)
| 差異 | 処理層 (新) | アプローチ | 安全性向上要点 |
|------|--------------|------------|------------------|
| Placeholder | Intermediate Pipeline (DialectProcessor) | トークン列再構築時にインデックス採番 | クォート/コメント境界を tokenizer 情報で判別 |
| LIMIT/OFFSET | 同上 | ノード種別ごとに正規化 (例: `LIMIT ALL` 削除) | SQL 部品化された構造利用 |
| RETURNING | 同上 | 非対応方言: ノードごと除去 (INSERT は常に保持) | 文字列探索排除 / 種別限定 |
| Bool 表現 | Type Mapping 層 | logical->GoType | 方言差異は最小限 (tinyint(1)) |
| Decimal | Type Mapping 層 | カスタム型 | 精度/スケールは meta へ保持予定 |
| JSON/UUID | Type Mapping 層 | GoTypeInfo へ集約 | Dialect 分岐不要なら共通 |

旧案の外部 `dialect.Normalize` (文字列ベース後処理) は削除済み。方言処理は intermediate パイプライン内 Processor 実装に統一。

## 9. エラーモデル
```go
type Phase string
const (
  PhaseParse Phase = "parse"
  PhaseAnalyze = "analyze"
  PhaseDialect = "dialect"
  PhaseGenerate = "generate"
)

type CodegenError struct {
  Phase Phase
  Msg   string
  Span  *Span // 位置情報(任意)
}
```

## 10. テスト戦略
| 種別 | 内容 |
|------|------|
| IR Golden | intermediate 出力を JSON 化しスナップショット |
| Dialect | Placeholder 変換テーブル駆動テスト |
| Types | Logical → GoTypeInfo の単体テスト |
| Generator | CodegenFunction → 生成 Go の golden |
| Integration | .snap.sql → 実行 (query) ↔ 生成 (gogen) の差異なし |

## 11. 移行ステップ
1. (S1) Adapter 型定義 (空実装) 追加
2. (S2) gogen 内部で intermediate を直接読まないよう Adapter 経由に変更
3. (S3) query も同 Adapter 利用に変更（現行ロジック囲い込み）
4. (S4) Dialect interface 抽出と既存置換ロジック移設
5. (S5) 型マッピング集約 + 既存参照差し替え
6. (S6) テンプレ統合 (重複削除) & Golden テスト導入
7. (S7) エラーモデル導入・CLI 表示改善
8. (S8) 将来拡張 (TS/OpenAPI) 用プレースホルダ追加 (未実装でコメント)

## 12. リスク/緩和
| リスク | 緩和 |
|--------|------|
| Adapter 層が過剰抽象化 | 最初は最小フィールドのみ / 追加は PR ベース |
| Dialect 増加で分岐複雑化 | Feature flag + 個別小関数化 |
| 生成差分が大量発生 | 先行で Golden テスト基盤構築 |
| Imports 競合 | GoTypeInfo に Imports 集約 + 重複排除ヘルパ |

## 13. オープン課題
- Result 列の型推論をどこまで自動化するか (現在: 手動/限定的)。
- 複数ステートメント (; 区切り) の扱い。
- エラーメッセージ国際化 (当面不要)。
- Decimal / Null 系の統一ポリシー (pointer vs sql.Null*)。

## 14. 今後の拡張アイデア
- Prepared Statement キャッシュ生成 (hash key)
- Tracing Hook インジェクション (Before/After Exec)
- Validation Only モード (生成せず IR 出力)

## 15. 成功指標 (KPI)
- 重複コード (placeholder/型マッピング) 30% 削減
- 方言追加工数: Dialect ファイル + テストのみ (他変更ゼロ)
- `.snap.sql` 追加から生成/実行までの失敗率低下

---
Draft 完。レビューコメント歓迎。修正点があれば指示ください。

## 16. Dialect 適用方針最終決定 (改訂): パイプライン内吸収 + IntermediateFormat 再利用

### 16.1 方針変更の背景
当初は Dialect 適用後に独自 `NormalizedIR` 構造を導入する案だったが、以下理由で **既存 `intermediate.IntermediateFormat` をそのまま再利用** する方針に統一:
- 命令列(Instructions) を保持することで FOR ループ / 可変 VALUES / 条件分岐を後段実行経路で再解釈可能。
- Dialect 適用でフィールド追加・削除が現状想定されず、構造の冗長コピーは不要。
- Downstream (Adapter, gogen, query runtime) は型マッピングやプレースホルダ番号解決を別層で行えるため、IntermediateFormat のままで十分。

### 16.2 変更点概要
| 項目 | 旧 | 新 |
|------|----|----|
| 適用タイミング | IR 生成後 外部関数 `Normalize` | IR パイプライン内 Processor (GenerateFromSQL/Markdown 内部) |
| 方言切替命令 | 生成され保持 | 生成段階で解決し非生成 |
| RETURNING 削除 | 文字列ヒューリスティック | ノード種別検出で除去 (UPDATE/DELETE のみ) |
| Placeholder 変換 | 文字列スキャン ('?') | トークン列再構築で確実変換 |
| Idempotent 制御 | 呼び出し側注意 | Pipeline ステップ一回のみ |

### 16.3 パイプライン更新図
```
Parse → (Optimizer) → (DialectProcessor) → IntermediateFormat(方言適用済) → Adapter → Codegen / Runtime
```

### 16.4 DialectProcessor 擬似インタフェース
```go
type DialectProcessor struct { Dialect Dialect }
func (p *DialectProcessor) Process(f *IntermediateFormat) error {
  // 1. 方言条件付きトークン解決 (パーサ段階でマークされている前提)
  // 2. RETURNING ノード Remove (UPDATE/DELETE 非対応時)
  // 3. Placeholder スキャン: tokenizer の種別情報で境界認識
  // 4. LIMIT/OFFSET 正規化
  return nil
}
```

### 16.5 方言命令の整理
| 理由 | 詳細 |
|------|------|
| 責務分散回避 | 後段 (adapter/templates) が命令解釈を気にする必要を無くす |
| 安全性 | 文字列操作より構造情報を利用した条件分岐が正確 |
| テスト簡素化 | 方言適用後の IR 差分で確認可能 |

### 16.6 マイグレーション計画 (追加)
1. 既存 `dialect/normalize.go` を Deprecated コメント付与
2. 新 `DialectProcessor` を `intermediate/pipeline.go` へ組み込み
3. Parser で dialect 条件付き構文があればフラグ付トークン化 (なければスキップ)
4. 既存テスト: `Normalize` 依存部を Processor 実行にリライト
5. 方言切替用命令関連定義/テストの見直し
6. 旧ファイル削除 (2～5 安定後)

### 16.7 リスク / 緩和 (更新)
| リスク | 緩和 |
|--------|------|
| Parser で方言条件構文をどう表現するか未確定 | 初期版は簡易コメントディレクティブ `--@dialect(pg,mysql): <fragment>` のみ対応検討 |
| Processor 実装が肥大 | サブ関数分割: placeholders.go / returning.go / limitoffset.go |
| 既存テスト壊れ | 並行で golden 再生成スクリプト整備 |

### 16.8 まとめ (改訂)
外部 `Normalize` 関数を廃止し、構造レベルで安全な Dialect 吸収をパイプライン内部に移行。方言切替に関する処理はパイプライン内で解決され、IR は方言適用済みの最終形を一貫して提供する。

### 16.3 パイプライン (最終)
```
Parse → IntermediateFormat → (Pipeline内Dialect処理) → Adapter → Codegen / Runtime
```

### 16.4 利点 (再評価)
| 項目 | 効果 |
|------|------|
| 構造複製削減 | アロケーション/複雑度低減 |
| メンテ容易性 | 単一 IR を追えば全フェーズを観察可能 |
| 実行時特性保持 | ループ/条件を失わず後段最適化 or 実行が可能 |
| テスト単純化 | 既存 intermediate JSON Golden をそのまま利用 |

### 16.5 リスク / 緩和
| リスク | 内容 | 緩和 |
|--------|------|------|
| Dialect 適用後と前を区別しづらい | 状態遷移が曖昧 | `FormatVersion` かヘッダコメントに dialect 適用済フラグを付与検討 (必要時のみ) |
| in-place 変換が副作用になる | 呼び出し側が複数回適用するバグ | idempotent チェックフラグを内部に追加予定 |
| 拡張時に追加メタが欲しくなる | プレースホルダマップ等 | `CacheKeys` 同様の拡張スライスを後日追加可 |

### 16.6 実装ステップ更新
S9: no-op Normalize 追加 (完了)  
S10: Op書き換え (EmitIfDialect → Static/除去) + プレースホルダ変換骨組み  
S10+: RETURNING, LIMIT/OFFSET, idempotent フラグ導入 (必要なら)  
Adapter 着手タイミングは S10 の最初の書き換え完了後でも可。

### 16.7 不採用案
- 専用 NormalizedIR: メリット (責務明確化) < デメリット (重複/同期コスト)。
- Adapter 内部で都度 Dialect 適用: 実行と生成で二重適用リスク。

### 16.8 まとめ
最小変更で方言差異吸収を実現するため IntermediateFormat の再利用を採用。Normalize は拡張可能な in-place 処理ポイントとし、後段層 (Adapter, Types, Templates) の責務境界を維持する。
