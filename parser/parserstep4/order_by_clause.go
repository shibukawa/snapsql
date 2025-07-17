package parserstep4

import (
	"fmt"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	orderByQualifier = pc.Seq(
		cmn.SP,
		pc.Optional(
			tag("collate", collate, cmn.SP, cmn.Identifier), // COLLATE clause
		),
		cmn.SP,
		pc.Optional(tag("order", pc.Or(
			asc,
			desc,
		))),
		cmn.SP,
		pc.Optional(tag("nulls", pc.Seq(nulls, pc.Or(first, last)))),
		cmn.SP,
		cmn.EOS)

	orderByFieldPrefix = pc.Or(
		tag("invalid-position-index", cmn.Number),
		tag("case", caseKeyword, cmn.SP, whenKeyword, cmn.SP),
		tag("invalid-position-index", cmn.Minus, cmn.SP, cmn.Number),
		tag("function", cmn.Identifier, cmn.ParenOpen, cmn.SP), // function call (only FIELD() is supported)
		tag("cast", cast, cmn.ParenOpen, cmn.SP),               // standard CAST
		tag("field",
			cmn.Identifier,
			pc.Optional(pc.Seq(cmn.Dot, cmn.Identifier)),
			pc.Optional(pc.Seq(cmn.SP, postgreSQLCast, cmn.SP, cmn.Identifier)), // PostgreSQL CAST
			cmn.SP, cmn.EOS),
	)

	fieldNamePattern = pc.Seq(cmn.Identifier, pc.Optional(pc.Seq(cmn.Dot, cmn.Identifier)))
)

func finalizeOrderByClause(clause *cmn.OrderByClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()
	pctx := pc.NewParseContext[tok.Token]()
	pTokens := cmn.ToParserToken(tokens)

	nameToPos := make(map[string][]string)
	for _, part := range fieldIter(pTokens) {
		field := cmn.OrderByField{
			Expression: cmn.ToToken(part.Skipped),
		}
		tokens := part.Skipped
		// Order (ASC/DESC) and COLLATE
		_, qualifiers, consume, _, _ := pc.Find(pctx, orderByQualifier, tokens)
		for _, q := range qualifiers {
			if q.Type == "order" {
				field.Desc = q.Val.Type == tok.DESC
			}
		}
		tokens = tokens[:len(tokens)-consume]

		consume, match, err := orderByFieldPrefix(pctx, tokens)
		tokens = tokens[consume:]
		if err != nil {
			perr.Add(fmt.Errorf("%w at %s: invalid ORDER BY field", cmn.ErrInvalidSQL, tokens[0].Val.Position.String()))
			continue
		}
		switch match[0].Type {
		case "invalid-position-index":
			pos := match[0].Val.Position
			perr.Add(fmt.Errorf("%w at %s: ORDER BY field cannot be a number", cmn.ErrInvalidForSnapSQL, pos.String()))
		case "case":
			fallthrough
		case "cast":
			fallthrough
		case "function":
			if match[0].Type == "function" && !strings.EqualFold(match[0].Val.Value, "field") {
				fi := match[0]
				pos := fi.Val.Position
				perr.Add(fmt.Errorf("%w at %s: ORDER BY function should be 'FIELD()' but %s", cmn.ErrInvalidForSnapSQL, pos.String(), fi.Val.Value))
				continue
			}
			_, fieldIdentifier, err := fieldNamePattern(pctx, tokens)
			if err == nil {
				var fieldName string
				switch len(fieldIdentifier) {
				case 1: // single field
					fieldName = fieldIdentifier[0].Val.Value
					nameToPos[fieldName] = append(nameToPos[fieldName], fieldIdentifier[0].Pos.String())
				case 3: // table.field field
					tableName := fieldIdentifier[0].Val.Value
					field.Field.TableName = tableName
					fullName := tableName + "." + fieldName
					nameToPos[fullName] = append(nameToPos[fullName], fieldIdentifier[0].Pos.String())
				}
				field.Field.Name = fieldName
				clause.Fields = append(clause.Fields, field)
			} else if len(fieldIdentifier) > 0 {
				fi := fieldIdentifier[0]
				pos := fi.Val.Position
				perr.Add(fmt.Errorf("%w at %s: ORDER BY field should be identifier after %s, but %s", cmn.ErrInvalidForSnapSQL, pos.String(), fi.Type, fi.Val.Type))
			} else {
				m := match[0]
				perr.Add(fmt.Errorf("%w at %s: ORDER BY field should have identifier after %s, but empty", cmn.ErrInvalidForSnapSQL, m.Val.Position.String(), m.Type))
			}
		case "field":
			// with table name
			var fieldName string
			if len(match) > 2 && match[1].Val.Type == tok.DOT {
				tableName := match[0].Val.Value
				fieldName = match[2].Val.Value
				field.Field.TableName = tableName
				fullName := tableName + "." + fieldName
				nameToPos[fullName] = append(nameToPos[fullName], match[0].Pos.String())
			} else {
				fieldName = match[0].Val.Value
				nameToPos[fieldName] = append(nameToPos[fieldName], match[0].Pos.String())
			}
			field.Field.Name = fieldName
			clause.Fields = append(clause.Fields, field)
		}
	}
	for name, pos := range nameToPos {
		if len(pos) > 1 {
			perr.Add(fmt.Errorf("%w: duplicate column name '%s' at %s", cmn.ErrInvalidSQL, name, strings.Join(pos, ", ")))
		}
	}
}
