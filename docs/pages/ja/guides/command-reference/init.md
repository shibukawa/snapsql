# init コマンド

`snapsql init` は、プロジェクトの初期ディレクトリ構成とサンプル設定ファイルを自動で作成するコマンドです。新しいプロジェクトをすばやく開始するための雛形を生成します。

主な動作
- 必要なディレクトリを作成します（例: `queries/`, `constants/`, `generated/`, `testdata/mock/`）。
- サンプル設定ファイル `snapsql.yaml` を生成します。
- サンプル定数ファイル（`constants/database.yaml`）を生成します。
- Visual Studio Code 用に `.vscode/settings.json` を作成または更新し、YAML スキーマを紐付けます（エディタで `snapsql.yaml` の補完・検証が使いやすくなります）。

生成される主なファイル・ディレクトリ（例）
- `queries/` — SQL テンプレートを配置するディレクトリ（空ディレクトリを生成）
- `constants/` — 定数定義ファイルを置くディレクトリ（`database.yaml` を生成）
- `internal/query` — Goのコードジェネレータの出力先
- `testdata/mock/` — モックデータ出力用ディレクトリ
- `snapsql.yaml` — 設定ファイル
- `.vscode/settings.json` — YAML スキーマの関連付けを含む設定（既存ファイルがあればマージされます）

tbls の設定ファイルサンプル
一部のコマンド（例: `snapsql query`、テスト実行）はデータベース接続情報を `.tbls.yml`（tbls の設定）から取得することがあります。以下は最低限のサンプルです。

`.tbls.yml` の例（簡易）:
```yaml
project: sample-project
environments:
	development:
		driver: postgres
		dsn: "postgres://user:password@localhost:5432/mydb"
	production:
		driver: postgres
		dsn: "postgres://prod_user:prod_pass@db.example.com:5432/proddb"
```

tbls の `database.yml`（別名）を用いる場合の例（tbls のバリエーションに合わせてください）:
```yaml
environments:
	development:
		database: mydb
		host: localhost
		user: user
		password: password
		port: 5432
```

用途の説明
- `.tbls.yml`（または tbls が使用する設定ファイル）は、実行時に接続先を決定するために利用されます。機密情報は環境変数や CI シークレットで管理してください。

注意点
- 既存の `snapsql.yaml` を上書きするため、プロジェクトですでに設定がある場合は事前にバックアップしてください。

