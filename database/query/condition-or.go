package query

import (
	"fmt"
	"strings"
)

// Or combines multiple conditions with a logical _OR_ operator.
func Or(conditions ...Condition) Condition {
	return &orCond{
		conditions: conditions,
	}
}

type orCond struct {
	conditions []Condition
}

func (c *orCond) complies(f Fetcher) bool {
	for _, cond := range c.conditions {
		if cond.complies(f) {
			return true
		}
	}
	return false
}

func (c *orCond) check() (err error) {
	for _, cond := range c.conditions {
		err = cond.check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *orCond) string() string {
	var all []string
	for _, cond := range c.conditions {
		all = append(all, cond.string())
	}
	return fmt.Sprintf("(%s)", strings.Join(all, " or "))
}
