# 設定

SnapSQLプロジェクトは、プロジェクトルートの`snapsql.yaml`ファイルを使用して設定されます。

## プロジェクト構造

```
my-project/
├── snapsql.yaml          # メイン設定ファイル
├── queries/              # SQLテンプレートファイル
│   ├── users.snap.sql
│   └── posts.snap.sql
├── params.json          # デフォルトパラメータ（オプション）
├── constants.yaml       # プロジェクト定数（オプション）
└── generated/           # 生成された中間ファイル
    └── queries.json
```

## 設定ファイル (snapsql.yaml)

### 基本設定

```yaml
# プロジェクトメタデータ
name: "my-project"
version: "1.0.0"
description: "My SnapSQL project"

# データベース設定
database:
  default_driver: "postgres"
  connection_string: "${DATABASE_URL}"
  timeout: "30s"

# クエリ設定
query:
  execute_dangerous_query: false
  default_format: "table"
  
# ファイルパス
paths:
  queries: "./queries"
  generated: "./generated"
  constants: "./constants.yaml"
  params: "./params.json"
```

### データベース設定

#### PostgreSQL

```yaml
database:
  default_driver: "postgres"
  connection_string: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
  timeout: "30s"
```

#### MySQL

```yaml
database:
  default_driver: "mysql"
  connection_string: "user:password@tcp(localhost:3306)/dbname"
  timeout: "30s"
```

#### SQLite

```yaml
database:
  default_driver: "sqlite3"
  connection_string: "./database.db"
  timeout: "30s"
```

### 環境変数

設定で環境変数を使用：

```yaml
database:
  connection_string: "${DATABASE_URL}"
  
# デフォルト値付き
database:
  connection_string: "${DATABASE_URL:-postgres://localhost:5432/mydb}"
```

### クエリ設定

```yaml
query:
  # 危険なクエリを許可（WHERE句なしのDELETE/UPDATE）
  execute_dangerous_query: false
  
  # デフォルト出力形式（table、json、csv）
  default_format: "table"
  
  # クエリのデフォルトタイムアウト
  timeout: "30s"
  
  # 返す最大行数
  max_rows: 1000
```

### ファイルパス

```yaml
paths:
  # .snap.sqlファイルを含むディレクトリ
  queries: "./queries"
  
  # 生成された中間ファイルのディレクトリ
  generated: "./generated"
  
  # 定数ファイル
  constants: "./constants.yaml"
  
  # デフォルトパラメータファイル
  params: "./params.json"
```

## 定数ファイル (constants.yaml)

プロジェクト全体の定数を定義：

```yaml
# テーブル名マッピング
tables:
  users: "users_v2"
  posts: "posts_archive"
  comments: "comments_new"

# 環境固有のプレフィックス
environments:
  dev: "dev_"
  staging: "staging_"
  prod: "prod_"

# 共通値
pagination:
  default_limit: 50
  max_limit: 1000

# 機能フラグ
features:
  enable_caching: true
  enable_analytics: false
```

## パラメータファイル (params.json)

開発用のデフォルトパラメータ：

```json
{
  "environment": "dev",
  "pagination": {
    "limit": 20,
    "offset": 0
  },
  "filters": {
    "active": true
  },
  "include_email": true,
  "table_suffix": "dev"
}
```

## 環境固有の設定

### 複数の設定ファイルの使用

```bash
# 開発環境
snapsql --config snapsql.dev.yaml query users.snap.sql

# 本番環境
snapsql --config snapsql.prod.yaml query users.snap.sql
```

### 環境変数

```yaml
# snapsql.yaml
database:
  connection_string: "${DATABASE_URL}"
  
query:
  execute_dangerous_query: "${ALLOW_DANGEROUS_QUERIES:-false}"
```

```bash
# 環境変数を設定
export DATABASE_URL="postgres://localhost:5432/mydb"
export ALLOW_DANGEROUS_QUERIES="true"

# クエリを実行
snapsql query users.snap.sql
```

## 高度な設定

### カスタム方言

```yaml
database:
  dialect: "postgresql"
  features:
    - "window_functions"
    - "cte"
    - "json_operators"
```

### ログ設定

```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json" # json, text
  output: "stdout" # stdout, stderr, ファイルパス
```

### パフォーマンス

```yaml
performance:
  connection_pool_size: 10
  max_idle_connections: 5
  connection_max_lifetime: "1h"
  query_timeout: "30s"
```

## 設定の検証

設定を検証：

```bash
# 設定構文をチェック
snapsql config validate

# 解決された設定を表示
snapsql config show

# データベース接続をテスト
snapsql config test-db
```

## 例

### 開発環境設定

```yaml
# snapsql.dev.yaml
name: "myapp-dev"
database:
  default_driver: "postgres"
  connection_string: "postgres://dev:dev@localhost:5432/myapp_dev"
query:
  execute_dangerous_query: true
  default_format: "table"
paths:
  queries: "./queries"
  constants: "./dev-constants.yaml"
```

### 本番環境設定

```yaml
# snapsql.prod.yaml
name: "myapp-prod"
database:
  default_driver: "postgres"
  connection_string: "${DATABASE_URL}"
query:
  execute_dangerous_query: false
  default_format: "json"
  max_rows: 10000
paths:
  queries: "./queries"
  constants: "./prod-constants.yaml"
logging:
  level: "warn"
  format: "json"
```
