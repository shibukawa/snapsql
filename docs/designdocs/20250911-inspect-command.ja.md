# SnapSQL `inspect` サブコマンド 設計ドキュメント

## 目的
SQLファイルを読み込み、構文解析の結果から以下の最小情報をJSONで出力する。

 - ステートメント種別（select/insert/update/delete）
- 参照テーブル一覧
  - 駆動表（FROMの先頭）
  - JOIN先（JOIN種別: inner/left/right/full/cross/natural/unknown）
  - WITH（CTE）由来か、サブクエリ由来かの区別
- 参考情報（デグレードの発生など）

本機能はコード生成・中間形式生成には関与しない。解析は“読み取り専用”で、既存コマンドへの影響は与えない。

## スコープ
- 入力: 単一のSQL（ファイル/標準入力）。将来拡張としてMarkdown対応を検討可能だが、本設計では非対象。
- 出力: 標準出力にJSON（1ドキュメント）。
- パーサー利用: Step1〜Step4に加え、Step5/Step6/Step7も実行するが、InspectModeでは個別チェックを緩和する。
- FunctionDefinition: 実体の解析は行わず、名前・パラメータが空のセンチネルを生成してパイプラインに渡す。

## 仕様

### CLI
```
snapsql inspect [--stdin | <path-to-sql>]

Options:
  --stdin         標準入力からSQLを読み込む
  --pretty        整形して出力（JSON時のみ、デフォルト: false）
  --strict        厳格モード。デグレードを許容せず、解析不能時は非ゼロ終了
  --format        出力形式。json|csv（デフォルト: json）
```

終了コード:
- 0: 正常終了（デグレードあり/なし問わず）。
- 2: `--strict`指定時、解析不能などの致命エラー。

### 出力JSON（ドラフト）
```
{
  "statement": "select|insert|update|delete",
  "tables": [
    {
      "name": "schema.table | alias | cte_name | <subquery>",
      "alias": "t1",            // 省略可
      "schema": "public",       // 省略可
      "source": "main|join|cte|subquery",
      "joinType": "inner|left|right|full|cross|natural|none|unknown" // mainはnone
    }
  ],
  "notes": ["partially parsed due to syntax error"]
}
```

### 解析ポリシー（InspectMode）
- InspectModeを導入し、以下の緩和を有効化する。
- センチネルFunctionDefinitionを使用する（関数名・パラメータ空）。
- Step5: ディレクティブの整合性検証（コメントリンク/必須チェック）をスキップし、エラーにしない。配列展開/暗黙IF等の変換は副作用最小に留める（実装でフラグ化）。
- Step6: 変数/CEL検証は行わない（Namespaceは空、または検証を無効化）。
- Step7: 実行する（依存解析を利用）。
- NATURAL/CROSS JOINなど、既存でエラーにしている箇所は“エラー化せずそのまま解析継続”する（警告は出さない）。

備考:
- CTEは文種ではなく句（WITH句）として扱うため、`statement`には含めない。CTEは参照元テーブルの`source=cte`として表現する。
- INSERT INTOは対応対象。SELECT INTOは非対応（非strict時はINTO句を無視してSELECTとして解析、strict時はエラー）。
- 現状パーサーはCOPY/CREATEなどの文種を対象にしていない。
- 現行のStep7（`parserstep7`）は`FunctionDefinition`引数を受け取るが内部で未使用のため、FunctionDefinitionを用意せずに実行可能。
 - センチネルFunctionDefinitionによりステップ間の依存を解消しつつ、厳格チェックのみを明示的に無効化する。

### パーサー利用方針
- 実行パイプライン: tokenizer → Step1 → Step2 → Step3 → Step4（AST構築）→ Step5（検証無効）→ Step6（検証無効）→ Step7（依存解析）
- 依存解析（Step7）を用いる。
  - 駆動表: FROM句の先頭要素（テーブル名 or サブクエリ）。
  - JOIN: JOIN句の連鎖を走査し、種類（INNER/LEFT/…/CROSS/NATURAL）を抽出。未判別は`unknown`。NATURAL/CROSSは通常の種類として扱い、警告は出さない。
  - WITH/CTE: WITH句で定義されたCTE名を収集。FROM/ JOINで参照された場合は`source=cte`として扱う。
  - サブクエリ: FROM要素やJOIN要素が副問合せである場合、`name=<subquery>`, `source=subquery`で表現。

### CSV出力（テーブル一覧のみ）
```
name,alias,schema,source,joinType
users,u,,main,none
orders,o,,join,left
recent_orders,r,,cte,left
```
備考:
- CSVはテーブル一覧のみを出力する。
- ヘッダ行は常に出力（将来フラグで切替検討）。

### エラーハンドリング/デグレード
- 構文エラー発生時でも、可能な範囲で以下を返す。
  1) `statement`が特定できる場合はそれを返す。
  2) FROM/ JOINを部分的に抽出できる場合は抽出した分のみ返す。
  3) デグレード理由（例: 部分的にしか解析できない）は`notes[]`に記録。NATURAL/CROSS JOINは記録しない。
- `--strict`時は致命エラーで非ゼロ終了（上記デグレードを行わない）。

## 内部設計

### パッケージ構成
```
inspect/
  inspector.go        // InspectOptions/InspectResult/Inspectのエントリ
  extract.go          // AST/句からテーブル情報を抽出
  jsonschema.go       // 出力モデル（JSONタグ）
```

`cmd/snapsql/inspect.go` にCLIハンドラを追加し、`inspect.Inspect`を呼び出す。

### API
```go
type InspectOptions struct {
    InspectMode bool // 常にtrueで利用（将来拡張のフック）
    Strict      bool
    Pretty      bool
}

type TableRef struct {
    Name     string  `json:"name"`
    Alias    string  `json:"alias,omitempty"`
    Schema   string  `json:"schema,omitempty"`
    Source   string  `json:"source"`   // main|join|cte|subquery
    JoinType string  `json:"joinType"` // mainはnone
}

type InspectResult struct {
    Statement string     `json:"statement"`
    Tables    []TableRef `json:"tables"`
    Notes     []string   `json:"notes,omitempty"`
}

func Inspect(r io.Reader, opt InspectOptions) (InspectResult, error)
```

### 解析アルゴリズム概要
1. 入力を`tokenizer.Tokenize`。
2. `parserstep1.Execute` → `parserstep2.Execute` → `parserstep3.Execute` → `parserstep4.Execute` を順に適用。
   - 途中エラーはInspectMode時に可能な範囲で継続し、`notes`に格納（`--strict`なら即時終了）。
3. AST（または句情報）から以下を抽出:
   - `statement`: ルートの文種を判定。
   - `tables`:
     - WITH句: 定義済みCTE名集合を作成。
     - FROM句: 先頭要素→`source=main`。テーブル名/エイリアス/スキーマを分解。サブクエリなら`name=<subquery>`, `source=subquery`。
     - JOIN句: 連鎖を走査して`source=join`, `joinType`を設定。NATURAL/CROSSはそのまま反映し、設計上エラーにはしない。
     - CTE参照: FROM/JOINの参照がCTE名集合に含まれる場合、`source=cte`に置換。

## 非目標（Non-Goals）
- 中間形式（intermediate.Instructions）やコード生成への出力。
- 型検証/CEL式検証（Step6）およびサブクエリ依存解析（Step7）。

## 互換性と影響
- 既存の`generate`/`query`には影響しない独立機能。
- Parserには`InspectMode`導入のための軽微なAPI追加が必要（後続フェーズ）。

## テスト方針
- 単体（inspectパッケージ）
  - SELECT（単一表/別名）
  - JOIN（INNER/LEFT/RIGHT/FULL/CROSS/NATURAL）
  - WITH/CTE定義と参照
  - FROMサブクエリ
  - 破損SQL（デグレード動作、`notes`確認）
- 統合（CLI）
  - ファイル/STDIN入力、`--pretty`有無、`--strict`時の終了コード

## 段階的導入計画（対応するTODO）
1) 設計ドキュメント（本書）
2) inspectパッケージの骨組み（型・JSONモデル・最小配線）
3) 句走査の実装と単体テスト
4) CLI追加と統合テスト
5) Parserへの`InspectMode`/緩和分岐の最小導入（必要箇所のみ）
6) ドキュメント更新（CLI/README）と英訳

## 入出力サンプル

以下は`--pretty`指定時の出力例。

### サンプル1: WITH + JOIN（LEFT）
入力SQL:
```
WITH recent_orders AS (
  SELECT user_id, max(ordered_at) AS last_order
  FROM orders
  GROUP BY user_id
)
SELECT u.id, u.name, r.last_order
FROM users u
LEFT JOIN recent_orders r ON r.user_id = u.id;
```

出力JSON:
```
{
  "statement": "select",
  "tables": [
    { "name": "users", "alias": "u", "source": "main", "joinType": "none" },
    { "name": "recent_orders", "alias": "r", "source": "cte", "joinType": "left" }
  ]
}
```

補足:
- CTE `recent_orders` はFROMで参照されているため `source=cte` として列挙。

### サンプル2: NATURAL/CROSS JOIN（警告なし）
入力SQL:
```
SELECT *
FROM accounts a
NATURAL JOIN profiles p
CROSS JOIN regions r;
```

出力JSON:
```
{
  "statement": "select",
  "tables": [
    { "name": "accounts", "alias": "a", "source": "main", "joinType": "none" },
    { "name": "profiles", "alias": "p", "source": "join", "joinType": "natural" },
    { "name": "regions",  "alias": "r", "source": "join", "joinType": "cross" }
  ]
}
```

補足:
- InspectModeではNATURAL/CROSSを通常のJOIN種別として扱い、警告やnotesは出力しない。

### サンプル3: UPDATE（ターゲット表=main、FROM参照=join）
入力SQL:
```
UPDATE orders o
SET total = o.subtotal - d.amount
FROM discounts d
WHERE o.discount_id = d.id;
```

出力JSON:
```
{
  "statement": "update",
  "tables": [
    { "name": "orders",    "alias": "o", "source": "main", "joinType": "none" },
    { "name": "discounts", "alias": "d", "source": "join", "joinType": "unknown" }
  ]
}
```

補足:
- UPDATE/DELETEでは、ターゲット表を`source=main`として扱う。
- `FROM`に列挙された参照表は`source=join`とし、JOIN種別が明示されない場合は`joinType=unknown`。

### サンプル4: 部分解析（構文エラーあり、非strict）
入力SQL:
```
SELECT u.id, u.name FROM users u LEFT JOIN orders o ON o.user_id = u.id WHERE (u.id = 1;
```

出力JSON:
```
{
  "statement": "select",
  "tables": [
    { "name": "users",  "alias": "u", "source": "main", "joinType": "none" },
    { "name": "orders", "alias": "o", "source": "join", "joinType": "left" }
  ],
  "notes": ["partially parsed due to syntax error"]
}
```

補足:
- `--strict`を指定した場合は本ケースで非ゼロ終了とし、JSONは出力しない。

### サンプル5: INSERT ... SELECT（参照元はjoin/cte/subqueryの扱い）
入力SQL:
```
WITH latest AS (
  SELECT user_id, max(ordered_at) AS last_order
  FROM orders
  GROUP BY user_id
)
INSERT INTO snapshots (user_id, last_order)
SELECT l.user_id, l.last_order
FROM latest l;
```

出力JSON:
```
{
  "statement": "insert",
  "tables": [
    { "name": "snapshots", "source": "main", "joinType": "none" },
    { "name": "latest",    "alias": "l",   "source": "cte",  "joinType": "unknown" }
  ]
}
```

補足:
- INSERTのターゲット表は`source=main`。SELECT側の参照は`source=cte|join|subquery`で表現し、JOIN種別が明示されなければ`unknown`。

### サンプル6: INSERT VALUES（即値）
入力SQL:
```
INSERT INTO audit_logs (event, created_at)
VALUES ('login', NOW());
```

出力JSON:
```
{
  "statement": "insert",
  "tables": [
    { "name": "audit_logs", "source": "main", "joinType": "none" }
  ]
}
```

補足:
- VALUES句のみの場合はターゲット表のみを列挙する（`source=main`）。

### サンプル7: SELECT INTO（非対応、非strictでは無視）
入力SQL:
```
SELECT u.id, u.name
INTO tmp_users
FROM users u;
```

非strict出力JSON（INTOを無視しSELECTとして解析）:
```
{
  "statement": "select",
  "tables": [
    { "name": "users", "alias": "u", "source": "main", "joinType": "none" }
  ],
  "notes": ["ignored select-into target in inspect"]
}
```

補足:
- `--strict`時は非ゼロ終了（SELECT INTOは非対応）。

> 注: INSERTは初期リリースでは非対応のため、サンプルは削除。将来的に対応範囲に含める場合は、INSERT ... SELECT と INSERT VALUES の双方に対する出力仕様を定義し、ここに再掲する。
