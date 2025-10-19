package codegenerator

import (
	"errors"
	"fmt"

	"github.com/shibukawa/snapsql/parser"
)

// generateSetClause は SET 節から命令を生成する
//
// SET節の構造:
//
//	SET column1 = value1, column2 = value2, ...
//
// Parameters:
//   - clause: *parser.SetClause（必須）
//   - builder: *InstructionBuilder
//
// Returns:
//   - error: エラー
//
// 備考:
//   - システムフィールド（updated_at等）はこの関数呼び出し後に別途追加される
func generateSetClause(clause *parser.SetClause, builder *InstructionBuilder) error {
	if clause == nil {
		return errors.New("SET clause is required")
	}

	// SET トークンを処理
	tokens := clause.RawTokens()
	if err := builder.ProcessTokens(tokens); err != nil {
		return fmt.Errorf("code generation: %w", err)
	}

	// 既存のアサインメントからフィールド名を抽出（重複排除用）
	existingFields := make(map[string]bool)
	for _, assign := range clause.Assigns {
		existingFields[assign.FieldName] = true
	}

	// システムフィールドを SET に追加（重複排除）
	fields := getUpdateSystemFieldsFiltered(builder.context, existingFields)
	appendSystemFieldUpdates(builder, fields)

	return nil
}
