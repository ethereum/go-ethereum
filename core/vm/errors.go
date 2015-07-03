package vm

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/params"
)

var OutOfGasError = errors.New("Out of gas")
var DepthError = fmt.Errorf("Max call depth exceeded (%d)", params.CallCreateDepth)

type StackError struct {
	req, has int
}

func StackErr(req, has int) StackError {
	return StackError{req, has}
}

func (self StackError) Error() string {
	return fmt.Sprintf("stack error! require %v, have %v", self.req, self.has)
}

func IsStack(err error) bool {
	_, ok := err.(StackError)
	return ok
}
