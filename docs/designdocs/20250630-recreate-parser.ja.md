# SnapSQL 新パーサー(parser2)設計ドキュメント

## 概要

SnapSQLのパーサーは保守性・拡張性・テスト容易性を重視した新構成で実装済みです。パース処理を段階（ステップ）に分割し、各ステップを独立した関数スタイルで実装しています。実装では `parsercombinator` パッケージを利用しています。

## パース処理フロー

### 1. 字句解析（Lexer）
- SQLテキストをトークン列に分解
- コメント、文字列、数値、演算子、キーワードなどを識別
- SnapSQLディレクティブの識別（/*# if */, /*= var */ など）

### 2. 基本構文チェック（parserstep1）
- 括弧の対応チェック（(), [], {}）
- SnapSQLディレクティブの対応チェック（if/for/else/elseif/end）
- ネスト構造の検証
- 基本的な構文エラーの検出

### 3. SQL文法チェック（parserstep2）
- SELECT、INSERT、UPDATE、DELETE文の基本構造確認
- 句（WHERE、ORDER BY、GROUP BY等）の順序チェック
- 末尾のOR/AND、カンマの検出（後続処理で自動処理）
- SQL構造のグルーピング

### 4. SnapSQLディレクティブ解析（parserstep3）
- ディレクティブの構文解析
- 条件式の解析（CEL式）
- 変数参照の解析
- ディレクティブのネスト関係の検証

### 5. AST構築（parserstep4）
- SQL構造をASTに変換
- 擬似IF文の挿入（句のON/OFF制御用）
- カンマノードの自動挿入
- センチネルノードの挿入

### 6. AST最適化（parserstep5）
- 不要なノードの削除
- ノードの結合
- 条件式の最適化
- 型情報の付加

### 7. 中間形式生成（parserstep6）
- ASTからintermediate.Instruction列への変換
- 実行時に必要な情報の付加
- デバッグ情報の付加

## ディレクトリ構成（現行実装）

```
parser/
  ├── parsercommon/   # 共通型・関数・ユーティリティ
  ├── parserstep1/    # 基本構文チェック
  ├── parserstep2/    # SQL文法チェック
  ├── parserstep3/    # 句レベル検証/割当
  ├── parserstep4/    # 句内容の検証
  ├── parserstep5/    # ディレクティブ構造検証（InspectMode時は緩和）
  ├── parserstep6/    # 変数/CEL検証・リテラル展開（InspectMode時は緩和）
  ├── parserstep7/    # サブクエリ依存解析（常時有効／失敗しても全体は継続）
  ├── parse.go        # 外部公開API（エントリポイント群）
  ├── options.go      # パーサーオプション（InspectModeのみ）
  └── ...
```

## エラーハンドリング

### 1. 構文エラー
- 括弧の不一致
- ディレクティブの不一致
- SQL文法エラー
- 詳細な位置情報（行、列）を含むエラーメッセージ

### 2. 意味エラー
- 未定義変数の参照
- 型の不一致
- 不正なディレクティブの使用
- コンテキスト情報を含むエラーメッセージ

### 3. 変換エラー
- AST構築エラー
- 中間形式生成エラー
- デバッグ情報を含むエラーメッセージ

## テスト戦略

### 1. ユニットテスト
- 各ステップの独立したテスト
- エッジケースの網羅
- エラーケースの検証

### 2. 統合テスト
- 全ステップを通した処理の検証
- 実際のSQLテンプレートを使用したテスト
- パフォーマンステスト

### 3. 回帰テスト
- 既存パーサーとの出力比較
- 既存テストケースの再利用
- バグ修正の検証

## 外部インターフェース（現行実装）

公開APIは以下です（`parser/parse.go`）。`ParseStepN` という公開関数は存在しません。段階ごとの検証は `parserstepN` パッケージの `Execute*` をテストから直接利用します。

```go
// 代表的なオプション（将来追加予定なし）
type Options struct {
    InspectMode bool
}

// トークン列からパース（関数定義と定数を伴う）
func RawParse(tokens []tokenizer.Token, functionDef *FunctionDefinition, constants map[string]any, opts Options) (StatementNode, error)

// SQLファイルをパース（関数定義はコメントから抽出）
func ParseSQLFile(reader io.Reader, constants map[string]any, basePath string, projectRootPath string, opts Options) (StatementNode, *FunctionDefinition, error)

// Markdownドキュメント（SnapSQL）をパース
func ParseMarkdownFile(doc *markdownparser.SnapSQLDocument, basePath string, projectRootPath string, constants map[string]any, opts Options) (StatementNode, *FunctionDefinition, error)
```

段階的な実行（テスト/デバッグ用途）は以下の内部APIを直接利用します。

```go
// parserstep1
func Execute(tokens []tokenizer.Token) ([]tokenizer.Token, error)

// parserstep2
func Execute(tokens []tokenizer.Token) (parser.StatementNode, error)

// parserstep3, step4
func Execute(stmt parser.StatementNode) error

// parserstep5（InspectMode対応）
func ExecuteWithOptions(stmt parser.StatementNode, functionDef *parser.FunctionDefinition, inspectMode bool) error

// parserstep6（InspectMode対応）
func ExecuteWithOptions(stmt parser.StatementNode, paramNs, constNs *parser.Namespace, inspectMode bool) error

// parserstep7（依存解析。失敗しても全体は継続）
func (p *SubqueryParserIntegrated) ParseStatement(stmt parser.StatementNode, functionDef *parser.FunctionDefinition) error
```

## 型システムと依存関係

### パッケージ構造（再掲）

上記「ディレクトリ構成（現行実装）」を参照。

### 型定義の階層構造

1. **内部共通型の定義（parsercommon）**
   ```go
   package parsercommon

   // パーサー内部で使用する基本ノード型
   type Node struct {
       Type     NodeType
       Children []Node
       // 内部処理用フィールド
   }

   // パース結果格納用の型（外部公開用）
   type ParseResult struct {
       Nodes    []Node
       Metadata ResultMetadata
   }

   // 内部処理用ユーティリティ型
   type TokenStack struct {
       // 内部実装詳細
   }
   ```

2. **ステップ固有の型（parserstepN）**
   ```go
   package parserstep4

   // ステップ固有の内部型
   type astNode struct {
       parsercommon.Node
       // ステップ固有のフィールド
   }

   // ステップの結果型（内部用）
   type step4Result struct {
       Nodes []astNode
       // メタデータ
   }
   ```

3. **公開インターフェース（parser）**
   ```go
   package parser

   // 外部公開用の新しい型（将来追加予定なし）
   type Options struct {
       InspectMode bool
   }
   ```

### 依存関係の制御（要点）

1. **パッケージ間の参照ルール**
   - `parsercommon`: パーサー内部の共通機能を提供（外部からは非公開）
   - `parserstepN`: `parsercommon`のみ参照可能（内部パッケージ）
   - `parser`: 公開インターフェースのみを提供（型エイリアスと新規型定義）

2. **型の可視性制御**
   ```go
   // parsercommon/types.go - 内部共通型
   type (
       // 内部処理用（非公開）
       tokenProcessor struct { ... }
       nodeVisitor struct { ... }

       // 結果格納用（一部を外部公開）
       ParseResult struct { ... }
       ResultMetadata struct { ... }
   )

   // parser/types.go - 公開インターフェース
   type (
       // 必要な型のみを再エクスポート
       ParseResult = parsercommon.ParseResult
       ResultMetadata = parsercommon.ResultMetadata
   )
   ```

3. **エラー情報の扱い**
   - 位置情報は `tokenizer.Position` を通じて行/列/オフセットを保持し、各ステップのエラーメッセージへ埋め込む（例: `... at %s`, `token.Position.String()`)。
   - 複数エラーは `parsercommon.ParseError` に集約し、`parser.AsParseError(err)` で取り出し可能。

### パッケージ構造の利点

1. **内部実装の隠蔽**
   - parsercommonは内部実装の詳細を含み、外部からは非公開
   - 必要な型のみをparserパッケージで再エクスポート
   - 内部処理の変更が外部に影響を与えない

2. **保守性と拡張性**
   - 内部共通処理をparsercommonに集約
   - 各ステップが独立して進化可能
   - 新しいステップの追加が容易

3. **型安全性**
   - 内部処理用の型と外部公開用の型を明確に分離
   - コンパイル時の型チェックによる安全性確保
   - インターフェースの一貫性維持

4. **テスト容易性**
   - 内部処理の単体テストが容易
   - モックやスタブの作成が簡単
   - 外部インターフェースの安定性確保

## パフォーマンス最適化（現状）

1. **メモリ効率**
   - トークンの再利用
   - 不要なコピーの削減
   - メモリプールの使用

2. **処理速度**
   - 早期エラー検出
   - 効率的なデータ構造
   - キャッシュの活用

3. **並列処理**
   - 現時点では未採用（必要性が出た時点で計測のうえ検討）
