package query

// Not negates the supplied condition.
func Not(c Condition) Condition {
	return &notCond{
		notC: c,
	}
}

type notCond struct {
	notC Condition
}

func (c *notCond) complies(f Fetcher) bool {
	return !c.notC.complies(f)
}

func (c *notCond) check() error {
	return c.notC.check()
}
