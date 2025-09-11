# queryコマンド修正設計ドキュメント

## 概要

現在のqueryコマンドは中間フォーマットを使ってクエリを実行する仕組みになっているが、実装が不完全で実際にクエリを実行できない状態。コード生成機能が実装されたため、queryコマンドを実際に動作するように修正する。

## 現在の問題点

### 1. 中間フォーマットの読み込み問題

```go
// 現在の実装（問題あり）
func LoadIntermediateFormat(templateFile string) (*intermediate.IntermediateFormat, error) {
    data, err := intermediate.FromJSON([]byte(templateFile))  // templateFileはファイルパスなのにバイト配列として扱っている
    if err != nil {
        return nil, fmt.Errorf("failed to load template: %w", err)
    }
    return data, nil
}
```

### 2. SQL生成の未実装

```go
// 現在の実装（プレースホルダー）
func (c *SimpleCompiler) Execute(params map[string]interface{}) (string, []interface{}, error) {
    return "SELECT 1", nil, nil  // 固定値を返すだけ
}
```

### 3. dry-runモードの問題

現在のdry-runは元のテンプレートファイルを表示するだけで、実際に生成されるSQLを確認できない。

## 修正設計

### 1. テンプレートファイル処理の改善

テンプレートファイル（`.snap.sql`、`.snap.md`）から中間フォーマットを動的に生成するか、既存の中間フォーマットファイル（`.json`）を読み込む。

```go
func LoadIntermediateFormat(templateFile string) (*intermediate.IntermediateFormat, error) {
    ext := strings.ToLower(filepath.Ext(templateFile))
    
    switch ext {
    case ".json":
        // 中間フォーマットファイルを直接読み込み
        return loadFromJSON(templateFile)
    case ".sql":
        // .snap.sqlファイルから中間フォーマットを生成
        return generateFromSQL(templateFile)
    case ".md":
        // .snap.mdファイルから中間フォーマットを生成
        return generateFromMarkdown(templateFile)
    default:
        return nil, fmt.Errorf("unsupported template file format: %s", ext)
    }
}
```

### 2. SQL生成エンジンの実装

中間フォーマットの命令を実行してSQLを生成する機能を実装。

```go
type SQLGenerator struct {
    instructions []intermediate.Instruction
    expressions  []intermediate.CELExpression
    dialect      string
}

func (g *SQLGenerator) Generate(params map[string]interface{}) (string, []interface{}, error) {
    var result strings.Builder
    var sqlParams []interface{}
    
    for _, instr := range g.instructions {
        switch instr.Op {
        case intermediate.OpEmitStatic:
            result.WriteString(instr.Value)
        case intermediate.OpEmitEval:
            value, err := g.evaluateExpression(instr.ExprIndex, params)
            if err != nil {
                return "", nil, err
            }
            result.WriteString("?")
            sqlParams = append(sqlParams, value)
        // ... 他の命令の処理
        }
    }
    
    return result.String(), sqlParams, nil
}
```

### 3. dry-runモードの改善

dry-runモードで実際に生成されるSQLとパラメータを表示。

```go
func (q *QueryCmd) executeDryRun(ctx *Context, params map[string]any, options query.QueryOptions) error {
    // 中間フォーマットを読み込み
    format, err := query.LoadIntermediateFormat(q.TemplateFile)
    if err != nil {
        return fmt.Errorf("failed to load template: %w", err)
    }
    
    // SQL生成
    generator := query.NewSQLGenerator(format.Instructions, format.CELExpressions, "postgresql")
    sql, sqlParams, err := generator.Generate(params)
    if err != nil {
        return fmt.Errorf("failed to generate SQL: %w", err)
    }
    
    // 結果表示
    color.Blue("Generated SQL:")
    fmt.Println(sql)
    
    if len(sqlParams) > 0 {
        color.Blue("Parameters:")
        for i, param := range sqlParams {
            fmt.Printf("  $%d: %v\n", i+1, param)
        }
    }
    
    return nil
}
```

### 4. エラーハンドリングの改善

- テンプレートファイルが見つからない場合の適切なエラーメッセージ
- パラメータが不足している場合の詳細なエラー情報
- SQL生成エラーの詳細な報告

## 実装方針

### フェーズ1: 基本的なSQL生成機能

1. `LoadIntermediateFormat`関数の修正
2. `SQLGenerator`の実装
3. 基本的な命令（`OpEmitStatic`, `OpEmitEval`）の処理

### フェーズ2: 条件分岐とループの対応

1. `OpIf`, `OpElse`, `OpEnd`の処理
2. `OpLoopStart`, `OpLoopEnd`の処理
3. CEL式の評価機能

### フェーズ3: システム機能とデータベース方言対応

1. システム命令（`OpEmitSystemLimit`等）の処理
2. データベース方言対応（`OpEmitIfDialect`）
3. 境界処理（`OpBoundary`, `OpEmitUnlessBoundary`）

### フェーズ4: dry-runモードの改善

1. 生成されるSQLの表示
2. パラメータの詳細表示
3. エラー情報の改善

## 後方互換性

- 既存のコマンドラインオプションは維持
- 既存のテンプレートファイル形式は引き続きサポート
- エラーメッセージの改善は行うが、基本的な動作は変更しない

## テスト方針

- 各フェーズごとに単体テストを作成
- 実際のテンプレートファイルを使った統合テスト
- dry-runモードの動作確認
- エラーケースのテスト

## リスク

- 中間フォーマットの構造が複雑で、すべての命令を正しく処理するのに時間がかかる可能性
- CEL式の評価でパフォーマンス問題が発生する可能性
- データベース方言の違いによる予期しない動作

## 成功基準

- `.snap.sql`と`.snap.md`ファイルからクエリが実行できる
- dry-runモードで生成されるSQLが確認できる
- エラーメッセージが分かりやすい
- 既存のテストが通る
