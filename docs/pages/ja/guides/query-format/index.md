# クエリーフォーマット

SnapSQL のクエリファイル（`.snap.md`）の書き方を説明します。

## 目次

### 基本構造

- [Markdownフォーマット](./markdown-format.md) - ファイル全体の構造
- [テンプレート構文](./template-syntax.md) - 条件分岐・ループの書き方
- [パラメータ](./parameters.md) - パラメータの定義と型
- [共通型](./common-types.md) - 再利用可能な型定義
- [レスポンス型](./response-types.md) - 戻り値の型定義
- [ネストしたレスポンス](./nested-response.md) - 階層化された結果の扱い

### テスト

- [テスト概要](./test-overview.md) - テストの基本
- [フィクスチャ](./fixtures.md) - テストデータの準備
- [Expected Results](./expected-results.md) - 結果の検証
- [アサーション](./assertions.md) - マッチャーの使い方
- [エラーテスト](./error-testing.md) - エラーケースの検証
- [特殊リテラル](./special-literals.md) - NULL、日付などの扱い

## クイックスタート

最小限のクエリファイル：

\`\`\`markdown
---
name: get_user
---

## Query

\```sql
SELECT id, name FROM users WHERE id = /*= user_id */
\```

## Parameters

\```yaml
user_id:
  type: integer
\```
\`\`\`

## 関連セクション

- [コマンドリファレンス](../command-reference/) - CLI コマンド
- [言語別リファレンス](../language-reference/) - 生成コードの使い方
