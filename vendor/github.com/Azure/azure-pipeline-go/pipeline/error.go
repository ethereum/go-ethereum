package pipeline

import (
	"fmt"
	"runtime"
)

type causer interface {
	Cause() error
}

// ErrorNode can be an embedded field in a private error object. This field
// adds Program Counter support and a 'cause' (reference to a preceding error).
// When initializing a error type with this embedded field, initialize the
// ErrorNode field by calling ErrorNode{}.Initialize(cause).
type ErrorNode struct {
	pc    uintptr // Represents a Program Counter that you can get symbols for.
	cause error   // Refers to the preceding error (or nil)
}

// Error returns a string with the PC's symbols or "" if the PC is invalid.
// When defining a new error type, have its Error method call this one passing
// it the string representation of the error.
func (e *ErrorNode) Error(msg string) string {
	s := ""
	if fn := runtime.FuncForPC(e.pc); fn != nil {
		file, line := fn.FileLine(e.pc)
		s = fmt.Sprintf("-> %v, %v:%v\n", fn.Name(), file, line)
	}
	s += msg + "\n\n"
	if e.cause != nil {
		s += e.cause.Error() + "\n"
	}
	return s
}

// Cause returns the error that preceded this error.
func (e *ErrorNode) Cause() error { return e.cause }

// Temporary returns true if the error occurred due to a temporary condition.
func (e ErrorNode) Temporary() bool {
	type temporary interface {
		Temporary() bool
	}

	for err := e.cause; err != nil; {
		if t, ok := err.(temporary); ok {
			return t.Temporary()
		}

		if cause, ok := err.(causer); ok {
			err = cause.Cause()
		} else {
			err = nil
		}
	}
	return false
}

// Timeout returns true if the error occurred due to time expiring.
func (e ErrorNode) Timeout() bool {
	type timeout interface {
		Timeout() bool
	}

	for err := e.cause; err != nil; {
		if t, ok := err.(timeout); ok {
			return t.Timeout()
		}

		if cause, ok := err.(causer); ok {
			err = cause.Cause()
		} else {
			err = nil
		}
	}
	return false
}

// Initialize is used to initialize an embedded ErrorNode field.
// It captures the caller's program counter and saves the cause (preceding error).
// To initialize the field, use "ErrorNode{}.Initialize(cause, 3)". A callersToSkip
// value of 3 is very common; but, depending on your code nesting, you may need
// a different value.
func (ErrorNode) Initialize(cause error, callersToSkip int) ErrorNode {
	// Get the PC of Initialize method's caller.
	pc := [1]uintptr{}
	_ = runtime.Callers(callersToSkip, pc[:])
	return ErrorNode{pc: pc[0], cause: cause}
}

// Cause walks all the preceding errors and return the originating error.
func Cause(err error) error {
	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}

// NewError creates a simple string error (like Error.New). But, this
// error also captures the caller's Program Counter and the preceding error.
func NewError(cause error, msg string) error {
	return &pcError{
		ErrorNode: ErrorNode{}.Initialize(cause, 3),
		msg:       msg,
	}
}

// pcError is a simple string error (like error.New) with an ErrorNode (PC & cause).
type pcError struct {
	ErrorNode
	msg string
}

// Error satisfies the error interface. It shows the error with Program Counter
// symbols and calls Error on the preceding error so you can see the full error chain.
func (e *pcError) Error() string { return e.ErrorNode.Error(e.msg) }
