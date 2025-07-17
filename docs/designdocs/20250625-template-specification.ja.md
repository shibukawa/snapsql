# SnapSQLテンプレート仕様

**ドキュメントバージョン:** 1.1  
**日付:** 2025-06-25  
**ステータス:** 実装完了

## 概要

SnapSQLは2-way SQL形式を採用したSQLテンプレートエンジンです。このドキュメントでは、実装済みのテンプレート構文と機能を定義します。

## 2-way SQL形式

SnapSQLテンプレートは、コメントを除去すると標準的なSQLとして動作するように設計されています：

- **IDE対応**: SQLの文法チェックとコード補完が有効
- **SQLリンター対応**: 標準的なSQLツールでの検証が可能
- **開発時実行**: ダミー値を使用した開発時の実行が可能
- **データベース互換性**: PostgreSQL、MySQL、SQLiteをサポート

## テンプレート構文

### 1. 制御構文

#### 条件分岐

```sql
/*# if condition */
    -- 条件がtrueの場合に含まれる内容
/*# elseif another_condition */
    -- another_conditionがtrueの場合に含まれる内容
/*# else */
    -- デフォルトの内容
/*# end */
```

**特徴:**
- `elseif`を使用（`else if`ではない）
- すべての制御構造は`/*# end */`で終了
- 条件式にはGoogle CELを使用
- 条件がfalseの場合、句全体を自動的に除去

#### ループ

```sql
/*# for variable : list_expression */
    -- リストの各要素に対して繰り返される内容
    /*= variable.field */
/*# end */
```

**特徴:**
- 配列やコレクションの反復処理
- ブロックスコープ内でのループ変数の利用
- ネストされたループのサポート
- SQLリストでのカンマの自動処理

### 2. 変数置換

#### 基本的な変数置換

```sql
/*= variable_expression */[dummy_value]
```

変数置換では、以下の2つの形式がサポートされています：

1. **明示的なダミー値（推奨）**
```sql
SELECT * FROM users WHERE id = /*= user_id */123;
SELECT * FROM users_/*= table_suffix */test;
```

2. **型推論によるダミー値の自動補完**
```sql
SELECT * FROM users WHERE id = /*= user_id */;
SELECT * FROM users WHERE active = /*= is_active */;
```

#### ダミー値を使用する利点

1. **SQL開発ツールとの互換性**
   - SQLエディタでの構文ハイライト
   - コード補完の有効化
   - SQLフォーマッターの正常動作
   - クエリプランナーでの実行計画確認

2. **開発時の直接実行**
   - コメントを除去すると有効なSQLとして動作
   - 開発環境でのクエリテスト
   - 実行計画の検証
   - インデックス使用の確認

#### 型変換

1. **標準SQL CAST**
```sql
-- 明示的な型変換
CAST(/*= value */123 AS INTEGER)
CAST(/*= date_str */'2024-01-01' AS DATE)

-- 暗黙の型変換（互換性のある型）
WHERE created_at > /*= start_date */'2024-01-01'
```

2. **PostgreSQL固有の型変換**
```sql
-- PostgreSQL形式のCAST
/*= value */123::INTEGER
/*= timestamp */'2024-01-01 12:34:56'::TIMESTAMP

-- 自動変換
-- PostgreSQL形式から標準SQLへ
WHERE created_at > CAST(/*= start_date */'2024-01-01' AS DATE)
-- 標準SQLからPostgreSQL形式へ
WHERE created_at > /*= start_date */'2024-01-01'::DATE
```

3. **配列型とJSON型**
```sql
-- PostgreSQL配列
WHERE tags = /*= tag_array */ARRAY['tag1', 'tag2']
WHERE tags @> /*= search_tags */ARRAY['tag1']

-- JSON/JSONB
WHERE data @> /*= json_filter */'{"status": "active"}'::JSONB
```

#### 型推論のデフォルト値
- 数値型（int, float）: `0`
- 文字列型: `''`
- 真偽値型: `false`
- 配列型: `[]`
- オブジェクト型: `{}`
- 日時型: `CURRENT_TIMESTAMP`

**例:**
```sql
-- 明示的なダミー値（推奨）
WHERE department IN (/*= departments */'sales', 'marketing')
  AND created_at > /*= start_date */'2024-01-01'
  AND points > /*= min_points */100

-- 型推論によるダミー値
WHERE status = /*= status */
  AND created_at > /*= start_date */
  AND is_active = /*= is_active */
  AND points > /*= min_points */
```

### 3. 自動調整機能

#### カンマの自動除去
```sql
SELECT 
    id,
    name,
    /*# if include_email */
        email,  -- 条件がfalseの場合、カンマも自動的に除去
    /*# end */
FROM users;
```

#### 空の句の自動除去

以下の場合に句全体を自動的に除去：

- **WHERE句**: すべての条件がnullまたは空
- **ORDER BY句**: ソートフィールドがnullまたは空
- **LIMIT句**: limitがnullまたは負数
- **OFFSET句**: offsetがnullまたは負数
- **AND/OR条件**: 変数がnullまたは空

```sql
SELECT * FROM users
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales')
ORDER BY /*= sort_field */name
LIMIT /*= page_size */10
OFFSET /*= offset */0;
```

#### 配列の展開

配列は自動的にカンマ区切りの引用符付き値に展開：

```sql
-- テンプレート
WHERE department IN (/*= departments */'sales', 'marketing')

-- 実行時（departments = ['engineering', 'design', 'product']）
WHERE department IN ('engineering', 'design', 'product')
```

## サポートされるSQL文

### SELECT文

```sql
SELECT 
    id,
    name,
    /*# if include_email */
        email,
    /*# end */
    /*# for field : additional_fields */
        /*= field */,
    /*# end */
FROM users
WHERE active = /*= filters.active */true
    /*# if filters.department */
    AND department = /*= filters.department */'sales'
    /*# end */
ORDER BY created_at DESC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
```

### INSERT文

#### 単一行INSERT
```sql
INSERT INTO products (name, price, category_id)
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

#### 複数行INSERT
```sql
INSERT INTO products (name, price, category_id)
VALUES 
    (/*= product1.name */'Product A', /*= product1.price */100.50, /*= product1.category_id */1),
    (/*= product2.name */'Product B', /*= product2.price */200.75, /*= product2.category_id */2);
```

#### 配列を使用した複数行INSERT
```sql
-- productsが[]map[string]anyの場合、自動的に複数のVALUESに展開
INSERT INTO products (name, price, category_id) 
VALUES /*= products */('Product A', 100.50, 1);
```

### UPDATE文

```sql
UPDATE products
SET 
    name = /*= updates.name */'Updated Product',
    price = /*= updates.price */150.00,
    /*# if updates.category_id */
    category_id = /*= updates.category_id */2,
    /*# end */
    updated_at = NOW()
WHERE id = /*= product_id */123;
```

### DELETE文

```sql
DELETE FROM logs
WHERE created_at < /*= cutoff_date */'2024-01-01'
    /*# if level_filter */
    AND level = /*= level_filter */'DEBUG'
    /*# end */;
```

## テーブル名のサフィックス

環境別のテーブル名をサフィックスで制御：

```sql
-- テンプレート
SELECT * FROM users_/*= env */test;

-- 実行時
SELECT * FROM users_dev;     -- env = "dev"
SELECT * FROM users_staging; -- env = "staging"  
SELECT * FROM users_prod;    -- env = "prod"
```

## セキュリティ制限

### ✅ 許可される操作
- WHERE条件の追加/削除
- ORDER BY句の追加/削除
- SELECT項目の追加/削除
- テーブル名のサフィックス変更
- IN句での配列展開
- 条件に基づく句の除去
- 複数行INSERTの操作
- SnapSQL変数を使用したDML操作

### ❌ 制限される操作
- SQLの構造的な変更
- テーブル名の動的な変更（サフィックス以外）
- 任意のSQL注入
- スキーマ変更文（DDL）

## エラー処理

### テンプレート検証
- 構文エラーの検出
- ディレクティブの形式チェック
- 変数参照の検証
- 制御ブロックの範囲チェック

### 実行時検証
- 型チェック
- NULL値の処理
- パラメータの検証

## ファイル形式

### `.snap.sql`ファイル
SQLテンプレートのみを含むファイル：

```sql
-- queries/users.snap.sql
SELECT id, name
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

### `.snap.md`ファイル
ドキュメントとテストケースを含むファイル：

````markdown
# ユーザークエリテンプレート

## SQLテンプレート
```sql
SELECT id, name
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

## テストケース
```json
{
    "env": "prod",
    "filters": {"active": true}
}
```
````

## ベストプラクティス

1. **シンプルに始める**: 基本的なテンプレートから開始し、徐々に複雑さを追加
2. **意図を文書化**: 複雑なロジックにはコメントを追加
3. **テストケースの作成**: 包括的なテストケースを提供
4. **インデックスを考慮**: 動的WHERE句でのインデックス使用を考慮
5. **一貫したパターン**: テンプレート間で一貫した命名規則と構造を使用
