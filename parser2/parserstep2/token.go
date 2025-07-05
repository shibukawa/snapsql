package parserstep2

import (
	pc "github.com/shibukawa/parsercombinator"
	"github.com/shibukawa/snapsql/tokenizer"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

func space() pc.Parser[Entity] {
	return primitiveType("space", tok.WHITESPACE)
}

func comment() pc.Parser[Entity] {
	return primitiveType("comment", tok.BLOCK_COMMENT, tok.LINE_COMMENT)
}

func selectKeyword() pc.Parser[Entity] {
	return ws(primitiveType("select", tok.SELECT))
}

func identifier() pc.Parser[Entity] {
	return primitiveType("identifier", tok.IDENTIFIER)
}

func number() pc.Parser[Entity] {
	return ws(primitiveType("number", tok.NUMBER))
}

func str() pc.Parser[Entity] {
	return ws(primitiveType("string", tok.STRING))
}

func comma() pc.Parser[Entity] {
	return ws(primitiveType("comma", tokenizer.COMMA))
}

func semicolon() pc.Parser[Entity] {
	return ws(primitiveType("semicolon", tokenizer.SEMICOLON))
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

func parenOpen() pc.Parser[Entity] {
	return primitiveType("parenOpen", tokenizer.OPENED_PARENS)
}

func parenClose() pc.Parser[Entity] {
	return primitiveType("parenClose", tokenizer.CLOSED_PARENS)
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

func recursive() pc.Parser[Entity] {
	return ws(primitiveType("recursive", tokenizer.RECURSIVE))
}

func as() pc.Parser[Entity] {
	return ws(primitiveType("as", tokenizer.AS))
}

// Select statement tokens

func selectStatement() pc.Parser[Entity] {
	return ws(primitiveType("select", tokenizer.SELECT))
}

func fromClause() pc.Parser[Entity] {
	return ws(primitiveType("from", tokenizer.FROM))
}

func whereClause() pc.Parser[Entity] {
	return ws(primitiveType("where", tokenizer.WHERE))
}

func havingClause() pc.Parser[Entity] {
	return ws(primitiveType("having", tokenizer.HAVING))
}

func limitClause() pc.Parser[Entity] {
	return ws(primitiveType("limit", tokenizer.LIMIT))
}

func offsetClause() pc.Parser[Entity] {
	return ws(primitiveType("offset", tokenizer.OFFSET))
}

func returningClause() pc.Parser[Entity] {
	return ws(primitiveType("returning", tokenizer.RETURNING))
}

func forClause() pc.Parser[Entity] {
	return ws(primitiveType("for", tokenizer.FOR))
}

// Update statement tokens

func updateStatement() pc.Parser[Entity] {
	return ws(primitiveType("update", tokenizer.UPDATE))
}

func setClause() pc.Parser[Entity] {
	return ws(primitiveType("set", tokenizer.SET))
}
