# データベースpull機能設計ドキュメント

## 概要

データベースpull機能は、既存のデータベース（PostgreSQL、MySQL、SQLite）からスキーマ情報を抽出し、YAMLスキーマファイルを生成する機能です。この機能により、SQLテンプレート生成と検証のための正確な型情報を提供し、開発ワークフローをサポートします。

## 要件

### 機能要件

1. **データベース接続サポート**
   - PostgreSQL: 標準接続文字列による接続
   - MySQL: 標準接続文字列による接続
   - SQLite: ローカルデータベースファイルへの接続
   - snapsql.yaml設定からの接続パラメータサポート

2. **スキーマ情報抽出**
   - テーブル名とスキーマ
   - カラム情報（名前、型、NULL許可、デフォルト値）
   - 主キー制約
   - 外部キー関係
   - ユニーク制約
   - インデックス情報
   - テーブルコメントとカラムコメント

3. **YAMLスキーマ生成**
   - 抽出したスキーマ情報の構造化YAMLファイル生成
   - 複数の出力形式サポート（単一ファイル vs テーブル別ファイル）
   - 抽出タイムスタンプやデータベースバージョンなどのメタデータ含有

4. **型マッピング**
   - データベース固有型からSnapSQL標準型へのマッピング
   - データベース方言の違いの適切な処理
   - 参照用の元の型情報保持

### 非機能要件

1. **パフォーマンス**
   - システムテーブル/information_schemaを使用した効率的なスキーマ抽出
   - 対象データベースのパフォーマンスへの最小限の影響
   - 数百テーブルを持つ大規模データベースのサポート

2. **セキュリティ**
   - 読み取り専用データベースアクセス
   - 安全な認証情報処理
   - 対象データベースの変更なし

3. **信頼性**
   - 接続失敗の適切な処理
   - 部分抽出サポート（個別テーブルエラー時の継続処理）
   - 包括的なエラー報告

## アーキテクチャ

### パッケージ構造

```
pull/
├── pull.go              // メインpullオーケストレーター
├── connector.go         // データベース接続管理
├── extractor.go         // スキーマ抽出インターフェース
├── postgresql.go        // PostgreSQL固有の抽出
├── mysql.go            // MySQL固有の抽出
├── sqlite.go           // SQLite固有の抽出
├── schema.go           // スキーマデータ構造
├── yaml_generator.go   // YAML出力生成
└── type_mapper.go      // データベース型マッピング
```

### コアコンポーネント

#### 1. Pullオーケストレーター (`pull.go`)

全体のpullプロセスを調整するメインエントリーポイント：

```go
type PullConfig struct {
    DatabaseURL    string
    DatabaseType   string
    OutputPath     string
    OutputFormat   OutputFormat
    IncludeTables  []string
    ExcludeTables  []string
    IncludeViews   bool
    IncludeIndexes bool
}

type PullResult struct {
    Schemas       []DatabaseSchema
    ExtractedAt   time.Time
    DatabaseInfo  DatabaseInfo
    Errors        []error
}

func Pull(config PullConfig) (*PullResult, error)
```

#### 2. データベースコネクター (`connector.go`)

適切なリソースクリーンアップを伴うデータベース接続管理：

```go
type Connector interface {
    Connect(url string) (*sql.DB, error)
    GetDatabaseInfo(db *sql.DB) (DatabaseInfo, error)
    Close() error
}

type DatabaseInfo struct {
    Type     string
    Version  string
    Name     string
    Charset  string
}
```

#### 3. スキーマエクストラクター (`extractor.go`)

データベース非依存のスキーマ抽出インターフェース：

```go
type Extractor interface {
    ExtractSchemas(db *sql.DB, config ExtractConfig) ([]DatabaseSchema, error)
    ExtractTables(db *sql.DB, schemaName string) ([]TableSchema, error)
    ExtractColumns(db *sql.DB, tableName string) ([]ColumnSchema, error)
    ExtractConstraints(db *sql.DB, tableName string) ([]ConstraintSchema, error)
    ExtractIndexes(db *sql.DB, tableName string) ([]IndexSchema, error)
}

type ExtractConfig struct {
    IncludeTables  []string
    ExcludeTables  []string
    IncludeViews   bool
    IncludeIndexes bool
}
```

#### 4. スキーマデータ構造 (`schema.go`)

統一されたスキーマ表現：

```go
type DatabaseSchema struct {
    Name        string        `yaml:"name"`
    Tables      []TableSchema `yaml:"tables"`
    Views       []ViewSchema  `yaml:"views,omitempty"`
    ExtractedAt time.Time     `yaml:"extracted_at"`
    DatabaseInfo DatabaseInfo `yaml:"database_info"`
}

type TableSchema struct {
    Name        string           `yaml:"name"`
    Schema      string           `yaml:"schema,omitempty"`
    Columns     []ColumnSchema   `yaml:"columns"`
    Constraints []ConstraintSchema `yaml:"constraints,omitempty"`
    Indexes     []IndexSchema    `yaml:"indexes,omitempty"`
    Comment     string           `yaml:"comment,omitempty"`
}

type ColumnSchema struct {
    Name         string `yaml:"name"`
    Type         string `yaml:"type"`
    SnapSQLType  string `yaml:"snapsql_type"`
    Nullable     bool   `yaml:"nullable"`
    DefaultValue string `yaml:"default_value,omitempty"`
    Comment      string `yaml:"comment,omitempty"`
    IsPrimaryKey bool   `yaml:"is_primary_key,omitempty"`
}

type ConstraintSchema struct {
    Name           string   `yaml:"name"`
    Type           string   `yaml:"type"` // PRIMARY_KEY, FOREIGN_KEY, UNIQUE, CHECK
    Columns        []string `yaml:"columns"`
    ReferencedTable string  `yaml:"referenced_table,omitempty"`
    ReferencedColumns []string `yaml:"referenced_columns,omitempty"`
    Definition     string   `yaml:"definition,omitempty"`
}

type IndexSchema struct {
    Name     string   `yaml:"name"`
    Columns  []string `yaml:"columns"`
    IsUnique bool     `yaml:"is_unique"`
    Type     string   `yaml:"type,omitempty"`
}

type ViewSchema struct {
    Name       string `yaml:"name"`
    Schema     string `yaml:"schema,omitempty"`
    Definition string `yaml:"definition"`
    Comment    string `yaml:"comment,omitempty"`
}
```

#### 5. データベース固有エクストラクター

各データベースは独自のエクストラクター実装を持ちます：

**PostgreSQLエクストラクター (`postgresql.go`)**
- `information_schema`と`pg_catalog`システムテーブルを使用
- PostgreSQL固有型（配列、JSON、カスタム型）の処理
- スキーマ修飾テーブル名の抽出

**MySQLエクストラクター (`mysql.go`)**
- `information_schema`テーブルを使用
- MySQL固有型とストレージエンジンの処理
- MySQLとMariaDBの両方のバリアントをサポート

**SQLiteエクストラクター (`sqlite.go`)**
- `sqlite_master`と`PRAGMA`文を使用
- SQLiteの動的型システムの処理
- テーブルとインデックス情報の抽出

#### 6. 型マッパー (`type_mapper.go`)

データベース固有型からSnapSQL標準型へのマッピング：

```go
type TypeMapper interface {
    MapType(dbType string) string
    GetSnapSQLType(dbType string) string
}

// SnapSQL標準型
const (
    TypeString   = "string"
    TypeInt      = "int"
    TypeFloat    = "float"
    TypeBool     = "bool"
    TypeDate     = "date"
    TypeTime     = "time"
    TypeDateTime = "datetime"
    TypeJSON     = "json"
    TypeArray    = "array"
    TypeBinary   = "binary"
)
```

#### 7. YAML生成器 (`yaml_generator.go`)

様々な形式でのYAML出力生成：

```go
type OutputFormat string

const (
    OutputSingleFile OutputFormat = "single"
    OutputPerTable   OutputFormat = "per_table"
    OutputPerSchema  OutputFormat = "per_schema"
)

type YAMLGenerator struct {
    Format OutputFormat
    Pretty bool
}

func (g *YAMLGenerator) Generate(schemas []DatabaseSchema, outputPath string) error
```

## データベース固有実装詳細

### PostgreSQL

**使用するシステムテーブル:**
- `information_schema.tables` - テーブル情報
- `information_schema.columns` - カラム情報
- `information_schema.table_constraints` - 制約情報
- `information_schema.key_column_usage` - キー関係
- `pg_catalog.pg_indexes` - インデックス情報
- `pg_catalog.pg_description` - コメント

**主要クエリ:**
```sql
-- テーブル
SELECT schemaname, tablename, tableowner 
FROM pg_tables 
WHERE schemaname NOT IN ('information_schema', 'pg_catalog');

-- 型付きカラム
SELECT c.column_name, c.data_type, c.is_nullable, c.column_default,
       c.character_maximum_length, c.numeric_precision, c.numeric_scale
FROM information_schema.columns c
WHERE c.table_schema = $1 AND c.table_name = $2
ORDER BY c.ordinal_position;
```

### MySQL

**使用するシステムテーブル:**
- `information_schema.TABLES` - テーブル情報
- `information_schema.COLUMNS` - カラム情報
- `information_schema.TABLE_CONSTRAINTS` - 制約情報
- `information_schema.KEY_COLUMN_USAGE` - キー関係
- `information_schema.STATISTICS` - インデックス情報

**主要クエリ:**
```sql
-- テーブル
SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE, TABLE_COMMENT
FROM information_schema.TABLES
WHERE TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys');

-- カラム
SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT,
       CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE,
       COLUMN_KEY, EXTRA, COLUMN_COMMENT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
ORDER BY ORDINAL_POSITION;
```

### SQLite

**使用するシステムテーブル:**
- `sqlite_master` - メインシステムテーブル
- `PRAGMA table_info()` - カラム情報
- `PRAGMA foreign_key_list()` - 外部キー情報
- `PRAGMA index_list()` - インデックス情報

**主要クエリ:**
```sql
-- テーブル
SELECT name, type, sql FROM sqlite_master 
WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%';

-- カラム（PRAGMA経由）
PRAGMA table_info(table_name);

-- 外部キー（PRAGMA経由）
PRAGMA foreign_key_list(table_name);
```

## 型マッピング戦略

### PostgreSQL型マッピング

| PostgreSQL型 | SnapSQL型 | 備考 |
|-------------|-----------|------|
| `varchar`, `text`, `char` | `string` | |
| `integer`, `bigint`, `smallint` | `int` | |
| `numeric`, `decimal`, `real`, `double precision` | `float` | |
| `boolean` | `bool` | |
| `date` | `date` | |
| `time`, `timetz` | `time` | |
| `timestamp`, `timestamptz` | `datetime` | |
| `json`, `jsonb` | `json` | |
| `array types` | `array` | 要素型付き |
| `bytea` | `binary` | |

### MySQL型マッピング

| MySQL型 | SnapSQL型 | 備考 |
|---------|-----------|------|
| `varchar`, `text`, `char` | `string` | |
| `int`, `bigint`, `smallint`, `tinyint` | `int` | |
| `decimal`, `numeric`, `float`, `double` | `float` | |
| `boolean`, `tinyint(1)` | `bool` | |
| `date` | `date` | |
| `time` | `time` | |
| `datetime`, `timestamp` | `datetime` | |
| `json` | `json` | |
| `blob`, `binary`, `varbinary` | `binary` | |

### SQLite型マッピング

| SQLite型 | SnapSQL型 | 備考 |
|----------|-----------|------|
| `TEXT` | `string` | |
| `INTEGER` | `int` | |
| `REAL` | `float` | |
| `BLOB` | `binary` | |
| 動的型 | `string` | デフォルトフォールバック |

## 出力形式例

### 単一ファイル形式

```yaml
# .snapsql/schema/database_schema.yaml
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
          - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false, default_value: "now()"}
        constraints:
          - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
          - {name: users_email_unique, type: UNIQUE, columns: [email]}
        indexes:
          - {name: idx_users_email, columns: [email], is_unique: true}
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
    - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false, default_value: "now()"}
    - {name: updated_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: true}
  constraints:
    - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
    - {name: users_email_unique, type: UNIQUE, columns: [email]}
  indexes:
    - {name: idx_users_email, columns: [email], is_unique: true}
    - {name: idx_users_created_at, columns: [created_at], is_unique: false}

metadata:
  extracted_at: 2025-06-25T23:00:00Z
  database_info:
    type: postgresql
    version: "14.2"
```

### スキーマ別形式

```yaml
# .snapsql/schema/public.yaml
schema:
  name: public
  tables:
    - name: users
      columns:
        - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
        - {name: email, type: "character varying(255)", snapsql_type: string, nullable: false}
        - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false}
      constraints:
        - {name: users_pkey, type: PRIMARY_KEY, columns: [id]}
        - {name: users_email_unique, type: UNIQUE, columns: [email]}
    - name: posts
      columns:
        - {name: id, type: integer, snapsql_type: int, nullable: false, is_primary_key: true}
        - {name: title, type: "character varying(255)", snapsql_type: string, nullable: false}
        - {name: user_id, type: integer, snapsql_type: int, nullable: false}
        - {name: created_at, type: "timestamp with time zone", snapsql_type: datetime, nullable: false}
      constraints:
        - {name: posts_pkey, type: PRIMARY_KEY, columns: [id]}
        - {name: posts_user_id_fkey, type: FOREIGN_KEY, columns: [user_id], referenced_table: users, referenced_columns: [id]}

metadata:
  extracted_at: 2025-06-25T23:00:00Z
  database_info:
    type: postgresql
    version: "14.2"
```

### ディレクトリ構造

データベーススキーマサポート付きのスキーマファイル推奨ディレクトリ構造：

```
project/
├── .snapsql/
│   └── schema/
│       ├── public/           # PostgreSQL/MySQLスキーマ
│       │   ├── users.yaml
│       │   ├── posts.yaml
│       │   └── comments.yaml
│       ├── auth/             # 別のスキーマ
│       │   ├── sessions.yaml
│       │   └── permissions.yaml
│       └── global/           # SQLiteまたはスキーマレスデータベース
│           ├── users.yaml
│           └── posts.yaml
├── queries/
│   ├── users.snap.sql
│   └── posts.snap.sql
└── snapsql.yaml
```

**スキーマディレクトリマッピング:**
- **PostgreSQL/MySQL**: 実際のスキーマ名を使用（`public`、`auth`、`inventory`など）
- **SQLite**: デフォルトスキーマディレクトリとして`global`を使用
- **スキーマレスデータベース**: フォールバックスキーマディレクトリとして`global`を使用

**スキーマ対応構造の利点:**
- **データベース忠実性**: 実際のデータベーススキーマ構成を反映
- **名前空間の明確性**: データベーススキーマによるテーブルの明確な分離
- **マルチスキーマサポート**: 複数スキーマを持つ複雑なデータベースの簡単な管理
- **競合解決**: 異なるスキーマ内の同名テーブルの処理
- **マイグレーション対応**: スキーマ固有の変更追跡が容易

## エラーハンドリング戦略

### 接続エラー
- 指数バックオフによるリトライロジック
- 一般的な接続問題の明確なエラーメッセージ
- 接続タイムアウト設定のサポート

### 抽出エラー
- 個別テーブル抽出失敗時の他テーブル処理継続
- 最終的な全エラーの収集と報告
- 詳細なエラーコンテキスト提供（テーブル名、クエリなど）

### 型マッピングエラー
- 未知のデータベース型に対するstring型へのフォールバック
- マップされていない型の警告ログ
- カスタム型マッピング設定の許可

## 設定統合

### snapsql.yaml設定

```yaml
databases:
  development:
    driver: postgres
    connection: "postgres://user:pass@localhost/myapp_dev"
  production:
    driver: postgres
    connection: "postgres://user:pass@prod.example.com/myapp"

pull:
  output_format: per_table     # single, per_table, per_schema
  output_path: ".snapsql/schema"
  schema_aware: true           # スキーマ対応ディレクトリ構造を有効化
  flow_style: true             # カラム、制約、インデックスにフロースタイルを使用
  include_views: true
  include_indexes: true
  include_schemas: ["public", "auth"]      # オプションスキーマフィルター
  exclude_schemas: ["information_schema"]  # オプションスキーマ除外
  include_tables: ["users", "posts", "comments"]  # オプションテーブルフィルター
  exclude_tables: ["migrations", "temp_*"]        # オプションテーブルフィルター
```

## CLI統合

pull機能はメインのSnapSQL CLIに統合されます：

```bash
# 設定されたデータベースからpull（デフォルトでスキーマ対応）
snapsql pull --database development

# カスタム接続でpull
snapsql pull --url "postgres://user:pass@localhost/myapp"

# 特定スキーマのpull
snapsql pull --database production --schemas public,auth

# 特定スキーマの特定テーブルのpull
snapsql pull --database production --schema public --tables users,posts,comments

# 特定出力パスへのpull（デフォルトは.snapsql/schema）
snapsql pull --database development --output .snapsql/custom_schema

# 異なる形式でのpull
snapsql pull --database development --format per_schema

# スキーマ対応構造を無効化（フラット構造）
snapsql pull --database development --no-schema-aware
```

## テスト戦略

### 単体テスト
- モックデータベースを使用した各エクストラクター実装のテスト
- サポートされる全データベース型の型マッピングテスト
- 様々なスキーマ設定でのYAML生成テスト
- エラーハンドリングシナリオのテスト

### 統合テスト
- PostgreSQLとMySQLテスト用のTestContainers使用
- SQLiteテスト用のインメモリSQLiteデータベース使用
- 様々な複雑さの実際のデータベーススキーマでのテスト
- 大規模スキーマ（100+テーブル）でのパフォーマンステスト

### テストデータ
- 各データベース型の代表的なテストスキーマ作成
- エッジケース（異常な型、複雑な制約など）の含有
- シンプルと複雑両方のデータベース構造でのテスト

## パフォーマンス考慮事項

### クエリ最適化
- ラウンドトリップ削減のための可能な限りのバッチクエリ使用
- 複数スキーマ抽出のための接続プーリング実装
- 繰り返しクエリでの準備済み文使用

### メモリ管理
- 全てをメモリにロードする代わりの大規模結果セットのストリーミング
- 多数テーブルを持つデータベースでのページネーション実装
- スキーマ表現での効率的なデータ構造使用

### キャッシュ
- 繰り返しクエリ回避のためのデータベースメタデータキャッシュ
- 増分更新サポート（変更されたテーブルのみ抽出）
- 変更検出のためのスキーマ比較実装

## セキュリティ考慮事項

### データベースアクセス
- 読み取り専用データベース接続の使用
- 限定されたデータベースユーザー権限のサポート
- 対象データベース構造やデータの変更なし

### 認証情報管理
- 接続文字列での環境変数展開サポート
- 外部認証情報管理システムとの統合
- メモリ内接続文字列の安全な処理

### 出力セキュリティ
- 生成されたYAMLでの機密情報のサニタイズ
- 機密テーブル/カラムの除外サポート
- 出力でのスキーマ名の難読化オプション

## 将来の拡張

### 高度な機能
- データベースビューとマテリアライズドビューのサポート
- ストアドプロシージャと関数の抽出
- データベーストリガーとイベントのサポート
- カスタム型定義とドメインのサポート

### 統合機能
- スキーママイグレーションツールとの統合
- スキーマバージョニングと変更追跡のサポート
- ドキュメント生成ツールとの統合
- スキーマ検証とリンティングのサポート

### パフォーマンス機能
- 複数スキーマの並列抽出
- データベース変更ログベースの増分抽出
- 大規模スキーマファイルの圧縮
- スキーマキャッシュと永続化のサポート

## 実装フェーズ

### フェーズ1: コアインフラストラクチャ
- 基本パッケージ構造の実装
- データベースコネクターインターフェースの作成
- PostgreSQLエクストラクターの実装
- 基本YAML生成

### フェーズ2: マルチデータベースサポート
- MySQLエクストラクターの実装
- SQLiteエクストラクターの実装
- 型マッピングシステムの追加
- 包括的なエラーハンドリング

### フェーズ3: 高度な機能
- 複数出力形式
- 設定統合
- CLIコマンド実装
- パフォーマンス最適化

### フェーズ4: テストと仕上げ
- 包括的なテストスイート
- パフォーマンステスト
- ドキュメントと例
- セキュリティレビュー

## 依存関係

### 外部ライブラリ
- データベースドライバー: `lib/pq` (PostgreSQL)、`go-sql-driver/mysql` (MySQL)、`modernc.org/sqlite` (SQLite)
- YAML処理: `github.com/goccy/go-yaml` (既存コードベースとの一貫性)
- 設定: 既存のsnapsql.yaml処理との統合
- テスト: 統合テスト用のTestContainers

### 内部依存関係
- 設定システム（snapsql.yaml解析）
- CLIフレームワーク（Kongベースのコマンド構造）
- エラーハンドリングパターン（センチネルエラー）
- ログと出力フォーマット

この設計は、既存のSnapSQLアーキテクチャとコーディング標準との一貫性を保ちながら、データベースpull機能を実装するための包括的な基盤を提供します。
