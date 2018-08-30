package query

import (
	"fmt"
	"regexp"
	"strings"
)

// Example:
// q.New("core:/",
//   q.Where("a", q.GreaterThan, 0),
//   q.Where("b", q.Equals, 0),
//   q.Or(
//       q.Where("c", q.StartsWith, "x"),
//       q.Where("d", q.Contains, "y")
//     )
//   )

var (
	prefixExpr = regexp.MustCompile("^[a-z-]+:")
)

// Query contains a compiled query.
type Query struct {
	prefix    string
	condition Condition
}

// New creates a new query.
func New(prefix string, condition Condition) (*Query, error) {
	// check prefix
	if !prefixExpr.MatchString(prefix) {
		return nil, fmt.Errorf("invalid prefix: %s", prefix)
	}

	// check condition
	if condition != nil {
		err := condition.check()
		if err != nil {
			return nil, err
		}
	} else {
		condition = &noCond{}
	}

	// return query
	return &Query{
		prefix:    prefix,
		condition: condition,
	}, nil
}

// MustCompile creates a new query and panics on an error.
func MustCompile(prefix string, condition Condition) *Query {
	q, err := New(prefix, condition)
	if err != nil {
		panic(err)
	}
	return q
}

// Matches checks whether the query matches the supplied data object.
func (q *Query) Matches(f Fetcher) bool {
	return q.condition.complies(f)
}

// String returns the string representation of the query.
func (q *Query) String() string {
	text := q.condition.string()
	if text == "" {
		return fmt.Sprintf("query %s", q.prefix)
	}
	if strings.HasPrefix(text, "(") {
		text = text[1 : len(text)-1]
	}
	return fmt.Sprintf("query %s where %s", q.prefix, text)
}
