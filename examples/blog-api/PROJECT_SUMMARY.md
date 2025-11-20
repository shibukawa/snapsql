# Blog API - プロジェクトサマリー

## 概要

FastAPI + SnapSQL Python Code Generatorを使用したブログAPIのサンプルプロジェクトです。

## 作成されたファイル

### 設定ファイル
- `README.md` - プロジェクトの詳細ドキュメント
- `QUICKSTART.md` - クイックスタートガイド（日本語）
- `requirements.txt` - Python依存関係
- `docker-compose.yml` - PostgreSQLデータベース設定
- `snapsql.yaml` - SnapSQL設定ファイル
- `schema.sql` - データベーススキーマ
- `.gitignore` - Git除外設定

### クエリ定義 (.snap.md形式)
- `queries/user_create.snap.md` - ユーザー作成
- `queries/user_get.snap.md` - ユーザー取得
- `queries/user_list.snap.md` - ユーザー一覧
- `queries/post_create.snap.md` - 投稿作成
- `queries/post_get.snap.md` - 投稿取得（著者情報付き）
- `queries/post_list.snap.md` - 投稿一覧（ページネーション対応）
- `queries/comment_create.snap.md` - コメント作成
- `queries/comment_list_by_post.snap.md` - 投稿別コメント一覧

### アプリケーションコード
- `app/__init__.py` - アプリケーションパッケージ
- `app/main.py` - FastAPIメインアプリケーション
- `app/database.py` - データベース接続管理
- `app/models.py` - Pydanticモデル（リクエスト/レスポンス）
- `app/routers/__init__.py` - ルーターパッケージ
- `app/routers/users.py` - ユーザーAPIエンドポイント
- `app/routers/posts.py` - 投稿APIエンドポイント
- `app/routers/comments.py` - コメントAPIエンドポイント

## データベーススキーマ

### テーブル

1. **users** - ユーザー情報
   - user_id (PK)
   - username (UNIQUE)
   - email (UNIQUE)
   - full_name
   - bio
   - created_at, updated_at

2. **posts** - ブログ投稿
   - post_id (PK)
   - title
   - content
   - author_id (FK → users)
   - published
   - view_count
   - created_at, updated_at
   - created_by, updated_by (FK → users)

3. **comments** - コメント
   - comment_id (PK)
   - post_id (FK → posts)
   - author_id (FK → users)
   - content
   - created_at, updated_at

## API エンドポイント

### Users
- `POST /users` - ユーザー作成
- `GET /users/{user_id}` - ユーザー取得
- `GET /users` - ユーザー一覧
- `PUT /users/{user_id}` - ユーザー更新
- `DELETE /users/{user_id}` - ユーザー削除

### Posts
- `POST /posts` - 投稿作成
- `GET /posts/{post_id}` - 投稿取得（著者情報付き）
- `GET /posts` - 投稿一覧（ページネーション）
- `PUT /posts/{post_id}` - 投稿更新
- `DELETE /posts/{post_id}` - 投稿削除

### Comments
- `POST /comments` - コメント作成
- `GET /comments/post/{post_id}` - 投稿のコメント一覧
- `DELETE /comments/{comment_id}` - コメント削除

## SnapSQL機能のデモンストレーション

### 1. Response Affinity
- **:one** - 単一レコード取得（user_get, post_get）
- **:many** - 複数レコード取得（user_list, post_list, comment_list）
- **:exec** - 実行のみ（create, update, delete）

### 2. 階層的レスポンス
- `author__username`, `author__full_name` - ネストされた著者情報
- JOINを使用した関連データの取得

### 3. パラメータ処理
- 必須パラメータ（username, email, title, content）
- オプショナルパラメータ（full_name, bio）
- システムパラメータ（created_by, updated_by）

### 4. エラーハンドリング
- **NotFoundError** - レコードが見つからない場合
- **ValidationError** - パラメータ検証失敗
- **DatabaseError** - データベース操作失敗
- **UnsafeQueryError** - WHERE句なしのUPDATE/DELETE

### 5. システムカラム
- `created_at`, `updated_at` - 自動タイムスタンプ
- `created_by`, `updated_by` - ユーザー追跡

### 6. クエリロギング
- 自動クエリログ記録
- スロークエリ検出
- エラー追跡

## 技術スタック

- **FastAPI** - 高速なPython Webフレームワーク
- **asyncpg** - PostgreSQL非同期ドライバ
- **Pydantic** - データバリデーション
- **PostgreSQL** - リレーショナルデータベース
- **Docker** - コンテナ化
- **SnapSQL** - SQLクエリからPythonコード生成

## 使用方法

詳細な手順は以下を参照:
- [QUICKSTART.md](QUICKSTART.md) - クイックスタートガイド
- [README.md](README.md) - 完全なドキュメント

### 基本的な流れ

1. データベース起動: `docker-compose up -d`
2. 依存関係インストール: `pip install -r requirements.txt`
3. コード生成: `go run ./cmd/snapsql generate --config examples/blog-api/snapsql.yaml --lang python`
4. サーバー起動: `uvicorn app.main:app --reload`
5. APIドキュメント: http://localhost:8000/docs

## テストデータ

`schema.sql` にサンプルデータが含まれています:
- 3人のユーザー（alice, bob, charlie）
- 4つの投稿（3つ公開、1つ下書き）
- 4つのコメント

## 拡張のアイデア

1. **認証・認可**
   - JWT トークン認証
   - ユーザーロールとパーミッション

2. **追加機能**
   - タグ機能
   - いいね機能
   - フォロー機能
   - 全文検索

3. **パフォーマンス**
   - Redis キャッシング
   - クエリ最適化
   - ページネーション改善

4. **テスト**
   - ユニットテスト
   - 統合テスト
   - E2Eテスト

5. **デプロイ**
   - Docker化
   - Kubernetes設定
   - CI/CD パイプライン

## ライセンス

MIT License
