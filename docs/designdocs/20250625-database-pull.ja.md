# データベースpull機能設計ドキュメント

## 概要

データベースpull機能は、既存のデータベース（PostgreSQL、MySQL、SQLite）からスキーマ情報を抽出し、YAMLスキーマファイルを生成する機能です。この機能により、SQLテンプレート生成と検証のための正確な型情報を提供し、開発ワークフローをサポートします。

## 処理フロー

1. **データベース接続**
   - snapsql.yamlまたはコマンドライン引数から接続情報を取得
   - データベースタイプに応じた適切なドライバーを選択
   - 接続テストと基本情報（バージョン、文字セット）の取得

2. **スキーマ情報の抽出**
   - データベースタイプに応じたシステムテーブルへのアクセス
   - テーブル一覧の取得（フィルタリング適用）
   - 各テーブルのカラム情報取得
   - 制約情報（主キー、外部キー、ユニーク制約）の取得
   - インデックス情報の取得
   - コメント情報の取得（サポートされている場合）

3. **型マッピング処理**
   - データベース固有の型をSnapSQL標準型にマッピング
   - カスタム型の処理（string型へのフォールバック）
   - 配列型やJSON型などの特殊型の処理

4. **YAML生成**
   - 出力形式の決定（単一ファイル、テーブル別、スキーマ別）
   - メタデータの付加（抽出時刻、データベース情報）
   - 指定された出力パスへのファイル生成

## データベース固有の処理

### PostgreSQL
1. information_schemaからのテーブル情報取得
2. pg_catalogからのインデックスと制約情報取得
3. pg_descriptionからのコメント情報取得
4. PostgreSQL固有型（配列、JSON）の処理

### MySQL
1. information_schemaからのテーブル、カラム情報取得
2. KEY_COLUMN_USAGEからの制約情報取得
3. STATISTICSからのインデックス情報取得
4. MySQL固有型とストレージエンジンの処理

### SQLite
1. sqlite_masterからのテーブル情報取得
2. PRAGMA table_info()でのカラム情報取得
3. PRAGMA foreign_key_list()での外部キー情報取得
4. PRAGMA index_list()でのインデックス情報取得

## 出力形式

### 単一ファイル形式
```yaml
database_info:
  type: postgresql
  version: "14.2"
  name: myapp_production
  charset: UTF8

extracted_at: 2025-06-25T23:00:00Z

schemas:
  - name: public
    tables:
      - name: users
        columns:
          - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
          - {name: email, type: "character varying(255)", snapsql_type: string, nullable: false}
          - {name: created_at, type: "timestamp with time zone", snapsql_type: timestamp, nullable: false}
```

### テーブル別形式
```yaml
# .snapsql/schema/public/users.yaml
table:
  name: users
  schema: public
  columns:
    - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
    - {name: email, type: "character varying(255)", snapsql_type: string, nullable: false}
  constraints:
    - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
    - {name: users_email_unique, type: UNIQUE, columns: [email]}

metadata:
  extracted_at: 2025-06-25T23:00:00Z
  database_info:
    type: postgresql
    version: "14.2"
```

## 型マッピング

### 共通型マッピング
| データベース型 | SnapSQL型 |
|--------------|-----------|
| 文字列型 | string |
| 整数型 | int |
| 浮動小数点型 | float |
| 真偽値型 | bool |
| タイムスタンプ型 | timestamp（エイリアス: datetime, date, time） |
| JSON型 | json |
| バイナリ型 | binary |

### データベース固有の型マッピング
- PostgreSQL: 配列型、カスタム型
- MySQL: ENUM型、SET型
- SQLite: 動的型システム

## エラーハンドリング

1. **接続エラー**
   - 接続タイムアウト
   - 認証エラー
   - ネットワークエラー

2. **抽出エラー**
   - アクセス権限エラー
   - システムテーブルアクセスエラー
   - メタデータ取得エラー

3. **型マッピングエラー**
   - 未知の型の処理
   - カスタム型の処理
   - 型変換エラー

## CLIインターフェース

```bash
# 基本的な使用方法
snapsql pull --database development

# カスタム接続
snapsql pull --url "postgres://user:pass@localhost/myapp"

# 出力形式指定
snapsql pull --database development --format per_table

# フィルタリング
snapsql pull --database production --schemas public,auth --tables users,posts
```

## 設定ファイル

```yaml
databases:
  development:
    driver: postgres
    connection: "postgres://user:pass@localhost/myapp_dev"

pull:
  output_format: per_table
  output_path: ".snapsql/schema"
  include_views: true
  include_indexes: true
  include_schemas: ["public", "auth"]
  exclude_tables: ["migrations", "temp_*"]
```

## セキュリティ考慮事項

1. **データベースアクセス**
   - 読み取り専用接続の使用
   - 最小権限原則の適用
   - システムテーブルへの制限付きアクセス

2. **認証情報管理**
   - 環境変数からの認証情報読み取り
   - 接続文字列の安全な処理
   - メモリ内での認証情報の保護

3. **出力ファイルの保護**
   - 機密情報のフィルタリング
   - 適切なファイルパーミッションの設定
   - 出力ディレクトリの制限
