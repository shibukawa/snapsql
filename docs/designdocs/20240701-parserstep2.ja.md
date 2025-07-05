# parserstep2 設計ドキュメント

## 目的

parserstep2は、トークン列からSQL文の構造をパーサーコンビネータで解析し、AST（Abstract Syntax Tree）を生成する責務を持つ。
この段階では厳密なSQL構文の検証や細かいエラーチェックは行わず、SQL文の「おおまかな構造の把握」に特化する。
細かいエラー検出をこの段階で行うと、ユーザーにとって解読困難なエラーメッセージが発生しやすいため、
SnapSQLでは「パース処理を複数のステップに分割し、まずは全体構造を柔軟に捉える」設計方針を採用している。
ASTは元のSQLをほぼ完全に復元できる情報を保持し、後続の意味解析や中間形式変換の基盤となる。

## 要件

- トークン列からSQL文の構造を認識し、ASTノードを生成する
- ASTノードは`parsercommon.AstNode`インターフェースを実装する
- すべてのASTノードは、元になったトークン列（RawToken）を保持し、`RawTokens() []tokenizer.Token`で取得できること
- SnapSQLディレクティブ（if/for/else/elseif/end等）や変数展開もASTノードとして表現する
- パーサーコンビネータ（parsercombinator）を活用し、柔軟な構文解析を実現する
- 元SQLの復元性を重視し、トークン列の順序・内容を極力保持する
- テスト容易性・拡張性を重視

## ASTノード設計

## 現行のノード設計（2025-07-05時点）

- すべてのノードは `parsercommon.AstNode` インターフェースを実装し、`RawTokens()` で元トークン列を返す。
- ノード設計は「Clauseノード（句単位）」と「Statementノード（文単位）」の2階層を基本とし、Clauseノードは元トークン列・種別・サブノード（必要に応じて）を保持。
- サブクエリはFROM句ではAS付き仮想テーブルとして、SET/WHERE等では値ノードとして扱う。
- CTE（WITH句）はWithClauseノード＋CTEDefinitionノードで表現。
- SnapSQLディレクティブもClauseノードとして扱い、if/endでON/OFF可能な句はOptionalClauseでラップ。

### 主要ノード例

- SelectStatement, InsertIntoStatement, UpdateStatement, DeleteFromStatement（Statementノード）
- SelectClause, FromClause, WhereClause, GroupByClause, HavingClause, OrderByClause, LimitClause, OffsetClause, ReturningClause, ValuesClause, OnConflictClause, SetClause, DeleteFromClause, WithClause, OptionalClause, SubQuery（Clauseノード）
- CTEDefinition, TableReference, JoinClause, TableAlias, FieldName, OrderByField など（サブノード）

### サブクエリの扱い
- FROM句のサブクエリは必ずASエイリアス付きの仮想テーブルとしてTableReferenceノード配下にSubQueryノードで格納
- SET/WHERE句等のサブクエリは値ノード（Expression/Value）として扱い、エイリアスは不要

### SnapSQLディレクティブ
- if/for/end等はClauseノードとしてASTに格納
- if/endでON/OFF可能な句はOptionalClauseノードでラップし、条件式を保持

### ASTノードの共通仕様
- すべてのノードはRawTokens()で元トークン列を返す
- ノード生成時に該当トークン列を必ず保持

---

## パーサー構成

## パーサー構成（2025-07-05時点）

- parserstep2/statement.go
    - 主要なSQL文（SELECT/INSERT/UPDATE/DELETE）のパースエントリーポイント（ParseStatement）を提供
    - 各文種ごとにClauseノードを順次抽出し、Statementノードにまとめる
    - clauseStart/parseClausesで句単位の分岐・ノード生成を管理
    - サブクエリ、CTE、FOR句、ON CONFLICT句、SnapSQLディレクティブ等もここで分岐・ノード化
- parserstep2/statement_test.go
    - テーブル駆動テストで主要なSQL文・SnapSQLテンプレートのClause/Statementノード生成を網羅的に検証
    - サブクエリ、CTE、FOR句、SnapSQLディレクティブのバリエーションもカバー
    - Clause数やノード種別、RawTokensの内容も検証

---

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
    - 現時点では未対応
- COPY/IMPORT/EXPORT
    - 現時点では未対応
- 複数命令
- その他
    - SnapSQLディレクティブ（if/for/else/elseif/end, =var, @env など）
    - パーサーコンビネータによる柔軟な構文解析
    - ASTノードごとの元トークン列保持（RawTokens）

---

## 実装状況（2025-07-05時点）

### 実装済み
- パーサーコンビネータによる主要DML（SELECT/INSERT/UPDATE/DELETE）文のパース・ASTノード生成
- 主要な句（SELECT, FROM, WHERE, GROUP BY, HAVING, ORDER BY, LIMIT, OFFSET, RETURNING, VALUES, ON CONFLICT, SET, etc.）のClauseノード化
- サブクエリのパース（FROM句のAS付きサブクエリ、SET/WHERE句のスカラサブクエリ/INサブクエリ）
- CTE（WITH句, RECURSIVE, 複数CTE, 末尾カンマ許容）のパース
- SnapSQLディレクティブ（if/for/end等）のASTノード化
- ClauseごとのON/OFF制御（OptionalClause）
- トークン列の順序・内容保持（RawTokens）
- テーブル駆動テストによる網羅的な構文検証（statement_test.go）
- サブクエリ・CTE・FOR句バリエーションのテスト網羅

### 制限・今後の課題
- サブクエリのASTノード構造は現状簡易的（FROM句以外は値扱い、ASエイリアスはFROM句のみ）
- HAVING句内サブクエリや複雑な式のパースは今後拡張予定
- DDL文（CREATE/ALTER等）、MERGE文、VALUES文単体は未実装
- SnapSQL独自拡張のさらなる対応・テスト拡充
- ASTノードの詳細設計・最適化

---

（2025-07-05 更新）
