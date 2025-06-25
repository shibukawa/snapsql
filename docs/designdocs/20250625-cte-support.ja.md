# CTE（Common Table Expression）サポート設計

## 概要

このドキュメントは、SnapSQLにCTE（WITH句）サポートを実装するための設計を説明します。CTEは、SELECT、INSERT、UPDATE、DELETE文内で参照できる一時的な名前付き結果セットを定義する機能です。

## 要件

### 機能要件

1. **基本CTE対応**
   - `WITH table_name AS (subquery)` 構文の解析
   - 単一文内での複数CTE対応
   - メインクエリおよび他のCTEでのCTE参照対応

2. **再帰CTE対応**
   - `WITH RECURSIVE table_name AS (...)` 構文の解析
   - 再帰CTE構造の検証処理

3. **SnapSQL統合**
   - CTE定義内でのSnapSQLディレクティブ対応
   - CTEクエリでの変数置換対応
   - 条件付きCTE包含対応

4. **AST統合**
   - CTE構造を表現するAST拡張
   - 既存パーサーとの互換性維持

### 非機能要件

1. **パフォーマンス**: CTE解析がパーサーパフォーマンスに大きな影響を与えない
2. **保守性**: 既存パーサーからのCTEロジックの明確な分離
3. **拡張性**: 将来のCTE機能拡張を可能にする設計

## 設計

### AST拡張

#### 新しいノードタイプ

```go
// NodeType enumに追加
WITH_CLAUSE     // CTEを含むWITH句
CTE_DEFINITION  // 個別のCTE定義

// 新しいAST構造
type WithClause struct {
    BaseAstNode
    CTEs []CTEDefinition
}

type CTEDefinition struct {
    BaseAstNode
    Name      string
    Recursive bool
    Query     *SelectStatement
    Columns   []string // オプションのカラムリスト
}
```

#### SelectStatement拡張

```go
type SelectStatement struct {
    BaseAstNode
    WithClause    *WithClause      // 新規: CTE対応
    SelectClause  *SelectClause
    FromClause    *FromClause
    WhereClause   *WhereClause
    OrderByClause *OrderByClause
    GroupByClause *GroupByClause
    HavingClause  *HavingClause
    LimitClause   *LimitClause
    OffsetClause  *OffsetClause
}
```

### パーサー拡張

#### CTE解析ロジック

```go
// 新しい解析メソッド
func (p *SqlParser) parseWithClause() (*WithClause, error)
func (p *SqlParser) parseCTEDefinition() (*CTEDefinition, error)
func (p *SqlParser) parseCTEColumnList() ([]string, error)
```

#### 統合ポイント

1. **parseSelectStatement()**: 開始時のWITHキーワードチェック
2. **節制約検証**: CTE制約のための節バリデーター拡張
3. **変数解決**: CTEテーブル参照の処理

### トークナイザー統合

トークナイザーは既に`WITH`キーワードをサポートしているため、変更は不要です。

### SnapSQL統合

#### サポートパターン

```sql
-- 条件付きCTE包含
/*# if include_stats */
WITH user_stats AS (
    SELECT user_id, COUNT(*) as post_count
    FROM posts 
    WHERE created_at > /*= date_filter */
    GROUP BY user_id
)
/*# end */
SELECT u.name, /*# if include_stats */s.post_count/*# end */
FROM users u
/*# if include_stats */
LEFT JOIN user_stats s ON u.id = s.user_id
/*# end */

-- CTE内での変数置換
WITH filtered_data AS (
    SELECT * FROM /*= table_name */
    WHERE status = /*= status_filter */
)
SELECT * FROM filtered_data
```

### 実装計画

#### フェーズ1: AST拡張
1. ast.goにCTE関連ノードタイプを追加
2. SelectStatement構造の拡張
3. 新しいノードのString()メソッド実装

#### フェーズ2: パーサー拡張
1. parseWithClause()メソッドの実装
2. parseCTEDefinition()メソッドの実装
3. parseSelectStatement()へのWITH句解析統合
4. CTE固有のエラーハンドリング追加

#### フェーズ3: 検証拡張
1. CTE制約のための節バリデーター拡張
2. CTE参照検証の追加
3. 再帰CTE検証の処理

#### フェーズ4: テスト
1. CTE解析の単体テスト
2. SnapSQLディレクティブとの統合テスト
3. エラーハンドリングテスト
4. パフォーマンステスト

## 例

### 基本CTE

```sql
WITH active_users AS (
    SELECT id, name FROM users WHERE active = true
)
SELECT * FROM active_users WHERE name LIKE 'A%'
```

### 複数CTE

```sql
WITH 
    active_users AS (
        SELECT id, name FROM users WHERE active = true
    ),
    user_posts AS (
        SELECT user_id, COUNT(*) as post_count
        FROM posts
        GROUP BY user_id
    )
SELECT u.name, COALESCE(p.post_count, 0) as posts
FROM active_users u
LEFT JOIN user_posts p ON u.id = p.user_id
```

### 再帰CTE

```sql
WITH RECURSIVE employee_hierarchy AS (
    -- アンカーメンバー
    SELECT id, name, manager_id, 0 as level
    FROM employees
    WHERE manager_id IS NULL
    
    UNION ALL
    
    -- 再帰メンバー
    SELECT e.id, e.name, e.manager_id, eh.level + 1
    FROM employees e
    INNER JOIN employee_hierarchy eh ON e.manager_id = eh.id
)
SELECT * FROM employee_hierarchy ORDER BY level, name
```

### SnapSQL統合

```sql
/*# if include_hierarchy */
WITH RECURSIVE employee_hierarchy AS (
    SELECT id, name, manager_id, 0 as level
    FROM employees
    WHERE department = /*= department_filter */
    
    UNION ALL
    
    SELECT e.id, e.name, e.manager_id, eh.level + 1
    FROM employees e
    INNER JOIN employee_hierarchy eh ON e.manager_id = eh.id
    WHERE eh.level < /*= max_depth */
)
/*# end */
SELECT 
    /*# if include_hierarchy */eh.level,/*# end */
    e.name,
    e.department
FROM employees e
/*# if include_hierarchy */
INNER JOIN employee_hierarchy eh ON e.id = eh.id
/*# end */
WHERE e.active = /*= active_filter */
```

## テスト戦略

### 単体テスト

1. **基本CTE解析**
   - 単一CTE
   - 複数CTE
   - カラムリスト付きCTE
   - 再帰CTE

2. **エラーハンドリング**
   - 無効なCTE構文
   - ASキーワード不足
   - 無効なサブクエリ
   - 循環参照

3. **SnapSQL統合**
   - 条件付きCTE
   - CTE内での変数置換
   - 複雑なネストシナリオ

### 統合テスト

1. **エンドツーエンド解析**
   - CTE付き完全SQL文
   - CTEと通常節の混在
   - 大規模CTEでのパフォーマンス

2. **AST検証**
   - 正しいAST構造生成
   - 適切なノード関係
   - 文字列表現の正確性

## リスク評価

### 技術的リスク

1. **パーサー複雑性**: CTE対応によりパーサーの複雑性が増加
   - **軽減策**: CTEロジックの明確な分離、包括的テスト

2. **パフォーマンス影響**: CTE解析がパーサーを遅くする可能性
   - **軽減策**: 効率的な解析アルゴリズム、パフォーマンステスト

3. **SnapSQL統合**: 既存SnapSQL機能との複雑な相互作用
   - **軽減策**: 慎重な設計、広範囲な統合テスト

### 実装リスク

1. **再帰CTE複雑性**: 再帰CTEには複雑な検証ルールがある
   - **軽減策**: 段階的実装、基本CTEを最初に重点化

2. **後方互換性**: 変更が既存機能を破壊する可能性
   - **軽減策**: 包括的な回帰テスト

## 成功基準

1. **機能性**: 全てのCTE構文バリエーションが正しく解析される
2. **統合**: SnapSQLディレクティブがCTEとシームレスに動作する
3. **パフォーマンス**: 大幅なパフォーマンス劣化がない
4. **品質**: CTE機能の100%テストカバレッジ
5. **互換性**: 既存の全テストが継続して通過する

## 将来の拡張

1. **CTE最適化**: 最適化ヒントのためのCTE使用パターン分析
2. **高度な検証**: CTE間参照検証
3. **ドキュメント**: CTE使用パターンのドキュメント生成
4. **IDE対応**: SnapSQL付きCTEの拡張構文ハイライト
