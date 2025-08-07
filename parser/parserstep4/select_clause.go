package parserstep4

import (
	"fmt"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
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

// finalizeSelectClause parses and validates the SELECT clause, populating Items and checking for asterisk usage.
// It appends errors to perr if asterisk is found.
func finalizeSelectClause(clause *cmn.SelectClause, perr *cmn.ParseError) {
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
			perr.Add(fmt.Errorf("%w at %s: ALL is not allowed in DISTINCT clause", cmn.ErrInvalidSQL, match[0].Val.Position.String()))
			return
		case "distinct-on": // DISTINCT ON (
			clause.Distinct = true
			consume, fields := parseFieldItems(pctx, pTokens)
			clause.DistinctOn = fields
			pTokens = pTokens[consume:]
		}
		// empty or ALL is valid
	}

	nameToPos := make(map[string][]string)

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

		field, fieldTokens := parseFieldQualifier(token.Skipped)

		// Main Part
		// JSON operator: it should be any type
		if _, _, _, _, ok := pc.Find(pctx, jsonOperator, fieldTokens); ok {
			field.FieldKind = cmn.ComplexField
			field.Expression = cmn.ToToken(fieldTokens)
			clause.Fields = append(clause.Fields, field)
			nameToPos[field.FieldName] = append(nameToPos[field.FieldName], field.Pos.String())
			continue
		}

		_, match, err = fieldItemStart(pctx, fieldTokens)
		if err != nil {
			// Skip invalid field tokens and continue processing
			continue
		}
		v := match[0].Val
		switch match[0].Type {
		case "field": // identifier(column name)
			switch len(match) {
			case 1:
				if field.FieldName == "" {
					field.FieldName = v.Value // use identity as alias
				}
				field.FieldKind = cmn.SingleField
				field.TableName = ""
				field.OriginalField = v.Value
				clause.Fields = append(clause.Fields, field)
				nameToPos[field.FieldName] = append(nameToPos[field.FieldName], field.Pos.String())
			case 3:
				if match[2].Val.Type == tok.MULTIPLY {
					// t.*
					p := match[0].Val.Position
					perr.Add(fmt.Errorf("%w: snapsql doesn't allow asterisk (*) at %s in SELECT clause", cmn.ErrInvalidForSnapSQL, p.String()))
				} else {
					// t.field
					if field.FieldName == "" {
						field.FieldName = match[2].Val.Value // use identity as alias
					}
					field.FieldKind = cmn.TableField
					field.TableName = match[0].Val.Value
					field.OriginalField = v.Value + "." + match[2].Val.Value
					clause.Fields = append(clause.Fields, field)
					nameToPos[field.FieldName] = append(nameToPos[field.FieldName], field.Pos.String())
				}
			}
		case "function": // function call
			if !field.ExplicitType {
				if returnType, ok := fixedFunctionReturnTypes[strings.ToLower(match[0].Val.Value)]; ok {
					field.TypeName = returnType
				}
			}
			field.FieldKind = cmn.FunctionField
			field.Expression = cmn.ToToken(match)
			clause.Fields = append(clause.Fields, field)
		case "subquery": // sub query
			field.FieldKind = cmn.ComplexField
			field.Expression = cmn.ToToken(match)
			clause.Fields = append(clause.Fields, field)
		case "asterisk": // asterisk (*)
			p := v.Position
			// エラーメッセージを明確にして、確実にperrに追加
			perr.Add(fmt.Errorf("%w at %s: snapsql doesn't allow asterisk (*) in SELECT clause", cmn.ErrInvalidForSnapSQL, p.String()))
			// 明示的にエラーフラグを設定
			clause.Fields = append(clause.Fields, cmn.SelectField{
				Pos:       p,
				FieldName: "*",
				FieldKind: cmn.InvalidField, // 無効なフィールドとしてマーク
			})
			// 確実にエラーが検出されるようにする
			return
		case "literal": // literal (boolean/number/string/null) or DUMMY_LITERAL
			// Check if this is a DUMMY_LITERAL token (allowed as variable placeholder)
			isDummyLiteral := false
			for _, token := range match {
				if token.Val.Type == tok.DUMMY_LITERAL {
					isDummyLiteral = true
					break
				}
			}

			if isDummyLiteral {
				// DUMMY_LITERAL tokens are allowed as they represent variable directives
				field.FieldKind = cmn.DummyField
				field.Expression = cmn.ToToken(match)
				clause.Fields = append(clause.Fields, field)
			} else {
				// Handle unrecognized field pattern as literal
				field.FieldKind = cmn.LiteralField
				field.Expression = cmn.ToToken(match)
				clause.Fields = append(clause.Fields, field)
			}
		case "not *": // not null/true/false
			p := match[0].Val.Position
			perr.Add(fmt.Errorf("%w at %s: snapsql doesn't allow literal in SELECT clause", cmn.ErrInvalidForSnapSQL, p.String()))
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
				perr.Add(fmt.Errorf("%w at %s: only field name is allowed but %s in SELECT clause points alias", cmn.ErrInvalidSQL, f.Pos.String(), d.Name))
			}
		}
	}

	for name, pos := range nameToPos {
		if len(pos) > 1 {
			perr.Add(fmt.Errorf("%w: duplicate column name '%s' at %s", cmn.ErrInvalidSQL, name, strings.Join(pos, ", ")))
		}
	}
}

func parseFieldQualifier(fieldTokens []pc.Token[tok.Token]) (cmn.SelectField, []pc.Token[tok.Token]) {
	result := cmn.SelectField{
		Pos: fieldTokens[0].Val.Position,
	}
	// Alias
	pctx := pc.NewParseContext[tok.Token]()
	beforeAlias, match, _, _, ok := pc.Find(pctx, alias, fieldTokens)
	if ok {
		switch match[0].Type {
		case "without-as": // it requires before the word
			if len(beforeAlias) > 0 {
				lastType := beforeAlias[len(beforeAlias)-1].Val.Type
				if lastType != tok.DOUBLE_COLON && lastType != tok.DOT {
					result.FieldName = match[0].Val.Value
					fieldTokens = beforeAlias
					result.ExplicitName = true
				}
			}
		case "with-as":
			result.FieldName = match[1].Val.Value
			fieldTokens = beforeAlias
			result.ExplicitName = true
		}
	}

	// Cast
	consume, _, err := standardCastStart(pctx, fieldTokens)
	if err == nil {
		fieldTokens = fieldTokens[consume:]
		// as type)
		fieldTokens, match, _, _, _ = pc.Find(pctx, standardCastEnd, fieldTokens)
		if len(match) > 1 {
			result.TypeName = match[1].Val.Value
			result.ExplicitType = true
		}
	} else {
		var ok bool
		newFieldToken, match, _, _, ok := pc.Find(pctx, postgreSQLCastEnd, fieldTokens)
		if ok {
			switch len(match) {
			case 3: // )::type
				consume, _, _ = postgreSQLCastStart(pctx, newFieldToken)
				fieldTokens = fieldTokens[consume:]
				result.TypeName = match[2].Val.Value
				result.ExplicitType = true
			case 2: // ::type
				fieldTokens = newFieldToken
				result.TypeName = match[1].Val.Value
				result.ExplicitType = true
			}
		}
	}
	return result, fieldTokens
}
