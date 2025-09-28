# `tbls` ベースのスキーマインポートコマンド設計

## 概要
既存の `snapsql pull` はライブデータベースへ直接接続して情報スキーマを走査しますが、新しい tbls ベースのランタイム は [`tbls`](https://github.com/k1LoW/tbls) が出力した JSON を入力とし、SnapSQL から外部コマンドを実行しません。ユーザーは `.tbls.yml` を管理し、`tbls doc --format json` の結果を SnapSQL が読み取ります。`generate` や `test` など他のサブコマンドも毎回 `.tbls.yml` を再読込して DSN やスキーマファイルの場所を取得し、SnapSQL 独自の複製設定を持たない方針とします。

## 目標
- `tbls` が生成した JSON を SnapSQL の内部スキーマ構造に変換するランタイム機能を提供する（外部バイナリ呼び出しは行わない）。
- 旧 `snapsql pull` と同等の出力レイアウトやフィルタ挙動を維持する。
- DSN とスキーマパスの取得を `.tbls.yml` に一元化し、他サブコマンドからも共通ヘルパーで再利用できるようにする。

## 非目標
- `tbls doc` の実行や `tbls` インストール管理を SnapSQL が肩代わりすること。
- JSON 以外の恒久的なスキーマ成果物（YAML など）を新たに生成すること。
- `tbls` が対応していないデータベースを新たにサポートすること。

## 想定ユーザーフロー
1. ユーザーは `.tbls.yml`（または `tbls.yml`）で DSN や include/exclude、`docPath` を定義し、ビルド/テスト工程で `tbls doc --format json` を実行して `<docPath>/schema.json` を生成する。
2. `Runtime API` を実行すると、コマンドが `.tbls.yml` を探索して JSON ファイルを特定し（既定は `<docPath>/schema.json`、CLI で上書き可能）、その JSON を読み込む。
3. SnapSQL が JSON を内部構造へ変換し、その場で `snapsql generate` や `snapsql test` が利用する。`.tbls.yml` は毎回読まれ、DSN やスキーマパスを取得する。

## 提供するランタイム API
```
Runtime API

Flags:
  --tbls-config path       .tbls.yml / tbls.yml へのパス（既定: CWD から探索）
  --schema-json path       tbls が生成した schema JSON へのパス（既定: <docPath>/schema.json）
  --include-views          VIEW を取り込む（既定: true）
  --include-indexes        インデックス情報を取り込む（既定: true）
  --include pattern        追加の include パターン（複数指定可）
  --exclude pattern        追加の exclude パターン（複数指定可）
  --dry-run                解決した設定と入力ファイルのみ表示して終了
  --experimental           ロールアウト中の実験フラグ（安定後に削除）
```

互換性メモ:
- 旧 `snapsql pull` と同じデフォルト値とし、`tbls doc` を実行するだけで移行できるようにする。
- SnapSQL フィルタと `.tbls.yml` の指定が競合した場合は警告を出して差分を把握しやすくする。

## 処理フロー
```text
.tlbs.yml / snapsql.yaml を解決
   ↓
JSON ファイルを決定（CLI または <docPath>/schema.json）
   ↓
JSON を schema.Schema として読み込む
   ↓
SnapSQL 内部構造へ変換（フィルタ適用・型判定）
   ↓
内部構造を下流コマンドへ提供
```

## 入力
- **`tbls` 設定**: `.tbls.yml`（または `tbls.yml`）。DSN、`docPath`、フィルタなどを提供し、SnapSQL は毎回これを読み直す。
- **スキーマ JSON**: 通常は `<docPath>/schema.json`。CLI の `--schema-json` で固定ファイルやテスト用フィクスチャを指定できる。
- **SnapSQL 設定 (`snapsql.yaml`)**: システムカラムやジェネレータ設定、既存の include/exclude 規則を保持し、JSON 読み込み後に追加フィルタとして適用する。

## `tbls` JSON から SnapSQL へのマッピング
- `schema.Schema` 全体を `snapsql.DatabaseSchema` の配列に変換し、`schema.Driver.Name` を `DatabaseInfo.Type`、`schema.Driver.DatabaseVersion` を `DatabaseInfo.Version` に設定する。
- 各 `schema.Table` は `snapsql.TableInfo` に対応し、スキーマ名・テーブル名・コメント・種別（TABLE/VIEW）をコピーする。
  - カラム順序を保持しつつ `Nullable`、`Default`、`Comment`、長さ/精度などを転写する。
  - 既存の型マッパー（現 `pull` 配下）を `schemaimport/typemapper` に移設し、`schema.Column.Type` とドライバ名から SnapSQL 型を決定する。
  - 主キー/外部キーは `table.Constraints` と `table.Relations` から導出する。
- `schema.Indexes` は `snapsql.IndexInfo` へ変換し、`Index.Unique` または `Def` を解析してユニーク性を判断する。
- `table.Type == "VIEW"` の場合は `snapsql.ViewInfo` として `Def` / `Comment` を格納する。
- 未対応フィールドは verbose ログで参照できるようにしつつ処理を継続する。

## フィルタと上書きルール
- JSON 読込後に SnapSQL の include/exclude を適用し、従来の `pull` と同じ順序（`tbls` → SnapSQL → CLI）を維持する。
- CLI の `--include` / `--exclude` は最終段で適用し、設定ファイルを上書きする。
- `--include-views=false` で VIEW を除外し、`--include-indexes=false` でインデックス情報のみを削除する（制約情報は維持）。

## 入出力の整理
- **設定解決**
  - *入力*: CLI で渡されたパス（`--tbls-config`, `--schema-json`）、カレントディレクトリ、SnapSQL 側の実行時設定、必要に応じた環境変数。
  - *出力*: 解決済みの `schemaimport.Config`（設定ファイルパス、docPath、スキーマ JSON パス、出力ディレクトリ、include/exclude、VIEW/INDEX/スキーマディレクトリ/Dry-run の各フラグ、`tbls/config.Config` の参照を含む）。
- **Importer 構築**
  - *入力*: 解決済み `schemaimport.Config`、JSON デコーダ、ファイルシステムハンドル。
  - *出力*: `LoadSchemaJSON`、`Convert`（`[]snapsql.DatabaseSchema` を返す）を備えた `schemaimport.Importer`。各メソッドは将来的に `context.Context` を受け取る。
## 実装計画
1. **パッケージ構成**: `schemaimport` を新設し、`Config`・`Importer`・型マッパー・ワイルドカードマッチャー・JSON ヘルパーを集約する。`cmd/snapsql` から直接ランタイム API を呼び出し、`pull` は廃止する。
2. **設定解決**: `.tbls.yml` / `tbls.yml`（`.yaml` 拡張子も含む）を `tbls/config` で読み込み、`DocPath`、`DSN`、include/exclude を取得する。SnapSQL 側設定と CLI フラグをマージして有効値を算出する。設定内容はキャッシュせず毎回読み込む。
3. **JSON ソース決定**: `--schema-json` > `DocPath` + `schema.SchemaFileName` > SnapSQL 既定の優先順位でパスを決定する。ファイルが無い場合は `tbls doc --format json` を実行するようガイダンスを表示して失敗する。
4. **パースと検証**: `github.com/k1LoW/tbls/schema` で JSON をデコードし、ドライバ情報やテーブル配列が存在するか確認する。欠落時はユーザーフレンドリーなエラーを返す。
5. **変換**: 既存の `pull` マッピング処理を移設し、カラム順序や `DatabaseInfo` 付与などを実装する。
6. **統合**: ランタイムから返した `[]snapsql.DatabaseSchema` を既存処理へ直接渡し、YAML へ書き出さない。
7. **統合**: 既存コマンド（generate/test など）からランタイムヘルパーを利用できるようにする。
8. **共有ヘルパー**: 他サブコマンド向けに `.tbls.yml` から DSN やスキーマ JSON パスを取得する `schemaimport.ReadConfig()`（仮称）を提供する。
9. **置き換え計画**: ランタイムヘルパーで `pull` 依存を段階的に排除する（詳細はロードマップ参照）。

## テスト計画
- 設定解決テスト: 複数ファイル名、`docPath` 上書き、環境変数、ファイル欠如などの分岐を網羅。
- 変換テスト: 既知の `tbls` JSON（例: `examples/kanban/dbdoc/schema.json`）を使用して生成される内部構造が期待通りかを検証。
- 失敗系テスト: ドライバ情報欠落、テーブル配列空、未対応 DB ドライバなどのケースで適切なエラーを返すか確認。
- 型マッピング回帰テスト: PostgreSQL / MySQL / SQLite の代表的スキーマでマッピング結果を検証する。

## ロールアウト計画
- 初期は `--experimental` として併存しつつ、pull はすでに削除済み。
- ドキュメントやサンプルを更新し、`tbls doc` 実行後にランタイムヘルパーを利用する手順を案内する。
- 出力パリティが確認できたら ランタイムを標準経路とし、pull は既に廃止済み。

## リスクと対策
- **JSON の鮮度低下**: `--dry-run` でファイルの更新日時を表示し、再生成を促す。
- **設定競合**: `.tbls.yml` と SnapSQL フィルタの差異を警告表示し、ユーザーが調整できるようにする。
- **`tbls` バージョン差異**: 最低サポートバージョンを明示し、JSON に含まれるバージョン情報を検証する。
- **依存ライブラリ負荷**: `tbls/config` と `tbls/schema` の導入による依存増を監視し、必要であれば構造体の写経も検討する。

## 未解決事項
- 複数 JSON ファイル（スキーマ別）への対応が必要か、それとも単一ファイルで十分か。
- JSON の鮮度を検証すべきか（タイムスタンプによるガードなど）。
- `.tbls.yml` と SnapSQL フィルタの矛盾を警告止まりにするか、致命的エラーとするか。
