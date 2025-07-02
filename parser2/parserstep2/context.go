package parserstep2

// SQLClause represents different SQL clause contexts
type SQLClause int

const (
	WhereClause SQLClause = iota
	GroupByClause
	HavingClause
	OrderByClause
	SelectClause
)

// String returns the string representation of SQLClause
func (c SQLClause) String() string {
	switch c {
	case WhereClause:
		return "WHERE"
	case GroupByClause:
		return "GROUP BY"
	case HavingClause:
		return "HAVING"
	case OrderByClause:
		return "ORDER BY"
	case SelectClause:
		return "SELECT"
	default:
		return "UNKNOWN"
	}
}
