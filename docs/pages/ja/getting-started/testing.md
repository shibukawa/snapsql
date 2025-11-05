# Testing

SQLクエリのテスト実行について説明します。

## テストの重要性

SnapSQLのテストにより：

- SQLクエリの正確性を保証
- リファクタリング時の回帰を防ぐ
- ドキュメントとしての役割を果たす

## クエリファイル内の Test Cases の書き方

クエリファイル（`.snap.md` / `.snap.sql`）の中では、`## Test Cases` セクション内に複数のテストケースを並べて記述できます。各テストケースは次の 3 つの節がセットになっているのが基本です：

- **Fixtures:** テスト用データ（YAML）
- **Parameters:** テンプレートに渡すパラメータ（YAML）
- **Expected Results:** 期待されるクエリ結果（YAML）

ファイル例は `examples/kanban/queries` に多数あります。以下はそのうちの一つを抜粋した簡潔なサンプルです（テストケースは複数持てます）：

````markdown
## Test Cases

### Board tree includes active lists and their cards

**Fixtures:**
```yaml
boards:
  - id: 5
    name: "Release Plan"
lists:
  - id: 50
    board_id: 5
    name: "Todo"
cards:
  - id: 500
    list_id: 50
    title: "Set up project"
```

**Parameters:**
```yaml
board_id: 5
```

**Expected Results:**
```yaml
- id: 5
  name: "Release Plan"
  lists__id: 50
  lists__cards__id: 500
```
````

テストデータ（フィクスチャ）はYAML形式で記述します。dbunit互換のXMLやCSVもサポートしていますが、主要実装はYAMLです。デフォルトのロード戦略は `clear-insert`（テスト前にテーブルを空にしてフィクスチャを挿入）です。他の戦略（upsert や delete など）の詳細や共有方法については `guides` 以下のドキュメントを参照してください。

### Expected Results のマッチャー

`Expected Results` では、厳密なリテラル値の代わりにマッチャーを指定して柔軟に検証できます。主要なマッチャー例：

- `[notnull]` : 値が NULL でないことを検証します。
- `[any]` : 任意の値を許容（存在することのみ検証）します。
- `[currentdate]` : 実行時の現在日付に近い値を許容します（タイムスタンプ比較に便利）。
- `[currentdate, 10d]` : 現在日付からの差が指定日数以内であることを検証します（例: `10d` は 10 日以内）。
- `[regexp, 正規表現]` : 与えた正規表現にマッチすることを検証します（例: `[regexp, ^user@.+\\.com$]`）。

これらのマッチャーを使うことで、タイムスタンプや自動生成値、外部システム由来の変動しやすいフィールドの検証が簡単になります。

行を返さない UPDATE / DELETE といったクエリのアサーション（影響行数の検証や副作用の確認）については、運用パターンが多岐に渡るため `guides` 以下の該当ページで詳細を説明しています。そちらを参照してください。

## テスト実行の基本

クエリのテストを実行するには、サンプルファイルを指定できます。詳細なサンプルは `samples/project_tasks_example.md` に移動しました。

```bash
snapsql test queries/project_tasks.md
```

サンプルファイル（SQLとフィクスチャ）の内容を確認するには：

[サンプルを見る →](/ja/samples/project_tasks_example.md)

## テスト結果の確認

テストが成功すると：

```
✅ Test passed: Get tasks for existing project
```

テストが失敗すると：

```
❌ Test failed: Get tasks for existing project
Expected: [...]
Actual: [...]
```

### 特定のテストケースのみ実行

テストはカレントフォルダ以下を探索して見つけたファイルを全て実行しますが、ディレクトリやファイルを指定することで、特定のテストのみを実行できます。

```bash
snapsql test "Get tasks for existing project" queries/project_tasks.md
```

### 詳細出力

```bash
snapsql test --verbose queries/project_tasks.md
```

## CI/CDとの統合

GitHub Actionsでの自動テスト：

```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go install github.com/shibukawa/snapsql/cmd/snapsql@latest
      - run: snapsql test queries/
```

## 次のステップ

テストの実行が完了したら、[コード生成](./code-generation) に進みましょう。

## 関連セクション

* [テスト用コマンドリファレンス](../guides/command-reference/test.md)
* [フィクスチャとロード戦略](../guides/query-format/fixtures.md)
* [Expected / マッチャーの解説](../guides/query-format/error-testing.md)

