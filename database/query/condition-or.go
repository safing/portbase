package query

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
