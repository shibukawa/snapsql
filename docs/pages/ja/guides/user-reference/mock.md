# モック機能

モック機能はテスト機能でテストした期待値を、アプリケーション側のユニットテストで活用できる機能です。既存のソリューションであった課題を解決します。

* SnapSQLによって生成されたコードに組み込まれているため、リポジトリインタフェースなどを作らず使えます。
* テスト機能で検証した結果をレスポンスとして使うため、実際のレスポンスとモックの値がずれて間違ったテストケースの実装を防ぎます

## Goでの使い方

特定のクエリーの関数がモックを返すべきか、実際にデータベースにアクセスするかは実行コンテキスト（`context.Context`）に記録し、実行時にその情報を元に動作が変わります。

モックデータはプログラムで直接登録する方法（`WithMock`）と、プロバイダ経由でファイルや埋め込み資産から読み込む方法（`WithMockProvider`）があります。

関数名ごとに FIFO のシナリオキューが作られ、シナリオ内の複数レスポンスが順次返されます（末尾はリピート、ただし `NoRepeat` オプションで消費後除去できます）。

コード生成時に以下の形式でモックのソースとなるデータが出力されます。通常はモックファイルは自動で生成されるため、手動でファイルを配置して参照する必要はありません。

```json
[
  {
    "name": "Fetch account",
    "responses": [
      {
        "expected": [
          {"id": 10, "name": "Primary", "status": "active"}
        ]
      }
    ]
  }
]
```

### ランタイム API（主な要素）

- `WithMock(ctx, functionName, cases, opts...)` — 指定した `MockCase` を実行コンテキストに登録します。主にテスト内でプログラム的に使います。
- `WithMockProvider(ctx, functionName, provider, opts...)` — `MockProvider` を通してモックケースをロードして登録します。ファイルや埋め込みからロードする際に便利です

それぞれ、次のようなオプションを渡せます。

```go
type MockOpt struct {
	Name         string
	Index        int
	Err          error
	LastInsertID int64
	RowsAffected int64
	NoRepeat     bool
}
```

- `Name string` — モックケース名で選択する場合に一致させるための文字列。大文字小文字を無視して照合されます。
- `Index int` — ケース配列中のインデックスで選択します（0 ベース）。範囲外はエラーになります。
- `Err error` — そのシナリオの取得時に即座に返すエラーを設定します（エラーを返すだけでレスポンスは返りません）。
- `LastInsertID int64` — `MockExecution.SQLResult()` が返す `lastInsertId` 値を上書きします。
- `RowsAffected int64` — `MockExecution.SQLResult()` が返す `rowsAffected` 値を上書きします。
- `NoRepeat bool` — シナリオが最後のレスポンスに到達したときに、そのシナリオをキューから削除する振る舞いを有効にします（削除後は `ErrMockSequenceDepleted` が返る可能性があります）。

これらのオプションは `WithMock` / `WithMockProvider` 呼び出し時に可変長引数として渡します。複数の `MockOpt` を渡すと、それぞれが順に `cases` の要素に対応するシナリオとしてキューに追加されます。

``LastInsertID``と``RowsAffected``は`RETURNING`がないDML(INSERT/UPDATE/DELETE)のレスポンスとして返される、`sql.Result`の結果として扱われます。また、`Err`を設定するとエラーレスポンスを返します。

### プログラム的に登録する例（テスト内）

下の例はテスト内で直接モックケースを登録して、生成された `AccountGet` 関数を実行して期待値を検証するパターンです（`langs/snapsqlgo/mock_runtime_test.go` を参照）。

```go
// cases は []snapsqlgo.MockCase
ctx, err := snapsqlgo.WithMock(context.Background(), "AccountGet", cases)
// その後、生成関数を通常どおり呼ぶ。生成関数内では
// if mockExec, mockMatched, mockErr := snapsqlgo.MatchMock(ctx, "AccountGet"); mockMatched { ... }
```

### ファイル/埋め込みから読み込む例

・ファイルから読み込む（テストツリーに `testdata/mock/*.json` がある想定）:

```go
provider, err := snapsqlgo.NewFilesystemMockProvider(startDir) // startDir はプロジェクトの testdata 配下ルート
ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountGet", provider)
```

・埋め込み資産から読み込む場合（`embed.FS` を利用）:

```go
var embedded embed.FS // files are embedded at build time
provider := snapsqlgo.NewEmbeddedMockProvider(embedded)
ctx, err := snapsqlgo.WithMockProvider(context.Background(), "AccountGet", provider)
```

### ファイル名のマッチング

プロバイダは関数名の複数表記（snake_case / CamelCase / lowerCamel）を探索します。大文字小文字・アンダースコア・キャメル変換）を試し、`<key>.json` が見つかればそれをロードします。したがってファイル名は慣習として小文字のスネーク（例: `account_get.json`）や Camel（`AccountGet.json`）のいずれでも動作します。

### テストケースマッチング

プロバイダ（または `WithMock` で渡す `cases`）は複数の `MockCase` を返すことがあります。どのケースが実行されるかは `WithMock`/`WithMockProvider` に渡す `MockOpt` によって決まります。実装上の挙動は次のとおりです。

- opts をまったく渡さない場合: provider が返した `[]MockCase` の全要素をそのままシナリオキューに追加します。したがって最初に呼ばれたときは配列の先頭（index 0）のケースが適用されます。
- `MockOpt` を渡す場合: `register` 内で各 `MockOpt` ごとに `selectCase(opt)` が実行され、該当する `MockCase` が選ばれてキューへ追加されます。

選択ルール（`selectCase` のロジック）:

- `Name` が指定されていれば、大文字小文字を無視して名前一致を探します。見つからない場合は `ErrMockCaseNotFound` になります。
- `Name` が空の場合は `Index` を使います。`Index` が負の場合は 0 として扱われ、`Index` が範囲外ならエラーになります。
- `MockOpt` を渡していても `Name`/`Index` を明示しない（`MockOpt{}` のように空の値を渡す）と、内部ではインデックス 0（先頭）が選ばれます。

簡単な例:

```go
// provider が次の順でケースを返す: [A, B, C]

// 省略時: 全件がキューに追加され、最初の呼び出しは A を使う
ctx, _ := snapsqlgo.WithMockProvider(ctx, "Fn", provider)

// インデックス指定: B を選ぶ (Index は 0 ベース)
ctx, _ := snapsqlgo.WithMockProvider(ctx, "Fn", provider, snapsqlgo.MockOpt{Index: 1})

// 名前指定: C を選ぶ
ctx, _ := snapsqlgo.WithMockProvider(ctx, "Fn", provider, snapsqlgo.MockOpt{Name: "C"})
```

注意点: `MockOpt` を与えた場合、`register` は各 `MockOpt` を順に処理してシナリオを組み立てます。`Name` が見つからない、または `Index` が不正な場合は `WithMockProvider`/`WithMock` の呼び出し時にエラーが返ります。


### 同じ関数が呼ばれるたびに別の結果を返す方法と仕組み

snapsqlgo のモックは「関数名ごとのキュー（mockQueue）」と「シナリオ（mockScenario）」で構成されます。主なポイントは次の通りです。

- 1 関数名につきキューが作られ、`WithMock` / `WithMockProvider` で渡した `MockCase` がシナリオとしてキューに追加されます。
- 各シナリオは `Responses` 配列を持ち、内部の `responseIndex` を使って呼び出しごとに次のレスポンスを指します。
- `responseIndex` が配列の末尾に達した場合の振る舞いは `MockOpt.NoRepeat` によって変わります。`NoRepeat=false`（既定）の場合は末尾のレスポンスが繰り返し返されます。`NoRepeat=true` の場合はシナリオはキューから削除され、次のシナリオへと移行します（キューが空になると `ErrMockSequenceDepleted` が返ります）。

設定例（1つのケース内に複数レスポンスを持たせる）:

```go
cases := []snapsqlgo.MockCase{
  {
    Name: "Alternate",
    Responses: []snapsqlgo.MockResponse{
      { Expected: []map[string]any{{"value": 1}} },
      { Expected: []map[string]any{{"value": 2}} },
    },
  },
}

ctx, _ := snapsqlgo.WithMock(context.Background(), "FunctionName", cases)

// 呼び出し1回目 -> Expected value 1
// 呼び出し2回目 -> Expected value 2
// 呼び出し3回目 -> NoRepeat=false の場合は value 2 を返し続ける
```

別の方法として、複数の `MockCase` を渡してキューに順次追加することもできます。例えば最初のケースは1回だけ使い、その後別のケースに切り替えたい場合は、それぞれに `Responses` を設定し、必要なら後者に `NoRepeat` を設定します。

```go
cases := []snapsqlgo.MockCase{
  { Name: "First", Responses: []snapsqlgo.MockResponse{{ Expected: []map[string]any{{"step": "one"}}}} },
  { Name: "Second", Responses: []snapsqlgo.MockResponse{{ Expected: []map[string]any{{"step": "two"}}}} },
}

ctx, _ := snapsqlgo.WithMock(context.Background(), "FunctionName", cases)

// 呼び出し1回目 -> First のレスポンス
// 呼び出し2回目 -> Second のレスポンス
```

さらに細かい制御が必要な場合は、`WithMock` に複数の `MockOpt` を渡して個々のシナリオに対して `Name`/`Index`/`NoRepeat`/`Err` 等を割り当てることができます。

