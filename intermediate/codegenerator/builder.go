package codegenerator

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/tokenizer"
)

// conditionalLevel はネストされた条件分岐のレベルを追跡する
type conditionalLevel struct {
	startPos   string
	hasElse    bool
	hasElseIf  bool
	exprIndex  int
	whereConds []*WhereDynamicCondition
}

// loopLevel はネストされたループのレベルを追跡する
type loopLevel struct {
	startPos   string
	expression string // CEL式
	exprIndex  int
	envIndex   int // このループが持つ CELEnvironment のインデックス
	whereCond  *WhereDynamicCondition
}

// InstructionBuilder は命令列を段階的に構築するビルダー
type InstructionBuilder struct {
	instructions           []Instruction
	context                *GenerationContext
	conditionalStack       []conditionalLevel
	loopStack              []loopLevel
	envStack               []int // CELEnvironment のインデックススタック（root=0から始まる）
	dummyValuesInitialized bool  // FunctionDefinition からのダミー値初期化済みフラグ
	systemFieldsAdded      bool  // VALUES句でシステムフィールド値が既に追加されているかフラグ
	whereMetaStack         []*WhereClauseMeta
}

// NewInstructionBuilder は新しい InstructionBuilder を作成する
func NewInstructionBuilder(ctx *GenerationContext) *InstructionBuilder {
	return &InstructionBuilder{
		context:                ctx,
		instructions:           make([]Instruction, 0, 64),
		envStack:               []int{0}, // root environment from start
		dummyValuesInitialized: false,
	}
}

func (b *InstructionBuilder) instructionCount() int {
	return len(b.instructions)
}

func appendUniqueInt(values []int, value int) []int {
	if slices.Contains(values, value) {
		return values
	}

	return append(values, value)
}

func (b *InstructionBuilder) pushWhereMeta(meta *WhereClauseMeta) {
	if meta == nil {
		return
	}

	b.whereMetaStack = append(b.whereMetaStack, meta)
}

func (b *InstructionBuilder) popWhereMeta() {
	if len(b.whereMetaStack) == 0 {
		return
	}

	b.whereMetaStack = b.whereMetaStack[:len(b.whereMetaStack)-1]
}

func (b *InstructionBuilder) currentWhereMeta() *WhereClauseMeta {
	if len(b.whereMetaStack) == 0 {
		return nil
	}

	return b.whereMetaStack[len(b.whereMetaStack)-1]
}

// BeginValueGroup resets the system field insertion flag before starting a new VALUES group.
func (b *InstructionBuilder) BeginValueGroup() {
	b.systemFieldsAdded = false
}

// lookupTypeDescriptor returns the parser-provided descriptor for the given token.
func (b *InstructionBuilder) lookupTypeDescriptor(token tokenizer.Token) any {
	if b.context == nil {
		return nil
	}

	return b.context.LookupTypeDescriptor(token.Position.String())
}

// annotateExpression registers position and type metadata for a CEL expression.
func (b *InstructionBuilder) annotateExpression(exprIndex int, token tokenizer.Token, explicitType any) {
	pos := Position{Line: token.Position.Line, Column: token.Position.Column}

	typeDesc := explicitType
	if typeDesc == nil {
		typeDesc = b.lookupTypeDescriptor(token)
	}

	if b.context != nil {
		b.context.SetExpressionMetadata(exprIndex, pos, typeDesc)
	}
}

// parsePosition converts "line:column" into Position.
func parsePosition(pos string) (Position, bool) {
	if pos == "" {
		return Position{}, false
	}

	parts := strings.Split(pos, ":")
	if len(parts) != 2 {
		return Position{}, false
	}

	line, err1 := strconv.Atoi(parts[0])

	col, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return Position{}, false
	}

	return Position{Line: line, Column: col}, true
}

// getCurrentEnvironmentIndex は現在アクティブな CEL 環境のインデックスを取得する
// envStack の最後の要素が現在のアクティブ環境
func (b *InstructionBuilder) getCurrentEnvironmentIndex() int {
	if len(b.envStack) == 0 {
		return 0 // fallback to root
	}

	return b.envStack[len(b.envStack)-1]
}

// pushEnvironment は新しい環境をスタックにプッシュ
func (b *InstructionBuilder) pushEnvironment(envIndex int) {
	b.envStack = append(b.envStack, envIndex)
}

// popEnvironment は現在の環境をスタックからポップ
func (b *InstructionBuilder) popEnvironment() {
	if len(b.envStack) > 1 {
		b.envStack = b.envStack[:len(b.envStack)-1]
	}
}

// EvaluateExpression は CEL 式を評価して現在の環境でのダミー値を返す
// github.com/google/cel-go/cel パッケージを使用して式をコンパイルし、
// ダミー値の環境でEvalして実際の値を取得する
//
// Parameters:
//   - expr: string - The CEL expression to evaluate
//
// Returns:
//   - result: interface{} - The evaluated result (can be []any, map[string]any, scalar, etc.)
//   - error: error if evaluation fails
func (b *InstructionBuilder) EvaluateExpression(expr string) (any, error) {
	// Get current environment index
	envIndex := b.getCurrentEnvironmentIndex()

	// Check bounds
	if envIndex < 0 || envIndex >= len(b.context.CELEnvironments) {
		panic(fmt.Sprintf("invalid environment index: %d", envIndex))
	}

	// Get environment
	env := b.context.CELEnvironments[envIndex]

	// Build parameter map from environment variables (dummy values)
	params := make(map[string]any)

	var celVars []cel.EnvOption

	for _, varInfo := range env.AdditionalVariables {
		params[varInfo.Name] = varInfo.Value

		// Create CEL variable option for each variable
		// We need to determine the CEL type from the Go value
		var celType *cel.Type

		if varInfo.Value != nil {
			switch varInfo.Value.(type) {
			case string:
				celType = cel.StringType
			case int, int32, int64:
				celType = cel.IntType
			case float32, float64:
				celType = cel.DoubleType
			case bool:
				celType = cel.BoolType
			case []any:
				celType = cel.ListType(cel.DynType)
			case map[string]any:
				celType = cel.MapType(cel.StringType, cel.DynType)
			default:
				celType = cel.DynType
			}
		} else {
			celType = cel.DynType
		}

		celVars = append(celVars, cel.Variable(varInfo.Name, celType))
	}

	// Create CEL environment with variables
	celEnvOpts := []cel.EnvOption{
		cel.Container(env.Container),
	}
	celEnvOpts = append(celEnvOpts, celVars...)

	celEnv, err := cel.NewEnv(celEnvOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	// Compile the expression
	ast, issues := celEnv.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL expression '%s': %w", expr, issues.Err())
	}

	// Create program from AST
	program, err := celEnv.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program for '%s': %w", expr, err)
	}

	// Evaluate the program with dummy values
	result, _, err := program.Eval(params)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate CEL expression '%s': %w", expr, err)
	}

	// Convert CEL value to Go value
	return result.Value(), nil
}

// ProcessTokens はトークン列を処理して命令列に追加する
// 各clause関数がカスタマイズしたトークン列を受け取り、
// CEL式の管理とディレクティブ処理、命令生成を行う
//
// 最適化：連続するホワイトスペース、ブロックコメント、ラインコメントは1スペースにマージ
type processTokensConfig struct {
	skipLeadingTrivia bool
}

// ProcessTokensOption allows configuring ProcessTokens behavior.
type ProcessTokensOption func(*processTokensConfig)

// WithSkipLeadingTrivia removes leading whitespace and non-directive comments before processing.
func WithSkipLeadingTrivia() ProcessTokensOption {
	return func(cfg *processTokensConfig) {
		cfg.skipLeadingTrivia = true
	}
}

func (b *InstructionBuilder) ProcessTokens(tokens []tokenizer.Token, opts ...ProcessTokensOption) error {
	cfg := processTokensConfig{}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.skipLeadingTrivia {
		tokens = skipLeadingTrivia(tokens)
	}

	// Step 1: 方言変換（トークン列全体を事前処理）
	convertedTokens := b.applyDialectConversions(tokens)

	// Step 1.5: FunctionDefinitionが設定されている場合、ダミー値を生成してCEL環境に登録
	if b.context.FunctionDefinition != nil {
		b.initializeDummyValuesFromFunctionDefinition()
	}

	// Step 2: 通常のトークン処理
	for i := 0; i < len(convertedTokens); i++ {
		token := convertedTokens[i]

		// ディレクティブの処理（コメントやホワイトスペースよりも優先）
		if token.Directive != nil {
			switch token.Directive.Type {
			case "variable":
				// 変数展開ディレクティブ: /*= expression */dummy_value
				// CEL式をコンテキストに追加し、EMIT_EVAL命令を生成
				// token.Directive.Condition に式が格納されている
				envIndex := b.getCurrentEnvironmentIndex()

				// 単一の EMIT_EVAL を生成
				exprIndex := b.context.AddExpression(token.Directive.Condition, envIndex)
				b.annotateExpression(exprIndex, token, nil)
				b.instructions = append(b.instructions, Instruction{
					Op:        OpEmitEval,
					Pos:       token.Position.String(),
					ExprIndex: &exprIndex,
				})

				if meta := b.currentWhereMeta(); meta != nil {
					meta.RecordEval()
				}

				// 次のトークンがDUMMY_STARTの場合、DUMMY_ENDまでスキップ
				// パーサーがディレクティブの後にDUMMY_START, <dummy_value>, DUMMY_ENDを挿入する
				dummyStartIdx := -1
				if i+1 < len(convertedTokens) && convertedTokens[i+1].Type == tokenizer.DUMMY_START {
					dummyStartIdx = i + 1
					i++ // Skip DUMMY_START
					// Skip all tokens until DUMMY_END
					for i+1 < len(convertedTokens) && convertedTokens[i+1].Type != tokenizer.DUMMY_END {
						i++
					}
					// Skip DUMMY_END
					if i+1 < len(convertedTokens) && convertedTokens[i+1].Type == tokenizer.DUMMY_END {
						i++
					}
				}

				// ディレクティブ処理後、IN句のリテラル値パターン（カンマ区切り）を自動ループ化
				// パターン: OPENED_PARENS, NUMBER/STRING, COMMA, NUMBER/STRING, COMMA, ... CLOSED_PARENS
				if dummyStartIdx >= 0 && i+1 < len(convertedTokens) && convertedTokens[i+1].Type == tokenizer.OPENED_PARENS {
					// 括弧内の値がカンマで区切られたリテラル値かチェック
					if b.isCommaDelimitedLiteralList(convertedTokens, i+1) {
						// リテラル値リストを自動的にループ化
						i = b.emitInClauseLoop(convertedTokens, i+1, token.Directive.Condition)
						continue
					}
				}

				continue

			case "if":
				// 条件分岐の開始: /*# if condition */
				// CEL式をコンテキストに追加し、IF命令を生成
				b.addIfCondition(token)

				continue

			case "elseif":
				// else if 分岐: /*# elseif condition */
				if len(b.conditionalStack) == 0 {
					return fmt.Errorf("%w: elseif directive at %s without matching if", ErrDirectiveMismatch, token.Position.String())
				}

				currentLevel := &b.conditionalStack[len(b.conditionalStack)-1]
				if currentLevel.hasElse {
					return fmt.Errorf("%w: elseif directive at %s after else directive", ErrDirectiveMismatch, token.Position.String())
				}

				currentLevel.hasElseIf = true

				// CEL式を追加してELSE_IF命令を生成
				envIndex := b.getCurrentEnvironmentIndex()
				exprIndex := b.context.AddExpression(token.Directive.Condition, envIndex)
				b.annotateExpression(exprIndex, token, nil)
				b.instructions = append(b.instructions, Instruction{
					Op:        OpElseIf,
					Pos:       token.Position.String(),
					ExprIndex: &exprIndex,
				})

				if meta := b.currentWhereMeta(); meta != nil {
					meta.Dynamic = true
					meta.ExpressionRefs = appendUniqueInt(meta.ExpressionRefs, exprIndex)
					whereCond := &WhereDynamicCondition{
						ExprIndex:        exprIndex,
						NegatedWhenEmpty: true,
						Description:      strings.TrimSpace(token.Directive.Condition),
					}
					meta.DynamicConditions = append(meta.DynamicConditions, whereCond)

					if currentLevel.whereConds != nil {
						currentLevel.whereConds = append(currentLevel.whereConds, whereCond)
					}

					meta.EnterElseIf(exprIndex)
				}

				continue

			case "else":
				// else 分岐: /*# else */
				err := b.addElseCondition(&token.Position)
				if err != nil {
					return err
				}

				continue

			case "end":
				// END命令：条件分岐またはループの終了を識別
				// conditionalStack と loopStack の両方をチェック
				// ループの終了の方が優先度が高い（ネストの場合）
				if len(b.loopStack) > 0 {
					// ループの終了
					b.loopStack = b.loopStack[:len(b.loopStack)-1]
					// 環境スタックからもポップ
					b.popEnvironment()

					// 親環境のインデックス：ポップ後の現在の環境
					parentEnvIndex := b.getCurrentEnvironmentIndex()
					parentEnvIndexPtr := &parentEnvIndex

					b.instructions = append(b.instructions, Instruction{
						Op:       OpLoopEnd,
						EnvIndex: parentEnvIndexPtr,
						Pos:      token.Position.String(),
					})
				} else if len(b.conditionalStack) > 0 {
					// 条件分岐の終了
					b.conditionalStack = b.conditionalStack[:len(b.conditionalStack)-1]
					b.addEndCondition(&token.Position)
				} else {
					return fmt.Errorf("%w: end directive at %s without matching if or for", ErrDirectiveMismatch, token.Position.String())
				}

				continue

			case "for":
				// FOR ループディレクティブ: /*# for variable : expression */
				if token.Directive.Condition == "" {
					return fmt.Errorf("%w: for directive at %s without expression", ErrLoopMismatch, token.Position.String())
				}

				// ネストの深さをチェック（最大10レベル）
				if len(b.loopStack) >= 10 {
					return fmt.Errorf("%w: for loop at %s exceeds maximum nesting depth", ErrLoopNesting, token.Position.String())
				}

				// ディレクティブの値から変数と式を抽出
				// 形式: "variable : expression"
				parts := strings.Split(token.Directive.Condition, ":")
				if len(parts) != 2 {
					return fmt.Errorf("%w: for directive at %s has invalid format (expected 'variable : expression')", ErrLoopMismatch, token.Position.String())
				}

				variable := strings.TrimSpace(parts[0])
				expression := strings.TrimSpace(parts[1])

				// CEL式をコンテキストに追加 (root environment)
				exprIndex := b.context.AddExpression(expression, 0)
				b.annotateExpression(exprIndex, token, nil)

				// ループ変数の CEL 環境を作成
				loopEnvIndex := b.context.AddCELEnvironment(CELEnvironment{
					AdditionalVariables: []CELVariableInfo{
						{
							Name: variable,
							Type: "any", // ループ変数は初期段階では any 型
						},
					},
					Container: fmt.Sprintf("for %s : %s", variable, expression),
				})

				// ループスタックにプッシュ
				b.loopStack = append(b.loopStack, loopLevel{
					startPos:   token.Position.String(),
					expression: expression,
					exprIndex:  exprIndex,
					envIndex:   loopEnvIndex,
				})

				// 環境スタックに新しい環境をプッシュ
				b.pushEnvironment(loopEnvIndex)

				// LOOP_START命令を生成（EnvIndex はループ環境のインデックス）
				b.instructions = append(b.instructions, Instruction{
					Op:                  OpLoopStart,
					Variable:            variable,
					CollectionExprIndex: &exprIndex,
					EnvIndex:            &loopEnvIndex,
					Pos:                 token.Position.String(),
				})

				continue

			default:
				// 未知のディレクティブは無視
				continue
			}
		}

		// DUMMY_START、DUMMY_END、DUMMY_LITERALトークンはスキップ
		// これらはパーサーがディレクティブのダミー値をマークするためのものであり、
		// 生成されるSQLには含めない
		if token.Type == tokenizer.DUMMY_START ||
			token.Type == tokenizer.DUMMY_END ||
			token.Type == tokenizer.DUMMY_LITERAL {
			continue
		}

		// CEL 式の処理（Phase 1 では未実装、空のエラーを返さずスキップ）
		// TODO: Phase 2 以降で CEL 式を context.Expressions に追加し、EMIT_CEL_EXPRESSION 命令を生成
		// if token.Type == tokenizer.SOME_CEL_TYPE {
		//     if err := b.processCELExpression(token); err != nil {
		//         return fmt.Errorf("failed to process CEL expression at token %d: %w", i, err)
		//     }
		//     continue
		// }

		// ホワイトスペースとコメントの最適化：連続するものは1スペースにマージ
		// ただし、ディレクティブを持つコメントは上で処理済みなので、ここには来ない
		if b.isWhitespaceOrComment(token.Type) {
			// 連続するホワイトスペース・コメントをスキップ
			// ただし、ディレクティブを持つトークンはスキップしない
			for i+1 < len(convertedTokens) &&
				b.isWhitespaceOrComment(convertedTokens[i+1].Type) &&
				convertedTokens[i+1].Directive == nil {
				i++
			}
			// 1スペースを追加（最初のトークンの位置を保持）
			b.addStatic(" ", &token.Position)

			continue
		}

		// 静的トークン（通常の SQL トークン）
		// 次のトークンが条件分岐ディレクティブで、現在のトークンが AND/OR/カンマの場合
		// EMIT_UNLESS_BOUNDARY を使用
		useUnlessBoundary := false

		if i+1 < len(convertedTokens) && convertedTokens[i+1].Directive != nil {
			nextDirectiveType := convertedTokens[i+1].Directive.Type
			if nextDirectiveType == "elseif" || nextDirectiveType == "else" || nextDirectiveType == "end" {
				// 現在のトークンが AND/OR/カンマかチェック
				trimmed := strings.TrimSpace(token.Value)
				if trimmed == "AND" || trimmed == "OR" || trimmed == "," {
					useUnlessBoundary = true
				}
			}
		}

		if useUnlessBoundary {
			// AND/OR の場合は前後にスペースを追加
			value := strings.TrimSpace(token.Value)
			if value == "AND" || value == "OR" {
				value = " " + value + " "
			}

			b.instructions = append(b.instructions, Instruction{
				Op:    OpEmitUnlessBoundary,
				Value: value,
				Pos:   token.Position.String(),
			})
		} else {
			b.addStatic(token.Value, &token.Position)
		}
	}

	return nil
}

func skipLeadingTrivia(tokens []tokenizer.Token) []tokenizer.Token {
	index := 0
	for index < len(tokens) {
		tok := tokens[index]

		switch tok.Type {
		case tokenizer.WHITESPACE:
			index++
			continue
		case tokenizer.BLOCK_COMMENT, tokenizer.LINE_COMMENT:
			if tok.Directive == nil {
				index++
				continue
			}
		}

		break
	}

	return tokens[index:]
}

func trimTrailingStaticWhitespace(instructions []Instruction) []Instruction {
	if len(instructions) == 0 {
		return instructions
	}

	last := instructions[len(instructions)-1]
	if last.Op == OpEmitStatic {
		last.Value = strings.TrimRight(last.Value, " \t\n\r")
		instructions[len(instructions)-1] = last
	}

	return instructions
}

func (b *InstructionBuilder) addElseCondition(pos *tokenizer.Position) error {
	if len(b.conditionalStack) == 0 {
		return fmt.Errorf("%w: else directive at %s without matching if", ErrDirectiveMismatch, pos.String())
	}

	currentLevel := &b.conditionalStack[len(b.conditionalStack)-1]
	if currentLevel.hasElse {
		return fmt.Errorf("%w: duplicate else directive at %s", ErrDirectiveMismatch, pos.String())
	}

	currentLevel.hasElse = true
	if len(currentLevel.whereConds) > 0 {
		for _, cond := range currentLevel.whereConds {
			if cond == nil {
				continue
			}

			cond.HasElse = true
			cond.NegatedWhenEmpty = false
		}
	}

	// ELSE命令を生成
	b.addRawElseCondition(pos)

	if meta := b.currentWhereMeta(); meta != nil {
		meta.EnterElse()
	}

	return nil
}

func (b *InstructionBuilder) addRawElseCondition(pos *tokenizer.Position) {
	b.instructions = append(b.instructions, Instruction{
		Op:  OpElse,
		Pos: pos.String(),
	})
}

func (b *InstructionBuilder) addIfCondition(token tokenizer.Token) {
	envIndex := b.getCurrentEnvironmentIndex()
	exprIndex := b.context.AddExpression(token.Directive.Condition, envIndex)
	b.annotateExpression(exprIndex, token, nil)
	b.instructions = append(b.instructions, Instruction{
		Op:        OpIf,
		Pos:       token.Position.String(),
		ExprIndex: &exprIndex,
	})
	// スタックに新しい条件レベルをプッシュ
	meta := b.currentWhereMeta()

	var whereCond *WhereDynamicCondition

	if meta != nil {
		meta.Present = true
		meta.Dynamic = true
		meta.ExpressionRefs = appendUniqueInt(meta.ExpressionRefs, exprIndex)
		whereCond = &WhereDynamicCondition{
			ExprIndex:        exprIndex,
			NegatedWhenEmpty: true,
			Description:      strings.TrimSpace(token.Directive.Condition),
		}
		meta.DynamicConditions = append(meta.DynamicConditions, whereCond)
		meta.EnterIf(exprIndex)
	}

	b.conditionalStack = append(b.conditionalStack, conditionalLevel{
		startPos:  token.Position.String(),
		hasElse:   false,
		hasElseIf: false,
		exprIndex: exprIndex,
		whereConds: func() []*WhereDynamicCondition {
			if whereCond == nil {
				return nil
			}

			return []*WhereDynamicCondition{whereCond}
		}(),
	})
}

func (b *InstructionBuilder) addEndCondition(pos *tokenizer.Position) {
	if meta := b.currentWhereMeta(); meta != nil {
		meta.ExitConditional()
	}

	b.instructions = append(b.instructions, Instruction{
		Op:  OpEnd,
		Pos: pos.String(),
	})
}

// isWhitespaceOrComment はトークンがホワイトスペースまたはコメントかを判定する
func (b *InstructionBuilder) isWhitespaceOrComment(tokenType tokenizer.TokenType) bool {
	return tokenType == tokenizer.WHITESPACE ||
		tokenType == tokenizer.BLOCK_COMMENT ||
		tokenType == tokenizer.LINE_COMMENT
}

// GetCELExpressions は収集された CEL 式のリストを返す
// Phase 1 では常に空配列を返す
func (b *InstructionBuilder) GetCELExpressions() []CELExpression {
	return []CELExpression{}
}

// GetCELEnvironments は収集された CEL 環境のリストを返す
func (b *InstructionBuilder) GetCELEnvironments() []CELEnvironment {
	return b.context.CELEnvironments
}

// Finalize は最適化を実行して最終的な命令列を返す
func (b *InstructionBuilder) Finalize() []Instruction {
	// Phase 0: ループ/条件分岐の END 直前のカンマ/AND/OR を EMIT_UNLESS_BOUNDARY に変換
	b.convertLoopAndConditionalEndDelimiters()

	// Phase 1: 連続する EMIT_STATIC 命令をマージ
	optimized := b.mergeStaticInstructions()

	// Phase 2: ループ/条件分岐終了後に BOUNDARY を挿入
	optimized = b.insertBoundariesAfterLoopsAndConditions(optimized)

	optimized = trimTrailingStaticWhitespace(optimized)

	// 将来的に以下を実装予定：
	// - b.optimizeBoundaries()

	return optimized
}

// convertLoopAndConditionalEndDelimiters は、ループと条件分岐の END 直前の
// カンマ/AND/OR を EMIT_UNLESS_BOUNDARY に変換する
func (b *InstructionBuilder) convertLoopAndConditionalEndDelimiters() {
	for i := 0; i < len(b.instructions); i++ {
		instr := b.instructions[i]

		// LOOP_END または END の直前の命令をチェック
		if (instr.Op == OpLoopEnd || instr.Op == OpEnd) && i > 0 {
			prevIdx := i - 1
			prevInstr := b.instructions[prevIdx]

			// 直前の命令が EMIT_STATIC でカンマ/AND/OR の場合、変換
			if prevInstr.Op == OpEmitStatic {
				value := strings.TrimSpace(prevInstr.Value)
				if value == "," || value == "AND" || value == "OR" {
					b.instructions[prevIdx] = Instruction{
						Op:    OpEmitUnlessBoundary,
						Value: normalizeDelimiter(prevInstr.Value),
						Pos:   prevInstr.Pos,
					}
				}
			}
		}
	}
}

func normalizeDelimiter(value string) string {
	trimmed := strings.TrimSpace(value)
	return normalizeDelimiterValue(trimmed)
}

func normalizeDelimiterValue(trimmed string) string {
	switch trimmed {
	case ",":
		return ", "
	case "AND", "OR":
		return " " + trimmed + " "
	default:
		return trimmed
	}
}

// mergeStaticInstructions は連続する EMIT_STATIC 命令をマージする
// 最初の命令の位置情報を保持する
// END/LOOP_END 直前のカンマ/AND/OR は EMIT_UNLESS_BOUNDARY に変換
func (b *InstructionBuilder) mergeStaticInstructions() []Instruction {
	if len(b.instructions) == 0 {
		return b.instructions
	}

	result := make([]Instruction, 0, len(b.instructions))

	for i := 0; i < len(b.instructions); i++ {
		current := b.instructions[i]

		// EMIT_STATIC 以外の命令はそのまま追加
		if current.Op != OpEmitStatic {
			result = append(result, current)
			continue
		}

		// 連続する EMIT_STATIC をマージ
		mergedValue := current.Value
		firstPos := current.Pos

		// 次の命令も EMIT_STATIC ならマージ
		var mergedValueSb692 strings.Builder

		for i+1 < len(b.instructions) && b.instructions[i+1].Op == OpEmitStatic {
			i++
			mergedValueSb692.WriteString(b.instructions[i].Value)
		}

		mergedValue += mergedValueSb692.String()

		// マージされた命令の直後が LOOP_END の場合、末尾のカンマ/AND/OR を分割
		// （IF/END の場合は分割しない）
		if i+1 < len(b.instructions) {
			nextInstr := b.instructions[i+1]
			if nextInstr.Op == OpLoopEnd {
				// 末尾のカンマ/AND/OR を分割してチェック
				delimiter, remaining := b.extractTrailingDelimiter(mergedValue)
				if delimiter != "" {
					// remaining が空でない場合のみ追加
					if remaining != "" {
						result = append(result, Instruction{
							Op:    OpEmitStatic,
							Value: remaining,
							Pos:   firstPos,
						})
					}
					// delimiter を EMIT_UNLESS_BOUNDARY で追加
					result = append(result, Instruction{
						Op:    OpEmitUnlessBoundary,
						Value: normalizeDelimiter(delimiter),
						Pos:   firstPos,
					})

					continue
				}
			}
		}

		// マージされた命令を追加（最初の位置を保持）
		result = append(result, Instruction{
			Op:    OpEmitStatic,
			Value: mergedValue,
			Pos:   firstPos,
		})
	}

	return result
}

// extractTrailingDelimiter は、文字列の末尾からカンマ/AND/OR を抽出する
// delimiter: 抽出されたカンマ/AND/OR（前後のスペース含む）
// remaining: カンマ/AND/OR を除いた残りの文字列
func (b *InstructionBuilder) extractTrailingDelimiter(value string) (string, string) {
	// 末尾のホワイトスペースを除去
	trimmed := strings.TrimRight(value, " \t\n\r")
	trailingSpace := value[len(trimmed):]

	// 末尾から delimiter を検索
	delimiters := []string{",", "AND", "OR"}
	for _, delim := range delimiters {
		if strings.HasSuffix(trimmed, delim) {
			// delimiter 前のスペースを除去
			before := trimmed[:len(trimmed)-len(delim)]
			beforeTrimmed := strings.TrimRight(before, " \t\n\r")
			spaceBefore := before[len(beforeTrimmed):]

			// remaining = beforeTrimmed + spaceBefore（delimiter 前は元のフォーマット保持）
			// delimiter 出力 = spaceBefore + delim + trailingSpace
			return spaceBefore + delim + trailingSpace, beforeTrimmed
		}
	}

	return "", ""
}

// insertBoundariesAfterLoopsAndConditions は LOOP_END/END の直後に BOUNDARY を挿入する
// ループや条件分岐が句の終わりにある場合、その終了後に BOUNDARY を挿入して境界を明確にする
// ただし、システム命令（IF_SYSTEM_LIMIT など）内の END は対象外
func (b *InstructionBuilder) insertBoundariesAfterLoopsAndConditions(instructions []Instruction) []Instruction {
	if len(instructions) == 0 {
		return instructions
	}

	result := make([]Instruction, 0, len(instructions)+10) // 余裕を持たせる

	for i := range instructions {
		instr := instructions[i]
		result = append(result, instr)

		// LOOP_END のみが対象（END はシステム命令内のものが多いため対象外）
		if instr.Op == OpLoopEnd {
			// 次の命令を確認
			hasNextInstruction := i+1 < len(instructions)

			if hasNextInstruction {
				nextInstr := instructions[i+1]

				// 次の命令が EMIT_STATIC の場合、BOUNDARY は不要（静的テキストが続く）
				// それ以外の場合（他の指令/システム命令）、または末尾の場合に BOUNDARY を挿入
				if nextInstr.Op != OpEmitStatic {
					// システム命令など他の指令が続く場合、BOUNDARY を挿入
					result = append(result, Instruction{
						Op:  OpBoundary,
						Pos: instr.Pos,
					})
				}
			} else {
				// 末尾の場合も BOUNDARY を挿入
				// （システム命令が後に付加される可能性があるため）
				result = append(result, Instruction{
					Op:  OpBoundary,
					Pos: instr.Pos,
				})
			}
		}
	}

	return result
}

// AddBoundary は句の終了時に BOUNDARY 命令を追加する
// 末尾の命令が END の場合のみ BOUNDARY を追加する
// （条件分岐で終わる場合、次の句との境界を明確にするため）
func (b *InstructionBuilder) AddBoundary() {
	if len(b.instructions) == 0 {
		return
	}

	// 末尾の命令が END の場合のみ BOUNDARY を追加
	lastInstr := b.instructions[len(b.instructions)-1]
	if lastInstr.Op != OpEnd {
		return
	}

	// 末尾が AND/OR/カンマの EMIT_STATIC の場合、EMIT_UNLESS_BOUNDARY に変換
	// （ENDの一つ前の命令をチェック）
	if len(b.instructions) >= 2 {
		beforeEndIdx := len(b.instructions) - 2

		beforeEndInstr := b.instructions[beforeEndIdx]
		if beforeEndInstr.Op == OpEmitStatic {
			value := strings.TrimSpace(beforeEndInstr.Value)
			if value == "AND" || value == "OR" || value == "," {
				b.instructions[beforeEndIdx] = Instruction{
					Op:    OpEmitUnlessBoundary,
					Value: beforeEndInstr.Value,
					Pos:   beforeEndInstr.Pos,
				}
			}
		}
	}

	// BOUNDARY 命令を追加
	b.instructions = append(b.instructions, Instruction{
		Op:  OpBoundary,
		Pos: "0:0",
	})
}

// applyDialectConversions はトークン列全体に方言変換を適用する
// Step 1: JOIN型の正規化 (LEFT OUTER JOIN → LEFT JOIN など)
// Step 2: その他の方言変換をトークン単位で行う
// Step 3: 変換されたトークン列を返す
func (b *InstructionBuilder) applyDialectConversions(tokens []tokenizer.Token) []tokenizer.Token {
	// Step 1: JOIN型の正規化を先に行う
	normalizedTokens := normalizeJoinType(tokens)

	result := make([]tokenizer.Token, 0, len(normalizedTokens))

	for i := 0; i < len(normalizedTokens); i++ {
		token := normalizedTokens[i]

		// CAST構文の変換: CAST(expr AS type) ⇔ (expr)::type
		if b.shouldConvertCast(token) {
			convertedTokens, skip := b.convertCastSyntaxInTokens(normalizedTokens, i)
			if len(convertedTokens) > 0 {
				result = append(result, convertedTokens...)
				i += skip

				continue
			}
		}

		// 時間関数の変換: NOW() ⇔ CURRENT_TIMESTAMP
		if b.shouldConvertTimeFunction(token) {
			convertedTokens, skip := b.convertTimeFunctionInTokens(normalizedTokens, i)
			if len(convertedTokens) > 0 {
				result = append(result, convertedTokens...)
				i += skip

				continue
			}
		}

		// 日時関数の変換: CURDATE() / CURTIME() ⇔ CURRENT_DATE / CURRENT_TIME
		if b.shouldConvertDateTime(token) {
			convertedTokens, skip := b.convertDateTimeFunctionInTokens(normalizedTokens, i)
			if len(convertedTokens) > 0 {
				result = append(result, convertedTokens...)
				i += skip

				continue
			}
		}

		// 真偽値の変換: PostgreSQL TRUE/FALSE → 1/0
		if b.shouldConvertBoolean(token) {
			convertedToken := b.convertBooleanInTokens(token)
			if convertedToken != nil {
				result = append(result, *convertedToken)
				continue
			}
		}

		// NOTE: COALESCE と IFNULL はすべての対応DB (PostgreSQL, MySQL, SQLite)で
		// 両方サポートされている。functionsigs.go で確認済み。変換不要。

		// 文字列連結の変換: CONCAT() ⇔ ||
		if b.shouldConvertStringConcatenation(token) {
			convertedTokens, skip := b.convertStringConcatenationInTokens(normalizedTokens, i)
			if len(convertedTokens) > 0 {
				result = append(result, convertedTokens...)
				i += skip

				continue
			}
		}

		// 変換不要な場合はそのまま追加
		result = append(result, token)
	}

	return result
}

// shouldConvertCast はCAST構文変換が必要かを判定
func (b *InstructionBuilder) shouldConvertCast(token tokenizer.Token) bool {
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// PostgreSQLの場合: CAST() → ()::
	if b.context.Dialect == snapsql.DialectPostgres && upper == "CAST" {
		return true
	}

	// MySQL/SQLite/MariaDBの場合: ():: → CAST()
	// "::"トークンを検出
	if (b.context.Dialect == snapsql.DialectMySQL || b.context.Dialect == snapsql.DialectSQLite || b.context.Dialect == snapsql.DialectMariaDB) &&
		token.Value == "::" {
		return true
	}

	return false
}

// shouldConvertTimeFunction は時間関数変換が必要かを判定
func (b *InstructionBuilder) shouldConvertTimeFunction(token tokenizer.Token) bool {
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// PostgreSQL/MySQL/MariaDB: CURRENT_TIMESTAMP → NOW()
	if (b.context.Dialect == snapsql.DialectPostgres || b.context.Dialect == snapsql.DialectMySQL || b.context.Dialect == snapsql.DialectMariaDB) &&
		upper == "CURRENT_TIMESTAMP" {
		return true
	}

	// SQLite: NOW() → CURRENT_TIMESTAMP
	if b.context.Dialect == snapsql.DialectSQLite && upper == "NOW" {
		return true
	}

	return false
}

// shouldConvertDateTime は日時関数変換が必要かを判定
// CURDATE() / CURTIME() と CURRENT_DATE / CURRENT_TIME の相互変換
func (b *InstructionBuilder) shouldConvertDateTime(token tokenizer.Token) bool {
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// MySQL: CURDATE() → CURRENT_DATE (PostgreSQL/SQLite)
	if b.context.Dialect == snapsql.DialectPostgres || b.context.Dialect == snapsql.DialectSQLite {
		if upper == "CURDATE" || upper == "CURTIME" {
			return true
		}
	}

	return false
}

// shouldConvertBoolean は真偽値の変換が必要かを判定
// PostgreSQL TRUE/FALSE をMySQL/SQLiteの 1/0 に変換
func (b *InstructionBuilder) shouldConvertBoolean(token tokenizer.Token) bool {
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// MySQL/SQLite: PostgreSQL の TRUE/FALSE → 1/0 に変換
	if (b.context.Dialect == snapsql.DialectMySQL || b.context.Dialect == snapsql.DialectSQLite || b.context.Dialect == snapsql.DialectMariaDB) &&
		(upper == "TRUE" || upper == "FALSE") {
		return true
	}

	return false
}

// shouldConvertStringConcatenation は文字列連結の変換が必要かを判定
// CONCAT() ⇔ || 演算子の相互変換
func (b *InstructionBuilder) shouldConvertStringConcatenation(token tokenizer.Token) bool {
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// PostgreSQL/SQLite: CONCAT() → || 演算子に変換
	if (b.context.Dialect == snapsql.DialectPostgres || b.context.Dialect == snapsql.DialectSQLite) && upper == "CONCAT" {
		return true
	}

	// MySQL/MariaDB: || (tokenizer.CONCAT) → CONCAT() に変換
	if (b.context.Dialect == snapsql.DialectMySQL || b.context.Dialect == snapsql.DialectMariaDB) && token.Type == tokenizer.CONCAT {
		return true
	}

	return false
}

// convertTimeFunctionInTokens は時間関数の複数トークンを変換
// NOW() ⇔ CURRENT_TIMESTAMP を処理する
// 返り値: 変換後のトークン列, スキップするトークン数
func (b *InstructionBuilder) convertTimeFunctionInTokens(tokens []tokenizer.Token, startIndex int) ([]tokenizer.Token, int) {
	token := tokens[startIndex]
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// PostgreSQL/MySQL/MariaDB: CURRENT_TIMESTAMP → NOW()
	if (b.context.Dialect == "postgres" || b.context.Dialect == "mysql" || b.context.Dialect == "mariadb") &&
		upper == "CURRENT_TIMESTAMP" {
		result := []tokenizer.Token{
			{
				Type:     token.Type,
				Value:    "NOW",
				Position: token.Position,
			},
			{
				Type:     tokenizer.OPENED_PARENS,
				Value:    "(",
				Position: token.Position,
			},
			{
				Type:     tokenizer.CLOSED_PARENS,
				Value:    ")",
				Position: token.Position,
			},
		}

		return result, 0 // スキップなし (単一トークン)
	}

	// SQLite: NOW() → CURRENT_TIMESTAMP
	// NOW の後に ( と ) があるかチェック
	if b.context.Dialect == "sqlite" && upper == "NOW" {
		// 次のトークンが ( かチェック
		if startIndex+1 < len(tokens) {
			nextToken := tokens[startIndex+1]
			if nextToken.Type == tokenizer.OPENED_PARENS {
				// さらに次が ) かチェック
				if startIndex+2 < len(tokens) {
					nextNextToken := tokens[startIndex+2]
					if nextNextToken.Type == tokenizer.CLOSED_PARENS {
						// NOW() を CURRENT_TIMESTAMP に変換
						result := []tokenizer.Token{
							{
								Type:     token.Type,
								Value:    "CURRENT_TIMESTAMP",
								Position: token.Position,
							},
						}

						return result, 2 // ( と ) の2トークンをスキップ
					}
				}
			}
		}
	}

	return nil, 0
}

// convertDateTimeFunctionInTokens は日時関数の複数トークンを変換
// CURDATE() / CURTIME() ⇔ CURRENT_DATE / CURRENT_TIME
// 返り値: 変換後のトークン列, スキップするトークン数
func (b *InstructionBuilder) convertDateTimeFunctionInTokens(tokens []tokenizer.Token, startIndex int) ([]tokenizer.Token, int) {
	token := tokens[startIndex]
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// MySQL: CURDATE() → CURRENT_DATE (PostgreSQL/SQLite)
	if upper == "CURDATE" {
		// 次のトークンが ( かチェック
		if startIndex+1 < len(tokens) {
			nextToken := tokens[startIndex+1]
			if nextToken.Type == tokenizer.OPENED_PARENS {
				// さらに次が ) かチェック
				if startIndex+2 < len(tokens) {
					nextNextToken := tokens[startIndex+2]
					if nextNextToken.Type == tokenizer.CLOSED_PARENS {
						// CURDATE() を CURRENT_DATE に変換
						result := []tokenizer.Token{
							{
								Type:     token.Type,
								Value:    "CURRENT_DATE",
								Position: token.Position,
							},
						}

						return result, 2 // ( と ) の2トークンをスキップ
					}
				}
			}
		}
	}

	// MySQL: CURTIME() → CURRENT_TIME (PostgreSQL/SQLite)
	if upper == "CURTIME" {
		// 次のトークンが ( かチェック
		if startIndex+1 < len(tokens) {
			nextToken := tokens[startIndex+1]
			if nextToken.Type == tokenizer.OPENED_PARENS {
				// さらに次が ) かチェック
				if startIndex+2 < len(tokens) {
					nextNextToken := tokens[startIndex+2]
					if nextNextToken.Type == tokenizer.CLOSED_PARENS {
						// CURTIME() を CURRENT_TIME に変換
						result := []tokenizer.Token{
							{
								Type:     token.Type,
								Value:    "CURRENT_TIME",
								Position: token.Position,
							},
						}

						return result, 2 // ( と ) の2トークンをスキップ
					}
				}
			}
		}
	}

	return nil, 0
}

// convertBooleanInTokens は真偽値を変換
// PostgreSQL TRUE/FALSE → 1/0 (MySQL/SQLite)
// 返り値: 変換後のトークン、または nil (変換不要)
func (b *InstructionBuilder) convertBooleanInTokens(token tokenizer.Token) *tokenizer.Token {
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// MySQL/SQLite/MariaDB: TRUE → 1
	if upper == "TRUE" {
		return &tokenizer.Token{
			Type:     token.Type,
			Value:    "1",
			Position: token.Position,
		}
	}

	// MySQL/SQLite/MariaDB: FALSE → 0
	if upper == "FALSE" {
		return &tokenizer.Token{
			Type:     token.Type,
			Value:    "0",
			Position: token.Position,
		}
	}

	return nil
}

// convertStringConcatenationInTokens は文字列連結を変換
// CONCAT() ⇔ || 演算子
// 返り値: 変換後のトークン列, スキップするトークン数
func (b *InstructionBuilder) convertStringConcatenationInTokens(tokens []tokenizer.Token, startIndex int) ([]tokenizer.Token, int) {
	token := tokens[startIndex]
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// PostgreSQL/SQLite: CONCAT(a, b, ...) → a || b || ...
	if (b.context.Dialect == "postgres" || b.context.Dialect == "sqlite") && upper == "CONCAT" {
		// 次のトークンが開き括弧かチェック
		if startIndex+1 >= len(tokens) {
			return nil, 0
		}

		i := startIndex + 1
		// ホワイトスペースをスキップ
		for i < len(tokens) && tokens[i].Type == tokenizer.WHITESPACE {
			i++
		}

		if i >= len(tokens) || tokens[i].Type != tokenizer.OPENED_PARENS {
			return nil, 0
		}

		i++ // 開き括弧をスキップ

		// 括弧内のコンテンツを収集し、引数を分割
		arguments := [][]tokenizer.Token{}
		currentArg := []tokenizer.Token{}
		depth := 1

		for i < len(tokens) && depth > 0 {
			t := tokens[i]
			trimmed := strings.TrimSpace(t.Value)

			if trimmed == "(" {
				depth++

				currentArg = append(currentArg, t)
			} else if trimmed == ")" {
				depth--
				if depth == 0 {
					// 関数の閉じ括弧に到達
					arguments = append(arguments, currentArg)
					break
				}

				currentArg = append(currentArg, t)
			} else if depth == 1 && trimmed == "," {
				// トップレベルのカンマで引数を分割
				arguments = append(arguments, currentArg)
				currentArg = []tokenizer.Token{}
			} else {
				currentArg = append(currentArg, t)
			}

			i++
		}

		// 引数を || で連結
		result := []tokenizer.Token{}

		for argIdx, arg := range arguments {
			// 引数のホワイトスペースをトリム
			trimmedArg := []tokenizer.Token{}

			for _, t := range arg {
				if t.Type != tokenizer.WHITESPACE {
					trimmedArg = append(trimmedArg, t)
				}
			}

			if len(trimmedArg) > 0 {
				result = append(result, trimmedArg...)

				// 最後の引数でない場合は || を追加
				if argIdx < len(arguments)-1 {
					// スペースを追加
					result = append(result, tokenizer.Token{
						Type:     tokenizer.WHITESPACE,
						Value:    " ",
						Position: token.Position,
					})
					result = append(result, tokenizer.Token{
						Type:     tokenizer.CONCAT,
						Value:    "||",
						Position: token.Position,
					})
					result = append(result, tokenizer.Token{
						Type:     tokenizer.WHITESPACE,
						Value:    " ",
						Position: token.Position,
					})
				}
			}
		}

		skipCount := i - startIndex

		return result, skipCount
	}

	// MySQL/MariaDB: a || b || c → CONCAT(a, b, c)
	// || 演算子の場合、この時点では変換しない
	// 理由: || は二項演算子であり、複数の || が連続する場合の処理が複雑
	// より高度な式解析が必要
	if (b.context.Dialect == "mysql" || b.context.Dialect == "mariadb") && token.Type == tokenizer.CONCAT {
		// 現在のバージョンでは || → CONCAT への変換は実装しない
		// （二項演算子の解析が複雑であるため）
		return nil, 0
	}

	return nil, 0
}

// convertCastSyntaxInTokens はCAST構文を含むトークン列を変換
// 返り値: 変換後のトークン列, スキップするトークン数
func (b *InstructionBuilder) convertCastSyntaxInTokens(tokens []tokenizer.Token, startIndex int) ([]tokenizer.Token, int) {
	token := tokens[startIndex]
	upper := strings.ToUpper(strings.TrimSpace(token.Value))

	// PostgreSQL: CAST(expr AS type) → (expr)::type
	if b.context.Dialect == "postgres" && upper == "CAST" {
		return b.convertCastToPostgres(tokens, startIndex)
	}

	// MySQL/SQLite/MariaDB: (expr)::type → CAST(expr AS type)
	// 注意: "::"の前の"(expr)"は既に処理済みなので、ここでは変換をスキップ
	// 代わりに、トークン列全体を先にスキャンして"::"を検出してから変換する必要がある
	// 現時点では複雑すぎるため、この変換はサポートしない

	return nil, 0
}

// convertCastToPostgres は CAST(expr AS type) を (expr)::type に変換
func (b *InstructionBuilder) convertCastToPostgres(tokens []tokenizer.Token, castIndex int) ([]tokenizer.Token, int) {
	// CAST の次のトークンが "(" であることを確認
	i := castIndex + 1
	// ホワイトスペースをスキップ
	for i < len(tokens) && tokens[i].Type == tokenizer.WHITESPACE {
		i++
	}

	if i >= len(tokens) || strings.TrimSpace(tokens[i].Value) != "(" {
		return nil, 0
	}

	i++

	// 式を抽出（ネストした括弧に対応）
	// CAST( の開き括弧から開始しているので depth=1
	// AS または CAST( の閉じ ) まで抽出
	exprTokens := []tokenizer.Token{}

	depth := 1
	for i < len(tokens) && depth > 0 {
		t := tokens[i]

		trimmed := strings.TrimSpace(t.Value)
		if trimmed == "(" {
			depth++

			exprTokens = append(exprTokens, t)
			i++
		} else if trimmed == ")" {
			depth--
			if depth == 0 {
				// CAST( の閉じ ) に到達
				break
			}
			// ネストした括弧の閉じ
			exprTokens = append(exprTokens, t)
			i++
		} else if depth == 1 && strings.ToUpper(trimmed) == "AS" {
			// AS キーワードを見つけたら式の終わり
			break
		} else {
			exprTokens = append(exprTokens, t)
			i++
		}
	}

	// AS キーワードをスキップ
	for i < len(tokens) && strings.ToUpper(strings.TrimSpace(tokens[i].Value)) != "AS" {
		i++
	}

	if i >= len(tokens) {
		return nil, 0
	}

	i++ // AS をスキップ

	// ホワイトスペースをスキップ
	for i < len(tokens) && tokens[i].Type == tokenizer.WHITESPACE {
		i++
	}

	// 型名を抽出
	typeTokens := []tokenizer.Token{}

	depth = 1 // CAST の開き括弧から数えて1
	for i < len(tokens) && depth > 0 {
		t := tokens[i]

		trimmed := strings.TrimSpace(t.Value)
		if trimmed == "(" {
			depth++

			typeTokens = append(typeTokens, t)
			i++
		} else if trimmed == ")" {
			depth--
			if depth == 0 {
				// CAST の閉じ括弧に到達
				// i はCASTの閉じ括弧の位置
				break
			}

			typeTokens = append(typeTokens, t)
			i++
		} else {
			typeTokens = append(typeTokens, t)
			i++
		}
	}

	// 変換: (expr)::type
	// 式トークンからホワイトスペースを除去
	filteredExprTokens := []tokenizer.Token{}

	for _, t := range exprTokens {
		if t.Type != tokenizer.WHITESPACE {
			filteredExprTokens = append(filteredExprTokens, t)
		}
	}

	// 型トークンからホワイトスペースを除去
	filteredTypeTokens := []tokenizer.Token{}

	for _, t := range typeTokens {
		if t.Type != tokenizer.WHITESPACE {
			filteredTypeTokens = append(filteredTypeTokens, t)
		}
	}

	result := []tokenizer.Token{}
	result = append(result, tokenizer.Token{
		Type:     tokenizer.OPENED_PARENS,
		Value:    "(",
		Position: tokens[castIndex].Position,
	})
	result = append(result, filteredExprTokens...)
	result = append(result, tokenizer.Token{
		Type:     tokenizer.CLOSED_PARENS,
		Value:    ")",
		Position: tokens[castIndex].Position,
	})
	result = append(result, tokenizer.Token{
		Type:     tokenizer.DOUBLE_COLON,
		Value:    "::",
		Position: tokens[castIndex].Position,
	})
	result = append(result, filteredTypeTokens...)

	// スキップするトークン数を返す
	// i は CAST の閉じ括弧の位置
	// castIndex は CAST の位置
	// スキップするのは: CAST, (, expr..., AS, type..., ) の全て
	// つまり i - castIndex 個 (CAST自体を含む)
	skipCount := i - castIndex

	return result, skipCount
}

// addStatic は静的な SQL トークンを命令列に追加する
func (b *InstructionBuilder) addStatic(value string, position *tokenizer.Position) {
	instr := Instruction{
		Op:    OpEmitStatic,
		Value: value,
		Pos:   position.String(),
	}
	b.instructions = append(b.instructions, instr)

	if meta := b.currentWhereMeta(); meta != nil {
		meta.RecordStatic(value)
	}
}

// AddInstruction は命令を直接追加する（clause関数用のヘルパー）
func (b *InstructionBuilder) AddInstruction(instr Instruction) {
	b.instructions = append(b.instructions, instr)
}

// AddForLoopStart adds a LOOP_START instruction for array iteration.
// This is used by syntax sugar handlers to directly generate loop instructions.
// This method handles the registration of the collection expression and loop environment internally.
//
// Parameters:
//   - elementVar: string - The loop variable name (e.g., "r" for "for r in rows")
//   - collectionVarName: string - The collection variable name (e.g., "rows")
//   - pos: string - The source position for debugging
//
// Returns: (none, modifies builder state)
func (b *InstructionBuilder) AddForLoopStart(elementVar, collectionVarName, pos string) {
	// Get current environment for expression registration
	currentEnvIndex := b.getCurrentEnvironmentIndex()

	// Register collection expression in the current environment
	collectionExprIndex := b.context.AddExpression(collectionVarName, currentEnvIndex)
	if position, ok := parsePosition(pos); ok {
		descriptor := b.context.LookupTypeDescriptor(pos)
		b.context.SetExpressionMetadata(collectionExprIndex, position, descriptor)
	}

	var whereLoopCond *WhereDynamicCondition

	if meta := b.currentWhereMeta(); meta != nil {
		meta.Dynamic = true
		meta.ExpressionRefs = appendUniqueInt(meta.ExpressionRefs, collectionExprIndex)
		whereLoopCond = &WhereDynamicCondition{
			ExprIndex:        collectionExprIndex,
			NegatedWhenEmpty: true,
			Description:      fmt.Sprintf("loop %s : %s", elementVar, collectionVarName),
		}
		meta.DynamicConditions = append(meta.DynamicConditions, whereLoopCond)
	}

	// Evaluate the collection to extract the first element
	// This allows us to determine the loop variable's type at build time
	collectionValue, err := b.EvaluateExpression(collectionVarName)

	var firstElement any = nil

	if err == nil && collectionValue != nil {
		// Try to extract first element from array
		if arr, ok := collectionValue.([]any); ok && len(arr) > 0 {
			firstElement = arr[0]
		}
	}

	// Create loop environment with the element variable
	// The element variable's value is set to the first element of the collection
	loopEnvIndex := len(b.context.CELEnvironments)
	collectionDescriptor := b.context.LookupTypeDescriptor(pos)
	elementDescriptor := ExtractElementDescriptor(collectionDescriptor)

	loopVarType := DescriptorToTypeString(elementDescriptor)
	if loopVarType == "" {
		loopVarType = "any"
	}

	loopEnv := CELEnvironment{
		Container:   fmt.Sprintf("loop_%d", loopEnvIndex),
		ParentIndex: &currentEnvIndex,
		AdditionalVariables: []CELVariableInfo{
			{
				Name:  elementVar,
				Type:  loopVarType,
				Value: firstElement, // Set to first element from collection
			},
		},
	}
	b.context.AddCELEnvironment(loopEnv)
	b.context.AddEnvironment(elementVar)

	// Add loop to loop stack
	b.loopStack = append(b.loopStack, loopLevel{
		startPos:   pos,
		expression: collectionVarName,
		exprIndex:  collectionExprIndex,
		envIndex:   loopEnvIndex,
		whereCond:  whereLoopCond,
	})

	// Push new environment - automatic environment management
	b.pushEnvironment(loopEnvIndex)

	// Generate LOOP_START instruction
	b.instructions = append(b.instructions, Instruction{
		Op:                  OpLoopStart,
		Variable:            elementVar,
		CollectionExprIndex: &collectionExprIndex,
		EnvIndex:            &loopEnvIndex,
		Pos:                 pos,
	})
}

// AddForLoopEnd adds a LOOP_END instruction to close an array iteration loop.
// This is used by syntax sugar handlers to directly generate loop instructions.
//
// Parameters:
//   - pos: string - The source position for debugging
//
// Returns: (none, modifies builder state)
func (b *InstructionBuilder) AddForLoopEnd(pos string) {
	// Pop environment from stack
	b.popEnvironment()

	// Pop loop from loop stack
	if len(b.loopStack) > 0 {
		b.loopStack = b.loopStack[:len(b.loopStack)-1]
	}

	// Get the environment index to restore (parent environment)
	currentEnvIndex := b.getCurrentEnvironmentIndex()

	// Generate LOOP_END instruction
	b.instructions = append(b.instructions, Instruction{
		Op:       OpLoopEnd,
		EnvIndex: &currentEnvIndex,
		Pos:      pos,
	})
}

// RegisterEmitEval registers an EMIT_EVAL instruction that evaluates and outputs a CEL expression.
// The caller provides the expression string; the builder manages environment and expression indexing.
//
// Parameters:
//   - expr: string - The CEL expression to evaluate
//   - pos: string - The source position for debugging
//
// Returns: (exprIndex int, error)
func (b *InstructionBuilder) RegisterEmitEval(expr string, pos string) (int, error) {
	envIndex := b.getCurrentEnvironmentIndex()
	exprIndex := b.context.AddExpression(expr, envIndex)

	if position, ok := parsePosition(pos); ok {
		b.context.SetExpressionMetadata(exprIndex, position, nil)
	}

	b.instructions = append(b.instructions, Instruction{
		Op:        OpEmitEval,
		Pos:       pos,
		ExprIndex: &exprIndex,
	})

	return exprIndex, nil
}

// RegisterEmitEvalWithDescriptor registers an EMIT_EVAL instruction using the given token for
// position metadata and attaches the provided type descriptor.
func (b *InstructionBuilder) RegisterEmitEvalWithDescriptor(token tokenizer.Token, expr string, descriptor any) (int, error) {
	envIndex := b.getCurrentEnvironmentIndex()
	exprIndex := b.context.AddExpression(expr, envIndex)
	b.annotateExpression(exprIndex, token, descriptor)

	pos := token.Position.String()
	b.instructions = append(b.instructions, Instruction{
		Op:        OpEmitEval,
		Pos:       pos,
		ExprIndex: &exprIndex,
	})

	return exprIndex, nil
}

// RegisterEmitSystemValue registers an EMIT_SYSTEM_VALUE instruction that outputs
// a system field value (created_at, updated_at, etc.).
//
// Parameters:
//   - systemField: string - The system field name (e.g., "created_at", "updated_at")
//   - pos: string - The source position for debugging
//
// Returns: error if operation fails
func (b *InstructionBuilder) RegisterEmitSystemValue(systemField string, pos string) error {
	b.instructions = append(b.instructions, Instruction{
		Op:          OpEmitSystemValue,
		SystemField: systemField,
		Pos:         pos,
	})

	return nil
}

// RegisterEmitUnlessBoundary registers an EMIT_UNLESS_BOUNDARY instruction that outputs
// text unless followed by a boundary delimiter.
//
// Parameters:
//   - text: string - The text to conditionally output
func (b *InstructionBuilder) RegisterEmitUnlessBoundary(text string) {
	b.instructions = append(b.instructions, Instruction{
		Op:    OpEmitUnlessBoundary,
		Value: text,
	})
}

// RegisterBoundary registers a BOUNDARY instruction that marks a boundary for delimiter removal.
func (b *InstructionBuilder) RegisterBoundary() {
	b.instructions = append(b.instructions, Instruction{
		Op: OpBoundary,
	})
}

// addIfSystemLimit registers an IF_SYSTEM_LIMIT instruction with optional expression override.
// If expressionOverride is empty, uses the default value; otherwise evaluates the expression.
//
// Parameters:
//   - defaultValue: string - The default limit value (as string for flexibility)
//   - expressionOverride: string - Optional CEL expression to evaluate (empty if using defaultValue)
//
// Returns: *int - pointer to exprIndex if expression was used, nil otherwise
func (b *InstructionBuilder) addIfSystemLimit() {
	b.instructions = append(b.instructions, Instruction{
		Op: OpIfSystemLimit,
	})
}

// addIfSystemOffset registers an IF_SYSTEM_OFFSET instruction with optional expression override.
// If expressionOverride is empty, uses the default value; otherwise evaluates the expression.
//
// Parameters:
//   - defaultValue: string - The default offset value (as string for flexibility)
//   - expressionOverride: string - Optional CEL expression to evaluate (empty if using defaultValue)
//
// Returns: *int - pointer to exprIndex if expression was used, nil otherwise
func (b *InstructionBuilder) addIfSystemOffset() {
	b.instructions = append(b.instructions, Instruction{
		Op: OpIfSystemOffset,
	})
}

// addEmitSystemLimit registers an EMIT_SYSTEM_LIMIT instruction that outputs the system limit value.
func (b *InstructionBuilder) addEmitSystemLimit() {
	b.instructions = append(b.instructions, Instruction{
		Op: OpEmitSystemLimit,
	})
}

// addEmitSystemOffset registers an EMIT_SYSTEM_OFFSET instruction that outputs the system offset value.
func (b *InstructionBuilder) addEmitSystemOffset() {
	b.instructions = append(b.instructions, Instruction{
		Op: OpEmitSystemOffset,
	})
}

// RegisterEmitSystemFor registers an EMIT_SYSTEM_FOR instruction that outputs the system FOR clause value.
func (b *InstructionBuilder) RegisterEmitSystemFor() {
	b.instructions = append(b.instructions, Instruction{
		Op: OpEmitSystemFor,
	})
}

// CheckEvalResultType determines the type of value that a CEL expression evaluates to.
// This method inspects the CEL environment and variable information to determine whether
// the expression represents a scalar value, an array, an object, or an array of objects.
//
// Parameters:
//   - expr: string - The CEL expression to evaluate
//
// Returns: EvalResultType - The determined type of the expression result
func (b *InstructionBuilder) CheckEvalResultType(expr string) EvalResultType {
	// Get current environment
	envIndex := b.getCurrentEnvironmentIndex()
	if envIndex >= len(b.context.CELEnvironments) {
		return EvalResultTypeUnknown
	}

	// Prefer type information already recorded for the expression
	for _, celExpr := range b.context.Expressions {
		if celExpr.Expression == expr && celExpr.EnvironmentIndex == envIndex {
			if celExpr.ResultType != EvalResultTypeUnknown {
				return celExpr.ResultType
			}

			break
		}
	}

	env := b.context.CELEnvironments[envIndex]

	// Find the variable definition that matches the expression
	var varInfo *CELVariableInfo

	for i := range env.AdditionalVariables {
		if env.AdditionalVariables[i].Name == expr {
			varInfo = &env.AdditionalVariables[i]
			break
		}
	}

	// Variable not found in this environment
	if varInfo == nil {
		return EvalResultTypeUnknown
	}

	// Parse the type string to determine the result type
	varType := varInfo.Type

	// Check for array types (ends with "[]")
	if before, ok := strings.CutSuffix(varType, "[]"); ok {
		baseType := before

		// Check if it's an array of objects
		if isObjectType(baseType) {
			return EvalResultTypeArrayOfObject
		}

		// It's an array of scalars
		return EvalResultTypeArray
	}

	// Check for pointer types (starts with "*")
	if after, ok := strings.CutPrefix(varType, "*"); ok {
		baseType := after

		// Check if it's a pointer to an object
		if isObjectType(baseType) {
			return EvalResultTypeObject
		}

		// Otherwise treat as scalar (pointer to primitive)
		return EvalResultTypeScalar
	}

	// Check if the type itself is an object type
	if isObjectType(varType) {
		return EvalResultTypeObject
	}

	// Default to scalar for known types (any, string, int, bool, etc.)
	return EvalResultTypeScalar
}

// isObjectType determines if a type name represents an object type.
// Object types include custom structs and "any" when used in an object context.
// Scalar types include: any, string, int, int32, int64, uint, uint32, uint64, float, float32, float64, bool, bytes.
func isObjectType(typeName string) bool {
	// Normalize type name (remove pointers and array suffixes)
	normalized := strings.TrimPrefix(typeName, "*")
	normalized = strings.TrimSuffix(normalized, "[]")

	// Known scalar types
	scalarTypes := map[string]bool{
		"any":     true,
		"string":  true,
		"int":     true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint32":  true,
		"uint64":  true,
		"float":   true,
		"float32": true,
		"float64": true,
		"double":  true,
		"bool":    true,
		"bytes":   true,
		"null":    true,
	}

	// If it's not a known scalar type, treat it as an object type
	return !scalarTypes[normalized]
}

// isCommaDelimitedLiteralList は括弧内の値がカンマ区切りのリテラル値かチェック
// パターン: '(', (NUMBER|STRING), ',', (NUMBER|STRING), ... ')'
func (b *InstructionBuilder) isCommaDelimitedLiteralList(tokens []tokenizer.Token, startIdx int) bool {
	if startIdx >= len(tokens) || tokens[startIdx].Type != tokenizer.OPENED_PARENS {
		return false
	}

	i := startIdx + 1
	hasValue := false

	for i < len(tokens) {
		token := tokens[i]

		// ホワイトスペース・コメントはスキップ
		if b.isWhitespaceOrComment(token.Type) {
			i++
			continue
		}

		// 閉じ括弧で終了
		if token.Type == tokenizer.CLOSED_PARENS {
			// リテラル値が存在したか確認
			return hasValue
		}

		// カンマの場合、その後に値が続くかチェック
		if token.Type == tokenizer.COMMA {
			i++
			hasValue = false // カンマの次は値を期待

			continue
		}

		// NUMBER または STRING の場合、リテラル値として認識
		if token.Type == tokenizer.NUMBER || token.Type == tokenizer.STRING {
			hasValue = true
			i++

			continue
		}

		// その他のトークン型は リテラル値リストではない
		return false
	}

	return false
}

// emitInClauseLoop は IN 句のリテラル値リストをループ化した命令列を生成
// 括弧内の値（1, 2, 3）を検出して、自動的に FOR ループに変換
// 戻り値: 処理後のトークンインデックス
func (b *InstructionBuilder) emitInClauseLoop(tokens []tokenizer.Token, startIdx int, varName string) int {
	if startIdx >= len(tokens) || tokens[startIdx].Type != tokenizer.OPENED_PARENS {
		return startIdx
	}

	// Type validation: varName must be a list type
	evalType := b.CheckEvalResultType(varName)
	isListType := evalType == EvalResultTypeArray || evalType == EvalResultTypeArrayOfObject

	if !isListType {
		// If not a list type, emit as static - shouldn't reach here but guard against it
		b.addStatic("(", &tokens[startIdx].Position)
		// Still process tokens to advance index correctly
		i := startIdx + 1
		for i < len(tokens) && tokens[i].Type != tokenizer.CLOSED_PARENS {
			i++
		}

		if i < len(tokens) {
			b.addStatic(")", &tokens[i].Position)
			return i
		}

		return i
	}

	// リスト型の場合、FOR ループ命令を生成
	// ループ変数名を生成
	loopVar := "__item"

	// LOOP_START命令を生成（引数: elementVar, collectionVarName, pos）
	b.AddForLoopStart(loopVar, varName, tokens[startIdx].Position.String())

	// 括弧を出力
	b.addStatic("(", &tokens[startIdx].Position)

	i := startIdx + 1
	valueTokens := []tokenizer.Token{} // リテラル値トークンを収集

	// 括弧内のトークンを走査して、リテラル値とカンマを抽出
	for i < len(tokens) {
		token := tokens[i]

		// ホワイトスペース・コメントはスキップ
		if b.isWhitespaceOrComment(token.Type) {
			i++
			continue
		}

		// 閉じ括弧で終了
		if token.Type == tokenizer.CLOSED_PARENS {
			break
		}

		// カンマの位置を記録
		if token.Type == tokenizer.COMMA {
			i++

			continue
		}

		// NUMBER または STRING の場合、リテラル値として収集
		if token.Type == tokenizer.NUMBER || token.Type == tokenizer.STRING {
			valueTokens = append(valueTokens, token)
			i++

			continue
		}

		// その他のトークンもスキップ
		i++
	}

	// リテラル値を出力（ループ内で各値を処理）
	for idx, token := range valueTokens {
		b.addStatic(token.Value, &token.Position)

		// 最後の値以外の後にはカンマを出力
		if idx < len(valueTokens)-1 {
			b.addStatic(", ", &token.Position)
		}
	}

	// 括弧を閉じる
	if i < len(tokens) && tokens[i].Type == tokenizer.CLOSED_PARENS {
		b.addStatic(")", &tokens[i].Position)
	}

	// LOOP_END命令を生成
	b.AddForLoopEnd(tokens[i].Position.String())

	return i
}

// initializeDummyValuesFromFunctionDefinition は FunctionDefinition からダミー値を生成
// して、ルート CEL 環境に登録する。重複登録を防ぐため、フラグをチェックしてから実行。
func (b *InstructionBuilder) initializeDummyValuesFromFunctionDefinition() {
	if b.dummyValuesInitialized {
		return
	}

	if b.context.FunctionDefinition == nil {
		return
	}

	funcDef := b.context.FunctionDefinition

	// ルート CEL 環境（インデックス 0）を取得
	if len(b.context.CELEnvironments) == 0 {
		return
	}

	rootEnv := &b.context.CELEnvironments[0]

	// FunctionDefinition の各パラメータについてダミー値を生成
	for _, paramName := range funcDef.ParameterOrder {
		paramValue, exists := funcDef.OriginalParameters[paramName]
		if !exists {
			continue
		}

		// パラメータの型文字列を取得
		var typeStr string

		switch v := paramValue.(type) {
		case string:
			typeStr = v
		case map[string]any:
			// 型情報を含むオブジェクト
			if t, ok := v["type"]; ok {
				if typeVal, ok := t.(string); ok {
					typeStr = typeVal
				}
			}
		}

		if typeStr == "" {
			continue
		}

		// ダミー値を生成
		dummyValue := b.generateDummyValueFromType(typeStr)

		// CEL 環境に変数を追加
		varInfo := CELVariableInfo{
			Name:  paramName,
			Type:  typeStr,
			Value: dummyValue,
		}
		rootEnv.AdditionalVariables = append(rootEnv.AdditionalVariables, varInfo)
	}

	// 初期化完了フラグを設定
	b.dummyValuesInitialized = true
}

// generateDummyValueFromType は型文字列からダミー値を生成する
func (b *InstructionBuilder) generateDummyValueFromType(typeStr string) any {
	t := strings.ToLower(strings.TrimSpace(typeStr))

	switch t {
	case "string", "text", "varchar", "str":
		return "dummy"
	case "int":
		return int64(1)
	case "int32":
		return int32(2)
	case "int16":
		return int16(3)
	case "int8":
		return int8(4)
	case "float":
		return 1.1
	case "float32":
		return float32(2.2)
	case "decimal":
		return "1.0"
	case "bool":
		return true
	case "date":
		return "2024-01-01"
	case "datetime":
		return "2024-01-01 00:00:00"
	case "timestamp":
		return "2024-01-02 00:00:00"
	case "email":
		return "user@example.com"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "json":
		return map[string]any{"#": "json"}
	case "any":
		return map[string]any{"#": "any"}
	case "object":
		return map[string]any{"#": "object"}
	}

	// リスト型: int[], string[] 等
	if len(t) > 2 && t[len(t)-2:] == "[]" {
		baseType := t[:len(t)-2]
		return []any{b.generateDummyValueFromType(baseType)}
	}

	return ""
}
