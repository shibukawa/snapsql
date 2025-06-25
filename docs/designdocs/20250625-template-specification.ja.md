# SnapSQL テンプレート仕様書

**ドキュメントバージョン:** 1.0  
**日付:** 2025-06-25  
**ステータス:** 実装完了

## 概要

SnapSQLは**2-way SQL形式**を使用して動的SQL生成を可能にするSQLテンプレートエンジンです。この仕様書では、開発時に標準SQLとして動作しながら、実行時に柔軟性を提供する動的SQLテンプレートを作成するための完全なテンプレート構文、機能、使用パターンを定義します。

## 基本原則

### 2-way SQL形式

SnapSQLテンプレートは、コメントを削除した際に**有効なSQL**として動作するよう設計されています：

- **IDE サポート**: 完全な構文ハイライトとIntelliSense
- **SQL リンティング**: 標準SQLツールによる基本構文検証
- **開発時テスト**: ダミー値を使用したテンプレートの実行
- **データベース互換性**: PostgreSQL、MySQL、SQLiteで動作

### コメントベースディレクティブ

すべてのSnapSQL機能はSQLコメントを通じて実装され、テンプレートが有効なSQLであることを保証します：

- 制御フロー: `/*# if */`, `/*# for */`, `/*# end */`
- 変数置換: `/*= variable */`
- 環境参照: 変数システムに組み込み

## テンプレート構文

### 1. 制御フローディレクティブ

#### 条件ブロック

```sql
/*# if condition */
    -- 条件がtrueの場合に含まれるコンテンツ
/*# elseif another_condition */
    -- another_conditionがtrueの場合に含まれるコンテンツ
/*# else */
    -- デフォルトコンテンツ
/*# end */
```

**主要機能:**
- `elseif`（`else if`ではない）による代替条件
- すべての制御構造に統一された`/*# end */`終端子
- Google CEL（Common Expression Language）を使用した条件
- 条件がfalseの場合の自動句削除

#### ループブロック

```sql
/*# for variable : list_expression */
    -- リスト内の各項目に対して繰り返されるコンテンツ
    /*= variable.field */
/*# end */
```

**主要機能:**
- 配列とコレクションの反復処理
- ブロックスコープ内でのループ変数利用
- ネストしたループのサポート
- SQLリストでの自動カンマ処理

### 2. 変数置換

#### 基本的な変数置換

```sql
/*= variable_expression */dummy_value
```

**例:**
```sql
SELECT * FROM users WHERE id = /*= user_id */123;
SELECT * FROM users_/*= table_suffix */test;
WHERE department IN (/*= departments */'sales', 'marketing');
```

#### ネストした変数アクセス

```sql
/*= object.field */
/*= array[0].property */
/*= nested.object.deep.field */
```

### 3. 自動実行時調整

SnapSQLは一般的なSQL構築の課題を自動的に処理します：

#### 末尾カンマの削除
```sql
SELECT 
    id,
    name,
    /*# if include_email */
        email,  -- 条件がfalseの場合、末尾カンマが自動削除される
    /*# end */
FROM users;
```

#### 空句の削除

すべての条件がnull/空の場合、句全体が自動的に削除されます：

- **WHERE句**: すべての条件がnull/空の場合に削除
- **ORDER BY句**: ソートフィールドがnull/空の場合に削除
- **LIMIT句**: limitがnullまたは負の場合に削除
- **OFFSET句**: offsetがnullまたは負の場合に削除
- **AND/OR条件**: 変数がnull/空の場合に個別条件を削除

```sql
SELECT * FROM users
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales')
ORDER BY /*= sort_field */name
LIMIT /*= page_size */10
OFFSET /*= offset */0;
```

#### 配列展開

配列は自動的にカンマ区切りのクォート付き値に展開されます：

```sql
-- テンプレート
WHERE department IN (/*= departments */'sales', 'marketing')

-- departments = ['engineering', 'design', 'product']での実行時
WHERE department IN ('engineering', 'design', 'product')
```

## SQL文のサポート

### SELECT文

#### 基本構造
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
FROM users_/*= env */prod
WHERE active = /*= filters.active */true
    /*# if filters.department */
    AND department = /*= filters.department */'sales'
    /*# end */
ORDER BY 
    /*# for sort : sort_fields */
        /*= sort.field */ /*= sort.direction */,
    /*# end */
    created_at DESC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */0;
```

#### 高度な機能
- **JOIN**: INNER、LEFT、RIGHT、FULL OUTER結合の完全サポート
- **サブクエリ**: 変数置換を含むネストしたSELECT文
- **ウィンドウ関数**: 動的パーティショニングを含むOVER句
- **CTE（共通テーブル式）**: 再帰サポートを含むWITH句
- **集約**: 動的グループ化を含むGROUP BY、HAVING句

### INSERT文

#### 標準INSERT
```sql
INSERT INTO products_/*= env */prod (name, price, category_id)
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

#### バルクINSERT（複数VALUES）
```sql
INSERT INTO products (name, price, category_id)
VALUES 
    (/*= product1.name */'Product A', /*= product1.price */100.50, /*= product1.category_id */1),
    (/*= product2.name */'Product B', /*= product2.price */200.75, /*= product2.category_id */2),
    (/*= product3.name */'Product C', /*= product3.price */150.25, /*= product3.category_id */1);
```

#### 条件付きバルクINSERT
```sql
INSERT INTO orders (user_id, product_id, quantity)
VALUES 
    (/*= order.user_id */1, /*= order.product_id */1, /*= order.quantity */2)
    /*# if include_bulk_orders */
    , (/*= bulk_order1.user_id */2, /*= bulk_order1.product_id */2, /*= bulk_order1.quantity */1)
    , (/*= bulk_order2.user_id */3, /*= bulk_order2.product_id */3, /*= bulk_order2.quantity */5)
    /*# end */;
```

#### Map配列バルクINSERT（自動展開）

**Map配列（バルク）:**
```sql
-- 'products'が[]map[string]anyの場合、自動的に複数のVALUESに展開
INSERT INTO products (name, price, category_id) 
VALUES /*= products */('Product A', 100.50, 1);
```

**単一Map（通常）:**
```sql
-- 'product'がmap[string]anyの場合、通常の変数として処理
INSERT INTO products (name, price, category_id) 
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

#### INSERT...SELECT
```sql
INSERT INTO archive_users_/*= env */prod (id, name, email)
SELECT id, name, email 
FROM users_/*= env */prod
WHERE created_at < /*= cutoff_date */'2024-01-01'
    /*# if department_filter */
    AND department = /*= department_filter */'sales'
    /*# end */;
```

### UPDATE文

#### 基本UPDATE
```sql
UPDATE products_/*= env */prod
SET 
    name = /*= updates.name */'Updated Product',
    price = /*= updates.price */150.00,
    /*# if updates.category_id */
    category_id = /*= updates.category_id */2,
    /*# end */
    updated_at = NOW()
WHERE id = /*= product_id */123
    /*# if additional_filters */
    AND status = /*= additional_filters.status */'active'
    /*# end */;
```

#### 条件付きフィールド更新
```sql
UPDATE users_/*= env */prod
SET 
    /*# if updates.name */
    name = /*= updates.name */'New Name',
    /*# end */
    /*# if updates.email */
    email = /*= updates.email */'new@example.com',
    /*# end */
    updated_at = NOW()
WHERE id = /*= user_id */123;
```

#### 動的SET句を含むバルクUPDATE
```sql
UPDATE products_/*= env */prod
SET 
    /*# for update : field_updates */
    /*= update.field */ = /*= update.value */,
    /*# end */
    updated_at = NOW()
WHERE id IN (/*= product_ids */1, 2, 3);
```

### DELETE文

#### 基本DELETE
```sql
DELETE FROM logs_/*= env */prod
WHERE created_at < /*= cutoff_date */'2024-01-01'
    /*# if level_filter */
    AND level = /*= level_filter */'DEBUG'
    /*# end */;
```

#### 条件付きDELETE
```sql
DELETE FROM users_/*= env */prod
WHERE 
    /*# if delete_inactive */
    status = 'inactive'
    /*# end */
    /*# if delete_old */
    /*# if delete_inactive */AND /*# end */last_login < /*= cutoff_date */'2023-01-01'
    /*# end */;
```

#### サブクエリを含むDELETE
```sql
DELETE FROM orders_/*= env */prod
WHERE user_id IN (
    SELECT id FROM users_/*= env */prod
    WHERE status = /*= user_status */'deleted'
        /*# if department_filter */
        AND department = /*= department_filter */'sales'
        /*# end */
);
```

## 環境とテーブル管理

### テーブルサフィックスパターン

SnapSQLはサフィックスパターンを通じて環境固有のテーブル命名をサポートします：

```sql
-- テンプレート
SELECT * FROM users_/*= env */test;

-- 実行時の例
SELECT * FROM users_dev;     -- env = "dev"
SELECT * FROM users_staging; -- env = "staging"  
SELECT * FROM users_prod;    -- env = "prod"
```

### 環境固有の設定

```sql
-- 環境ごとの異なるデータベーススキーマ
SELECT * FROM /*= schema */public.users_/*= env */prod;

-- 環境固有の接続パラメータ
CONNECT TO /*= database_url */postgresql://localhost/myapp_/*= env */dev;
```

## 高度な機能

### 暗黙的条件生成

SnapSQLはnullまたは空の可能性がある変数に対して自動的に条件ブロックを生成します：

```sql
-- テンプレート
WHERE status = /*= filters.status */'active'

-- 自動的に以下のようになる
/*# if filters.status != null && filters.status != "" */
WHERE status = /*= filters.status */'active'
/*# end */
```

### ネストした変数参照

複雑なオブジェクトナビゲーションのサポート：

```sql
SELECT 
    /*= user.profile.personal.first_name */'John',
    /*= user.profile.personal.last_name */'Doe',
    /*= user.settings.preferences.theme */'dark'
FROM users;
```

### 動的フィールド選択

```sql
SELECT 
    id,
    /*# for field : dynamic_fields */
    /*= field */,
    /*# end */
    created_at
FROM /*= table_name */users;
```

### 複雑な条件ロジック

```sql
WHERE 1=1
    /*# if filters.status */
    AND status = /*= filters.status */'active'
    /*# end */
    /*# if filters.date_range */
        /*# if filters.date_range.start */
        AND created_at >= /*= filters.date_range.start */'2024-01-01'
        /*# end */
        /*# if filters.date_range.end */
        AND created_at <= /*= filters.date_range.end */'2024-12-31'
        /*# end */
    /*# end */;
```

## テンプレート書式ガイドライン

### インデントルール

1. **制御ブロックコンテンツ**: `/*# if */`と`/*# for */`ブロック内のコンテンツをインデント
2. **一貫した構造**: 可読性のために一貫したインデントを維持
3. **改行**: 視認性のために制御ブロックを新しい行で開始

```sql
-- 良い書式
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
    /*# end */;
```

### カンマの配置

1. **末尾カンマ**: テンプレート作成を容易にするため`field,`と記述（`,field`ではない）
2. **自動クリーンアップ**: SnapSQLが末尾カンマを自動削除
3. **条件付きフィールド**: 条件ブロックでのカンマ管理を気にする必要なし

### ダミー値ガイドライン

1. **常に含める**: 2-way SQL互換性のために現実的なダミー値を提供
2. **型の一致**: ダミー値が期待されるデータ型と一致することを確認
3. **現実的なデータ**: より良い開発体験のために代表的な値を使用

```sql
-- 良いダミー値
WHERE user_id = /*= user_id */123
AND email = /*= email */'user@example.com'
AND created_at > /*= start_date */'2024-01-01'
AND price BETWEEN /*= min_price */10.00 AND /*= max_price */100.00
```

## セキュリティ考慮事項

### 許可される操作

SnapSQLは安全な操作に変更を制限します：

✅ **許可:**
- WHERE条件の追加/削除
- ORDER BY句の追加/削除
- SELECTフィールドの追加/削除
- テーブル名サフィックスの変更
- IN句での配列展開
- 条件付き句の削除
- バルクINSERT操作
- SnapSQL変数を含む動的DML操作

❌ **制限:**
- SQLの主要構造変更
- 動的テーブル名変更（サフィックス以外）
- 任意のSQLインジェクション
- スキーマ変更文（DDL）

### パラメータバインディング

すべての変数置換は適切なエスケープを含むパラメータ化クエリを生成します：

```sql
-- テンプレート
WHERE name = /*= user_name */'John'

-- 生成（概念的）
WHERE name = ?  -- パラメータ: "John"
```

## エラーハンドリング

### テンプレート検証

SnapSQLはビルド時にテンプレートを検証します：

- **構文エラー**: 無効なSQL構造の検出
- **ディレクティブエラー**: 不正な制御フローディレクティブ
- **変数エラー**: 未定義変数参照
- **句違反**: 複数のSQL句にまたがる制御ブロック

### 実行時検証

- **型チェック**: スキーマに対する変数型検証
- **Null処理**: 自動null/空値処理
- **パラメータ検証**: 必須パラメータチェック

## ファイル構成

### テンプレートファイル

#### `.snap.sql`ファイル
SnapSQLディレクティブを含む純粋なSQLテンプレート：

```sql
-- queries/users.snap.sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

#### `.snap.md`ファイル（文芸的プログラミング）
テンプレートとドキュメント、テストケースを組み合わせ：

````markdown
# ユーザークエリテンプレート

## 目的
オプションのメール含有と環境固有テーブルでユーザーデータを取得。

## パラメータ
- `include_email`: boolean - 結果にメールフィールドを含める
- `env`: string - テーブル名の環境サフィックス
- `filters.active`: boolean - アクティブステータスでフィルタ

## SQLテンプレート

```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
FROM users_/*= env */test
WHERE active = /*= filters.active */true;
```

## テストケース

### 基本クエリ
**入力:**
```json
{
    "include_email": false,
    "env": "prod",
    "filters": {"active": true}
}
```

**期待される出力:**
```sql
SELECT id, name FROM users_prod WHERE active = true;
```
````

## ベストプラクティス

### テンプレート設計

1. **シンプルに始める**: 基本的なテンプレートから始めて徐々に複雑さを追加
2. **意味のある名前を使用**: 説明的な変数とパラメータ名を選択
3. **意図を文書化**: 複雑なロジックを説明するコメントを含める
4. **徹底的にテスト**: 包括的なテストケースを提供

### パフォーマンス考慮事項

1. **インデックス認識**: 動的WHERE句を設計する際にデータベースインデックスを考慮
2. **クエリプランニング**: 生成されたクエリをEXPLAIN/ANALYZEでテスト
3. **パラメータ制限**: バルク操作でのデータベースパラメータ制限を意識

### 保守性

1. **一貫したパターン**: テンプレート全体で一貫した命名と構造を使用
2. **モジュラー設計**: 複雑なテンプレートを小さく再利用可能なコンポーネントに分割
3. **バージョン管理**: 意味のあるコミットメッセージでテンプレート変更を追跡
4. **コードレビュー**: アプリケーションコードのようにテンプレートをレビュー

## 移行と互換性

### データベース互換性

SnapSQLテンプレートは軽微な方言考慮事項でデータベースシステム間で動作します：

- **PostgreSQL**: 完全機能サポート
- **MySQL**: 方言固有構文での完全機能サポート
- **SQLite**: 高度機能の制限付き完全機能サポート

### バージョン互換性

- **後方互換性**: 新しいSnapSQLバージョンはテンプレート互換性を維持
- **非推奨ポリシー**: 事前通知と移行パスを含む機能非推奨
- **アップグレードガイダンス**: 破壊的変更に対する明確なアップグレード指示

## 結論

SnapSQLは動的SQL生成に対する強力で安全で保守可能なアプローチを提供します。2-way SQL形式に従い、コメントベースディレクティブを活用することで、開発者は標準SQLツールと実践の利点を維持しながら、開発環境と本番環境でシームレスに動作する柔軟なSQLテンプレートを作成できます。

テンプレート仕様により、SnapSQLテンプレートは以下を保証します：

- **開発者フレンドリー**: 既存のSQLツールとワークフローで動作
- **実行時柔軟性**: 変化する要件と条件に適応
- **セキュリティ重視**: 制御された変更によるSQLインジェクション防止
- **パフォーマンス認識**: 効率的でパラメータ化されたクエリを生成
- **保守可能**: 長期的なアプリケーション進化をサポート

この仕様書は、すべてのサポートされる実行時環境でSnapSQLテンプレートを作成・保守するための決定版ガイドとして機能します。
