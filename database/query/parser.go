package query

var (
	operatorNames = map[string]uint8{
		"==":         Equals,
		">":          GreaterThan,
		">=":         GreaterThanOrEqual,
		"<":          LessThan,
		"<=":         LessThanOrEqual,
		"f==":        FloatEquals,
		"f>":         FloatGreaterThan,
		"f>=":        FloatGreaterThanOrEqual,
		"f<":         FloatLessThan,
		"f<=":        FloatLessThanOrEqual,
		"sameas":     SameAs,
		"s==":        SameAs,
		"contains":   Contains,
		"co":         Contains,
		"startswith": StartsWith,
		"sw":         StartsWith,
		"endswith":   EndsWith,
		"ew":         EndsWith,
		"in":         In,
		"matches":    Matches,
		"re":         Matches,
		"is":         Is,
		"exists":     Exists,
		"ex":         Exists,
	}
)

func getOpName(operator uint8) string {
	for opName, op := range operatorNames {
		if op == operator {
			return opName
		}
	}
	return "[unknown]"
}

// ParseQuery parses a plaintext query.
func ParseQuery(query string) (*Query, error) {
	return nil, nil
}
