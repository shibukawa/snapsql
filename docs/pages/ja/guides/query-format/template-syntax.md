# テンプレート構文

このページはクエリテンプレートで利用できる構文の参照ページです。

## 概要

SnapSQLは**two-way SQL**（コメントベースのディレクティブ）を採用しています。通常のSQLとしてValidなテンプレート表現であるため、そのままSQLクライアントに渡して実行したりできます。

- **変数展開**: `/*= expression */[dummy_value]` の書式
- **制御構造**: `/*# if ... */`, `/*# for ... */` と `/*# end */`
- **表現言語**: [CEL (Common Expression Language)](https://cel.dev/?hl=ja) を使用

プログラミング言語向けのクライアントコードに変換したり、する際は、出力先のデータベースに合わせて出力されるSQLが変わります。[データベース方言](../user-reference/dialects.md)のページを参照してください。

## 変数埋め込み

### 基本的な変数展開

```sql
SELECT id, name
FROM users
WHERE status = /*= status */'active'
```

- `/*= status */` でパラメータを展開
- `'active'` はダミー値（直接SQLとして実行する際のプレースホルダ）。デフォルト値としては使われません。

### 複数の変数

```sql
SELECT id, name, email
FROM users
WHERE 
  status = /*= status */'active'
  AND age >= /*= min_age */18
  AND created_at >= /*= created_after */'2024-01-01'
```

### 式を使った展開

```sql
SELECT 
  id,
  /*= display_full_name ? "CONCAT(first_name, ' ', last_name)" : "username" */username AS display_name
FROM users
```

## 条件分岐（IF）

### 基本的なIF文

```sql
SELECT id, name
FROM users
WHERE status = 'active'
  /*# if created_after != "" */
  AND created_at >= /*= created_after */'2024-01-01'
  /*# end */
```

### 複数条件のIF

```sql
SELECT id, name, email
FROM users
WHERE 1=1
  /*# if status != "" */
  AND status = /*= status */'active'
  /*# end */
  /*# if age_min != 0 */
  AND age >= /*= age_min */18
  /*# end */
  /*# if age_max != 0 */
  AND age <= /*= age_max */100
  /*# end */
```

### ネストしたIF

```sql
SELECT id, name
FROM users
WHERE status = 'active'
  /*# if filters != null */
    /*# if filters.department != "" */
    AND department = /*= filters.department */'sales'
    /*# end */
    /*# if filters.role != "" */
    AND role = /*= filters.role */'manager'
    /*# end */
  /*# end */
```

### 複雑な条件式

```sql
SELECT id, name
FROM users
WHERE 
  /*# if start_date != "" && end_date != "" */
  created_at BETWEEN /*= start_date */'2023-01-01' AND /*= end_date */'2023-12-31'
  /*# end */
  /*# if sort_field != "" */
ORDER BY /*= sort_field + " " + (sort_direction != "" ? sort_direction : "ASC") */name ASC
  /*# end */
```

## ループ（FOR）

### 基本的なFORループ

```sql
SELECT * FROM products
WHERE id IN (
  /*# for id in product_ids */
  /*= id */0
  /*# if not loop.last */,/*# end */
  /*# end */
)
```

- `loop.last` でループの最後の要素を判定
- カンマの自動管理に利用

### バルクインサート

```sql
INSERT INTO users (name, email, age)
VALUES
/*# for user in users */
  (/*= user.name */'John', /*= user.email */'john@example.com', /*= user.age */25)
  /*# if not loop.last */,/*# end */
/*# end */
```

### ネストしたFORループ

部門と所属する社員を一括挿入する例：

```sql
INSERT INTO employees (dept_code, dept_name, emp_id, emp_name)
VALUES
/*# for dept in departments */
  /*# for emp in dept.employees */
  (
    /*= dept.code */'D001',
    /*= dept.name */'Engineering', 
    /*= dept.code + "-" + emp.id */'D001-E001',
    /*= emp.name */'Alice'
  )
  /*# if not loop.last || not loop.parent.last */,/*# end */
  /*# end */
/*# end */
```

- `loop.parent.last` で親ループの最後を判定
- カンマ管理を正確に行う

### 動的なフィルタ条件

```sql
SELECT * FROM products
WHERE 1=1
/*# for filter in filters */
  /*# if filter.field == "category" */
  AND category = /*= filter.value */'electronics'
  /*# end */
  /*# if filter.field == "price_min" */
  AND price >= /*= filter.value */0
  /*# end */
  /*# if filter.field == "price_max" */
  AND price <= /*= filter.value */10000
  /*# end */
/*# end */
```

## ループ変数

FORループ内で利用できる特殊変数：

| 変数 | 説明 | 型 |
|-----|------|-----|
| `loop.index` | 0から始まるインデックス | int |
| `loop.index1` | 1から始まるインデックス | int |
| `loop.first` | 最初の要素の場合true | bool |
| `loop.last` | 最後の要素の場合true | bool |
| `loop.length` | 配列の長さ | int |
| `loop.parent` | 親ループの`loop`オブジェクト | object |

## CELの詳細

### 利用可能な演算子

- 算術: `+`, `-`, `*`, `/`, `%`
- 比較: `==`, `!=`, `<`, `>`, `<=`, `>=`
- 論理: `&&`, `||`, `!`
- 三項演算子: `condition ? true_value : false_value`
- 文字列連結: `+`

### 利用可能な関数

SnapSQLで利用できる主なCEL関数：

- `size(list)` - リストのサイズ
- `has(map.field)` - フィールドの存在チェック
- `string(value)` - 文字列変換
- `int(value)` - 整数変換

### 型安全性

CELは型安全な式言語です：

- 型が不明な場合は暗黙変換や警告が出る
- 可能な限りダミー値で期待型を明示
- パラメータの型定義と一致させる

## 注意点とベストプラクティス

### カンマの扱い

FOR/IFでは末尾のカンマ、AND、ORなどは削除されます。

### 空配列

IN句で空配列が渡ると構文エラーになります：

```sql
-- ❌ 空配列の場合エラー
WHERE id IN (/*# for id in ids */ /*= id */0 /*# end */)

-- ✅ 空配列チェックを追加
/*# if size(ids) > 0 */
WHERE id IN (/*# for id in ids */ /*= id */0 /*# if not loop.last */,/*# end */ /*# end */)
/*# end */
```

### ダミー値の重要性

ダミー値はSQLエディタやCIでの構文チェックに使用されます：

```sql
-- ✅ Good: ダミー値が適切
WHERE status = /*= status */'active'

-- ❌ Bad: ダミー値なし（エディタで構文エラー）
WHERE status = /*= status */
```

### セキュリティ

- CELの関数呼び出しは設定で許可されたものだけ使用
- 外部呼び出しはセキュリティとテストの再現性に影響

## 関連ドキュメント

- [Markdownフォーマット](./markdown-format.md) - ファイル全体の構造
- [パラメータ](./parameters.md) - パラメータ定義
- [テスト概要](./test-overview.md) - テストの書き方
- [CEL公式ドキュメント](https://cel.dev/?hl=ja)
