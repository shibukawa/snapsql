package parserstep4

import (
	"fmt"
	"strings"

	pc "github.com/shibukawa/parsercombinator"
	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
	tok "github.com/shibukawa/snapsql/tokenizer"
)

var (
	insertIntoClauseTableName = cmn.WS2(pc.Or(
		pc.Seq(cmn.Identifier, cmn.Dot, cmn.Identifier),
		pc.Seq(cmn.Identifier),
	))
	columnListStart     = pc.Seq(cmn.ParenOpen, cmn.SP)
	columnListSeparator = cmn.WS2(pc.Or(
		cmn.Comma,
		cmn.ParenClose,
	))
	columnName    = pc.Seq(cmn.Identifier, cmn.SP)
	columnListEnd = pc.Seq(cmn.SP, cmn.EOS)
)

func finalizeInsertIntoClause(clause *cmn.InsertIntoClause, selectClause *cmn.SelectClause, perr *cmn.ParseError) {
	clause.Columns = []string{}
	tokens := clause.ContentTokens()

	pctx := pc.NewParseContext[tok.Token]()
	pTokens := cmn.ToParserToken(tokens)
	consume, tableName, err := parseTableName(pctx, pTokens, false)
	if err != nil {
		perr.Add(err)
	}
	pTokens = pTokens[consume:]
	clause.Table = tableName

	consume, _, err = columnListStart(pctx, pTokens)
	if err != nil {
		if selectClause != nil {
			return // No column list, valid for INSERT ... SELECT
		}
		perr.Add(fmt.Errorf("%w at %s: column list is required unless select clause", cmn.ErrInvalidForSnapSQL, tokens[0].Position.String()))
		return
	}

	nameToPos := make(map[string][]string)
	pTokens = pTokens[consume:]

	for _, part := range pc.FindIter(pctx, columnListSeparator, pTokens) {
		pTokens = pTokens[part.Consume+len(part.Skipped):]
		_, columnName, err := columnName(pctx, part.Skipped)
		if err != nil {
			perr.Add(fmt.Errorf("%w at %s: invalid column name", cmn.ErrInvalidSQL, part.Skipped[0].Val.Position.String()))
			continue
		}
		if len(columnName) > 0 {
			n := columnName[0].Val
			clause.Columns = append(clause.Columns, n.Value)
			nameToPos[n.Value] = append(nameToPos[n.Value], n.Position.String())
		}
		if part.Match[0].Val.Type == tok.CLOSED_PARENS {
			break
		}
	}
	if _, _, err = columnListEnd(pctx, pTokens); err != nil {
		// ダミートークンを許容するために、エラーを無視する
		// perr.Add(fmt.Errorf("%w at %s: unnecessary token is at after column list", cmn.ErrInvalidSQL, tokens[len(tokens)-1].Position.String()))
		// return
	}
	for name, pos := range nameToPos {
		if len(pos) > 1 {
			perr.Add(fmt.Errorf("%w: duplicate column name '%s' at %s", cmn.ErrInvalidSQL, name, strings.Join(pos, ", ")))
		}
	}
}
