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
)

var (
	alias = pc.Or(
		pc.Seq(cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS),             // alias without AS
		pc.Seq(cmn.SP, as, cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS), // alias with AS
	)
)
