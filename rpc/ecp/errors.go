package ecp

import (
	"fmt"
	"reflect"
)

type errorCode int

const (
	_                   errorCode = iota
	ProtocolError                 // Received message invalid
	InvalidRequestError           // Received message isn't a valid request
	MethodNotFoundError           // Requested method not found
	InvalidArguments              // Received arguments are invalid
	LogicError                    // Service returned an error
)

// isRecoverable returns an indication if the server can recover from the error
func isRecoverable(err error) bool {
	switch err.(type) {
	case *unknownServiceError, *unknownMethodError, *invalidNumberOfArgumentsError, *invalidStructArgumentError,
		*invalidArgumentError, *invalidRequestError, *unsupportedTypeError, *callbackError:
		return true
	}

	return false
}

// InvalidLeadInError is raised when the received message doesn't for a valid
// RESP message. It is a bit more detailed than InvalidRequestError.
type invalidLeadInError struct {
	got byte
}

func (e *invalidLeadInError) Error() string {
	return fmt.Sprintf("%d ECP error, received unexpected byte 0x%x", ProtocolError, e.got)
}

// UnexpectedByteError occurs when incoming data doesn't follow the protocol specifications
type unexpectedByteError struct {
	expected, got byte
}

func (e *unexpectedByteError) Error() string {
	return fmt.Sprintf("%d ECP error, expected 0x%x got 0x%x", ProtocolError, e.expected, e.got)
}

type unknownServiceError struct {
	svc string
}

func (e *unknownServiceError) Error() string {
	return fmt.Sprintf("%d service '%s' not found", MethodNotFoundError, e.svc)
}

type unknownMethodError struct {
	svc    string
	method string
}

func (e *unknownMethodError) Error() string {
	return fmt.Sprintf("%d method '%s.%s' not found", MethodNotFoundError, e.svc, e.method)
}

type invalidNumberOfArgumentsError struct {
	svc    string
	method string
	exp    int
	got    int
}

func (e *invalidNumberOfArgumentsError) Error() string {
	return fmt.Sprintf("%d %s.%s expect %d argument(s) but got %d",
		InvalidArguments, e.svc, e.method, e.exp, e.got)
}

type invalidArgumentError struct {
	pos int
	exp reflect.Type
	got reflect.Type
}

func (e *invalidArgumentError) Error() string {
	return fmt.Sprintf("%d expected type for argument %d is %s but got %s", InvalidArguments, e.pos, e.exp, e.got)
}

type invalidStructArgumentError struct {
	to  reflect.Type
	exp int
	got int
}

func (e *invalidStructArgumentError) Error() string {
	return fmt.Sprintf("%s expected %d values but got %d", e.to.Name(), e.exp, e.got)
}

type invalidRequestError struct {
}

func (e *invalidRequestError) Error() string {
	return fmt.Sprintf("%d invalid request", InvalidRequestError)
}

type unsupportedTypeError struct {
	name string
}

func (e *unsupportedTypeError) Error() string {
	return fmt.Sprintf("%d '%s' is an unsupported type", ProtocolError, e.name)
}

type callbackError struct {
	msg string
}

func (e *callbackError) Error() string {
	return fmt.Sprintf("%d %s", LogicError, e.msg)
}
