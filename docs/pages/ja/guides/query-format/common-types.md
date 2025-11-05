# 共通型

このページでは、SnapSQLで使用できる共通の型と、それらがどのようにマッピングされるかを説明します。

## 基本型

### 整数型

| SnapSQL型 | 説明 | Go型 | PostgreSQL型 | MySQL型 |
|-----------|------|------|--------------|---------|
| `int` | 汎用整数 | `int` | `INTEGER` | `INT` |
| `int32` | 32ビット整数 | `int32` | `INTEGER` | `INT` |
| `int64` | 64ビット整数 | `int64` | `BIGINT` | `BIGINT` |

使用例：

```yaml
# パラメータ定義
user_id: int
count: int32
big_number: int64
```

```yaml
# レスポンス定義
properties:
  id: int
  age: int32
  population: int64
```

### 文字列型

| SnapSQL型 | 説明 | Go型 | PostgreSQL型 | MySQL型 |
|-----------|------|------|--------------|---------|
| `string` | 可変長文字列 | `string` | `TEXT` or `VARCHAR` | `TEXT` or `VARCHAR` |

使用例：

```yaml
# パラメータ定義
name: string
email: string

# レスポンス定義
properties:
  name:
    type: string
    max_length: 100
  description:
    type: string
```

### 真偽値型

| SnapSQL型 | 説明 | Go型 | PostgreSQL型 | MySQL型 |
|-----------|------|------|--------------|---------|
| `bool` | 真偽値 | `bool` | `BOOLEAN` | `TINYINT(1)` |

使用例：

```yaml
# パラメータ定義
is_active: bool
has_permission: bool

# レスポンス定義
properties:
  active: bool
  deleted: bool
```

### 浮動小数点型

| SnapSQL型 | 説明 | Go型 | PostgreSQL型 | MySQL型 |
|-----------|------|------|--------------|---------|
| `float32` | 単精度浮動小数点 | `float32` | `REAL` | `FLOAT` |
| `float64` | 倍精度浮動小数点 | `float64` | `DOUBLE PRECISION` | `DOUBLE` |

使用例：

```yaml
# パラメータ定義
score: float64
ratio: float32

# レスポンス定義
properties:
  rating: float64
  percentage: float32
```

### 高精度小数型

| SnapSQL型 | 説明 | Go型 | PostgreSQL型 | MySQL型 |
|-----------|------|------|--------------|---------|
| `decimal` | 高精度小数 | `decimal.Decimal` | `NUMERIC` | `DECIMAL` |

**注意**: Goでは`github.com/shopspring/decimal`パッケージが必要です。

使用例：

```yaml
# パラメータ定義
price:
  type: decimal
  precision: 10
  scale: 2

# レスポンス定義
properties:
  amount:
    type: decimal
    precision: 10
    scale: 2
  tax:
    type: decimal
    precision: 10
    scale: 2
```

Go使用例：

```go
import "github.com/shopspring/decimal"

type Product struct {
    ID     int             `json:"id"`
    Price  decimal.Decimal `json:"price"`
    Amount decimal.Decimal `json:"amount"`
}

// 計算例
total := product.Price.Add(product.Tax)
```

## 日時型

### タイムスタンプ型

| SnapSQL型 | 説明 | Go型 | PostgreSQL型 | MySQL型 |
|-----------|------|------|--------------|---------|
| `timestamp` | 日時（推奨） | `time.Time` | `TIMESTAMP` | `TIMESTAMP` |
| `datetime` | 日時（非推奨） | `time.Time` | `TIMESTAMP` | `DATETIME` |
| `date` | 日付のみ | `time.Time` | `DATE` | `DATE` |
| `time` | 時刻のみ | `time.Time` | `TIME` | `TIME` |

**注意**: `datetime`は`timestamp`に統一されました。既存コードでは互換性のため使用可能です。

使用例：

```yaml
# パラメータ定義
created_after: timestamp
birth_date: date
business_hours_start: time

# レスポンス定義
properties:
  created_at: timestamp
  updated_at: timestamp
  deleted_at:
    type: timestamp
    is_nullable: true
  birth_date: date
```

Go使用例：

```go
import "time"

type User struct {
    ID        int       `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// 使用例
user := User{
    ID:        1,
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
}
```

## NULL許容型

ポインタ型を使用してNULL許容にできます。

### NULL許容の基本型

```yaml
# パラメータ定義
email:
  type: "*string"
age:
  type: "*int"
score:
  type: "*float64"

# レスポンス定義
properties:
  email:
    type: string
    is_nullable: true
  age:
    type: int
    is_nullable: true
  deleted_at:
    type: timestamp
    is_nullable: true
```

Go使用例：

```go
type User struct {
    ID        int        `json:"id"`
    Email     *string    `json:"email"`      // NULL許容
    Age       *int       `json:"age"`        // NULL許容
    DeletedAt *time.Time `json:"deleted_at"` // NULL許容
}

// 使用例
email := "test@example.com"
user := User{
    ID:    1,
    Email: &email,  // 値がある場合
    Age:   nil,     // NULLの場合
}

// 値の取り出し
if user.Email != nil {
    fmt.Println("Email:", *user.Email)
}
```

## 配列型

配列は`[]`プレフィックスで表現します。

```yaml
# パラメータ定義
user_ids: "[]int"
tags: "[]string"
scores: "[]float64"

# 使用例
departments:
  type: "[]object"
  properties:
    id: int
    name: string
```

Go使用例：

```go
// パラメータとして使用
userIds := []int{1, 2, 3}
tags := []string{"tag1", "tag2"}

// レスポンスとして使用
type ListUsersResponse []User
```

## バイナリ型

| SnapSQL型 | 説明 | Go型 | PostgreSQL型 | MySQL型 |
|-----------|------|------|--------------|---------|
| `bytes` | バイナリデータ | `[]byte` | `BYTEA` | `BLOB` |

使用例：

```yaml
# パラメータ定義
image_data: bytes
file_content: bytes

# レスポンス定義
properties:
  thumbnail: bytes
  attachment: bytes
```

## 任意型

| SnapSQL型 | 説明 | Go型 | 使用例 |
|-----------|------|------|--------|
| `any` | 任意の型 | `any` (interface{}) | JSON、動的データ |

使用例：

```yaml
# パラメータ定義
metadata: any
config: any

# レスポンス定義
properties:
  settings: any
  extra_data: any
```

Go使用例：

```go
type Document struct {
    ID       int    `json:"id"`
    Metadata any    `json:"metadata"`  // 任意の型
}

// JSON形式のデータを格納
metadata := map[string]interface{}{
    "author": "John",
    "tags":   []string{"tech", "golang"},
}
```

## 型のマッピング一覧

### SnapSQL → Go

| SnapSQL | Go | 備考 |
|---------|-----|------|
| `int` | `int` | |
| `int32` | `int32` | |
| `int64` | `int64` | |
| `string` | `string` | |
| `bool` | `bool` | |
| `float32` | `float32` | |
| `float64` | `float64` | |
| `decimal` | `decimal.Decimal` | `github.com/shopspring/decimal` |
| `timestamp` | `time.Time` | |
| `datetime` | `time.Time` | `timestamp`推奨 |
| `date` | `time.Time` | |
| `time` | `time.Time` | |
| `bytes` | `[]byte` | |
| `any` | `any` (interface{}) | |
| `*T` | `*T` | NULL許容 |
| `[]T` | `[]T` | 配列 |

### Go → PostgreSQL

| Go | PostgreSQL | 備考 |
|----|------------|------|
| `int`, `int32` | `INTEGER` | |
| `int64` | `BIGINT` | |
| `string` | `TEXT` or `VARCHAR(n)` | |
| `bool` | `BOOLEAN` | |
| `float32` | `REAL` | |
| `float64` | `DOUBLE PRECISION` | |
| `decimal.Decimal` | `NUMERIC(p,s)` | |
| `time.Time` | `TIMESTAMP` or `DATE` | |
| `[]byte` | `BYTEA` | |

### Go → MySQL

| Go | MySQL | 備考 |
|----|-------|------|
| `int`, `int32` | `INT` | |
| `int64` | `BIGINT` | |
| `string` | `TEXT` or `VARCHAR(n)` | |
| `bool` | `TINYINT(1)` | |
| `float32` | `FLOAT` | |
| `float64` | `DOUBLE` | |
| `decimal.Decimal` | `DECIMAL(p,s)` | |
| `time.Time` | `TIMESTAMP` or `DATETIME` | |
| `[]byte` | `BLOB` | |

## カスタム型（計画中機能）

共通的に使う型定義を別ファイルで管理できます：

```yaml
# common_types.yaml
User:
  type: object
  properties:
    id: int
    name: string
    email: string

Address:
  type: object
  properties:
    zip_code: string
    prefecture: string
    city: string
```

使用例：

```yaml
# query.snap.md
## Parameters

```yaml
user:
  type: "../common_types.yaml#User"
address:
  type: "../common_types.yaml#Address"
```
```

## ベストプラクティス

### 1. 適切な型を選ぶ

```yaml
# ✅ Good: 金額はdecimal
price:
  type: decimal
  precision: 10
  scale: 2

# ❌ Bad: 金額にfloat（精度が失われる）
price: float64
```

### 2. NULL許容性を明確にする

```yaml
# ✅ Good: NULL許容を明示
email:
  type: string
  is_nullable: true

# ❌ Bad: 不明確
email: string
```

### 3. 制約を指定する

```yaml
# ✅ Good: 制約付き
name:
  type: string
  max_length: 100

price:
  type: decimal
  precision: 10
  scale: 2

# ❌ Bad: 制約なし
name: string
price: decimal
```

### 4. timestampを優先する

```yaml
# ✅ Good: timestamp使用
created_at: timestamp

# ⚠️ Warning: datetime（互換性のため残存）
created_at: datetime
```

## 依存パッケージ

### Go言語

生成されるコードで必要になる外部パッケージ：

```go
import (
    "time"                                    // 標準ライブラリ
    "github.com/shopspring/decimal"          // decimal型
)
```

インストール：

```bash
go get github.com/shopspring/decimal
```

## 関連ドキュメント

- [パラメータ](./parameters.md) - パラメータでの型の使い方
- [レスポンス型](./response-types.md) - レスポンスでの型の使い方
- [Goリファレンス](../language-reference/go.md) - Go生成コードの詳細
- [テンプレート構文](./template-syntax.md) - テンプレートでの型の扱い
