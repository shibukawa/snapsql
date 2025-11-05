# レスポンス型

このページでは、クエリが返すレスポンスの型定義とその使い方を説明します。

## Response Affinityとは

**Response Affinity**は、クエリが返すデータの形式を示します：

| Affinity | 説明 | 使用例 | Go生成型 |
|----------|------|--------|----------|
| `one` | 単一レコード | 主キーでの取得 | `*Struct` |
| `many` | 複数レコード | リスト取得 | `[]Struct` |
| `none` | 行を返さない | INSERT/UPDATE/DELETE | `sql.Result` or `int64` |

### フロントマターでの指定

````markdown
---
function_name: find_user
response_affinity: one
---
````

## 基本的なレスポンス定義

### 単一レコード（one）

主キーでの取得など、1件だけ返すクエリ：

````markdown
---
function_name: get_user_by_id
response_affinity: one
---

## Response

```yaml
type: object
properties:
  id:
    type: int
  name:
    type: string
  email:
    type: string
  created_at:
    type: timestamp
```
````

生成されるGoコード：

```go
type GetUserByIDResponse struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func GetUserByID(ctx context.Context, db *sql.DB, userId int) (*GetUserByIDResponse, error) {
    // ...
}
```

### 複数レコード（many）

リスト取得など、複数行を返すクエリ：

````markdown
---
function_name: list_users
response_affinity: many
---

## Response

```yaml
type: object
properties:
  id: int
  name: string
  status: string
```
````

生成されるGoコード：

```go
type ListUsersResponse struct {
    ID     int    `json:"id"`
    Name   string `json:"name"`
    Status string `json:"status"`
}

func ListUsers(ctx context.Context, db *sql.DB) ([]ListUsersResponse, error) {
    // ...
}
```

### 行を返さない（none）

INSERT/UPDATE/DELETEなど、結果を返さないクエリ：

````markdown
---
function_name: delete_user
response_affinity: none
---

## SQL

```sql
DELETE FROM users WHERE id = /*= user_id */0
```
````

生成されるGoコード：

```go
func DeleteUser(ctx context.Context, db *sql.DB, userId int) (int64, error) {
    // 影響行数を返す
}
```

## レスポンスフィールドの型

### 基本型

```yaml
properties:
  id:
    type: int
  name:
    type: string
  age:
    type: int
  score:
    type: float64
  active:
    type: bool
  amount:
    type: decimal
  created_at:
    type: timestamp
```

### NULL許容型

NULL許容フィールドは`is_nullable: true`を指定：

```yaml
properties:
  email:
    type: string
    is_nullable: true
  deleted_at:
    type: timestamp
    is_nullable: true
```

生成されるGoコード：

```go
type Response struct {
    Email     *string    `json:"email"`      // ポインタ型
    DeletedAt *time.Time `json:"deleted_at"` // ポインタ型
}
```

### 制約情報

```yaml
properties:
  name:
    type: string
    max_length: 100
  age:
    type: int
    precision: 3
  price:
    type: decimal
    precision: 10
    scale: 2
```

## 階層化されたレスポンス

### ネストしたオブジェクト

`parent__child`のようなフィールド名で階層を表現：

````markdown
## SQL

```sql
SELECT
    b.id,
    b.name,
    l.id AS lists__id,
    l.name AS lists__name
FROM boards b
LEFT JOIN lists l ON l.board_id = b.id
WHERE b.id = /*= board_id */1
```

## Response

```yaml
type: object
properties:
  id: int
  name: string
  lists:
    type: array
    items:
      type: object
      properties:
        id: int
        name: string
```
````

生成されるGoコード：

```go
type BoardList struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type GetBoardWithListsResponse struct {
    ID    int         `json:"id"`
    Name  string      `json:"name"`
    Lists []BoardList `json:"lists"`
}
```

### 多段階のネスト

`parent__child__grandchild`で3階層以上のネスト：

````markdown
## SQL

```sql
SELECT
    b.id,
    b.name,
    l.id AS lists__id,
    l.name AS lists__name,
    c.id AS lists__cards__id,
    c.title AS lists__cards__title
FROM boards b
LEFT JOIN lists l ON l.board_id = b.id
LEFT JOIN cards c ON c.list_id = l.id
WHERE b.id = /*= board_id */1
```
````

詳細は[ネストしたレスポンス](./nested-response.md)を参照してください。

## RETURNING句の使用

PostgreSQLのRETURNING句で挿入/更新したデータを返す：

````markdown
---
function_name: create_user
response_affinity: one
---

## SQL

```sql
INSERT INTO users (name, email, created_at)
VALUES (
    /*= name */'',
    /*= email */'',
    CURRENT_TIMESTAMP
)
RETURNING id, name, email, created_at
```

## Response

```yaml
type: object
properties:
  id: int
  name: string
  email: string
  created_at: timestamp
```
````

**注意**: RETURNINGのサポートはDBMSに依存します：
- ✅ PostgreSQL: フルサポート
- ⚠️ MySQL 8.0+: 限定的なサポート
- ❌ SQLite: 限定的なサポート（3.35.0+）

## 実際の使用例

### 単一ユーザー取得

````markdown
---
function_name: get_user
response_affinity: one
---

## SQL

```sql
SELECT id, name, email, age, created_at
FROM users
WHERE id = /*= user_id */1
```

## Response

```yaml
type: object
properties:
  id: int
  name: string
  email: string
  age: int
  created_at: timestamp
```
````

使用例：

```go
user, err := GetUser(ctx, db, 123)
if err != nil {
    return err
}
if user == nil {
    return errors.New("user not found")
}
fmt.Printf("User: %s (%s)\n", user.Name, user.Email)
```

### ユーザーリスト取得

````markdown
---
function_name: list_users
response_affinity: many
---

## SQL

```sql
SELECT id, name, email, status
FROM users
WHERE status = /*= status */'active'
ORDER BY created_at DESC
LIMIT /*= limit */10
```

## Response

```yaml
type: object
properties:
  id: int
  name: string
  email: string
  status: string
```
````

使用例：

```go
users, err := ListUsers(ctx, db, "active", 10)
if err != nil {
    return err
}
for _, user := range users {
    fmt.Printf("- %s (%s)\n", user.Name, user.Email)
}
```

### ユーザー削除

````markdown
---
function_name: delete_user
response_affinity: none
---

## SQL

```sql
DELETE FROM users
WHERE id = /*= user_id */0
```
````

使用例：

```go
rowsAffected, err := DeleteUser(ctx, db, 123)
if err != nil {
    return err
}
if rowsAffected == 0 {
    return errors.New("user not found")
}
fmt.Printf("Deleted %d rows\n", rowsAffected)
```

## ベストプラクティス

### 1. Response Affinityを明示する

```yaml
# ✅ Good: 明示的
---
response_affinity: one
---

# ❌ Bad: 省略（推論に依存）
---
# response_affinityなし
---
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

### 3. 階層構造は主キーを含める

```yaml
# ✅ Good: 各階層に主キー
lists:
  type: array
  items:
    properties:
      id: int  # 主キー
      name: string

# ❌ Bad: 主キーなし（重複排除できない）
lists:
  type: array
  items:
    properties:
      name: string
```

### 4. RETURNING句の互換性を確認

```sql
-- ✅ Good: PostgreSQL用と明記
-- @dialect: postgresql
INSERT INTO users (...) RETURNING *

-- ✅ Good: MySQL用の代替
-- @dialect: mysql
INSERT INTO users (...)
-- 別のSELECTで取得
```

## トラブルシューティング

### レコードが見つからない場合

**単一レコード（one）の場合**：
- `nil`を返すか`ErrNotFound`を返すかは実装に依存
- アプリケーション側で統一する

```go
user, err := GetUser(ctx, db, 999)
if err != nil {
    return err
}
if user == nil {
    // 見つからなかった場合の処理
}
```

### 空の結果セット

**複数レコード（many）の場合**：
- 空のスライス`[]Response{}`を返す
- `nil`は返さない（nilチェック不要）

```go
users, err := ListUsers(ctx, db, "inactive", 10)
if err != nil {
    return err
}
if len(users) == 0 {
    // 空の場合の処理
}
```

### 階層構造が正しく生成されない

- フィールド名が`parent__child`形式になっているか確認
- 各階層に主キーが含まれているか確認
- `snapsql inspect`で中間形式を確認

## 関連ドキュメント

- [ネストしたレスポンス](./nested-response.md) - 階層化の詳細
- [Markdownフォーマット](./markdown-format.md) - フロントマターの書き方
- [パラメータ](./parameters.md) - パラメータ定義
- [Goリファレンス](../language-reference/go.md) - Go生成コードの使い方
