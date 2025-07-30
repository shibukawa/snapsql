# インストールガイド

このガイドでは、SnapSQLをインストールして設定する様々な方法を説明します。

## 前提条件

- Go 1.24以降（ソースからビルドする場合）
- データベースサーバー（PostgreSQL、MySQL、またはSQLite）

## インストール方法

### 1. ソースからインストール（推奨）

```bash
# 最新版をインストール
go install github.com/shibukawa/snapsql@latest

# インストールを確認
snapsql version
```

### 2. バイナリダウンロード（計画中）

ビルド済みバイナリがダウンロード可能になる予定です：

```bash
# Linux/macOS
curl -L https://github.com/shibukawa/snapsql/releases/latest/download/snapsql-linux-amd64 -o snapsql
chmod +x snapsql
sudo mv snapsql /usr/local/bin/

# Windows
# GitHubリリースページからダウンロード
```

### 3. Docker（計画中）

```bash
# Dockerで実行
docker run --rm -v $(pwd):/workspace shibukawa/snapsql:latest generate

# 便利なエイリアスを作成
alias snapsql='docker run --rm -v $(pwd):/workspace shibukawa/snapsql:latest'
```

### 4. パッケージマネージャー（計画中）

```bash
# Homebrew（macOS/Linux）
brew install shibukawa/tap/snapsql

# Chocolatey（Windows）
choco install snapsql

# Snap（Linux）
snap install snapsql
```

## データベース設定

### PostgreSQL

1. **PostgreSQLをインストール**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install postgresql postgresql-contrib
   
   # macOS
   brew install postgresql
   
   # サービスを開始
   sudo systemctl start postgresql  # Linux
   brew services start postgresql   # macOS
   ```

2. **データベースとユーザーを作成**:
   ```sql
   -- postgresユーザーとして接続
   sudo -u postgres psql
   
   -- データベースを作成
   CREATE DATABASE myapp_dev;
   
   -- ユーザーを作成
   CREATE USER myapp_user WITH PASSWORD 'myapp_password';
   
   -- 権限を付与
   GRANT ALL PRIVILEGES ON DATABASE myapp_dev TO myapp_user;
   ```

3. **接続文字列**:
   ```
   postgres://myapp_user:myapp_password@localhost:5432/myapp_dev
   ```

### MySQL

1. **MySQLをインストール**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install mysql-server
   
   # macOS
   brew install mysql
   
   # サービスを開始
   sudo systemctl start mysql  # Linux
   brew services start mysql   # macOS
   ```

2. **データベースとユーザーを作成**:
   ```sql
   -- rootとして接続
   mysql -u root -p
   
   -- データベースを作成
   CREATE DATABASE myapp_dev;
   
   -- ユーザーを作成
   CREATE USER 'myapp_user'@'localhost' IDENTIFIED BY 'myapp_password';
   
   -- 権限を付与
   GRANT ALL PRIVILEGES ON myapp_dev.* TO 'myapp_user'@'localhost';
   FLUSH PRIVILEGES;
   ```

3. **接続文字列**:
   ```
   myapp_user:myapp_password@tcp(localhost:3306)/myapp_dev
   ```

### SQLite

1. **SQLiteをインストール**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install sqlite3
   
   # macOS
   brew install sqlite
   ```

2. **データベースを作成**:
   ```bash
   # データベースファイルを作成
   sqlite3 myapp_dev.db
   ```

3. **接続文字列**:
   ```
   ./myapp_dev.db
   ```

## プロジェクト設定

### 1. 新しいプロジェクトを初期化

```bash
# 新しいプロジェクトを作成
snapsql init my-project
cd my-project
```

これにより以下の構造が作成されます：
```
my-project/
├── snapsql.yaml          # 設定ファイル
├── queries/              # SQLテンプレート
│   └── users.snap.sql    # サンプルテンプレート
├── params.json           # サンプルパラメータ
├── constants.yaml        # プロジェクト定数
└── README.md            # プロジェクトドキュメント
```

### 2. データベースを設定

`snapsql.yaml`を編集：

```yaml
name: "my-project"
version: "1.0.0"

database:
  default_driver: "postgres"
  connection_string: "postgres://user:password@localhost:5432/mydb"
  timeout: "30s"

paths:
  queries: "./queries"
  generated: "./generated"
```

### 3. 設定をテスト

```bash
# データベース接続をテスト
snapsql config test-db

# 設定を検証
snapsql config validate
```

### 4. 中間ファイルを生成

```bash
# テンプレートから生成
snapsql generate

# 生成を確認
ls generated/
```

### 5. クエリをテスト

```bash
# テンプレートをテストするドライラン
snapsql query queries/users.snap.sql --dry-run --params-file params.json

# クエリを実行
snapsql query queries/users.snap.sql --params-file params.json
```

## 開発環境

### VS Code設定

1. **Go拡張機能をインストール**
2. **`.vscode/settings.json`を作成**:
   ```json
   {
     "go.toolsManagement.checkForUpdates": "local",
     "go.useLanguageServer": true,
     "files.associations": {
       "*.snap.sql": "sql"
     }
   }
   ```

3. **ビルドタスク用の`.vscode/tasks.json`を作成**:
   ```json
   {
     "version": "2.0.0",
     "tasks": [
       {
         "label": "snapsql generate",
         "type": "shell",
         "command": "snapsql generate",
         "group": "build",
         "presentation": {
           "echo": true,
           "reveal": "always"
         }
       }
     ]
   }
   ```

### Git設定

`.gitignore`を作成：
```gitignore
# 生成ファイル
generated/
*.db
*.log

# 環境ファイル
.env
.env.local

# IDEファイル
.vscode/
.idea/
*.swp
*.swo

# OSファイル
.DS_Store
Thumbs.db
```

## 環境変数

異なる環境用の環境変数を設定：

### 開発環境 (.env.dev)
```bash
DATABASE_URL=postgres://dev:dev@localhost:5432/myapp_dev
SNAPSQL_CONFIG=snapsql.dev.yaml
SNAPSQL_VERBOSE=true
```

### 本番環境 (.env.prod)
```bash
DATABASE_URL=postgres://user:password@prod-server:5432/myapp
SNAPSQL_CONFIG=snapsql.prod.yaml
SNAPSQL_QUIET=true
```

環境変数を読み込み：
```bash
# 開発環境を読み込み
source .env.dev
snapsql query queries/users.snap.sql

# または特定の設定で使用
snapsql --config snapsql.prod.yaml query queries/users.snap.sql
```

## トラブルシューティング

### よくある問題

1. **コマンドが見つからない**:
   ```bash
   # Go binがPATHにあるかチェック
   echo $PATH | grep $(go env GOPATH)/bin
   
   # 不足している場合はPATHに追加
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

2. **データベース接続失敗**:
   ```bash
   # 手動で接続をテスト
   psql "postgres://user:password@localhost:5432/dbname"
   
   # サービスが実行中かチェック
   sudo systemctl status postgresql
   ```

3. **権限拒否**:
   ```bash
   # ファイル権限をチェック
   ls -la snapsql.yaml
   
   # 権限を修正
   chmod 644 snapsql.yaml
   ```

4. **テンプレートが見つからない**:
   ```bash
   # ファイルが存在するかチェック
   ls -la queries/
   
   # 絶対パスを使用
   snapsql query /full/path/to/template.snap.sql
   ```

### ヘルプの取得

- コマンドヘルプをチェック: `snapsql --help`
- 設定を検証: `snapsql config validate`
- 詳細出力を有効化: `snapsql --verbose <コマンド>`
- GitHubイシューをチェック: [https://github.com/shibukawa/snapsql/issues](https://github.com/shibukawa/snapsql/issues)

## 次のステップ

インストール後：

1. [テンプレート構文](template-syntax.ja.md)ガイドを読む
2. [設定](configuration.ja.md)オプションについて学ぶ
3. [CLIコマンド](cli-commands.ja.md)を探索する
4. コントリビューションについては[開発ガイド](development.ja.md)をチェック
