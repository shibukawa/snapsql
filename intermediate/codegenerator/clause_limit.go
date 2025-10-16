package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// generateLimitClause は LIMIT 句から命令列を生成する
// SQLテンプレートに LIMIT が記述されている場合、そのリテラル値をデフォルト値として使用し、
// 実行時にシステムパラメータで上書き可能にする
func generateLimitClause(clause *parser.LimitClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: LIMIT clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()

	// LIMIT リテラル値を抽出
	var defaultValue string

	for _, token := range tokens {
		if token.Type == tokenizer.NUMBER {
			defaultValue = token.Value
			break
		}
	}

	// LIMIT キーワードとスペースを出力
	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: " LIMIT ",
		Pos:   "0:0",
	})

	// システム LIMIT ブロックを生成
	addSystemLimitBlock(builder, defaultValue)

	return nil
}

// addSystemLimitBlock は LIMIT 句のシステム命令ブロックを追加する
// defaultValue が空の場合は IF_SYSTEM_LIMIT でブロック全体を囲む（LIMIT句が存在しない場合）
// defaultValue がある場合は IF_SYSTEM_LIMIT/ELSE/END で分岐（LIMIT句が存在する場合）
func addSystemLimitBlock(builder *InstructionBuilder, defaultValue string) {
	if defaultValue == "" {
		// LIMIT句がSQLに存在しない場合: IF_SYSTEM_LIMIT で全体を囲む
		// この場合、LIMIT キーワード自体も条件ブロック内に含める必要があるため、
		// この関数は呼ばれない想定（statement_select.goで直接処理）
		// ここでは念のため実装
		builder.AddInstruction(Instruction{
			Op:  OpIfSystemLimit,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpEmitSystemLimit,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpEnd,
			Pos: "0:0",
		})
	} else {
		// LIMIT句がSQLに存在する場合: デフォルト値を使用
		builder.AddInstruction(Instruction{
			Op:  OpIfSystemLimit,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpEmitSystemLimit,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpElse,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:    OpEmitStatic,
			Value: defaultValue,
			Pos:   "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpEnd,
			Pos: "0:0",
		})
	}
}

// GenerateSystemLimitIfNotExists はLIMIT句が存在しない場合のシステム命令を生成する
// statement_select.go から呼び出される
func GenerateSystemLimitIfNotExists(builder *InstructionBuilder) {
	builder.AddInstruction(Instruction{
		Op:  OpIfSystemLimit,
		Pos: "0:0",
	})
	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: " LIMIT ",
		Pos:   "0:0",
	})
	builder.AddInstruction(Instruction{
		Op:  OpEmitSystemLimit,
		Pos: "0:0",
	})
	builder.AddInstruction(Instruction{
		Op:  OpEnd,
		Pos: "0:0",
	})
}
