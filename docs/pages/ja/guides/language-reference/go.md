# Go 言語リファレンス

このページでは、SnapSQL で生成された Go コードの使い方を説明します。

## セットアップ

### インストール

```bash
go get github.com/shibukawa/snapsql
```

### コード生成

```bash
snapsql generate --lang go --output queries queries/*.snap.md
```

生成されるファイル：

```
queries/
├── queries.go              # 生成されたクエリコード
└── queries_test.go         # テストコード（オプション）
```

## 基本的な使い方

### クエリの実行

```go
package main

import (
    "context"
    "database/sql"
    "log"
    
    _ "github.com/lib/pq"
    "yourproject/queries"
)

func main() {
    db, err := sql.Open("postgres", "postgres://localhost/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    ctx := context.Background()
    
    // ユーザーを取得
    user, err := queries.GetUserByID(ctx, db, 1)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("User: %+v", user)
}
```

### パラメータの指定

生成されたコードは型安全です：

```go
// パラメータを構造体で指定
params := queries.CreateUserParams{
    Name:  "太郎",
    Email: "taro@example.com",
    Age:   30,
}

user, err := queries.CreateUser(ctx, db, params)
```

### 複数行の結果

```go
// リストを取得
users, err := queries.ListUsers(ctx, db, queries.ListUsersParams{
    MinAge: 20,
    Limit:  10,
})

for _, user := range users {
    log.Printf("User: %+v", user)
}
```

## システムカラムの使い方

システムカラムは `context.Context` 経由で渡します：

```go
import "github.com/shibukawa/snapsql/runtime"

// コンテキストにユーザーIDを設定
ctx = runtime.WithUserID(ctx, 42)
ctx = runtime.WithTimezone(ctx, "Asia/Tokyo")

// クエリ実行時に自動的に適用される
task, err := queries.CreateTask(ctx, db, queries.CreateTaskParams{
    Title: "新しいタスク",
})
// created_by, created_at などが自動的に設定される
```

### カスタムシステムカラム

```go
// カスタムヘルパーを定義
func WithTenantID(ctx context.Context, tenantID int64) context.Context {
    return context.WithValue(ctx, "tenant_id", tenantID)
}

ctx = WithTenantID(ctx, 123)
```

## トランザクション

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    log.Fatal(err)
}
defer tx.Rollback()

// トランザクション内でクエリを実行
user, err := queries.CreateUser(ctx, tx, params)
if err != nil {
    return err
}

task, err := queries.CreateTask(ctx, tx, taskParams)
if err != nil {
    return err
}

// コミット
if err := tx.Commit(); err != nil {
    return err
}
```

## モック機能

テスト時にはモックを使用できます：

```go
package main_test

import (
    "context"
    "testing"
    
    "github.com/DATA-DOG/go-sqlmock"
    "yourproject/queries"
)

func TestGetUser(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()
    
    // 期待される結果を設定
    rows := sqlmock.NewRows([]string{"id", "name", "email"}).
        AddRow(1, "太郎", "taro@example.com")
    
    mock.ExpectQuery("SELECT id, name, email FROM users").
        WithArgs(1).
        WillReturnRows(rows)
    
    // クエリを実行
    ctx := context.Background()
    user, err := queries.GetUserByID(ctx, db, 1)
    if err != nil {
        t.Fatal(err)
    }
    
    // 検証
    if user.Name != "太郎" {
        t.Errorf("expected 太郎, got %s", user.Name)
    }
    
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("unfulfilled expectations: %s", err)
    }
}
```

## 生成されるコードの構造

### クエリ関数

```go
// 単一行を返すクエリ
func GetUserByID(ctx context.Context, db DB, userID int64) (User, error)

// 複数行を返すクエリ
func ListUsers(ctx context.Context, db DB, params ListUsersParams) ([]User, error)

// 実行のみ（結果なし）
func DeleteUser(ctx context.Context, db DB, userID int64) error
```

### パラメータ構造体

```go
type CreateUserParams struct {
    Name  string
    Email string
    Age   int
}
```

### レスポンス構造体

```go
type User struct {
    ID    int64
    Name  string
    Email string
}
```

### ネストした構造

```go
type Task struct {
    ID          int64
    Title       string
    AssignedTo  *User  // ポインタ = NULL 許可
    Comments    []Comment
}
```

## エラーハンドリング

```go
user, err := queries.GetUserByID(ctx, db, 999)
if err == sql.ErrNoRows {
    // レコードが見つからない
    log.Println("User not found")
} else if err != nil {
    // その他のエラー
    log.Printf("Error: %v", err)
}
```

## パフォーマンス最適化

### プリペアドステートメント

```go
// 繰り返し実行する場合はプリペアドステートメントを使用
stmt, err := db.PrepareContext(ctx, queries.GetUserByIDSQL)
if err != nil {
    log.Fatal(err)
}
defer stmt.Close()

for _, id := range userIDs {
    var user User
    err := stmt.QueryRowContext(ctx, id).Scan(&user.ID, &user.Name, &user.Email)
    // ...
}
```

### バッチ処理

```go
// 一括挿入
users := []queries.CreateUserParams{
    {Name: "太郎", Email: "taro@example.com"},
    {Name: "花子", Email: "hanako@example.com"},
}

for _, u := range users {
    _, err := queries.CreateUser(ctx, tx, u)
    if err != nil {
        return err
    }
}
```

## 関連ドキュメント

- [システムカラム](../user-reference/system-columns.md)
- [モック機能](../user-reference/mock.md)
- [トランザクション](../user-reference/transactions.md)
