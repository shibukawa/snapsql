package parserstep2

import (
	"github.com/shibukawa/snapsql/tokenizer"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

// Primitives
var (
	space      = primitiveType("space", tok.WHITESPACE)
	comment    = primitiveType("comment", tok.BLOCK_COMMENT, tok.LINE_COMMENT)
	comma      = ws(primitiveType("comma", tokenizer.COMMA))
	parenOpen  = primitiveType("parenOpen", tokenizer.OPENED_PARENS)
	parenClose = primitiveType("parenClose", tokenizer.CLOSED_PARENS)
)

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
