# CLIコマンド

`snapsql`コマンドラインツールは、SnapSQLプロジェクトで作業するための様々なコマンドを提供します。

## グローバルオプション

```bash
snapsql [グローバルオプション] <コマンド> [コマンドオプション]
```

### グローバルフラグ

- `--config <ファイル>` - 設定ファイルパス（デフォルト: `snapsql.yaml`）
- `--verbose` - 詳細出力を有効化
- `--quiet` - 必須でない出力を抑制
- `--help` - ヘルプ情報を表示

## コマンド

### init - プロジェクト初期化

サンプルファイル付きの新しいSnapSQLプロジェクトを作成します。

```bash
snapsql init <プロジェクト名>
```

**オプション:**
- `--template <名前>` - 特定のプロジェクトテンプレートを使用
- `--database <ドライバー>` - デフォルトデータベースドライバーを設定（postgres、mysql、sqlite3）

**例:**
```bash
# 新しいプロジェクトを作成
snapsql init my-project

# PostgreSQLテンプレートで作成
snapsql init my-project --database postgres
```

### generate - 中間ファイル生成

SQLテンプレートを処理し、中間JSONファイルを生成します。

```bash
snapsql generate [オプション]
```

**オプション:**
- `--output <ディレクトリ>` - 生成ファイルの出力ディレクトリ（デフォルト: `./generated`）
- `--force` - 既存の生成ファイルを上書き

**例:**
```bash
# すべてのテンプレートを生成
snapsql generate

# カスタム出力ディレクトリで生成
snapsql generate --output ./build

# 生成
snapsql generate
```

### query - クエリ実行

パラメータ付きでSQLテンプレートを実行します。

```bash
snapsql query <テンプレートファイル> [オプション]
```

**オプション:**
- `--dry-run` - 実行せずに生成されたSQLを表示
- `--params-file <ファイル>` - JSON/YAMLファイルからパラメータを読み込み
- `--param <キー=値>` - 個別パラメータを設定（複数回使用可能）
- `--output <ファイル>` - 結果を標準出力ではなくファイルに書き込み
- `--format <形式>` - 出力形式: `table`、`json`、`csv`（デフォルト: `table`）
- `--limit <n>` - 返す行数を制限
- `--offset <n>` - 結果セットのオフセット
- `--timeout <期間>` - クエリタイムアウト（例: `30s`、`5m`）
- `--explain` - クエリ実行計画を表示
- `--explain-analyze` - 統計付きの詳細実行計画を表示
- `--execute-dangerous-query` - WHERE句なしのDELETE/UPDATEを許可

**例:**
```bash
# 生成されたSQLを確認するドライラン
snapsql query queries/users.snap.sql --dry-run --params-file params.json

# パラメータ付きで実行
snapsql query queries/users.snap.sql --param active=true --param limit=50

# JSON形式で出力
snapsql query queries/users.snap.sql --format json --output results.json

# 実行計画を表示
snapsql query queries/users.snap.sql --explain --params-file params.json
```

### validate - テンプレート検証

SQLテンプレートの構文とパラメータの一貫性を検証します。

```bash
snapsql validate [テンプレートファイル...]
```

**オプション:**
- `--all` - プロジェクト内のすべてのテンプレートを検証
- `--strict` - 厳密検証モードを有効化
- `--check-params` - パラメータ使用を検証

**例:**
```bash
# 特定のテンプレートを検証
snapsql validate queries/users.snap.sql

# すべてのテンプレートを検証
snapsql validate --all

# 厳密検証
snapsql validate --all --strict
```

### config - 設定管理

プロジェクト設定を管理します。

```bash
snapsql config <サブコマンド>
```

**サブコマンド:**
- `show` - 現在の設定を表示
- `validate` - 設定ファイルを検証
- `test-db` - データベース接続をテスト

**例:**
```bash
# 現在の設定を表示
snapsql config show

# 設定を検証
snapsql config validate

# データベース接続をテスト
snapsql config test-db
```

### pull - リモートテンプレート取得

リモートソースからSQLテンプレートを取得します（計画中の機能）。

```bash
snapsql pull <ソース> [オプション]
```

**オプション:**
- `--branch <名前>` - ブランチ/バージョンを指定
- `--output <ディレクトリ>` - 出力ディレクトリ

### version - バージョン表示

バージョン情報を表示します。

```bash
snapsql version
```

## パラメータ形式

### コマンドラインパラメータ

```bash
# シンプルパラメータ
--param name=john --param active=true --param limit=50

# ネストされたパラメータ（ドット記法を使用）
--param filters.active=true --param pagination.limit=20
```

### JSONパラメータファイル

```json
{
  "name": "john",
  "active": true,
  "filters": {
    "active": true,
    "department": "engineering"
  },
  "pagination": {
    "limit": 20,
    "offset": 0
  }
}
```

### YAMLパラメータファイル

```yaml
name: john
active: true
filters:
  active: true
  department: engineering
pagination:
  limit: 20
  offset: 0
```

## 出力形式

### テーブル形式（デフォルト）

```
+----+----------+-------------------+
| id | name     | email             |
+----+----------+-------------------+
|  1 | John Doe | john@example.com  |
|  2 | Jane Doe | jane@example.com  |
+----+----------+-------------------+
```

### JSON形式

```json
{
  "rows": [
    {"id": 1, "name": "John Doe", "email": "john@example.com"},
    {"id": 2, "name": "Jane Doe", "email": "jane@example.com"}
  ],
  "count": 2,
  "execution_time": "15ms"
}
```

### CSV形式

```csv
id,name,email
1,John Doe,john@example.com
2,Jane Doe,jane@example.com
```

## 環境変数

- `SNAPSQL_CONFIG` - デフォルト設定ファイルパス
- `DATABASE_URL` - データベース接続文字列
- `SNAPSQL_VERBOSE` - 詳細出力を有効化（true/false）
- `SNAPSQL_QUIET` - 静寂モードを有効化（true/false）

## 終了コード

- `0` - 成功
- `1` - 一般エラー
- `2` - 設定エラー
- `3` - テンプレート検証エラー
- `4` - データベース接続エラー
- `5` - クエリ実行エラー

## 例

### 完全なワークフロー

```bash
# 1. プロジェクトを初期化
snapsql init my-project
cd my-project

# 2. 中間ファイルを生成
snapsql generate

# 3. ドライランでクエリをテスト
snapsql query queries/users.snap.sql --dry-run --params-file params.json

# 4. クエリを実行
snapsql query queries/users.snap.sql --params-file params.json --format json

# 5. すべてのテンプレートを検証
snapsql validate --all
```

### 開発ワークフロー

```bash
# 中間ファイルを生成
snapsql generate

# 開発中にクエリをテスト
snapsql query queries/new-query.snap.sql --dry-run --param test=true

# コミット前に検証
snapsql validate --all --strict
```

### 本番使用

```bash
# 本番データベースを設定
export DATABASE_URL="postgres://prod-server:5432/mydb"

# 本番パラメータで実行
snapsql query queries/report.snap.sql \
  --params-file prod-params.json \
  --format csv \
  --output daily-report.csv \
  --timeout 5m
```
