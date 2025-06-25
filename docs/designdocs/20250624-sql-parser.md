# SQLパーサー設計ドキュメント（Phase 1: トークナイザー）

## 概要

SnapSQLのコアコンポーネントとして、SQLテンプレートを解析してトークン列を生成するSQLトークナイザーを実装する。doma2のSqlTokenizerを参考にしつつ、SnapSQL独自の要件に対応した設計とする。

**Phase 1の範囲**: トークナイズのみ。CEL評価は含まず、SnapSQLのテンプレート構文はブロックコメントとして処理する。

## 設計目標

### 主要目標
1. **データベース非依存**: PostgreSQL、MySQL、SQLiteの方言に対応
2. **高い表現力**: 複雑なSQL式を`Other`トークンとして処理し、SQLの記述性を損なわない
3. **SnapSQL拡張認識**: コメントベースのテンプレート構文（`/*# if */`、`/*= var */`等）をブロックコメントとして認識
4. **SQL構文検証**: 基本的なSQL構文の正しさを検証
5. **ウインドウ関数対応**: OVER句を含むウインドウ関数のパースに対応
6. **複雑なクエリ対応**: WHERE句の複雑な条件、サブクエリ、WITH句を含むクエリのパース
7. **パフォーマンス**: 大きなSQLファイルでも高速に処理

### 非機能要件
- メモリ効率: ストリーミング処理でメモリ使用量を最小化
- 拡張性: 新しいSQL方言やトークンタイプを容易に追加可能
- テスタビリティ: 単体テストが容易な設計
- モダンなGo実装: Go 1.24のイテレータパターンを活用した効率的な実装

## アーキテクチャ

### コンポーネント構成（Phase 1）

```
SqlTokenizer
├── Tokenizer (字句解析)
│   ├── SqlTokenizer
│   ├── TokenType (enum)
│   └── Token (struct)
└── Validator (構文検証)
    ├── BasicSqlValidator
    └── DialectValidator
```

### データフロー

```
SQL文字列
    ↓
Tokenizer (字句解析)
    ↓
Token列
    ↓
BasicSqlValidator (基本構文検証)
    ↓
検証済みToken列 + エラー情報
```

## トークン設計

### 基本トークンタイプ

doma2のSqlTokenizerを参考に以下のトークンタイプを定義：

```go
type TokenType int

const (
    // 基本トークン
    EOF TokenType = iota
    WHITESPACE
    WORD           // 識別子、キーワード
    QUOTE          // 文字列リテラル ('text', "text")
    NUMBER         // 数値リテラル
    OPENED_PARENS  // (
    CLOSED_PARENS  // )
    COMMA          // ,
    SEMICOLON      // ;
    DOT            // .
    
    // SQL演算子
    EQUAL          // =
    NOT_EQUAL      // <>, !=
    LESS_THAN      // <
    GREATER_THAN   // >
    LESS_EQUAL     // <=
    GREATER_EQUAL  // >=
    PLUS           // +
    MINUS          // -
    MULTIPLY       // *
    DIVIDE         // /
    
    // ウインドウ関数関連
    OVER           // OVER キーワード
    PARTITION      // PARTITION キーワード
    ORDER          // ORDER キーワード
    BY             // BY キーワード
    ROWS           // ROWS キーワード
    RANGE          // RANGE キーワード
    UNBOUNDED      // UNBOUNDED キーワード
    PRECEDING      // PRECEDING キーワード
    FOLLOWING      // FOLLOWING キーワード
    CURRENT        // CURRENT キーワード
    ROW            // ROW キーワード
    
    // 論理演算子・条件式関連
    AND            // AND キーワード
    OR             // OR キーワード
    NOT            // NOT キーワード
    IN             // IN キーワード
    EXISTS         // EXISTS キーワード
    BETWEEN        // BETWEEN キーワード
    LIKE           // LIKE キーワード
    IS             // IS キーワード
    NULL           // NULL キーワード
    
    // サブクエリ・CTE関連
    WITH           // WITH キーワード
    AS             // AS キーワード
    SELECT         // SELECT キーワード
    FROM           // FROM キーワード
    WHERE          // WHERE キーワード
    GROUP          // GROUP キーワード
    HAVING         // HAVING キーワード
    UNION          // UNION キーワード
    ALL            // ALL キーワード
    DISTINCT       // DISTINCT キーワード
    
    // コメント
    LINE_COMMENT   // -- 行コメント
    BLOCK_COMMENT  // /* ブロックコメント */ (SnapSQL拡張含む)
    
    // その他
    OTHER          // 複雑な式、データベース固有構文
)
```

### Token構造体

```go
type Token struct {
    Type     TokenType
    Value    string
    Position Position
    
    // SnapSQL拡張情報（Phase 1では解析せず保持のみ）
    IsSnapSQLDirective bool
    DirectiveType      string // "if", "for", "variable" など
}

type Position struct {
    Line   int
    Column int
    Offset int
}
```

### SnapSQL拡張の処理（Phase 1）

Phase 1では、SnapSQLのテンプレート構文は特別なブロックコメントとして認識するのみ：

```sql
-- 通常のブロックコメント
/* これは通常のコメント */

-- SnapSQL拡張（Phase 1では内容は解析しない）
/*# if condition */     → BLOCK_COMMENT (IsSnapSQLDirective=true, DirectiveType="if")
/*# elseif condition */ → BLOCK_COMMENT (IsSnapSQLDirective=true, DirectiveType="elseif")  
/*# else */             → BLOCK_COMMENT (IsSnapSQLDirective=true, DirectiveType="else")
/*# endif */            → BLOCK_COMMENT (IsSnapSQLDirective=true, DirectiveType="endif")
/*# for var : list */   → BLOCK_COMMENT (IsSnapSQLDirective=true, DirectiveType="for")
/*# end */              → BLOCK_COMMENT (IsSnapSQLDirective=true, DirectiveType="end")
/*= variable */         → BLOCK_COMMENT (IsSnapSQLDirective=true, DirectiveType="variable")
```

### Otherトークンの活用

データベース固有の構文や複雑な式は`OTHER`トークンとして処理：

```sql
-- PostgreSQL固有の構文
SELECT array_agg(name ORDER BY id) FROM users;
--     ^^^^^^^^^^^^^^^^^^^^^^^^^ → OTHER トークン

-- MySQL固有の構文  
SELECT name FROM users LIMIT 10 OFFSET 5;
--                     ^^^^^^^^^^^^^^^^^ → OTHER トークン

-- 複雑な式
SELECT CASE WHEN age > 18 THEN 'adult' ELSE 'minor' END FROM users;
--     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ → OTHER トークン

-- 複雑なWHERE句（OR/AND、括弧のネスト）
SELECT * FROM users 
WHERE (age > 18 AND status = 'active') 
   OR (age <= 18 AND parent_consent = true AND status IN ('pending', 'active'));
-- ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
-- 複雑な条件式全体をOTHERトークンとして処理

-- サブクエリを含むクエリ
SELECT u.name, 
       (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) as post_count
--     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ → OTHER（サブクエリ）
FROM users u
WHERE u.id IN (SELECT user_id FROM active_users WHERE last_login > '2024-01-01');
--            ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ → OTHER（サブクエリ）

-- WITH句（CTE）を含むクエリ
WITH active_users AS (
--   ^^^^^^^^^^^^^^^^^ → OTHER（CTE定義）
    SELECT user_id, name FROM users WHERE active = true
), user_stats AS (
-- ^^^^^^^^^^^^^^^ → OTHER（CTE定義）
    SELECT user_id, COUNT(*) as post_count 
    FROM posts 
    GROUP BY user_id
)
SELECT au.name, COALESCE(us.post_count, 0) as posts
FROM active_users au
LEFT JOIN user_stats us ON au.user_id = us.user_id;

-- ウインドウ関数（基本的なOVER句は認識、複雑な部分はOTHER）
SELECT 
    name,
    ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) as rank,
--  ^^^^^^^^^^^^ ^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ 
--  WORD         OVER OTHER（複雑なウインドウ仕様）
    SUM(salary) OVER (ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) as running_total
--  ^^^^^^^^^^^ ^^^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
--  WORD        OVER OTHER（複雑なフレーム仕様）
FROM employees;
```

## トークナイザー実装

### Go 1.24 イテレータ実装

Go 1.24のイテレータパターンを活用して、メモリ効率的で使いやすいトークナイザーを実装：

```go
package sqltokenizer

import (
    "iter"
    "strings"
)

// TokenIterator は Go 1.24 のイテレータパターンを使用
type TokenIterator iter.Seq2[Token, error]

// SqlTokenizer はイテレータを返すトークナイザー
type SqlTokenizer struct {
    input    string
    dialect  SqlDialect
    options  TokenizerOptions
}

type TokenizerOptions struct {
    SkipWhitespace bool
    SkipComments   bool
    PreserveCase   bool
}

func NewSqlTokenizer(input string, dialect SqlDialect, options ...TokenizerOptions) *SqlTokenizer {
    opts := TokenizerOptions{
        SkipWhitespace: false,
        SkipComments:   false,
        PreserveCase:   false,
    }
    if len(options) > 0 {
        opts = options[0]
    }
    
    return &SqlTokenizer{
        input:   input,
        dialect: dialect,
        options: opts,
    }
}

// Tokens はトークンのイテレータを返す
func (t *SqlTokenizer) Tokens() TokenIterator {
    return func(yield func(Token, error) bool) {
        tokenizer := &tokenizer{
            input:    t.input,
            position: 0,
            line:     1,
            column:   1,
            dialect:  t.dialect,
            options:  t.options,
        }
        
        tokenizer.readChar()
        
        for {
            token, err := tokenizer.nextToken()
            if err != nil {
                if !yield(Token{}, err) {
                    return
                }
                continue
            }
            
            if token.Type == EOF {
                yield(token, nil)
                return
            }
            
            // オプションに基づくフィルタリング
            if t.options.SkipWhitespace && token.Type == WHITESPACE {
                continue
            }
            if t.options.SkipComments && (token.Type == LINE_COMMENT || token.Type == BLOCK_COMMENT) {
                continue
            }
            
            if !yield(token, nil) {
                return
            }
        }
    }
}

// AllTokens は全てのトークンをスライスとして取得（デバッグ用）
func (t *SqlTokenizer) AllTokens() ([]Token, error) {
    var tokens []Token
    var lastError error
    
    for token, err := range t.Tokens() {
        if err != nil {
            lastError = err
            continue
        }
        tokens = append(tokens, token)
        if token.Type == EOF {
            break
        }
    }
    
    return tokens, lastError
}
```

### 内部実装構造

```go
// 内部のトークナイザー実装
type tokenizer struct {
    input    string
    position int
    line     int
    column   int
    current  rune
    dialect  SqlDialect
    options  TokenizerOptions
}

func (t *tokenizer) nextToken() (Token, error) {
    // 実装詳細
    for {
        switch t.current {
        case 0:
            return t.newToken(EOF, ""), nil
        case ' ', '\t', '\r', '\n':
            return t.readWhitespace(), nil
        case '(':
            token := t.newToken(OPENED_PARENS, string(t.current))
            t.readChar()
            return token, nil
        case ')':
            token := t.newToken(CLOSED_PARENS, string(t.current))
            t.readChar()
            return token, nil
        case ',':
            token := t.newToken(COMMA, string(t.current))
            t.readChar()
            return token, nil
        case ';':
            token := t.newToken(SEMICOLON, string(t.current))
            t.readChar()
            return token, nil
        case '.':
            token := t.newToken(DOT, string(t.current))
            t.readChar()
            return token, nil
        case '\'', '"':
            return t.readString(t.current)
        case '-':
            if t.peekChar() == '-' {
                return t.readLineComment()
            }
            token := t.newToken(MINUS, string(t.current))
            t.readChar()
            return token, nil
        case '/':
            if t.peekChar() == '*' {
                return t.readBlockComment()
            }
            token := t.newToken(DIVIDE, string(t.current))
            t.readChar()
            return token, nil
        case '=':
            token := t.newToken(EQUAL, string(t.current))
            t.readChar()
            return token, nil
        case '<':
            if t.peekChar() == '=' {
                t.readChar()
                t.readChar()
                return t.newToken(LESS_EQUAL, "<="), nil
            } else if t.peekChar() == '>' {
                t.readChar()
                t.readChar()
                return t.newToken(NOT_EQUAL, "<>"), nil
            }
            token := t.newToken(LESS_THAN, string(t.current))
            t.readChar()
            return token, nil
        case '>':
            if t.peekChar() == '=' {
                t.readChar()
                t.readChar()
                return t.newToken(GREATER_EQUAL, ">="), nil
            }
            token := t.newToken(GREATER_THAN, string(t.current))
            t.readChar()
            return token, nil
        case '!':
            if t.peekChar() == '=' {
                t.readChar()
                t.readChar()
                return t.newToken(NOT_EQUAL, "!="), nil
            }
            // '!' 単体は OTHER として処理
            return t.readOther()
        case '+':
            token := t.newToken(PLUS, string(t.current))
            t.readChar()
            return token, nil
        case '*':
            token := t.newToken(MULTIPLY, string(t.current))
            t.readChar()
            return token, nil
        default:
            if isLetter(t.current) || t.current == '_' {
                return t.readWord()
            } else if isDigit(t.current) {
                return t.readNumber()
            } else {
                // その他の文字は OTHER として処理
                return t.readOther()
            }
        }
    }
}
```

### 主要メソッド

```go
// 文字読み取り
func (t *tokenizer) readChar()
func (t *tokenizer) peekChar() rune
func (t *tokenizer) skipWhitespace()

// トークン認識
func (t *tokenizer) readWord() (Token, error)
func (t *tokenizer) readString(delimiter rune) (Token, error)
func (t *tokenizer) readNumber() (Token, error)
func (t *tokenizer) readLineComment() (Token, error)
func (t *tokenizer) readBlockComment() (Token, error)
func (t *tokenizer) readWhitespace() Token
func (t *tokenizer) readOther() (Token, error)

// SnapSQL拡張
func (t *tokenizer) parseSnapSQLDirective(comment string) (string, bool)

// ウインドウ関数関連
func (t *tokenizer) isWindowKeyword(word string) bool

// 論理演算子・条件式関連
func (t *tokenizer) isLogicalOperator(word string) bool

// サブクエリ・CTE関連
func (t *tokenizer) isSqlKeyword(word string) bool

// ユーティリティ
func (t *tokenizer) newToken(tokenType TokenType, value string) Token
func isLetter(ch rune) bool
func isDigit(ch rune) bool
func isAlphaNumeric(ch rune) bool
```

### SnapSQL拡張の認識

```go
func (t *SqlTokenizer) parseSnapSQLDirective(comment string) (directiveType string, isDirective bool) {
    trimmed := strings.TrimSpace(comment)
    
    // /*# で始まる場合
    if strings.HasPrefix(trimmed, "/*#") && strings.HasSuffix(trimmed, "*/") {
        content := strings.TrimSpace(trimmed[3 : len(trimmed)-2])
        
        if strings.HasPrefix(content, "if ") {
            return "if", true
        } else if strings.HasPrefix(content, "elseif ") {
            return "elseif", true
        } else if content == "else" {
            return "else", true
        } else if content == "endif" {
            return "endif", true
        } else if strings.HasPrefix(content, "for ") {
            return "for", true
        } else if content == "end" {
            return "end", true
        }
    }
    
    // /*= で始まる場合
    if strings.HasPrefix(trimmed, "/*=") && strings.HasSuffix(trimmed, "*/") {
        return "variable", true
    }
    
    return "", false
}

// ウインドウ関数キーワードの判定
func (t *SqlTokenizer) isWindowKeyword(word string) bool {
    windowKeywords := map[string]bool{
        "OVER":       true,
        "PARTITION":  true,
        "ORDER":      true,
        "BY":         true,
        "ROWS":       true,
        "RANGE":      true,
        "UNBOUNDED":  true,
        "PRECEDING":  true,
        "FOLLOWING":  true,
        "CURRENT":    true,
        "ROW":        true,
        "BETWEEN":    true,
        "AND":        true,
    }
    
    return windowKeywords[strings.ToUpper(word)]
}

// 論理演算子の判定
func (t *SqlTokenizer) isLogicalOperator(word string) bool {
    logicalOperators := map[string]bool{
        "AND":      true,
        "OR":       true,
        "NOT":      true,
        "IN":       true,
        "EXISTS":   true,
        "BETWEEN":  true,
        "LIKE":     true,
        "IS":       true,
        "NULL":     true,
    }
    
    return logicalOperators[strings.ToUpper(word)]
}

// SQLキーワードの判定
func (t *SqlTokenizer) isSqlKeyword(word string) bool {
    sqlKeywords := map[string]bool{
        "WITH":     true,
        "AS":       true,
        "SELECT":   true,
        "FROM":     true,
        "WHERE":    true,
        "GROUP":    true,
        "HAVING":   true,
        "ORDER":    true,
        "BY":       true,
        "UNION":    true,
        "ALL":      true,
        "DISTINCT": true,
        "INSERT":   true,
        "UPDATE":   true,
        "DELETE":   true,
        "CREATE":   true,
        "DROP":     true,
        "ALTER":    true,
        "INDEX":    true,
        "TABLE":    true,
        "VIEW":     true,
        "TRIGGER":  true,
        "FUNCTION": true,
        "PROCEDURE": true,
    }
    
    return sqlKeywords[strings.ToUpper(word)]
}
```

## SQL構文検証

### 基本構文検証

Phase 1では基本的なSQL構文の検証を行う：

```go
type BasicSqlValidator struct {
    tokens []Token
    pos    int
}

func (v *BasicSqlValidator) Validate() []ValidationError {
    var errors []ValidationError
    
    // 基本的な構文チェック
    errors = append(errors, v.validateParentheses()...)
    errors = append(errors, v.validateQuotes()...)
    errors = append(errors, v.validateBasicStructure()...)
    errors = append(errors, v.validateWindowFunctions()...)
    errors = append(errors, v.validateComplexConditions()...)
    errors = append(errors, v.validateSubqueries()...)
    errors = append(errors, v.validateCTEs()...)
    
    return errors
}

type ValidationError struct {
    Message  string
    Position Position
    Severity ErrorSeverity
}

type ErrorSeverity int

const (
    WARNING ErrorSeverity = iota
    ERROR
    FATAL
)
```

### 検証項目

1. **括弧の対応**: `(` と `)` の対応関係
2. **引用符の対応**: `'` と `"` の対応関係  
3. **基本構造**: SELECT、FROM、WHERE等の基本的な順序
4. **SnapSQL構文**: if/endif、for/endの対応関係
5. **ウインドウ関数**: OVER句の基本的な構文チェック
6. **複雑な条件式**: WHERE句のOR/AND、括弧のネストチェック
7. **サブクエリ**: FROM句、WHERE句のサブクエリの基本構文チェック
8. **CTE**: WITH句の基本構文チェック

## データベース方言対応

### 方言インターフェース

```go
type SqlDialect interface {
    Name() string
    IsKeyword(word string) bool
    IsReservedWord(word string) bool
    QuoteIdentifier(identifier string) string
    QuoteLiteral(literal string) string
    
    // Phase 1では基本的な識別のみ
    RecognizeSpecialSyntax(tokenizer *SqlTokenizer) (Token, bool)
}

type PostgreSQLDialect struct {
    keywords     map[string]bool
    reservedWords map[string]bool
}

type MySQLDialect struct {
    keywords     map[string]bool
    reservedWords map[string]bool
}

type SQLiteDialect struct {
    keywords     map[string]bool
    reservedWords map[string]bool
}
```

### 方言の自動検出

```go
func DetectDialect(sql string) SqlDialect {
    // SQL内のキーワードや構文から方言を推定
    upperSQL := strings.ToUpper(sql)
    
    // PostgreSQL特有
    if strings.Contains(upperSQL, "RETURNING") ||
       strings.Contains(upperSQL, "ARRAY_AGG") ||
       strings.Contains(upperSQL, "::") { // キャスト演算子
        return NewPostgreSQLDialect()
    }
    
    // MySQL特有
    if strings.Contains(upperSQL, "LIMIT") && strings.Contains(upperSQL, "OFFSET") ||
       strings.Contains(upperSQL, "`") { // バッククォート
        return NewMySQLDialect()
    }
    
    // デフォルトはSQLite
    return NewSQLiteDialect()
}
```

## エラー処理

### エラー情報の詳細化

```go
type TokenizeError struct {
    Message  string
    Position Position
    Context  string
    Severity ErrorSeverity
}

func (e *TokenizeError) Error() string {
    return fmt.Sprintf("[%s] %s at line %d, column %d: %s", 
        e.Severity, e.Message, e.Position.Line, e.Position.Column, e.Context)
}
```

### エラーリカバリ戦略

1. **文字レベル**: 不正な文字は無視して継続
2. **トークンレベル**: 不正なトークンはOTHERとして処理
3. **構文レベル**: 基本構文エラーは警告として報告

## パフォーマンス最適化

### Go 1.24 イテレータの利点

1. **メモリ効率**: 全トークンを一度にメモリに保持する必要がない
2. **遅延評価**: 必要な分だけトークンを生成
3. **早期終了**: エラー時や条件満足時に処理を中断可能
4. **関数型プログラミング**: range-over-func パターンによる直感的な処理

### メモリ効率の実装

```go
// オブジェクトプール（必要に応じて）
var tokenPool = sync.Pool{
    New: func() interface{} {
        return &Token{}
    },
}

func (t *tokenizer) newToken(tokenType TokenType, value string) Token {
    // 大きなSQLファイルの場合のみプールを使用
    if len(t.input) > 10000 {
        token := tokenPool.Get().(*Token)
        token.Type = tokenType
        token.Value = value
        token.Position = Position{
            Line:   t.line,
            Column: t.column,
            Offset: t.position,
        }
        token.IsSnapSQLDirective = false
        token.DirectiveType = ""
        return *token
    }
    
    // 通常は直接作成
    return Token{
        Type:  tokenType,
        Value: value,
        Position: Position{
            Line:   t.line,
            Column: t.column,
            Offset: t.position,
        },
    }
}

// トークンをプールに戻す（必要に応じて）
func (t *Token) Release() {
    if len(t.Value) > 100 { // 大きなトークンのみプールに戻す
        tokenPool.Put(t)
    }
}
```

### 処理速度の最適化

1. **バッファリング**: 文字読み取りのバッファリング
2. **先読み最小化**: 必要最小限のlookahead
3. **文字列操作最適化**: strings.Builderの活用
4. **イテレータの効率的な実装**: 不要なアロケーションを避ける

```go
// 効率的な文字列構築
func (t *tokenizer) readWord() (Token, error) {
    var builder strings.Builder
    builder.Grow(32) // 一般的な識別子長を想定
    
    for isAlphaNumeric(t.current) || t.current == '_' {
        builder.WriteRune(t.current)
        t.readChar()
    }
    
    word := builder.String()
    
    // キーワード判定
    if t.isSqlKeyword(word) {
        return t.newToken(getKeywordTokenType(word), word), nil
    }
    
    return t.newToken(WORD, word), nil
}
```

## テスト戦略

### 単体テスト

doma2のテストケースを参考に、以下のパターンをカバー：

```go
func TestTokenIterator(t *testing.T) {
    sql := "SELECT id, name FROM users WHERE active = true;"
    tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect())
    
    expectedTypes := []TokenType{
        WORD, WHITESPACE, WORD, COMMA, WHITESPACE, WORD, WHITESPACE,
        WORD, WHITESPACE, WORD, WHITESPACE, WORD, WHITESPACE, EQUAL,
        WHITESPACE, WORD, SEMICOLON, EOF,
    }
    
    var actualTypes []TokenType
    for token, err := range tokenizer.Tokens() {
        if err != nil {
            t.Errorf("Unexpected error: %v", err)
            continue
        }
        
        actualTypes = append(actualTypes, token.Type)
        
        if token.Type == EOF {
            break
        }
    }
    
    if !reflect.DeepEqual(expectedTypes, actualTypes) {
        t.Errorf("Expected %v, got %v", expectedTypes, actualTypes)
    }
}

func TestTokenIteratorWithOptions(t *testing.T) {
    sql := "SELECT id, name FROM users -- comment\nWHERE active = true;"
    tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect(), TokenizerOptions{
        SkipWhitespace: true,
        SkipComments:   true,
    })
    
    expectedTypes := []TokenType{
        WORD, WORD, COMMA, WORD, WORD, WORD, WORD, EQUAL, WORD, SEMICOLON, EOF,
    }
    
    var actualTypes []TokenType
    for token, err := range tokenizer.Tokens() {
        if err != nil {
            t.Errorf("Unexpected error: %v", err)
            continue
        }
        
        actualTypes = append(actualTypes, token.Type)
        
        if token.Type == EOF {
            break
        }
    }
    
    if !reflect.DeepEqual(expectedTypes, actualTypes) {
        t.Errorf("Expected %v, got %v", expectedTypes, actualTypes)
    }
}

func TestIteratorEarlyTermination(t *testing.T) {
    sql := "SELECT id, name FROM users WHERE active = true;"
    tokenizer := NewSqlTokenizer(sql, NewSQLiteDialect())
    
    count := 0
    for token, err := range tokenizer.Tokens() {
        if err != nil {
            t.Errorf("Unexpected error: %v", err)
            continue
        }
        
        count++
        
        // 5つ目のトークンで終了
        if count >= 5 {
            break
        }
    }
    
    if count != 5 {
        t.Errorf("Expected to process 5 tokens, got %d", count)
    }
}

func TestSnapSQLDirectives(t *testing.T) {
    tests := []struct {
        input          string
        expectedType   TokenType
        isDirective    bool
        directiveType  string
    }{
        {"/*# if condition */", BLOCK_COMMENT, true, "if"},
        {"/*= variable */", BLOCK_COMMENT, true, "variable"},
        {"/* normal comment */", BLOCK_COMMENT, false, ""},
    }
    
func TestWindowFunctions(t *testing.T) {
    tests := []struct {
        input       string
        description string
        expectError bool
    }{
        {
            "SELECT ROW_NUMBER() OVER (ORDER BY id) FROM users",
            "基本的なウインドウ関数",
            false,
        },
        {
            "SELECT SUM(salary) OVER (PARTITION BY dept ORDER BY salary ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM emp",
            "複雑なウインドウ関数",
            false,
        },
        {
            "SELECT ROW_NUMBER() OVER FROM users",
            "不完全なOVER句",
            true,
        },
        {
            "SELECT SUM(salary) OVER (PARTITION BY FROM emp",
            "不完全なPARTITION BY句",
            true,
        },
    }
    
    // テスト実装
}

func TestComplexConditions(t *testing.T) {
    tests := []struct {
        input       string
        description string
        expectError bool
    }{
        {
            "SELECT * FROM users WHERE (age > 18 AND status = 'active') OR (vip = true)",
            "複雑なWHERE句（OR/AND、括弧のネスト）",
            false,
        },
        {
            "SELECT * FROM users WHERE age > 18 AND status IN ('active', 'pending')",
            "IN句を含む条件",
            false,
        },
        {
            "SELECT * FROM users WHERE age > 18 AND OR status = 'active'",
            "不正な論理演算子の組み合わせ",
            true,
        },
        {
            "SELECT * FROM users WHERE (age > 18 AND status = 'active' OR",
            "不完全な条件式",
            true,
        },
    }
    
    // テスト実装
}

func TestSubqueries(t *testing.T) {
    tests := []struct {
        input       string
        description string
        expectError bool
    }{
        {
            "SELECT * FROM users WHERE id IN (SELECT user_id FROM active_users)",
            "WHERE句のサブクエリ",
            false,
        },
        {
            "SELECT u.name, (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) FROM users u",
            "SELECT句のサブクエリ",
            false,
        },
        {
            "SELECT * FROM (SELECT * FROM users WHERE active = true) AS active_users",
            "FROM句のサブクエリ",
            false,
        },
        {
            "SELECT * FROM (SELECT * FROM users WHERE",
            "不完全なサブクエリ",
            true,
        },
    }
    
    // テスト実装
}

func TestCTEs(t *testing.T) {
    tests := []struct {
        input       string
        description string
        expectError bool
    }{
        {
            "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users",
            "基本的なCTE",
            false,
        },
        {
            "WITH RECURSIVE emp_hierarchy AS (SELECT * FROM employees UNION ALL SELECT * FROM emp_hierarchy) SELECT * FROM emp_hierarchy",
            "再帰CTE",
            false,
        },
        {
            "WITH users AS (SELECT * FROM employees), stats AS (SELECT COUNT(*) FROM users) SELECT * FROM stats",
            "複数のCTE",
            false,
        },
        {
            "WITH users AS (SELECT * FROM",
            "不完全なCTE定義",
            true,
        },
    }
    
    // テスト実装
}
```

### テストデータ

```sql
-- 基本的なSELECT
SELECT id, name FROM users WHERE active = true;

-- 複雑なJOIN
SELECT u.name, p.title 
FROM users u 
LEFT JOIN posts p ON u.id = p.user_id 
WHERE u.active = true;

-- 複雑なWHERE句（OR/AND、括弧のネスト）
SELECT * FROM users 
WHERE (age > 18 AND status = 'active') 
   OR (age <= 18 AND parent_consent = true AND status IN ('pending', 'active'))
   OR (vip_status = true AND (subscription_type = 'premium' OR subscription_type = 'enterprise'));

-- サブクエリを含むクエリ
SELECT u.name, 
       (SELECT COUNT(*) FROM posts p WHERE p.user_id = u.id) as post_count,
       (SELECT MAX(created_at) FROM posts p WHERE p.user_id = u.id) as last_post_date
FROM users u
WHERE u.id IN (
    SELECT user_id 
    FROM active_users 
    WHERE last_login > '2024-01-01'
      AND status IN (
          SELECT status_name 
          FROM valid_statuses 
          WHERE is_active = true
      )
);

-- WITH句（CTE）を含む複雑なクエリ
WITH RECURSIVE employee_hierarchy AS (
    -- アンカー部分
    SELECT employee_id, name, manager_id, 0 as level
    FROM employees
    WHERE manager_id IS NULL
    
    UNION ALL
    
    -- 再帰部分
    SELECT e.employee_id, e.name, e.manager_id, eh.level + 1
    FROM employees e
    INNER JOIN employee_hierarchy eh ON e.manager_id = eh.employee_id
),
department_stats AS (
    SELECT 
        department_id,
        COUNT(*) as employee_count,
        AVG(salary) as avg_salary,
        MAX(salary) as max_salary
    FROM employees
    WHERE active = true
    GROUP BY department_id
    HAVING COUNT(*) > 5
),
top_performers AS (
    SELECT 
        employee_id,
        name,
        department_id,
        salary,
        ROW_NUMBER() OVER (PARTITION BY department_id ORDER BY salary DESC) as dept_rank
    FROM employees
    WHERE performance_rating >= 4.0
)
SELECT 
    eh.name,
    eh.level,
    ds.employee_count,
    ds.avg_salary,
    tp.dept_rank,
    CASE 
        WHEN tp.dept_rank <= 3 THEN 'Top Performer'
        WHEN eh.level = 0 THEN 'Executive'
        ELSE 'Regular'
    END as employee_category
FROM employee_hierarchy eh
LEFT JOIN department_stats ds ON eh.department_id = ds.department_id
LEFT JOIN top_performers tp ON eh.employee_id = tp.employee_id
WHERE eh.level <= 5
ORDER BY eh.level, eh.name;

-- ウインドウ関数を含むSQL
SELECT 
    employee_id,
    name,
    department,
    salary,
    ROW_NUMBER() OVER (PARTITION BY department ORDER BY salary DESC) as dept_rank,
    RANK() OVER (ORDER BY salary DESC) as overall_rank,
    SUM(salary) OVER (PARTITION BY department ORDER BY salary ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) as running_total,
    LAG(salary, 1) OVER (PARTITION BY department ORDER BY salary) as prev_salary,
    LEAD(salary, 1) OVER (PARTITION BY department ORDER BY salary) as next_salary
FROM employees
WHERE active = true;

-- SnapSQL拡張（Phase 1では構文認識のみ）
SELECT 
    id,
    name
    /*# if include_email */,
    email
    /*# endif */
FROM users_/*= table_suffix */
/*# if filters.active */
WHERE active = /*= filters.active */
/*# endif */;

-- SnapSQL拡張とウインドウ関数の組み合わせ
SELECT 
    id,
    name
    /*# if include_analytics */,
    ROW_NUMBER() OVER (ORDER BY /*= sort_field */) as row_num,
    PERCENT_RANK() OVER (PARTITION BY department ORDER BY salary) as percentile
    /*# endif */
FROM employees_/*= table_suffix */
/*# if filters.department */
WHERE department = /*= filters.department */
/*# endif */;

-- SnapSQL拡張と複雑なクエリの組み合わせ
/*# if use_cte */
WITH filtered_users AS (
    SELECT * FROM users_/*= table_suffix */
    /*# if filters.active */
    WHERE active = /*= filters.active */
    /*# endif */
    /*# if filters.department */
    AND department IN (/*# for dept : departments *//*= dept *//*# if !@last */,/*# endif *//*# end */)
    /*# endif */
)
/*# endif */
SELECT 
    /*# if use_cte */fu.id, fu.name/*# else */u.id, u.name/*# endif */
    /*# if include_stats */,
    (SELECT COUNT(*) FROM posts p WHERE p.user_id = /*# if use_cte */fu.id/*# else */u.id/*# endif */) as post_count
    /*# endif */
FROM /*# if use_cte */filtered_users fu/*# else */users_/*= table_suffix */ u/*# endif */
/*# if complex_where */
WHERE (
    /*# if age_filter */age > /*= min_age *//*# endif */
    /*# if age_filter && status_filter */ AND /*# endif */
    /*# if status_filter */status IN (/*# for status : valid_statuses *//*= status *//*# if !@last */,/*# endif *//*# end */)/*# endif */
)
/*# endif */;

-- データベース固有構文
-- PostgreSQL
SELECT ARRAY_AGG(name) FROM users;
-- MySQL  
SELECT name FROM users LIMIT 10 OFFSET 5;
-- SQLite
SELECT name FROM users LIMIT 5, 10;

-- エラーケース
SELECT id, name FROM users WHERE; -- 不完全なWHERE句
SELECT id, name FROM users WHERE id = 'unclosed string; -- 閉じられていない文字列
SELECT id, name FROM users WHERE (id = 1; -- 閉じられていない括弧
SELECT ROW_NUMBER() OVER FROM users; -- 不完全なOVER句
SELECT SUM(salary) OVER (PARTITION BY; -- 不完全なPARTITION BY句
SELECT * FROM users WHERE age > 18 AND OR status = 'active'; -- 不正な論理演算子の組み合わせ
SELECT * FROM users WHERE age > 18 AND (status = 'active' OR; -- 不完全な条件式
SELECT * FROM (SELECT * FROM users WHERE; -- 不完全なサブクエリ
WITH users AS (SELECT * FROM; -- 不完全なCTE定義
WITH users AS (SELECT * FROM employees) SELECT * FROM users_table; -- CTE名の重複
```
```

## 実装計画（Phase 1）

### Step 1: 基本トークナイザー
- [ ] Go 1.24 イテレータインターフェースの実装
- [ ] 基本トークンタイプの実装
- [ ] 文字列、数値、識別子の認識
- [ ] 演算子の認識
- [ ] TokenizerOptions の実装

### Step 2: コメント処理
- [ ] 行コメント（`--`）の処理
- [ ] ブロックコメント（`/* */`）の処理
- [ ] SnapSQL拡張の認識（解析はしない）

### Step 3: 基本構文検証
- [ ] 括弧の対応チェック
- [ ] 引用符の対応チェック
- [ ] SnapSQL構文の対応チェック
- [ ] ウインドウ関数の基本構文チェック
- [ ] 複雑な条件式の構文チェック
- [ ] サブクエリの基本構文チェック
- [ ] CTE（WITH句）の基本構文チェック

### Step 4: データベース方言
- [ ] 方言インターフェースの実装
- [ ] PostgreSQL、MySQL、SQLite方言の基本実装
- [ ] 方言自動検出

### Step 5: テストとエラー処理
- [ ] 包括的なテストスイート
- [ ] エラー処理の実装
- [ ] パフォーマンス測定

## 参考資料

- [doma2 SqlTokenizer](https://github.com/domaframework/doma/blob/master/doma-core/src/main/java/org/seasar/doma/internal/jdbc/sql/SqlTokenizer.java)
- [doma2 SqlTokenizerTest](https://github.com/domaframework/doma/blob/master/doma-core/src/test/java/org/seasar/doma/internal/jdbc/sql/SqlTokenizerTest.java)
- [PostgreSQL SQL Syntax](https://www.postgresql.org/docs/current/sql.html)
- [MySQL SQL Syntax](https://dev.mysql.com/doc/refman/8.0/en/sql-syntax.html)
- [SQLite SQL Syntax](https://www.sqlite.org/lang.html)

---

## Phase 2: パーサー設計（構文解析とランタイム最適化）

### 概要

Phase 1のトークナイザーで生成されたトークン列を構文解析し、SnapSQLテンプレートの実行に最適化された中間表現（AST）を生成する。

### 設計目標

1. **ランタイム最適化**: 実行時の処理を軽量化するためのヒント情報を中間表現に埋め込む
2. **自動削除ルールの実装**: WHERE句、ORDER BY句、LIMIT句等の条件付き削除
3. **配列展開の最適化**: IN句での配列変数の効率的な展開
4. **ダミーリテラルの処理**: 開発時ダミー値と実行時値の適切な置換

## ランタイム最適化ヒントの挿入

### 自動条件分岐の生成

パーサーは、ソースコードに明示的に記述されていない条件分岐を、ランタイム最適化のために中間表現に自動挿入する。

#### 例1: ORDER BY句の自動条件分岐

**ソースコード（簡潔な記述）**:
```sql
ORDER BY /*# for sort : sort_fields *//*= sort.field */ /*= sort.direction *//*# end */name ASC
```

**中間表現（最適化ヒント付き）**:
```json
{
  "type": "conditional_clause",
  "clause_type": "ORDER_BY",
  "condition": {
    "type": "auto_generated",
    "variable": "sort_fields",
    "check": "not_null_and_not_empty"
  },
  "content": {
    "type": "order_by_clause",
    "dynamic_fields": {
      "type": "for_loop",
      "variable": "sort",
      "list": "sort_fields",
      "template": "/*= sort.field */ /*= sort.direction */"
    },
    "fallback": "name ASC"
  }
}
```

#### 例2: WHERE句の自動条件分岐

**ソースコード**:
```sql
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales', 'marketing')
```

**中間表現（最適化ヒント付き）**:
```json
{
  "type": "conditional_clause", 
  "clause_type": "WHERE",
  "conditions": [
    {
      "type": "condition",
      "operator": "WHERE",
      "expression": "active = /*= filters.active */",
      "auto_remove": {
        "condition": "filters.active == null"
      }
    },
    {
      "type": "condition",
      "operator": "AND", 
      "expression": "department IN (/*= filters.departments */)",
      "auto_remove": {
        "condition": "filters.departments == null || filters.departments.length == 0"
      }
    }
  ],
  "auto_remove_entire_clause": {
    "condition": "all_conditions_removed"
  }
}
```

### 自動挿入されるヒントの種類

#### 1. **句レベルの条件分岐**
- `ORDER BY` 句: `sort_fields` の存在チェック
- `WHERE` 句: 全条件の有効性チェック  
- `LIMIT` 句: `limit` の有効値チェック
- `OFFSET` 句: `offset` の有効値チェック

#### 2. **条件レベルの分岐**
- `AND/OR` 条件: 個別変数の null/空チェック
- `IN` 句: 配列変数の存在・非空チェック
- 変数置換: null値での句削除チェック

#### 3. **配列展開の最適化**
```json
{
  "type": "array_expansion",
  "variable": "filters.departments", 
  "expansion_type": "IN_clause",
  "quote_elements": true,
  "separator": ", ",
  "auto_remove_if_empty": true
}
```

### パーサーの処理フロー

```
トークン列
    ↓
構文解析
    ↓
基本AST生成
    ↓
自動条件分岐の挿入 ← 【新規追加処理】
    ↓
ランタイム最適化ヒント付きAST
    ↓
中間表現出力
```

### 最適化ヒント挿入のルール

#### 1. **暗黙的条件分岐の検出**
- `/*# for variable : list */` → `list` の存在チェック条件を自動生成
- `/*= variable */` → `variable` の null チェック条件を自動生成
- SQL句の構造 → 句全体の削除条件を自動生成

#### 2. **ヒント情報の種類**
```typescript
interface RuntimeHint {
  type: 'auto_condition' | 'array_expansion' | 'clause_removal';
  target: string;           // 対象変数名
  condition: string;        // 削除・実行条件
  fallback?: string;        // フォールバック値
  optimization_level: 'clause' | 'condition' | 'variable';
}
```

#### 3. **ランタイムでの活用**
- **事前チェック**: 変数値の事前評価で不要な処理をスキップ
- **効率的な文字列構築**: 条件に応じた最適なSQL構築パスの選択
- **メモリ効率**: 不要な中間文字列の生成回避

### 実装上の考慮事項

#### 1. **パフォーマンス**
- ヒント生成のオーバーヘッドを最小化
- ランタイムでの条件評価コストの削減
- 中間表現のサイズ最適化

#### 2. **保守性**
- 自動生成ルールの明確な定義
- デバッグ情報の保持
- ソースコードとの対応関係の維持

#### 3. **拡張性**
- 新しい最適化パターンの追加容易性
- カスタム条件分岐の対応
- 複雑なSQL構造への対応

### テンプレート記述の簡素化効果

#### **Before（明示的条件分岐）**
```sql
/*# if filters.active */
WHERE active = /*= filters.active */
/*# end */
/*# if filters.departments */
AND department IN (/*= filters.departments */)
/*# end */
/*# if sort_fields */
ORDER BY /*# for sort : sort_fields */.../*# end */
/*# end */
```

#### **After（自動削除ルール + 最適化ヒント）**
```sql
WHERE active = /*= filters.active */true
    AND department IN (/*= filters.departments */'sales', 'marketing')
ORDER BY /*# for sort : sort_fields *//*= sort.field */ /*= sort.direction *//*# end */name ASC
```

この設計により、シンプルなテンプレート記述でありながら、ランタイムでの高効率な処理が実現できます。パーサーが自動的に最適化ヒントを挿入することで、開発者は複雑な条件分岐を意識せずに済み、かつランタイムでの処理性能も向上します。

---

## Phase 2: パーサー実装詳細

### AST（抽象構文木）設計

#### ASTノード階層

```go
type AstNode interface {
    Type() NodeType
    Children() []AstNode
    Position() Position
    String() string
    Accept(visitor AstVisitor) error
}

type NodeType int

const (
    // SQL文構造
    SELECT_STATEMENT NodeType = iota
    INSERT_STATEMENT
    UPDATE_STATEMENT
    DELETE_STATEMENT
    
    // SQL句
    SELECT_CLAUSE
    FROM_CLAUSE
    WHERE_CLAUSE
    ORDER_BY_CLAUSE
    GROUP_BY_CLAUSE
    HAVING_CLAUSE
    LIMIT_CLAUSE
    OFFSET_CLAUSE
    
    // SnapSQL拡張
    TEMPLATE_IF_BLOCK
    TEMPLATE_FOR_BLOCK
    VARIABLE_SUBSTITUTION
    CONDITIONAL_CLAUSE    // 自動生成された条件分岐
    
    // 式・リテラル
    IDENTIFIER
    LITERAL
    EXPRESSION
    OTHER_EXPRESSION
    
    // 特殊ノード
    RUNTIME_HINT         // ランタイム最適化ヒント
    AUTO_CONDITION       // 自動生成条件
)
```

#### 具体的なASTノード実装

```go
// 基本ASTノード
type BaseAstNode struct {
    nodeType NodeType
    position Position
    children []AstNode
}

func (n *BaseAstNode) Type() NodeType { return n.nodeType }
func (n *BaseAstNode) Position() Position { return n.position }
func (n *BaseAstNode) Children() []AstNode { return n.children }

// SELECT文ノード
type SelectStatement struct {
    BaseAstNode
    SelectClause *SelectClause
    FromClause   *FromClause
    WhereClause  *WhereClause
    OrderByClause *OrderByClause
    LimitClause  *LimitClause
    OffsetClause *OffsetClause
}

// 条件付き句ノード（自動生成）
type ConditionalClause struct {
    BaseAstNode
    ClauseType   string           // "WHERE", "ORDER_BY", "LIMIT", "OFFSET"
    Condition    *AutoCondition   // 自動生成された条件
    Content      AstNode          // 実際の句内容
    Fallback     AstNode          // フォールバック内容
}

// 自動生成条件
type AutoCondition struct {
    BaseAstNode
    Variable     string    // 対象変数名
    CheckType    string    // "not_null", "not_empty", "not_null_and_not_empty"
    Expression   string    // 条件式
}

// SnapSQL変数置換
type VariableSubstitution struct {
    BaseAstNode
    Variable     string    // 変数名
    DummyValue   string    // ダミーリテラル
    AutoRemove   *AutoCondition // 自動削除条件
}

// SnapSQLループ
type TemplateForBlock struct {
    BaseAstNode
    Variable     string    // ループ変数名
    List         string    // リスト変数名
    Template     AstNode   // ループ内容
    Separator    string    // 区切り文字（自動判定）
    AutoCondition *AutoCondition // 自動生成条件
}

// ランタイム最適化ヒント
type RuntimeHint struct {
    BaseAstNode
    HintType     string    // "auto_condition", "array_expansion", "clause_removal"
    Target       string    // 対象変数名
    Condition    string    // 削除・実行条件
    Fallback     string    // フォールバック値
    OptLevel     string    // "clause", "condition", "variable"
}
```

### パーサー実装構造

#### 再帰下降パーサー

```go
type SqlParser struct {
    tokens    []Token
    current   int
    errors    []ParseError
    hints     []RuntimeHint
}

func NewSqlParser(tokens []Token) *SqlParser {
    return &SqlParser{
        tokens:  tokens,
        current: 0,
        errors:  make([]ParseError, 0),
        hints:   make([]RuntimeHint, 0),
    }
}

// メインパース関数
func (p *SqlParser) Parse() (*SelectStatement, error) {
    stmt, err := p.parseSelectStatement()
    if err != nil {
        return nil, err
    }
    
    // 自動最適化ヒントの挿入
    p.insertOptimizationHints(stmt)
    
    return stmt, nil
}

// SELECT文のパース
func (p *SqlParser) parseSelectStatement() (*SelectStatement, error) {
    stmt := &SelectStatement{}
    
    // SELECT句
    selectClause, err := p.parseSelectClause()
    if err != nil {
        return nil, err
    }
    stmt.SelectClause = selectClause
    
    // FROM句
    if p.match(FROM) {
        fromClause, err := p.parseFromClause()
        if err != nil {
            return nil, err
        }
        stmt.FromClause = fromClause
    }
    
    // WHERE句
    if p.match(WHERE) {
        whereClause, err := p.parseWhereClause()
        if err != nil {
            return nil, err
        }
        stmt.WhereClause = whereClause
    }
    
    // ORDER BY句
    if p.match(ORDER) {
        orderByClause, err := p.parseOrderByClause()
        if err != nil {
            return nil, err
        }
        stmt.OrderByClause = orderByClause
    }
    
    // LIMIT句
    if p.matchWord("LIMIT") {
        limitClause, err := p.parseLimitClause()
        if err != nil {
            return nil, err
        }
        stmt.LimitClause = limitClause
    }
    
    // OFFSET句
    if p.matchWord("OFFSET") {
        offsetClause, err := p.parseOffsetClause()
        if err != nil {
            return nil, err
        }
        stmt.OffsetClause = offsetClause
    }
    
    return stmt, nil
}
```

#### SnapSQL拡張のパース

```go
// SnapSQL変数置換のパース
func (p *SqlParser) parseVariableSubstitution() (*VariableSubstitution, error) {
    if !p.check(BLOCK_COMMENT) {
        return nil, p.error("Expected SnapSQL variable substitution")
    }
    
    token := p.advance()
    if !token.IsSnapSQLDirective || token.DirectiveType != "variable" {
        return nil, p.error("Expected variable substitution directive")
    }
    
    // /*= variable */dummy_value の形式をパース
    variable := p.extractVariableName(token.Value)
    dummyValue := ""
    
    // ダミーリテラルの検出
    if p.current < len(p.tokens) {
        nextToken := p.tokens[p.current]
        if p.isDummyLiteral(nextToken) {
            dummyValue = nextToken.Value
            p.advance()
        }
    }
    
    // 自動削除条件の生成
    autoCondition := &AutoCondition{
        Variable:   variable,
        CheckType:  "not_null",
        Expression: fmt.Sprintf("%s != null", variable),
    }
    
    return &VariableSubstitution{
        Variable:     variable,
        DummyValue:   dummyValue,
        AutoRemove:   autoCondition,
    }, nil
}

// SnapSQLループのパース
func (p *SqlParser) parseTemplateForBlock() (*TemplateForBlock, error) {
    if !p.checkSnapSQLDirective("for") {
        return nil, p.error("Expected for directive")
    }
    
    forToken := p.advance()
    
    // /*# for variable : list */ の解析
    variable, list := p.parseForDirective(forToken.Value)
    
    // ループ内容のパース
    template, err := p.parseTemplateContent()
    if err != nil {
        return nil, err
    }
    
    // /*# end */ の確認
    if !p.checkSnapSQLDirective("end") {
        return nil, p.error("Expected end directive")
    }
    p.advance()
    
    // 区切り文字の自動判定
    separator := p.determineSeparator(template)
    
    // 自動条件の生成
    autoCondition := &AutoCondition{
        Variable:   list,
        CheckType:  "not_null_and_not_empty",
        Expression: fmt.Sprintf("%s != null && %s.length > 0", list, list),
    }
    
    return &TemplateForBlock{
        Variable:      variable,
        List:          list,
        Template:      template,
        Separator:     separator,
        AutoCondition: autoCondition,
    }, nil
}
```

### 自動最適化ヒント挿入

#### ヒント挿入エンジン

```go
type OptimizationHintInserter struct {
    parser *SqlParser
}

func (o *OptimizationHintInserter) InsertHints(stmt *SelectStatement) {
    // WHERE句の最適化ヒント
    if stmt.WhereClause != nil {
        o.insertWhereClauseHints(stmt.WhereClause)
    }
    
    // ORDER BY句の最適化ヒント
    if stmt.OrderByClause != nil {
        o.insertOrderByClauseHints(stmt.OrderByClause)
    }
    
    // LIMIT/OFFSET句の最適化ヒント
    if stmt.LimitClause != nil {
        o.insertLimitClauseHints(stmt.LimitClause)
    }
    if stmt.OffsetClause != nil {
        o.insertOffsetClauseHints(stmt.OffsetClause)
    }
}

func (o *OptimizationHintInserter) insertWhereClauseHints(whereClause *WhereClause) {
    // WHERE句内の各条件をチェック
    conditions := o.extractConditions(whereClause)
    
    for _, condition := range conditions {
        if varSub := o.findVariableSubstitution(condition); varSub != nil {
            // 変数置換に対する自動削除ヒント
            hint := &RuntimeHint{
                HintType:  "auto_condition",
                Target:    varSub.Variable,
                Condition: fmt.Sprintf("%s != null", varSub.Variable),
                OptLevel:  "condition",
            }
            o.parser.hints = append(o.parser.hints, *hint)
        }
    }
    
    // WHERE句全体の削除ヒント
    allConditionsHint := &RuntimeHint{
        HintType:  "clause_removal",
        Target:    "WHERE",
        Condition: "all_conditions_removed",
        OptLevel:  "clause",
    }
    o.parser.hints = append(o.parser.hints, *allConditionsHint)
}

func (o *OptimizationHintInserter) insertOrderByClauseHints(orderByClause *OrderByClause) {
    // ORDER BY句内のforループをチェック
    if forBlock := o.findForBlock(orderByClause); forBlock != nil {
        hint := &RuntimeHint{
            HintType:  "auto_condition",
            Target:    forBlock.List,
            Condition: fmt.Sprintf("%s != null && %s.length > 0", forBlock.List, forBlock.List),
            OptLevel:  "clause",
        }
        o.parser.hints = append(o.parser.hints, *hint)
    }
}
```

### エラー処理とリカバリ

#### パースエラー定義

```go
type ParseError struct {
    Message   string
    Position  Position
    Token     Token
    Severity  ErrorSeverity
    Context   string
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("[%s] %s at line %d, column %d (token: %s): %s",
        e.Severity, e.Message, e.Position.Line, e.Position.Column, e.Token.Value, e.Context)
}

type ErrorSeverity int

const (
    WARNING ErrorSeverity = iota
    ERROR
    FATAL
)
```

#### エラーリカバリ戦略

```go
func (p *SqlParser) error(message string) error {
    err := &ParseError{
        Message:  message,
        Position: p.currentToken().Position,
        Token:    p.currentToken(),
        Severity: ERROR,
        Context:  p.getContext(),
    }
    p.errors = append(p.errors, *err)
    return err
}

func (p *SqlParser) recover() {
    // パニックモードリカバリ
    for !p.isAtEnd() && !p.isStatementBoundary() {
        p.advance()
    }
}

func (p *SqlParser) isStatementBoundary() bool {
    return p.check(SEMICOLON) || 
           p.check(SELECT) || 
           p.check(INSERT) || 
           p.check(UPDATE) || 
           p.check(DELETE)
}
```

### 中間表現出力

#### JSON形式での出力

```go
type IntermediateRepresentation struct {
    AST   *SelectStatement  `json:"ast"`
    Hints []RuntimeHint     `json:"optimization_hints"`
    Meta  *ParseMetadata    `json:"metadata"`
}

type ParseMetadata struct {
    SourceSQL     string    `json:"source_sql"`
    Dialect       string    `json:"dialect"`
    ParseTime     time.Time `json:"parse_time"`
    TokenCount    int       `json:"token_count"`
    ErrorCount    int       `json:"error_count"`
    WarningCount  int       `json:"warning_count"`
}

func (p *SqlParser) GenerateIR() (*IntermediateRepresentation, error) {
    stmt, err := p.Parse()
    if err != nil {
        return nil, err
    }
    
    return &IntermediateRepresentation{
        AST:   stmt,
        Hints: p.hints,
        Meta: &ParseMetadata{
            Dialect:      p.dialect.Name(),
            ParseTime:    time.Now(),
            TokenCount:   len(p.tokens),
            ErrorCount:   p.countErrors(ERROR),
            WarningCount: p.countErrors(WARNING),
        },
    }, nil
}
```

### Visitor パターンによるAST処理

#### Visitor インターフェース

```go
type AstVisitor interface {
    VisitSelectStatement(stmt *SelectStatement) error
    VisitSelectClause(clause *SelectClause) error
    VisitWhereClause(clause *WhereClause) error
    VisitOrderByClause(clause *OrderByClause) error
    VisitVariableSubstitution(varSub *VariableSubstitution) error
    VisitTemplateForBlock(forBlock *TemplateForBlock) error
    VisitConditionalClause(condClause *ConditionalClause) error
    VisitRuntimeHint(hint *RuntimeHint) error
}

// SQL生成Visitor
type SqlGeneratorVisitor struct {
    output    strings.Builder
    params    map[string]interface{}
    context   *GenerationContext
}

func (v *SqlGeneratorVisitor) VisitSelectStatement(stmt *SelectStatement) error {
    v.output.WriteString("SELECT ")
    
    if err := stmt.SelectClause.Accept(v); err != nil {
        return err
    }
    
    if stmt.FromClause != nil {
        v.output.WriteString(" FROM ")
        if err := stmt.FromClause.Accept(v); err != nil {
            return err
        }
    }
    
    // WHERE句の条件付き生成
    if stmt.WhereClause != nil && v.shouldIncludeWhereClause(stmt.WhereClause) {
        v.output.WriteString(" WHERE ")
        if err := stmt.WhereClause.Accept(v); err != nil {
            return err
        }
    }
    
    return nil
}

func (v *SqlGeneratorVisitor) shouldIncludeWhereClause(whereClause *WhereClause) bool {
    // ランタイムヒントに基づく判定
    for _, hint := range v.context.Hints {
        if hint.Target == "WHERE" && hint.HintType == "clause_removal" {
            return v.evaluateCondition(hint.Condition)
        }
    }
    return true
}
```

この設計により、パーサーは以下を実現します：

1. **完全なAST生成**: SQL構造の完全な表現
2. **自動最適化ヒント**: ランタイム処理の効率化
3. **エラー処理**: 堅牢なパース処理
4. **拡張性**: 新しいSQL構文への対応
5. **中間表現**: 言語非依存な出力形式

パーサー実装の準備が整いました。実装を開始しますか？
