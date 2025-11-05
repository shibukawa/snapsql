
## 設定ファイル (`snapsql.yaml`)

- JSON Schema: プロジェクトルートの `snapsql-config.schema.json` を参照してください。エディタのスキーマ検証を利用することを推奨します。
- 環境変数展開: 設定ファイル中の文字列に `${VAR}` または `$VAR` を書くと実行時に環境変数で展開されます。またルートの `.env` が存在すれば読み込みます（`LoadConfig` が `.env` を読みます）。
- データベースへの接続情報は `.tbls.yaml`（tbls runtime）または CLI の `--db` を使います。

---

## トップレベル項目（概要）

主要なトップレベル設定と主な参照箇所:

- `dialect` (string)
  - 使用箇所: クエリの方言に依存する正規化・最適化・生成処理。
- `input_dir` (string)
  - 使用箇所: `snapsql generate` がテンプレートを探索するルートディレクトリ。
- `constant_files` (string[])
  - 使用箇所: テンプレート/生成時に読み込む定数ファイル。
- `generation` (object)
  - 使用箇所: `snapsql generate` と各ジェネレータ。ジェネレータ単位で出力や有効/無効を設定します。
- `validation` (object)
  - 使用箇所: テンプレート検証（CI や generate 実行時のチェック）。
- `query` (object)
  - 使用箇所: `snapsql query` のデフォルト値（CLI フラグがあればフラグが優先されます）。
- `system` (object)
  - 使用箇所: コード生成段階でのシステムカラム（例: created_at / updated_at 等）の扱い（INSERT/UPDATE の自動注入など）。
- `performance` (object)
  - 使用箇所: クエリ実行時間の閾値や警告に使われます。
- `tables` (map)
  - 使用箇所: テーブル単位のパフォーマンスメタデータ（期待行数など）。

---

## 各項目の詳細

### dialect
- 型: string
- 例: `postgres`, `mysql`, `sqlite`, `mariadb`
- デフォルト: `postgres`
- 備考: 無効な値は LoadConfig の検証でエラーになります。方言は code generation / SQL 正規化 に影響します。

### input_dir
- 型: string
- デフォルト: `./queries`
- 備考: `snapsql generate` は設定ファイルが指定され、かつ CLI で `--input` が与えられていない場合に、設定ファイルのディレクトリを基準に相対パスを解決します。明示的に `--input` が指定されていればそちらが優先されます。

### constant_files
- 型: string[]
- デフォルト: 空配列
- 備考: 実行時に参照される YAML などの定数定義ファイルのパスを列挙します。ファイルが存在しない場合は通常は警告扱いとなります。

### generation
ジェネレーション関連。

主なフィールド:
- `validate` (bool): テンプレート検証を行うか（デフォルト: true）
- `generate_mock_data` (bool): モックデータ生成の有無（デフォルト: false）
- `generators` (map[string]GeneratorConfig): ジェネレータ毎の設定

- `output` (string): 出力先ディレクトリ。ジェネレータが有効な場合は空だとエラーになります。
- `disabled`: `true` で明示的に無効化。未指定または `false` の場合は「有効」と見なされます。
- `preserve_hierarchy`: 入力ディレクトリの階層を保持するか（既定: true）
- `settings`: ジェネレータ固有設定

デフォルトのジェネレータ設定（`getDefaultConfig` に定義）:

- `json` ジェネレータ
  - output: `./generated`
  - デフォルトでは有効（Disabled は nil）
  - settings: `pretty: true`, `include_metadata: true`
- `go` ジェネレータ
  - output: `./internal/queries`
  - デフォルトでは無効（Disabled: true）
- `typescript` ジェネレータ
  - output: `./src/generated`
  - デフォルトでは無効（Disabled: true）

注意: `disabled` の扱いは少し特殊です。YAML で `disabled:` を省略すると内部的に `nil` になり、有効と扱われます。明示的に無効にするには `disabled: true` を指定してください。

### validation
- `strict` (bool): 厳格な検証モード（デフォルト: false）
- `rules` (string[]): 有効な検証ルールのリスト（デフォルト: `no-dynamic-table-names`, `require-parameter-types`）

### query
`snapsql query` コマンドのデフォルト値をまとめたものです。

主なフィールドとデフォルト:
- `default_format`: `table`（`table` / `json` / `csv` / `markdown`）
- `default_environment`: `development`（ただし接続情報自体は `.tbls.yaml` または `--db` によって解決されます）
- `timeout`: 30 (秒)
- `max_rows`: 1000
- `limit`: 0
- `offset`: 0
- `execute_dangerous_query`: false

接続解決の挙動:

- 実際の接続文字列やホスト情報は `snapsql.yaml` 内に書くのではなく、tbls のランタイム情報（`.tbls.yaml`）や CLI の `--db` フラグを利用して解決します。`default_environment` は tbls 側で有効な環境名を指すために使われます。

### system
システムカラム（アプリケーション共通カラム）の定義です。実装は `Config.System.Fields` を通じて読み込まれ、コード生成段階で参照されます。

構造:
- `fields` (array of SystemField)
  - `name` (string)
  - `type` (string) — ドキュメント用／検証用。実際の SQL 型変換は方言やジェネレータ側で扱われます。
  - `exclude_from_select` (bool) — デフォルトで SELECT に含めない (実装の利用箇所に依存)
  - `on_insert` / `on_update` (SystemFieldOperation)
    - `default`: デフォルト式（例: `NOW()`）
    - `parameter`: パラメータの扱い（`explicit` / `implicit` / `error` / 空文字）

実装上の重要な挙動（必ず読んでください）:

- システムカラムの自動注入はコード生成フェーズで行われます。注入が行われるのは INSERT 系文と UPDATE 文に限られます（SELECT には注入されません）。
- `on_insert` / `on_update` に `default` がある場合、ジェネレータはそのデフォルト式を INSERT/UPDATE の値として注入します。デフォルト式は方言に応じて正規化されます（例: `NOW()` → 方言依存の現在時刻関数に変換される場合があります）。
- `parameter` の意味：
  - `explicit`：ユーザが明示的にパラメータとして指定する必要がある（自動注入はされない）。
  - `implicit`：アプリケーションのコンテキスト等から内部的に供給される想定（codegen 側で暗黙パラメータとして扱われる）。
  - `error`：ユーザが値を与えた場合にエラーとする、または自動的に値を与えるべきではないことを示す。
  - 空文字 / 未指定：パラメータ扱いは行われず、`default` がある場合のみ自動注入されます。

- `explicit` のフィールドは生成時に未提供だった場合にバリデーションエラーまたは明示的な要求を行います（具体的な動作は処理系で検査されます）。

デフォルトとして `getDefaultConfig` は次の4つを用意します（ユーザが明示しなければこれらが使われます）:

- `created_at` — on_insert.default: `NOW()`
- `updated_at` — on_insert.default: `NOW()`, on_update.default: `NOW()`
- `created_by` — on_insert.parameter: `implicit`
- `updated_by` — on_insert.parameter: `implicit`, on_update.parameter: `implicit`

### performance
- `slow_query_threshold` (duration): 遅いクエリの閾値（デフォルト: `3s`）

### tables
- `tables` はテーブル名をキーにして `expected_rows` / `allow_full_scan` 等のメタデータを与えます。`expected_rows` は正の整数である必要があります。

## 接続情報（運用上の注意）

- 以前の `databases` トップレベルは現在利用されていません。接続は tbls runtime（`.tbls.yaml`）または CLI の `--db` で与えてください。
- そのため接続文字列等の機密情報は `snapsql.yaml` に直接書く運用は推奨されません。代わりに環境変数・CI シークレット・tbls 設定で管理してください。

## .tbls 設定ファイルの探索ルール

.tbls 設定（`tbls` ランタイム）からスキーマや接続情報を取得する場合、SnapSQL は次の優先順で tbls 設定ファイルを解決します（実装: `schemaimport.ResolveConfig` / `buildTblsOptions`）：

- 1) グローバル CLI フラグ `--tbls-config <path>` が指定されている場合は、そのパスを優先して使用します（相対パスはカレントディレクトリまたは `--config` の場所を基準に解決されます）。
- 2) グローバル `--config`（`snapsql.yaml` のパス）に指定されたファイル名が tbls 設定ファイル名と判定される場合（例: ファイル名が `tbls.yaml` / `tbls.yml`、または拡張子が `.tbls.yaml` / `.tbls.yml` の場合）、そのファイルを tbls 設定として使用します。
- 3) 上記いずれも指定がない場合は、ワーキングディレクトリ（または `--config` の基準ディレクトリ）から tbls の既定の候補パス（`tbls` のデフォルト設定ファイル名群）を順に探索して最初に見つかったものを使用します。

もし tbls 設定が見つからない、または tbls が DSN を返さない場合は、tbls 経由の接続解決は失敗し（`ErrTblsDatabaseUnavailable`）、CLI の `--db` など他の手段にフォールバックするか、コマンドはエラーになります。実際の探索ルールの詳細は `schemaimport.ResolveConfig` の実装に従います。

## サブコマンドごとの事前要件（設定ファイル）

各サブコマンドが実行時に参照する、あるいは事前に用意しておくと良い設定ファイルをまとめます。ここに挙げる「必要」は厳密な依存関係を示すものではなく、コマンド実行時に期待される一般的な構成を示します。

- `init` : 実行前に特に必要な設定ファイルはありません。プロジェクトを初期化するためのコマンドです。
- `generate` : `snapsql.yaml` は必須です。スキーマ情報を元にコードや JSON を生成する場合は、tbls が生成した `schema.json`（tbls の出力）を利用します（schema-aware な生成を行うジェネレータはこのファイルを参照します）。
- `query` : `snapsql.yaml` は通常必須（設定値のデフォルト参照のため）。DB に接続して実行する場合は、`--db` で DSN を直接指定するか、tbls 設定（上記探索ルールに従って `.tbls.yaml` など）を用意してください。
- `test` : `snapsql.yaml` は必須です。テスト実行時の DB 接続は次のいずれかを利用します: (a) tbls 設定ファイル（`.tbls.yaml` 等）からの既存 DB 接続、または (b) `--schema` オプションにより指定した DDL を用いてプロビジョニングされるエフェメラル DB（in-memory SQLite など）。どちらかが用意されていればテストは実行できます。
- `pull` : データベースからスキーマ抽出を行うため、実行時に `--db`（DSN）を指定するか、tbls 設定で接続情報を解決できる必要があります。`snapsql.yaml` 自体は必須ではありませんが、プロジェクト／出力パス指定のためにあると便利です。
- `validate` : `snapsql.yaml` を参照してテンプレート検証の設定（`validation`）を読む場合があります。テンプレート単体の検証は設定ファイルがなくても実行可能ですが、CI などでデフォルト設定を使いたい場合は `snapsql.yaml` を用意してください。
- `format`, `inspect` : 単体ファイルの整形／解析が主目的のため、基本的に事前設定は不要です（ただしテンプレート内の参照解決や dry-run を行う場合は `snapsql.yaml` と tbls 設定または `--db` が必要になることがあります）。

上記を踏まえ、プロジェクトでの運用例:

- 開発環境: `snapsql.yaml` をリポジトリに置き、ローカルでは `.tbls.yaml` をプロジェクトルートに配置して既存 DB に接続して `query`/`test` を実行する。CI では `--db` を用いてシークレットから DSN を渡す。
- 生成パイプライン: `generate` は `snapsql.yaml` を使い、必要に応じて tbls が出力した `schema.json` を投入して schema-aware な生成を行う。


## 最小サンプル

```yaml
dialect: postgres
input_dir: ./queries
constant_files: [./constants/database.yaml]

generation:
  validate: true
  generators:
    json:
      output: ./generated
      preserve_hierarchy: true
      settings:
        pretty: true
        include_metadata: true

query:
  default_format: table
  default_environment: development
  timeout: 30
  max_rows: 1000
  execute_dangerous_query: false

system:
  fields:
    - name: created_at
      type: timestamp
      on_insert:
        default: "NOW()"

performance:
  slow_query_threshold: "3s"

tables: {}
```

---

## 運用上の補足

- 設定ファイルは厳格パース（未知フィールドはエラー）されます。編集時は `snapsql-config.schema.json` と合わせてください。
- `input_dir` やジェネレータの `output` は設定ファイルのディレクトリを基準に相対パスが解決されます（ただし CLI の明示的な引数が優先されます）。

