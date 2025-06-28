# SQL式の型推論仕様（PostgreSQL/MySQL/SQLite対応）

## 目的

SQLのSELECT句等で利用される数式・演算式・関数呼び出しについて、PostgreSQL/MySQL/SQLiteの主要な仕様をカバーし、型推論エンジンで一貫した型判定ができるようにする。

---

## 1. サポート対象の式

- 算術演算（+、-、*、/、%）
- 比較演算（=、<、>、<=、>=、<>、!=）
- 論理演算（AND、OR、NOT）
- 文字列連結（CONCAT, || など）
- 関数呼び出し（SUM, AVG, COALESCE, EXTRACT, DATE_ADD など）
- CASE式
- 型キャスト（CAST, ::）
- サブクエリ（スカラ/テーブル）

---

## 2. 型推論ルール

### 2.1 算術演算
- int + int → int
- int + decimal → decimal
- decimal + float → float
- float + int → float
- decimal + decimal → decimal
- string + string（連結）→ string
- NULLが含まれる場合はnullable

### 2.2 比較・論理演算
- 比較演算の結果はbool型
- 論理演算の結果もbool型

### 2.3 文字列連結
- PostgreSQL: ||
- MySQL/SQLite: CONCAT関数
- いずれもstring型

### 2.4 型キャスト
- CAST(expr AS type) → type
- PostgreSQL: expr::type もサポート

### 2.5 CASE式
- WHEN/ELSEの全分岐の型を昇格し、最も広い型を採用
- 例: intとdecimalならdecimal、stringとintならstring
- 全分岐がnullableならnullable

### 2.6 関数
- SUM, AVG, MAX, MIN, COUNT などの集約関数はDBごとに返却型が異なる場合があるが、基本的に以下：
    - COUNT: int（nullable: false）
    - SUM: 引数型に準拠（nullable: true）
    - AVG: decimal/float（nullable: true）
    - MAX/MIN: 引数型に準拠
- 文字列関数（CONCAT, SUBSTRING, LENGTHなど）はstring/int型
- 日付関数（EXTRACT, DATE_ADD, NOWなど）はDBごとに返却型を定義

### 2.7 サブクエリ
- スカラサブクエリ: 返却カラムの型
- テーブルサブクエリ: 各カラムごとに型推論

---

## 3. DBごとの差異

### 3.1 PostgreSQL
- 文字列連結: ||
- 型キャスト: CAST, ::
- bool型あり
- 型の昇格: int→numeric→float8

### 3.2 MySQL
- 文字列連結: CONCAT
- 型キャスト: CAST
- bool型はtinyint(1)で表現
- 型の昇格: int→decimal→double

### 3.3 SQLite
- 文字列連結: ||, CONCAT
- 型キャスト: CAST
- bool型はintで表現
- 型の昇格: INTEGER→NUMERIC→REAL

---

## 4. 型昇格（プロモーション）表

| 左辺/右辺 | int    | decimal | float  | string |
|-----------|--------|---------|--------|--------|
| int       | int    | decimal | float  | string |
| decimal   | decimal| decimal | float  | string |
| float     | float  | float   | float  | string |
| string    | string | string  | string | string |

---

## 5. NULLの扱い
- どちらかがnullableなら結果もnullable
- COALESCEなど一部関数は全引数がnullableでなければnullableでない

---

## 6. 実装上の注意
- DBごとの演算子・関数の返却型差異に注意
- 型名はpullで正規化されたもの（int, decimal, float, string, bool, time, date, json, bytes など）を使う
- 型推論エンジンはDB方言をcontextで受け取り、分岐する

---

## 7. 参考
- PostgreSQL公式: https://www.postgresql.org/docs/current/datatype.html
- MySQL公式: https://dev.mysql.com/doc/refman/8.0/en/data-types.html
- SQLite公式: https://www.sqlite.org/datatype3.html
