# 中間命令セット設計ドキュメント

**日付**: 2025-06-26  
**作成者**: 開発チーム  
**ステータス**: ドラフト  

## 概要

このドキュメントは、中間ファイルの階層ASTを置き換える手続型命令セットの設計について説明します。システムは**2層アーキテクチャ**を使用します：

1. **低レベル**: 直接実行のための命令セットインタープリター
2. **高レベル**: 命令を制御構造と関数呼び出しにマッピングした生成プログラミング言語コード

### 多言語ランタイムアーキテクチャ

SnapSQLは言語特性と使用ケースに基づいて、異なる実装戦略で複数のプログラミング言語をサポートします：

#### Goランタイム: デュアル層アーキテクチャ
**低レベルインタープリター + 高レベルコード生成**

Goは異なるシナリオをサポートするため、両方の実行モードを提供します：

1. **低レベルインタープリター** (`runtime/snapsqlgo`)
   - 直接命令実行
   - 即座のSQL生成
   - **主要用途**: REPLとインタラクティブ開発
   - **利点**: コンパイル手順なし、即座のフィードバック
   - **パフォーマンス**: 開発とプロトタイピングに適している

2. **高レベルコード生成** (予定)
   - コンパイル時コード生成
   - 型安全な関数生成
   - **主要用途**: 本番アプリケーション
   - **利点**: 最大パフォーマンス、型安全性、IDE支援
   - **パフォーマンス**: 本番ワークロード向けに最適化

#### 他言語: 高レベルのみ
**Python、Node.js、Java、C#など**

他の言語は高レベルコード生成のみを実装します：

1. **コード生成のみ**
   - ネイティブ言語構造にコンパイルされた命令
   - 適用可能な場合は型安全API
   - **理由**: ほとんどの言語はSQL用のREPLサポートを必要としない
   - **利点**: よりシンプルな実装、より良いパフォーマンス
   - **焦点**: 本番対応、最適化されたコード

### アーキテクチャ比較

| 言語 | 低レベル | 高レベル | 主要用途 | REPLサポート |
|------|----------|----------|----------|--------------|
| **Go** | ✅ | ✅ | 開発 + 本番 | ✅ |
| **Python** | ❌ | ✅ | 本番 | ❌ |
| **Node.js** | ❌ | ✅ | 本番 | ❌ |
| **Java** | ❌ | ✅ | 本番 | ❌ |
| **C#** | ❌ | ✅ | 本番 | ❌ |

### Go REPL要件

GoランタイムはREPL機能をサポートするために特別に低レベルインタープリターを含みます：

#### REPL使用ケース
```go
// インタラクティブSQL開発
> snapsql repl
SnapSQL> load template users.snap.sql
SnapSQL> set param user_id 123
SnapSQL> set param include_email true
SnapSQL> execute
SQL: SELECT id, name, email FROM users WHERE id = ?
Args: [123]

SnapSQL> set param include_email false
SnapSQL> execute
SQL: SELECT id, name FROM users WHERE id = ?
Args: [123]
```

#### 開発ワークフロー
1. **テンプレート開発**: SQL生成への即座のフィードバック
2. **パラメータテスト**: 素早いパラメータ値変更
3. **デバッグ**: ステップバイステップ命令実行
4. **プロトタイピング**: コンパイルなしでの高速反復

#### 実装の利点
- **即座のフィードバック**: コンパイル遅延なし
- **インタラクティブ開発**: リアルタイムSQLプレビュー
- **デバッグサポート**: 命令レベルデバッグ
- **教育ツール**: SQL生成プロセスの理解

## 目標

### 主要目標
- 階層ASTを線形命令シーケンスに変換
- 制御構造（if/for）を条件ジャンプとgotoに変換
- ランタイムでシンプルなインタープリターベースのSQL生成を実現
- SQL書式設定（カンマ、括弧など）の細かい制御を含める
- ランタイム複雑性を最小化し、パフォーマンスを最大化

### 副次目標
- 命令シーケンスの可読性とデバッグ可能性を維持
- 既存のSnapSQLテンプレート機能をすべてサポート
- 命令シーケンスの最適化を容易にする
- 元のテンプレートから命令への明確なマッピングを提供

## 命令セットアーキテクチャ

### コア概念

#### 1. 線形実行モデル
- 命令はインデックス0から順次実行
- 制御フローはジャンプ命令で管理
- ネストした構造や再帰評価なし
- シンプルなプログラムカウンタ（PC）ベースの実行

#### 2. ループ変数スコープ付き直接パラメータアクセス
- 入力マップから名前で直接パラメータにアクセス
- ループ変数はループスコープ内で作成され、ループ終了時に削除
- 条件のシンプルなブール値評価
- SQL出力での直接パラメータ置換
- 階層変数解決（ループ変数がパラメータをシャドウ）

#### 3. SQL出力バッファ
- 命令はSQL出力バッファに直接書き込み
- 自動書式制御（スペース、カンマ、括弧）
- ランタイムパラメータ値に基づく条件付き出力

### 命令タイプ

#### 1. 出力命令
SQLテキストを生成し、書式を管理します。

```json
{
  "op": "EMIT_LITERAL",
  "value": "SELECT id, name"
}
```

```json
{
  "op": "EMIT_PARAM",
  "param": "user_id",
  "placeholder": "123"
}
```

```json
{
  "op": "EMIT_EVAL",
  "exp": "user.age + 1",
  "placeholder": "25"
}
```

```json
{
  "op": "EMIT_LITERAL", "value": ",",
  "condition": "not_last_field"
}
```

#### 2. 制御フロー命令
プログラム実行フローを管理します。

```json
{
  "op": "JUMP",
  "target": 15
}
```

```json
{
  "op": "JUMP_IF_TRUE",
  "condition": "include_email",
  "target": 10
}
```

#### 3. 条件評価命令
条件ジャンプのためのCEL式評価を処理します。

```json
{
  "op": "JUMP_IF_EXP",
  "exp": "!include_email",
  "target": 10
}
```

```json
{
  "op": "JUMP_IF_EXP_TRUE",
  "exp": "!include_email",
  "target": 10
}
```

```json
{
  "op": "JUMP_IF_EXP_FALSE",
  "exp": "!filters.active",
  "target": 20
}
```

```json
{
  "op": "JUMP_IF_EXP_EQ",
  "exp": "status",
  "value": "active",
  "target": 15
}
```

```json
{
  "op": "JUMP_IF_EXP_NE",
  "exp": "role",
  "value": "admin",
  "target": 25
}
```

#### 4. ループ命令
コレクションの反復処理を行います。

```json
{
  "op": "LOOP_START",
  "variable": "field",
  "collection": "additional_fields",
  "end_label": "end_field_loop"
}
```

#### 5. 状態管理命令
**削除**: フラグによる状態管理は削除されました。すべての条件ロジックは`JUMP_IF_EXP`命令のCEL式で処理されます。

```json
{
  "op": "JUMP_IF_FLAG_TRUE",
  "flag": "has_where_clause",
  "target": 30
}
```

```json
{
  "op": "JUMP_IF_FLAG_FALSE",
  "flag": "has_where_clause",
  "target": 35
}
```

## 命令セットリファレンス

### 出力命令

| 命令 | 説明 | パラメータ |
|------|------|-----------|
| `EMIT_LITERAL` | リテラルSQLテキストを出力 | `value`: 文字列 |
| `EMIT_PARAM` | 単独変数プレースホルダーを出力 | `param`: 変数名, `placeholder`: ダミー値 |
| `EMIT_EVAL` | CEL式結果プレースホルダーを出力 | `exp`: CEL式, `placeholder`: ダミー値 |
| `EMIT_LPAREN` | 左括弧を出力 | なし |
| `EMIT_RPAREN` | 右括弧を出力 | なし |

### 制御フロー命令

| 命令 | 説明 | パラメータ |
|------|------|-----------|
| `JUMP` | 無条件ジャンプ | `target`: 命令インデックス |
| `JUMP_IF_EXP_TRUE` | パラメータが真の場合ジャンプ | `param`: CEL式, `target`: 命令インデックス |
| `JUMP_IF_EXP_FALSE` | パラメータが偽の場合ジャンプ | `param`: CEL式, `target`: 命令インデックス |
| `JUMP_IF_EXP_EQ` | パラメータが値と等しい場合ジャンプ | `param`: CEL式, `value`: 比較値, `target`: 命令インデックス |
| `JUMP_IF_EXP_NE` | パラメータが値と等しくない場合ジャンプ | `param`: CEL式, `value`: 比較値, `target`: 命令インデックス |
| `JUMP_IF_FLAG_TRUE` | フラグが真の場合ジャンプ | `flag`: フラグ名, `target`: 命令インデックス |
| `JUMP_IF_FLAG_FALSE` | フラグが偽の場合ジャンプ | `flag`: フラグ名, `target`: 命令インデックス |
| `LABEL` | ジャンプターゲットを定義 | `name`: ラベル名 |
| `NOP` | 何もしない | なし |

### CEL式命令

| 命令 | 説明 | パラメータ |
|------|------|-----------|
| `JUMP_IF_EXP` | CEL式が真の場合ジャンプ | `exp`: CEL式, `target`: 命令インデックス |

### ループ命令

| 命令 | 説明 | パラメータ |
|------|------|-----------|
| `LOOP_START` | コレクションのループを初期化 | `variable`: ループ変数, `collection`: コレクションパラメータ, `end_label`: 終了ラベル |
| `LOOP_NEXT` | 次の反復に続行 | `start_label`: ループ開始ラベル |
| `LOOP_END` | ループ終了と変数クリーンアップ | `variable`: 削除するループ変数, `label`: ループ終了ラベル |

### 状態管理命令

| 命令 | 説明 | パラメータ |
|------|------|-----------|
| `SET_FLAG` | ブール値フラグを設定 | `flag`: フラグ名, `value`: ブール値 |
| `JUMP_IF_FLAG_TRUE` | フラグが真の場合ジャンプ | `flag`: フラグ名, `target`: 命令インデックス |
| `JUMP_IF_FLAG_FALSE` | フラグが偽の場合ジャンプ | `flag`: フラグ名, `target`: 命令インデックス |

### EMIT命令の使用例

#### シンプルな変数出力
```sql
-- テンプレート: /*= user_id */123
{"op": "EMIT_PARAM", "param": "user_id", "placeholder": "123"}

-- テンプレート: /*= field */field_name  
{"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"}
```

#### 複雑な式の出力
```sql
-- テンプレート: /*= user.age + 1 */25
{"op": "EMIT_EVAL", "exp": "user.age + 1", "placeholder": "25"}

-- テンプレート: /*= table.name */table_name
{"op": "EMIT_EVAL", "exp": "table.name", "placeholder": "table_name"}

-- テンプレート: /*= len(items) */5
{"op": "EMIT_EVAL", "exp": "len(items)", "placeholder": "5"}
```

#### リテラル出力（スペース、カンマ、改行）
```sql
-- すべてのフォーマットはEMIT_LITERALで処理
{"op": "EMIT_LITERAL", "value": ", "}      // カンマとスペース
{"op": "EMIT_LITERAL", "value": "\n"}      // 改行
{"op": "EMIT_LITERAL", "value": "  "}      // インデント
{"op": "EMIT_LITERAL", "value": " AND "}   // SQLキーワード
```

#### パフォーマンス考慮事項
- `EMIT_LITERAL`: 最高速、直接文字列出力
- `EMIT_PARAM`: 高速な直接変数参照、CELエンジンのオーバーヘッドなし
- `EMIT_EVAL`: 完全なCEL式評価、より柔軟だが低速

## 変換例

### シンプルな条件付きフィールド

**元のテンプレート:**
```sql
SELECT id, name
/*# if include_email */
, email
/*# end */
FROM users
```

**生成された命令:**
```json
[
  {"op": "EMIT_LITERAL", "value": "SELECT id, name"},
  {"op": "JUMP_IF_EXP_FALSE", "exp": "!include_email", "target": 5},
  {"op": "EMIT_LITERAL", "value": ", email"},
  {"op": "LABEL", "name": "end_email_field"},
  {"op": "EMIT_LITERAL", "value": " FROM users"}
]
```

### カンマ制御付きループ

**元のテンプレート:**
```sql
SELECT 
/*# for field : additional_fields */
    /*= field */,
/*# end */
    created_at
FROM users
```

**生成された命令:**
```json
[
  {"op": "EMIT_LITERAL", "value": "SELECT"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_START", "variable": "field", "collection": "additional_fields", "end_label": "end_field_loop"},
  {"op": "LABEL", "name": "field_loop_start"},
  {"op": "EMIT_LITERAL", "value": "    "},
  {"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"},
  {"op": "EMIT_LITERAL", "value": ","},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_NEXT", "start_label": "field_loop_start"},
  {"op": "LOOP_END", "variable": "field", "label": "end_field_loop"},
  {"op": "EMIT_LITERAL", "value": "    created_at"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "EMIT_LITERAL", "value": "FROM users"}
]
```

## ランタイム実行モデル

### 実行エンジン

```go
type InstructionExecutor struct {
    instructions []Instruction
    pc          int
    params      map[string]any
    output      strings.Builder
    loops       []LoopState
    variables   map[string]any  // スコープ付きループ変数
}

type LoopState struct {
    Variable   string
    Collection []any
    Index      int
    StartPC    int
}

func (e *InstructionExecutor) Execute() (string, []any, error) {
    e.variables = make(map[string]any)
    for e.pc < len(e.instructions) {
        inst := e.instructions[e.pc]
        if err := e.executeInstruction(inst); err != nil {
            return "", nil, err
        }
        e.pc++
    }
    return e.output.String(), e.extractParameters(), nil
}
```

### 変数解決とループスコープ

```go
func (e *InstructionExecutor) getVariableValue(name string) any {
    // ループ変数を最初にチェック（パラメータをシャドウ）
    if value, exists := e.variables[name]; exists {
        return value
    }
    // 入力パラメータにフォールバック
    return e.getParamValue(name)
}

func (e *InstructionExecutor) executeInstruction(inst Instruction) error {
    switch inst.Op {
    case "LOOP_START":
        collection := e.getParamValue(inst.Collection)
        if collectionSlice, ok := collection.([]any); ok && len(collectionSlice) > 0 {
            // ループ変数を最初の要素に設定
            e.variables[inst.Variable] = collectionSlice[0]
            // ループ状態をプッシュ
            e.loops = append(e.loops, LoopState{
                Variable:   inst.Variable,
                Collection: collectionSlice,
                Index:      0,
                StartPC:    e.pc + 1,
            })
        } else {
            // 空のコレクション、終了にジャンプ
            e.pc = e.findLabel(inst.EndLabel) - 1
        }
    case "LOOP_NEXT":
        if len(e.loops) > 0 {
            loop := &e.loops[len(e.loops)-1]
            loop.Index++
            if loop.Index < len(loop.Collection) {
                // ループ変数を更新してジャンプバック
                e.variables[loop.Variable] = loop.Collection[loop.Index]
                e.pc = loop.StartPC - 1
            }
            // そうでなければLOOP_ENDに続行
        }
    case "LOOP_END":
        if len(e.loops) > 0 {
            // スコープからループ変数を削除
            delete(e.variables, inst.Variable)
            // ループ状態をポップ
            e.loops = e.loops[:len(e.loops)-1]
        }
    // ... その他の命令
    }
    return nil
}
```

### 命令実行

```go
func (e *InstructionExecutor) executeInstruction(inst Instruction) error {
    switch inst.Op {
    case "EMIT_LITERAL":
        e.output.WriteString(inst.Value)
    case "EMIT_PARAM":
        e.output.WriteString("?")
        e.addParameter(inst.Param)
    case "EMIT_EVAL":
        e.output.WriteString("?")
        e.addExpression(inst.Exp)
    case "JUMP":
        e.pc = inst.Target - 1 // pcがインクリメントされるため-1
    case "JUMP_IF_EXP_TRUE":
        if e.isParamTruthy(inst.Exp) {
            e.pc = inst.Target - 1
        }
    case "JUMP_IF_EXP_FALSE":
        if !e.isParamTruthy(inst.Exp) {
            e.pc = inst.Target - 1
        }
    case "JUMP_IF_EXP_EQ":
        if e.getParamValue(inst.Exp) == inst.Value {
            e.pc = inst.Target - 1
        }
    case "SET_FLAG":
        e.flags[inst.Flag] = inst.Value.(bool)
    case "JUMP_IF_FLAG_TRUE":
        if e.flags[inst.Flag] {
            e.pc = inst.Target - 1
        }
    // ... その他の命令
    }
    return nil
}

func (e *InstructionExecutor) isParamTruthy(paramPath string) bool {
    value := e.getParamValue(paramPath)
    // Go の真偽値処理: false, 0, "", nil, 空のスライス/マップは偽値
    switch v := value.(type) {
    case bool:
        return v
    case int, int8, int16, int32, int64:
        return v != 0
    case uint, uint8, uint16, uint32, uint64:
        return v != 0
    case float32, float64:
        return v != 0
    case string:
        return v != ""
    case []any:
        return len(v) > 0
    case map[string]any:
        return len(v) > 0
    case nil:
        return false
    default:
        return true
    }
}
```

### 変数スコープ付きネストループ

**元のテンプレート:**
```sql
SELECT 
/*# for table : tables */
  /*# for field : table.fields */
    /*= table.name */./*= field */ AS /*= table.name */_/*= field */,
  /*# end */
/*# end */
  1 as dummy
FROM dual
```

**生成された命令:**
```json
[
  {"op": "EMIT_LITERAL", "value": "SELECT"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_START", "variable": "table", "collection": "tables", "end_label": "end_table_loop"},
  {"op": "LABEL", "name": "table_loop_start"},
  {"op": "LOOP_START", "variable": "field", "collection": "table.fields", "end_label": "end_field_loop"},
  {"op": "LABEL", "name": "field_loop_start"},
  {"op": "EMIT_LITERAL", "value": "  "},
  {"op": "EMIT_EVAL", "exp": "table.name", "placeholder": "table_name"},
  {"op": "EMIT_LITERAL", "value": "."},
  {"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"},
  {"op": "EMIT_LITERAL", "value": " AS "},
  {"op": "EMIT_EVAL", "exp": "table.name", "placeholder": "table_name"},
  {"op": "EMIT_LITERAL", "value": "_"},
  {"op": "EMIT_PARAM", "param": "field", "placeholder": "field_name"},
  {"op": "EMIT_LITERAL", "value": ","},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "LOOP_NEXT", "start_label": "field_loop_start"},
  {"op": "LOOP_END", "variable": "field", "label": "end_field_loop"},
  {"op": "LOOP_NEXT", "start_label": "table_loop_start"},
  {"op": "LOOP_END", "variable": "table", "label": "end_table_loop"},
  {"op": "EMIT_LITERAL", "value": "  1 as dummy"},
  {"op": "EMIT_LITERAL", "value": "\n"},
  {"op": "EMIT_LITERAL", "value": "FROM dual"}
]
```

**実行中の変数解決:**
1. **ループ外**: 入力パラメータのみが見える
2. **tableループ内**: `table`変数が`table`という名前の入力パラメータをシャドウ
3. **ネストしたfieldループ内**: `table`と`field`の両方の変数が見える
4. **fieldループ終了後**: `field`変数が削除され、`table`のみが見える
5. **tableループ終了後**: `table`変数が削除され、入力パラメータのみに戻る

## 中間ファイルフォーマット

### 更新されたスキーマ

```json
{
  "source": {
    "file": "/path/to/users.snap.sql",
    "content": "SELECT id, name /*# if include_email */, email /*# end */ FROM users"
  },
  "interface_schema": {
    "name": "user_query",
    "function_name": "getUsers",
    "parameters": [
      {
        "name": "include_email",
        "type": "bool",
        "required": false,
        "default": false
      }
    ]
  },
  "instructions": [
    {"op": "EMIT_LITERAL", "value": "SELECT id, name"},
    {"op": "JUMP_IF_EXP_FALSE", "exp": "!include_email", "target": 5},
    {"op": "EMIT_LITERAL", "value": ", email"},
    {"op": "LABEL", "name": "end_email_field"},
    {"op": "EMIT_LITERAL", "value": " FROM users"}
  ],
  "labels": {
    "end_email_field": 5
  },
  "metadata": {
    "version": "1.0.0",
    "generated_at": "2025-06-26T01:00:00Z",
    "template_hash": "sha256:abc123..."
  }
}
```

## 命令セットアプローチの利点

### ランタイムパフォーマンス
- プログラムカウンタによるシンプルな線形実行
- スタック操作や複雑な値管理なし
- 直接パラメータアクセスと評価
- 最小限の分岐と意思決定
- 直接的なSQL出力生成

### シンプルさ
- 明確でデバッグ可能な命令シーケンス
- 実装が容易なインタープリター
- スタック管理の複雑性なし
- 直接的なパラメータ評価
- ランタイム複雑性の削減

### 柔軟性
- 新しい命令タイプの追加が容易
- 複雑な制御フローのサポート
- 細かい書式制御
- 将来の機能に対する拡張性

### デバッグと解析
- 明確な実行トレース
- プログラムフローの視覚化が容易
- シンプルなブレークポイントとステップ実行サポート
- パフォーマンスプロファイリング機能

## 多言語高レベルインタフェース設計

### Go高レベルインタフェース（予定）

#### 生成されるGoコードの例
```go
// SQLテンプレートメタデータから生成
type SearchUserResult struct {
    ID          string `db:"id"`
    Name        string `db:"name"`
    Org         string `db:"org"`
    PhoneNumber string `db:"phone_number"`
}

func SearchUsers[D DB](ctx context.Context, db D, id string, name string) iter.Seq2[SearchUserResult, error] {
    // 命令セットから最適化されたGoコードにコンパイル
    var sql strings.Builder
    var args []any
    
    sql.WriteString("SELECT id, name, org, phone_number FROM users WHERE id = ")
    args = append(args, strings.ToUpper(id)) // CEL: id.upper()
    sql.WriteString(" AND name = ")
    args = append(args, name)
    
    return executeQuery[SearchUserResult](ctx, db, sql.String(), args)
}
```

### Python高レベルインタフェース（予定）

#### 生成されるPythonコードの例
```python
from typing import Iterator, NamedTuple
from dataclasses import dataclass

@dataclass
class SearchUserResult:
    id: str
    name: str
    org: str
    phone_number: str

def search_users(db: Connection, id: str, name: str) -> Iterator[SearchUserResult]:
    """searchuser.snap.sqlから生成"""
    sql = "SELECT id, name, org, phone_number FROM users WHERE id = %s AND name = %s"
    args = [id.upper(), name]  # CEL式がPythonにコンパイル
    
    cursor = db.execute(sql, args)
    for row in cursor:
        yield SearchUserResult(*row)
```

### Node.js高レベルインタフェース（予定）

#### 生成されるTypeScriptコードの例
```typescript
interface SearchUserResult {
    id: string;
    name: string;
    org: string;
    phoneNumber: string;
}

export async function* searchUsers(
    db: Database, 
    id: string, 
    name: string
): AsyncIterableIterator<SearchUserResult> {
    const sql = "SELECT id, name, org, phone_number FROM users WHERE id = ? AND name = ?";
    const args = [id.toUpperCase(), name]; // CEL式がJSにコンパイル
    
    const rows = await db.query(sql, args);
    for (const row of rows) {
        yield {
            id: row.id,
            name: row.name,
            org: row.org,
            phoneNumber: row.phone_number
        };
    }
}
```

### Java高レベルインタフェース（予定）

#### 生成されるJavaコードの例
```java
public record SearchUserResult(String id, String name, String org, String phoneNumber) {}

public class SearchUserQuery {
    public static Stream<SearchUserResult> searchUsers(
            Connection db, String id, String name) throws SQLException {
        
        String sql = "SELECT id, name, org, phone_number FROM users WHERE id = ? AND name = ?";
        Object[] args = {id.toUpperCase(), name}; // CEL式がJavaにコンパイル
        
        PreparedStatement stmt = db.prepareStatement(sql);
        for (int i = 0; i < args.length; i++) {
            stmt.setObject(i + 1, args[i]);
        }
        
        ResultSet rs = stmt.executeQuery();
        return StreamSupport.stream(new ResultSetSpliterator<>(rs, row -> 
            new SearchUserResult(
                row.getString("id"),
                row.getString("name"), 
                row.getString("org"),
                row.getString("phone_number")
            )), false);
    }
}
```

### 言語固有機能

#### Go機能
- **ジェネリクス**: 型安全なデータベースインタフェース
- **イテレータ**: Go 1.23+ range-over-funcサポート
- **コンテキスト**: 組み込みキャンセレーションとタイムアウト
- **REPLサポート**: 低レベルインタープリターによるインタラクティブ開発

#### Python機能
- **データクラス**: 自動結果型生成
- **型ヒント**: 完全な型付けサポート
- **Async/Await**: 非同期データベース操作
- **コンテキストマネージャー**: リソース管理

#### Node.js/TypeScript機能
- **非同期イテレータ**: ストリーミング結果処理
- **型安全性**: 完全なTypeScriptサポート
- **Promiseベース**: モダンな非同期パターン
- **ESM/CommonJS**: モジュールシステム互換性

#### Java機能
- **レコード**: 不変結果型
- **ストリーム**: 関数型結果処理
- **JDBC統合**: 標準データベース接続
- **アノテーション処理**: コンパイル時コード生成

### 言語別命令からコードへのマッピング

| 命令 | Go | Python | Node.js | Java |
|------|----|---------|---------|----- |
| `EMIT_LITERAL` | `sql.WriteString("text")` | `sql += "text"` | `sql += "text"` | `sql.append("text")` |
| `EMIT_PARAM` | `args = append(args, var)` | `args.append(var)` | `args.push(var)` | `stmt.setObject(i, var)` |
| `EMIT_EVAL` | `args = append(args, expr())` | `args.append(eval_expr())` | `args.push(evalExpr())` | `stmt.setObject(i, evalExpr())` |
| `JUMP_IF_EXP` | `if condition { ... }` | `if condition:` | `if (condition) {` | `if (condition) {` |
| `LOOP_START` | `for item := range coll {` | `for item in coll:` | `for (const item of coll) {` | `for (var item : coll) {` |

### 実装戦略別の利点

#### Go（デュアル実装）
- **開発**: インタープリターによる高速反復
- **本番**: 生成コードによる最大パフォーマンス
- **柔軟性**: 使用ケースに基づく実行モード選択
- **REPL**: インタラクティブSQL開発とデバッグ

#### 他言語（高レベルのみ）
- **シンプルさ**: 単一実装パス
- **パフォーマンス**: 最適化された生成コード
- **保守性**: 可動部分の削減
- **焦点**: 本番対応アプリケーション

## 実装計画

### フェーズ1: 命令セット定義 ✅
1. 命令フォーマットとJSONスキーマ定義 ✅
2. 命令セットドキュメント作成 ✅
3. 多言語アーキテクチャ設計 ✅

### フェーズ2: Go低レベル実装 ✅
1. 命令インタープリター実装 ✅
2. ASTから命令へのコンパイラ作成 ✅
3. CEL式評価追加 ✅
4. ループ変数スコープ実装 ✅
5. パッケージ整理（`runtime/snapsqlgo`） ✅

### フェーズ3: 多言語高レベル実装（予定）
1. **Go高レベルジェネレータ**
   - 型安全関数生成設計
   - プリペアドステートメントキャッシュ実装
   - イテレータベース結果処理追加
   - REPL用低レベルインタープリター統合

2. **Pythonコードジェネレータ**
   - データクラスベース結果型生成
   - async/awaitパターン実装
   - 型ヒントサポート追加
   - pipインストール可能パッケージ作成

3. **Node.js/TypeScriptジェネレータ**
   - TypeScriptインタフェース生成
   - 非同期イテレータパターン実装
   - ESM/CommonJSサポート追加
   - npmパッケージ作成

4. **Javaコードジェネレータ**
   - レコードベース結果型生成
   - ストリームベース処理実装
   - アノテーション処理追加
   - Maven/Gradleアーティファクト作成

### フェーズ4: REPLとツール（Go固有）
1. **インタラクティブREPL**
   - コマンドラインインタフェース
   - テンプレート読み込みと管理
   - パラメータ操作
   - リアルタイムSQLプレビュー

2. **開発ツール**
   - テンプレート検証
   - パフォーマンスプロファイリング
   - デバッグサポート
   - IDE統合

### フェーズ5: 統合とテスト
1. 既存SnapSQLパイプラインとの統合
2. 全言語の包括的テストカバレッジ追加
3. 言語間パフォーマンスベンチマーク
4. 各言語のドキュメントと例

## アーキテクチャの利点

### Goデュアル実装
- **開発柔軟性**: インタープリター（高速反復）と生成コード（パフォーマンス）の選択
- **REPLサポート**: インタラクティブSQL開発とデバッグ
- **本番対応**: 本番ワークロード向け最適化生成コード
- **教育価値**: 命令レベルデバッグによるSQL生成理解

### 多言語一貫性
- **統一命令セット**: 全言語で同じ中間フォーマット
- **言語固有最適化**: 各言語の強みを活用
- **保守可能コードベース**: 命令ロジックと言語固有生成の明確な分離
- **拡張可能設計**: 新しいターゲット言語の追加が容易

## 未解決の問題

1. **型推論**: SQL から生成コードへの型情報伝播方法
2. **エラーハンドリング**: 各言語で使用すべきエラーハンドリングパターン
3. **パフォーマンス**: インタープリター vs 生成コードのパフォーマンス特性
4. **デバッグ**: 言語間でのデバッグ情報埋め込み方法
5. **パッケージング**: 各言語ランタイムの配布戦略

## 参考文献

### フェーズ1: 命令セット定義
1. 命令セット仕様の確定
2. 中間フォーマット用JSONスキーマの定義
3. 命令検証ロジックの作成
4. 命令セマンティクスの文書化

### フェーズ2: ASTから命令へのコンパイラ
1. AST走査と命令生成の実装
2. 制御フロー変換（if/for → ジャンプ）の処理
3. カンマと書式ロジックの実装
4. 最適化パスの追加

### フェーズ3: ランタイムインタープリター
1. 命令エグゼキュータの実装
2. パラメータ処理とスタック管理の追加
3. 制御フロー（ジャンプ、ループ）の実装
4. 状態管理とフラグの追加

### フェーズ4: 統合とテスト
1. 中間ファイル生成の更新
2. 命令インタープリターを使用するランタイムの修正
3. 既存テンプレートでの包括的テスト
4. パフォーマンスベンチマークと最適化

## 未解決の問題

1. **最適化**: どの命令レベルの最適化を実装すべきか？
2. **デバッグ**: デバッグ情報を命令にどのように埋め込むべきか？
3. **拡張**: 将来の機能に必要な追加命令は何か？
4. **パフォーマンス**: 最適化すべきパフォーマンス重要な命令パターンはあるか？
5. **検証**: 命令シーケンス検証をどのように実装すべきか？

## 参考資料

- [SnapSQL README](../README.md)
- [Go Runtime Design](./20250625-go-runtime.md)
- [コーディング標準](../coding-standard.md)
