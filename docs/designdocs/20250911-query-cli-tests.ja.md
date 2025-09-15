# `snapsql query` 受け入れテスト整備（設計メモ）

## 目的
- `snapsql query` の基本動作（テンプレートの読み込み・最適化・DRY-RUNのSQL生成とパラメータ評価・フォーマット検証・危険クエリ警告）の自動テストを追加し、CLIの信頼性を高める。

## スコープ
- DB接続不要（DRY-RUN中心）。
- 入力テンプレートは `.snap.sql` と `.snap.md` の両方を対象。
- パラメータは `--param` および `--params-file`（YAML/JSON）の最小確認は別途追加予定（今回は `--param` のみ）。
- 危険クエリ検出（DELETE/UPDATE without WHERE）のユニット確認。
- 無効な出力フォーマット時のエラー確認。

## テストケース
1. `.snap.sql` DRY-RUN
   - 入力: `SELECT * FROM users WHERE id = /*= user_id */1`
   - パラメータ: `user_id=1`
   - 期待: 生成SQLに `?` が含まれ、Args が `[1]` となる。

2. `.snap.md` DRY-RUN
   - Description/Parameters/SQL 最小ドキュメントを生成。
   - 期待: 1と同等のSQL/Args。

3. 危険クエリ検出
   - 入力: `DELETE FROM users`（WHEREなし）
   - 期待: `isDangerousQuery` が true。

4. 無効な出力フォーマット
   - `Format=invalid` で Run 実行（DRY-RUN）。
   - 期待: `ErrInvalidOutputFormat` を含むエラー。

## 実装方針
- `cmd/snapsql/command_query_test.go`（package main）で `QueryCmd` の内部関数（`buildSQLFromOptimized`, `isDangerousQuery`）を直接呼ぶ。
- テンポラリディレクトリでテスト用テンプレートを生成、`query.LoadIntermediateFormat` → `intermediate.OptimizeInstructions` → `buildSQLFromOptimized` の順に評価。
- `LoadConfig` はファイル非存在時にデフォルト構成で動作するため、特別な設定は不要。

