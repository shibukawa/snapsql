package parserstep4

import (
	"fmt"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	groupByAdvanced = pc.Or(
		tag("null", pc.Seq(cmn.Null, cmn.SP, cmn.EOS)),
		tag("advanced", pc.Seq(rollup, cmn.SP)),
		tag("advanced", pc.Seq(cube, cmn.SP)),
		tag("advanced", pc.Seq(grouping, sets, cmn.SP)),
	)

	groupByFields = pc.Or(
		tag("field", cmn.Identifier, pc.Optional(pc.Seq(cmn.Dot, cmn.Identifier)), cmn.SP, cmn.EOS),
		tag("index", cmn.Number, cmn.SP, cmn.EOS),
		tag("case", caseKeyword, cmn.SP, whenKeyword, cmn.SP,
			cmn.Identifier, pc.Optional(pc.Seq(cmn.Dot, cmn.Identifier))),
	)

	groupBySplitter = pc.Or(
		cmn.WS2(cmn.Comma),
		cmn.WS2(cmn.ParenOpen),
		cmn.WS2(cmn.ParenClose),
	)
)

func finalizeGroupByClause(clause *cmn.GroupByClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()
	pctx := pc.NewParseContext[tok.Token]()
	pTokens := cmn.ToParserToken(tokens)

	if consume, match, err := groupByAdvanced(pctx, pTokens); err == nil {
		switch match[0].Type {
		case "null":
			clause.Null = true
			return
		case "advanced":
			clause.AdvancedGrouping = true
			pTokens = pTokens[consume:]
		}
	}

	nameToPos := make(map[string][]string)

	for _, part := range pc.FindIter(pctx, groupBySplitter, pTokens) {
		if len(part.Skipped) == 0 {
			continue
		}

		_, match, err := groupByFields(pctx, part.Skipped)
		if err != nil {
			perr.Add(fmt.Errorf("%w at %s: expression in group by clause is not supported by SnapSQL", cmn.ErrInvalidForSnapSQL, part.Skipped[0].Val.Position.String()))
		} else {
			field := cmn.FieldName{
				Expression: cmn.ToToken(part.Skipped),
			}

			switch match[0].Type {
			case "field":
				var (
					fieldName string
					fullName  string
				)

				switch len(match) {
				case 1:
					fieldName = match[0].Val.Value
					fullName = fieldName
				case 3:
					fieldName = match[2].Val.Value
					field.TableName = match[0].Val.Value
					fullName = field.TableName + "." + fieldName
				}

				field.Name = fieldName
				field.Pos = match[0].Val.Position
				nameToPos[fullName] = append(nameToPos[fullName], match[0].Val.Position.String())
				clause.Fields = append(clause.Fields, field)
			case "index":
				perr.Add(fmt.Errorf("%w: GROUP BY index at %s is not allowed in SnapSQL", cmn.ErrInvalidForSnapSQL, match[0].Val.Position.String()))
			case "case":
				var (
					fieldName string
					fullName  string
				)

				if len(match) > 4 {
					fieldName = match[4].Val.Value
					field.TableName = match[2].Val.Value
					fullName = field.TableName + "." + fieldName
				} else {
					fieldName = match[2].Val.Value
					fullName = fieldName
				}

				field.Name = fieldName
				field.Pos = match[0].Val.Position
				nameToPos[fullName] = append(nameToPos[fullName], match[0].Val.Position.String())
				clause.Fields = append(clause.Fields, field)
			}
		}
	}

	for name, pos := range nameToPos {
		if len(pos) > 1 {
			perr.Add(fmt.Errorf("%w: duplicate column name '%s' at %s", cmn.ErrInvalidSQL, name, strings.Join(pos, ", ")))
		}
	}
}
