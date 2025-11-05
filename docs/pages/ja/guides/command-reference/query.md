# query コマンド

`snapsql query` はクエリテンプレート（`.snap.md` / `.snap.sql`）を評価し、生成された SQL を指定したデータベースに対して実行または検査するためのコマンドです。アプリケーションコードを作成することなく、SQL テンプレートをそのまま実行して動作確認や検証を行えます。


## 利用例

```bash
# dry-run で SQL を確認（Postgres 方言で整形）
snapsql query --dry-run --dialect postgresql -P params.yaml queries/board_list.snap.md

# tbls 設定の environment を使って実行（config の default を明示的に上書き）
snapsql query --env development -P params.yaml queries/board_list.snap.md

# 直接 DSN を指定して実行し、結果を JSON ファイルに出力
snapsql query --db "postgres://user:pass@localhost:5432/db" --format json -o result.json queries/board_list.snap.md
```

## フラグ

- `--params, -P <file>`: パラメータファイル（JSON/YAML）を読み込みます。拡張子により `.json` / `.yaml` / `.yml` を判別してパースします。
- `--param, -p key=value` : 個別のパラメータをコマンドラインで渡します。`key=value` の形式で複数指定可。値が JSON リテラル（`{...}` / `[...]`）なら自動でパースされます。
- `--const <file>` : 定数ファイルを追加読み込みし、テンプレート実行時のパラメータにマージします（コマンドライン `--param` が優先されます）。
- `--db-connection=<dsn>` : 直接 DSN を指定して接続します（例: `postgres://user:pass@host:5432/db`）。接続文字列から自動的にドライバを推測します。
-- `--tbls-config=<path>` : tbls 設定ファイル（`.tbls.yaml` / `.tbls.yml`）のパスを明示的に指定します。指定しない場合はカレントディレクトリや `--config` 経由で自動検出されます。
- `--format=<table|json|csv|yaml|markdown>` : 出力フォーマットを指定します（デフォルト: `table`）。
- `--output, -o <file>` : 出力先ファイルを指定（デフォルトは stdout）。
- `--timeout=<seconds>` : クエリタイムアウト（秒、デフォルト `30`）。
 - `--explain` / `--explain-analyze` : 実行計画を表示します。`--explain-analyze` を指定すると内部的に `--explain` が有効になります。
 - `--limit=<n>` / `--offset=<n>` : 実行時に `LIMIT`/`OFFSET` 相当の処理を適用します（テンプレートの指示と合わせて利用できます）。注意: これらは基本的に **SELECT** クエリにのみ適用されます。INSERT/UPDATE/DELETE などには適用されません。
 - `--execute-dangerous-query` : WHERE 句のない DELETE/UPDATE 等の「危険なクエリ」を明示的に実行するためのフラグ。コマンドラインで指定がない場合は設定ファイルの `query.execute_dangerous_query` を参照します。
- `--dry-run` : DB に接続せずに SQL のレンダリング結果（およびバインドされるパラメータ）を表示します。`--dialect` を指定すると方言に合わせた整形（CAST/CONCAT などの方言変換）を適用して表示します。
- `--dialect=<postgresql|mysql|sqlite|mariadb>` : dry-run や DB がない場合に方言を指定して SQL を整形します。指定がない場合は接続先ドライバから推測します。

## DB接続設定

- 通常はプロジェクトで管理している `.tbls.yaml`（tbls 設定）を使って接続情報を取得します。特定の tbls 設定ファイルを指定したい場合は `--tbls-config <path>` を使って明示的にパスを渡せます。
- 単発で別のデータベースに対して試す場合や CI から直接接続したい場合は `--db-connection` で DSN を直接上書きできます（ローカルでの検証や CI ジョブ向け）。

## 生成される SQL と dry-run の挙動

- `--dry-run` ではテンプレート検証とパラメータ検証の後、最適化済みの命令列から SQL と引数を生成します。生成された SQL は指定された方言向けに整形されて出力されます。
- 出力には以下が含まれます（冗長モードでない限り出力の一部省略されることがあります）:
	- テンプレートファイル名
	- 生成された SQL（方言変換済み）
	- バインドされるパラメータ（位置付き $1, $2...）
	- 入力パラメータマップ（`--params` / `--param`）
	- 危険なクエリ検出時の警告メッセージ

## 実行時の動作

- コード生成を行って実行するのと同じように、DBに実際に接続して実テンプレートを評価・実行し、SQLを組み立ててPREPAREステートメントを使って準備したあとにパラメータを渡してクエリーを送信します。
- 実行結果は `table`/`json`/`csv`/`yaml`/`markdown` のいずれかに整形して出力します。
- `--explain` を指定すると実行計画を取得して整形して返します。`--explain-analyze` を使うと実行統計を収集し詳細な解析を行います。

## パラメータの扱い

- `--params=<file>` `-P <file>`で読み込んだ YAML/JSON はテンプレート評価時に `map[string]any` として渡されます。
- `--param=key=value` `-p key=value` はファイルパラメータより優先してマージされます。値が `true`/`false`/数値/JSON リテラルに適合する場合は自動で型変換されます。

## 危険なクエリ保護

- SnapSQLは実行前に SQL を簡易解析してWHERE句が欠落した `DELETE`/`UPDATE` の「危険なクエリ」を検出します。
- 生成されたコードでも許可オプションを渡さないと実行されませんが、それと同様にqueryコマンドでも検出された場合は`--execute-dangerous-query` が指定されていない限りエラーメッセージを出力し実行を中止します。

## テンプレート内のパフォーマンス閾値

- テンプレート（`.snap.md` の front-matter や `.snap.sql` の定義部）に `slow_query_threshold` などを設定しておくと、実行後の性能解析（`explain` による解析）でその閾値を参照します。`query` 実装はテンプレートからこの閾値を抽出して `analyzePerformance` に渡します。
