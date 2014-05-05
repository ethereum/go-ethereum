package ethchain

import (
	"fmt"
)

// Parent error. In case a parent is unknown this error will be thrown
// by the block manager
type ParentErr struct {
	Message string
}

func (err *ParentErr) Error() string {
	return err.Message
}

func ParentError(hash []byte) error {
	return &ParentErr{Message: fmt.Sprintf("Block's parent unkown %x", hash)}
}

func IsParentErr(err error) bool {
	_, ok := err.(*ParentErr)

	return ok
}

// Block validation error. If any validation fails, this error will be thrown
type ValidationErr struct {
	Message string
}

func (err *ValidationErr) Error() string {
	return err.Message
}

func ValidationError(format string, v ...interface{}) *ValidationErr {
	return &ValidationErr{Message: fmt.Sprintf(format, v...)}
}

func IsValidationErr(err error) bool {
	_, ok := err.(*ValidationErr)

	return ok
}

type NonceErr struct {
	Message string
	Is, Exp uint64
}

func (err *NonceErr) Error() string {
	return err.Message
}

func NonceError(is, exp uint64) *NonceErr {
	return &NonceErr{Message: fmt.Sprintf("Nonce err. Is %d, expected %d", is, exp), Is: is, Exp: exp}
}

func IsNonceErr(err error) bool {
	_, ok := err.(*NonceErr)

	return ok
}
