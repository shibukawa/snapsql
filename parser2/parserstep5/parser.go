package parserstep5

/*import "github.com/shibukawa/snapsql/tokenizer"

func directive(name, directiveType string) pc.Parser[Entity] {
	return ws(pc.Trace(name, func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if tokens[0].Type == "raw" {
			o := tokens[0].Val.Original
			if o.Type == tokenizer.BLOCK_COMMENT && o.IsSnapSQLDirective && o.DirectiveType == directiveType {
				return 1, []pc.Token[Entity]{
					{
						Type: name,
						Pos:  tokens[0].Pos,
						Val: Entity{
							Original: o,
						},
						Raw: o.Value,
					},
				}, nil
			}
		}
		return 0, nil, pc.ErrNotMatch
	}))
}

func ifDirective() pc.Parser[Entity] {
	return directive("if-directive", "if")
}

func endDirective() pc.Parser[Entity] {
	return directive("end-directive", "end")
}
*/
