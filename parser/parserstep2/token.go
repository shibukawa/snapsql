package parserstep2

import (
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// Primitives
var (
	space      = primitiveType("space", tok.WHITESPACE)
	comment    = primitiveType("comment", tok.BLOCK_COMMENT, tok.LINE_COMMENT)
	comma      = ws(primitiveType("comma", tok.COMMA))
	parenOpen  = primitiveType("parenOpen", tok.OPENED_PARENS)
	parenClose = primitiveType("parenClose", tok.CLOSED_PARENS)
)

// CTE tokens

var (
	recursive = ws(primitiveType("recursive", tok.RECURSIVE))
	as        = ws(primitiveType("as", tok.AS))
)

// Select statement tokens

var (
	selectStatement = ws(primitiveType("select", tok.SELECT))
	fromClause      = ws(primitiveType("from", tok.FROM))
	whereClause     = ws(primitiveType("where", tok.WHERE))
	havingClause    = ws(primitiveType("having", tok.HAVING))
	limitClause     = ws(primitiveType("limit", tok.LIMIT))
	offsetClause    = ws(primitiveType("offset", tok.OFFSET))
	returningClause = ws(primitiveType("returning", tok.RETURNING))
	forClause       = ws(primitiveType("for", tok.FOR))
)

// Update statement tokens

var (
	updateStatement = ws(primitiveType("update", tok.UPDATE))
	setClause       = ws(primitiveType("set", tok.SET))
)
