package codegenerator

import (
	"fmt"

	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

// generateOffsetClause は OFFSET 句から命令列を生成する
// SQLテンプレートに OFFSET が記述されている場合、そのリテラル値をデフォルト値として使用し、
// 実行時にシステムパラメータで上書き可能にする
func generateOffsetClause(clause *parser.OffsetClause, builder *InstructionBuilder) error {
	if clause == nil {
		return fmt.Errorf("%w: OFFSET clause is nil", ErrClauseNil)
	}

	tokens := clause.RawTokens()

	// OFFSET リテラル値を抽出
	var defaultValue string

	for _, token := range tokens {
		if token.Type == tokenizer.NUMBER {
			defaultValue = token.Value
			break
		}
	}

	// OFFSET キーワードとスペースを出力
	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: " OFFSET ",
		Pos:   "0:0",
	})

	// システム OFFSET ブロックを生成
	addSystemOffsetBlock(builder, defaultValue)

	return nil
}

// addSystemOffsetBlock は OFFSET 句のシステム命令ブロックを追加する
// defaultValue が空の場合は IF_SYSTEM_OFFSET でブロック全体を囲む（OFFSET句が存在しない場合）
// defaultValue がある場合は IF_SYSTEM_OFFSET/ELSE/END で分岐（OFFSET句が存在する場合）
func addSystemOffsetBlock(builder *InstructionBuilder, defaultValue string) {
	if defaultValue == "" {
		// OFFSET句がSQLに存在しない場合: IF_SYSTEM_OFFSET で全体を囲む
		// この場合、OFFSET キーワード自体も条件ブロック内に含める必要があるため、
		// この関数は呼ばれない想定（statement_select.goで直接処理）
		// ここでは念のため実装
		builder.AddInstruction(Instruction{
			Op:  OpIfSystemOffset,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpEmitSystemOffset,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpEnd,
			Pos: "0:0",
		})
	} else {
		// OFFSET句がSQLに存在する場合: デフォルト値を使用
		builder.AddInstruction(Instruction{
			Op:  OpIfSystemOffset,
			Pos: "0:0",
		})
		builder.AddInstruction(Instruction{
			Op:  OpEmitSystemOffset,
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

// GenerateSystemOffsetIfNotExists はOFFSET句が存在しない場合のシステム命令を生成する
// statement_select.go から呼び出される
func GenerateSystemOffsetIfNotExists(builder *InstructionBuilder) {
	builder.AddInstruction(Instruction{
		Op:  OpIfSystemOffset,
		Pos: "0:0",
	})
	builder.AddInstruction(Instruction{
		Op:    OpEmitStatic,
		Value: " OFFSET ",
		Pos:   "0:0",
	})
	builder.AddInstruction(Instruction{
		Op:  OpEmitSystemOffset,
		Pos: "0:0",
	})
	builder.AddInstruction(Instruction{
		Op:  OpEnd,
		Pos: "0:0",
	})
}
