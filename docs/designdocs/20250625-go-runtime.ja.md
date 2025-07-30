# Go 低レベルランタイム設計ドキュメント

**日付**: 2025-06-25  
**作成者**: 開発チーム  
**ステータス**: ドラフト（修正版）  

## 概要

このドキュメントは、Go 1.24の機能を使用したSQL生成と実行に特化したSnapSQL Go低レベルランタイムライブラリの設計について説明します。

## 目標

### 主要目標
- 実行時パラメータでSnapSQLテンプレートからSQLを生成
- インメモリステートメントキャッシュを使用した効率的なPrepareContextの提供
- database/sqlインターフェースでのExecContext/QueryContextを使用したクエリ実行
- Row、Columns、ColumnTypesのGo 1.24イテレータパターンを使用した結果返却
- SQL生成と実行に特化した最小限で焦点を絞ったAPI

### 副次目標
- パフォーマンス最適化のためのステートメントキャッシュ
- database/sql.DBとdatabase/sql.Txとのクリーンな統合
- メモリ効率的な結果イテレーション
- 全体を通した適切なコンテキスト処理

## 非目標
- 高レベルORM機能
- データベース接続管理
- トランザクション管理
- モック/テストフレームワーク（別の関心事）
- 設定管理
- テンプレート解析（事前生成された中間JSONを使用）

## アーキテクチャ

### コアコンポーネント

#### 1. Template Loader (`runtime/snapsqlgo/loader.go`)
- 任意のfs.FS実装から中間JSONファイルを読み込み
- go:embed、fs.Dir、fs.Root、カスタムファイルシステム実装をサポート
- 中間形式をTemplate構造体に解析
- 第1レベルキャッシュ（ファイル名 → 解析済みテンプレート）を提供

```go
type TemplateLoader struct {
    fs            fs.FS
    templateCache map[string]*Template
    mutex         sync.RWMutex
}

func NewTemplateLoader(fsys fs.FS) *TemplateLoader
func (l *TemplateLoader) LoadTemplate(templateName string) (*Template, error)
func (l *TemplateLoader) ClearTemplateCache()
```

#### 2. SQL Generator (`runtime/snapsqlgo/generator.go`)
- 実行時パラメータでテンプレートからSQLを生成
- 第2レベルキャッシュ（構造影響パラメータ → 生成SQL）を提供
- パラメータ置換と条件ロジックを処理
- すべてのキャッシュレベルの集約キャッシュサイズ設定

```go
type SQLGeneratorConfig struct {
    TemplateCacheSize int // デフォルト: 無制限（埋め込みテンプレート数で制限）
    SQLCacheSize      int // デフォルト: 1000
    StmtCacheSize     int // デフォルト: 100（Executorごとに上書き可能）
}

type SQLGenerator struct {
    loader    *TemplateLoader
    sqlCache  map[string]*CachedSQL
    config    SQLGeneratorConfig
    mutex     sync.RWMutex
}

type CachedSQL struct {
    SQL        string
    ParamNames []string
    GeneratedAt time.Time
}

func NewSQLGenerator(loader *TemplateLoader) *SQLGenerator
func NewSQLGeneratorWithConfig(loader *TemplateLoader, config SQLGeneratorConfig) *SQLGenerator
func (g *SQLGenerator) GenerateSQL(templateName string, params map[string]any) (string, []any, error)
func (g *SQLGenerator) GetDefaultStmtCacheSize() int
func (g *SQLGenerator) ClearSQLCache()
func (g *SQLGenerator) ClearAllCaches()
```

#### 3. Statement Cache (`runtime/snapsqlgo/cache.go`)
- 階層的再利用性を持つコンテキスト対応プリペアドステートメントキャッシュ
- sql.DB、sql.Conn、sql.Txコンテキストの分離キャッシュ管理
- 異なるコンテキスト間でのステートメント再利用性ルールを理解
- 適切なクリーンアップを伴うスレッドセーフな操作

```go
type StatementCache struct {
    dbCache   map[string]*sql.Stmt                    // グローバルDBステートメント
    connCache map[uintptr]map[string]*sql.Stmt       // 接続ごとのステートメント
    txCache   map[uintptr]map[string]*sql.Stmt       // トランザクションごとのステートメント
    mutex     sync.RWMutex
    maxSize   int
}

func NewStatementCache(maxSize int) *StatementCache
func (c *StatementCache) Get(ctx ContextInfo, sql string) (*sql.Stmt, bool)
func (c *StatementCache) Set(ctx ContextInfo, sql string, stmt *sql.Stmt)
func (c *StatementCache) CleanupContext(ctx ContextInfo) // コンテキスト終了時のクリーンアップ
func (c *StatementCache) Clear()
```

#### 4. Executor (`runtime/snapsqlgo/executor.go`)
- 任意のDBExecutor実装で動作するジェネリックエグゼキュータ
- 自動コンテキスト検出と適切なキャッシュ戦略
- Go 1.24イテレータベースの結果を返却
- ステートメントキャッシュの自動コンテキストクリーンアップ
- プリペアドステートメント（デフォルト）と直接実行（DDL等）の両方をサポート
- エグゼキュータインスタンスごとにステートメントキャッシュサイズを上書き可能

```go
type Executor[DB DBExecutor] struct {
    generator *SQLGenerator
    stmtCache *StatementCache
    db        DB
    context   ContextInfo
}

func NewExecutor[DB DBExecutor](generator *SQLGenerator, db DB) *Executor[DB]
func NewExecutorWithStmtCacheSize[DB DBExecutor](generator *SQLGenerator, db DB, stmtCacheSize int) *Executor[DB]

// プリペアドステートメントを使用する標準メソッド（ほとんどのケースで推奨）
func (e *Executor[DB]) Query(ctx context.Context, templateName string, params map[string]any) (*ResultIterator, error)
func (e *Executor[DB]) Exec(ctx context.Context, templateName string, params map[string]any) (sql.Result, error)

// 直接実行メソッド（DDL、一回限りのクエリ等用）
func (e *Executor[DB]) QueryDirect(ctx context.Context, templateName string, params map[string]any) (*ResultIterator, error)
func (e *Executor[DB]) ExecDirect(ctx context.Context, templateName string, params map[string]any) (sql.Result, error)

// 生SQL実行（テンプレートシステムを完全にバイパス）
func (e *Executor[DB]) QueryRaw(ctx context.Context, sql string, args ...any) (*ResultIterator, error)
func (e *Executor[DB]) ExecRaw(ctx context.Context, sql string, args ...any) (sql.Result, error)

// ステートメントキャッシュ管理（ジェネレータのデフォルトを上書き可能）
func (e *Executor[DB]) SetStmtCacheSize(size int)
func (e *Executor[DB]) GetStmtCacheSize() int
func (e *Executor[DB]) ClearStmtCache()

// キャッシュとライフサイクル管理
func (e *Executor[DB]) ClearAllCaches()
func (e *Executor[DB]) Close() error

// コンテキスト検出ヘルパー
func detectContextInfo[DB DBExecutor](db DB) ContextInfo
func (e *Executor[DB]) cleanup() // エグゼキュータクローズ時に自動呼び出し
```

#### 5. Result Iterator (`runtime/snapsqlgo/iterator.go`)
- クエリ結果のGo 1.24イテレータ実装（前の設計から変更なし）
- Rows、Columns、ColumnTypesへのアクセスを提供
- メモリ効率的な結果ストリーミング

### データフロー

1. **テンプレート読み込み（第1レベルキャッシュ）**
   - テンプレートは`go:embed`ディレクティブで埋め込まれる
   - `TemplateLoader.LoadTemplate(templateName)`がテンプレートキャッシュをチェック
   - キャッシュされていない場合、中間JSONファイルを読み込み解析
   - 解析されたテンプレートがファイル名をキーとしてキャッシュされる

2. **SQL生成（第2レベルキャッシュ）**
   - `SQLGenerator.GenerateSQL(templateName, params)`が構造影響パラメータを抽出
   - 構造影響パラメータには：テーブルサフィックス、条件付きフィールド選択など
   - テンプレート名 + 構造影響パラメータハッシュからキャッシュキーを生成
   - キャッシュされたSQLが存在する場合、パラメータ値がキャッシュされたSQLに置換される
   - キャッシュされていない場合、テンプレートASTからSQLが生成されキャッシュされる

3. **ステートメント準備とキャッシュ（第3レベルキャッシュ）**
   - `Executor.Query/Exec`が生成されたSQLとパラメータを受け取る
   - コンテキストタイプがジェネリックDBパラメータから自動検出される
   - ステートメントキャッシュ検索は階層再利用性ルールに従う：
     - **DBコンテキスト**: グローバルDBキャッシュをチェック
     - **Connコンテキスト**: 接続固有キャッシュをチェック、DBキャッシュにフォールバック
     - **Txコンテキスト**: トランザクション固有キャッシュをチェック、親接続キャッシュ、次にDBキャッシュにフォールバック
   - キャッシュされていない場合、DBインスタンスで`PrepareContext`が呼ばれ結果がキャッシュされる
   - プリペアドステートメントが実行に使用される
   - コンテキスト固有のステートメントはコンテキスト終了時にクリーンアップされる

4. **クエリ実行**
   - クエリの場合：`Query`が呼ばれ、`ResultIterator`を返す
   - コマンドの場合：`Exec`が呼ばれ、`sql.Result`を返す
   - 結果は効率的な処理のためにGo 1.24イテレータパターンを使用

### ステートメントキャッシュ戦略詳細

#### コンテキスト対応ステートメントキャッシュ
プリペアドステートメントは、そのコンテキストに応じて異なるライフサイクルと再利用性を持ちます：

**sql.DBコンテキスト:**
- ステートメントは複数のトランザクション間で再利用可能
- 長寿命で、ゴルーチン間で共有される
- SQLをキーとしてグローバルにキャッシュ
- 明示的なキャッシュクリアまたはアプリケーション終了時のみクリーンアップ

**sql.Txコンテキスト:**
- ステートメントは特定のトランザクションに束縛される
- トランザクションのcommit/rollback後は再利用不可
- （txポインタ + SQL）を複合キーとしてトランザクションごとにキャッシュ
- トランザクション終了時に自動的にクリーンアップ

#### キャッシュキー戦略
```go
type StatementCacheKey struct {
    SQL         string
    ContextType ContextType // DB または TX
    TxID        uintptr     // トランザクションポインタ（TXコンテキストのみ）
}

type ContextType int

const (
    DBContext ContextType = iota
    TxContext
)

func (k StatementCacheKey) String() string {
    if k.ContextType == DBContext {
        return fmt.Sprintf("db:%s", k.SQL)
    }
    return fmt.Sprintf("tx:%x:%s", k.TxID, k.SQL)
}
```

#### トランザクションライフサイクル管理
```go
// トランザクション終了時の自動クリーンアップ
func (e *Executor) handleTransactionEnd(tx *sql.Tx) {
    e.stmtCache.CleanupTx(tx)
}

// deferを使用した使用パターン
func executeInTransaction(db *sql.DB, executor *Executor) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer func() {
        executor.CleanupTransaction(tx) // キャッシュされたステートメントをクリーンアップ
        if err != nil {
            tx.Rollback()
        } else {
            tx.Commit()
        }
    }()
    
    // トランザクションコンテキストでエグゼキュータを使用
    _, err = executor.QueryContextTx(ctx, tx, "users", params)
    return err
}
```

### キャッシュ戦略詳細

#### 構造影響パラメータ
SQL構造に影響し、別々のキャッシュエントリが必要なパラメータ：
- テーブル名サフィックス（例：`table_suffix: "prod"` vs `table_suffix: "test"`）
- 条件付きフィールド包含（例：`include_email: true` vs `include_email: false`）
- SQL構造を変更する動的WHERE句条件
- ORDER BYフィールド選択
- 条件付きJOIN句

#### 非構造影響パラメータ
値のみに影響し、キャッシュされたSQLを再利用できるパラメータ：
- フィルタ値（例：`user_id: 123` vs `user_id: 456`）
- ページネーションパラメータ（LIMIT/OFFSET値）
- WHERE句の検索語
- 日付範囲やその他のフィルタ条件

#### キャッシュキー生成
```go
type CacheKeyBuilder struct{}

func (b *CacheKeyBuilder) BuildSQLCacheKey(templateName string, params map[string]any) string {
    structuralParams := b.extractStructuralParams(params)
    hash := b.hashParams(structuralParams)
    return fmt.Sprintf("%s:%s", templateName, hash)
}

func (b *CacheKeyBuilder) extractStructuralParams(params map[string]any) map[string]any {
    // SQL構造に影響するパラメータのみを抽出
    // 実装はテンプレートメタデータに依存
}
```

## API設計

### コアインターフェース

```go
package snapsqlgo

// DBExecutor represents the unified interface for database operations
type DBExecutor interface {
    PrepareConSQtext(ctx context.Context, query string) (*sql.Stmt, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// sql.DB、sql.Conn、sql.TxがDBExecutorを実装することを保証
var (
    _ DBExecutor = (*sql.DB)(nil)
    _ DBExecutor = (*sql.Conn)(nil)
    _ DBExecutor = (*sql.Tx)(nil)
)

// ContextType represents the type of database context
type ContextType int

const (
    DBContext ContextType = iota
    ConnContext
    TxContext
)

// ContextInfo holds information about the database context
type ContextInfo struct {
    Type     ContextType
    ID       uintptr // 実際のDB/Conn/Txインスタンスへのポインタ
    ParentID uintptr // Txの場合: 親Conn ID、Connの場合: 親DB ID
}
```

// Generator handles SQL generation from templates
type Generator struct {
    // private fields
}

func NewGenerator(templatesDir string) (*Generator, error)
func (g *Generator) GenerateSQL(templateName string, params map[string]any) (string, []any, error)

// Executor handles SQL execution with statement caching
type Executor struct {
    // private fields
}

func NewExecutor(cacheSize int) *Executor
func (e *Executor) QueryContext(ctx context.Context, db QueryExecutor, sql string, args []any) (*ResultIterator, error)
func (e *Executor) ExecContext(ctx context.Context, db ExecExecutor, sql string, args []any) (sql.Result, error)
func (e *Executor) Close() error

// ResultIterator provides Go 1.24 iterator access to query results
type ResultIterator struct {
    // private fields
}

func (r *ResultIterator) All() iter.Seq2[[]any, error]
func (r *ResultIterator) Columns() []string
func (r *ResultIterator) ColumnTypes() []*sql.ColumnType
func (r *ResultIterator) Close() error
```

### 使用例

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/shibukawa/snapsql/runtime/snapsqlgo"
)

func main() {
    // 集約キャッシュ設定でコンポーネントを初期化
    
    // オプション1: go:embed使用（プロダクションで最も一般的）
    //go:embed templates/*.json
    var templatesFS embed.FS
    loader := snapsqlgo.NewTemplateLoader(templatesFS)
    
    // オプション2: ファイルシステムディレクトリ使用（開発に便利）
    // loader := snapsqlgo.NewTemplateLoader(os.DirFS("./generated"))
    
    // オプション3: サブディレクトリ用のfs.Sub使用
    // subFS, _ := fs.Sub(templatesFS, "templates")
    // loader := snapsqlgo.NewTemplateLoader(subFS)
    
    // キャッシュサイズを集約設定
    config := snapsqlgo.SQLGeneratorConfig{
        TemplateCacheSize: 0,    // 無制限（埋め込みテンプレート数で制限）
        SQLCacheSize:      1000, // 1000のユニークSQL組み合わせ
        StmtCacheSize:     100,  // デフォルトステートメントキャッシュサイズ
    }
    generator := snapsqlgo.NewSQLGeneratorWithConfig(loader, config)
    ctx := context.Background()
    db, _ := sql.Open("postgres", "connection_string")
    
    // デフォルトのステートメントキャッシュサイズでエグゼキュータを作成
    dbExecutor := snapsqlgo.NewExecutor(generator, db)
    defer dbExecutor.Close()
    
    ctx := context.Background()
    db, _ := sql.Open("postgres", "connection_string")
    
    // 標準実行（プリペアドステートメント使用、推奨）
    results, err := dbExecutor.Query(ctx, "users", map[string]any{
        "include_email": true,        // 構造影響パラメータ
        "table_suffix": "prod",       // 構造影響パラメータ
        "active": true,               // 値パラメータ
        "department": "engineering",  // 値パラメータ
    })
    if err != nil {
        panic(err)
    }
    defer results.Close()
    
    // Go 1.24イテレータを使用した結果の反復処理
    for row, err := range results.All() {
        if err != nil {
            panic(err)
        }
        fmt.Printf("Row: %v\n", row)
    }
    
    // DDLや一回限りの操作のための直接実行
    _, err = dbExecutor.ExecDirect(ctx, "create_index", map[string]any{
        "table_name": "users",
        "index_name": "idx_users_email",
        "column": "email",
    })
    
    // トランザクション内で実行 - ステートメントはトランザクションごとにキャッシュ
    err = executeInTransaction(db, executor)
    if err != nil {
        panic(err)
    }
    
    // 必要時のキャッシュクリア（例：開発中）
    executor.ClearAllCaches()
}

func executeInTransaction(db *sql.DB, executor *snapsqlgo.Executor) error {
    ctx := context.Background()
    
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    
    defer func() {
        // トランザクション固有のキャッシュされたステートメントをクリーンアップ
        executor.CleanupTransaction(tx)
        if err != nil {
            tx.Rollback()
        } else {
            tx.Commit()
        }
    }()
    
    // クエリ実行 - ステートメントは接続ごとにキャッシュ、DBステートメント再利用可能
    results, err := connExecutor.Query(ctx, "orders", map[string]any{
        "user_id": 123,
        "status": "pending",
    })
    if err != nil {
        panic(err)
    }
    defer results.Close()
    
    for row, err := range results.All() {
        if err != nil {
            panic(err)
        }
        fmt.Printf("Order: %v\n", row)
    }
}

func executeInTransaction(ctx context.Context, db *sql.DB, generator *snapsqlgo.SQLGenerator) {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        panic(err)
    }
    
    // sql.Txでエグゼキュータを作成 - ジェネリック型推論
    txExecutor := snapsqlgo.NewExecutor(generator, tx, 25)
    
    defer func() {
        txExecutor.Close() // トランザクション固有のステートメントを自動クリーンアップ
        if err != nil {
            tx.Rollback()
        } else {
            tx.Commit()
        }
    }()
    
    // トランザクション内でクエリ実行 - ConnとDBステートメント再利用可能
    results, err := txExecutor.Query(ctx, "user_orders", map[string]any{
        "user_id": 456,
        "include_details": true,
    })
    if err != nil {
        return
    }
    defer results.Close()
    
    for row, err := range results.All() {
        if err != nil {
            return
        }
        fmt.Printf("User Order: %v\n", row)
    }
    
    // 同じトランザクション内で更新実行
    _, err = txExecutor.Exec(ctx, "update_order_status", map[string]any{
        "order_id": 789,
        "status": "processed",
    })
}

// 手動SQL生成の例（デバッグ用）
func debugSQLGeneration() {
    //go:embed templates/*.json
    var templatesFS embed.FS
    loader := snapsqlgo.NewTemplateLoader(templatesFS)
    generator := snapsqlgo.NewSQLGenerator(loader)
    
    sql, args, err := generator.GenerateSQL("users", map[string]any{
        "include_email": true,
        "active": true,
    })
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Generated SQL: %s\n", sql)
    fmt.Printf("Parameters: %v\n", args)
}
```

## エラーハンドリング

### センチネルエラー

```go
var (
    ErrTemplateNotFound     = errors.New("template not found")
    ErrInvalidParameters    = errors.New("invalid parameters")
    ErrSQLGeneration       = errors.New("SQL generation error")
    ErrStatementCache      = errors.New("statement cache error")
    ErrResultIteration     = errors.New("result iteration error")
)
```

### エラーラッピング

パラメータありのエラーはすべてセンチネルエラーをラップします：

```go
func (g *Generator) GenerateSQL(templateName string, params map[string]any) (string, []any, error) {
    template, exists := g.templateCache[templateName]
    if !exists {
        return "", nil, fmt.Errorf("%w: template '%s'", ErrTemplateNotFound, templateName)
    }
    // ... 残りの実装
}
```

## パフォーマンス考慮事項

### ジェネリクスによる階層キャッシュ戦略
1. **テンプレートキャッシュ**: ファイル名 → 解析済みテンプレート（インメモリ、永続的）
2. **SQLキャッシュ**: 構造影響パラメータ → 生成SQL（インメモリ、永続的）
3. **ステートメントキャッシュ**: 階層コンテキスト対応SQL → プリペアドステートメント
   - **DBコンテキスト**: グローバルキャッシュ、最高の再利用性、すべての接続間で共有
   - **Connコンテキスト**: 接続ごとキャッシュ、DBステートメント再利用可能、中程度の再利用性
   - **Txコンテキスト**: トランザクションごとキャッシュ、ConnとDBステートメント再利用可能、最低の再利用性

### ジェネリック型安全性とステートメント再利用性
- **型安全性**: ジェネリック`Executor[DB DBExecutor]`がコンパイル時型チェックを保証
- **自動コンテキスト検出**: ジェネリック型パラメータからのランタイムコンテキスト検出
- **階層フォールバック**: 下位レベルコンテキストが上位レベルキャッシュされたステートメントを再利用可能
- **最適再利用**: database/sql制約を尊重しながらステートメント再利用を最大化

### キャッシュ効率とステートメントライフサイクル
- テンプレート解析は高コスト、初回読み込み後は永続的にキャッシュ
- SQL生成は中程度のコスト、構造パラメータでキャッシュ
- **DBステートメント準備**: 高コスト、グローバルキャッシュ、最大再利用性
- **Connステートメント準備**: 高コスト、接続ごとキャッシュ、DBキャッシュにフォールバック可能
- **Txステートメント準備**: 高コスト、トランザクションごとキャッシュ、ConnとDBキャッシュにフォールバック可能
- 値のみのパラメータ変更は異なるバインド値でキャッシュされたステートメントを再利用

### コンテキスト対応メモリ管理
- テンプレートキャッシュ: 埋め込みテンプレート数で制限
- SQLキャッシュ: 構造パラメータのユニークな組み合わせで制限
- DBステートメントキャッシュ: 設定可能な最大サイズのLRU、アプリケーションライフタイム
- Connステートメントキャッシュ: 接続クローズ時に自動クリーンアップ
- Txステートメントキャッシュ: トランザクション終了時に自動クリーンアップ
- メモリ使用量を最小化するイテレータベースの結果処理

### ジェネリクスの利点
- **単一API**: 1つの`NewExecutor`関数がすべてのデータベースコンテキストタイプで動作
- **型推論**: Goコンパイラが正しいジェネリック型を自動推論
- **コンパイル時安全性**: 互換性のないデータベースコンテキストタイプの混合を防止
- **ランタイム効率**: データベース操作でのインターフェースボクシング/アンボクシングオーバーヘッドなし

### go:embedの利点
- 埋め込みFSを使用する場合のテンプレート読み込み時のランタイムファイルI/Oゼロ
- バイナリにテンプレートをバンドル、外部依存なし
- テンプレートファイル存在のコンパイル時検証
- 開発とテストシナリオのための柔軟なファイルシステムサポート

## Go 1.24イテレータ統合

### イテレータパターン実装

```go
// ResultIterator implements Go 1.24 iterator patterns
type ResultIterator struct {
    rows        *sql.Rows
    columns     []string
    columnTypes []*sql.ColumnType
    closed      bool
}

// All returns an iterator over all rows
func (r *ResultIterator) All() iter.Seq2[[]any, error] {
    return func(yield func([]any, error) bool) {
        defer r.Close()
        
        for r.rows.Next() {
            values := make([]any, len(r.columns))
            valuePtrs := make([]any, len(r.columns))
            for i := range values {
                valuePtrs[i] = &values[i]
            }
            
            if err := r.rows.Scan(valuePtrs...); err != nil {
                yield(nil, fmt.Errorf("%w: %v", ErrResultIteration, err))
                return
            }
            
            if !yield(values, nil) {
                return
            }
        }
        
        if err := r.rows.Err(); err != nil {
            yield(nil, fmt.Errorf("%w: %v", ErrResultIteration, err))
        }
    }
}
```

## テスト戦略

### 単体テスト
- 様々なシナリオでの埋め込みファイルからのテンプレート読み込み
- 異なるパラメータ組み合わせとキャッシュ動作でのSQL生成
- 構造パラメータ vs 値パラメータのキャッシュキー生成
- ステートメントキャッシュ操作（取得、設定、削除）
- イテレータ機能とリソースクリーンアップ
- キャッシュクリア操作
- エラーハンドリングシナリオ

### 統合テスト
- TestContainersを使用したエンドツーエンドSQL生成と実行
- 実際のデータベース接続でのマルチレベルキャッシュ
- 大きな結果セットでのイテレータパフォーマンス
- すべてのキャッシュレベルでの並行アクセスパターン
- キャッシュ無効化シナリオ

### テスト構造

```go
func TestSQLGenerator_GenerateSQL_Caching(t *testing.T) {
    ctx := t.Context()
    
    tests := []struct {
        name                    string
        templateName           string
        params1                map[string]any
        params2                map[string]any
        expectSameCachedSQL    bool
        expectedSQL            string
        expectedArgs1          []any
        expectedArgs2          []any
    }{
        {
            name:         "same_structural_params_different_values",
            templateName: "users",
            params1: map[string]any{
                "include_email": true,
                "active":       true,
                "user_id":      123,
            },
            params2: map[string]any{
                "include_email": true,
                "active":       false,
                "user_id":      456,
            },
            expectSameCachedSQL: true,
            expectedSQL:        "SELECT id, name, email FROM users WHERE active = ? AND user_id = ?",
            expectedArgs1:      []any{true, 123},
            expectedArgs2:      []any{false, 456},
        },
        {
            name:         "different_structural_params",
            templateName: "users",
            params1: map[string]any{
                "include_email": true,
                "active":       true,
            },
            params2: map[string]any{
                "include_email": false,
                "active":       true,
            },
            expectSameCachedSQL: false,
        },
        // その他のテストケース
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            loader := NewTemplateLoader()
            generator := NewSQLGenerator(loader)
            
            // 最初の呼び出し
            sql1, args1, err := generator.GenerateSQL(tt.templateName, tt.params1)
            assert.NoError(t, err)
            
            // 2回目の呼び出し
            sql2, args2, err := generator.GenerateSQL(tt.templateName, tt.params2)
            assert.NoError(t, err)
            
            if tt.expectSameCachedSQL {
                assert.Equal(t, sql1, sql2, "構造パラメータが同一の場合SQLは同じであるべき")
                assert.Equal(t, tt.expectedSQL, sql1)
                assert.Equal(t, tt.expectedArgs1, args1)
                assert.Equal(t, tt.expectedArgs2, args2)
            } else {
                assert.NotEqual(t, sql1, sql2, "構造パラメータが異なる場合SQLは異なるべき")
            }
        })
    }
}
```

## 実装計画

### フェーズ1: テンプレート読み込みと第1レベルキャッシュ
1. 中間JSONファイルのgo:embed統合実装
2. 埋め込みファイルからのテンプレート解析
3. 第1レベルキャッシュ（ファイル名 → テンプレート）
4. テンプレート読み込みとキャッシュの単体テスト

### フェーズ2: SQL生成と第2レベルキャッシュ
1. テンプレートASTからのSQL生成
2. 構造影響パラメータの識別
3. キャッシュキー生成と第2レベルキャッシュ
4. キャッシュされたSQLのパラメータ置換
5. SQL生成とキャッシュ動作の単体テスト

### フェーズ3: ステートメントキャッシュと実行
1. プリペアドステートメント用第3レベルLRUキャッシュ
2. 統合キャッシュ付きエグゼキュータ実装
3. Go 1.24イテレータベースの結果処理
4. 実際のデータベースでの統合テスト

### フェーズ4: キャッシュ管理と最適化
1. キャッシュクリアメカニズム
2. パフォーマンス最適化
3. メモリ使用量改善
4. 包括的ベンチマーク

## ディレクトリ構造

```
runtime/snapsqlgo/
├── templates/            # 埋め込み中間JSONファイル
│   ├── users.json
│   ├── orders.json
│   └── ...
├── loader.go            # go:embedでのテンプレート読み込み
├── generator.go         # 第2レベルキャッシュ付きSQL生成
├── cache.go             # ステートメントキャッシュ（第3レベル）
├── executor.go          # 全キャッシュ統合実行
├── iterator.go          # Go 1.24イテレータ実装
├── template.go          # テンプレート表現
├── cache_key.go         # キャッシュキー生成ロジック
├── errors.go            # センチネルエラー
├── loader_test.go       # テンプレートローダーテスト
├── generator_test.go    # SQLジェネレータテスト
├── cache_test.go        # キャッシュテスト
├── executor_test.go     # エグゼキュータテスト
├── iterator_test.go     # イテレータテスト
└── integration_test.go  # エンドツーエンド統合テスト
```

## 参考資料

- [SnapSQL README](../README.md)
- [中間フォーマットスキーマ](../intermediate-format-schema.json)
- [コーディング標準](../coding-standard.md)

// 異なるファイルシステム実装の例
func demonstrateFilesystemOptions() {
    // 1. go:embed（プロダクションデプロイメント）
    //go:embed templates/*.json
    var embeddedFS embed.FS
    loader1 := snapsqlgo.NewTemplateLoader(embeddedFS)
    
    // 2. ディレクトリファイルシステム（開発）
    dirFS := os.DirFS("./generated")
    loader2 := snapsqlgo.NewTemplateLoader(dirFS)
    
    // 3. 埋め込みファイルシステムのサブディレクトリ
    //go:embed assets/templates/*.json
    var assetsFS embed.FS
    subFS, err := fs.Sub(assetsFS, "assets/templates")
    if err != nil {
        panic(err)
    }
    loader3 := snapsqlgo.NewTemplateLoader(subFS)
    
    // 4. カスタムファイルシステム実装（例：データベース、ネットワークなど）
    // customFS := &MyCustomFS{...}
    // loader4 := snapsqlgo.NewTemplateLoader(customFS)
    
    // すべてのローダーはファイルシステム実装に関係なく同じように動作
    generator1 := snapsqlgo.NewSQLGenerator(loader1)
    generator2 := snapsqlgo.NewSQLGenerator(loader2)
    generator3 := snapsqlgo.NewSQLGenerator(loader3)
    
    // ジェネレータを通常通り使用
    _ = generator1
    _ = generator2
    _ = generator3
}
