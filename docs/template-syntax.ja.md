# テンプレート構文

SnapSQLは、テンプレートが標準SQLとして動作しながら実行時の柔軟性を提供する2-way SQL形式を使用します。

## 基本概念

### 2-way SQL形式

核となる原則は、SQLテンプレートが開発時に直接実行できる有効なSQLであるべきということです：

```sql
-- これは標準SQLとして動作します
SELECT id, name, email
FROM users_dev
WHERE active = true
ORDER BY created_at DESC
LIMIT 10;
```

SnapSQL構文を使った同じテンプレート：

```sql
-- これも標準SQLとして動作します（コメントは無視されます）
SELECT 
    id, 
    name,
    /*# if include_email */
    email,
    /*# end */
FROM users_/*= table_suffix */dev
WHERE active = /*= filters.active */true
ORDER BY created_at DESC
LIMIT /*= pagination.limit */10;
```

## テンプレートディレクティブ

### 変数置換

変数置換には`/*= expression */default_value`を使用します：

```sql
-- 基本的な置換
FROM users_/*= table_suffix */dev

-- パラメータ付き
WHERE active = /*= filters.active */true
LIMIT /*= pagination.limit */10
```

### 条件ブロック

条件付きコンテンツには`/*# if condition */`と`/*# end */`を使用します：

```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    /*# if include_profile */
    profile_data,
    /*# end */
    created_at
FROM users
```

### ループ（計画中）

```sql
/*# for field in selected_fields */
    /*= field */,
/*# end */
```

## パラメータ型

### シンプルパラメータ

```json
{
  "table_suffix": "prod",
  "limit": 50,
  "active": true
}
```

### ネストされたパラメータ

```json
{
  "filters": {
    "active": true,
    "department": "engineering"
  },
  "pagination": {
    "limit": 20,
    "offset": 0
  }
}
```

### 配列パラメータ

```json
{
  "departments": ["engineering", "design", "product"],
  "user_ids": [1, 2, 3, 4, 5]
}
```

## テンプレートメタデータ

各テンプレートは、先頭のコメントブロックにメタデータを含めることができます：

```sql
/*#
name: getUserList
description: オプションのフィルタリング付きユーザーリストを取得
function_name: getUserList
parameters:
  include_email: bool
  table_suffix: string
  filters:
    active: bool
    departments:
      - string
  pagination:
    limit: int
    offset: int
*/

SELECT id, name FROM users;
```

## 式言語

SnapSQLはパラメータ参照のためのシンプルな式言語を使用します：

- `table_suffix` - シンプルパラメータ
- `filters.active` - ネストされたパラメータ
- `pagination.limit` - ネストされたパラメータ
- `departments` - 配列パラメータ

## セキュリティ機能

SnapSQLはSQLインジェクションを防ぐための制御された変更を提供します：

- **許可**: フィールド選択、シンプルな条件、テーブルサフィックス変更
- **防止**: 任意のSQLインジェクション、複雑なWHERE句の変更
- **検証**: すべてのパラメータは型チェックと検証が行われます

## ベストプラクティス

1. **常にデフォルトを提供**: テンプレートが標準SQLとして動作することを確保
2. **意味のあるパラメータ名を使用**: テンプレートを自己文書化
3. **条件をシンプルに保つ**: 複雑なロジックはアプリケーションコードに
4. **ドライランでテスト**: `--dry-run`を使用してテンプレート処理を検証
5. **パラメータを文書化**: より良い保守性のためにメタデータを含める

## 例

### フィルタリング付きユーザークエリ

```sql
/*#
name: searchUsers
description: 様々なフィルターでユーザーを検索
parameters:
  search_term: string
  department: string
  active_only: bool
  limit: int
*/

SELECT 
    u.id,
    u.name,
    u.email,
    u.department
FROM users u
WHERE 1=1
    /*# if search_term */
    AND (u.name ILIKE /*= search_term */'%john%' OR u.email ILIKE /*= search_term */'%john%') AND
    /*# end */
    /*# if department */
    AND u.department = /*= department */'engineering' AND
    /*# end */
    /*# if active_only */
    AND u.active = true AND
    /*# end */
ORDER BY u.created_at DESC
LIMIT /*= limit */50;
```

### 動的テーブル選択

```sql
/*#
name: getTableData
description: 異なるテーブルバリアントからデータを取得
parameters:
  environment: string
  include_archived: bool
*/

SELECT *
FROM data_/*= environment */prod
/*# if include_archived */
UNION ALL
SELECT * FROM data_archive_/*= environment */prod
/*# end */
ORDER BY created_at DESC;
```
