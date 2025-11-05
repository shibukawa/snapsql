# inspect コマンド — 使い方（日本語）

## 概要

`snapsql inspect` は SQL クエリを解析し、テーブル参照やクエリ構造のメタ情報を抽出するコマンドです。
解析結果は JSON または CSV 形式で出力でき、クエリの依存関係分析やデバッグ、ドキュメント生成などに利用できます。

- 入力: SQL ファイルまたは stdin
- 出力: JSON（デフォルト）または CSV 形式でテーブル参照情報を出力

## 使い方（概要）

基本構文:

```sh
snapsql inspect [flags] <sql-file>
```

主なフラグ:

- `--stdin`
  - 説明: 標準入力から SQL を読み込みます。
  - 利用例: パイプ経由で SQL を渡す場合に使用します。

- `--pretty`
  - 説明: JSON 出力を整形して読みやすくします（インデント付き）。

- `--strict`
  - 説明: 厳密モード。構文エラーや未対応の構造がある場合にエラーで終了します。
  - デフォルト: false（部分的な解析結果を返す）

- `--format <format>`
  - 説明: 出力形式を指定します。`json`（デフォルト）または `csv` を選択できます。
  - 利用例: `--format csv` でテーブル参照情報を CSV 形式で出力します。

## 出力形式

### JSON 形式（デフォルト）

JSON 形式では以下の情報が出力されます：

```json
{
  "statement": "<クエリ種別>",
  "tables": [
    {
      "name": "<テーブル名>",
      "alias": "<エイリアス>",
      "schema": "<スキーマ名>",
      "source": "<参照元種別>",
      "join_type": "<結合種別>",
      "query_name": "<CTE/サブクエリ名>",
      "is_table": true
    }
  ],
  "notes": ["<注意事項>"]
}
```

**フィールド説明:**

- `statement`: クエリの種類（`select`, `insert`, `update`, `delete`）
- `tables`: 参照されるテーブルの配列
  - `name`: テーブル名
  - `alias`: エイリアス（AS で指定された別名）
  - `schema`: スキーマ名（schema.table 形式の場合）
  - `source`: 参照元の種別
    - `main`: メインのテーブル（FROM 句の最初のテーブル、INSERT/UPDATE/DELETE のターゲットテーブル）
    - `join`: JOIN で結合されたテーブル
    - `cte`: CTE（WITH 句で定義された Common Table Expression）内部のテーブル参照
    - `subquery`: サブクエリ内部のテーブル参照
  - `join_type`: 結合の種別
    - `none`: 結合なし（メインテーブル）
    - `inner`: INNER JOIN
    - `left`: LEFT JOIN / LEFT OUTER JOIN
    - `right`: RIGHT JOIN / RIGHT OUTER JOIN
    - `full`: FULL OUTER JOIN
    - `cross`: CROSS JOIN
    - `natural`: NATURAL JOIN（およびその変種）
  - `query_name`: (オプション) CTE またはサブクエリの名前。このテーブルがどの CTE/サブクエリ内で参照されているかを示します。
  - `is_table`: テーブルが実データベーステーブルであるか、CTE やサブクエリなどの仮想テーブルであるかを示すブール値。
    - `true`: 実テーブル（データベースに存在するテーブル）
    - `false`: CTE またはサブクエリ（仮想テーブル参照）
- `notes`: 解析時の注意事項やメッセージ（部分解析時など）

### CSV 形式

CSV 形式では以下のカラムでテーブル参照情報が出力されます：

```csv
name,alias,schema,source,joinType,queryName,isTable
<テーブル名>,<エイリアス>,<スキーマ名>,<参照元種別>,<結合種別>,<CTE/サブクエリ名>,<実テーブル/仮想テーブル>
```

ヘッダー行が含まれ、各テーブル参照が 1 行で表現されます。
- `queryName` カラムは、そのテーブルがどの CTE またはサブクエリ内で参照されているかを示します（メインクエリのテーブルの場合は空）
- `isTable` カラムは、テーブルが実テーブルか仮想テーブル（CTE/サブクエリ）かを示します（`true` または `false`）

## 例: コマンド & 出力

### 1) シンプルな SELECT クエリ

**入力 SQL:**

```sql
SELECT id, name FROM users;
```

**実行:**

```sh
echo "SELECT id, name FROM users;" | snapsql inspect --stdin --pretty
```

**出力:**

```json
{
  "statement": "select",
  "tables": [
    {
      "name": "users",
      "source": "main",
      "join_type": "none",
      "is_table": true
    }
  ]
}
```

### 2) JOIN を含む SELECT クエリ

**入力 SQL:**

```sql
SELECT u.id, o.total
FROM users u
INNER JOIN orders o ON o.user_id = u.id;
```

**実行:**

```sh
snapsql inspect query.sql --pretty
```

**出力:**

```json
{
  "statement": "select",
  "tables": [
    {
      "name": "users",
      "alias": "u",
      "source": "main",
      "join_type": "none",
      "is_table": true
    },
    {
      "name": "orders",
      "alias": "o",
      "source": "join",
      "join_type": "inner",
      "is_table": true
    }
  ]
}
```

### 3) CTE（WITH 句）を含むクエリ

**入力 SQL:**

```sql
WITH recent AS (
  SELECT user_id FROM orders WHERE created_at > NOW() - INTERVAL '7 days'
)
SELECT u.id, u.name
FROM users u
LEFT JOIN recent r ON r.user_id = u.id;
```

**実行:**

```sh
cat query_with_cte.sql | snapsql inspect --stdin --pretty
```

**出力:**

```json
{
  "statement": "select",
  "tables": [
    {
      "name": "users",
      "alias": "u",
      "source": "main",
      "join_type": "none",
      "is_table": true
    },
    {
      "name": "recent",
      "alias": "r",
      "source": "join",
      "join_type": "left",
      "is_table": false
    },
    {
      "name": "orders",
      "source": "cte",
      "join_type": "none",
      "query_name": "recent",
      "is_table": true
    }
  ]
}
```

**説明:**

- `recent` は CTE として定義されていますが、メインクエリで `LEFT JOIN` されているため `source` は `"join"` になります。また、CTE 参照なので `is_table` は `false` です。
- **CTE 内部のテーブル `orders` も抽出されます**。`source` が `"cte"` で、`query_name` が `"recent"` となっており、この `orders` テーブルが CTE `recent` の定義内で参照されている実テーブルであることがわかります。`is_table` は `true` です（実データベーステーブル）。

### 4) サブクエリを含むクエリ

**入力 SQL:**

```sql
SELECT s.id
FROM (SELECT id, name FROM users WHERE active = true) s
JOIN orders o ON o.user_id = s.id;
```

**実行:**

```sh
snapsql inspect subquery.sql --pretty
```

**出力:**

```json
{
  "statement": "select",
  "tables": [
    {
      "name": "users",
      "alias": "s",
      "source": "subquery",
      "join_type": "none",
      "is_table": true
    },
    {
      "name": "orders",
      "alias": "o",
      "source": "join",
      "join_type": "inner",
      "is_table": true
    }
  ]
}
```

**説明:**

- サブクエリ `(SELECT ... FROM users)` 内の実テーブル `users` は `source` が `"subquery"` として認識されます。
- サブクエリ内の実際のテーブル名（`users`）が `name` として抽出され、サブクエリのエイリアス（`s`）が `alias` に設定されます。
- 両テーブルとも実テーブルなので `is_table` は `true` です。

### 5) スキーマ修飾されたテーブル

**入力 SQL:**

```sql
SELECT u.id
FROM public.users u
JOIN sales.orders o ON o.user_id = u.id;
```

**実行:**

```sh
snapsql inspect schema_qualified.sql --pretty
```

**出力:**

```json
{
  "statement": "select",
  "tables": [
    {
      "name": "users",
      "alias": "u",
      "schema": "public",
      "source": "main",
      "join_type": "none",
      "is_table": true
    },
    {
      "name": "orders",
      "alias": "o",
      "schema": "sales",
      "source": "join",
      "join_type": "inner",
      "is_table": true
    }
  ]
}
```

**説明:**

- `schema.table` 形式のテーブル参照では、`schema` フィールドにスキーマ名が設定されます。

### 6) INSERT ... SELECT with CTE

**入力 SQL:**

```sql
WITH latest AS (
  SELECT user_id FROM orders WHERE status = 'completed'
)
INSERT INTO snapshots (user_id)
SELECT l.user_id FROM latest l;
```

**実行:**

```sh
snapsql inspect insert_cte.sql --pretty
```

**出力:**

```json
{
  "statement": "insert",
  "tables": [
    {
      "name": "latest",
      "alias": "l",
      "source": "main",
      "join_type": "none",
      "is_table": false
    },
    {
      "name": "snapshots",
      "source": "main",
      "join_type": "none",
      "is_table": true
    },
    {
      "name": "orders",
      "source": "cte",
      "join_type": "none",
      "query_name": "latest",
      "is_table": true
    }
  ]
}
```

**説明:**

- `INSERT` 文の場合、ターゲットテーブル（`snapshots`）と SELECT 部分で参照される CTE（`latest`）、さらに **CTE 内部で参照される実テーブル**（`orders`）が出力されます。
- `snapshots` と `latest` は `source: "main"` ですが、意味が異なります：
  - `snapshots`: INSERT の対象テーブル（実テーブル、`is_table: true`）
  - `latest`: SELECT 部分で参照される CTE（仮想テーブル、`is_table: false`）
- `orders` は CTE 内部で参照される実テーブルなので、`source: "cte"` かつ `query_name: "latest"` で識別され、`is_table: true` です。
- **まとめ:** `is_table` フィールドを使用することで、以下の区別が明確になります：
  - `is_table: true` → 実データベーステーブル（永続データ）
  - `is_table: false` → CTE またはサブクエリ（仮想テーブル）

### 7) CSV 形式での出力

**入力 SQL:**

```sql
SELECT u.id FROM users u JOIN orders o ON o.user_id = u.id;
```

**実行:**

```sh
snapsql inspect query.sql --format csv
```

**出力:**

```csv
name,alias,schema,source,joinType,queryName,isTable
users,u,,main,none,,true
orders,o,,join,inner,,true
```

**説明:**

- CSV 形式では、各テーブル参照が 1 行で表現されます。
- 空のフィールド（`schema` がない場合など）は空白で表示されます。
- `isTable` カラムに `true` または `false` が出力されます。

## 用途と活用例

1. **クエリ依存関係の分析**
   - プロジェクト内のすべての SQL ファイルに対して `inspect` を実行し、どのクエリがどのテーブルを参照しているかを一覧化できます。

2. **ドキュメント自動生成**
   - テーブル参照情報を元に、クエリのドキュメントやER図を自動生成できます。

3. **リファクタリング支援**
   - テーブル名やスキーマを変更する際に、影響を受けるクエリを特定できます。

4. **デバッグ**
   - 複雑なクエリ（CTE、サブクエリ、複数の JOIN）がどのように解釈されているかを確認できます。

5. **CI/CD パイプライン**
   - `--strict` モードを使って、構文エラーや未対応の構造を含むクエリを検出できます。

## よくある質問 / 注意点

- Q: CTE 内のテーブル参照は抽出されますか？
  - A: はい、CTE 内部で参照されるテーブルはすべて抽出されます。CTE 自体は仮想テーブルなので `is_table: false` で出力され、CTE 内部で参照される実テーブルは `source: "cte"` かつ `query_name: "<CTE名>"` で識別されます。

- Q: サブクエリのネストが深い場合はどうなりますか？
  - A: 最も外側のレベルで認識されるサブクエリとそのベーステーブルが抽出されます。深くネストされたサブクエリ内部のテーブルについても `source: "subquery"` かつ `query_name` が設定されて抽出されます。

- Q: `is_table` フィールドの用途は？
  - A: `is_table` フィールドにより、テーブルが実データベーステーブル（`true`）なのか、CTE やサブクエリなどの仮想テーブル（`false`）なのかが一目瞭然になります。これにより、クエリの依存関係分析やデータ影響範囲の特定が容易になります。

- Q: `--strict` モードはいつ使うべきですか？
  - A: CI/CD パイプラインでクエリの妥当性を検証したい場合や、完全に解析可能なクエリのみを扱いたい場合に使用します。開発中のデバッグでは `--strict` なしで部分的な結果を得る方が便利です。

- Q: JSON と CSV のどちらを使うべきですか？
  - A: プログラムで解析結果を処理する場合は JSON、スプレッドシートや簡易的な分析には CSV が便利です。

## 追加のヒント

- 複数のクエリファイルを一括で解析する場合は、シェルスクリプトでループ処理を組み合わせると効率的です：

```sh
for file in queries/*.sql; do
  echo "=== $file ==="
  snapsql inspect "$file" --format csv
done
```

- `jq` などの JSON 処理ツールと組み合わせて、特定のテーブルを参照するクエリをフィルタリングできます：

```sh
snapsql inspect query.sql | jq '.tables[] | select(.name == "users")'
```
