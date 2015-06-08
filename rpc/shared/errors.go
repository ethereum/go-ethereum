package shared

import "fmt"

type InvalidTypeError struct {
	method string
	msg    string
}

func (e *InvalidTypeError) Error() string {
	return fmt.Sprintf("invalid type on field %s: %s", e.method, e.msg)
}

func NewInvalidTypeError(method, msg string) *InvalidTypeError {
	return &InvalidTypeError{
		method: method,
		msg:    msg,
	}
}

type InsufficientParamsError struct {
	have int
	want int
}

func (e *InsufficientParamsError) Error() string {
	return fmt.Sprintf("insufficient params, want %d have %d", e.want, e.have)
}

func NewInsufficientParamsError(have int, want int) *InsufficientParamsError {
	return &InsufficientParamsError{
		have: have,
		want: want,
	}
}

type NotImplementedError struct {
	Method string
}

func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("%s method not implemented", e.Method)
}

func NewNotImplementedError(method string) *NotImplementedError {
	return &NotImplementedError{
		Method: method,
	}
}

type DecodeParamError struct {
	err string
}

func (e *DecodeParamError) Error() string {
	return fmt.Sprintf("could not decode, %s", e.err)

}

func NewDecodeParamError(errstr string) error {
	return &DecodeParamError{
		err: errstr,
	}
}

type ValidationError struct {
	ParamName string
	msg       string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s not valid, %s", e.ParamName, e.msg)
}

func NewValidationError(param string, msg string) error {
	return &ValidationError{
		ParamName: param,
		msg:       msg,
	}
}

type NotAvailableError struct {
	Method string
	Reason string
}

func (e *NotAvailableError) Error() string {
	return fmt.Sprintf("%s method not available: %s", e.Method, e.Reason)
}

func NewNotAvailableError(method string, reason string) *NotAvailableError {
	return &NotAvailableError{
		Method: method,
		Reason: reason,
	}
}
