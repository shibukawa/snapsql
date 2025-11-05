# Code Generation

型安全なクライアントコードの生成について説明します。
## 基本的なコード生成

このツールでは、クエリ Markdown やプロジェクト設定から Go クライアントコードを生成します。生成は次のコマンドで行います（ファイルを個別に指定しません）：

```bash
snapsql generate
```

生成されたコードはプロジェクト内に出力され、生成器の実装やテンプレートによってシグネチャや型が決まります。

## ライブラリの依存

生成されたランタイムは標準ライブラリを中心に最小限の依存のみを持つよう設計しています。プロジェクトで明示的に必要となる外部依存は次の通りです：

- 式言語（CEL）の評価: `github.com/google/cel-go`
- Decimal 型サポート: `github.com/shopspring/decimal`

また、データベース接続用のドライバとして公式にサポートしているものは以下です：

- MySQL: `github.com/go-sql-driver/mysql`
- PostgreSQL: `github.com/jackc/pgx/v5`
- SQLite: `github.com/mattn/go-sqlite3`

これら以外の依存は生成済みクライアントのランタイムでは最小化されています。独自型マッピングや追加のユーティリティを使う場合は、必要に応じてプロジェクト側で依存を追加してください。

---

## 生成コードでできること — 代表的なパターン（Go の使い方）

このツールが自動生成する Go クライアントコードの代表的な利用パターンを、実際の呼び出し方に焦点を当てて紹介します。以下は「このツールでこんなことができる」を端的に伝えるためのサンプルです。

### 共通（全パターン共通の取り扱い）

- `ctx context.Context`:
    - システムカラム（ユーザーID、タイムゾーン、トレーシング情報等）の伝搬や、モックレスポンスを有効化するフラグなど、実行時メタデータを渡すために用います。
    - 生成コードは `ctx` を参照してモックデータ返却や機能フラグを切り替えることができます。
- `executor snapsqlgo.DBExecutor`:
    - `*sql.DB`, `*sql.Conn`, `*sql.Tx` を透過的に扱える共通インタフェースです。呼び出し側でトランザクションを開始した場合はその `*sql.Tx` を `executor` として渡してクエリを発行できます。
    - 生成されたコードはトランザクションを隠蔽したりはしません。アプリ側でトランザクション境界を明示的に管理できる設計です。
    - アプリ開発者がリードレプリカ／マスター切り替えやトランザクション制御を自分で行えるようにして、DB パフォーマンスをシンプルに最大化できることを目的としています。

---


### 1) 単一要素を返す SELECT（主キーで取得）

想定生成関数シグネチャ例（トップレベル関数）:

```go
func GetBoardByID(
    ctx context.Context,
    executor snapsqlgo.DBExecutor,
    boardID int64,
) (Board, error)
```

サンプルクエリ（テンプレート）:

```sql
SELECT
    id,
    name,
    status,
    archived_at,
    created_at,
    updated_at
FROM boards
WHERE id = /*= board_id */1
```

呼び出しのイメージ:

```go
res, err := GetBoardByID(ctx, executor, 42)
if err != nil {
    // DB エラー処理
}
// res を利用
```

利点: 型安全に単一行を扱えるため、呼び出し側でのアンラップやフィールド参照が簡単になります。

---

### 2) 複数レコードを返す SELECT（イテレータ返却）

クエリーを解析し、複数レコードが返るクエリーの場合、一括でスライスで返すのではなく、イテレータを返して逐次処理する設計を採用しています。


想定生成関数シグネチャ例（トップレベル関数、イテレータ返却）:

```go
func IterateCardsByListID(
    ctx context.Context,
    executor snapsqlgo.DBExecutor,
    listID int64
) iter.Seq2[*CardResult, err]
```

サンプルクエリ（テンプレート）:

```sql
SELECT
    id,
    list_id,
    title,
    description,
FROM cards
WHERE list_id = /*= list_id */1
ORDER BY position ASC
```

呼び出しのイメージ（`examples/kanban/internal/handler` のパターンに合わせる）:

```go
for item, err := range IterateCardsByListID(ctx, executor, 10) {
    if err != nil {
        // ストリーム途中のエラー処理
        break
    }
    card := *item
    // card を処理
}
```

ここで `CardResult` は典型的に次のような構造体です:

```go
type CardResult struct {
	ID          int        `json:"id"`
	ListID      int        `json:"list_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
}
```

利点: 大量データをメモリに展開せずに逐次処理でき、ネットワークや DB からのストリーミング処理に向いています。

---

### 3) 階層を持つ SELECT（親子構造）

レスポンスのフィールドに``AS``で``__``区切りの名前を与えることで、JOINした子要素をグルーピングし、スライスとして階層構造を持ったオブジェクトとして返します。2段以上の階層も可能です。それぞれの階層では主キーにあたる要素を含める必要があります。

想定生成関数シグネチャ例（トップレベル関数）:

```go
func GetBoardWithLists(
    ctx context.Context,
    executor snapsqlgo.DBExecutor,
    boardID int64,
) (BoardWithLists, error)
```

サンプルクエリ（テンプレート）:

```sql
SELECT
    b.id,
    b.name,
    b.status,
    b.archived_at,
    b.created_at,
    b.updated_at,
    l.id AS lists__id,
    l.board_id AS lists__board_id,
    l.name AS lists__name,
    c.id AS lists__cards__id,
    c.list_id AS lists__cards__list_id,
    c.title AS lists__cards__title
FROM boards b
LEFT JOIN lists l ON l.board_id = b.id AND l.is_archived = 0
LEFT JOIN cards c ON c.list_id = l.id
WHERE b.id = /*= board_id */[1]
ORDER BY l.stage_order ASC, l.position ASC, c.position ASC
```

呼び出しのイメージ:

```go
b, err := GetBoardWithLists(ctx, executor, 42)
if err != nil {
    // エラーハンドリング
}
for _, l := range b.Lists {
    // 子要素のスライスにアクセス
}
```

---

### 4) 行を返さない DML（影響行数を返す）

INSERT/UPDATE/DELETEで、RETURNINGを持たないクエリーの場合は標準ライブラリの``database/sql``のResultと同じものを返します。

想定生成関数シグネチャ例（トップレベル関数）:

```go
func DeleteCard(
    ctx context.Context,
    executor snapsqlgo.DBExecutor,
    id int64,
) (sql.Result, error)
```

サンプルクエリ:

```sql
DELETE FROM cards
WHERE id = /*= id */[1]
```

呼び出しのイメージ:

```go
res, err := DeleteCard(ctx, executor, 123)
if err != nil {
    // エラー処理
}
if rows, err := res.RowsAffected(); rows == 0 {
    // 期待した削除が行われなかった
}
```

---

### 5) RETURNING を含む DML（実行と同時に行を返す）

``RETURNING`` が付与されているクエリーの場合はSELECTと同じように単独の要素やイテレータを返します。ただし、PostgreSQLとSQLiteはフルにサポートしていますが、MariaDBはINSERTとDELETEでのみサポートしており、UPDATEではサポートしていません。MySQLはサポートしていません。

想定生成関数シグネチャ例（トップレベル関数）:

```go
func UpdateCardTitle(
    ctx context.Context,
    executor snapsqlgo.DBExecutor,
    id int64,
    title string,
) (Card, error)
```

サンプルクエリ:

```sql
UPDATE cards
SET
    title = /*= title */''
WHERE
    id = /*= id */`
RETURNING
    id,
    list_id,
    title,
    updated_at
```

呼び出しのイメージ:

```go
updated, err := UpdateCardTitle(ctx, executor, 123, "New Title")
if err != nil {
    // エラー処理
}
// 更新後の行（構造体）を利用
```

---

これらのパターンは、このツールが提供する「型安全なクライアント API」を短時間で得られることを示しています。実際の生成挙動や関数シグネチャの詳細は、ジェネレータのテンプレート実装（`langs/gogen` / `langs/snapsqlgo` 等）に依存します。

## アーキテクチャと内部設計

この節では `docs/designdocs/` にある設計文書の要点をまとめ、テンプレートから実行可能コードまでの主要な内部コンポーネントとデータの流れを示します。

### 概要
- 入力: SQL テンプレート（Markdown または .snap.sql など）
- パース/解析: テンプレート内のディレクティブ（CEL式、if/for、変数など）を抽出
- 中間形式: 言語非依存の JSON 中間形式（IR）へ変換
- 生成: IR を元に各言語ジェネレータが型安全なクライアントコードを出力
- 実行時: 生成コードはランタイム（例: snapsqlgo）を利用して SQL を生成・実行・結果を返却

### テンプレート仕様（概略）
- 2-way SQL 形式を採用。コメントを除けば有効な SQL になるため、IDE や SQL リンターが活用可能。
- 制御構造: `/*# if ... */`, `/*# for ... */`, `/*# end */`（条件には Google CEL を使用）
- 変数埋め込み: `/*= expr */[dummy]`。ダミー値は開発中の直接実行やフォーマッタ互換のため推奨。
- 自動調整: 空の WHERE/ORDER/LIMIT 句の除去、自動カンマ削除、配列展開などをサポート。

（詳細は `docs/designdocs/20250625-template-specification.md` を参照）

### 中間形式 (Intermediate Format, IR)
- 目的: パーサーと各ジェネレータ間の言語非依存インターフェース。
- 主なフィールド: `format_version`, `description`, `function_name`, `parameters`, `implicit_parameters`, `instructions`, `expressions`, `envs`, `responses`, `response_affinity`。
- レスポンスの多段階階層化は `hierarchy_key_level` で表現され、JOIN の親子集約（a__b__c 形式）を正しく構築するために使われます。

（詳細は `docs/designdocs/20250625-intermediate-format.ja.md` を参照）

#### IR（中間形式）詳細

中間形式（IR）はジェネレータが取り扱いやすいようにパーサー側でテンプレートの振る舞いを平滑化した表現です。ここでは設計上の契約（契約＝generator ↔ IR）および主要フィールド／命令の意味を整理します。

- format_version: IR のバージョン（互換性チェックに使用）。ジェネレータはサポートする最大／最小バージョンを確認して適切にフォールバックまたはエラーを出します。
- function_name: 生成されるトップレベル関数名。
- description: テンプレートの説明（任意、ドキュメンテーション用途）。
- parameters / implicit_parameters: 呼び出し側に露出するパラメータ定義と、内部で生成される暗黙パラメータ（例えば LIMIT/OFFSET のデフォルトなど）。各パラメータは型、必須性、デフォルト値、説明を持ちます。
- expressions: 実行時に評価される CEL 式などのリスト。式は参照先パラメータ名と期待型を含みます。
- instructions: 実行時に順次解釈・合成される命令列。ジェネレータはこれを元にターゲット言語の SQL 組立ロジックを出力します（後述の命令一覧参照）。
- responses: SELECT / RETURNING の列情報。各列は出力名、ソースカラム、型推論結果、`hierarchy_key_level`、および nullability 情報を持ちます。
- response_affinity: レスポンスの優先度や集約ヒント（例: 主キーの位置、階層化ポリシー）を示すメタデータ。

#### 代表的な instruction（命令）タイプ

IR の `instructions` は小さな命令列の集合であり、SQL 文字列の断片や制御フロー、方言選択用の分岐を表現します。主要な命令例を列挙します:

- EMIT: そのまま出力する SQL 断片（文字列）。
- EMIT_PARAM: プレースホルダを出力する命令（プレースホルダの形式はジェネレータが方言に合わせて決定）。対応するパラメータ名を指す。
- EMIT_IF / JUMP_IF_FALSE: 条件付きで出力を行うための制御命令（式インデックスを参照）。
- 方言差分の表現: 方言ごとに異なる SQL 断片はパーサー／正規化フェーズで解決され、IR 上では最終的な断片が表現されます。
- EMIT_ARRAY_EXPAND: 配列パラメータを展開してカンマ区切りのプレースホルダ列を出力する命令。
- BEGIN_CLAUSE / END_CLAUSE: WHERE / ORDER BY 等の句を開始／終了し、空になった場合は全体をスキップするためのメタ制御。
- RESPONSE_MARKER: レスポンス列のスキャン位置や `hierarchy_key_level` を注記する命令（ランタイムでの集約に利用）。

これら命令の組合せにより、元のテンプレートにあった if/for/array 展開や方言差分が表現されます。重要な点はジェネレータが命令を解釈して SQL 文字列を「組み立てる責任」を持つことです。

#### 方言（dialect）統合の考え方

パーサーはテンプレート解析中に方言ごとの差分を検出しますが、現在はパイプラインで早期に正規化しており、IR には方言ごとに分岐する命令が残らない設計になっています（結果的に IR は方言適用済みの最終形を保持します）。このため、ジェネレータは受け取った IR をそのまま出力フォーマットに変換すればよく、実行時に方言判定して断片を切り替える必要は原則ありません。

この方式により、ジェネレータは「どの場面で方言判定が必要か」と「方言ごとにどう出力するか」という最小限のロジックだけを実装すればよく、テンプレートの詳細（複雑なネストや自動カンマ削除など）を気にせずに済みます。

#### IR のサンプル（簡易例）

次は最小限の IR JSON 例です（可読性のため整形済み）。実際は `intermediate` パッケージの構造体にマップされます。

```json
{
    "format_version": "1",
    "function_name": "GetBoardWithLists",
    "parameters": [
        {"name":"board_id","type":"int","required":true}
    ],
    "expressions": [
        {"id":0,"src":"board_id != null","type":"bool"}
    ],
    "instructions": [
        {"op":"EMIT","text":"SELECT b.id, b.name, ... FROM boards b\n"},
        // 方言差分はパイプラインで解決済みのため、IR 上では最終的な SQL 断片が直接表現されます。
        {"op":"EMIT","text":"CAST(... AS JSON)"},
        {"op":"EMIT_PARAM","param":"board_id"},
        {"op":"RESPONSE_MARKER","response_index":0}
    ],
    "responses": [
        {"name":"id","source":"b.id","type":"int","hierarchy_key_level":0},
        {"name":"lists__id","source":"l.id","type":"int","hierarchy_key_level":1}
    ]
}
```

#### ジェネレータとランタイムへの契約（責任分担）

- パーサー: テンプレートの意図（制御構造、配列展開、方言候補）を IR 命令として正確に表現する。型推論とレスポンス構造（`hierarchy_key_level` 等）を確定する。
- ジェネレータ: IR の `instructions` を解釈してターゲット言語の SQL 文字列（およびプレースホルダ）を生成する。接続先方言に応じた最終フォーマット（プレースホルダ記法、キャスト表現など）を決定する。
- ランタイム: 実行時にパラメータを受け取り、ジェネレータが作成した SQL をキャッシュ・プリペア・実行する。また `RESPONSE_MARKER` に基づくスキャン／階層化処理を実行する。

#### バージョニングと互換性

IR の `format_version` は後方互換性を意識してインクリメントされます。ジェネレータは対応する最小／最大バージョンをチェックし、非互換がある場合は明示的なエラーを出すか、互換性レイヤーで変換を試みます。IR 拡張は通常、既存の命令に非破壊的なフィールド追加で行います。

#### テストとデバッグのヒント

- 中間形式を JSON にダンプして CI の期待値に含めると、テンプレート→IR の変化を厳密に検出できます。
- 小さなテンプレート単位で `instructions` を検査し、方言別分岐や配列展開が期待どおりに表現されているか確認してください。
- `RESPONSE_MARKER` と `hierarchy_key_level` を使った集約ロジックは統合テストで検証する（JOIN を含む実 DB テストが望ましい）。

以上が IR に関する概要と実践的な注意点です。より詳細なスキーマ定義や命令セットは `docs/designdocs/20250625-intermediate-format.ja.md` を参照してください。

### パーサーと中間形式生成のフェーズ
- トークン化 → 基本構造解析 → Clause ごとの詳細解析 → 高度な処理（配列展開、ダミー検出、暗黙パラメータ処理）→ 中間命令（instructions）生成。
- 各フェーズは細かな検証（制御ブロックの整合性、変数の参照チェック等）を行い、テンプレートの安全性と一貫性を担保します。

パーサー処理は内部的に7つの明確なステップに分かれており（字句解析 → parserstep1..7）、これらを順に実行したあと型推論や検証を行い、最終的に中間形式（IR）を出力します。中間形式では方言（dialect）や命令（instructions）を統合・正規化しており、この IR を各言語のジェネレータが受け取ることで、ジェネレータ実装は方言差分やテンプレート細部に煩わされずにコード生成が行えるようになっています。

（参照: `docs/designdocs/*parser*` 系統）

### Go ランタイム（snapsqlgo）設計の要点
ランタイムは生成コードが依存する低レベルコンポーネントを提供します。主要な役割はテンプレートのロード、SQL 生成、ステートメントキャッシュ、実行です。

- Template Loader
    - 埋め込み（go:embed）やファイルシステムから中間 JSON を読み込み、テンプレートオブジェクトをキャッシュする。

- SQL Generator
    - 実行時パラメータを受け取り、IR の `instructions` を基に SQL を組み立てる。
    - 構造に影響するパラメータのみを使って SQL キャッシュキーを作成し、第2レベルキャッシュを行う。

- Statement Cache
    - sql.DB / sql.Conn / sql.Tx のコンテキスト別にプリペアドステートメントをキャッシュする第3レベルキャッシュを提供。
    - トランザクション終了時に tx 層のキャッシュをクリアするライフサイクル管理を行う。

- Executor
    - 生成 SQL の実行責任を持ち、プリペア・実行・イテレータ返却を行う。
    - Go 1.24 のイテレータパターンを使った `ResultIterator` でメモリ効率よく行を返す設計。

（参照: `docs/designdocs/20250625-go-runtime.ja.md`）

### キャッシュ戦略（簡易まとめ）
- 第1レベル: テンプレートファイル名 → 解析済テンプレート（TemplateLoader）
- 第2レベル: 構造影響パラメータハッシュ → 生成済 SQL（SQLGenerator）
- 第3レベル: コンテキスト別プリペアドステートメント（StatementCache）

「構造影響パラメータ」とはテーブルサフィックスや SELECT/ORDER の有無など、SQL の構造を変更するパラメータのことです。これらが異なると別のキャッシュキーが必要になります。一方で単純なフィルタ値は非構造影響パラメータとして SQL を再利用できます。

### 方言対応
- 中間命令（instructions）は方言固有の選択肢を含めて出力可能（例: PostgreSQL の `::` キャストと標準 SQL の `CAST(...)` を切り替える命令）。
- 実行時には接続先の方言を検出して該当する命令列を選択実行する設計です。

### レスポンス集約（階層レスポンス）
- `a__b__c` のような列名プレフィックスを解析して木構造へ展開し、重複行を親子関係として集約するロジックが IR と生成器に組み込まれています。
- 各列には `hierarchy_key_level` を付与し、ノードの PK を検出して重複を避けるアルゴリズムが適用されます。

### 運用上の注意点
- 生成コードとランタイムは Go のバージョンや DB ドライバによる影響を受けます。CI による定期的なビルドとテストを推奨します。
- テンプレートの複雑化はキャッシュヒット率に影響するため、頻繁に変化する構造的ロジックは見直しを検討してください。

---

### パーサー内部実装（詳細）

この節はパーサーの内部ステップと主要データ構造、エラー処理についての実装上の説明です（実装は `parser/` と各 `parserstepN/` パッケージに分かれています）。

1) 処理フェーズ（ステップ別）
    - 字句解析（Lexer）: コメント、文字列、数値、演算子、キーワード、そして SnapSQL ディレクティブ（`/*# if */`, `/*= ... */` 等）をトークン化します。位置情報（行/列/オフセット）をすべてのトークンに保持します。
    - 基本構文チェック（parserstep1）: 括弧整合、ディレクティブのマッチ、ネスト検査を行い早期に構文エラーを検出します。
    - SQL文法チェック（parserstep2）: 文種別（SELECT/INSERT/UPDATE/DELETE）と句の順序・構造を検証します。末尾の余分なカンマや OR/AND の不正連結はここで検出し、後続の自動調整処理へ渡します。
    - ディレクティブ解析（parserstep3）: CEL 式のパース、変数参照の解決、ディレクティブの入れ子構造検査を行います。
    - AST 構築（parserstep4）: SQL 構造を AST に変換し、擬似 if ノード、カンマノード、センチネルノードなどテンプレート処理に便利な中間ノードを挿入します。
    - AST 最適化（parserstep5）: 不要ノード除去、ノード結合、条件式の簡約、型情報付与などを行います。
    - 中間形式生成（parserstep6）: AST から `intermediate.Instruction` 列へ変換し、実行時に必要なメタ情報（式リスト、レスポンス候補、デバッグ情報など）を付加します。

2) 主要データ構造
    - Token / tokenizer.Token: 位置情報と型を持つ字句単位。
    - Node / parsercommon.Node: AST の基本ノード。ステップ固有の情報は `parserstepN` 側で拡張されます。
    - StatementNode: パースされた SQL 文のルートノード（外部公開型）。
    - ParseResult / ResultMetadata: パース結果と補助メタデータ（使用される関数定義、定数、テンプレート箇所など）。

3) エラー処理とメッセージ
    - 構文エラー: 括弧不整合、ディレクティブ不一致、SQL 構文エラー。すべて `tokenizer.Position` を使って行/列情報を含めます。
    - 意味エラー: 未定義変数参照、タイプエラー（後述の型推論で検出される場合）、不正なディレクティブ使用。
    - 変換エラー: AST → IR 変換時の不整合。デバッグ用に該当ノードのスナップショットをエラーに含めます。
    - エラー集合: 複数のエラーは `parsercommon.ParseError` に集約され、呼び出し側で個別処理可能です。

4) 実装上の注意点
    - 各ステップは独立してユニットテスト可能に設計されています（`parserstepN.Execute` を直接呼べる）。
    - InspectMode（解析専用モード）では一部の検証を緩和してインスペクション用途に適した出力を生成します。
    - パフォーマンス: トークン再利用、メモリプール、早期エラー検出で不要な後続処理を避けます。

（参照: `parser/`, `parserstep1/` ... `parserstep7/`）

### 型推論の内部実装（詳細）

型推論はテンプレート内の変数、式、結果列に対して型情報を付与し、生成コードとランタイムの型安全性を担保します。以下は設計と実装上の主要点です。

1) 全体フロー
    - スキーマ解析: 利用可能な場合はデータベーススキーマから列型情報を読み取り、初期 `TypeContext` を構築します。
    - テンプレート解析: パーサーが抽出した変数一覧、式（CEL）リスト、SELECT 列情報を取り込む。
    - 推論・解決: `InferenceEngine` が `TypeResolver` を用いて式/変数の型を推論し、必要に応じて型の統一（unify）を行う。
    - 検証: `TypeValidator` が演算ごとの型互換性をチェックし、エラーを返す。

2) 主要データ構造（概要）
    - Type / *Type: 基本型（string/int/float/bool/date/datetime/json/binary 等）や配列・オブジェクトを表現する構造体。
    - TypeContext: 変数、関数、テーブル情報を保持するコンテキスト。
    - InferenceEngine: 型推論のエントリポイント。`InferTypes(template *Template) (*TypeInfo, error)`。
    - TypeResolver: 式（Expr）から型を解決する責務。`ResolveType(expr Expr) (*Type, error)`。
    - TypeValidator: 演算や比較、関数呼び出しの型整合性検査を担当。

3) 型推論ルール（要約）
    - リテラルからの推論: ダミー値・リテラル（'abc', 123, true, '2024-01-01' 等）から対応する基本型を推論します。
    - カラム参照: スキーマ情報がある場合、カラムの DB 型を基本型へマッピングして利用します。
    - 式の推論: 算術 -> 数値優先、文字列連結 -> string、比較演算子 -> bool を返す、などのルールを適用。
    - 配列/オブジェクト: IN 句や JSON/配列操作から要素型を推論します。
    - 暗黙変換: 互換性のある型（例: 数値リテラル '123' -> int）については暗黙変換ルールを適用するが、曖昧な場合は警告または明示的キャストを推奨。

4) Type Unification（型の統一）
    - 複数ソースから得られた型情報を `UnifyTypes` で統合します。優先順位はスキーマ > 明示的ダミー値 > 式結果 > デフォルト推論。
    - 互換性がない場合は `TypeError` を返し、問題箇所の位置情報と期待型/実際型を含めます。

5) レスポンス型推論と `hierarchy_key_level` の決定
    - SELECT 列を走査し、列名とスキーマ情報、AS エイリアスを元にレスポンス列を構成します。
    - 階層化判定: 列名を `__` で分割し、プレフィックス数+1 を深さとして扱う。主キー列は `hierarchy_key_level > 0` としてマーキングします。
    - 主キー判定: スキーマ上の主キー情報が使える場合はそれを優先。無ければ `*_id` などの命名規則や `NOT NULL` 制約をヒューリスティックに利用。レベル1 が見つからない場合は最初の列をレベル1 にフォールバックして木構築の破綻を防ぎます。

6) エラー・警告の種類
    - TypeError: 型互換性違反（期待型と実際型が合わない）
    - ValidationError: サポートされない演算や明らかな不整合（例: 日付と文字列の比較で明らかに不正）
    - AmbiguityWarning: 推論が曖昧で開発者による注釈（ダミー値や明示キャスト）が望ましい場合に出す警告

7) 実装上の注意点
    - `FunctionDefinition.Finalize()` でパラメータ型解決（共通型の展開）を行い、その結果を型推論エンジンに渡す。
    - 型推論は IDE 統合や静的解析のために十分に早期に実行されるべき（生成前の検証段階でエラーを出す）。
    - DB スキーマの取得が遅い/不可能な環境ではスキーマをオプションで提供する仕組みを用意する。テスト用にダミースキーマを用意する運用も推奨。

---


## 次のステップ

## 次のステップ

ここまでで、生成されるコードのパターンと想定される利用方法を説明しました。次は生成コードを実際のプロジェクトに組み込み、動作確認と運用準備を行うための具体的な手順です。

- チェックリスト（順序どおり推奨）
    - `snapsql generate` を実行してコードを生成する（リポジトリルートで実行）。
    - 生成先のパッケージ/ファイルをプロジェクトにコミットするか、またはコード生成を CI で自動化する。（例: `go:generate` や CI ジョブ）
    - 依存を追加・整理する: `go mod tidy` を実行し、`github.com/google/cel-go`、`github.com/shopspring/decimal`、DB ドライバ（`pgx`, `mysql`, `sqlite3` 等）が `go.mod` に含まれていることを確認する。
    - ビルドとテストを実行する: `go build ./...` と `go test ./...` で生成コードが正しくコンパイル／テストされることを確認する。

- 便利なコマンド例

```bash
# 生成
snapsql generate

# 依存整理とビルド検証
go mod tidy
go build ./...
go test ./...
```

- 動作確認のポイント
    - イテレータを返すクエリ（複数行）については、生成されたイテレータの使い方を手元のコードで試し、途中で発生するエラー（ストリーム中のエラー）やリソース解放を確認してください。
    - トランザクション境界をアプリ側で管理するケースでは、`*sql.Tx` を `executor` として渡したときの挙動を確認してください（ロールバック／コミットが期待どおりに動くか）。
    - RETURNING を含む DML の戻り値（行／イテレータ）が期待通りの型になるかを確認します。DB によってサポート状況が異なる点に注意してください（PostgreSQL/SQLite は比較的フルサポート、MySQL の一部差異など）。

- 生成コードのカスタマイズと拡張
    - `langs/gogen` のテンプレートをカスタマイズして、出力する構造体タグやエラーハンドリングを変更できます。必要な場合はテンプレートを fork／編集して独自ビルドを行ってください。
    - 型マッピング（例えば decimal → 独自型）や追加ユーティリティ関数はプロジェクト側でラップして使うのが簡単です。

- トラブルシュート（よくある問題）
    - コンパイルエラー: 生成コードとプロジェクトの Go バージョンや依存の不整合が原因になることがあります。`go env` と `go.mod` を確認してください。
    - 実行時エラー: SQL のスキーマと期待カラムがずれているとパース/スキャンで失敗します。スキーマと生成クエリの整合性を検証してください（`examples/kanban` の生成例が参考になります）。

- 次にやると良いこと（任意）
    - CI に `snapsql generate` を組み込み、ソース生成をビルド前に自動実行する。
    - ドメインごとの型マッピングやユーティリティを共有ライブラリとして切り出す。
    - `examples/kanban` を読み、生成コードの実使用例（`examples/kanban/internal/query` と `internal/handler`）をプロジェクトへ取り込む参考にする。

以上でこのページは終わりです。生成コードをプロジェクトに組み込みながら、具体的な課題（例: トランザクション管理、独自型対応、パフォーマンス観察）が出てきたら、該当するガイドへ内容を追記してください。

### `snapsql init` — プロジェクトの初期化

`snapsql init` は新しい SnapSQL プロジェクトの雛形を作成する便利コマンドです。主な動作は次のとおりです。

- ディレクトリ作成
    - `queries/` (SQL テンプレート格納先)
    - `constants/` (定数ファイル用)
    - `generated/`（通常）または `internal/query`（`examples/kanban` ディレクトリ内で実行した場合の Kanban 向けスキャフォールド）

- サンプル設定ファイルの作成
    - ルートに `snapsql.yaml` を作成します。デフォルトのテンプレートには `dialect`、`databases`、`generation`（`json` / `go` などのジェネレータ設定）、`constant_files`、`validation` 等のサンプル設定が含まれます。
    - `examples/kanban` 配下で実行すると、Kanban 用に `dialect: "sqlite"` や Go 出力先が `internal/query` に設定された専用のスニペットが生成されます。

- サンプル定数ファイルの作成
    - `constants/database.yaml`（テーブル名や環境接頭辞などのサンプル）が作成されます。

- エディタ設定の作成
    - `.vscode/settings.json` に YAML スキーマの紐付けを追加して、`snapsql.yaml` の編集時にスキーマバリデーションが有効になるようにします。

注意点:
- `snapsql init` はデータベースの接続情報（パスワードや環境ごとの DSN）を自動で埋めません。必ず `snapsql.yaml` を編集して `dialect` と `databases` を環境に合わせて設定してください。
- `init` コマンド自体はオプションやフラグを取らず（グローバルな `--config` / `--verbose` / `--quiet` は反映）、実行場所により Kanban 用のスキャフォールドを切り替えます。

コマンド例:

```bash
# プロジェクトルートで初期化
snapsql init

# Kanban サンプルを開いている場合（examples/kanban 内で実行すると internal/query が作られる）
cd examples/kanban
snapsql init
```

### 中間形式（IR）を JSON として出力する（JSON ジェネレータ）

IR（中間形式）は `snapsql generate` によって JSON ファイルとして出力できます。これはテンプレートの解析結果（命令列・レスポンス定義・型情報など）を確認したり、外部ジェネレータやデバッグに利用する際に便利です。

出力方法は主に 2 通りあります:

1. `--lang` で JSON を指定して出力

```bash
snapsql generate --lang json
```

2. `snapsql.yaml` に JSON ジェネレータを有効化して `snapsql generate` を実行

例（`snapsql.yaml` 内の生成設定）:

```yaml
generation:
    input_dir: "./queries"
    generators:
        json:
            output: "./generated"
            preserve_hierarchy: true
            settings:
                pretty: true
                include_metadata: true
```

ポイント:
- デフォルトの出力先は `./generated` です（`snapsql.yaml` の `generation.generators.json.output` で変更可能）。
- `preserve_hierarchy: true` を指定すると入力のディレクトリ構成に合わせてサブディレクトリを作成して JSON を出力します（`generateOutputFilename` の挙動）。
- `settings.pretty` を有効にすると人間向けに整形された JSON が出力されます。`include_metadata` を付けると追加の解析メタデータ（出力元のファイル位置など）が含まれます。
- 実装注記: `snapsql generate` はテンプレートをパースして `intermediate.GenerateFromMarkdown` / `intermediate.GenerateFromSQL` を呼び、得られた `intermediate.IntermediateFormat` を `MarshalJSON()` してファイルに書き出します。出力される JSON はジェネレータや外部プラグインが読み取れる正式な中間形式です。

利用例:

```bash
# 1ファイルだけ中間形式を確認したい場合
snapsql generate --lang json -i queries/boards.snap.sql

# プロジェクト全体を中間形式で出力（snapsql.yaml に json generator を設定している場合）
snapsql generate
```

活用方法:
- カスタム言語ジェネレータを作る際、JSON IR を受け取って出力を作成できます（プラグインは `stdin` 経由で中間 JSON を受け取ることも可能）。
- CI ではテンプレート→IR の差分を検出することでテンプレートの意図変更を確実にトラックできます（テストに `format.MarshalJSON()` の出力を用いる）。

## 関連セクション

* [ジェネレータの概要（アーキテクチャ）](../guides/architecture/code-generation.md)
* [Go 言語リファレンス（生成コードの使い方）](../guides/language-reference/go.md)
* [`snapsql init` / プロジェクト初期化](../guides/command-reference/init.md)
