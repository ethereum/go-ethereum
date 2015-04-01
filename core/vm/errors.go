package vm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

type OutOfGasError struct {
	req, has *big.Int
}

func OOG(req, has *big.Int) OutOfGasError {
	return OutOfGasError{req, has}
}

func (self OutOfGasError) Error() string {
	return fmt.Sprintf("out of gas! require %v, have %v", self.req, self.has)
}

func IsOOGErr(err error) bool {
	_, ok := err.(OutOfGasError)
	return ok
}

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

type DepthError struct{}

func (self DepthError) Error() string {
	return fmt.Sprintf("Max call depth exceeded (%d)", params.CallCreateDepth)
}

func IsDepthErr(err error) bool {
	_, ok := err.(DepthError)
	return ok
}
