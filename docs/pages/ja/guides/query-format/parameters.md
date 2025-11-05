# パラメータ

パラメータはクエリテンプレートに渡される外部データです。YAML形式で**型のみ**を定義し、テンプレート内では `/*= param_name */` のように参照します。

## 基本的な定義

### シンプルな例

```yaml
user_id: int
status: string
email: string
```

**重要**: SnapSQLのパラメータ定義では、型のみをシンプルに記述します。`type:`や`description:`などのキーは不要です。

### ネストしたパラメータ

```yaml
user_id: int
filters:
  active: bool
  departments: [string]
pagination:
  limit: int
  offset: int
```

## 対応している型

### 基本型

| SnapSQL型 | 説明 | Go型 | 使用例 |
|-----------|------|------|--------|
| `int` | 整数 | `int` | ユーザーID、カウント |
| `int32` | 32ビット整数 | `int32` | 小さい範囲の整数 |
| `int64` | 64ビット整数 | `int64` | 大きい範囲の整数 |
| `string` | 文字列 | `string` | 名前、メール、テキスト |
| `bool` | 真偽値 | `bool` | フラグ、有効/無効 |
| `float64` | 浮動小数点数 | `float64` | スコア、割合 |
| `decimal` | 高精度小数 | `decimal.Decimal` | 金額、精度が必要な数値 |
| `timestamp` | 日時 | `time.Time` | 作成日時、更新日時 |
| `date` | 日付のみ | `time.Time` | 誕生日、開始日 |
| `time` | 時刻のみ | `time.Time` | 営業開始時刻 |
| `bytes` | バイナリデータ | `[]byte` | 画像、ファイル |
| `any` | 任意の型 | `any` | 汎用データ |

### NULL許容型

ポインタ型を使用してNULL許容にできます：

```yaml
email:
  type: "*string"
  description: メールアドレス（オプション）

age:
  type: "*int"
  description: 年齢（未入力可）
```

Go生成コードでは`*string`, `*int`などのポインタ型になります。

### 配列型

配列は`[]`プレフィックスで表現：

```yaml
user_ids:
  type: "[]int"
  description: ユーザーIDのリスト

tags:
  type: "[]string"
  description: タグのリスト
```

テンプレートではFORループで使用：

```sql
SELECT * FROM users
WHERE id IN (
  /*# for id in user_ids */
  /*= id */0
  /*# if not loop.last */,/*# end */
  /*# end */
)
```

### オブジェクト型

構造化されたデータは配列+オブジェクトで表現：

```yaml
filters:
  type: "[]object"
  description: フィルタ条件のリスト
  properties:
    field:
      type: string
    operator:
      type: string
    value:
      type: any
```

使用例：

```sql
SELECT * FROM products
WHERE 1=1
/*# for filter in filters */
  /*# if filter.operator == "eq" */
  AND /*= filter.field */'name' = /*= filter.value */'example'
  /*# end */
  /*# if filter.operator == "gt" */
  AND /*= filter.field */'price' > /*= filter.value */0
  /*# end */
/*# end */
```

## テンプレート内での参照

### 基本的な参照

```sql
WHERE id = /*= user_id */1
```

### 式を使った参照

```sql
WHERE 
  status = /*= status */'active'
  AND created_at >= /*= created_at */'2024-01-01'
  /*# if age_min != 0 */
  AND age >= /*= age_min */18
  /*# end */
```

### ネストしたフィールド

```sql
WHERE 
  department = /*= user.department */'sales'
  AND role = /*= user.role */'manager'
```

## バリデーション

パラメータのバリデーションは実行前に行われます（計画中機能）。

詳細は設計ドキュメント参照：
- `designdocs/20250912-params-preflight-validation.ja.md`

## 共通型の再利用

共通的に使う型は別ファイルで定義して再利用できます（計画中機能）：

```yaml
# common_types.yaml
User:
  type: object
  properties:
    id: int
    name: string
    email: string
```

```yaml
# query.snap.md の Parameters セクション
user:
  type: "../common_types.yaml#User"
  description: ユーザー情報
```

## Go生成コードでの使用

パラメータは関数の引数として生成されます：

```go
// パラメータ定義
// user_id: int
// status: string

func FindUsers(ctx context.Context, db *sql.DB, userId int, status string) ([]User, error) {
    // ...
}
```

配列やオブジェクトの場合：

```go
// パラメータ定義
// user_ids: []int
// filters: []object

type Filter struct {
    Field    string
    Operator string
    Value    any
}

func FindProducts(ctx context.Context, db *sql.DB, userIds []int, filters []Filter) ([]Product, error) {
    // ...
}
```

## テストでのパラメータ指定

テストケース内でパラメータを指定：

```yaml
## Test Cases

### Test Case 1: 単一ユーザー検索

**Parameters:**
```yaml
user_id: 123
status: "active"
```

**Expected Results:**
```yaml
- id: 123
  name: "John Doe"
  status: "active"
```
```

## ベストプラクティス

### 1. 型を明示する

```yaml
# ✅ Good: 型が明確
user_id: int

# ❌ Bad: 型が不明
user_id: {}
```

### 2. 説明を追加する

```yaml
# ✅ Good: 説明付き
user_id:
  type: int
  description: 検索対象のユーザーID

# ❌ Bad: 説明なし
user_id: int
```

### 3. NULL許容性を明確にする

```yaml
# ✅ Good: NULL許容
email:
  type: "*string"
  description: メールアドレス（オプション）

# ✅ Good: NOT NULL
user_id:
  type: int
  description: ユーザーID（必須）
```

### 4. 配列の空チェック

```sql
-- ✅ Good: 空配列チェック
/*# if size(user_ids) > 0 */
WHERE id IN (/*# for id in user_ids */ /*= id */0 /*# if not loop.last */,/*# end */ /*# end */)
/*# end */

-- ❌ Bad: 空配列でエラー
WHERE id IN (/*# for id in user_ids */ /*= id */0 /*# end */)
```

## 関連ドキュメント

- [テンプレート構文](./template-syntax.md) - パラメータの使い方
- [共通型](./common-types.md) - 型の詳細
- [Goリファレンス](../language-reference/go.md) - Go生成コードでの使い方
- [テスト概要](./test-overview.md) - テストでのパラメータ指定
