## エラークエリーのテスト

このテストツールは、パースエラーや DB 接続エラーなどのランタイムエラーを「エラー」として扱います。一方で、テーブルの状態や入力データに依存して発生するエラー（外部キー制約違反、UNIQUE 制約違反、NOT NULL 制約違反、データ型の不整合など）は「期待されるエラー」として明示的に検証できます。本ページでは `**Expected Error:**` セクションの書式、サポートされるエラー種別、検証フロー、実例、注意点を実装に基づいて説明します。

### 概要と目的

- 通常のテストは成功（エラーなし）を期待しますが、エラー条件そのものを仕様の一部とする場合（不正な入力による制約違反など）は、期待されるエラーが発生することをテストで検証します。
- 期待エラーはテストケースで `Expected Error` セクションに記述します。パーサはこの項目を正規化して `TestCase.ExpectedError` に格納します。

### セクションの書式

Markdown のテストケース内で次のように記述します（大文字・小文字・アンダースコア・ハイフンの違いは正規化されます）：

```
**Expected Error:** unique violation
```

この文字列は内部で正規化され（小文字化、アンダースコア/ハイフンをスペースに戻す等）、既知のエラー種別と照合されます。無効なエラー名を指定するとパース時にエラーになります。

詳細なパース処理は `markdownparser/error_type.go` の `ParseExpectedError` を参照してください。

### サポートされるエラー種別（正規化後の文字列）

実装で定義されている代表的なエラー種別は以下です（正規化後の表記を示します）：

- "unique violation" — 一意制約違反（Postgres の 23505 等、MySQL の ER_DUP_ENTRY 等）
- "foreign key violation" — 外部キー制約違反（Postgres 23503 等）
- "not null violation" — NOT NULL 制約違反（Postgres 23502 等）
- "check violation" — CHECK 制約違反（Postgres 23514 等）
- "not found" — 該当レコードが見つからない（`no rows` 相当の状況）
- "data too long" — 文字列長制限違反（Postgres 22001 / MySQL ER_DATA_TOO_LONG 等）
- "numeric overflow" — 数値のオーバーフロー（Postgres 22003 等）
- "invalid text representation" — 不正なテキスト表現（数値パース失敗など、Postgres 22P02 等）

これらは `markdownparser/error_type.go` に列挙され、`ParseExpectedError` で検証されます。入力時は `unique_violation` や `NOT-NULL-VIOLATION` といった別表記も受け付け、正規化されて比較されます。

### 実際のエラー分類（ランタイム）

実行時のエラーはGoのDBドライバ固有のエラー情報から分類されます（`ClassifyDatabaseError`、`classifyPostgresError`、`classifyMySQLError`、`classifySQLiteError`）。代表的な分類ルール:

- PostgreSQL (pgx): SQLSTATE コードによって分類（例: 23505 → unique violation）
- MySQL: エラー番号によって分類（例: 1062 → unique violation）
- SQLite: extended error code による分類
- 汎用フォールバック: エラーメッセージに "no rows" や "not found" が含まれる場合は `not found` とする

### 検証フロー（実行時）

テストランナーは次の流れで期待エラーを扱います:

1. テストを実行し、エラーが発生したかどうかを取得します。
3. 実際のエラーが `nil` なら失敗（期待エラーが発生しなかった）。
4. 実エラーがある場合、`ClassifyDatabaseError` によりエラーの種類を判定し、期待値と比較します。
5. 比較結果に応じてテストは成功/失敗と判断され、詳細なメッセージが出力されます。

### 期待エラーの表現例

- シンプルな期待エラー指定

```
**Expected Error:** unique_violation
```

- 別表記も許容（正規化されます）

```
**Expected Error:** UNIQUE VIOLATION
```

### 実例１: UNIQUE 制約違反を期待する

```markdown
**TestCase**: insert duplicate

Fixtures:
  users:
    - id: 1
      email: "alice@example.com"

Queries:
  - sql: |
      INSERT INTO users (id, email) VALUES (2, /*= email */'alice@example.com')

  - Expected Error: unique_violation
```

期待される動作: INSERT 実行で一意制約違反が起き、テストは成功となる。

### 実例２: 外部キー制約違反を期待する

```markdown
Fixtures:
  users: []

Queries:
  - sql: |
      INSERT INTO posts (id, user_id, title) VALUES (1, 9999, 'x')

  - Expected Error: foreign_key_violation
```

### 注意点と運用ガイド

- DB 実装差: Postgres / MySQL / SQLite で発生するエラーコードやメッセージが異なるため、CI 環境では対象 DB を明確にし、必要なら DB 固有の期待値を分けて管理してください。
- 検証対象のエラーは主に DB 側のランタイム例外（制約違反、型パースエラー、データ長超過など）です。ネットワーク接続エラーやドライバ初期化エラー等は通常 `ClassifyDatabaseError` で分類できないため、これらは一般的なエラー扱いになります。
- エラーメッセージでの厳密マッチングは不安定になりがちです。可能であれば `Expected Error` に限定した種別名（例: `unique violation`）でのマッチングを利用してください。
