# Project Setup

SnapSQLプロジェクトの初期設定について説明します。

## プロジェクトの作成

SnapSQLプロジェクトを初期化するには、作業したいプロジェクトフォルダの中で `snapsql init` を実行します。コマンドは現在のディレクトリを初期化し、新しいフォルダを作成する動作は行いません：

```bash
# プロジェクトフォルダを作成済みで、フォルダ内で実行します
mkdir my-project
cd my-project
snapsql init
```

`snapsql init` は以下のファイルとディレクトリ構造を作成します：

```
./
├── snapsql.yaml          # プロジェクト設定ファイル
├── queries/              # SQLクエリファイルのディレクトリ
├── tests/                # テストファイルのディレクトリ
├── fixtures/             # テストデータ（フィクスチャ）のディレクトリ
└── generated/            # 生成されたコードの出力先
```

## 設定ファイルの説明

`snapsql.yaml` はプロジェクトの設定ファイルです（例）：

```yaml
# コード生成設定
generate:
  language: go
  package: client
  output: generated/client.go

# テスト設定
test:
  parallel: true
  timeout: 30s
```

※ データベース接続設定はtblsの接続設定を流用する方針です（下記参照）。

## データベース接続の設定（.tbls.yml を利用）

現在、プロジェクトのデータベース接続設定は tbls の接続設定フォーマットを利用する方針で統一しています。SnapSQL はプロジェクトルートに置かれた `.tbls.yml` を読み込みます。tbls の `tbls config` コマンドでも接続情報を管理できますが、ここではファイル方式（`.tbls.yml`）の例を示します。

`.tbls.yml` の接続設定例（PostgreSQL の場合）：

```yaml
# .tbls.yml
dsn: "postgres://username:password@localhost:5432/database_name?sslmode=disable"
driver: postgres
```

または環境変数で設定する場合：

```bash
export TBLS_DSN="postgres://username:password@localhost:5432/database_name?sslmode=disable"
export TBLS_DRIVER=postgres
```

SnapSQL は `.tbls.yml`（または上記の環境変数）を参照して、テスト実行やスキーマ情報の取得に利用します。プロジェクト内で `tbls` を使って接続を検証してから SnapSQL の設定を行うとスムーズです。

## テーブル設計の見本

SnapSQLはテーブル設計がある程度完了している前提で使用します。設計は以下のツールなどを利用してください：

- **JetBrains DataGrip**: プロフェッショナルなデータベースIDE
- **DbDocs**: データベーススキーマのドキュメント化
- **DBeaver**: 無料のデータベース管理ツール
- **A5:ER**: ER図作成ツール
- **生成AI**: ChatGPTやClaudeなどのAIアシスタント

### サンプルスキーマ（DB別）

方言差があるため、PostgreSQL / MySQL / SQLite 向けにそれぞれのサンプルスキーマを用意しました。ドキュメント上でタブを切り替えて該当するスキーマをコピーし、プロジェクトルートに `schema_postgres.sql` / `schema_mysql.sql` / `schema_sqlite.sql` のいずれかとして保存してください。

::: tabs

== PostgreSQL

```sql
-- Users table
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Projects table
CREATE TABLE projects (
  id SERIAL PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  description TEXT,
  owner_id INTEGER REFERENCES users(id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tasks table
CREATE TABLE tasks (
  id SERIAL PRIMARY KEY,
  project_id INTEGER REFERENCES projects(id),
  title VARCHAR(300) NOT NULL,
  description TEXT,
  status VARCHAR(20) DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'done')),
  assignee_id INTEGER REFERENCES users(id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX idx_tasks_status ON tasks(status);
```

== MySQL

```sql
-- Users table
CREATE TABLE users (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Projects table
CREATE TABLE projects (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(200) NOT NULL,
  description TEXT,
  owner_id BIGINT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (owner_id) REFERENCES users(id)
);

-- Tasks table
CREATE TABLE tasks (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  project_id BIGINT,
  title VARCHAR(300) NOT NULL,
  description TEXT,
  status ENUM('todo','in_progress','done') DEFAULT 'todo',
  assignee_id BIGINT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (project_id) REFERENCES projects(id),
  FOREIGN KEY (assignee_id) REFERENCES users(id)
);

-- Indexes
CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX idx_tasks_status ON tasks(status);
```

== SQLite

```sql
-- SQLite does not support SERIAL/AUTO_INCREMENT keywords; use INTEGER PRIMARY KEY AUTOINCREMENT
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  description TEXT,
  owner_id INTEGER,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id INTEGER,
  title TEXT NOT NULL,
  description TEXT,
  status TEXT DEFAULT 'todo',
  assignee_id INTEGER,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (project_id) REFERENCES projects(id),
  FOREIGN KEY (assignee_id) REFERENCES users(id)
);

CREATE INDEX idx_tasks_project_id ON tasks(project_id);
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);
CREATE INDEX idx_tasks_status ON tasks(status);
```

:::

### スキーマファイルの配置

プロジェクトルートに、使用するデータベースに応じて次のいずれかのファイル名で保存してください：

- `schema_postgres.sql`
- `schema_mysql.sql`
- `schema_sqlite.sql`

コマンドから読み込む際は、該当ファイル名を指定してください（例：`sqlite3 snapsql.db < schema_sqlite.sql`）。

## データベースにスキーマを投入する

ここではローカルで簡単に動作確認できるよう、各データベース向けに独立した手順をタブで示します。

各タブ内の手順はローカルで独立したコンテナ／ファイルを使うため、既存のデータベース設定や本番データに触れずに試せます。もし既に同じポートで DB が稼働している場合はポート番号を変更するか、別名前のコンテナ名を指定してください。

> 注意: 同じ `schema.sql` が各データベースの方言差を吸収しているとは限りません。必要なら `schema_postgres.sql` / `schema_mysql.sql` / `schema_sqlite.sql` のように分けてください。

::: tabs

== PostgreSQL

以下はローカルの隔離された PostgreSQL コンテナを立ち上げ、`schema.sql` を初期化時に実行する例です（カレントディレクトリに `schema.sql` がある前提）。ホスト上の Postgres に影響を与えないためには、既存の 5432 ポートや同名コンテナが使われていないことを確認してください。

```bash
# プロジェクトルートで実行
docker run --name snapsql-postgres \
  -e POSTGRES_USER=snapsql \
  -e POSTGRES_PASSWORD=pass \
  -e POSTGRES_DB=snapsqldb \
  -v "$PWD/schema.sql":/docker-entrypoint-initdb.d/schema.sql:ro \
  -p 5432:5432 \
  -d postgres:15
```

手動で流し込みたい場合はホストの psql、またはコンテナ経由で実行できます（ユーザー名/パスワードはコンテナ内のものです）：

```bash
# ホストの psql がある場合
psql "postgresql://snapsql:pass@localhost:5432/snapsqldb?sslmode=disable" -f schema.sql

# またはコンテナ内 psql に流し込む
cat schema.sql | docker exec -i snapsql-postgres psql -U snapsql -d snapsqldb
```

接続情報（例、tbls/SnapSQL 用）:

```bash
export TBLS_DSN="postgres://snapsql:pass@localhost:5432/snapsqldb?sslmode=disable"
export TBLS_DRIVER=postgres
```

== MySQL

ローカルの隔離された MySQL コンテナを使う例です。既存の MySQL が同じポートで動作していないことを確認してください。

```bash
docker run --name snapsql-mysql \
  -e MYSQL_ROOT_PASSWORD=rootpass \
  -e MYSQL_DATABASE=snapsqldb \
  -e MYSQL_USER=snapsql \
  -e MYSQL_PASSWORD=pass \
  -v "$PWD/schema.sql":/docker-entrypoint-initdb.d/schema.sql:ro \
  -p 3306:3306 \
  -d mysql:8 --default-authentication-plugin=mysql_native_password
```

手動で流し込む例：

```bash
# コンテナ内の mysql クライアントを使う
docker exec -i snapsql-mysql mysql -usnapsql -ppass snapsqldb < schema.sql

# ホスト側に mysql クライアントがある場合
mysql -h 127.0.0.1 -P 3306 -u snapsql -p"pass" snapsqldb < schema.sql
```

接続情報（例、tbls/SnapSQL 用）:

```bash
export TBLS_DSN="snapsql:pass@tcp(localhost:3306)/snapsqldb?parseTime=true"
export TBLS_DRIVER=mysql
```

== SQLite

SQLite はファイルを作るだけなので、ローカルにある `schema.sql` から DB ファイルを生成します。既存のファイルを上書きしないように注意してください。

```bash
# 新しいファイルを作成する（既存ファイルがある場合は上書きされます）
sqlite3 snapsql.db < schema.sql

# テーブル一覧を確認
sqlite3 snapsql.db "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;"
```

接続情報（例、tbls/SnapSQL 用）:

```bash
export TBLS_DSN="./snapsql.db"
export TBLS_DRIVER=sqlite3
```

:::

## 次のステップ

プロジェクトのセットアップが完了したら、[SQLクエリの作成](./write-sql-query) に進みましょう。

## 関連セクション

* [initコマンド（プロジェクト初期化）](../guides/command-reference/init.md)
* [Go 言語リファレンス（生成コードの利用方法）](../guides/language-reference/go.md)
* [ユーザーリファレンス：設定](../guides/user-reference/configuration.md)