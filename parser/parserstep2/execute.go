package parserstep2

import (
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// Execute is the entry point for parserstep2. It parses a token slice and returns a StatementNode.
// Execute parses a slice of tokenizer.Token and returns a StatementNode and error.
func Execute(tokens []tok.Token) (cmn.StatementNode, error) {
	entityTokens := tokenToEntity(tokens)
	pctx := pc.NewParseContext[Entity]()
	_, parsed, err := ParseStatement()(pctx, entityTokens)
	if err != nil {
		return nil, err
	}
	if len(parsed) == 0 || parsed[0].Val.NewValue == nil {
		return nil, pc.ErrNotMatch
	}
	node, ok := parsed[0].Val.NewValue.(cmn.StatementNode)
	if !ok {
		return nil, pc.ErrNotMatch
	}
	return node, nil
}
