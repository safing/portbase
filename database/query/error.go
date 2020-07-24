package query

import "fmt"

type conditionKeyError string

func (key conditionKeyError) Error() string {
	return string(key)
}

// OperatorError is returned for unknown or unsupported operators.
type OperatorError struct {
	Operator uint8
}

func (oe *OperatorError) Error() string {
	return fmt.Sprintf("unknown or unsupported operator with ID %d", oe.Operator)
}

// SyntaxError is a generic sytax error.
type SyntaxError struct {
	Msg    string
	Pos    int
	Symbol string
}

func (se *SyntaxError) Error() string {
	if se.Symbol != "" {
		return fmt.Sprintf("syntax error: %q at position %d: %s", se.Symbol, se.Pos, se.Msg)
	}
	return fmt.Sprintf("syntax error: position %d: %s", se.Pos, se.Msg)
}

func syntaxErr(sn *snippet, msg string) error {
	return &SyntaxError{Symbol: sn.text, Pos: sn.globalPosition, Msg: msg}
}
