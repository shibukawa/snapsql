# コーディング標準

## Goコーディング基準

* Goは1.24を基準にする
* `interface{}`ではなく`any`を使う
* `for`ループのうち、シンプルなものは`slices`パッケージや`maps`パッケージを使えないか検討する
* ジェネリクスは指示があった場合に利用する
* 後方互換性のために同じ役割の機能セットを複数コピーするようなことはしないこと
* Linterの実行: go tool golangci-lint run
* エラーを返すときにパラメータがない場合は必ず ``Errエラー概要`` = errors.New()という形式のセンチネルエラーを返し、fmt.Errorf()で作ったエラーを返さないこと。センチネルエラーは各ファイルのimport文の後にグローバルに定義する。パラメータがある場合も必ず先頭でセンチネルエラーをラップしてから返すこと
* テストコードでcontext.Context()が必要になったら、testing.T.Context()を使うこと
* Goの利用ライブラリ
    * Webサーバーのルーター: net/httpのServeMuxのルーター機能を使う。メソッド違いは必ず入り口で分ける
    * テストアサーション: github.com/alecthomas/assert/v2
    * CLIコマンドパーサー: github.com/alecthomas/kong
    * CLI色付け: github.com/fatih/color
    * YAML処理: github.com/goccy/go-yaml
    * 式言語: github.com/google/cel-go

