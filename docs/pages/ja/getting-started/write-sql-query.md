# Write SQL Query

このページでは、まずテンプレート化を行う前に「実際に実行できる SQL」（two-way-sql）としてクエリを確認する流れを推奨します。two-way-sql とは、テンプレートの制御構文や変数指定をコメント形式で書くことで、コメントを除去すればそのまま標準 SQL として実行できる方式です。

ワークフローの要点：

- 1) まず SQL を通常の SQL として作成・実行して動作を確認する（コメントを取り除いた状態で有効な SQL になること）
- 2) 動作が確認できたら Markdown 形式にして、Description や Parameters を追加してドキュメント化する

この順に進めることで、開発中に SQL エディタやリンター、実行計画ツールをそのまま利用できます。

## まずは実行してみる（two-way-sql）

テンプレート化の前に、DBMSごとに実際に SQL を実行して動作を確認します。以下は PostgreSQL / MySQL / SQLite ごとの最小実行例をタブで切り替えられるようにしたものです。各タブ内の SQL はコメントを残したままでも読みやすい two-way-sql の例になっています。

::: tabs

== PostgreSQL

```sql
-- two-way-sql の例（PostgreSQL）
SELECT id, name
FROM users
WHERE active = true
  AND created_at > '2024-01-01';
```

実行例（ホストに psql がある場合）:

```bash
psql -h localhost -U your_user -d your_db -c "SELECT id, name FROM users WHERE active = true;"
```

コンテナ環境で実行する場合は、コンテナを立ち上げて psql に流し込む方法もあります。

== MySQL

```sql
-- two-way-sql の例（MySQL）
SELECT id, name
FROM users
WHERE active = TRUE
  AND created_at > '2024-01-01';
```

実行例（ホストに mysql クライアントがある場合）:

```bash
mysql -h 127.0.0.1 -P 3306 -u your_user -p your_db -e "SELECT id, name FROM users WHERE active = TRUE;"
```

コンテナ環境であれば `docker exec -i <container> mysql ...` のように実行できます。

== SQLite

```sql
-- two-way-sql の例（SQLite）
SELECT id, name
FROM users
WHERE active = 1
  AND created_at > '2024-01-01';
```

実行例:

```bash
sqlite3 snapsql.db "SELECT id, name FROM users WHERE active = 1;"
```

:::

各 DBMS で期待する結果が返ることを確認してください。動作確認が取れたら、次のステップとして Markdown 化し、`Description` や `Parameters`、そしてテンプレートディレクティブ（/*= ... */, /*# if ... */, /*# for ... */）を追加していきます。

## Markdown 化（ファイル構成の例）

基本的な Markdown 構成例です。先ほどのSQLをテンプレート化したものを書きます。

- ファイルは Markdown 形式（拡張子は `.snap.md` など）で保存できます。簡単な SQL のみであれば `.snap.sql` でも構いません。
- ドキュメントの最初の見出し（`# Query Title`）が、コード生成時に出力される関数名になります。分かりやすく、機能を表す名前を付けてください。
- `Description` セクションにはクエリの目的、どのような場面で使うか、期待される入力・副作用などの仕様をまとめて書けます。つまり、説明と仕様（どの環境で使うか、制約、注意点など）を同じ場所に記述できます。これにより文芸的プログラミング（Literate Programming）的に、説明と実装を同一ファイルで管理できます。
- パラメータにはテンプレートの中の変数展開や条件分岐で利用されるパラメータを記述します

````markdown
# Get Active Users

## Description
アクティブなユーザーを取得します。管理画面の一覧表示や API の一覧取得で利用します。

## Parameters
```yaml
limit: int
offset: int
status: bool
```

## SQL
```sql
SELECT id, name, email
FROM users
WHERE 1=1
  /*# if status */
  AND active = /*= status */true
  /*# end */
ORDER BY created_at DESC
LIMIT /*= limit */10
OFFSET /*= offset */0;
```
````

上の例では、`# Get Active Users` が出力される関数名のベースになります。`Description` に使用される場面や制約を書けるため、読み手と実装者の両方に情報が伝わります。

## テンプレート仕様（概要）

以下は SnapSQL のテンプレートでサポートされる主要構文の概要です（実装仕様は設計ドキュメントに準拠）。テンプレートはすべてコメント形式のディレクティブを使うため、コメントを除去すると標準 SQL として動作します。

### 変数展開（パラメータ）

書式:

```sql
/*= expression */[dummy_value]
```

例:

```sql
SELECT * FROM users WHERE id = /*= user_id */1;
```

- `expression` は変数名またはオブジェクト参照（例: `user.id`）が使えます。
- `dummy_value` はテンプレートを SQL エディタ等でそのまま実行できるように付ける「実行用ダミー」です。重要: このダミーは実行時のデフォルト値ではありません。実際のクエリ実行時にはテンプレート置換や呼び出し側のパラメータが使われます。```

### 条件分岐（if / elseif / else）

書式:

```sql
/*# if condition */
   -- 条件が true のときに含める SQL
/*# elseif other_condition */
   -- 別の条件
/*# else */
   -- デフォルト
/*# end */
```

例:

```sql
WHERE 1=1
/*# if filters.status */
  AND status = /*= filters.status */'active'
/*# end */
```

-- 条件式には Google CEL を使用できます（例: `filters.count > 0`）。詳細は [CEL ドキュメント](https://cel.dev/?hl=ja) を参照してください。
- ブロックは `/*# end */` で閉じます。

### ループ（for）

書式:

```sql
/*# for item : collection */
  -- 繰り返し内で item を参照
  /*= item.field */
/*# end */
```

例（IN句に展開）:

```sql
WHERE id IN (
  /*# for id : user_ids */
    /*= id */1
  /*# end */
)
```

- ループ内でのカンマ処理や SQL リストの整形はエンジン側で自動的に行われます。ループの末尾かどうかを判断するディレクティブはサポートしていません。末尾のカンマは不要であれば自動的に除去されます。

### 配列／リストの自動展開

配列を直接展開できる構文もサポートしています。例えば：

```sql
WHERE department IN (/*= departments */'sales', 'marketing')
```

実行時に `departments = ['engineering','design']` のような値が入ると、

```sql
WHERE department IN ('engineering','design')
```

のように展開されます。

### 自動調整機能

- カンマの自動除去: 条件が除去された場合、その行のカンマが自動的に整理されます。
- 空の句の除去: WHERE/ORDER BY/LIMIT などが空のときは該当句を削除します。

### 型変換とダミー値

- 明示的なダミー値を推奨します: `/*= value */123`。
- 必要に応じて CAST を使用できます: `CAST(/*= date_str */'2024-01-01' AS DATE)`。

## パラメータで使える型（簡潔）

ここでは簡潔に、テンプレートのパラメータでよく使う基本型を紹介します。詳細な型仕様やエイリアス、拡張は `guides` 以下で別途説明することを想定しています。

- プリミティブ型（基本）: `string`, `int`, `float`, `bool`
- 日時系: `date`, `datetime`, `timestamp`
- 特殊: `uuid`, `json`, `any`

- 配列: `int[]`, `string[]` のように `[]` を付けるか、YAML でリスト記法（`[int]` や `- item: int`）で表現できます。配列は IN句や複数行INSERTの展開で利用されます。

- 複合型（オブジェクト/ネスト）: パラメータはネストしたオブジェクトとして定義できます。

```yaml
parameters:
  user:
    id: int
    name: string
    profile:
      email: string
      age: int
  tags: [string]
```

上記のように、複合型や配列の組み合わせが可能です。ここではエイリアスや細かい制約は省略しています。

## ファイル形式と命名

- `.snap.sql`: SQL テンプレートのみを含むファイル（軽量）
- `.snap.md`: Markdown で説明と SQL を併記するファイル（文芸的プログラミングに最適）

推奨ファイル名例: `queries/get_active_users.snap.md`

## ベストプラクティス（簡潔）

- まず実行可能な SQL を作る（two-way-sql）
- その後 Markdown にして Description と Parameters を追加する
- 変数には明示的ダミー値を付け、型を分かりやすくする
- 複雑なロジックはテンプレート側でなく呼び出し側で前処理する

## 次のステップ

テンプレート化やテスト実行を行いたい場合は [Testing](./testing) に進んでください。

## 関連セクション

* [Markdown 形式のクエリ記述](../guides/query-format/markdown-format.md)
* [テンプレート構文の詳細](../guides/query-format/template-syntax.md)
* [パラメータと型の解説](../guides/query-format/parameters.md)