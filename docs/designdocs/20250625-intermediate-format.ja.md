# SnapSQL 中間形式仕様

**ドキュメントバージョン:** 2.0  
**日付:** 2025-07-28  
**ステータス:** 実装済み

## 概要

このドキュメントは、SnapSQLテンプレートの中間JSON形式を定義します。中間形式は、SQLテンプレートパーサーとコードジェネレータ間の橋渡しとして機能し、解析されたSQLテンプレートのメタデータ、CEL式、関数定義の言語非依存表現を提供します。

## 設計目標

### 1. 言語非依存
- あらゆるプログラミング言語で利用可能なJSON形式
- 言語固有の構造や前提条件なし
- SQL構造と言語固有メタデータの明確な分離

### 2. 完全な情報保持
- テンプレートメタデータと関数定義
- CEL式の完全な抽出と型情報

### 3. コード生成対応
- テンプレートベースのコード生成に適した構造化データ
- 強く型付けされた言語のための型情報
- 関数シグネチャ情報
- パラメータ順序の保持

### 4. 拡張可能
- 将来のSnapSQL機能のサポート
- 後方互換性のためのバージョン管理形式
- カスタムジェネレータ用のプラグインフレンドリー構造

## 中間形式に含まれる項目

### トップレベル構造

```json
{
  "format_version": "1",
  "description": "ユーザーIDによるユーザー情報取得",
  "function_name": "get_user_by_id", 
  "parameters": [/* パラメータ定義 */],
  "implicit_parameters": [/* 暗黙的パラメータ定義 */],
  "instructions": [/* 命令列 */],
  "expressions": [/* CEL式リスト */],
  "envs": [/* 環境変数の階層構造 */],
  "responses": [/* レスポンス型定義 */],
  "response_affinity": {/* レスポンス親和性情報 */}
}
```

### 各項目の詳細

#### 1. **format_version** (string)
- **目的**: 中間形式のバージョン管理
- **値**: 現在は `"1"`
- **用途**: 後方互換性の保証、パーサーの対応バージョン確認

#### 2. **description** (string)
- **目的**: テンプレートの説明・概要
- **抽出元**: 関数定義
- **用途**: ドキュメント生成、コメント出力、開発者向け説明

#### 3. **function_name** (string)
- **目的**: 生成される関数の名前
- **抽出元**: 関数定義
- **用途**: コード生成時の関数名、snake_case形式

#### 4. **parameters** (array)
- **目的**: 明示的なテンプレートパラメータの定義
- **構造**: `{"name": string, "type": string}`
- **抽出元**: 関数定義
- **用途**: 型安全な関数シグネチャ生成、バリデーション

#### 5. **implicit_parameters** (array)
- **目的**: システムが自動提供するパラメータ（システムフィールド等）
- **構造**: `{"name": string, "type": string, "default": any}`
- **抽出元**: システムフィールド設定、LIMIT/OFFSET句の解析
- **用途**: ランタイムでの自動パラメータ注入

#### 6. **instructions** (array)
- **目的**: SQLテンプレートの実行可能表現
- **構造**: 命令オブジェクトの配列
- **抽出元**: トークン解析、制御フロー解析
- **用途**: ランタイムでのSQL動的生成

#### 7. **expressions** (array)
- **目的**: テンプレート内のすべてのCEL式
- **構造**: CEL式文字列の配列
- **抽出元**: SQL中のディレクティブコメント
- **用途**: CEL環境の構築、式の事前コンパイル

#### 8. **envs** (array)
- **目的**: ループ変数の階層構造
- **構造**: `[[{"name": string, "type": string}]]`
- **抽出元**: SQL中のディレクティブコメント
- **用途**: ネストしたループでの変数スコープ管理

#### 9. **responses** (array)
- **目的**: クエリ結果の型定義
- **構造**: レスポンス型オブジェクトの配列
- **抽出元**: 型推論
- **用途**: 結果型の生成、型安全なレスポンス処理

#### 10. **response_affinity** (object)
- **目的**: クエリ結果の構造情報
- **構造**: テーブル、カラム、型情報
- **抽出元**: 型推論
- **用途**: 結果型の生成、ORM連携

## 情報源マッピング表

| 中間形式項目 | 主要情報源 |
|-------------|-----------|
| `format_version` | 固定値 |
| `description` | 関数定義 |
| `function_name` | 関数定義 |
| `parameters` | 関数定義 |
| `implicit_parameters` | システムフィールド設定 |
| `instructions` | トークン列 |
| `expressions` | SQL中のディレクティブコメント |
| `envs` | SQL中のディレクティブコメント |
| `responses` | 型推論 |
| `response_affinity` | 型推論 |

### 詳細な情報源

#### 関数定義形式（SQLファイル）
```sql
/*#
function_name: get_user_by_id
description: ユーザーIDによるユーザー情報取得
parameters:
  user_id: int
  include_details: bool
*/
```

#### 関数定義形式（Markdownファイル）
```markdown
## Function Definition
- **Name**: get_user_by_id
- **Description**: ユーザーIDによるユーザー情報取得

## Parameters
```yaml
user_id: int
include_details: bool
```
```

#### SQL中のディレクティブコメント
```sql
-- 変数置換
/*= user_id */

-- 条件分岐
/*# if min_age > 0 */

-- ループ
/*# for dept : departments */
```

#### システムフィールド設定（snapsql.yaml）
```yaml
system:
  fields:
    - name: updated_at
      type: timestamp
      on_update:
        parameter: implicit
```

## 処理フロー

### 1. パーサーフェーズ（parser パッケージ）

#### parserstep2: 基本構造解析
- SQL文字列をトークン化
- StatementNode（AST）の基本構造を構築
- WITH句、各種clause の識別

#### parserstep3: 構文検証
- 基本的な構文エラーの検出

#### parserstep4: Clause詳細解析
各clause別に詳細な解析と検証を実行：

**SELECT文の処理順序:**
1. `finalizeSelectClause()` - SELECT句の詳細解析
2. `finalizeFromClause()` - FROM句の詳細解析  
3. `emptyCheck(WHERE)` - WHERE句の空チェック
4. `finalizeGroupByClause()` - GROUP BY句の詳細解析
5. `finalizeHavingClause()` - HAVING句の詳細解析（GROUP BYとの関連チェック）
6. `finalizeOrderByClause()` - ORDER BY句の詳細解析
7. `finalizeLimitOffsetClause()` - LIMIT/OFFSET句の詳細解析
8. `emptyCheck(FOR)` - FOR句の空チェック

**INSERT文の処理順序:**
1. `finalizeInsertIntoClause()` - INSERT INTO句の詳細解析（カラムリスト解析）
2. `InsertIntoStatement.Columns`への反映
3. `emptyCheck(WITH)` - WITH句の空チェック
4. SELECT部分がある場合は上記SELECT文の処理を実行
5. `finalizeReturningClause()` - RETURNING句の詳細解析

**UPDATE文の処理順序:**
1. `finalizeUpdateClause()` - UPDATE句の詳細解析
2. `finalizeSetClause()` - SET句の詳細解析
3. `emptyCheck(WHERE)` - WHERE句の空チェック
4. `finalizeReturningClause()` - RETURNING句の詳細解析

**DELETE文の処理順序:**
1. `finalizeDeleteFromClause()` - DELETE FROM句の詳細解析
2. `emptyCheck(WHERE)` - WHERE句の空チェック
3. `finalizeReturningClause()` - RETURNING句の詳細解析

#### parserstep5: 高度な処理
1. `expandArraysInValues()` - VALUES句での配列展開
2. `detectDummyRanges()` - ダミー値の検出
3. `applyImplicitIfConditions()` - LIMIT/OFFSET句への暗黙的if条件適用
4. `validateAndLinkDirectives()` - ディレクティブの検証とリンク

### 2. 中間形式生成フェーズ（intermediate パッケージ）

#### 前処理
1. `ExtractFromStatement()` - CEL式と環境変数の抽出
2. 関数定義からのパラメータ抽出
3. `extractSystemFieldsInfo()` - システムフィールド情報の抽出

#### システムフィールド処理
1. `CheckSystemFields()` - システムフィールドの検証と暗黙パラメータ生成
2. Statement別のシステムフィールド追加:
   - UPDATE文: `AddSystemFieldsToUpdate()` - SET句への追加
   - INSERT文: `AddSystemFieldsToInsert()` - カラムリストとVALUES句への追加

#### 命令生成
1. `extractTokensFromStatement()` - StatementNodeからトークン列抽出
2. `detectDialectPatterns()` - 方言固有パターンの検出
3. `generateInstructions()` - 実行命令の生成
4. `detectResponseAffinity()` - レスポンス親和性の検出

## Clause別処理詳細

### INSERT INTO Clause
- **処理内容**: カラムリストの解析、テーブル名の抽出
- **出力**: `InsertIntoClause.Columns[]`, `InsertIntoClause.Table`
- **後続処理**: `InsertIntoStatement.Columns`への反映

### VALUES Clause  
- **処理内容**: 値リストの解析、配列展開
- **出力**: 値トークンの構造化
- **後続処理**: システムフィールド値の追加

### SELECT Clause
- **処理内容**: 選択フィールドの解析、関数呼び出しの検証
- **出力**: フィールドリストの構造化

### FROM Clause
- **処理内容**: テーブル参照、JOIN構文の解析
- **出力**: テーブル参照の構造化

### WHERE/HAVING Clause
- **処理内容**: 条件式の検証（空チェックのみ）
- **出力**: 構文エラーの検出

### GROUP BY Clause
- **処理内容**: グループ化フィールドの解析
- **出力**: グループ化条件の構造化

### ORDER BY Clause
- **処理内容**: ソート条件の解析、ASC/DESC指定の検証
- **出力**: ソート条件の構造化

### LIMIT/OFFSET Clause
- **処理内容**: 制限値の検証、暗黙的if条件の適用
- **出力**: 制限条件の構造化

### SET Clause (UPDATE)
- **処理内容**: 更新フィールドと値の解析
- **出力**: 更新条件の構造化
- **後続処理**: システムフィールドの追加

### RETURNING Clause
- **処理内容**: 戻り値フィールドの解析
- **出力**: 戻り値の構造化

## 方言対応（Database Dialect Support）

### 概要

SnapSQLは複数のデータベース方言（PostgreSQL、MySQL、SQLite）をサポートし、方言固有の構文を自動的に変換します。中間形式では、方言固有の処理を命令として表現します。

### 対応する方言固有構文

#### 1. **PostgreSQL Cast演算子（`::`）**
```sql
-- PostgreSQL固有
SELECT price::DECIMAL(10,2) FROM products

-- 標準SQL変換
SELECT CAST(price AS DECIMAL(10,2)) FROM products
```

#### 2. **MySQL LIMIT構文**
```sql
-- MySQL固有
SELECT * FROM users LIMIT 10, 20

-- 標準SQL変換  
SELECT * FROM users LIMIT 20 OFFSET 10
```

#### 3. **SQLite型アフィニティ**
```sql
-- SQLite固有
CREATE TABLE users (id INTEGER PRIMARY KEY)

-- 他DB変換
CREATE TABLE users (id SERIAL PRIMARY KEY)
```

### 中間形式での表現

方言固有の処理は専用の命令として表現されます：

```json
{
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT ", "pos": "1:1"},
    {"op": "EMIT_IF_DIALECT", "dialect": "postgresql", "sql_fragment": "price::DECIMAL(10,2)", "pos": "1:8"},
    {"op": "EMIT_IF_DIALECT", "dialect": "mysql", "sql_fragment": "CAST(price AS DECIMAL(10,2))", "pos": "1:8"},
    {"op": "EMIT_IF_DIALECT", "dialect": "sqlite", "sql_fragment": "CAST(price AS DECIMAL(10,2))", "pos": "1:8"},
    {"op": "EMIT_STATIC", "value": " FROM products", "pos": "1:25"}
  ]
}
```

### 方言検出パターン

#### PostgreSQL Cast検出
```go
// detectPostgreSQLCast関数
// パターン: expression::type
if token.Type == tok.DOUBLE_COLON {
    // 前の式と後の型を解析
    // CAST(expression AS type)に変換
}
```

#### MySQL関数検出
```go
// detectMySQLFunctions関数  
// NOW(), RAND()等のMySQL固有関数を検出
// 他方言での等価関数に変換
```

### 実行時の方言選択

ランタイムライブラリは、接続先データベースに応じて適切な方言の命令を実行：

```go
// 実行時の方言選択例
switch currentDialect {
case "postgresql":
    executePostgreSQLInstructions()
case "mysql": 
    executeMySQLInstructions()
case "sqlite":
    executeSQLiteInstructions()
}
```

## 末尾カンマ・AND・OR処理（Trailing Delimiter Handling）

### 概要

SnapSQLは、条件分岐によって動的に追加・削除される要素に対して、末尾の区切り文字（カンマ、AND、OR）を自動的に処理します。これにより、開発者は構文エラーを気にせずにテンプレートを記述できます。

### 対象となる区切り文字

#### 1. **カンマ（`,`）**
- SELECT句のフィールドリスト
- INSERT文のカラムリスト
- VALUES句の値リスト

#### 2. **AND演算子**
- WHERE句の条件結合
- HAVING句の条件結合

#### 3. **OR演算子**
- WHERE句の条件結合（選択的）

### 処理例

#### SELECT句での末尾カンマ処理

**テンプレート:**
```sql
SELECT 
    id,
    name,
    /*# if include_email */
    email,
    /*# end */
    /*# if include_phone */
    phone,
    /*# end */
    created_at
FROM users
```

**条件による出力:**
```sql
-- include_email=true, include_phone=false の場合
SELECT id, name, email, created_at FROM users

-- include_email=false, include_phone=true の場合  
SELECT id, name, phone, created_at FROM users

-- 両方false の場合
SELECT id, name, created_at FROM users
```

#### WHERE句でのAND処理

**テンプレート:**
```sql
SELECT * FROM users 
WHERE active = true
    /*# if min_age > 0 */
    AND age >= /*= min_age */18
    /*# end */
    /*# if department != "" */
    AND department = /*= department */'Engineering'
    /*# end */
```

**条件による出力:**
```sql
-- min_age=25, department="Sales" の場合
SELECT * FROM users WHERE active = true AND age >= 25 AND department = 'Sales'

-- min_age=0, department="" の場合
SELECT * FROM users WHERE active = true

-- min_age=25, department="" の場合
SELECT * FROM users WHERE active = true AND age >= 25
```

### 中間形式での表現

末尾区切り文字の処理は、境界検出（`BOUNDARY`）命令で表現されます：

```json
{
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name", "pos": "1:1"},
    {"op": "IF", "condition": "include_email", "pos": "4:5"},
    {"op": "BOUNDARY", "pos": "4:5"},
    {"op": "EMIT_STATIC", "value": ", email", "pos": "5:5"},
    {"op": "END", "pos": "6:5"},
    {"op": "IF", "condition": "include_phone", "pos": "7:5"},
    {"op": "BOUNDARY", "pos": "7:5"},
    {"op": "EMIT_STATIC", "value": ", phone", "pos": "8:5"},
    {"op": "END", "pos": "9:5"},
    {"op": "EMIT_STATIC", "value": ", created_at FROM users", "pos": "11:5"}
  ]
}
```

### 境界検出アルゴリズム

#### 1. **境界トークンの識別**
```go
// 境界となるトークンタイプ
switch token.Type {
case tok.FROM, tok.WHERE, tok.ORDER, tok.GROUP, 
     tok.HAVING, tok.LIMIT, tok.OFFSET, tok.UNION,
     tok.CLOSED_PARENS:
    return true
}
```

#### 2. **区切り文字の種類判定**
```go
// 文脈に応じた区切り文字の決定
func detectBoundaryDelimiter(context string) string {
    switch context {
    case "SELECT_FIELDS":
        return ","
    case "WHERE_CONDITIONS":
        return "AND"
    case "VALUES_LIST":
        return ","
    }
}
```

#### 3. **実行時の区切り文字挿入**
```go
// ランタイムでの動的区切り文字処理
if hasNextElement && !isLastElement {
    output += delimiter // カンマまたはAND/ORを挿入
}
```

### 特殊ケース

#### 1. **LIMIT/OFFSET句の条件付き出力**
```sql
-- LIMIT/OFFSET句がない場合、システムが自動追加
SELECT * FROM users

-- 出力: システムLIMIT/OFFSETが利用可能な場合のみ追加
SELECT * FROM users LIMIT 10 OFFSET 0
```

中間形式での表現：
```json
{
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT * FROM users", "pos": "1:1"},
    {"op": "IF_SYSTEM_LIMIT", "pos": "0:0"},
    {"op": "EMIT_STATIC", "value": " LIMIT ", "pos": "0:0"},
    {"op": "EMIT_SYSTEM_LIMIT", "pos": "0:0"},
    {"op": "END", "pos": "0:0"},
    {"op": "IF_SYSTEM_OFFSET", "pos": "0:0"},
    {"op": "EMIT_STATIC", "value": " OFFSET ", "pos": "0:0"},
    {"op": "EMIT_SYSTEM_OFFSET", "pos": "0:0"},
    {"op": "END", "pos": "0:0"}
  ]
}
```

#### 2. **ネストした条件での処理**
```sql
SELECT *
FROM users
WHERE (
    /*# if condition1 */
    field1 = 'value1'
    /*# end */
    /*# if condition2 */
    AND field2 = 'value2'  
    /*# end */
)
```

#### 3. **OR演算子の処理**
```sql
WHERE (
    /*# if search_name */
    name LIKE /*= search_name */'%John%'
    /*# end */
    /*# if search_email */
    OR email LIKE /*= search_email */'%john%'
    /*# end */
)
```

### 利点

1. **構文エラーの防止**: 手動での区切り文字管理が不要
2. **可読性の向上**: テンプレートがより自然な形で記述可能
3. **保守性**: 条件の追加・削除時の構文エラーリスクを削減
4. **柔軟性**: 複雑な条件分岐でも正しいSQLを生成
5. **システム統合**: LIMIT/OFFSET句の自動追加によるページネーション対応

## 課題と改善案

### 現在の問題点
1. **StatementNode更新の複雑さ**: システムフィールド追加でStatementNodeを変更後、再度トークン抽出が必要
2. **処理の分散**: clause別処理がparserstep4とintermediateに分散
3. **トークンレベル操作の複雑さ**: InsertIntoClauseのトークン直接操作が必要

### 改善案: パイプライン処理
```
SQL文字列
↓
tokenizer: SQL → Token列
↓  
parser: Token列 → StatementNode (構造解析のみ)
↓
システムフィールド解析: StatementNode → ImplicitParameter[]
↓
トークン処理パイプライン:
  Token列 → clause別変換 → システムフィールド挿入 → 命令生成
↓
中間形式JSON
```

**利点:**
- StatementNodeは構造解析にのみ使用
- 各clause別にトークン変換ルールを定義
- テストしやすい独立したパイプライン段階
- トークンレベルでの柔軟な操作が可能
- 制御フロー構造（if/forブロック）の命令表現

### 1. パーサーフェーズ（parser パッケージ）

#### parserstep2: 基本構造解析
- SQL文字列をトークン化
- StatementNode（AST）の基本構造を構築
- WITH句、各種clause の識別

#### parserstep3: 構文検証
- 基本的な構文エラーの検出

#### parserstep4: Clause詳細解析
各clause別に詳細な解析と検証を実行：

**SELECT文の処理順序:**
1. `finalizeSelectClause()` - SELECT句の詳細解析
2. `finalizeFromClause()` - FROM句の詳細解析  
3. `emptyCheck(WHERE)` - WHERE句の空チェック
4. `finalizeGroupByClause()` - GROUP BY句の詳細解析
5. `finalizeHavingClause()` - HAVING句の詳細解析（GROUP BYとの関連チェック）
6. `finalizeOrderByClause()` - ORDER BY句の詳細解析
7. `finalizeLimitOffsetClause()` - LIMIT/OFFSET句の詳細解析
8. `emptyCheck(FOR)` - FOR句の空チェック

**INSERT文の処理順序:**
1. `finalizeInsertIntoClause()` - INSERT INTO句の詳細解析（カラムリスト解析）
2. `InsertIntoStatement.Columns`への反映
3. `emptyCheck(WITH)` - WITH句の空チェック
4. SELECT部分がある場合は上記SELECT文の処理を実行
5. `finalizeReturningClause()` - RETURNING句の詳細解析

**UPDATE文の処理順序:**
1. `finalizeUpdateClause()` - UPDATE句の詳細解析
2. `finalizeSetClause()` - SET句の詳細解析
3. `emptyCheck(WHERE)` - WHERE句の空チェック
4. `finalizeReturningClause()` - RETURNING句の詳細解析

**DELETE文の処理順序:**
1. `finalizeDeleteFromClause()` - DELETE FROM句の詳細解析
2. `emptyCheck(WHERE)` - WHERE句の空チェック
3. `finalizeReturningClause()` - RETURNING句の詳細解析

#### parserstep5: 高度な処理
1. `expandArraysInValues()` - VALUES句での配列展開
2. `detectDummyRanges()` - ダミー値の検出
3. `applyImplicitIfConditions()` - LIMIT/OFFSET句への暗黙的if条件適用
4. `validateAndLinkDirectives()` - ディレクティブの検証とリンク

### 2. 中間形式生成フェーズ（intermediate パッケージ）

#### 前処理
1. `ExtractFromStatement()` - CEL式と環境変数の抽出
2. 関数定義からのパラメータ抽出
3. `extractSystemFieldsInfo()` - システムフィールド情報の抽出

#### システムフィールド処理
1. `CheckSystemFields()` - システムフィールドの検証と暗黙パラメータ生成
2. Statement別のシステムフィールド追加:
   - UPDATE文: `AddSystemFieldsToUpdate()` - SET句への追加
   - INSERT文: `AddSystemFieldsToInsert()` - カラムリストとVALUES句への追加

#### 命令生成
1. `extractTokensFromStatement()` - StatementNodeからトークン列抽出
2. `detectDialectPatterns()` - 方言固有パターンの検出
3. `generateInstructions()` - 実行命令の生成
4. `detectResponseAffinity()` - レスポンス親和性の検出

## Clause別処理詳細

### INSERT INTO Clause
- **処理内容**: カラムリストの解析、テーブル名の抽出
- **出力**: `InsertIntoClause.Columns[]`, `InsertIntoClause.Table`
- **後続処理**: `InsertIntoStatement.Columns`への反映

### VALUES Clause  
- **処理内容**: 値リストの解析、配列展開
- **出力**: 値トークンの構造化
- **後続処理**: システムフィールド値の追加

### SELECT Clause
- **処理内容**: 選択フィールドの解析、関数呼び出しの検証
- **出力**: フィールドリストの構造化

### FROM Clause
- **処理内容**: テーブル参照、JOIN構文の解析
- **出力**: テーブル参照の構造化

### WHERE/HAVING Clause
- **処理内容**: 条件式の検証（空チェックのみ）
- **出力**: 構文エラーの検出

### GROUP BY Clause
- **処理内容**: グループ化フィールドの解析
- **出力**: グループ化条件の構造化

### ORDER BY Clause
- **処理内容**: ソート条件の解析、ASC/DESC指定の検証
- **出力**: ソート条件の構造化

### LIMIT/OFFSET Clause
- **処理内容**: 制限値の検証、暗黙的if条件の適用
- **出力**: 制限条件の構造化

### SET Clause (UPDATE)
- **処理内容**: 更新フィールドと値の解析
- **出力**: 更新条件の構造化
- **後続処理**: システムフィールドの追加

### RETURNING Clause
- **処理内容**: 戻り値フィールドの解析
- **出力**: 戻り値の構造化

## 課題と改善案

### 現在の問題点
1. **StatementNode更新の複雑さ**: システムフィールド追加でStatementNodeを変更後、再度トークン抽出が必要
2. **処理の分散**: clause別処理がparserstep4とintermediateに分散
3. **トークンレベル操作の複雑さ**: InsertIntoClauseのトークン直接操作が必要

### 改善案: パイプライン処理
```
SQL文字列
↓
tokenizer: SQL → Token列
↓  
parser: Token列 → StatementNode (構造解析のみ)
↓
システムフィールド解析: StatementNode → ImplicitParameter[]
↓
トークン処理パイプライン:
  Token列 → clause別変換 → システムフィールド挿入 → 命令生成
↓
中間形式JSON
```

**利点:**
- StatementNodeは構造解析にのみ使用
- 各clause別にトークン変換ルールを定義
- テストしやすい独立したパイプライン段階
- トークンレベルでの柔軟な操作が可能
- 制御フロー構造（if/forブロック）の命令表現

### 3. コード生成対応
- テンプレートベースのコード生成に適した構造化データ
- 強く型付けされた言語のための型情報
- 関数シグネチャ情報
- パラメータ順序の保持

### 4. 拡張可能
- 将来のSnapSQL機能のサポート
- 後方互換性のためのバージョン管理形式
- カスタムジェネレータ用のプラグインフレンドリー構造

## 中間形式構造

### トップレベル構造

```json
{
  "format_version": "1",
  "name": "get_user_by_id",
  "function_name": "get_user_by_id",
  "parameters": [/* パラメータ定義 */],
  "implicit_parameters": [/* 暗黙的パラメータ定義 */],
  "instructions": [/* 命令列 */],
  "expressions": [/* CEL式リスト */],
  "envs": [/* 環境変数の階層構造 */]
}
```

## CEL式の抽出

SnapSQLでは、テンプレート内のすべてのCEL式を抽出します。これには以下が含まれます：

1. **変数置換**: `/*= expression */` 形式の式
2. **条件式**: `/*# if condition */` や `/*# elseif condition */` の条件部分
3. **ループ式**: `/*# for variable : collection */` のコレクション部分

### CEL式の例

```sql
-- 変数置換
SELECT * FROM users WHERE id = /*= user_id */123

-- 条件式
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */

-- ループ式
/*# for dept : departments */
SELECT /*= dept.name */'Engineering'
/*# end */

-- 複雑な式
ORDER BY /*= sort_field + " " + (sort_direction != "" ? sort_direction : "ASC") */name
```

### 中間形式での表現

```json
{
  "expressions": [
    "user_id",
    "min_age > 0",
    "min_age",
    "departments",
    "dept",
    "dept.name",
    "sort_field + \" \" + (sort_direction != \"\" ? sort_direction : \"ASC\")"
  ],
  "envs": [
    [{"name": "dept", "type": "any"}]
  ]
}
```

`envs` セクションには、ループ変数の階層構造が含まれます。各レベルは、そのレベルで定義されたループ変数のリストを含みます。

## 関数定義セクション

関数定義は、テンプレートのヘッダーコメントから抽出されたメタデータを含みます。

```json
{
  "description": "ユーザーIDによるユーザー情報取得",
  "function_name": "get_user_by_id",
  "parameters": [
    {"name": "user_id", "type": "int"},
    {"name": "include_details", "type": "bool"}
  ]
}
```

### パラメータ定義

パラメータ定義は、テンプレートのヘッダーコメントまたはMarkdownのParametersセクションから抽出されます。

```yaml
# SQLファイルのヘッダーコメント
/*#
function_name: get_user_by_id
parameters:
  user_id: int
  include_details: bool
*/
```

````markdown
# Markdownファイルのパラメータセクション
## Parameters

```yaml
user_id: int
include_details: bool
```
````

## 命令セット

命令セットは、SQLテンプレートの実行可能な表現です。現在の実装では、以下の命令タイプがサポートされています。

### 命令タイプ

#### 基本出力命令
- **EMIT_STATIC**: 静的なSQLテキストを出力
- **EMIT_EVAL**: CEL式を評価してパラメータを出力

#### 制御フロー命令
- **IF**: 条件分岐の開始
- **ELSE_IF**: else if条件
- **ELSE**: else分岐
- **END**: 制御ブロックの終了

#### ループ命令
- **LOOP_START**: forループの開始
- **LOOP_END**: forループの終了

#### システム命令
- **EMIT_SYSTEM_LIMIT**: システムLIMIT句の出力
- **EMIT_SYSTEM_OFFSET**: システムOFFSET句の出力
- **EMIT_SYSTEM_VALUE**: システムフィールド値の出力

#### 境界処理命令
- **BOUNDARY**: 末尾区切り文字（カンマ、AND、OR）の処理

#### 方言対応命令
- **EMIT_IF_DIALECT**: データベース方言固有のSQL断片出力

### 命令の例

```json
{"op": "EMIT_STATIC", "value": "SELECT id, name FROM users WHERE ", "pos": "1:1"}
{"op": "EMIT_EVAL", "param": "user_id", "pos": "1:43"}
{"op": "IF", "condition": "min_age > 0", "pos": "2:1"}
{"op": "BOUNDARY", "pos": "2:1"}
{"op": "EMIT_STATIC", "value": " AND age >= ", "pos": "3:1"}
{"op": "EMIT_EVAL", "param": "min_age", "pos": "3:12"}
{"op": "ELSE_IF", "condition": "max_age > 0", "pos": "4:1"}
{"op": "BOUNDARY", "pos": "4:1"}
{"op": "EMIT_STATIC", "value": " AND age <= ", "pos": "5:1"}
{"op": "EMIT_EVAL", "param": "max_age", "pos": "5:12"}
{"op": "ELSE", "pos": "6:1"}
{"op": "EMIT_STATIC", "value": " -- No age filter", "pos": "7:1"}
{"op": "END", "pos": "8:1"}
{"op": "LOOP_START", "variable": "dept", "collection": "departments", "pos": "9:1"}
{"op": "EMIT_EVAL", "param": "dept.name", "pos": "10:5"}
{"op": "LOOP_END", "pos": "11:1"}
{"op": "EMIT_SYSTEM_LIMIT", "default_value": "100", "pos": "12:1"}
{"op": "EMIT_SYSTEM_OFFSET", "default_value": "0", "pos": "13:1"}
{"op": "EMIT_IF_DIALECT", "dialect": "postgresql", "sql_fragment": "price::DECIMAL(10,2)", "pos": "14:1"}
```

## 実装例

### 単純な変数置換

```sql
SELECT id, name, email FROM users WHERE id = /*= user_id */123
```

中間形式：

```json
{
  "format_version": "1",
  "description": "ユーザーIDによるユーザー取得",
  "function_name": "get_user_by_id",
  "parameters": [{"name": "user_id", "type": "int"}],
  "expressions": ["user_id"],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name, email FROM users WHERE id = ", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "user_id", "pos": "1:43"}
  ]
}
```

### 条件付きクエリ

```sql
SELECT id, name, age, department 
FROM users
WHERE 1=1
/*# if min_age > 0 */
AND age >= /*= min_age */18
/*# end */
/*# if max_age > 0 */
AND age <= /*= max_age */65
/*# end */
```

中間形式：

```json
{
  "format_version": "1",
  "description": "年齢条件によるユーザー検索",
  "function_name": "get_filtered_users",
  "parameters": [
    {"name": "min_age", "type": "int"},
    {"name": "max_age", "type": "int"}
  ],
  "expressions": ["min_age > 0", "min_age", "max_age > 0", "max_age"],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name, age, department \nFROM users\nWHERE 1=1", "pos": "1:1"},
    {"op": "IF", "condition": "min_age > 0", "pos": "4:1"},
    {"op": "EMIT_STATIC", "value": "\nAND age >= ", "pos": "5:1"},
    {"op": "EMIT_EVAL", "param": "min_age", "pos": "5:11"},
    {"op": "END", "pos": "6:1"},
    {"op": "IF", "condition": "max_age > 0", "pos": "7:1"},
    {"op": "EMIT_STATIC", "value": "\nAND age <= ", "pos": "8:1"},
    {"op": "EMIT_EVAL", "param": "max_age", "pos": "8:11"},
    {"op": "END", "pos": "9:1"}
  ]
}
```

### IF-ELSE_IF-ELSE構造

```sql
SELECT 
    id,
    name,
    /*# if user_type == "admin" */
    'Administrator' as role
    /*# elseif user_type == "manager" */
    'Manager' as role
    /*# else */
    'User' as role
    /*# end */
FROM users
WHERE age >= /*= age */18
```

中間形式：

```json
{
  "format_version": "1",
  "name": "get_user_with_role",
  "function_name": "get_user_with_role",
  "parameters": [
    {"name": "user_type", "type": "string"},
    {"name": "age", "type": "int"}
  ],
  "expressions": ["user_type == \"admin\"", "user_type == \"manager\"", "age"],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "SELECT id, name,", "pos": "1:1"},
    {"op": "IF", "condition": "user_type == \"admin\"", "pos": "4:5"},
    {"op": "EMIT_STATIC", "value": "'Administrator' as role", "pos": "5:5"},
    {"op": "ELSE_IF", "condition": "user_type == \"manager\"", "pos": "6:5"},
    {"op": "EMIT_STATIC", "value": "'Manager' as role", "pos": "7:5"},
    {"op": "ELSE", "pos": "8:5"},
    {"op": "EMIT_STATIC", "value": "'User' as role", "pos": "9:5"},
    {"op": "END", "pos": "10:5"},
    {"op": "EMIT_STATIC", "value": "FROM users WHERE age >= ", "pos": "11:1"},
    {"op": "EMIT_EVAL", "param": "age", "pos": "12:14"},
    {"op": "EMIT_STATIC", "value": "18", "pos": "12:24"}
  ]
}
```

### ネストされたループ

```sql
INSERT INTO sub_departments (id, name, department_code, department_name)
VALUES
/*# for dept : departments */
    /*# for sub : dept.sub_departments */
    (/*= dept.department_code + "-" + sub.id */'1-101', /*= sub.name */'Engineering Team A', /*= dept.department_code */'1', /*= dept.department_name */'Engineering')
    /*# end */
/*# end */;
```

中間形式：

```json
{
  "format_version": "1",
  "name": "insert_all_sub_departments",
  "function_name": "insert_all_sub_departments",
  "parameters": [{"name": "departments", "type": "any"}],
  "expressions": [
    "departments", "dept", "dept.sub_departments", "sub",
    "dept.department_code + \"-\" + sub.id", "sub.name",
    "dept.department_code", "dept.department_name"
  ],
  "envs": [
    [{"name": "dept", "type": "any"}],
    [{"name": "sub", "type": "any"}]
  ],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "INSERT INTO sub_departments (id, name, department_code, department_name) VALUES (", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "dept.department_code + \"-\" + sub.id", "pos": "5:6"},
    {"op": "EMIT_STATIC", "value": "'1-101',", "pos": "5:48"},
    {"op": "EMIT_EVAL", "param": "sub.name", "pos": "5:57"},
    {"op": "EMIT_STATIC", "value": "'Engineering Team A',", "pos": "5:72"},
    {"op": "EMIT_EVAL", "param": "dept.department_code", "pos": "5:94"},
    {"op": "EMIT_STATIC", "value": "'1',", "pos": "5:121"},
    {"op": "EMIT_EVAL", "param": "dept.department_name", "pos": "5:126"},
    {"op": "EMIT_STATIC", "value": "'Engineering') ;", "pos": "5:153"}
  ]
}
```

### 複雑な式とシステム命令

```sql
SELECT 
  id, 
  name,
  /*= display_name ? username : "Anonymous" */'Anonymous'
FROM users
WHERE 
  /*# if start_date != "" && end_date != "" */
  created_at BETWEEN /*= start_date */'2023-01-01' AND /*= end_date */'2023-12-31'
  /*# end */
  /*# if sort_field != "" */
ORDER BY /*= sort_field + " " + (sort_direction != "" ? sort_direction : "ASC") */name
  /*# end */
LIMIT /*= page_size != 0 ? page_size : 10 */10
OFFSET /*= page > 0 ? (page - 1) * page_size : 0 */0
```

中間形式：

```json
{
  "format_version": "1",
  "name": "getComplexData",
  "function_name": "getComplexData",
  "parameters": [
    {"name": "user_id", "type": "int"},
    {"name": "username", "type": "string"},
    {"name": "display_name", "type": "bool"},
    {"name": "start_date", "type": "string"},
    {"name": "end_date", "type": "string"},
    {"name": "sort_field", "type": "string"},
    {"name": "sort_direction", "type": "string"},
    {"name": "page_size", "type": "int"},
    {"name": "page", "type": "int"}
  ],
  "expressions": [
    "display_name ? username : \"Anonymous\"",
    "start_date != \"\" && end_date != \"\"",
    "start_date", "end_date",
    "sort_field + \" \" + (sort_direction != \"\" ? sort_direction : \"ASC\")",
    "page_size != 0 ? page_size : 10",
    "page > 0 ? (page - 1) * page_size : 0"
  ],
  "instructions": [
    {"op": "IF", "condition": "page > 0 ? (page - 1) * page_size : 0 != null", "pos": "0:0"},
    {"op": "IF", "condition": "page_size != 0 ? page_size : 10 != null", "pos": "0:0"},
    {"op": "IF", "condition": "sort_field != \"\"", "pos": "0:0"},
    {"op": "EMIT_STATIC", "value": "SELECT id, name,", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "display_name ? username : \"Anonymous\"", "pos": "4:3"},
    {"op": "EMIT_STATIC", "value": "FROM users WHERE created_at BETWEEN", "pos": "4:3"},
    {"op": "EMIT_EVAL", "param": "start_date", "pos": "8:22"},
    {"op": "EMIT_STATIC", "value": "'2023-01-01' AND", "pos": "8:39"},
    {"op": "EMIT_EVAL", "param": "end_date", "pos": "8:56"},
    {"op": "EMIT_STATIC", "value": "'2023-12-31' ORDER BY", "pos": "8:71"},
    {"op": "EMIT_EVAL", "param": "sort_field + \" \" + (sort_direction != \"\" ? sort_direction : \"ASC\")", "pos": "11:10"},
    {"op": "EMIT_STATIC", "value": "name", "pos": "11:83"},
    {"op": "EMIT_SYSTEM_LIMIT", "default_value": "10", "pos": "13:1"},
    {"op": "EMIT_STATIC", "value": "10", "pos": "13:45"},
    {"op": "EMIT_SYSTEM_OFFSET", "default_value": "0", "pos": "14:1"},
    {"op": "END", "pos": "0:0"},
    {"op": "END", "pos": "0:0"},
    {"op": "END", "pos": "0:0"}
  ]
}
```

## JSONスキーマ定義

中間形式には検証用のJSONスキーマ定義が含まれます：

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/shibukawa/snapsql/schemas/intermediate-format-v1.json",
  "title": "SnapSQL中間形式",
  "description": "SnapSQLテンプレートの中間JSON形式",
  "type": "object",
  "properties": {
    "format_version": {"type": "string", "enum": ["1"]},
    "name": {"type": "string"},
    "function_name": {"type": "string"},
    "parameters": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string"}
        },
        "required": ["name", "type"]
      }
    },
    "instructions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "op": {
            "type": "string",
            "enum": [
              "EMIT_STATIC", "EMIT_EVAL", "IF", "ELSE_IF", "ELSE", "END", 
              "LOOP_START", "LOOP_END", "EMIT_SYSTEM_LIMIT", "EMIT_SYSTEM_OFFSET", 
              "EMIT_SYSTEM_VALUE", "BOUNDARY", "EMIT_IF_DIALECT"
            ]
          },
          "value": {"type": "string"},
          "param": {"type": "string"},
          "condition": {"type": "string"},
          "variable": {"type": "string"},
          "collection": {"type": "string"},
          "default_value": {"type": "string"},
          "system_field": {"type": "string"},
          "dialect": {"type": "string"},
          "sql_fragment": {"type": "string"},
          "pos": {"type": "string"}
        },
        "required": ["op"]
      }
    },
    "expressions": {
      "type": "array",
      "items": {"type": "string"}
    },
    "implicit_parameters": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string"},
          "default": {"type": ["string", "number", "boolean", "null"]}
        },
        "required": ["name", "type"]
      }
    },
    "envs": {
      "type": "array",
      "items": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": {"type": "string"},
            "type": {"type": "string"}
          },
          "required": ["name", "type"]
        }
      }
    },
    "responses": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "type": {"type": "string"},
          "nullable": {"type": "boolean"}
        },
        "required": ["name", "type"]
      }
    },
    "response_affinity": {
      "type": "object",
      "properties": {
        "tables": {
          "type": "array",
          "items": {"type": "string"}
        },
        "columns": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "type": {"type": "string"},
              "table": {"type": "string"}
            },
            "required": ["name"]
          }
        }
      }
    }
  },
  "required": ["format_version", "instructions"]
}
```

## 位置情報（pos）

各命令には位置情報（`pos`）が含まれ、元のSQLテンプレート内での行と列を示します。形式は `"行:列"` です。

- `"1:1"`: 1行目1列目
- `"5:43"`: 5行目43列目
- `"0:0"`: システム生成命令（位置情報なし）

この情報は、デバッグやエラーレポートで使用されます。

## レスポンス親和性（Response Affinity）

### 概要

レスポンス親和性は、SQLクエリの結果がどのような構造を持つかを解析し、型推論によって決定される情報です。この情報は、コード生成時の結果型定義やORM連携に使用されます。

### 算出ロジック

#### 1. **親和性タイプの決定**

```go
type ResponseAffinity string

const (
    ResponseAffinityOne  ResponseAffinity = "one"  // 単一レコード
    ResponseAffinityMany ResponseAffinity = "many" // 複数レコード
    ResponseAffinityNone ResponseAffinity = "none" // レコードなし
)
```

#### 2. **判定アルゴリズム**

**単一レコード（`one`）の条件:**
- SELECT
  - PRIMARY KEYによる完全一致検索
  - UNIQUE制約フィールドによる完全一致検索
  - `LIMIT 1`が明示的に指定されている
  - COUNT(), SUM(), AVG()等の集約関数（単一の値を返すため）
  - JOINを含むが、LEFT INNER JOINもしくはINNER JOINで駆動表の要素が一位に定まり、結合するテーブルは配列として処理する指定がされている
- INSERT
  - RETURNING句がある単一要素のINSERT
- UPDATE
  - RETURNING句があり、PRIMARY KEYによる完全一致検索が行われているUPDATE
- DELETE
  - RETURNING句があり、PRIMARY KEYによる完全一致検索が行われているDELETE

**複数レコード（`many`）の条件:**
- 上記以外のSELECT文
- RETURNING句があり、複数要素のINSERT
- RETURNING句があるがWHERE句で主キーの完全一致をしていないUPDATE/DELETE句

**レスポンスなし（`none`）の条件:**
- INSERT文でRETURNING句がない
- UPDATE文でRETURNING句がない  
- DELETE文でRETURNING句がない

#### 3. **実装例**

```go
func detectResponseAffinity(stmt parser.StatementNode, tableInfo *TableInfo) ResponseAffinity {
    switch s := stmt.(type) {
    case *parser.SelectStatement:
        return detectSelectAffinity(s, tableInfo)
    case *parser.InsertStatement:
        if s.Returning != nil {
            return detectReturningAffinity(s.Returning, tableInfo)
        }
        return ResponseAffinityNone
    case *parser.UpdateStatement:
        if s.Returning != nil {
            return detectReturningAffinity(s.Returning, tableInfo)
        }
        return ResponseAffinityNone
    case *parser.DeleteStatement:
        if s.Returning != nil {
            return detectReturningAffinity(s.Returning, tableInfo)
        }
        return ResponseAffinityNone
    default:
        return ResponseAffinityNone
    }
}

func detectSelectAffinity(stmt *parser.SelectStatement, tableInfo *TableInfo) ResponseAffinity {
    // LIMIT 1が明示されている場合
    if hasExplicitLimitOne(stmt.Limit) {
        return ResponseAffinityOne
    }
    
    // PRIMARY KEYまたはUNIQUE制約による検索
    if hasUniqueKeyCondition(stmt, tableInfo) {
        return ResponseAffinityOne
    }
    
    // GROUP BYがある場合は複数行の可能性
    if stmt.GroupBy != nil {
        return ResponseAffinityMany
    }
    
    // デフォルトは複数レコード
    return ResponseAffinityMany
}

func detectReturningAffinity(returning *parser.ReturningClause, tableInfo *TableInfo) ResponseAffinity {
    // INSERT文のRETURNINGは挿入レコード数次第
    if stmt, ok := returning.Parent.(*parser.InsertStatement); ok {
        if isMultiRowInsert(stmt) {
            return ResponseAffinityMany // 複数行INSERT
        }
        return ResponseAffinityOne // 単一行INSERT
    }
    
    // UPDATE/DELETE文のRETURNINGは条件次第
    if stmt, ok := returning.Parent.(*parser.UpdateStatement); ok {
        if hasUniqueKeyCondition(stmt, tableInfo) {
            return ResponseAffinityOne
        }
        return ResponseAffinityMany
    }
    
    if stmt, ok := returning.Parent.(*parser.DeleteStatement); ok {
        if hasUniqueKeyCondition(stmt, tableInfo) {
            return ResponseAffinityOne
        }
        return ResponseAffinityMany
    }
    
    return ResponseAffinityMany
}

func isMultiRowInsert(stmt *parser.InsertStatement) bool {
    // VALUES句で複数行が指定されている場合
    if stmt.ValuesList != nil && len(stmt.ValuesList.Values) > 1 {
        return true
    }
    
    // SELECT文からのINSERT（INSERT INTO ... SELECT）の場合
    if stmt.SelectStatement != nil {
        return true // SELECTの結果は複数行の可能性
    }
    
    return false
}
```

#### 4. **テーブル情報の活用**

```go
type TableInfo struct {
    Name        string
    PrimaryKeys []string
    UniqueKeys  [][]string // 複合UNIQUE制約に対応
    Columns     []ColumnInfo
}

type ColumnInfo struct {
    Name     string
    Type     string
    Nullable bool
}
```

#### 5. **WHERE句の解析**

```go
func hasUniqueKeyCondition(stmt *parser.SelectStatement, tableInfo *TableInfo) bool {
    whereConditions := extractWhereConditions(stmt.Where)
    
    // PRIMARY KEY完全一致チェック
    if matchesAllPrimaryKeys(whereConditions, tableInfo.PrimaryKeys) {
        return true
    }
    
    // UNIQUE制約完全一致チェック
    for _, uniqueKey := range tableInfo.UniqueKeys {
        if matchesAllUniqueKeys(whereConditions, uniqueKey) {
            return true
        }
    }
    
    return false
}
```

### 中間形式での表現

```json
{
  "response_affinity": {
    "type": "one",
    "tables": ["users"],
    "columns": [
      {"name": "id", "type": "int", "table": "users"},
      {"name": "name", "type": "string", "table": "users"},
      {"name": "email", "type": "string", "table": "users"}
    ],
    "reasoning": "PRIMARY KEY condition detected: users.id = ?"
  }
}
```

### 算出例

#### 例1: 単一レコード（PRIMARY KEY検索）
```sql
SELECT id, name, email FROM users WHERE id = /*= user_id */123
```

**算出結果:**
- **Type**: `one`
- **理由**: PRIMARY KEY (`id`) による完全一致検索
- **テーブル**: `users`
- **カラム**: `id`, `name`, `email`

#### 例2: 単一レコード（集約関数）
```sql
SELECT COUNT(*) as total FROM users WHERE active = true
```

**算出結果:**
- **Type**: `one`
- **理由**: 集約関数は単一の値を返す
- **テーブル**: `users`
- **カラム**: `total` (計算フィールド)

#### 例3: 複数レコード
```sql
SELECT id, name FROM users WHERE department = /*= dept */'Engineering'
```

**算出結果:**
- **Type**: `many`
- **理由**: 非UNIQUE条件による検索
- **テーブル**: `users`
- **カラム**: `id`, `name`

#### 例4: レスポンスなし（INSERT文）
```sql
INSERT INTO users (name, email) VALUES (/*= name */'John', /*= email */'john@example.com')
```

**算出結果:**
- **Type**: `none`
- **理由**: RETURNING句がないINSERT文
- **テーブル**: `users`
- **カラム**: なし

#### 例5: 単一レコード（単一行INSERT with RETURNING）
```sql
INSERT INTO users (name, email) VALUES (/*= name */'John', /*= email */'john@example.com') RETURNING id, created_at
```

**算出結果:**
- **Type**: `one`
- **理由**: 単一行INSERT文
- **テーブル**: `users`
- **カラム**: `id`, `created_at`

#### 例6: 複数レコード（複数行INSERT with RETURNING）
```sql
INSERT INTO users (name, email) VALUES 
    (/*= user1.name */'John', /*= user1.email */'john@example.com'),
    (/*= user2.name */'Jane', /*= user2.email */'jane@example.com')
RETURNING id, created_at
```

**算出結果:**
- **Type**: `many`
- **理由**: 複数行INSERT文
- **テーブル**: `users`
- **カラム**: `id`, `created_at`

#### 例8: 複数レコード（SELECT文からのINSERT with RETURNING）
```sql
INSERT INTO users_backup (name, email) 
SELECT name, email FROM users WHERE active = false 
RETURNING id, created_at
```

**算出結果:**
- **Type**: `many`
- **理由**: SELECT文からのINSERT（複数行の可能性）
- **テーブル**: `users_backup`
- **カラム**: `id`, `created_at`
#### 例9: 複数レコード（UPDATE with RETURNING）
```sql
UPDATE users SET active = false WHERE department = /*= dept */'Engineering' RETURNING id, name
```

**算出結果:**
- **Type**: `many`
- **理由**: 複数行が更新される可能性
- **テーブル**: `users`
- **カラム**: `id`, `name`

**算出結果:**
- **Type**: `many`
- **理由**: 複数行が更新される可能性
- **テーブル**: `users`
- **カラム**: `id`, `name`

### 利用用途

#### 1. **コード生成での活用**
```go
// Go言語での生成例
func GetUser(ctx context.Context, userID int) (*User, error)             // one (SELECT)
func CountUsers(ctx context.Context) (int, error)                       // one (COUNT)
func GetUsers(ctx context.Context, dept string) ([]*User, error)        // many (SELECT)
func CreateUser(ctx context.Context, user *User) error                  // none (単一INSERT)
func CreateUserWithID(ctx context.Context, user *User) (*User, error)   // one (単一INSERT RETURNING)
func CreateUsers(ctx context.Context, users []*User) ([]*User, error)   // many (複数INSERT RETURNING)
func UpdateUsers(ctx context.Context, dept string) ([]*User, error)     // many (UPDATE RETURNING)
func DeleteUser(ctx context.Context, userID int) error                  // none (DELETE)
```

#### 2. **型安全性の向上**
- **`one`**: 単一オブジェクトまたはnullを返す
- **`many`**: 配列を返す（空配列の可能性あり）
- **`none`**: エラーのみを返す（レスポンスデータなし）

#### 3. **ORM連携**
- **`one`**: `FindOne()`, `First()`, `Count()`メソッドの生成
- **`many`**: `FindAll()`, `Where()`メソッドの生成
- **`none`**: `Create()`, `Update()`, `Delete()`メソッドの生成（戻り値なし）

## システムフィールド機能

### 概要

システムフィールド機能は、データベースの監査ログやバージョン管理に必要な共通フィールド（`created_at`, `updated_at`, `created_by`, `updated_by`, `lock_no`など）を自動的に管理する機能です。

### 設定

システムフィールドは設定ファイル（`snapsql.yaml`）で定義されます：

```yaml
system:
  fields:
    - name: created_at
      type: timestamp
      on_insert:
        default: "NOW()"
      on_update:
        parameter: error
    - name: updated_at
      type: timestamp
      on_insert:
        default: "NOW()"
      on_update:
        default: "NOW()"
    - name: created_by
      type: string
      on_insert:
        parameter: implicit
      on_update:
        parameter: error
    - name: updated_by
      type: string
      on_insert:
        parameter: implicit
      on_update:
        parameter: implicit
    - name: lock_no
      type: int
      on_insert:
        default: 1
      on_update:
        parameter: explicit
```

### パラメータ設定

各システムフィールドには、INSERT文とUPDATE文での動作を個別に設定できます：

- **`explicit`**: 明示的にパラメータを提供する必要がある
- **`implicit`**: ランタイムが自動的に値を提供する（例：ユーザーID、セッション情報）
- **`error`**: パラメータが提供された場合はエラーとする
- **`default`**: デフォルト値を使用する

### 中間形式への影響

#### 暗黙的パラメータ

システムフィールドの設定に基づいて、中間形式に暗黙的パラメータが追加されます：

```json
{
  "format_version": "1",
  "name": "update_user",
  "function_name": "update_user",
  "parameters": [
    {"name": "name", "type": "string"},
    {"name": "email", "type": "string"},
    {"name": "lock_no", "type": "int"}
  ],
  "implicit_parameters": [
    {"name": "updated_at", "type": "timestamp", "default": "NOW()"},
    {"name": "updated_by", "type": "string"}
  ],
  "instructions": [
    {"op": "EMIT_STATIC", "value": "UPDATE users SET name = ", "pos": "1:1"},
    {"op": "EMIT_EVAL", "param": "name", "pos": "1:25"},
    {"op": "EMIT_STATIC", "value": ", email = ", "pos": "1:30"},
    {"op": "EMIT_EVAL", "param": "email", "pos": "1:40"},
    {"op": "EMIT_STATIC", "value": ", EMIT_SYSTEM_VALUE(updated_at), EMIT_SYSTEM_VALUE(updated_by) WHERE id = ", "pos": "1:46"},
    {"op": "EMIT_EVAL", "param": "user_id", "pos": "1:100"}
  ]
}
```

#### UPDATE文の自動修正

UPDATE文の場合、暗黙的パラメータに対応するシステムフィールドが自動的にSET句に追加されます：

**元のSQL:**
```sql
UPDATE users SET name = 'John', email = 'john@example.com' WHERE id = 1
```

**システムフィールド追加後:**
```sql
UPDATE users SET 
  name = 'John', 
  email = 'john@example.com',
  EMIT_SYSTEM_VALUE(updated_at),
  EMIT_SYSTEM_VALUE(updated_by)
WHERE id = 1
```

### バリデーション

システムフィールドの設定に基づいて、以下のバリデーションが実行されます：

1. **明示的パラメータの存在確認**: `explicit`設定のフィールドに対応するパラメータが提供されているかチェック
2. **エラーパラメータの検出**: `error`設定のフィールドに対応するパラメータが提供されていないかチェック
3. **型整合性の確認**: パラメータの型がシステムフィールドの定義と一致するかチェック

### 実行時の動作

#### 暗黙的パラメータの解決

ランタイムライブラリは、暗黙的パラメータを以下のソースから自動的に解決します：

- **ユーザーコンテキスト**: 認証されたユーザーID（`created_by`, `updated_by`）
- **システム時刻**: 現在時刻（`created_at`, `updated_at`のデフォルト値）
- **セッション情報**: リクエストメタデータ
- **設定値**: 設定ファイルで定義されたデフォルト値

#### 楽観的ロック

`lock_no`フィールドを使用した楽観的ロックをサポート：

1. **SELECT時**: 現在の`lock_no`を取得
2. **UPDATE時**: `lock_no`を明示的パラメータとして提供
3. **実行時**: `lock_no`が一致しない場合は楽観的ロック例外を発生

### 利点

1. **一貫性**: すべてのテーブルで統一されたシステムフィールドの管理
2. **自動化**: 手動でのシステムフィールド設定が不要
3. **セキュリティ**: 改ざん防止（`created_at`, `created_by`の更新禁止）
4. **監査**: 完全な変更履歴の自動記録
5. **並行制御**: 楽観的ロックによるデータ整合性の保証
