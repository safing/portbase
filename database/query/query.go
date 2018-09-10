package query

import (
	"fmt"
	"strings"

	"github.com/Safing/portbase/database/accessor"
	"github.com/Safing/portbase/database/record"
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

// Query contains a compiled query.
type Query struct {
	checked     bool
	dbName      string
	dbKeyPrefix string
	where       Condition
	orderBy     string
	limit       int
	offset      int
}

// New creates a new query with the supplied prefix.
func New(prefix string) *Query {
	dbName, dbKeyPrefix := record.ParseKey(prefix)
	return &Query{
		dbName:      dbName,
		dbKeyPrefix: dbKeyPrefix,
	}
}

// Where adds filtering.
func (q *Query) Where(condition Condition) *Query {
	q.where = condition
	return q
}

// Limit limits the number of returned results.
func (q *Query) Limit(limit int) *Query {
	q.limit = limit
	return q
}

// Offset sets the query offset.
func (q *Query) Offset(offset int) *Query {
	q.offset = offset
	return q
}

// OrderBy orders the results by the given key.
func (q *Query) OrderBy(key string) *Query {
	q.orderBy = key
	return q
}

// Check checks for errors in the query.
func (q *Query) Check() (*Query, error) {
	if q.checked {
		return q, nil
	}

	// check condition
	if q.where != nil {
		err := q.where.check()
		if err != nil {
			return nil, err
		}
	} else {
		q.where = &noCond{}
	}

	q.checked = true
	return q, nil
}

// MustBeValid checks for errors in the query and panics if there is an error.
func (q *Query) MustBeValid() *Query {
	_, err := q.Check()
	if err != nil {
		panic(err)
	}
	return q
}

// IsChecked returns whether they query was checked.
func (q *Query) IsChecked() bool {
	return q.checked
}

// Matches checks whether the query matches the supplied data object.
func (q *Query) Matches(acc accessor.Accessor) bool {
	return q.where.complies(acc)
}

// Print returns the string representation of the query.
func (q *Query) Print() string {
	where := q.where.string()
	if where != "" {
		if strings.HasPrefix(where, "(") {
			where = where[1 : len(where)-1]
		}
		where = fmt.Sprintf(" where %s", where)
	}

	var orderBy string
	if q.orderBy != "" {
		orderBy = fmt.Sprintf(" orderby %s", q.orderBy)
	}

	var limit string
	if q.limit > 0 {
		limit = fmt.Sprintf(" limit %d", q.limit)
	}

	var offset string
	if q.offset > 0 {
		offset = fmt.Sprintf(" offset %d", q.offset)
	}

	return fmt.Sprintf("query %s:%s%s%s%s%s", q.dbName, q.dbKeyPrefix, where, orderBy, limit, offset)
}

// DatabaseName returns the name of the database.
func (q *Query) DatabaseName() string {
	return q.dbName
}

// DatabaseKeyPrefix returns the key prefix for the database.
func (q *Query) DatabaseKeyPrefix() string {
	return q.dbKeyPrefix
}
