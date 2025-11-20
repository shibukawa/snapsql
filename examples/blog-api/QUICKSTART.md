# Blog API - Quick Start Guide

このガイドでは、Blog APIサンプルプロジェクトを最速でセットアップして実行する手順を説明します。

## 前提条件

- [uv](https://docs.astral.sh/uv/) - 高速Pythonパッケージインストーラー
- Docker と Docker Compose
- Go 1.21以上 (SnapSQLコード生成用)

## 最速セットアップ（ワンコマンド）

リポジトリルートから以下のコマンドを実行するだけ:

```bash
./examples/blog-api/run.sh
```

このスクリプトは以下を自動的に実行します:
1. PostgreSQLデータベースの起動
2. Pythonクエリコードの生成
3. uvで依存関係のインストール
4. FastAPIサーバーの起動

## 手動セットアップ手順（リポジトリルートから）

手動で各ステップを実行したい場合:

### 1. データベースの起動

```bash
# リポジトリルートから
cd examples/blog-api
docker-compose up -d
cd ../..  # ルートに戻る
```

データベースが起動するまで数秒待ちます:

```bash
docker-compose -f examples/blog-api/docker-compose.yml logs -f postgres
# "database system is ready to accept connections" が表示されたらCtrl+Cで終了
```

### 2. SnapSQLでPythonコードを生成

```bash
# リポジトリルートから
go run ./cmd/snapsql generate \
  --config examples/blog-api/snapsql.yaml \
  --lang python \
  --output examples/blog-api/dataaccess
```

### 3. Pythonの依存関係をインストール（uv使用）

```bash
cd examples/blog-api

# uvで依存関係をインストール（自動的に仮想環境を作成）
uv pip install -r requirements.txt
```

### 4. FastAPIサーバーを起動

```bash
# uvで実行（自動的に仮想環境を使用）
uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

### 5. APIを試す

ブラウザで以下のURLを開きます:

- **API ドキュメント**: http://localhost:8000/docs
- **代替ドキュメント**: http://localhost:8000/redoc
- **ヘルスチェック**: http://localhost:8000/health

## APIの使用例

### ユーザーを作成

```bash
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "full_name": "Alice Smith",
    "bio": "Software engineer"
  }'
```

### ユーザー一覧を取得

```bash
curl http://localhost:8000/users
```

### ブログ投稿を作成

```bash
curl -X POST "http://localhost:8000/posts?author_id=1" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Post",
    "content": "This is my first blog post using SnapSQL!",
    "published": true
  }'
```

### 投稿一覧を取得

```bash
curl "http://localhost:8000/posts?limit=10&offset=0"
```

### コメントを追加

```bash
curl -X POST "http://localhost:8000/comments?author_id=1" \
  -H "Content-Type: application/json" \
  -d '{
    "post_id": 1,
    "content": "Great post!"
  }'
```

### 投稿のコメントを取得

```bash
curl http://localhost:8000/comments/post/1
```

## トラブルシューティング

### データベース接続エラー

データベースが起動しているか確認:

```bash
docker-compose ps
```

データベースに直接接続してテスト:

```bash
psql -h localhost -U bloguser -d blogdb
# Password: blogpass
```

### 生成されたコードが見つからない

`dataaccess/` ディレクトリが作成されているか確認:

```bash
ls -la dataaccess/
```

存在しない場合は、ステップ3のコード生成を再実行してください。

### ポート8000が使用中

別のポートを使用:

```bash
uvicorn app.main:app --reload --host 0.0.0.0 --port 8001
```

## クリーンアップ

### サーバーを停止

FastAPIサーバーで `Ctrl+C` を押します。

### データベースを停止

```bash
# examples/blog-api ディレクトリから
docker-compose down

# データも削除する場合
docker-compose down -v

# またはリポジトリルートから
docker-compose -f examples/blog-api/docker-compose.yml down
```

## 次のステップ

- `queries/` ディレクトリの `.snap.md` ファイルを編集してクエリをカスタマイズ
- `app/routers/` のルーターを編集してビジネスロジックを追加
- `schema.sql` を編集してデータベーススキーマを変更
- テストを `tests/` ディレクトリに追加

## SnapSQL機能のデモ

このサンプルプロジェクトは以下のSnapSQL機能を実演しています:

1. **Response Affinity**
   - `:one` - 単一レコード取得 (user_get, post_get)
   - `:many` - 複数レコード取得 (user_list, post_list)
   - `:exec` - 実行のみ (create, update, delete)

2. **階層的レスポンス**
   - `author__username` - ネストされた著者情報

3. **エラーハンドリング**
   - NotFoundError - レコードが見つからない
   - ValidationError - パラメータ検証エラー
   - DatabaseError - データベースエラー

4. **システムカラム**
   - `created_at`, `updated_at` - 自動タイムスタンプ
   - `created_by`, `updated_by` - ユーザー追跡

詳細は [README.md](README.md) を参照してください。
