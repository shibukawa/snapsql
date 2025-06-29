# SnapSQL 新パーサー(parser2)設計ドキュメント

## 概要

SnapSQLのパーサーは、保守性・拡張性・テスト容易性を重視し、フルスクラッチで再設計します。
パース処理は複数のステップ（parserstepN）に分割し、各ステップは独立した関数スタイルで実装します。
パーサの組み立てには`parsercombinator`パッケージを利用します。

---

## 目的・背景（2025-06-30追記）

現行パーサーは拡張性・保守性・エラー検出力に課題があり、今後の機能追加やバグ修正のコストが高い。
新パーサーは段階的なパース・詳細なエラー情報・テスト容易性・既存資産との連携を重視して再設計する。

---

## 要件（2025-06-30追記）

- SQLテンプレートの構文を段階的にパースできること
- エラー箇所の特定が容易で、詳細なエラー情報を返せること
- 拡張性（新しい構文要素や制御フローの追加）が高いこと
- テスト容易性（ユニットテスト・統合テストのしやすさ）
- 既存のインターフェーススキーマや中間形式出力との連携
- パフォーマンスの維持

---

## 実装計画（2025-06-30追記）

### フェーズ1: 設計・テスト雛形作成
- 設計ドキュメント作成（本ファイル）
- テスト雛形（parser2/parserstep1/step1_test.go）作成

### フェーズ2: 字句解析（Lexer）
- トークン定義
- 字句解析器の実装
- ユニットテスト

### フェーズ3: 構文解析（Parser）
- AST構造体定義
- 構文解析器の実装
- ユニットテスト

### フェーズ4: 意味解析・中間形式出力
- 意味解析ロジックの実装
- 中間形式への変換
- ユニットテスト

### フェーズ5: 統合・E2Eテスト
- 既存パーサーとの比較テスト
- 実運用SQLテンプレートでの動作検証

---

## ディレクトリ構成

```
parser2/
  ├── parsercommon/   # 共通型・関数・ユーティリティ
  ├── parserstep1/    # ステップ1: 基本構文チェック
  ├── parserstep2/    # ステップ2: SQL文法・SnapSQLコメント種別判別
  ├── parserstep3/    # ステップ3: SnapSQLディレクティブのエラーチェック
  ├── parserstep4/    # ステップ4: AST加工（擬似IF、カンマノード等）
  ├── parserstep5/    # ステップ5: intermediate.Instructionへの変換
  ├── parser.go       # 外部公開API（Parse関数）、必要な型・エラーの再エクスポート
  └── errors.go       # センチネルエラー再エクスポート
```

---

## ステップごとの責務

### parserstep1
- **責務**: 括弧・SnapSQLディレクティブの対応など、事前の基本構文チェック
    - `ValidateParentheses`: tokenizerのトークン列を受け取り、すべての括弧（(), [], {}など）が正しく対応しているかを検証する
    - `ValidateSnapSQLDirectives`: SnapSQLのif/for/else/elseif/endディレクティブが正しくペアになっているか、ネスト構造・順序が正しいかを検証する
        - `if`/`for`でpush、`end`でpop（直前が`if`または`for`でなければエラー）
        - `else`/`elseif`は直前が`if`でなければエラー、popしない
        - stackが空で`end`ならエラー、stackが残っていればエラー
- **特徴**: parsercombinatorは使わず、シンプルな関数スタイル
- **出力**: チェック済みトークン列

### parserstep2
- **責務**: SQL文法の確認とSnapSQLコメントの種類の判別
    - 末尾のOR/AND、末尾のカンマは後続処理で自動省略・自動挿入するため、エラーチェックは緩やか
- **特徴**: parsercombinatorを利用し、SQL構造の大まかなグルーピング・分類を行う
- **出力**: 構造化されたノード列

### parserstep3
- **責務**: SnapSQLディレクティブ（if/for/else/elseif/end等）のエラーチェック
- **特徴**: ディレクティブの対応関係やネスト構造の検証
- **出力**: ディレクティブ検証済みノード列

### parserstep4
- **責務**: 
    - clause自体のON/OFFを表現するための擬似的なIF文の挿入
    - 末尾OR/ANDや末尾カンマに関する自動カンマノードやセンチネルノードの挿入
    - ASTの加工・最終調整
- **特徴**: ASTノードの加工・補正
- **出力**: 加工済みAST

### parserstep5
- **責務**: intermediateパッケージの出力に使われるinstructionへの変換
- **特徴**: ASTからinstruction列への変換
- **出力**: intermediate.Instruction列

---

## 依存関係ルール

- 各parserstepNは**parsercommonのみ参照**し、他のstepやparser2は参照しない
- parser2直下はparsercommonやparserstepNの型・関数・エラーを**再エクスポート**し、外部公開する
- センチネルエラーもparserstepNでエクスポートされたものをparser2で最低限必要なもののみ再エクスポート
- 循環参照を防ぐため、parsercommonやparserstepNはparser2や他のstepを参照しない

---

## 外部インターフェース

- `parser2.Parse(tokens []tokenizer.Token) (any, error)`
    - tokenizerのトークン列を受け取り、最終ASTまたはパース結果を返す
- parser2パッケージが外部公開する型・関数・エラーは**parsercommonやparserstepNのもののみ**（再エクスポート）

---

## 呼び出し順序例

```go
func Parse(tokens []tokenizer.Token) (any, error) {
    step1Result, err := parserstep1.ExecuteStep1(tokens)
    if err != nil {
        return nil, err
    }
    step2Result, err := parserstep2.ExecuteStep2(step1Result.Tokens)
    if err != nil {
        return nil, err
    }
    step3Result, err := parserstep3.ExecuteStep3(step2Result)
    if err != nil {
        return nil, err
    }
    step4Result, err := parserstep4.ExecuteStep4(step3Result)
    if err != nil {
        return nil, err
    }
    instructions, err := parserstep5.ExecuteStep5(step4Result)
    if err != nil {
        return nil, err
    }
    return instructions, nil
}
```

---

## コーディング標準・運用ルール

- ソースコードのコメントは英語で記述
- Go 1.24基準、`any`型を使用
- センチネルエラーは各ファイルのimport文の直後にグローバル定義
- テスト名は英語
- Linter: `golangci-lint run`
- TODOリスト管理を徹底
- 設計ドキュメントは`docs/designdocs/{日付}-{機能名}.ja.md`で管理

---

## 備考

- parsercombinatorパッケージのParser型・Evaluate関数を活用し、パーサーの組み立て・テストを行う
- ASTは各ステップで段階的に生成されるため、parser2/ast.goは作成しない
- テストコードは各parserstepN配下に配置し、テスト名は英語で統一
