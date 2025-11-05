# 方言 (SQL Dialect) 対応ガイド

このページは SnapSQL がサポートするデータベース方言（PostgreSQL / MySQL / SQLite など）への対応方針と、テンプレート／中間形式（IR）／ランタイムでの扱い、ジェネレータ実装時の注意点、テストと運用のコツをまとめたガイドです。

## 概要

SnapSQL はテンプレートを一度パースして中間形式（IR）へ変換し、その IR を元に各言語ジェネレータとランタイムが最終 SQL を組み立てて実行します。方言差分は次の二つの段階で扱われます：

- 静的生成側（ジェネレータ）で方言固有の SQL 片を作る（必要な場合）
- 実行時ランタイムが接続先方言を検出して方言別の命令を選択して実行する

## サポートしている方言

- PostgreSQL（推奨: フルサポート）
- MySQL / MariaDB（差異あり。MariaDB の一部機能は異なる挙動）
- SQLite（組み込み DB に最適化された扱い）

※ 現在の設定では次の4つをサポートしています: `postgres`, `mysql`, `sqlite`, `mariadb`。
  無効な方言を設定すると `LoadConfig` の検証でエラーになります。デフォルトは `postgres` です。

方言はコード生成時に対応されるため、実行時に実行プログラム側で動的に切り替えたい場合はそれぞれの方言ごとに出力してプログラム側で呼び出す関数を切り替えてください。

## 構文のレベルの方言対応

以下は代表的な差分と SnapSQL としての推奨対応です。

### LIMIT / OFFSET の書式
- MySQL 旧形式: `LIMIT offset, count` はパースエラーになります
- 標準/Postgres: `LIMIT count OFFSET offset`

推奨: IR で `LIMIT`/`OFFSET` を抽象化し、方言別命令で適切に展開する。

### RETURNING と DML
- PostgreSQL / SQLite: `RETURNING` をサポートしている場合、DML が結果行を返す
- MySQL / MariaDB: サポート状況に差があり、MySQL では基本非対応

推奨: クエリテンプレート側で `RETURNING` を使う場合は方言サポートを明示し、IR でサポートしない方言では代替ロジック（`SELECT` + `LAST_INSERT_ID()` 等）を用意する。

## 式のレベルの方言対応

以下は実装されている主な変換ルールです。

- タイムスタンプ / 日付関数
  - CURRENT_TIMESTAMP ⇄ NOW()
    - 動作: Postgres/MySQL/MariaDB をターゲットとする場合は `CURRENT_TIMESTAMP` を `NOW()` に変換します（`CURRENT_TIMESTAMP` トークンを `NOW` `(` `)` の3トークンに置換）。
      SQLite をターゲットとする場合は `NOW()` を `CURRENT_TIMESTAMP` に変換します（`NOW` の直後に `(` `)` が来ることを確認して単一トークン `CURRENT_TIMESTAMP` に置換、括弧分のトークンをスキップします）。
    - 注意: `NOW()` や `CURRENT_TIMESTAMP` に引数が付く等の非標準な用法は想定していません。

  - CURDATE() / CURTIME() ⇄ CURRENT_DATE / CURRENT_TIME
    - 動作: `CURDATE()` や `CURTIME()` を見つけた場合、Postgres/SQLite 側で `CURRENT_DATE` / `CURRENT_TIME` に変換します（`()` を除去して単一トークンに置換、括弧分のトークンをスキップ）。
    - 注意: 引数付きやネストした呼び出しには対応が限定的です。

- 真偽値
  - TRUE / FALSE ⇄ 1 / 0
    - 動作: MySQL / SQLite / MariaDB 向けに `TRUE` → `1`、`FALSE` → `0` に変換します（トークン値が文字列 "TRUE"/"FALSE" の場合に置換）。PostgreSQL では `TRUE`/`FALSE` をそのまま残します。
    - 注意: 上記は主に SQL リテラルレベルの変換です。アプリケーション側で boolean 型を扱う場合は生成コードの型マッピングにも注意してください。

- 文字列連結
  - CONCAT(...) ⇄ ||
    - 動作: `CONCAT(a,b,...)` ⇔ `a || b || ...` の変換は双方向でサポートされます。
      - `CONCAT(a,b,...)` → `a || b || ...`: Postgres / SQLite 向けに変換されます（関数引数を分割して `||` で連結するトークン列を生成）。
      - `a || b || ...` → `CONCAT(a,b,...)`: MySQL / MariaDB 向けに変換されます（複数の `||` 演算子を認識して、それらを単一の `CONCAT()` 関数呼び出しにまとめます）。
    - 注意: `||` の解析では括弧のネスト、式の境界（WHERE句、JOIN句など）を正確に識別して変換を行うため、複雑な式でも正しく処理されます。ただし、ユーザー定義関数名が `||` と紛らわしい場合や、特殊な文脈での使用は事前テストを推奨します。

- CAST と `::` の相互変換
  - 動作: `CAST(expr AS TYPE)` ⇔ `(expr)::TYPE` の相互変換が双方向でサポートされます。
    - `CAST(expr AS TYPE)` → `(expr)::TYPE`: PostgreSQL / SQLite 向けに変換されます。
    - `(expr)::TYPE` → `CAST(expr AS TYPE)`: MySQL / MariaDB 向けに変換されます（PostgreSQL/SQLite ではそのまま保持）。
  - 注意: トークン列単位での置換を行うため、複雑にネストした括弧や演算子優先度に依存する式でも正しく処理されます。ただし、極めて特殊な文法や非標準な型名を使用している場合は検証を推奨します。

実装はトークン列単位での置換に依存しており、変換時にスキップするトークン数（括弧分など）を明示的に扱っています。
