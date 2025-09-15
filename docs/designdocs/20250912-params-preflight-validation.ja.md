# パラメータ事前検証（Preflight Validation）

日付: 2025-09-12

## 背景
- 現状、欠落パラメータがあると CEL 評価時の実行エラー（"no such attribute" 等）が出力され、どのパラメータが不足しているのか分かりづらい。
- .snap.sql / .snap.md のヘッダで必要パラメータは宣言されており、事前に判定可能。

## 目的
- 実行前（dry-run/実行の双方）に必要パラメータの不足を検出し、分かりやすいメッセージで失敗させる。
- 例: "Missing required parameters: pid (int), user_id (int)"。

## 仕様
- 対象: IntermediateFormat.Parameters（optional=falseのもの）。
- 除外: ImplicitParameters（TLS/コンテキストから供給されるため、CLI入力は不要）。
- 判定: `params` のトップレベルキーに、全ての必須パラメータ名が存在すること。
- エラー形式: `ErrMissingRequiredParam` をベースに、複数の場合はカンマ区切りで列挙。
- 表示: CLIはこのエラーをそのまま表示（既存のラップ方針に従う）。

## 実装方針
- query.ValidateParameters(format, params) を追加し、欠落名を収集してエラー化。
- 呼び出し箇所:
  - cmd/snapsql/command_query.go: executeDryRun() 冒頭
  - query/executor.go: ExecuteWithTemplate()（fast path/通常経路の双方に効く）

## テスト
- cmd/snapsql にユニットテストを追加。
  - 必須パラメータが不足しているとエラーとなり、名前と型が列挙されることを確認。

