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
	distinctQualifier = pc.Or(
		tag("distinct-on", pc.Seq(distinct, on, cmn.WS2(cmn.ParenOpen))),
		tag("distinct-all", pc.Seq(distinct, all)),
		tag("distinct", distinct),
		tag("all", all),
	)

	commaOrParenClose = pc.Or(
		cmn.WS2(cmn.ParenClose),
		cmn.WS2(cmn.Comma),
	)

	field = cmn.WS2(pc.Seq(cmn.Identifier, pc.Optional(pc.Seq(cmn.Dot, cmn.Identifier))))

	fieldItemStart = pc.Or(
		// *
		tag("asterisk", pc.Seq(asterisk, cmn.SP, cmn.EOS)),
		// string, number, boolean, null
		tag("literal", pc.Seq(cmn.Literal, cmn.SP, cmn.EOS)),
		// not null/true/false
		tag("not *", pc.Seq(cmn.Not, pc.Or(cmn.Null, cmn.Boolean), cmn.EOS)),
		// negative number
		tag("literal", pc.Seq(cmn.Minus, cmn.Number, cmn.EOS)),
		// field
		tag("field", pc.Seq(cmn.Identifier,
			pc.Optional(pc.Seq(cmn.Dot, pc.Or(cmn.Identifier, asterisk))),
		)),
		// function call
		tag("function", pc.Seq(cmn.Identifier, cmn.ParenOpen, cmn.SP)),
		// sub query
		tag("subquery", pc.Seq(cmn.ParenOpen, cmn.SP, subQuery)),
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
		switch match[0].Type {
		case "distinct": // DISTINCT
			clause.Distinct = true
		case "distinct-all": // DISTINCT ALL -> error
			perr.Add(fmt.Errorf("%w: ALL is not allowed in DISTINCT clause", ErrDistinctParse))
			return
		case "distinct-on": // DISTINCT ON (
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
		beforeAlias, match, _, _, ok := pc.Find(pctx, alias, fieldTokens)
		if ok {
			switch match[0].Type {
			case "without-as": // it requires before the word
				if len(beforeAlias) > 0 {
					lastType := beforeAlias[len(beforeAlias)-1].Val.Type
					if lastType != tok.DOUBLE_COLON && lastType != tok.DOT {
						fieldName = match[0].Val.Value
						fieldTokens = beforeAlias
						explicitName = true
					}
				}
			case "with-as":
				fieldName = match[1].Val.Value
				fieldTokens = beforeAlias
				explicitName = true
			}
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

		_, match, err = fieldItemStart(pctx, fieldTokens)
		if err != nil {
			continue // todo: panic
		}
		v := match[0].Val
		switch match[0].Type {
		case "field": // identifier(column name)
			switch len(match) {
			case 1:
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
			case 3:
				if match[2].Val.Type == tok.MULTIPLY {
					// t.*
					p := match[0].Val.Position
					perr.Add(fmt.Errorf("%w: snapsql doesn't allow asterisk (*) at %s in SELECT clause", ErrAsteriskInSelect, p.String()))
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
			}
		case "function": // function call
			if !explicitType {
				if returnType, ok := fixedFunctionReturnTypes[strings.ToLower(match[0].Val.Value)]; ok {
					fieldType = returnType
				}
			}
			clause.Fields = append(clause.Fields, cmn.SelectField{
				FieldKind:    cmn.FunctionField,
				Expression:   cmn.ToToken(match),
				FieldName:    fieldName,
				ExplicitName: explicitName,
				TypeName:     fieldType,
				ExplicitType: explicitType,
				Pos:          match[0].Val.Position,
			})
		case "subquery": // sub query
			clause.Fields = append(clause.Fields, cmn.SelectField{
				FieldKind:    cmn.ComplexField,
				Expression:   cmn.ToToken(match),
				FieldName:    fieldName,
				ExplicitName: explicitName,
				TypeName:     fieldType,
				ExplicitType: explicitType,
				Pos:          match[0].Val.Position,
			})
		case "asterisk": // asterisk (*)
			p := v.Position
			perr.Add(fmt.Errorf("%w: snapsql doesn't allow asterisk (*) at %d:%d in SELECT clause", ErrAsteriskInSelect, p.Line, p.Column))
		case "literal": // literal (boolean/number/string/null)
			p := v.Position
			perr.Add(fmt.Errorf("%w: snapsql doesn't allow literal('%s') at %s in SELECT clause", ErrLiteralInSelect, v.Value, p.String()))
		case "not *": // not null/true/false
			p := match[0].Val.Position
			perr.Add(fmt.Errorf("%w: snapsql doesn't allow literal at %s in SELECT clause", ErrLiteralInSelect, p.String()))
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
