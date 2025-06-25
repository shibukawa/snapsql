# SnapSQL

SnapSQLは、2-way SQL形式を使用した動的SQL生成を可能にするSQLテンプレートエンジンです。開発者は、開発時に標準SQLとして実行できるSQLテンプレートを記述しながら、実行時の動的クエリ構築の柔軟性を提供します。

## 機能

- **2-way SQL形式**: コメントを削除すると標準SQLとして動作するSQLテンプレートを記述
- **動的クエリ構築**: 実行時にWHERE句、ORDER BY、SELECTフィールドを動的に追加
- **セキュリティ第一**: 制御された変更によりSQLインジェクションを防止 - フィールド選択、単純な条件、テーブルサフィックス変更などの安全な操作のみを許可
- **マルチデータベースサポート**: PostgreSQL、MySQL、SQLiteデータベースで動作するよう設計
- **Google CEL統合**: 条件とパラメータ参照にCommon Expression Languageを使用
- **高度なSQL解析**: 複雑なクエリ、CTE、DML操作をサポートする包括的なSQLパーサー
- **テンプレートエンジン**: 条件ブロック、ループ、変数置換を備えた強力なテンプレート処理
- **バルク操作**: 動的フィールドマッピングによるバルクINSERT操作のサポート
- **型安全性**: 強力な型チェックとパラメータ検証

## 動作原理

SnapSQLは既存のデータベースインフラストラクチャとテーブルスキーマと連携します：

1. **ビルド時**: `snapsql`ツール（Go）がSQLテンプレートとSQLを含むマークダウンファイルをASTとメタデータを含む中間JSONファイルに変換 *（計画中）*
2. **実行時**: 言語固有のライブラリが中間ファイルを使用して、実行時パラメータに基づいて動的にSQLクエリを構築 *（計画中）*
3. **現在の状態**: コアSQLパーサーとテンプレートエンジンがGoで実装済み

**前提条件**: SnapSQLは、テーブルが既に作成された既存のデータベースがあることを前提としています。PostgreSQL、MySQL、SQLiteデータベースで動作します。

## 実行時機能

### 型安全なクエリ関数

実行時ライブラリは、解析されたSQL構造に基づいて型安全な関数を生成し、以下を提供します：

- **強く型付けされたパラメータ**: SQLテンプレートパラメータに一致する関数シグネチャ
- **結果型マッピング**: SQL結果の言語固有の型への自動マッピング
- **コンパイル時安全性**: パラメータの不一致と型エラーをコンパイル時にキャッチ
- **IDE サポート**: クエリパラメータと結果の完全な自動補完とIntelliSenseサポート

### モックとテストサポート

SnapSQLは、データベース接続を必要としない組み込みテスト機能を提供します：

- **モックデータ生成**: 単体テスト用にクエリ構造に一致するダミーデータを返す
- **YAMLベースのモックデータ**: [dbtestify](https://github.com/shibukawa/dbtestify)ライブラリ統合でYAML形式を使用してモック応答を定義
- **設定可能な応答**: 異なるテストシナリオ用のカスタムモック応答を定義
- **ゼロデータベース依存**: テストデータベースの設定なしでテストを実行
- **一貫したデータ形状**: モックデータは実際のクエリ結果と同じ型構造に従う

### パフォーマンス分析

組み込みのパフォーマンス分析ツールがクエリの最適化を支援します：

- **実行プラン分析**: クエリ実行プランの生成と分析
- **パフォーマンス推定**: テーブル統計とクエリ複雑性に基づくクエリパフォーマンスの予測
- **ボトルネック検出**: デプロイ前の潜在的なパフォーマンス問題の特定
- **最適化提案**: クエリ改善の推奨事項を受信

これらの機能は、アプリケーションコードの変更を必要とせずにシームレスに動作します - 設定を通じて本番、モック、分析モードを切り替えるだけです。

## テンプレート構文

SnapSQLは、標準SQL実行を妨げないコメントベースのディレクティブを使用し、**2-way SQL形式**に従います：

### 制御フローディレクティブ
- `/*# if condition */` - 条件ブロック
- `/*# elseif condition */` - 代替条件（注意：`elseif`を使用、`else if`ではない）
- `/*# else */` - デフォルト条件
- `/*# end */` - 条件ブロックの終了（すべての制御構造の統一された終了）
- `/*# for variable : list */` - コレクションのループ
- `/*# end */` - ループブロックの終了

### 変数置換
- `/*= variable */` - 2-way SQL互換性のためのダミーリテラルを持つ変数プレースホルダー

### 2-way SQL形式

SnapSQLテンプレートは、コメントが削除されたときに**有効なSQL**として動作するよう設計されており、以下を可能にします：
- **IDEサポート**: 完全な構文ハイライトとIntelliSense
- **SQLリンティング**: 標準SQLツールが基本構文を検証可能
- **開発テスト**: 開発中にダミー値でテンプレートを実行

#### ダミーリテラルを使用した変数置換

変数には、コメントが削除されたときにSQLを有効にするダミーリテラルが含まれます：

```sql
-- ダミーリテラル付きテンプレート
SELECT * FROM users_/*= table_suffix */test
WHERE active = /*= filters.active */true
  AND department IN (/*= filters.departments */'sales', 'marketing')
LIMIT /*= pagination.limit */10;

-- コメント削除時の有効なSQL
SELECT * FROM users_test
WHERE active = true
  AND department IN ('sales', 'marketing')
LIMIT 10;

-- 実際のパラメータでの実行時結果
SELECT * FROM users_prod
WHERE active = false
  AND department IN ('engineering', 'design', 'product')
LIMIT 20;
```

#### 自動実行時調整

実行時は自動的に以下を処理します：
- **末尾カンマ**: 条件フィールドが除外されたときに削除
- **空の句**: すべての条件がfalseのときにWHERE/ORDER BY句を削除
- **配列展開**: `/*= array_var */`が`'val1', 'val2', 'val3'`に展開
- **ダミーリテラル削除**: 開発用ダミー値を実際のパラメータに置換
- **条件句削除**: 変数がnull/空のときに句全体を削除
  - `WHERE`句: すべての条件がnull/空のときに削除
  - `ORDER BY`句: ソートフィールドがnull/空のときに削除
  - `LIMIT`句: limitがnullまたは負の値のときに削除
  - `OFFSET`句: offsetがnullまたは負の値のときに削除
  - `AND/OR`条件: 変数がnull/空のときに個別の条件を削除

### テンプレート書式ガイドライン

1. **制御ブロック内のインデント**: `/*# if */`と`/*# for */`ブロック内のコンテンツは1レベルインデントする
2. **可読性のための改行**: `/*# for */`ブロックは視認性向上のため新しい行で開始
3. **一貫した構造**: テンプレートの可読性向上のため一貫したインデントを維持

```sql
-- 適切なインデントでの良い書式
SELECT 
    id,
    name,
    /*# if include_email */
        email,
    /*# end */
    /*# for field : additional_fields */
        /*= field */
    /*# end */
FROM users
ORDER BY 
    /*# for sort : sort_fields */
        /*= sort.field */ /*= sort.direction */
    /*# end */name ASC;

-- 避けるべき：インデントなしの悪い書式
SELECT id, name, /*# if include_email */email,/*# end */ /*# for field : additional_fields *//*= field *//*# end */ FROM users;
```

### テンプレート記述ガイドライン

1. **要素の後にカンマを配置**: テンプレート作成を容易にするため`field,`と記述し、`,field`は避ける
2. **ダミーリテラルを含める**: コメント削除時にSQLが有効であることを確保
3. **一貫した終了を使用**: すべての制御構造に常に`/*# end */`を使用
4. **自動調整を活用**: 末尾カンマや空の句を心配する必要なし
5. **明白な条件を省略**: 自動句削除（WHERE、ORDER BY、LIMIT、OFFSET）のため`/*# if */`ブロックをスキップ

### 自動句削除の例

```sql
-- 単純に記述可能：
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales', 'marketing')
ORDER BY 
    /*# for sort : sort_fields */
        /*= sort.field */ /*= sort.direction */
    /*# end */name ASC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */5

-- 冗長な条件ブロックの代わりに：
/*# if filters.active */
WHERE active = /*= filters.active */
    /*# if filters.departments */
    AND department IN (/*= filters.departments */)
    /*# end */
/*# end */
/*# if sort_fields */
ORDER BY /*# for sort : sort_fields *//*= sort.field */ /*= sort.direction *//*# end */
/*# end */
/*# if pagination.limit > 0 */
LIMIT /*= pagination.limit */
/*# end */
```

### テンプレート例

```sql
SELECT 
    id,
    name,
    /*# if include_email */
        email,
    /*# end */
    /*# if include_profile */
        profile_image,
        bio
    /*# end */
    /*# for field : additional_fields */
        /*= field */
    /*# end */
FROM users_/*= table_suffix */test
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales', 'marketing')
ORDER BY 
    /*# for sort : sort_fields */
        /*= sort.field */ /*= sort.direction */
    /*# end */name ASC
LIMIT /*= pagination.limit */10
OFFSET /*= pagination.offset */5;
```

### バルクインサート例

SnapSQLは複数のVALUES句を持つバルクINSERT操作をサポートします：

```sql
-- 基本的なバルクインサート
INSERT INTO users (name, email, created_at) 
VALUES 
    ('John Doe', 'john@example.com', NOW()),
    ('Jane Smith', 'jane@example.com', NOW()),
    ('Bob Wilson', 'bob@example.com', NOW());

-- SnapSQL変数を使用した動的バルクインサート
INSERT INTO products (name, price, category_id) 
VALUES 
    (/*= product1.name */'Product A', /*= product1.price */100.50, /*= product1.category_id */1),
    (/*= product2.name */'Product B', /*= product2.price */200.75, /*= product2.category_id */2),
    (/*= product3.name */'Product C', /*= product3.price */150.25, /*= product3.category_id */1);

-- 条件付きバルクインサート
INSERT INTO orders (user_id, product_id, quantity) 
VALUES 
    (/*= order.user_id */1, /*= order.product_id */1, /*= order.quantity */2)
    /*# if include_bulk_orders */
    , (/*= bulk_order1.user_id */2, /*= bulk_order1.product_id */2, /*= bulk_order1.quantity */1)
    , (/*= bulk_order2.user_id */3, /*= bulk_order2.product_id */3, /*= bulk_order2.quantity */5)
    /*# end */;

-- マップ配列バルクインサート（自動展開）
-- 'products'が[]map[string]anyの場合、自動的に複数のVALUES句に展開
INSERT INTO products (name, price, category_id) 
VALUES /*= products */('Product A', 100.50, 1);

-- 単一マップインサート（非バルク）
-- 'product'がmap[string]anyの場合、通常の変数として扱われる
INSERT INTO products (name, price, category_id) 
VALUES (/*= product.name */'Product A', /*= product.price */100.50, /*= product.category_id */1);
```

コメントが削除されると、これは有効なSQLになります：
```sql
SELECT 
    id,
    name,
        email,
        profile_image,
        bio
        field1
FROM users_test
WHERE active = true
    AND department IN ('sales', 'marketing')
ORDER BY 
        name ASC, created_at DESC
    name ASC
LIMIT 10
OFFSET 5;
```

実際のパラメータでの実行時結果：
```sql
SELECT 
    id,
    name,
    email
FROM users_prod
WHERE active = false
    AND department IN ('engineering', 'design', 'product')
ORDER BY created_at DESC
LIMIT 20
OFFSET 40;
```

## サポートされる動的操作

### ✅ 許可される操作
- WHERE条件の追加/削除
- ORDER BY句の追加/削除  
- SELECTフィールドの追加/削除
- テーブル名サフィックスの変更（例：`users_test`、`log_202412`）
- IN句での配列展開（例：`/*= departments */` → `'sales', 'marketing', 'engineering'`）
- 末尾カンマと括弧の制御
- 句全体の条件削除（WHERE、ORDER BY）
- **バルクINSERT操作**（複数のVALUES句）
- **動的DML操作**（SnapSQL変数を使用したINSERT、UPDATE、DELETE）

### ❌ 制限される操作
- SQLの主要な構造変更
- 動的テーブル名変更（サフィックス以外）
- 任意のSQLインジェクション

### 実行時処理機能

#### 自動クリーンアップ
- **末尾カンマ**: 条件フィールドが除外されたときに自動削除
- **空のWHERE句**: 条件がアクティブでないときにWHERE句全体を削除
- **空のORDER BY**: ソートフィールドが指定されていないときにORDER BY句を削除
- **ダミーリテラル**: 開発用ダミー値を実際の実行時値に置換

#### 配列処理
- **IN句展開**: 配列変数を自動的にカンマ区切りの引用値に展開
- **バルク操作**: 動的フィールドマッピングによるバルクインサート操作のサポート

## インストール

### ビルドツール *（計画中）*
```bash
go install github.com/shibukawa/snapsql@latest
```

### ランタイムライブラリ *（計画中）*

#### Go
```bash
go get github.com/shibukawa/snapsql
```

#### Java *（計画中）*
```xml
<dependency>
    <groupId>com.github.shibukawa</groupId>
    <artifactId>snapsql-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

#### Python *（計画中）*
```bash
pip install snapsql-python
```

#### Node.js *（計画中）*
```bash
npm install snapsql-js
```

### 現在の使用方法（開発）

現在、SQLテンプレートの解析と処理のためのGoライブラリとしてSnapSQLを使用できます：

```bash
go get github.com/shibukawa/snapsql
```

## 使用方法

### 1. SQLテンプレートの作成

SnapSQL構文で`.snap.sql`または`.snap.md`ファイルを作成：

#### シンプルなSQLテンプレート（`.snap.sql`）
```sql
-- queries/users.snap.sql
SELECT 
    id,
    name
    /*# if include_email */,
    email
    /*# end */
FROM users_/*= env */test
/*# if filters.active */
WHERE active = /*= filters.active */true
/*# end */
/*# if sort_by != "" */
ORDER BY /*= sort_by */name
/*# end */
```

#### マークダウンによる文芸的プログラミング（`.snap.md`）

SnapSQLは、SQLテンプレートをドキュメント、設計ノート、包括的なテストと組み合わせる`.snap.md`ファイルによる文芸的プログラミングをサポートします：

````markdown
# ユーザークエリテンプレート

## 設計概要

このテンプレートは、動的フィールド選択とフィルタリング機能を持つユーザーデータ取得を処理します。

### 要件
- ユーザー権限に基づく条件付きメールフィールド包含のサポート
- 環境固有のテーブル選択（dev/staging/prod）
- ユーザーステータスによるオプションフィルタリング
- 設定可能なソート

## SQLテンプレート

```sql
SELECT 
    id, 
    name
    /*# if include_email */,
    email
    /*# end */
FROM users_/*= env */
/*# if filters.active */
WHERE active = true
/*# end */
/*# if sort_by != "" */
ORDER BY /*= sort_by */
/*# end */
```

## テストケース

### テストケース1: 基本クエリ
**入力パラメータ:**
```json
{
    "include_email": false,
    "env": "prod",
    "filters": {"active": false},
    "sort_by": ""
}
```

**期待される出力:**
```sql
SELECT id, name FROM users_prod
```

### テストケース2: すべてのオプション付き完全クエリ
**入力パラメータ:**
```json
{
    "include_email": true,
    "env": "dev",
    "filters": {"active": true},
    "sort_by": "created_at"
}
```

**期待される出力:**
```sql
SELECT id, name, email FROM users_dev WHERE active = true ORDER BY created_at
```

## モックデータ例

```yaml
# ユーザークエリのモック応答
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    active: true
    created_at: "2024-01-15T10:30:00Z"
  - id: 2
    name: "Jane Smith"
    email: "jane@example.com"
    active: true
    created_at: "2024-02-20T14:45:00Z"
  - id: 3
    name: "Bob Wilson"
    email: "bob@example.com"
    active: false
    created_at: "2024-03-10T09:15:00Z"
```
````

### 2. 中間ファイルのビルド *（計画中）*

```bash
snapsql build -i queries/ -o generated/
```

### 3. ランタイムライブラリの使用 *（計画中）*

#### Go例 *（計画中）*
```go
package main

import (
    "github.com/shibukawa/snapsql"
)

func main() {
    engine := snapsql.New("generated/")
    
    params := map[string]any{
        "include_email": true,
        "env": "prod",
        "filters": map[string]any{
            "active": true,
        },
        "sort_by": "created_at",
    }
    
    query, args, err := engine.Build("users", params)
    if err != nil {
        panic(err)
    }
    
    // データベースドライバーで実行
    rows, err := db.Query(query, args...)
}
```

#### Python例 *（計画中）*
```python
import snapsql

engine = snapsql.Engine("generated/")

params = {
    "include_email": True,
    "env": "prod", 
    "filters": {
        "active": True
    },
    "sort_by": "created_at"
}

query, args = engine.build("users", params)

# データベースドライバーで実行
cursor.execute(query, args)
```

### 現在の使用方法（開発）

現在、中間ファイルの生成とSQLテンプレートの解析のためのCLIツールとしてSnapSQLを使用できます：

```bash
# 新しいプロジェクトを初期化
snapsql init

# 中間JSONファイルを生成
snapsql generate

# 特定の言語を生成（実装時）
snapsql generate --lang go

# テンプレートを検証
snapsql validate
```

また、SQLテンプレートの解析のためのGoライブラリとしてSnapSQLを使用することもできます：

```go
package main

import (
    "fmt"
    "github.com/shibukawa/snapsql/parser"
    "github.com/shibukawa/snapsql/tokenizer"
)

func main() {
    sql := `SELECT id, name FROM users WHERE active = /*= active */true`
    
    tokens, err := tokenizer.Tokenize(sql)
    if err != nil {
        panic(err)
    }
    
    ast, err := parser.Parse(tokens)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Parsed AST: %+v\n", ast)
}
```

## 開発状況

🚧 **開発中** - このプロジェクトは現在設計と初期開発段階にあります。

## 貢献

貢献を歓迎します！イシューやプルリクエストをお気軽に提出してください。

## ライセンス

- **ビルドツール（`snapsql`）**: AGPL-3.0でライセンス
- **ランタイムライブラリ**: Apache-2.0でライセンス

このデュアルライセンスアプローチにより、ビルドツールはオープンソースのままで、ランタイムライブラリは様々なプロジェクトで柔軟に使用できます。

## リポジトリ

[https://github.com/shibukawa/snapsql](https://github.com/shibukawa/snapsql)
