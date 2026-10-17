package business

import "fmt"

// TerminalError is a validation / bad-input failure — do not retry.
type TerminalError struct {
	Msg string
}

func (e *TerminalError) Error() string { return e.Msg }

func Terminal(format string, args ...any) error {
	return &TerminalError{Msg: fmt.Sprintf(format, args...)}
}

func IsTerminal(err error) bool {
	_, ok := err.(*TerminalError)
	return ok
}
