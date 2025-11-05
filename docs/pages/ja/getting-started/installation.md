# Installation

SnapSQLのインストール方法について説明します。

## 前提条件

SnapSQLを使用するには、以下の環境が必要です：

- アプリケーション実装言語
  - Go 1.24以上
- データベース(以下のうちいずれか)
  - PostgreSQL
  - MySQL
  - SQLit

## データベースのセットアップ

Dockerを使用して各データベースを簡単にセットアップできます：

::: tabs

== PostgreSQL

```bash
# PostgreSQLコンテナ起動
docker run --name snapsql-postgres \
  -e POSTGRES_DB=snapsql \
  -e POSTGRES_USER=snapsql \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  -d postgres:15

# 接続確認
psql -h localhost -U snapsql -d snapsql
```

== MySQL

```bash
# MySQLコンテナ起動
docker run --name snapsql-mysql \
  -e MYSQL_DATABASE=snapsql \
  -e MYSQL_USER=snapsql \
  -e MYSQL_PASSWORD=password \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -p 3306:3306 \
  -d mysql:8.0

# 接続確認
mysql -h localhost -u snapsql -p snapsql
```

== SQLite

SQLiteはファイルベースなのでDockerは不要です。Goの標準ライブラリでサポートされています。

ただし、開発時にデータベースの内容を確認するために`sqlite3`コマンドラインツールが必要です：

#### macOS
macOSには標準でインストールされています。

#### Ubuntu/Debian
```bash
sudo apt install sqlite3
```

#### Windows (Chocolatey)
```bash
choco install sqlite3
```

#### 確認
```bash
sqlite3 --version
```

:::

## SnapSQLと推奨ツールのインストール

Goがインストールされたら、SnapSQL本体と推奨ツールをインストールします。

### SnapSQLのインストール

```bash
go install github.com/shibukawa/snapsql/cmd/snapsql@latest
```

インストール後、PATHに`~/go/bin`（またはGoのGOPATH/bin）が含まれていることを確認してください：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

#### Go経由でのインストール

Goがインストールされている場合、以下のコマンドで最新版をインストールできます：

```bash
go install github.com/shibukawa/snapsql/cmd/snapsql@latest
```

インストール後、PATHに`~/go/bin`（またはGoのGOPATH/bin）が含まれていることを確認してください：

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

#### バイナリリリースからのインストール

GitHub Releasesからプラットフォームに合ったバイナリをダウンロードできます：

1. [GitHub Releases](https://github.com/shibukawa/snapsql/releases) にアクセス
2. 最新版のリリースから、お使いのプラットフォームに合ったバイナリをダウンロード
3. 実行権限を付与してPATHの通った場所に配置：

```bash
# Linux/macOSの場合
chmod +x snapsql
sudo mv snapsql /usr/local/bin/

# Windowsの場合
# ダウンロードしたexeファイルをPATHの通ったフォルダに移動
```

#### SnapSQLのインストール確認

SnapSQLのインストールが成功したか確認するには：

```bash
snapsql --version
```

バージョン情報が表示されればインストール成功です。

### tblsのインストール

SnapSQLでは、データベーススキーマのドキュメント生成に[tbls](https://github.com/k1LoW/tbls)を使用することを推奨します。

#### Go経由でのインストール

```bash
go install github.com/k1LoW/tbls@latest
```

#### バイナリリリースからのインストール

[GitHub Releases](https://github.com/k1LoW/tbls/releases)からプラットフォームに合ったバイナリをダウンロード：

```bash
# Linux/macOSの場合
curl -L https://github.com/k1LoW/tbls/releases/latest/download/tbls_linux_amd64.tar.gz | tar xvz
sudo mv tbls /usr/local/bin/

# Windowsの場合
# ダウンロードしたzipファイルを展開し、PATHに追加
```

#### Homebrew (macOS)

```bash
brew install k1low/tap/tbls
```

#### インストール確認

```bash
tbls --version
```

## 次のステップ

インストールが完了したら、[プロジェクトのセットアップ](./project-setup) に進みましょう。

## 関連セクション

* [initコマンド](../guides/command-reference/init.md)