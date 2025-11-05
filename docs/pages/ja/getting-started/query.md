# Query サブコマンド

`snapsql query` サブコマンドは、作成した SQL テンプレート（`.snap.md` / `.snap.sql` などの Markdown/SQL ファイル）を読み込み、指定されたパラメータでテンプレート置換を行い、任意のデータベースで実行します。

このページでは、基本的な使い方、パラメータの渡し方、接続情報の指定方法を紹介します。

## 概要

- 入力: Markdown または SQL テンプレートファイル（例: `queries/get_active_users.snap.md`）
- 出力: SQL を実行して標準出力に結果を表示（またはフォーマットして出力）
- パラメータ: ファイル内の `Parameters` セクションを読み込む、または CLI から指定可能
- 接続情報: `.tbls.yml` / 環境変数 / コマンドラインフラグで指定

## 使い方（例）

基本的な実行例（仮定の CLI 形式）:

```bash
# ファイルを読み込んで実行（DSN をコマンドラインで渡す例）
snapsql query queries/get_active_users.snap.md --dsn "postgres://user:pass@localhost:5432/dbname?sslmode=disable"

# .tbls.yml / .tbls 設定を使う場合（プロジェクトルートに配置されている前提）
snapsql query queries/get_active_users.snap.md --use-tbls

# パラメータを CLI で JSON または YAML 形式で直接渡す
snapsql query queries/get_active_users.snap.md --params '{"limit":10,"status":true}'

# パラメータをファイルで渡す場合
snapsql query queries/get_active_users.snap.md --params-file params.yaml
```

`params.yaml` の例:

```yaml
limit: 10
offset: 0
status: true
```

## ファイルの読み取りルール

- コマンドはファイル内の `Parameters` セクション（YAML）を優先して読み込みます。CLI の `--params` / `--params-file` が指定された場合はそれが上書きされます。
- ファイル内で最初の見出し（`#`）は、生成される関数名やロギングに使用されます（例: `# Get Active Users` → `GetActiveUsers`）。
- テンプレート置換は SnapSQL のテンプレート仕様（two-way-sql）に従います。変数展開（`/*= ... */`）、条件（`/*# if ... */`）、ループ（`/*# for ... */`）が使えます。

## 接続情報の指定

- 環境変数方式（例）:

```bash
export TBLS_DSN="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
export TBLS_DRIVER=postgres
snapsql query queries/get_active_users.snap.md --use-tbls
```

- 直接 DSN 指定（コマンドライン引数）:

```bash
snapsql query queries/get_active_users.snap.md --dsn "user:pass@tcp(localhost:3306)/dbname?parseTime=true" --driver mysql
```

- SQLite の場合は DB ファイルを指定:

```bash
snapsql query queries/get_active_users.snap.md --dsn "./snapsql.db" --driver sqlite3
```

## 実行結果の取り扱い

- デフォルトで標準出力に結果（CSV/JSON/表形式）を表示します。オプションでファイルへ出力することも可能です（例: `--output results.json`）。
- 実行前にテンプレートの展開後 SQL を表示する `--dry-run` フラグを用意すると便利です。

### 実行オプション: `--explain`, `--limit`, `--offset`

`snapsql query` は実行時に追加オプションを渡して動作を変えられるようになっています。これらはコマンドラインで直接渡すことも、テスト用に `options.yaml` のようなファイルで指定することもできます。

- `--explain`（真偽フラグ）
	- 目的: テンプレートが正しく設定されているかを確認したり、アドホックに実際のデータベースで実行計画を取得して結果を検証するために使用します。実行プランを取得し、表示します。通常の結果比較（expected.yaml）ではなく、実行計画が非空であることを確認する用途に適しています。
	- 実装上の方針（DBMS差分）:
		- SQLite: `EXPLAIN QUERY PLAN <sql>` を実行してテキストとして返します。
		- PostgreSQL: `EXPLAIN <sql>` を実行します。
	- 例:

```bash
snapsql query queries/get_active_users.snap.md --dsn "postgres://user:pass@localhost/db" --explain
```

- `--limit` / `--offset`（整数）
	- 目的: SQL テンプレート内に LIMIT / OFFSET が無い場合、実行時にページングを簡単に試せるようにランタイムで追記します（ただし、テンプレート内に既に LIMIT/OFFSET がある場合は追記しません）。
	- 例:

```bash
snapsql query queries/get_active_users.snap.md --dsn "./snapsql.db" --driver sqlite3 --limit 10 --offset 20
```

- `options.yaml` 経由での指定
	- テストハーネスや再現可能な実行のため、`options.yaml` を用意して `limit` / `offset` / `explain` / `explain_analyze` / `driver` を指定できます（設計ドキュメントに準拠）。

注意: `--explain` を指定した実行は通常の `expected.yaml` による行列比較はスキップされ、実行計画の文字列が非空であることを検証する用途で使います（特に SQLite では計画の内容比較は行いません）。

## エッジケースと注意点

- ローカルの DB に影響を与えたくない場合は、ローカルのテスト用 DB（Docker コンテナや SQLite ファイル）で実行してください。
- テンプレートに埋め込まれたダミー値（`/*= ... */default`）は実行用のダミーであり、実際のデフォルト値ではありません。必ずパラメータを正しく渡してください。
- large result sets の取り扱い（ストリーミング / ページング）はクライアント側で検討してください。

## まとめ

1. まずはテンプレートを two-way-sql として実行して動作確認
2. Markdown 化して `Parameters` を整理
3. `snapsql query` で実行（`.tbls.yml` か `--dsn`）

---

注: このドキュメントは CLI のフラグ名や挙動について合理的に推定した例を使っています。実際のコマンドライン引数名やオプションは実装に従って微調整してください。

## 関連セクション

* [クエリフォーマット入門](../guides/query-format/index.md)
* [レスポンス型の扱い](../guides/query-format/response-types.md)
* [パラメータ仕様](../guides/query-format/parameters.md)
