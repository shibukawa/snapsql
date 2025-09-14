# `query` 受け入れテスト: 外部パラメータと実行オプション対応（設計）

## 目的
- 既存の `query` パッケージ受け入れテストに、外部ファイルからのパラメータ注入と実行オプション指定の仕組みを追加し、CLI 相当の挙動（パラメータ・LIMIT/OFFSET・EXPLAIN/ANALYZE）を網羅的に検証できるようにする。

## スコープ
- 対象は `query` パッケージの受け入れテスト（SQLite 経路）
- データディレクトリ: `testdata/query/<case>/`
  - `input.snap.sql|input.snap.md`（必須）
  - `setup.sql`（任意）
  - `param.yaml|params.yaml`（任意・YAML のみ。CLI の `--params-file` 互換）
  - `options.yaml`（任意。実行時オプション指定）
  - `expected.yaml`（通常ケースは必須。EXPLAIN 指定時は不要）

## 仕様
- パラメータ
  - `param.yaml` または `params.yaml` を読み込み `map[string]any` としてテンプレート評価へ渡す。
  - `.snap.sql` の場合はヘッダコメントで `parameters:` を宣言しておく（型: 簡易型名）。
- 実行オプション（`options.yaml`）
  - `driver`: デフォルト `sqlite3`
  - `limit`: 整数。SELECT かつ SQL 内に LIMIT/OFFSET が無い場合に追記（後述のランタイム注入）。
  - `offset`: 整数。上に同じ。
  - `explain`: 真偽。計画のみ取得。
  - `explain_analyze`: 真偽。PostgreSQL のみ `EXPLAIN ANALYZE` を使用（SQLite は `EXPLAIN QUERY PLAN`）。
- 検証ルール
  - `explain|explain_analyze` が有効なケースは `expected.yaml` を参照せず、`ExplainPlan` 非空のみを検証。
  - それ以外は既存どおり `expected.yaml` と結果（Columns/Rows/Count）を比較。

## 実装方針
- 受け入れテストハーネス（`query/acceptance_test.go`）
  - `param.yaml|params.yaml`・`options.yaml` を読み込み、`ExecuteWithTemplate` に渡す。
  - EXPLAIN 指定時は YAML 比較をスキップし、`ExplainPlan` 非空を検証。
- ランタイム注入（LIMIT/OFFSET）
  - `query.Executor.Execute()` で生成 SQL に対し、既に LIMIT/OFFSET が含まれていない SELECT の場合に限り `options.Limit/Offset` を追記（セミコロン考慮）。
  - `ExecuteWithTemplate()` の静的高速経路でも同様に追記して整合を取る。
- EXPLAIN 実行
  - `executeSQL()` 内で `options.Explain` を解釈し、
    - SQLite: `EXPLAIN QUERY PLAN <sql>` を実行して全列を文字列連結して `ExplainPlan` に格納。
    - PostgreSQL: `EXPLAIN [ANALYZE] <sql>` を実行。
  - フォーマッタ側は既存の `FormatExplain` を利用。
- CEL 評価（ADD_PARAM）
  - 直接参照（例: `/*= user_id */`）はまず `params["user_id"]` を直接解決。
  - 足りない場合に限り、`params` マップ＋個別変数を CEL 環境へ宣言して評価。

## 追加テスト
- `011_select_with_params`
  - `param.yaml` から `user_id` を注入し、`WHERE id = /*= user_id */` を解決。
- `012_select_limit_offset_options`
  - SQL に LIMIT/OFFSET を書かず、`options.yaml`（limit/offset）でページング結果を検証。
- `013_select_explain_option`
  - `options.yaml`（explain: true）で計画文字列の非空を検証（SQLite）。

## 非対象 / 留意点
- ダイナミック命令（IF/FOR 系）やシステムフィールドの完全対応は別スコープ。
- SQLite での EXPLAIN は方言差が大きいため、内容の厳密比較は行わない（非空のみ）。

## 移行・互換性
- 既存ケースはそのまま動作。オプションやパラメータを追加しても後方互換性は保持。

