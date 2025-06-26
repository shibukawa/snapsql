# コーディング標準

フォルダ構成は以下の通り

* /: Goのロジック
    * cmd
        * snapsql: コマンドラインツールソース
    * runtime: ランタイム
        * snapsqlgo: Goライブラリ
        * python
        * node
        * java
    * examples: サンプル
    * testdata: テストデータ
    * contrib: 周辺ツール、ライブラリ、他言語のランタイムなど

## 共通

* ソースコードのコメントは英語で書く
* テストコードのテスト名は英語にする

## Goコーディング基準

* Goは1.24を基準にする
* `interface{}`ではなく`any`を使う
* `for`ループのうち、マップのキーをスライスにする、値をスライスにするといったシンプルなものは`slices`パッケージや`maps`パッケージを使えないか検討する。イテレータは指示が合った時にのみ提供する。
* ジェネリクスは指示があった場合に利用する
* 後方互換性のために同じ役割の機能セットを複数コピーするようなことはしないこと
* Linterの実行: golangci-lint run
* エラーを返すときにパラメータがない場合は必ず ``Errエラー概要`` = errors.New()という形式のセンチネルエラーを返し、fmt.Errorf()で作ったエラーを返さないこと。センチネルエラーは各ファイルのimport文の後にグローバルに定義する。パラメータがある場合も必ず先頭でセンチネルエラーをラップしてから返すこと
* テストコードでcontext.Context()が必要になったら、testing.T.Context()を使うこと
* Goの利用ライブラリ
    * Webサーバーのルーター: net/httpのServeMuxのルーター機能を使う。メソッド違いは必ず入り口で分ける
    * テストアサーション: github.com/alecthomas/assert/v2
    * CLIコマンドパーサー: github.com/alecthomas/kong
    * CLI色付け: github.com/fatih/color
    * YAML処理: github.com/goccy/go-yaml
    * 式言語: github.com/google/cel-go
    * Markdownのパース: github.com/yuin/goldmark
    * PostgreSQLとMySQLのテストではTestContainers
    * MySQLはgithub.com/go-sql-driver/mysql
	* PostgreSQLはgithub.com/jackc/pgx/v5
	* SQLiteはgithub.com/mattn/go-sqlite3
* 明示した以外、後方互換性の維持は不要です

## Dockerコーディング規約

* buildxの書き方を基準とする
* docker composeのファイル名はcompose.yaml
* ビルドイメージにはAlpine系は使わない
* デプロイ用イメージはdistroless系を使う

## TypeScriptコーディング標準

* ECMAScript 2025をベースにしてください。usingは利用可能です。
