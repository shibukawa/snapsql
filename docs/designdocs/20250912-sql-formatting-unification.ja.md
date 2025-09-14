# SQL整形ロジックの共通化（実行系/表示系）

日付: 2025-09-12

## 背景 / 課題
- dry-run 表示で PostgreSQL 方言時の `? → $1..` 変換と、プレースホルダ直後スペース補正（`$1ORDER` → `$1 ORDER`）を追加した。
- 実行系（DB接続あり）のほうは `? → $n` 変換のみで、直後スペース補正が無く、生成文字列の可読性・一貫性が不足。
- また、dry-run と実行で別実装になっており、将来の仕様追加時に重複保守が発生する。

## 目的 / スコープ
- 実行系・表示系の整形ロジック（方言別プレースホルダ変換／直後スペース補正）を query パッケージへ集約し、両者で共通化する。
- ドライバ指定（実行系）／方言指定（dry-run）の双方に対応する API を提供する。
- 既存の SQL 意味は変えない（クオート内は対象外／括弧直後にスペースは入れない等）。

## 仕様
- 共有関数：
  - `query.FormatSQLForDriver(sql string, driver string) string`
    - 用途: 実行系。`postgres|pgx|postgresql` のみ `? → $1..` 変換、かつ直後が識別子のとき 1 スペース付与。
  - `query.FormatSQLForDialect(sql string, dialect string) string`
    - 用途: dry-run。方言名（`postgresql|mysql|sqlite`）で同様の整形を行う。
- クオート（'…' / "…"）内の `?` / `$<digits>` は対象外。
- スペース付与は「プレースホルダの直後が英数字 or '_' のときのみ」。

## 互換性 / 影響
- 実行系でも `$n` の直後スペースが入ることがある（例: `=$1 ORDER`）。SQL構文上問題なし。直後が `)` や `,` の場合は挿入しないため、フォーマット変化も最小。
- 既存のドライバ/方言判定は踏襲。

## 実装方針
- query パッケージに `sqlfmt.go` を追加し、上記2関数と内部ヘルパ（`convertPlaceholdersForDialect`, `ensureSpaceAfterPlaceholders`）を実装。
- dry-run (`cmd/snapsql/command_query.go`) のローカル整形関数を削除し、`query.FormatSQLForDialect` を使用。
- 実行系（`query/executor.go`）は `convertPlaceholdersForDriver` 呼出しを `FormatSQLForDriver` に差し替え。

## テスト
- 既存の dry-run テスト（cmd/snapsql）を `query.FormatSQLForDialect` 利用に更新。
- 追加で query パッケージの単体テストを最小限用意可（今回は cmd テストでカバー）。

## 将来拡張
- 方言別の追加整形（例: ベンダ固有関数の別名化）を同ファイルに集約可能。

