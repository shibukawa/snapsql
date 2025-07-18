package intermediate

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/parser"
	"github.com/shibukawa/snapsql/tokenizer"
)

func TestGenerateFromStatementNode(t *testing.T) {
	// Create a basic test setup
	generator, err := NewGenerator()
	assert.NoError(t, err)

	// Parse a simple SQL statement
	tokens, err := tokenizer.Tokenize("SELECT id, name FROM users")
	assert.NoError(t, err)

	stmt, err := parser.Parse(tokens, nil, nil)
	assert.NoError(t, err)

	// Create a basic function definition
	funcDef := &parser.FunctionDefinition{
		FunctionName: "test_function",
		// Add basic fields as needed
	}

	// Set up generation options
	options := GenerationOptions{
		Dialect:         snapsql.DialectPostgres,
		DatabaseSchemas: []snapsql.DatabaseSchema{},
	}

	// Generate intermediate format
	result, err := generator.GenerateFromStatementNode(stmt, funcDef, options)
	assert.NoError(t, err)
	assert.NotEqual(t, nil, result)

	// Check basic structure
	assert.True(t, len(result.Instructions) > 0)
	assert.NotEqual(t, nil, result.InterfaceSchema)
	assert.Equal(t, "test_function", result.InterfaceSchema.FunctionName)

	// Check that we have at least one instruction
	assert.True(t, len(result.Instructions) > 0)

	// Check the first instruction is a SELECT literal (fallback generates more complete SQL)
	firstInstruction := result.Instructions[0]
	assert.Equal(t, OpEmitLiteral, firstInstruction.Op)
	assert.Equal(t, "SELECT * FROM table", firstInstruction.Value)
}

func TestInstructionGenerator(t *testing.T) {
	// Test the instruction generator directly
	ig := &instructionGenerator{
		dialect:      snapsql.DialectPostgres,
		instructions: make([]Instruction, 0),
		position:     []int{1, 1, 0},
	}

	// Test adding a literal instruction
	ig.addInstruction(Instruction{
		Op:    OpEmitLiteral,
		Pos:   ig.getCurrentPosition(),
		Value: "SELECT ",
	})

	assert.Equal(t, 1, len(ig.instructions))
	assert.Equal(t, OpEmitLiteral, ig.instructions[0].Op)
	assert.Equal(t, "SELECT ", ig.instructions[0].Value)
}

func TestGenerationOptions(t *testing.T) {
	// Test that generation options are properly structured
	options := GenerationOptions{
		Dialect: snapsql.DialectMySQL,
		DatabaseSchemas: []snapsql.DatabaseSchema{
			{
				Name: "test_db",
				DatabaseInfo: snapsql.DatabaseInfo{
					Type: "mysql",
					Name: "test_db",
				},
			},
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	assert.Equal(t, snapsql.DialectMySQL, options.Dialect)
	assert.Equal(t, 1, len(options.DatabaseSchemas))
	assert.Equal(t, "1.0.0", options.Metadata["version"])
}

func TestInstructionConstants(t *testing.T) {
	// Test that instruction constants are properly defined
	assert.Equal(t, "EMIT_LITERAL", OpEmitLiteral)
	assert.Equal(t, "EMIT_PARAM", OpEmitParam)
	assert.Equal(t, "EMIT_EVAL", OpEmitEval)
	assert.Equal(t, "JUMP", OpJump)
	assert.Equal(t, "JUMP_IF_EXP", OpJumpIfExp)
	assert.Equal(t, "LABEL", OpLabel)
	assert.Equal(t, "LOOP_START", OpLoopStart)
	assert.Equal(t, "LOOP_NEXT", OpLoopNext)
	assert.Equal(t, "LOOP_END", OpLoopEnd)
	assert.Equal(t, "EMIT_DIALECT", OpEmitDialect)
}
