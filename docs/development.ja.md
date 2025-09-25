# 開発ガイド

このガイドでは、SnapSQL開発への貢献方法を説明します。

## 開発環境設定

### 前提条件

- Go 1.24以降
- Git
- Make（ビルドスクリプト用、オプション）
- Docker（データベーステスト用）

### リポジトリのクローン

```bash
git clone https://github.com/shibukawa/snapsql.git
cd snapsql
```

### ソースからビルド

```bash
# CLIツールをビルド
go build ./cmd/snapsql

# テストを実行
go test ./...

# レース検出付きで実行
go test -race ./...

# リンターを実行
golangci-lint run
```

### プロジェクト構造

```
snapsql/
├── cmd/
│   └── snapsql/           # CLIツールソースコード
├── runtime/
│   ├── snapsqlgo/         # Goランタイムライブラリ
│   ├── python/            # Pythonランタイム（計画中）
│   ├── node/              # Node.jsランタイム（計画中）
│   └── java/              # Javaランタイム（計画中）
├── examples/              # サンプルプロジェクト
├── testdata/              # テストデータファイル
├── contrib/               # コミュニティ貢献
├── docs/                  # ドキュメント
├── intermediate/          # 中間形式処理
├── query/                 # クエリ実行エンジン
├── parser/                # SQLパーサー
└── template/              # テンプレートエンジン
```

## コーディング標準

### Goガイドライン

- 標準的なGo規約に従う
- Go 1.24の機能を使用
- `interface{}`より`any`を優先
- 適切な場所で`slices`と`maps`パッケージを使用
- 明示的に要求された場合にジェネリクスを使用
- 機能の後方互換性コピーは作成しない

### エラーハンドリング

- パラメータなしエラーにはセンチネルエラーを使用：
  ```go
  var ErrTemplateNotFound = errors.New("template not found")
  ```
- コンテキスト付きでエラーをラップ：
  ```go
  return fmt.Errorf("failed to parse template: %w", err)
  ```

### テスト

- アサーションには`github.com/alecthomas/assert/v2`を使用
- テストでコンテキストが必要な場合は`testing.T.Context()`を使用
- データベーステストにはTestContainersを使用
- テストファイル命名: `*_test.go`

### 依存関係

**承認されたライブラリ:**
- Webルーティング: `net/http` ServeMux
- CLIパース: `github.com/alecthomas/kong`
- 色付け: `github.com/fatih/color`
- YAML: `github.com/goccy/go-yaml`
- 式: `github.com/google/cel-go`
- Markdown: `github.com/yuin/goldmark`
- PostgreSQL: `github.com/jackc/pgx/v5`
- MySQL: `github.com/go-sql-driver/mysql`
- SQLite: `github.com/mattn/go-sqlite3`

## 開発ワークフロー

### 1. 機能開発

すべての作業はTODOリスト管理システムを通して行う必要があります：

1. **`docs/TODO.md`を確認** 作業開始前に必須
2. **新しいタスクを追加** 優先度とフェーズ付きでTODO.mdに追加
3. **フェーズに従う**:
   - フェーズ1: 情報収集と設計ドキュメント作成
   - フェーズ2: 単体テスト作成（テストは最初失敗する状態）
   - フェーズ3: ソースコード修正とテスト
   - フェーズ4: リファクタリング提案
   - フェーズ5: リファクタリング実施

### 2. 設計ドキュメント

50行を超える機能や大規模リファクタリングの場合：
- `docs/designdocs/{YYYYMMDD}-{機能名}.ja.md`を日本語で作成
- 後方互換性の考慮を含める
- 要件が不明な場合は質問する

### 3. コード変更

実装後、新しいライブラリやコーディングスタイルが導入された場合：
- `docs/coding-standard-suggest.md`にコメントを追加
- 日付と関連タスク情報を含める

### 4. Gitワークフロー

```bash
# フィーチャーブランチを作成
git checkout -b feature/new-feature

# フェーズに従って変更を行う
# ... 開発作業 ...

# 説明的なメッセージでコミット
git commit -m "feat: add dry-run support for query command"

# プッシュしてPRを作成
git push origin feature/new-feature
```

## テスト

### 単体テスト

```bash
# すべてのテストを実行
go test ./...

# 特定のパッケージテストを実行
go test ./cmd/snapsql

# カバレッジ付きで実行
go test -cover ./...

# レース検出付きで実行
go test -race ./...
```

### 統合テスト

```bash
# Dockerでテストデータベースを開始
docker-compose -f docker-compose.test.yml up -d

# 統合テストを実行
go test -tags=integration ./...

# クリーンアップ
docker-compose -f docker-compose.test.yml down
```

### テストデータベース設定

データベーステスト用のTestContainers使用：

```go
func TestWithPostgreSQL(t *testing.T) {
    ctx := t.Context()
    
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    assert.NoError(t, err)
    defer container.Terminate(ctx)
    
    connStr, err := container.ConnectionString(ctx)
    assert.NoError(t, err)
    
    // テスト用にconnStrを使用
}
```

## アーキテクチャ

### 階層レスポンスと HierarchyKeyLevel

多段 `a__b__c__col` 形式の列に対し `hierarchy_key_level` を付与し、主キー階層と集約ロジックを一元化しています。

| hierarchy_key_level | 意味 |
|---------------------|------|
| 0 | 非キー列 |
| 1 | ルート(親)主キー列 |
| 2 | 第一階層子ノード主キー列 |
| 3 | 第二階層子ノード主キー列 |
| … | 深さに対応 |

旧 `is_primary_key` は 2025-09 に削除され、`hierarchy_key_level > 0` がキー判定の唯一の指標です。

集約生成アルゴリズム概要:
1. ルート PK (level=1 複合可) で `_parentMap` を索引
2. 各階層ノードは `depth == hierarchy_key_level` の列セットを PK と見なしローカルキー生成
3. chain key = 親の chain key + `|<path>:<localKey>` を用い `_nodeMap*` で重複排除
4. 行ストリーミング中に木構造を再構築

スキーマが無い場合でも最初のルート列を level=1 にフォールバックし破綻を防ぎます。

### パーサーコンビネーターシステム

SnapSQLはSQLパース用のカスタムパーサーコンビネーターライブラリを使用：

```go
// 基本パーサー構造
type Parser[T any] func(*ParseContext[T], []Token[T]) (consumed int, newTokens []Token[T], err error)

// ASTノードインターフェース
type ASTNode interface {
    Type() string
    Position() *pc.Pos
}
```

詳細な使用方法は`docs/.amazonq/rules/parsercombinator.md`を参照。

### テンプレートエンジン

テンプレートエンジンは2-way SQLテンプレートを処理：

1. **字句解析**: テンプレートディレクティブ付きSQLをトークン化
2. **パース**: トークンからASTを構築
3. **中間生成**: 実行命令を作成
4. **ランタイム実行**: パラメータ付きで最終SQLを生成

### クエリ実行

クエリ実行エンジン：

1. **テンプレート処理**: テンプレートを実行可能SQLに変換
2. **パラメータバインディング**: インジェクションを防ぐ安全なパラメータバインディング
3. **データベース実行**: 適切なドライバーでクエリを実行
4. **結果フォーマット**: 様々な出力形式で結果をフォーマット

## 貢献

### プルリクエストプロセス

1. **リポジトリをフォーク**
2. **`main`からフィーチャーブランチを作成**
3. **開発ワークフローに従う**（TODO.md管理）
4. **新機能のテストを書く**
5. **必要に応じてドキュメントを更新**
6. **明確な説明付きでプルリクエストを提出**

### PR要件

- [ ] すべてのテストが通る
- [ ] コードがスタイルガイドラインに従っている
- [ ] ドキュメントが更新されている
- [ ] 該当する場合はTODO.mdが更新されている
- [ ] 議論なしに破壊的変更がない

### コードレビュー

- 正確性と保守性に焦点を当てる
- セキュリティ問題をチェック（特にSQLインジェクション防止）
- テストカバレッジを確認
- ドキュメントの正確性を確保

## リリースプロセス

### バージョン管理

SnapSQLはセマンティックバージョニングを使用：
- `MAJOR.MINOR.PATCH`
- Major: 破壊的変更
- Minor: 新機能（後方互換）
- Patch: バグ修正

### リリース手順

1. **関連ファイルのバージョンを更新**
2. **CHANGELOG.mdを更新**
3. **リリースタグを作成**
4. **バイナリをビルドして公開**
5. **ドキュメントを更新**

## デバッグ

### デバッグログを有効化

```bash
# 詳細出力
snapsql --verbose query template.snap.sql

# パーサーをデバッグ
export SNAPSQL_DEBUG_PARSER=true
snapsql generate
```

### 一般的なデバッグシナリオ

1. **テンプレートパース問題**:
   ```bash
   snapsql validate --strict template.snap.sql
   ```

2. **パラメータバインディング問題**:
   ```bash
   snapsql query template.snap.sql --dry-run --params-file params.json
   ```

3. **データベース接続問題**:
   ```bash
   snapsql config test-db
   ```

## パフォーマンス考慮事項

### パーサーパフォーマンス

- 効率的なパーサーコンビネーターを使用
- バックトラッキングを最小化
- 可能な場合はパース結果をキャッシュ

### クエリパフォーマンス

- 生成されたSQLの効率性を検証
- クエリ実行時間を監視
- クエリ分析ツールを提供

### メモリ使用量

- 大きな結果セットをストリーム
- データセット全体をメモリに読み込まない
- 適切にコネクションプールを使用

## セキュリティ

### SQLインジェクション防止

- すべてのパラメータ置換は安全でなければならない
- テンプレート変更を検証
- パラメータ化クエリを使用
- ユーザー入力をサニタイズ

### アクセス制御

- データベース権限を検証
- クエリ制限を実装
- 危険な操作を監査

## ドキュメント

### ドキュメント作成

- 明確で簡潔な言語を使用
- 実用的な例を提供
- ドキュメントを最新に保つ
- トラブルシューティングセクションを含める

### ドキュメント構造

- `README.md`: プロジェクト概要とクイックスタート
- `docs/`: 詳細ドキュメント
- `examples/`: 動作する例
- コードコメント: 実装詳細

## ヘルプの取得

- **GitHubイシュー**: バグレポートと機能リクエスト
- **ディスカッション**: 一般的な質問とアイデア
- **コードレビュー**: 実装フィードバック
- **ドキュメント**: 明確化と改善
