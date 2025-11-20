# Blog API - セットアップ完了ガイド

## ✅ プロジェクト作成完了

FastAPI + SnapSQL Python Code Generatorのサンプルプロジェクトが完成しました！

## 🚀 起動方法

### 最速起動（推奨）

リポジトリルートから:

```bash
./examples/blog-api/run.sh
```

### 手動起動

```bash
# 1. データベース起動
cd examples/blog-api && docker-compose up -d && cd ../..

# 2. コード生成
go run ./cmd/snapsql generate \
  --config examples/blog-api/snapsql.yaml \
  --lang python \
  --output examples/blog-api/dataaccess

# 3. 依存関係インストール & サーバー起動
cd examples/blog-api
uv pip install -r requirements.txt
uv run uvicorn app.main:app --reload
```

## 📚 ドキュメント

- **README.md** - 完全なプロジェクトドキュメント（英語）
- **QUICKSTART.md** - クイックスタートガイド（日本語）
- **PROJECT_SUMMARY.md** - プロジェクトサマリー（日本語）

## 🌐 アクセスURL

サーバー起動後:

- **API ドキュメント**: http://localhost:8000/docs
- **代替ドキュメント**: http://localhost:8000/redoc
- **ヘルスチェック**: http://localhost:8000/health
- **ルートエンドポイント**: http://localhost:8000/

## 📁 プロジェクト構成

```
examples/blog-api/
├── run.sh                         # ワンコマンド起動スクリプト
├── README.md                      # 完全ドキュメント
├── QUICKSTART.md                  # クイックスタート（日本語）
├── PROJECT_SUMMARY.md             # プロジェクトサマリー
├── requirements.txt               # Python依存関係
├── docker-compose.yml             # PostgreSQL設定
├── snapsql.yaml                   # SnapSQL設定
├── schema.sql                     # DBスキーマ
├── queries/                       # クエリ定義（.snap.md）
│   ├── user_create.snap.md
│   ├── user_get.snap.md
│   ├── user_list.snap.md
│   ├── post_create.snap.md
│   ├── post_get.snap.md
│   ├── post_list.snap.md
│   ├── comment_create.snap.md
│   └── comment_list_by_post.snap.md
├── dataaccess/                    # 生成コード（自動生成）
│   ├── __init__.py
│   └── *.py
└── app/                           # アプリケーション
    ├── main.py
    ├── database.py
    ├── models.py
    └── routers/
        ├── users.py
        ├── posts.py
        └── comments.py
```

## 🎯 実装されている機能

### SnapSQL機能

1. **Response Affinity**
   - `:one` - 単一レコード取得
   - `:many` - 複数レコード取得（async generator）
   - `:exec` - 実行のみ

2. **階層的レスポンス**
   - `author__username` - ネストされた著者情報
   - JOINによる関連データ取得

3. **エラーハンドリング**
   - NotFoundError
   - ValidationError
   - DatabaseError
   - UnsafeQueryError

4. **システムカラム**
   - created_at, updated_at
   - created_by, updated_by

### API エンドポイント

#### Users
- POST /users - ユーザー作成
- GET /users/{user_id} - ユーザー取得
- GET /users - ユーザー一覧
- PUT /users/{user_id} - ユーザー更新
- DELETE /users/{user_id} - ユーザー削除

#### Posts
- POST /posts - 投稿作成
- GET /posts/{post_id} - 投稿取得
- GET /posts - 投稿一覧（ページネーション）
- PUT /posts/{post_id} - 投稿更新
- DELETE /posts/{post_id} - 投稿削除

#### Comments
- POST /comments - コメント作成
- GET /comments/post/{post_id} - 投稿のコメント一覧
- DELETE /comments/{comment_id} - コメント削除

## 🧪 APIテスト例

### ユーザー作成

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

### 投稿作成

```bash
curl -X POST "http://localhost:8000/posts?author_id=1" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Post",
    "content": "Hello, SnapSQL!",
    "published": true
  }'
```

### 投稿一覧取得

```bash
curl "http://localhost:8000/posts?limit=10&offset=0"
```

## 🛠️ 開発ワークフロー

### クエリを変更した場合

1. `queries/*.snap.md` ファイルを編集
2. コード再生成:
   ```bash
   go run ./cmd/snapsql generate \
     --config examples/blog-api/snapsql.yaml \
     --lang python \
     --output examples/blog-api/dataaccess
   ```
3. サーバーが自動リロード（`--reload`オプション使用時）

### スキーマを変更した場合

1. `schema.sql` を編集
2. データベースに適用:
   ```bash
   docker-compose -f examples/blog-api/docker-compose.yml \
     exec postgres psql -U bloguser -d blogdb \
     -f /docker-entrypoint-initdb.d/schema.sql
   ```

## 🧹 クリーンアップ

### サーバー停止

`Ctrl+C` でFastAPIサーバーを停止

### データベース停止

```bash
cd examples/blog-api
docker-compose down

# データも削除する場合
docker-compose down -v
```

## 📖 次のステップ

1. **APIドキュメントを確認**: http://localhost:8000/docs
2. **クエリファイルを編集**: `queries/*.snap.md`
3. **ルーターをカスタマイズ**: `app/routers/*.py`
4. **新しいエンドポイントを追加**
5. **テストを作成**: `tests/`

## 💡 ヒント

- **Interactive API Docs** (http://localhost:8000/docs) を使うと、ブラウザから直接APIをテストできます
- `uv run` を使うと、仮想環境を自動的に管理してくれます
- `--reload` オプションでコード変更時に自動リロードされます
- サンプルデータは `schema.sql` に含まれています

## 🐛 トラブルシューティング

### ポート8000が使用中

```bash
uv run uvicorn app.main:app --reload --port 8001
```

### データベース接続エラー

```bash
# データベースの状態確認
docker-compose -f examples/blog-api/docker-compose.yml ps

# ログ確認
docker-compose -f examples/blog-api/docker-compose.yml logs postgres
```

### 生成コードが見つからない

```bash
# コード生成を再実行
go run ./cmd/snapsql generate \
  --config examples/blog-api/snapsql.yaml \
  --lang python \
  --output examples/blog-api/dataaccess
```

## 📝 ライセンス

MIT License

---

**Happy Coding! 🎉**
