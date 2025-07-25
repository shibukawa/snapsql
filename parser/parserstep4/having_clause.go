package parserstep4

import (
	"fmt"

	cmn "github.com/shibukawa/snapsql/parser/parsercommon"
)

// finalizeHavingClause validates HAVING clause
// 引数: having句, groupBy句, ParseError
func finalizeHavingClause(having *cmn.HavingClause, groupBy *cmn.GroupByClause, perr *cmn.ParseError) {
	if having != nil && groupBy == nil {
		perr.Add(fmt.Errorf("%w: HAVING clause at %s requires GROUP BY clause", cmn.ErrInvalidSQL, having.RawTokens()[0].Position.String()))
	}
}
