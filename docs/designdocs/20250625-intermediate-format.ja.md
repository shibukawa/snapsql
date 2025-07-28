# SnapSQL 中間形式仕様

**ドキュメントバージョン:** 2.0  
**日付:** 2025-07-25  
**ステータス:** 実装済み

## 概要

このドキュメントは、SnapSQLテンプレートの中間JSON形式を定義します。中間形式は、SQLテンプレートパーサーとコードジェネレータ間の橋渡しとして機能し、解析されたSQLテンプレートのメタデータ、CEL式、関数定義の言語非依存表現を提供します。

## 設計目標

### 1. 言語非依存
- あらゆるプログラミング言語で利用可能なJSON形式
- 言語固有の構造や前提条件なし
- SQL構造と言語固有メタデータの明確な分離

## 設計目標

### 1. 言語非依存
- あらゆるプログラミング言語で利用可能なJSON形式
- 言語固有の構造や前提条件なし
- SQL構造と言語固有メタデータの明確な分離

### 2. 完全な情報保持
- テンプレートメタデータと関数定義
- CEL式の完全な抽出と型情報

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
  "name": "get_user_by_id",
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

```markdown
# Markdownファイルのパラメータセクション
## Parameters

```yaml
user_id: int
include_details: bool
```
```

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

### 命令の例

```json
{"op": "EMIT_STATIC", "value": "SELECT id, name FROM users WHERE ", "pos": "1:1"}
{"op": "EMIT_EVAL", "param": "user_id", "pos": "1:43"}
{"op": "IF", "condition": "min_age > 0", "pos": "2:1"}
{"op": "EMIT_STATIC", "value": " AND age >= ", "pos": "3:1"}
{"op": "EMIT_EVAL", "param": "min_age", "pos": "3:12"}
{"op": "ELSE_IF", "condition": "max_age > 0", "pos": "4:1"}
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
  "name": "get_user_by_id",
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
  "name": "get_filtered_users",
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
            "enum": ["EMIT_STATIC", "EMIT_EVAL", "IF", "ELSE_IF", "ELSE", "END", "LOOP_START", "LOOP_END", "EMIT_SYSTEM_LIMIT", "EMIT_SYSTEM_OFFSET", "EMIT_SYSTEM_VALUE"]
          },
          "value": {"type": "string"},
          "param": {"type": "string"},
          "condition": {"type": "string"},
          "variable": {"type": "string"},
          "collection": {"type": "string"},
          "default_value": {"type": "string"},
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

## 今後の拡張

現在の中間形式は、基本的なCEL式の抽出と命令セットの実装に焦点を当てています。将来的には、以下の機能が追加される予定です：

1. **命令セットの最適化**: 命令列の最適化による実行効率の向上
2. **型推論の強化**: より正確な型情報の提供
3. **レスポンス型定義**: クエリ結果の型情報
4. **テーブルスキーマ統合**: データベーススキーマ情報との統合

これらの拡張により、より強力なコード生成と実行時の最適化が可能になります。

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
