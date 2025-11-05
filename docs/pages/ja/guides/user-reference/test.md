````markdown
# テスト: 利用者向けドキュメント

このページは SnapSQL を使ってクエリテストを記述・実行する利用者向けの入門ガイドです。
ここでは主に「テストケースの書式」「テスト実行のライフサイクル」「実行コマンド」「ベストプラクティス」を説明します。フォーマットの細かい仕様（Fixtures 内の特殊リテラル等）は `docs/pages/ja/guides/query-format/` 以下の各ページを参照してください。

## テストケースの基本構造

テストは Markdown ドキュメントの `## Test Cases` セクション内に記述します。各テストケースは `###`（H3 見出し）で区切ります。

- サブセクションは見出しではなく太字のラベル（例: `**Fixtures:**`）で記述します。
- 主に使うサブセクション:
  - `**Fixtures:**` — テスト用データ（YAML / CSV / 外部ファイル参照）
  - `**Parameters:**` — クエリに渡すパラメータ（テストごとに 1 セットを想定）
  - `**Expected Results:**` — 正常系の期待値（SELECT/RETURNING の結果、またはテーブル状態の検証）
  - `**Expected Error:**` — エラーケースで期待するエラー種別
  - `**Verify Query:**` — 任意の追加検証クエリ（副作用の検証など）

### 最小構成の例

````markdown
## Test Cases

### Test: Basic user retrieval

**Fixtures:**
```yaml
users:
  - id: 1
    name: "太郎"
```

**Parameters:**
```yaml
user_id: 1
```

**Expected Results:**
```yaml
- id: 1
  name: "太郎"
```
````

## テストのライフサイクル（実行フロー）

一般的なテスト実行フローは次の通りです（実装の挙動に沿った説明）:

1. フィクスチャのロード（`clear-insert` / `upsert` / `delete` / `transaction-wrapped` のいずれか）
2. パラメータの解決とテンプレートの適用
3. クエリ実行（SELECT / INSERT / UPDATE / DELETE 等）
4. 結果検証（`Expected Results` または `Expected Error`）
5. 必要に応じて `Verify Query` を実行して副作用を検証

このライフサイクルはローカル実行でも CI 実行でも共通です。並列実行する際はワーカごとに分離されたプレフィックス付きテーブルや独立した DB インスタンスを使う運用が推奨されます。

## テストの実行（コマンド）

基本的なコマンド例（CLI 実装に依存します）:

```bash
snapsql test path/to/query.snap.md
snapsql test path/to/queries/
# ケース名でフィルタ
snapsql test path/to/query.snap.md --run-pattern="UserGet"
```

より詳細なオプション（タイムアウト、並列数、fixture-only 等）は `snapsql --help` を参照してください。

## ベストプラクティス（利用者視点）

- テスト名を具体的に: 何を検証するかが明確になる名前にする（例: `正常系: 年齢範囲でユーザーを検索`）。
- フィクスチャは最小限に: テスト対象に不要なデータは含めない。
- 可変値はマッチャーで: 時刻やトークン等はマッチャーで許容幅やパターンを使う。
- エッジケースを用意する: 境界値、NULL、空配列など。
- エラーケースを明記する: 期待するエラーは `**Expected Error:**` で明示しておく。

## どの情報をどこに書くか（目安）

- フォーマット仕様（特殊リテラル、YAML/CSV の細かい扱い等）は `docs/pages/ja/guides/query-format/` 以下に置きます。
- 利用者が知るべきワークフロー（書き方、実行方法、ベストプラクティス）はこの `user-reference/test.md` にまとめてください。

---

このページに記載した運用や例は利用者視点での案内です。より細かいフォーマット仕様や実装依存の挙動は `query-format` 側のページを参照してください。
````
