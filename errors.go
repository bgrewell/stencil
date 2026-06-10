package stencil

import "fmt"

type ErrCode int

const (
	ExitOK           ErrCode = iota
	ExitNoChange             = 10 // reserved for app logic
	ExitVerifyFailed         = 20
	ExitNetworkError         = 30

	ExitUsage   = 2
	ExitRuntime = 1
)

type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

type ExecError struct{ Msg string }

func (e *ExecError) Error() string { return e.Msg }

func newUsagef(format string, a ...any) *UsageError {
	return &UsageError{Msg: fmt.Sprintf(format, a...)}
}
func newExecf(format string, a ...any) *ExecError { return &ExecError{Msg: fmt.Sprintf(format, a...)} }
