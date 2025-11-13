package intermediate

// ExplangExpression stores parsed explang steps aligned with CELExpressions.
type ExplangExpression struct {
	ID               string    `json:"id"`
	EnvironmentIndex int       `json:"environment_index"`
	Position         Position  `json:"position,omitzero"`
	Steps            []Expressions `json:"steps"`
}
