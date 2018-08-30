package query

import (
	"fmt"
	"strings"
)

// And combines multiple conditions with a logical _AND_ operator.
func And(conditions ...Condition) Condition {
	return &andCond{
		conditions: conditions,
	}
}

type andCond struct {
	conditions []Condition
}

func (c *andCond) complies(f Fetcher) bool {
	for _, cond := range c.conditions {
		if !cond.complies(f) {
			return false
		}
	}
	return true
}

func (c *andCond) check() (err error) {
	for _, cond := range c.conditions {
		err = cond.check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *andCond) string() string {
	var all []string
	for _, cond := range c.conditions {
		all = append(all, cond.string())
	}
	return fmt.Sprintf("(%s)", strings.Join(all, " and "))
}
