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

// finalizeLimitOffsetClause finalizes the LIMIT and OFFSET clauses.
func finalizeLimitOffsetClause(limitClause *cmn.LimitClause, offsetClause *cmn.OffsetClause, perr *cmn.ParseError) {
	// OFFSET without LIMIT is not allowed (MySQL/SQLite behavior)
	if limitClause == nil && offsetClause != nil {
		perr.Add(fmt.Errorf("%w: OFFSET clause without LIMIT is not allowed", cmn.ErrInvalidSQL))
		return
	}

	var inputs [][]pc.Token[tok.Token]
	var labels []string

	// Add LIMIT clause if present
	if limitClause != nil {
		inputs = append(inputs, cmn.ToParserToken(limitClause.ContentTokens()))
		labels = append(labels, "LIMIT")

		// Check for comma-separated LIMIT/OFFSET (not supported)
		pctx := pc.NewParseContext[tok.Token]()
		_, _, err := commaSeparatedNumber(pctx, inputs[0])
		if err == nil {
			perr.Add(fmt.Errorf("%w at %s: comma-separated LIMIT/OFFSET is not supported by SnapSQL", cmn.ErrInvalidForSnapSQL, inputs[0][0].Val.Position.String()))
			return
		}
	}

	// Add OFFSET clause if present
	if offsetClause != nil {
		inputs = append(inputs, cmn.ToParserToken(offsetClause.ContentTokens()))
		labels = append(labels, "OFFSET")
	}

	// Validate each clause
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
			if labels[i] == "LIMIT" {
				limitClause.Count = num
			} else {
				offsetClause.Count = num
			}
		case "negative-number":
			perr.Add(fmt.Errorf("%w at %s: negative number in %s clause is not supported", cmn.ErrInvalidForSnapSQL, match[0].Val.Position.String(), labels[i]))
		}
	}
}

// finalizeOffsetClause finalizes the OFFSET clause when used without LIMIT.
func finalizeOffsetClause(offsetClause *cmn.OffsetClause, perr *cmn.ParseError) {
	if offsetClause == nil {
		return
	}

	pctx := pc.NewParseContext[tok.Token]()
	input := cmn.ToParserToken(offsetClause.ContentTokens())

	// Validate OFFSET clause content
	_, match, err := number(pctx, input)
	if err != nil {
		if len(input) > 0 {
			perr.Add(fmt.Errorf("%w at %s: invalid number in OFFSET clause", cmn.ErrInvalidSQL, input[0].Val.Position.String()))
		} else {
			perr.Add(fmt.Errorf("%w: OFFSET clause requires number for its content", cmn.ErrInvalidSQL))
		}
		return
	}

	if len(match) > 0 {
		switch match[0].Type {
		case "number":
			// Valid positive number - OK
		case "negative-number":
			perr.Add(fmt.Errorf("%w at %s: negative number in OFFSET clause is not supported", cmn.ErrInvalidForSnapSQL, match[0].Val.Position.String()))
		}
	}
}
