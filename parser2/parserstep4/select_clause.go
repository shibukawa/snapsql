package parserstep4

import (
	"errors"
	"fmt"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser2/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	ErrDistinctParse    = errors.New("error at parsing DISTINCT qualifier")
	ErrAsteriskInSelect = errors.New("asterisk (*) is not allowed in SELECT clause")
	ErrLiteralInSelect  = errors.New("literal (boolean/number/string/null) is not allowed in SELECT clause")
	ErrDistinctOnAlias  = errors.New("alias name is not allowed in DISTINCT ON list")
)

var (
	distinct       = cmn.WS2(cmn.PrimitiveType("distinct", tok.DISTINCT))
	all            = cmn.WS2(cmn.PrimitiveType("all", tok.ALL))
	on             = cmn.WS2(cmn.PrimitiveType("on", tok.ON))
	as             = cmn.WS2(cmn.PrimitiveType("as", tok.AS))
	subQuery       = cmn.WS2(cmn.PrimitiveType("select", tok.SELECT))
	cast           = cmn.PrimitiveType("cast", tok.CAST)
	postgreSQLCast = cmn.PrimitiveType("postgresqlCast", tok.DOUBLE_COLON)
	asterisk       = cmn.PrimitiveType("asterisk", tok.MULTIPLY)
	jsonOperator   = cmn.WS2(cmn.PrimitiveType("jsonOperator", tok.JSON_OPERATOR))

	distinctQualifier = pc.Or(
		pc.Seq(distinct, on, cmn.WS2(cmn.ParenOpen)),
		pc.Seq(distinct, all),
		distinct,
		all,
	)

	commaOrParenClose = pc.Or(
		cmn.WS2(cmn.ParenClose),
		cmn.WS2(cmn.Comma),
	)

	field = cmn.WS2(pc.Seq(cmn.Identifier, pc.Optional(pc.Seq(cmn.Dot, cmn.Identifier))))

	fieldItemStart = pc.Or(
		// *
		pc.Seq(asterisk, cmn.SP, cmn.EOS),
		// string, number, boolean, null
		pc.Seq(cmn.Literal, cmn.SP, cmn.EOS),
		// not null/true/false
		pc.Seq(cmn.Not, pc.Or(cmn.Null, cmn.Boolean), cmn.EOS),
		// negative number
		pc.Seq(cmn.Minus, cmn.Number, cmn.EOS),
		// field
		pc.Seq(cmn.Identifier,
			pc.Optional(pc.Seq(cmn.Dot, pc.Or(cmn.Identifier, asterisk))),
		),
		// function call
		pc.Seq(cmn.Identifier, cmn.ParenOpen, cmn.SP),
		// sub query
		pc.Seq(cmn.ParenOpen, cmn.SP, subQuery),
	)
	alias = pc.Or(
		pc.Seq(cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS),             // alias without AS
		pc.Seq(cmn.SP, as, cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS), // alias with AS
	)
	standardCastStart   = pc.Seq(cast, cmn.ParenOpen, cmn.SP)
	standardCastEnd     = pc.Seq(cmn.SP, as, cmn.SP, cmn.Identifier, cmn.SP, cmn.ParenClose, cmn.SP, cmn.EOS)
	postgreSQLCastStart = pc.Seq(cmn.ParenOpen, cmn.SP)
	postgreSQLCastEnd   = pc.Seq(
		pc.Optional(pc.Seq(cmn.SP, cmn.ParenClose)),
		cmn.SP, postgreSQLCast, cmn.SP, cmn.Identifier, cmn.SP, cmn.EOS)
)

func parseFieldItems(pctx *pc.ParseContext[tok.Token], tokens []pc.Token[tok.Token]) (int, []cmn.FieldName) {
	consume := 0
	var fields []cmn.FieldName
	for _, part := range pc.FindIter(pctx, commaOrParenClose, tokens) {
		consume = part.Consume + len(part.Skipped)
		if len(part.Skipped) == 0 {
			continue
		}
		_, f, err := field(pctx, part.Skipped)
		if err == nil {
			switch len(f) {
			case 1: // single field
				fields = append(fields, cmn.FieldName{
					Name: f[0].Val.Value,
					Pos:  f[0].Val.Position,
				})
			case 3: // field with dot notation
				fields = append(fields, cmn.FieldName{
					TableName: f[0].Val.Value,
					Name:      f[2].Val.Value,
					Pos:       f[0].Val.Position,
				})
			}
		}
		if part.Match[0].Val.Type == tok.CLOSED_PARENS {
			break
		}
	}
	return consume, fields
}

// FinalizeSelectClause parses and validates the SELECT clause, populating Items and checking for asterisk usage.
// It appends errors to perr if asterisk is found.
func FinalizeSelectClause(clause *cmn.SelectClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()

	pctx := pc.NewParseContext[tok.Token]()
	pTokens := cmn.ToParserToken(tokens)

	consume, match, err := distinctQualifier(pctx, pTokens)
	// DISTINCT found
	if err == nil {
		pTokens = pTokens[consume:]
		switch len(match) {
		case 1: // DISTINCT or ALL
			if match[0].Val.Type == tok.DISTINCT {
				clause.Distinct = true
			}
		case 2: // DISTINCT ALL -> error
			perr.Add(fmt.Errorf("%w: ALL is not allowed in DISTINCT clause", ErrDistinctParse))
			return
		case 3: // DISTINCT ON (
			clause.Distinct = true
			consume, fields := parseFieldItems(pctx, pTokens)
			clause.DistinctOn = fields
			pTokens = pTokens[consume:]
		}
		// empty or ALL is valid
	}

	// field analysis
	// supported::
	//   1. type casts
	//   2. alias
	//   3. identifier(column name)
	//   4. table.column (qualified identifier)
	//   5. asterisk (*) -> error
	//   6. function calls(aggregate functions, scalar functions, distinct)
	//
	// passively supported:
	//   7. sub query
	//   8. expressions
	//   9. literal values
	//   10. JSON
	for _, token := range pc.FindIter(pctx, cmn.WS2(cmn.Comma), pTokens) {
		if len(token.Skipped) == 0 {
			continue
		}

		fieldTokens := token.Skipped

		// Alias
		var fieldName string
		var explicitName bool
		beforeAlias, match, _, _, _ := pc.Find(pctx, alias, fieldTokens)
		switch len(match) {
		case 1: // alias without AS, but it requires before the word
			if len(beforeAlias) > 0 {
				lastType := beforeAlias[len(beforeAlias)-1].Val.Type
				if lastType != tok.DOUBLE_COLON && lastType != tok.DOT {
					fieldName = match[0].Val.Value
					fieldTokens = beforeAlias
					explicitName = true
				}
			}
		case 2:
			fieldName = match[1].Val.Value
			fieldTokens = beforeAlias
			explicitName = true
		}

		// Cast
		var fieldType string
		var explicitType bool
		consume, _, err := standardCastStart(pctx, fieldTokens)
		if err == nil {
			fieldTokens = fieldTokens[consume:]
			// as type)
			fieldTokens, match, _, _, _ = pc.Find(pctx, standardCastEnd, fieldTokens)
			fieldType = match[1].Val.Value
			explicitType = true
		} else {
			var ok bool
			newFieldToken, match, _, _, ok := pc.Find(pctx, postgreSQLCastEnd, fieldTokens)
			if ok {
				switch len(match) {
				case 3: // )::type
					consume, _, _ = postgreSQLCastStart(pctx, newFieldToken)
					fieldTokens = fieldTokens[consume:]
					fieldType = match[2].Val.Value
					explicitType = true
				case 2: // ::type
					fieldTokens = newFieldToken
					fieldType = match[1].Val.Value
					explicitType = true
				}
			}
		}

		// Main Part
		// JSON operator: it should be any type
		if _, _, _, _, ok := pc.Find(pctx, jsonOperator, fieldTokens); ok {
			clause.Fields = append(clause.Fields, cmn.SelectField{
				FieldKind:    cmn.ComplexField,
				Expression:   cmn.ToToken(fieldTokens),
				FieldName:    fieldName,
				ExplicitName: explicitName,
				TypeName:     fieldType,
				ExplicitType: explicitType,
				Pos:          fieldTokens[0].Val.Position,
			})
			continue
		}

		consume, match, err = fieldItemStart(pctx, fieldTokens)
		if err != nil {
			continue // todo: panic
		}
		switch len(match) {
		case 1: // asterisk (*), identity
			v := match[0].Val
			switch v.Type {
			case tok.MULTIPLY:
				p := v.Position
				perr.Add(fmt.Errorf("%w: snapsql doesn't allow asterisk (*) at %d:%d in SELECT clause", ErrAsteriskInSelect, p.Line, p.Column))
			case tok.IDENTIFIER:
				if fieldName == "" {
					fieldName = v.Value // use identity as alias
				}
				clause.Fields = append(clause.Fields, cmn.SelectField{
					FieldKind:    cmn.SingleField,
					Expression:   cmn.ToToken(match[:1]),
					FieldName:    fieldName,
					ExplicitName: explicitName,
					TypeName:     fieldType,
					ExplicitType: explicitType,
					Pos:          v.Position,
				})
			default: // literal
				p := v.Position
				perr.Add(fmt.Errorf("%w: snapsql doesn't allow literal at %d:%d in SELECT clause", ErrLiteralInSelect, p.Line, p.Column))
			}
		case 2: // sub query, function call
			// function call
			var fieldKind cmn.FieldType
			if match[0].Val.Type == tok.NOT || match[0].Val.Type == tok.MINUS {
				p := match[0].Val.Position
				perr.Add(fmt.Errorf("%w: snapsql doesn't allow literal at %d:%d in SELECT clause", ErrLiteralInSelect, p.Line, p.Column))
			}
			if match[1].Val.Type == tok.OPENED_PARENS {

				if !explicitType {
					if returnType, ok := fixedFunctionReturnTypes[strings.ToLower(match[0].Val.Value)]; ok {
						fieldType = returnType
					}
				}
				fieldKind = cmn.FunctionField
			} else {
				fieldKind = cmn.ComplexField
			}
			clause.Fields = append(clause.Fields, cmn.SelectField{
				FieldKind:    fieldKind,
				Expression:   cmn.ToToken(match),
				FieldName:    fieldName,
				ExplicitName: explicitName,
				TypeName:     fieldType,
				ExplicitType: explicitType,
				Pos:          match[0].Val.Position,
			})
		case 3: // identity with table name
			if match[2].Val.Type == tok.MULTIPLY {
				// t.*
				p := match[0].Val.Position
				perr.Add(fmt.Errorf("%w: snapsql doesn't allow asterisk (*) at %d:%d in SELECT clause", ErrAsteriskInSelect, p.Line, p.Column))
			} else {
				// t.field
				clause.Fields = append(clause.Fields, cmn.SelectField{
					FieldKind:    cmn.TableField,
					Expression:   cmn.ToToken(match[:3]),
					FieldName:    fieldName,
					ExplicitName: explicitName,
					TypeName:     fieldType,
					ExplicitType: explicitType,
					Pos:          match[0].Val.Position,
				})
			}
		default:
			panic(fmt.Sprintf("unexpected match length: %d", len(match)))
		}
	}

	for _, d := range clause.DistinctOn {
		if d.TableName != "" {
			continue
		}
		for _, f := range clause.Fields {
			if f.FieldName == d.Name && f.ExplicitName {
				perr.Add(fmt.Errorf("%w: only field name is allowed but %s at %d:%d in SELECT clause points alias at %d:%d", ErrDistinctOnAlias,
					d.Name, d.Pos.Line, d.Pos.Column, f.Pos.Line, f.Pos.Column))
			}
		}
	}
}
