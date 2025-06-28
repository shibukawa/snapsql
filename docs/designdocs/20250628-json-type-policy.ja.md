# 20250628-json-type-policy.ja.md

## SnapSQL型推論エンジンにおけるJSON型の扱いについて

### 制約・方針

- 現状、JSON型（例: PostgreSQLのjson/jsonb, MySQLのjson, SQLiteのjson）は、型推論エンジン上では `any` 型として扱う。
- これは、各DB方言ごとにJSON型の厳密な型表現や構造を持たせるとpull型推論の共通化・正規化が困難になるため。
- たとえば `JSONB_BUILD_OBJECT` などの関数の返却型も `any` として扱う。
- 今後、より厳密な型表現（object, array, scalar, null など）やスキーマ推論が必要になった場合は、設計・pull仕様を拡張する。

### 影響範囲

- 型推論テストでは、JSON型の返却値やカラム型は `any` であることを期待値とする。
- DB方言ごとの型名（json, jsonb, json1 など）は内部的に `any` に正規化される。
- 返却型が `any` であることによる制約や注意点は、アプリケーション側で考慮すること。

---

本方針は今後pull型推論仕様の拡張やユースケースに応じて見直す可能性がある。
