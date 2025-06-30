# parserstep2 設計ドキュメント

## 目的

parserstep2は、トークン列からSQL文の構造をパーサーコンビネータで解析し、AST（Abstract Syntax Tree）を生成する責務を持つ。ASTは元のSQLをほぼ完全に復元できる情報を保持し、後続の意味解析や中間形式変換の基盤となる。

## 要件

- トークン列からSQL文の構造を認識し、ASTノードを生成する
- ASTノードは`parsercommon.AstNode`インターフェースを実装する
- すべてのASTノードは、元になったトークン列（RawToken）を保持し、`RawTokens() []tokenizer.Token`で取得できること
- SnapSQLディレクティブ（if/for/else/elseif/end等）や変数展開もASTノードとして表現する
- パーサーコンビネータ（parsercombinator）を活用し、柔軟な構文解析を実現する
- 元SQLの復元性を重視し、トークン列の順序・内容を極力保持する
- テスト容易性・拡張性を重視

## ASTノード設計

- コアとなるAstNode:
    - SelectStatement
        - WithClause: OptionalClause<[]CTEDefinition>
        - SelectClause: []SelectItem
        - FromClause: []TableReference
        - WhereClause: OptionalClause<Expression>
        - GroupByClause: OptionalClause<[]FieldName> // if/endでON/OFF可能な句。条件式を保持できる
        - HavingClause: OptionalClause<Expression>
        - OrderByClause: OptionalClause<[]OrderByField>
        - LimitClause: OptionalClause<Expression>
        - OffsetClause: OptionalClause<Expression>
    - InsertStatement
        - WithClause: OptionalClause<[]CTEDefinition>
        - Table: TableName
        - Columns: []FieldName
        - ValuesList: Values
        - SelectStmt: Expression
        - OnConflictClause: OptionalClause<OnConflictClause> // ON CONFLICT/ON DUPLICATE KEY UPDATE用のオプション句
        - ReturningClause: OptionalClause<[]FieldName> // すべてのDMLで利用可能なRETURNING句
    - UpdateStatement
        - WithClause: OptionalClause<[]CTEDefinition>
        - Table: TableName
        - SetClauses: []SetClause
        - WhereClause: OptionalClause<Expression>
        - ReturningClause: OptionalClause<[]FieldName>
    - DeleteStatement
        - WithClause: OptionalClause<[]CTEDefinition>
        - Table: TableName
        - WhereClause: OptionalClause<Expression>
    - MergeStatement // MERGE文は別Statementとして定義
        - WithClause: OptionalClause<[]CTEDefinition>
        - TargetTable: TableName
        - SourceTable: TableName | SubQuery
        - OnClause: Expression
        - WhenMatched: []SetClause
        - WhenNotMatched: []SetClause
    - ValuesStatement // VALUES文単体
        - Rows: [][]Expression

- サブのAstNode
    - CTEDefinition: {Name string, Recursive bool, Query *SelectStatement, Columns []string}
    - SelectItem:  SelectField | Expression | SubQuery | FunctionCall | ...
    - SelectField: {TableName: Identifier, FieldName: FieldName, Alias: Identifier, Window: WindowSpec}
    - WindowSpec: {Name: Identifier, Definition: Expression}
    - TableReference: Identifier | SubQuery | JoinClause | ...
    - JoinClause: { TableName Identifier, Alias: Identifier, JoinType, OnClause: Expression, Using: []Identifier}
    - Mapping: {Left: FieldName, Right: FieldName}
    - FieldName: Identifier
    - OrderByField: FieldName + Asc/Desc
    - Values: ValueList | SelectStatement
    - ValueList: [][]Expression
    - OnConflictClause: {Target: []FieldName, Action: []SetClause}

- AstNodeインターフェース
    - Type() NodeType
    - Position() tokenizer.Position
    - String() string
    - RawTokens() []tokenizer.Token  // 追加: 元トークン列を返す
- すべてのノード型はRawTokens()を実装し、ノード生成時に該当トークン列を保持する
- 既存のBaseAstNodeにtokensフィールドを追加し、RawTokens()で返す

## パーサー構成

- parserstep2/step2.go
    - パーサーコンビネータを使い、SQL文の主要構造（SELECT, INSERT, UPDATE, DELETE, 各種句、SnapSQLディレクティブ）をASTノードに変換
    - サブパーサーを組み合わせて柔軟な構文解析を実現
- parserstep2/step2_test.go
    - 代表的なSQL・SnapSQLテンプレートのAST生成テスト
    - RawTokens()の内容検証も含める

## サンプル: AstNodeインターフェース

```go
type AstNode interface {
    Type() NodeType
    Position() tokenizer.Position
    String() string
    RawTokens() []tokenizer.Token // 元トークン列
}
```

## サンプル: BaseAstNode

```go
type BaseAstNode struct {
    nodeType NodeType
    position tokenizer.Position
    tokens   []tokenizer.Token // 元トークン列
}

func (n *BaseAstNode) Type() NodeType { return n.nodeType }
func (n *BaseAstNode) Position() tokenizer.Position { return n.position }
func (n *BaseAstNode) RawTokens() []tokenizer.Token { return n.tokens }
```

## OptionalClauseノードについて

- GROUP BY, HAVING, ORDER BY, LIMIT, OFFSETなどのOptionalな句は、if/endで丸ごとON/OFFできる
    - その場合、ifの条件式（AstNode/Expression）をOptionalClauseノードのConditionとして保持する
    - else/elseifは不可、if/endのみ許可
    - 例: 
        - GroupByClause: OptionalClause{Condition: AstNode, Clause: GroupByClause}
    - if/endで囲まれていない場合はConditionはnil
- 節をまたぐif/endは不可。1つの句単位でのみON/OFF可能

## 今後の拡張

- ASTノードの詳細設計・追加
- パーサーコンビネータの具体的な実装方針
- SnapSQL独自拡張への対応
- テストケースの拡充

## 現時点で対応している主なSQL機能（参考情報）

- DML（データ操作言語）
    - SELECT（WITH/CTE, GROUP BY, HAVING, ORDER BY, LIMIT/OFFSET, WINDOW, RETURNING, サブクエリ, ウィンドウ関数, if/endによる句ON/OFF）
    - INSERT（VALUES, SELECT, ON CONFLICT/ON DUPLICATE KEY UPDATE, RETURNING, if/endによる句ON/OFF）
    - UPDATE（SET, WHERE, RETURNING, if/endによる句ON/OFF）
    - DELETE（WHERE, RETURNING, if/endによる句ON/OFF）
    - MERGE（ON, WHEN MATCHED/NOT MATCHED, if/endによる句ON/OFF）
    - VALUES文単体
- DDL（データ定義言語）
    - 現時点では未対応（今後の拡張候補: CREATE TABLE, ALTER TABLE, DROP TABLE等）
- COPY/IMPORT/EXPORT
    - 現時点では未対応（今後の拡張候補: COPY, LOAD DATA, EXPORT等）
- その他
    - SnapSQLディレクティブ（if/for/else/elseif/end, =var, @env など）
    - パーサーコンビネータによる柔軟な構文解析
    - ASTノードごとの元トークン列保持（RawTokens）

---

（2025-07-01 作成）
