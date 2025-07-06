package parserstep2

import (
	pc "github.com/shibukawa/parsercombinator"
	"github.com/shibukawa/snapsql/tokenizer"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// Primitives
var (
	space      = primitiveType("space", tok.WHITESPACE)
	comment    = primitiveType("comment", tok.BLOCK_COMMENT, tok.LINE_COMMENT)
	identifier = primitiveType("identifier", tok.IDENTIFIER)
	comma      = ws(primitiveType("comma", tokenizer.COMMA))
	semicolon  = ws(primitiveType("semicolon", tokenizer.SEMICOLON))
	parenOpen  = primitiveType("parenOpen", tokenizer.OPENED_PARENS)
	parenClose = primitiveType("parenClose", tokenizer.CLOSED_PARENS)
)

func number() pc.Parser[Entity] {
	return ws(primitiveType("number", tok.NUMBER))
}

func str() pc.Parser[Entity] {
	return ws(primitiveType("string", tok.STRING))
}

func between() pc.Parser[Entity] {
	return ws(primitiveType("between", tokenizer.BETWEEN))
}

func andOp() pc.Parser[Entity] {
	return ws(primitiveType("and", tokenizer.AND))
}

func null() pc.Parser[Entity] {
	return ws(primitiveType("null", tokenizer.NULL))
}

func not() pc.Parser[Entity] {
	return ws(primitiveType("not", tokenizer.NOT))
}

func minus() pc.Parser[Entity] {
	return ws(primitiveType("minus", tokenizer.MINUS))
}

func similar() pc.Parser[Entity] {
	return ws(primitiveType("similar", tokenizer.SIMILAR))
}

func to() pc.Parser[Entity] {
	return ws(primitiveType("to", tokenizer.TO))
}

// dot parses dot operator without ws wrapper (no spaces allowed)
func dot() pc.Parser[Entity] {
	return ws(primitiveType("dot", tokenizer.DOT))
}

// CTE tokens

var (
	recursive = ws(primitiveType("recursive", tokenizer.RECURSIVE))
	as        = ws(primitiveType("as", tokenizer.AS))
)

// Select statement tokens

var (
	selectStatement = ws(primitiveType("select", tokenizer.SELECT))
	fromClause      = ws(primitiveType("from", tokenizer.FROM))
	whereClause     = ws(primitiveType("where", tokenizer.WHERE))
	havingClause    = ws(primitiveType("having", tokenizer.HAVING))
	limitClause     = ws(primitiveType("limit", tokenizer.LIMIT))
	offsetClause    = ws(primitiveType("offset", tokenizer.OFFSET))
	returningClause = ws(primitiveType("returning", tokenizer.RETURNING))
	forClause       = ws(primitiveType("for", tokenizer.FOR))
)

// Update statement tokens

var (
	updateStatement = ws(primitiveType("update", tokenizer.UPDATE))
	setClause       = ws(primitiveType("set", tokenizer.SET))
)
