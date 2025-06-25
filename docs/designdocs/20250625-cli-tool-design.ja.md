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
- `--package <name>` - パッケージ名（言語固有）
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

## 設定ファイル

### ファイル場所
- デフォルト: `./snapsql.yaml`
- 上書き: `--config <file>`

### 設定構造

```yaml
# SQLダイアレクト
dialect: "postgres"  # postgres, mysql, sqlite

# データベース接続
databases:
  development:
    driver: "postgres"
    connection: "postgres://user:pass@localhost/dev_db"
    schema: "public"
  production:
    driver: "postgres"
    connection: "postgres://user:pass@prod-host/prod_db"
    schema: "public"

# 定数定義ファイル
constant_files:
  - "./constants/database.yaml"
  - "./constants/tables.yaml"

# スキーマ抽出設定
schema_extraction:
  include_views: false
  include_indexes: true
  table_patterns:
    include: ["*"]
    exclude: ["pg_*", "information_schema*", "sys_*"]

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
- `--package <name>` - パッケージ/名前空間名
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

### パフォーマンステスト
- 大きなテンプレートセット処理
- メモリ使用量最適化
- 生成速度ベンチマーク
