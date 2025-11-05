# システムカラム

このページは、SnapSQL における「システムカラム」の設計・設定・実行時振る舞いを実装に基づいて正しく説明します。ビジネスドメインに関係ない、システム共通のメタ情報（例：ユーザー ID、トレース ID、時刻）をミドルウェア層で隠蔽し、クエリテンプレート／アプリのビジネスロジックを簡潔に保つために導入された仕組みです。共通設定に従い、すべてのINSERT/UPDATEに共通的にパラメータを追加します。

## 要旨

- システムカラムはアプリケーション全体で共通に使われるメタ情報（例：作成者・更新者・作成時刻・更新時刻）を表します。
- これらは `snapsql.yaml` の `system.fields` で定義します（グローバル設定）。
- INSERT / UPDATE 等の操作時に自動で挿入・更新できるよう、中間パイプライン（`intermediate`）で命令を注入し、生成コード／ランタイムが実行時に値を補完します。
- 生成コードごとに実際の受け渡し方法は異なります。Go 向け実装では `context.Context` に値を登録して渡す仕組みを用意しています（`langs/snapsqlgo` を参照）。

## 設定方法（snapsql.yaml の書き方）

`snapsql.yaml` における典型的な設定例:

```yaml
system:
   fields:
      - name: created_at
         type: timestamp
         exclude_from_select: false
         on_insert:
            default: NOW()
         on_update:
            parameter: error

      - name: updated_at
         type: timestamp
         exclude_from_select: false
         on_insert:
            default: NOW()
         on_update:
            default: NOW()

      - name: created_by
         type: string
         exclude_from_select: false
         on_insert:
            parameter: implicit
         on_update:
            parameter: error

      - name: updated_by
         type: string
         exclude_from_select: false
         on_insert:
            parameter: implicit
         on_update:
            parameter: implicit
```

主なフィールド
- `name`：システムカラムの名前（DB 側の列名と一致することが期待されます）。
- `type`：期待する型（生成コードや実行時検証で参照されます）。
- `exclude_from_select`：true の場合、生成コードはデフォルトで SELECT に含めないようにできます（ジェネレータ依存）。
- `on_insert` / `on_update`：各操作時の取り扱い（`default` または `parameter`）を指定します。

## 生成コードの利用例

以下は Go 向けに生成されたコードを呼び出したときの利用例と、生成コード（あるいはランタイム）によって最終的に実行されうる SQL のイメージです。実際の生成結果はジェネレータとダイアレクト（Postgres/MySQL/SQLite）によって異なりますが、概念的には次のようになります。

前提：`cards` テーブルのカラムは `title`, `description`, `created_at`, `created_by`, `updated_at`, `updated_by` とする。`created_by` / `updated_by` は `implicit`、`created_at` / `updated_at` は DB 側や `default` で補う想定。

1) 生成コードの呼び出し（Go）

```go
ctx := context.Background()
// ミドルウェア等でユーザー情報をセット
ctx = snapsqlgo.WithSystemColumnValues(ctx, map[string]any{
   "created_by": "alice",
   "updated_by": "alice",
})

// 生成された関数を呼び出す（関数名/引数は生成コード依存）
// ここでは簡潔化のため InsertCard(ctx, db, title, description) とする
res, err := generated.InsertCard(ctx, db, "Buy milk", "2 liters")
if err != nil {
   return err
}
_ = res
```

2) ランタイムが組み立てる模擬 SQL（Postgres 風の例）

上の呼び出しで、生成コードは中間命令に基づきプレースホルダとパラメータ配列を組み立てます。

例 A — `created_at` / `updated_at` を DB 側の関数（NOW()）で補う場合:

```sql
INSERT INTO cards (title, description, created_at, created_by, updated_at, updated_by)
VALUES ($1, $2, NOW(), $3, NOW(), $4);
-- パラメータ配列: ["Buy milk", "2 liters", "alice", "alice"]
```

例 B — すべて値をアプリ側で供給する場合（ランタイムが時刻も計算して渡す）:

```sql
INSERT INTO cards (title, description, created_at, created_by, updated_at, updated_by)
VALUES ($1, $2, $3, $4, $5, $6);
-- パラメータ配列: ["Buy milk", "2 liters", "2025-11-03T12:34:56Z", "alice", "2025-11-03T12:34:56Z", "alice"]
```

ポイント:
- `implicit` に指定された `created_by`/`updated_by` は生成関数の引数リストには現れず、`ctx` 経由でランタイムが補填します。
- `explicit` に指定されている場合、生成関数はそのカラムを引数として受け取る・パラメータ配列に含める設計になります。
- `error` は、呼び出し側がその列を明示的に操作しようとした場合にランタイムがエラーにする（生成コード側でチェック）ため、誤った使用を防げます。

3) UPDATE の例（暗黙パラメータを利用）

生成された Update 関数の呼び出し:

```go
ctx = snapsqlgo.WithSystemValue(ctx, "updated_by", "bob")
_, err = generated.UpdateCard(ctx, db, cardID, "New title")
```

内部ではシステムカラム月以下された次のようなSQLが発行されます。

```sql
UPDATE cards
SET title = $1, updated_at = NOW(), updated_by = $2
WHERE id = $3;
-- パラメータ配列: ["New title", "bob", <cardID>]
```

このように、生成コードは中間パイプラインで決まったルール（どの列を暗黙的に補うか、どの列を明示パラメータにするか）に従って SQL とパラメータ配列を組み立てます。

追加説明: 実際のプレースホルダの表記（`$1` / `?` / `:1` 等）は方言/ジェネレータ実装によって変わります。上記は Postgres 形式の一例です。

## parameter（モード）の意味

`on_insert.parameter` / `on_update.parameter` の代表的な値：
- `explicit`：呼び出しコード側が明示的に値を渡す（生成コードはパラメータとして扱う）。
- `implicit`：ランタイム（ミドルウェア等）が暗黙的に提供することを期待する。Go 実装なら `context.Context` 経由で渡す。必須の場合、未提供だとランタイムがエラーや panic を出すことがある。
- `error`：呼び出し側が値を与えようとするとエラーにする（アプリから無効な操作を防ぐため）。
- 空（`none` 相当）：パラメータで扱わず、`default` を使う等の運用を想定。

これらは中間パイプライン側で検証・注入され、生成コードはその情報に基づいてパラメータ一覧や SQL 片を作成します。

## 言語ランタイムのAPI

### Go 向けランタイム API（`langs/snapsqlgo` を参照）

Go 向け実装では `context.Context` を使ってシステムカラム値を渡すためのユーティリティが提供されています。代表的な関数：

- `snapsqlgo.WithSystemValue(ctx, key, value)` — 単一のシステムカラム値を ``context.Context`` に登録（`langs/snapsqlgo/context.go`）。
- `snapsqlgo.WithSystemColumnValues(ctx, map[string]any)` — 複数のシステムカラム値をまとめて ``context.Context`` に登録。
- 実行時に生成コードは ``context.Context`` から暗黙のパラメータを取り出し、型チェック・必須チェックを行います。

簡単な利用例（Go）:

```go
import (
      "context"
      snapsqlgo "github.com/shibukawa/snapsql/langs/snapsqlgo"
)

ctx := context.Background()
// 単体セット
ctx = snapsqlgo.WithSystemValue(ctx, "created_by", "alice")

// 複数まとめてセット
ctx = snapsqlgo.WithSystemColumnValues(ctx, map[string]any{
      "created_by": "alice",
      "updated_by": "alice",
})

// 生成された関数は通常 ctx を第一引数にとる（実際のシグネチャは生成コードに依存）
// 例: generated.InsertCard(ctx, db, otherParams...)
```

注意点（Go）:
- `WithSystemValue` / `WithSystemColumnValues` に登録するキーは `snapsql.yaml` の `system.fields[].name` と一致させる必要があります。
- 生成コードは暗黙パラメータの仕様（型・必須性）に基づいて ``context.Context`` から値を取り出し、型が合わない場合はランタイムでエラーになります。

## 生成コードとの関係・運用上の注意
- 生成コードは中間命令に従ってパラメータの順序や SQL 片を決めるため、`system.fields` を変更した場合は再生成が必要です。
- `exclude_from_select` の効果はジェネレータ次第です。SELECT に含めたくない場合はテンプレート側で明示的に列指定するのが安全です。
