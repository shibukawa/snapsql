# SnapSQL 新パーサー(parser2)設計ドキュメント

## 概要

SnapSQLのパーサーを保守性・拡張性・テスト容易性を重視して再設計しました。パース処理を7つのステップに分割し、各ステップを独立した関数スタイルで実装しています。パーサーの組み立てには`parsercombinator`パッケージを利用しています。

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

## ディレクトリ構成

```
parser2/
  ├── parsercommon/   # 共通型・関数・ユーティリティ
  ├── parserstep1/    # 基本構文チェック
  ├── parserstep2/    # SQL文法チェック
  ├── parserstep3/    # SnapSQLディレクティブ解析
  ├── parserstep4/    # AST構築
  ├── parserstep5/    # AST最適化
  ├── parserstep6/    # 中間形式生成
  ├── parser.go       # 外部公開API
  └── errors.go       # エラー定義
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

## 外部インターフェース

```go
// メインのパース関数
func Parse(tokens []tokenizer.Token) (any, error)

// 段階的なパース関数（デバッグ用）
func ParseStep1(tokens []tokenizer.Token) (*Step1Result, error)
func ParseStep2(result *Step1Result) (*Step2Result, error)
func ParseStep3(result *Step2Result) (*Step3Result, error)
func ParseStep4(result *Step3Result) (*Step4Result, error)
func ParseStep5(result *Step4Result) (*Step5Result, error)
func ParseStep6(result *Step5Result) (*intermediate.Instructions, error)
```

## 型システムと依存関係

### パッケージ構造

```
parser/
  ├── parsercommon/   # パーサー内部の共通型・関数・ユーティリティ
  ├── parserstep1/    # 基本構文チェック
  ├── parserstep2/    # SQL文法チェック
  ├── parserstep3/    # SnapSQLディレクティブ解析
  ├── parserstep4/    # AST構築
  ├── parserstep5/    # AST最適化
  ├── parserstep6/    # 中間形式生成
  ├── parser.go       # 外部公開API
  └── errors.go       # 公開エラー定義
```

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

   // 外部公開用の型エイリアス（必要最小限）
   type ParseResult = parsercommon.ParseResult
   type ResultMetadata = parsercommon.ResultMetadata

   // 外部公開用の新しい型
   type Options struct {
       // パース設定オプション
   }
   ```

### 依存関係の制御

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

3. **エラー型の管理**
   ```go
   // parsercommon/errors.go - 内部エラー
   var (
       errInvalidSyntax = errors.New("invalid syntax")
       errInternalError = errors.New("internal parser error")
   )

   // parser/errors.go - 公開エラー
   var (
       // 公開用のエラー型（内部エラーをラップ）
       ErrSyntax = fmt.Errorf("syntax error: %w", parsercommon.errInvalidSyntax)
       ErrParse = errors.New("parse error")
   )
   ```

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

## パフォーマンス最適化

1. **メモリ効率**
   - トークンの再利用
   - 不要なコピーの削減
   - メモリプールの使用

2. **処理速度**
   - 早期エラー検出
   - 効率的なデータ構造
   - キャッシュの活用

3. **並列処理**
   - 独立したファイルの並列パース
   - ステップ内の並列処理
   - リソース使用の最適化
