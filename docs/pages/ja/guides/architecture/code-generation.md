# コード生成

SnapSQLは中間形式（IR）から各言語のコードを生成します。このページでは、コード生成の仕組みと実装方針を説明します。

## 概要

コード生成は以下の流れで行われます：

```
Markdown → パーサー → AST → 中間形式（IR） → コードジェネレータ → 言語別コード
```

## 中間形式からの生成

中間形式（Intermediate Representation）は言語に依存しない命令列として表現されます。

### IR の主要要素

- **命令列（Instructions）**: SQLテンプレートの実行手順
- **パラメータ定義（Parameters）**: 型情報と説明
- **レスポンス定義（Response）**: 戻り値の型情報
- **メタデータ**: クエリ名、説明、依存関係など

### コードジェネレータの役割

各言語のコードジェネレータは以下を行います：

1. IR を読み込む
2. 言語固有の型へマッピング
3. テンプレート実行ロジックを生成
4. パラメータバインディングコードを生成
5. 結果のマッピングコードを生成

## 実装方針

### テンプレート評価（CEL）

条件分岐やループは [CEL (Common Expression Language)](https://github.com/google/cel-spec) で評価されます。

```sql
SELECT * FROM users
/*# if age_filter */
WHERE age > /*= min_age */
/*# end */
```

生成されるコード（Go の例）：

```go
var sql strings.Builder
sql.WriteString("SELECT * FROM users\n")

// CEL で条件評価
if params["age_filter"].(bool) {
    sql.WriteString("WHERE age > ")
    sql.WriteString(fmt.Sprint(params["min_age"]))
}
```

### ループとカンマ処理

配列のループでは自動的にカンマを挿入します。

```sql
INSERT INTO users (name, email) VALUES
/*# for user in users */
  (/*= user.name */, /*= user.email */)
/*# end */
```

生成されるコード（Go の例）：

```go
sql.WriteString("INSERT INTO users (name, email) VALUES\n")
for i, user := range params["users"].([]map[string]any) {
    if i > 0 {
        sql.WriteString(",\n")
    }
    sql.WriteString("(?, ?)")
    args = append(args, user["name"], user["email"])
}
```

### 型マッピング

IR の型情報は各言語の型へマッピングされます：

| IR型 | Go | TypeScript | Python |
|---|---|---|---|
| integer | int64 | number | int |
| string | string | string | str |
| boolean | bool | boolean | bool |
| timestamp | time.Time | Date | datetime |
| array | []T | T[] | List[T] |
| object | map[string]any | object | Dict[str, Any] |

## 独自ジェネレータの追加

新しい言語のジェネレータを追加する手順：

1. `langs/` 配下に新しいパッケージを作成
2. `Generator` インターフェースを実装
3. IR を読み込んでコードを生成するロジックを実装
4. `cmd/snapsql/generate.go` にジェネレータを登録

### Generator インターフェース

```go
type Generator interface {
    // IR からコードを生成
    Generate(ir *intermediate.IntermediateFormat) ([]byte, error)
    
    // 出力ファイルの拡張子
    FileExtension() string
    
    // 言語名
    LanguageName() string
}
```

## 独自コマンドの追加

カスタムコマンドを追加することも可能です：

```go
// cmd/snapsql/custom.go
func init() {
    rootCmd.AddCommand(&cobra.Command{
        Use:   "custom",
        Short: "カスタムコマンド",
        RunE: func(cmd *cobra.Command, args []string) error {
            // カスタム処理
            return nil
        },
    })
}
```

## 関連ドキュメント

- [中間コード生成](./intermediate-generation.md) - IR の詳細
- [パーサーフロー](./parser-flow.md) - パースから IR 生成までの流れ
- [言語別リファレンス](../language-reference/) - 生成されたコードの使い方
