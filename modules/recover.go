package modules

// Recoverf can be used to recover a goroutine from a panic.
// If recovered a new panic error will be reported and if errp is
// not nil, the value of errp will be set to the module error report.
func Recoverf(m *Module, errp *error, name, taskType string) {
	if x := recover(); x != nil {
		me := m.NewPanicError(name, taskType, x)
		me.Report()
		if errp != nil {
			*errp = me
		}
	}
}
