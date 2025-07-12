package parserstep4

import (
	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
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
	thenKeyword = cmn.WS2(cmn.PrimitiveType("then", tok.THEN))
	elseKeyword = cmn.WS2(cmn.PrimitiveType("else", tok.ELSE))
	endKeyword  = cmn.WS2(cmn.PrimitiveType("end", tok.END))
)

var (
	alias = pc.Or(
		pc.Seq(cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS),             // alias without AS
		pc.Seq(cmn.SP, as, cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS), // alias with AS
	)
)

func tag(typeStr string, p ...pc.Parser[tok.Token]) pc.Parser[tok.Token] {
	return pc.Trans(pc.Seq(p...), func(pctx *pc.ParseContext[tok.Token], src []pc.Token[tok.Token]) (converted []pc.Token[tok.Token], err error) {
		if len(src) > 0 {
			src[0].Type = typeStr
		}
		return src, nil
	})
}
