package parserstep4

import (
	"fmt"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	assign      = pc.Seq(cmn.Identifier, cmn.SP, equal)
	singleField = pc.Seq(cmn.Identifier, cmn.SP, cmn.EOS)
)

// finalizeDeleteFromClause validates DeleteFromClause
func finalizeDeleteFromClause(clause *cmn.DeleteFromClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()

	pctx := pc.NewParseContext[tok.Token]()
	pTokens := cmn.ToParserToken(tokens)
	consume, tableName, err := parseTableName(pctx, pTokens, false)
	if err != nil {
		perr.Add(err)
	}
	clause.Table = tableName
	if consume != len(pTokens) {
		perr.Add(fmt.Errorf("%w: at %s: there are extra token exists", cmn.ErrInvalidSQL, tokens[consume].Position.String()))
	}
}

// finalizeUpdateClause validates UpdateClause
func finalizeUpdateClause(clause *cmn.UpdateClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()

	pctx := pc.NewParseContext[tok.Token]()
	pTokens := cmn.ToParserToken(tokens)
	consume, tableName, err := parseTableName(pctx, pTokens, false)
	if err != nil {
		perr.Add(err)
	}
	clause.Table = tableName
	if consume != len(pTokens) {
		perr.Add(fmt.Errorf("%w: at %s: there are extra token exists", cmn.ErrInvalidSQL, tokens[consume].Position.String()))
	}
}

// finalizeSetClause validates SetClause for UPDATE
func finalizeSetClause(clause *cmn.SetClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()
	if len(tokens) == 0 {
		perr.Add(fmt.Errorf("%w: SET clause must not be empty", cmn.ErrInvalidSQL))
		return
	}
	pTokens := cmn.ToParserToken(tokens)

	nameToPos := make(map[string][]string)
	for _, part := range fieldIter(pTokens) {
		pctx := pc.NewParseContext[tok.Token]()
		consume, name, err := assign(pctx, part.Skipped)
		if err != nil {
			perr.Add(fmt.Errorf("%w at %s: invalid SET clause", cmn.ErrInvalidSQL, tokens[0].Position.String()))
			continue
		}
		value := part.Skipped[consume:]
		if len(value) == 0 {
			perr.Add(fmt.Errorf("%w at %s: SET clause must have value", cmn.ErrInvalidSQL, tokens[0].Position.String()))
			continue
		}
		field := cmn.SetAssign{
			FieldName: name[0].Val.Value,
			Value:     cmn.ToToken(value),
		}
		clause.Assigns = append(clause.Assigns, field)
		nameToPos[field.FieldName] = append(nameToPos[field.FieldName], name[0].Val.Position.String())
	}
	for name, pos := range nameToPos {
		if len(pos) > 1 {
			perr.Add(fmt.Errorf("%w: duplicate column name '%s' at %s", cmn.ErrInvalidSQL, name, strings.Join(pos, ", ")))
		}
	}
}

// finalizeReturningClause validates ReturningClause
func finalizeReturningClause(clause *cmn.ReturningClause, perr *cmn.ParseError) {
	tokens := clause.ContentTokens()
	if len(tokens) == 0 {
		perr.Add(fmt.Errorf("%w: RETURNING clause must not be empty", cmn.ErrInvalidSQL))
		return
	}
	pTokens := cmn.ToParserToken(tokens)
	nameToPos := make(map[string][]string)
	pctx := pc.NewParseContext[tok.Token]()
	for _, part := range fieldIter(pTokens) {
		if len(part.Skipped) == 0 {
			continue
		}
		field, fieldTokens := parseFieldQualifier(part.Skipped)
		// standard single field
		if _, match, err := singleField(pctx, fieldTokens); err == nil {
			if field.FieldName == "" {
				field.FieldName = match[0].Val.Value // use identity as alias
			}
			field.FieldKind = cmn.SingleField
			field.OriginalField = match[0].Val.Value
			nameToPos[field.FieldName] = append(nameToPos[field.FieldName], field.Pos.String())
		} else {
			// complex field
			field.FieldKind = cmn.ComplexField
			field.Expression = cmn.ToToken(fieldTokens)
		}
		clause.Fields = append(clause.Fields, field)
	}
	for name, pos := range nameToPos {
		if len(pos) > 1 {
			perr.Add(fmt.Errorf("%w: duplicate column name '%s' at %s", cmn.ErrInvalidSQL, name, strings.Join(pos, ", ")))
		}
	}
}

func emptyCheck(clause cmn.ClauseNode, perr *cmn.ParseError) {
	if len(clause.ContentTokens()) == 0 {
		rawToken := clause.RawTokens()[0]
		perr.Add(fmt.Errorf("%w: at %s: %s clause must not be empty", cmn.ErrInvalidSQL, rawToken.Position.String(), rawToken.Value))
	}
}
