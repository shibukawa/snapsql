package parserstep4

import (
	"fmt"
	"strconv"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	number = pc.Or(
		tag("number", cmn.Number, cmn.SP, cmn.EOS),
		tag("negative-number", pc.Seq(cmn.Minus, cmn.Number, cmn.SP, cmn.EOS)),
	)
	commaSeparatedNumber = pc.Seq(
		pc.Optional(cmn.Minus), cmn.Number, cmn.SP, cmn.Comma, cmn.SP,
		pc.Optional(cmn.Minus), cmn.Number, cmn.SP, cmn.EOS,
	)
)

// FinalizeLimitOffsetClause finalizes the LIMIT and OFFSET clauses.
func FinalizeLimitOffsetClause(limitClause *cmn.LimitClause, offsetClause *cmn.OffsetClause, perr *cmn.ParseError) {
	pctx := pc.NewParseContext[tok.Token]()

	inputs := [][]pc.Token[tok.Token]{
		cmn.ToParserToken(limitClause.ContentTokens()),
	}

	_, _, err := commaSeparatedNumber(pctx, inputs[0])
	if err == nil {
		perr.Add(fmt.Errorf("%w at %s: comma-separated LIMIT/OFFSET is not supported by SnapSQL", cmn.ErrInvalidForSnapSQL, inputs[0][0].Val.Position.String()))
		return
	}

	if offsetClause != nil {
		inputs = append(inputs, cmn.ToParserToken(offsetClause.ContentTokens()))
	}

	labels := []string{"LIMIT", "OFFSET"}
	for i, pTokens := range inputs {
		pctx := pc.NewParseContext[tok.Token]()

		_, match, err := number(pctx, pTokens)
		if err != nil {
			if len(pTokens) > 0 {
				perr.Add(fmt.Errorf("%w at %s: invalid number in %s clause", cmn.ErrInvalidSQL, pTokens[0].Val.Position.String(), labels[i]))
			} else {
				perr.Add(fmt.Errorf("%w: %s clause requires number for its content", cmn.ErrInvalidSQL, labels[i]))
			}
			return
		}
		switch match[0].Type {
		case "number":
			num, _ := strconv.Atoi(match[0].Val.Value)
			if i == 0 {
				limitClause.Count = num
			} else {
				offsetClause.Count = num
			}
		case "negative-number":
			perr.Add(fmt.Errorf("%w at %s: negative number in %s clause is not supported", cmn.ErrInvalidForSnapSQL, match[0].Val.Position.String(), labels[i]))
		}
	}
}
