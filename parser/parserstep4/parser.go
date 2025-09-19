package parserstep4

import (
	"fmt"
	"iter"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	// Select
	distinct       = cmn.WS2(cmn.PrimitiveType("distinct", tok.DISTINCT))
	all            = cmn.WS2(cmn.PrimitiveType("all", tok.ALL))
	on             = cmn.WS2(cmn.PrimitiveType("on", tok.ON))
	as             = cmn.WS2(cmn.PrimitiveType("as", tok.AS))
	subQuery       = cmn.WS2(cmn.PrimitiveType("select", tok.SELECT))
	cast           = cmn.PrimitiveType("cast", tok.CAST)
	postgreSQLCast = cmn.PrimitiveType("postgresqlCast", tok.DOUBLE_COLON)

	// Type identifiers - can be keywords or identifiers
	typeIdentifier = pc.Or(cmn.Identifier, cmn.Keyword)
	asterisk       = cmn.PrimitiveType("asterisk", tok.MULTIPLY)
	jsonOperator   = cmn.WS2(cmn.PrimitiveType("jsonOperator", tok.JSON_OPERATOR))

	// From
	natural = cmn.WS2(cmn.PrimitiveType("natural", tok.NATURAL))
	left    = cmn.WS2(cmn.PrimitiveType("left", tok.LEFT))
	right   = cmn.WS2(cmn.PrimitiveType("right", tok.RIGHT))
	full    = cmn.WS2(cmn.PrimitiveType("full", tok.FULL))
	inner   = cmn.WS2(cmn.PrimitiveType("inner", tok.INNER))
	outer   = cmn.WS2(cmn.PrimitiveType("outer", tok.OUTER))
	cross   = cmn.WS2(cmn.PrimitiveType("cross", tok.CROSS))
	join    = cmn.WS2(cmn.PrimitiveType("join", tok.JOIN))
	using   = cmn.WS2(cmn.PrimitiveType("using", tok.USING))

	// Order By
	asc     = cmn.WS2(cmn.PrimitiveType("asc", tok.ASC))
	desc    = cmn.WS2(cmn.PrimitiveType("desc", tok.DESC))
	collate = cmn.WS2(cmn.PrimitiveType("collate", tok.COLLATE))

	nulls = cmn.WS2(cmn.KeywordType("nulls", "NULLS"))
	first = cmn.WS2(cmn.KeywordType("first", "FIRST"))
	last  = cmn.WS2(cmn.KeywordType("last", "LAST"))

	// Group By
	rollup   = cmn.WS2(cmn.PrimitiveType("rollup", tok.ROLLUP))
	cube     = cmn.WS2(cmn.PrimitiveType("cube", tok.CUBE))
	grouping = cmn.WS2(cmn.PrimitiveType("grouping", tok.GROUPING))
	sets     = cmn.WS2(cmn.PrimitiveType("sets", tok.SETS))

	// Expression
	caseKeyword = cmn.WS2(cmn.PrimitiveType("case", tok.CASE))
	whenKeyword = cmn.WS2(cmn.PrimitiveType("when", tok.WHEN))
	equal       = cmn.WS2(cmn.PrimitiveType("equal", tok.EQUAL))
)

var (
	alias = pc.Or(
		tag("without-as", pc.Seq(cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS)), // alias without AS
		tag("with-as", pc.Seq(cmn.SP, as, cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS)),
	)

	standardCastStart   = pc.Seq(cast, cmn.ParenOpen, cmn.SP)
	standardCastEnd     = pc.Seq(cmn.SP, as, cmn.SP, typeIdentifier, cmn.SP, cmn.ParenClose, cmn.SP, cmn.EOS)
	postgreSQLCastStart = pc.Seq(cmn.ParenOpen, cmn.SP)
	postgreSQLCastEnd   = pc.Seq(
		pc.Optional(pc.Seq(cmn.SP, cmn.ParenClose)),
		cmn.SP, postgreSQLCast, cmn.SP, typeIdentifier, cmn.SP, cmn.EOS)
)

func tag(typeStr string, p ...pc.Parser[tok.Token]) pc.Parser[tok.Token] {
	return pc.Trans(pc.Seq(p...), func(pctx *pc.ParseContext[tok.Token], src []pc.Token[tok.Token]) (converted []pc.Token[tok.Token], err error) {
		if len(src) > 0 {
			src[0].Type = typeStr
		}

		return src, nil
	})
}

func parseTableName(pctx *pc.ParseContext[tok.Token], tokens []pc.Token[tok.Token], withAlias bool) (int, cmn.TableReference, error) {
	consume, match, err := insertIntoClauseTableName(pctx, tokens)
	if err != nil {
		return 0, cmn.TableReference{}, err
	}

	if len(match) == 0 {
		return 0, cmn.TableReference{}, fmt.Errorf("%w: table name is required", cmn.ErrInvalidSQL)
	}

	tableRef := cmn.TableReference{}

	switch len(match) {
	case 1:
		tableRef.Name = match[0].Val.Value
	case 3:
		tableRef.SchemaName = match[0].Val.Value
		tableRef.Name = match[2].Val.Value
	}

	return consume, tableRef, nil
}

func fieldIter(tokens []pc.Token[tok.Token]) iter.Seq2[int, pc.Consume[tok.Token]] {
	splitter := cmn.WS2(pc.Or(cmn.Comma, cmn.ParenOpen, cmn.ParenClose))

	return func(yield func(index int, consume pc.Consume[tok.Token]) bool) {
		count := 0
		nest := 0

		var skipped []pc.Token[tok.Token]

		for _, part := range pc.FindIter(pc.NewParseContext[tok.Token](), splitter, tokens) {
			if part.Last {
				yield(count, pc.Consume[tok.Token]{
					Consume: part.Consume + len(skipped),
					Skipped: append(skipped, part.Skipped...),
					Match:   nil,
					Last:    true,
				})
			} else if nest > 0 {
				switch part.Match[0].Val.Type {
				case tok.OPENED_PARENS:
					nest++
				case tok.CLOSED_PARENS:
					nest--
				}

				skipped = append(skipped, part.Skipped...)
				skipped = append(skipped, part.Match...)
			} else {
				if part.Match[0].Val.Type == tok.OPENED_PARENS {
					skipped = append(skipped, part.Skipped...)
					skipped = append(skipped, part.Match...)
					nest = 1
				} else {
					yield(count, pc.Consume[tok.Token]{
						Consume: part.Consume + len(skipped),
						Skipped: append(skipped, part.Skipped...),
						Match:   part.Match,
						Last:    part.Last,
					})

					skipped = nil
					count++
				}
			}
		}
	}
}
