# SnapSQL CLIツール設計

## 概要

SnapSQLコマンドラインツール（`snapsql`）は、SQLテンプレートの管理、コード生成、データベーススキーマ抽出のための包括的なインターフェースを提供します。複数のプログラミング言語をサポートし、SnapSQLテンプレートエンジンと統合されています。

## アーキテクチャ

### コマンド構造

```
snapsql <command> [options] [arguments]
```

### 主要コマンド

1. **generate** - 中間ファイルまたはランタイムコードの生成
2. **validate** - SQLテンプレートの検証
3. **pull** - データベーススキーマ情報の抽出
4. **init** - 新しいSnapSQLプロジェクトの初期化
5. **query** - SQLテンプレートを使用してデータベースにクエリを実行

### グローバルオプション

- `--config <file>` - 設定ファイルパス（デフォルト: `./snapsql.yaml`）
- `--verbose, -v` - 詳細出力を有効にする
- `--quiet, -q` - 静寂モードを有効にする
- `--help, -h` - ヘルプ情報を表示
- `--version` - バージョン情報を表示
- `--no-color` - カラー出力を無効にする

## コマンド仕様

### 1. generateコマンド

SQLテンプレートから中間ファイルまたは言語固有のコードを生成します。

#### 構文
```bash
snapsql generate [options]
```

#### オプション
- `-i, --input <dir>` - 入力ディレクトリ（デフォルト: `./queries`）
- `--lang <language>` - 出力言語/形式（デフォルト: `json`）
  - サポート: `json`, `go`, `typescript`, `java`, `python`
- `--package <n>` - パッケージ名（言語固有）
- `--validate` - 生成前にテンプレートを検証
- `--watch` - ファイル変更を監視して自動再生成

#### 内蔵ジェネレータ

| 言語 | 内蔵 | 出力 |
|------|------|------|
| `json` | はい | 中間JSONファイル |
| `go` | はい | Go言語コード |
| `typescript` | はい | TypeScriptコード |
| `java` | はい | Java言語コード |
| `python` | はい | Python言語コード |

#### 外部ジェネレータプラグインサポート

内蔵されていない言語については、外部実行可能ファイルを探します：
- パターン: `snapsql-gen-<language>`
- 場所: システムPATH
- インターフェース: JSON入力を持つコマンドライン

#### 例
```bash
# JSON中間ファイルを生成
snapsql generate

# Goコードを生成
snapsql generate --lang go --package queries

# 検証付きで生成
snapsql generate --lang typescript --validate

# 開発用の監視モード
snapsql generate --watch
```

### 2. validateコマンド

SQLテンプレートの構文と構造を検証します。

#### 構文
```bash
snapsql validate [options] [files...]
```

#### オプション
- `-i, --input <dir>` - 入力ディレクトリ（デフォルト: `./queries`）
- `--files <files...>` - 検証する特定のファイル
- `--strict` - 厳密な検証モードを有効にする
- `--format <format>` - 出力形式（`text`, `json`）（デフォルト: `text`）

#### 検証ルール
- SnapSQL構文検証
- パラメータ型チェック
- テンプレート構造検証
- 定数との相互参照検証

#### 例
```bash
# すべてのテンプレートを検証
snapsql validate

# 特定のファイルを検証
snapsql validate queries/users.snap.sql queries/posts.snap.md

# JSON出力での厳密モード
snapsql validate --strict --format json
```

### 3. pullコマンド

データベーススキーマ情報を抽出してYAMLファイルに保存します。

#### 構文
```bash
snapsql pull [options]
```

#### オプション
- `--db <connection>` - データベース接続文字列
- `--env <environment>` - 設定からの環境名
- `-o, --output <file>` - 出力ファイル（デフォルト: `./schema.yaml`）
- `--tables <pattern>` - 含めるテーブルパターン（ワイルドカード対応）
- `--exclude <pattern>` - 除外するテーブルパターン
- `--include-views` - データベースビューを含める
- `--include-indexes` - インデックス情報を含める

#### サポートデータベース
- PostgreSQL
- MySQL
- SQLite

#### 例
```bash
# 環境設定から抽出
snapsql pull --env development

# 直接接続で抽出
snapsql pull --db "postgres://user:pass@localhost/mydb"

# ビュー付きで特定テーブルを抽出
snapsql pull --env production --tables "user*,post*" --include-views

# システムテーブルを除外
snapsql pull --env development --exclude "pg_*,information_schema*"
```

### 4. initコマンド

ディレクトリ構造とサンプルファイルで新しいSnapSQLプロジェクトを初期化します。

#### 構文
```bash
snapsql init
```

#### 生成される構造
```
./
├── snapsql.yaml           # 設定ファイル
├── queries/               # SQLテンプレートディレクトリ
│   └── users.snap.sql     # サンプルSQLテンプレート
├── constants/             # 定数ディレクトリ
│   └── database.yaml      # サンプル定数
└── generated/             # 出力ディレクトリ（初回生成時に作成）
```

#### 生成されるファイル
- **snapsql.yaml**: 例付きの完全な設定
- **queries/users.snap.sql**: サンプルSnapSQLテンプレート
- **constants/database.yaml**: サンプル定数ファイル

#### 例
```bash
# 現在のディレクトリで初期化
snapsql init
```

### 5. queryコマンド

SQLテンプレートを使用してデータベースにクエリを実行し、結果を返します。

#### 構文
```bash
snapsql query [options] <template-file>
```

#### オプション
- `-p, --params <file>` - パラメータファイル（JSON/YAML形式）
- `--param <key=value>` - 個別のパラメータ指定（複数指定可能）
- `--const <file>` - 定数定義ファイル（複数指定可能）
- `--db <connection>` - データベース接続文字列
- `--env <environment>` - 設定からの環境名
- `--format <format>` - 出力形式（`table`, `json`, `csv`, `yaml`）（デフォルト: `table`）
- `--output <file>` - 結果を保存するファイル（指定しない場合は標準出力）
- `--timeout <seconds>` - クエリタイムアウト（秒）（デフォルト: `30`）
- `--explain` - クエリの実行計画を表示
- `--explain-analyze` - 実際に実行しながら詳細な実行計画を表示（`--explain`を含む）
- `--limit <n>` - 結果の行数を制限
- `--offset <n>` - 結果の開始位置を指定
- `--execute-dangerous-query` - WHERE句のないDELETE/UPDATEクエリを実行（危険！）
- `--dry-run` - 実際にクエリを実行せず、生成されたSQLを表示

#### パラメータ指定
- JSONファイル: `--params params.json`
- YAMLファイル: `--params params.yaml`
- コマンドライン: `--param user_id=123 --param include_email=true`
- 複合パラメータ: `--param 'filters={"active":true,"department":"sales"}'`

#### 出力形式
- `table`: 整形されたテーブル（デフォルト、ターミナル向け）
- `json`: JSON形式（プログラム処理向け）
- `csv`: CSV形式（スプレッドシート向け）
- `yaml`: YAML形式（可読性向け）

#### サポートデータベース
- PostgreSQL
- MySQL
- SQLite

#### 例
```bash
# 基本的なクエリ実行
snapsql query queries/get-users.snap.sql --param user_id=123

# パラメータファイルを使用
snapsql query queries/complex-report.snap.md --params params.json

# 環境設定からのDB接続
snapsql query queries/analytics.snap.sql --env production --format json

# 結果をファイルに保存
snapsql query queries/export-data.snap.sql --format csv --output data.csv

# 複合パラメータ
snapsql query queries/update-user.snap.sql --param 'user={"id":123,"name":"New Name"}'

# 生成されるSQLの確認（実行なし）
snapsql query queries/complex-query.snap.sql --params test-params.yaml --dry-run

# 実行計画の表示
snapsql query queries/performance-critical.snap.sql --explain

# 詳細な実行計画の表示
snapsql query queries/performance-critical.snap.sql --explain-analyze

# 結果の制限
snapsql query queries/large-result.snap.sql --limit 100

# オフセット付きの結果取得
snapsql query queries/paginated-result.snap.sql --limit 20 --offset 40

# 危険なクエリの実行（WHERE句なしのDELETE/UPDATE）
snapsql query queries/cleanup.snap.sql --execute-dangerous-query

# 危険なクエリの実行（WHERE句なしのDELETE/UPDATE）
snapsql query queries/cleanup.snap.sql --execute-dangerous-query
```

## 設定ファイル

### ファイル場所
- デフォルト: `./snapsql.yaml`
- 上書き: `--config <file>`

### 設定構造

```yaml
# SQLダイアレクト
# SQLダイアレクト
dialect: "postgres"  # postgres, mysql, sqlite

# データベース接続は通常 `.tbls.yaml` か、接続が必要なコマンドでは `--db` フラグを経由して指定します。

# 定数定義ファイル
constant_files:
  - "./constants/database.yaml"
  - "./constants/tables.yaml"

# 生成設定
generation:
  default_lang: "json"
  validate: true
  
  # 言語固有設定
  json:
    output: "./generated"
    pretty: true
    include_metadata: true
  
  go:
    output: "./internal/queries"
    package: "queries"
  
  typescript:
    output: "./src/generated"
    types: true
  
  java:
    output: "./src/main/java"
    package: "com.example.queries"
  
  python:
    output: "./src/queries"
    package: "queries"

# 検証設定
validation:
  strict: false
  rules:
    - "no-dynamic-table-names"
    - "require-parameter-types"

# クエリ実行設定
query:
  default_format: "table"
  default_environment: "development"
  timeout: 30
  max_rows: 1000
  limit: 0
  offset: 0
  execute_dangerous_query: false
  execute_dangerous_query: false
```

## ファイル処理

### サポートファイル形式

1. **`.snap.sql`** - 純粋なSQLテンプレート
2. **`.snap.md`** - Markdownベースのリテラルプログラミングファイル

### 処理パイプライン

1. **発見**: すべての`.snap.sql`と`.snap.md`ファイルを検索
2. **解析**: SnapSQL構文を解析してメタデータを抽出
3. **検証**: 構文と構造を検証
4. **定数解決**: `/*@ */`定数を解決
5. **生成**: 対象言語のコードを生成
6. **出力**: 生成されたファイルを書き込み

### テンプレート処理

#### SnapSQL構文サポート
- `/*# if condition */` - 条件ブロック
- `/*# for variable : list */` - ループブロック
- `/*# end */` - 終了ブロック
- `/*= variable */` - 変数置換
- `/*@ constant */` - 定数展開

#### 定数ファイル処理
- YAML形式の定数定義
- 階層的定数アクセス
- 複数定数ファイルサポート

## エラーハンドリング

### 終了コード
- `0` - 成功
- `1` - 一般的なエラー
- `2` - 無効な引数
- `3` - ファイルI/Oエラー
- `4` - 検証エラー
- `5` - データベース接続エラー

### エラーメッセージ
- カラー出力（`--no-color`でない限り）
- 詳細なエラー説明
- ファイルと行番号情報
- 一般的な問題の提案

### 詳細モード
- 進行状況情報
- ファイル処理詳細
- 生成統計
- パフォーマンスメトリクス

## プラグインシステム

### 外部ジェネレータ発見
- 実行可能ファイル名パターン: `snapsql-gen-<language>`
- 検索場所: システムPATH
- 自動発見と実行

### プラグインインターフェース
```bash
snapsql-gen-<lang> [options] <intermediate-json-file>
```

#### 標準オプション
- `--output-dir <path>` - 出力ディレクトリ
- `--package <n>` - パッケージ/名前空間名
- `--schema-file <path>` - データベーススキーマファイル
- `--constants <files>` - 定数定義ファイル
- `--config <file>` - SnapSQL設定ファイル
- `--verbose` - 詳細出力を有効にする

### プラグイン通信
- **入力**: ファイル経由の中間JSON
- **出力**: 生成されたコードファイル
- **ログ**: メッセージ用のstdout/stderr
- **終了コード**: 標準終了コード規約

## 開発ワークフロー

### 典型的な使用パターン

#### 1. 新規プロジェクトセットアップ
```bash
# プロジェクト初期化
snapsql init

# 設定編集
vim snapsql.yaml

# SQLテンプレート作成
vim queries/my-query.snap.sql

# コード生成
snapsql generate --lang go
```

#### 2. 既存データベース統合
```bash
# スキーマ抽出
snapsql pull --env production

# スキーマに基づいてテンプレート作成
# queries/*.snap.sqlを編集

# テンプレート検証
snapsql validate

# コード生成
snapsql generate --lang typescript
```

#### 3. 開発モード
```bash
# 継続的生成のための監視モード
snapsql generate --watch --lang go

# 別のターミナルでテンプレート編集
vim queries/users.snap.sql
# ファイルが自動的に再生成される
```

#### 4. CI/CD統合
```bash
# CIでの検証
snapsql validate --strict --format json

# デプロイ用のコード生成
snapsql generate --lang java --validate
```

#### 5. データベースクエリ実行
```bash
# 基本的なクエリ実行
snapsql query queries/get-users.snap.sql --param user_id=123

# 複雑なレポート生成
snapsql query queries/monthly-report.snap.md --params report-params.yaml --format csv --output report.csv

# データ更新操作
snapsql query queries/update-status.snap.sql --param status=active --transaction
```

## パフォーマンス考慮事項

### ファイル処理
- 独立したテンプレートの並列処理
- 増分生成（変更されたファイルのみ）
- 大きなテンプレートセットでの効率的なメモリ使用

### 監視モード
- ファイルシステムイベントベースの監視
- デバウンスされた再生成
- 変更されたファイルの選択的処理

### データベース操作
- スキーマ抽出のための接続プーリング
- 効率的なメタデータクエリ
- 遅い接続のためのタイムアウト処理

### クエリ実行
- 接続プーリングによる効率的なDB接続
- パラメータ化クエリによるSQLインジェクション防止
- 大きな結果セットのストリーミング処理
- タイムアウト処理による長時間実行クエリの制御

## セキュリティ考慮事項

### データベース接続
- 安全な認証情報処理
- 接続文字列検証
- タイムアウトと再試行メカニズム

### ファイル操作
- パストラバーサル保護
- アトミック操作による安全なファイル書き込み
- 権限検証

### コード生成
- SQLインジェクション防止
- 安全なテンプレート処理
- 出力サニタイゼーション

### クエリ実行
- パラメータ化クエリによるSQLインジェクション防止
- 機密データの安全な処理
- 権限制限によるデータアクセス制御
- トランザクション分離レベルの適切な設定

## テスト戦略

### 単体テスト
- コマンド解析と検証
- 設定ファイル処理
- テンプレート解析と生成
- エラーハンドリングシナリオ

### 統合テスト
- エンドツーエンドコマンド実行
- データベーススキーマ抽出
- 多言語コード生成
- プラグインシステム統合
- クエリ実行と結果処理

### パフォーマンステスト
- 大きなテンプレートセット処理
- メモリ使用量最適化
- 生成速度ベンチマーク
- クエリ実行パフォーマンス
