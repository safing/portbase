package query

import (
	"fmt"
	"regexp"
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
	prefix     string
	conditions []Condition
}

// New creates a new query.
func New(prefix string, conditions ...Condition) (*Query, error) {
	// check prefix
	if !prefixExpr.MatchString(prefix) {
		return nil, fmt.Errorf("invalid prefix: %s", prefix)
	}

	// check conditions
	var err error
	for _, cond := range conditions {
		err = cond.check()
		if err != nil {
			return nil, err
		}
	}

	// return query
	return &Query{
		prefix:     prefix,
		conditions: conditions,
	}, nil
}

// MustCompile creates a new query and panics on an error.
func MustCompile(prefix string, conditions ...Condition) *Query {
	q, err := New(prefix, conditions...)
	if err != nil {
		panic(err)
	}
	return q
}

// Prepend prepends (check first) new query conditions to the query.
func (q *Query) Prepend(conditions ...Condition) error {
	// check conditions
	var err error
	for _, cond := range conditions {
		err = cond.check()
		if err != nil {
			return err
		}
	}

	q.conditions = append(conditions, q.conditions...)
	return nil
}

// Append appends (check last) new query conditions to the query.
func (q *Query) Append(conditions ...Condition) error {
	// check conditions
	var err error
	for _, cond := range conditions {
		err = cond.check()
		if err != nil {
			return err
		}
	}

	q.conditions = append(q.conditions, conditions...)
	return nil
}

// Matches checks whether the query matches the supplied data object.
func (q *Query) Matches(f Fetcher) bool {
	for _, cond := range q.conditions {
		if !cond.complies(f) {
			return false
		}
	}
	return true
}
